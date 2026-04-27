// Package client — thin HTTP client wrapper for the bench harness.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client wraps net/http.Client with bench-friendly defaults: a sized connection
// pool, a default 10s timeout, and a small retry policy (refused / EOF only).
type Client struct {
	BaseURL string
	HTTP    *http.Client
	Prover  *Prover // optional DPoP prover; nil disables DPoP
}

// New builds a Client targeting baseURL with a connection pool sized for `concurrency`.
func New(baseURL string, concurrency int) *Client {
	if concurrency <= 0 {
		concurrency = 10
	}
	maxConns := concurrency * 2
	tr := &http.Transport{
		MaxIdleConns:        maxConns,
		MaxIdleConnsPerHost: maxConns,
		MaxConnsPerHost:     maxConns,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		},
	}
}

// SetProver attaches a DPoP prover so subsequent requests can ask for DPoP signing.
func (c *Client) SetProver(p *Prover) { c.Prover = p }

// resolve returns BaseURL+path or path if path is already absolute.
func (c *Client) resolve(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.BaseURL + path
}

// Reply is the result of a request; Body is fully read.
type Reply struct {
	Status     int
	Body       []byte
	Header     http.Header
	LatencyDur time.Duration
}

// JSON decodes Reply.Body into `into`.
func (r *Reply) JSON(into any) error {
	if into == nil {
		return nil
	}
	return json.Unmarshal(r.Body, into)
}

// Headers is a small helper map type for request-header injection.
type Headers map[string]string

// Post sends a POST with optional body. body may be []byte, string, or any (JSON-marshalled).
func (c *Client) Post(ctx context.Context, path string, body any, h Headers) (*Reply, error) {
	return c.do(ctx, http.MethodPost, path, body, h)
}

// Get sends a GET.
func (c *Client) Get(ctx context.Context, path string, h Headers) (*Reply, error) {
	return c.do(ctx, http.MethodGet, path, nil, h)
}

// JSON calls the named method+path with JSON body and decodes into `into`.
func (c *Client) JSON(ctx context.Context, method, path string, body any, h Headers, into any) (*Reply, error) {
	r, err := c.do(ctx, method, path, body, h)
	if err != nil {
		return r, err
	}
	if into != nil && len(r.Body) > 0 {
		if err := json.Unmarshal(r.Body, into); err != nil {
			return r, fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	}
	return r, nil
}

// PostForm submits application/x-www-form-urlencoded.
func (c *Client) PostForm(ctx context.Context, path string, form url.Values, h Headers) (*Reply, error) {
	if h == nil {
		h = Headers{}
	}
	h["Content-Type"] = "application/x-www-form-urlencoded"
	return c.do(ctx, http.MethodPost, path, []byte(form.Encode()), h)
}

func (c *Client) do(ctx context.Context, method, path string, body any, h Headers) (*Reply, error) {
	full := c.resolve(path)

	var rawBody []byte
	switch b := body.(type) {
	case nil:
		rawBody = nil
	case []byte:
		rawBody = b
	case string:
		rawBody = []byte(b)
	default:
		var err error
		rawBody, err = json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		if h == nil {
			h = Headers{}
		}
		if _, ok := h["Content-Type"]; !ok {
			h["Content-Type"] = "application/json"
		}
	}

	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var rdr io.Reader
		if rawBody != nil {
			rdr = bytes.NewReader(rawBody)
		}
		req, err := http.NewRequestWithContext(ctx, method, full, rdr)
		if err != nil {
			return nil, err
		}
		for k, v := range h {
			req.Header.Set(k, v)
		}

		start := time.Now()
		resp, err := c.HTTP.Do(req)
		dur := time.Since(start)
		if err != nil {
			lastErr = err
			if isRetriable(err) && attempt < maxAttempts {
				time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
				continue
			}
			return nil, err
		}
		bts, rerr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if rerr != nil {
			lastErr = rerr
			if isRetriable(rerr) && attempt < maxAttempts {
				time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
				continue
			}
			return nil, rerr
		}
		return &Reply{
			Status:     resp.StatusCode,
			Body:       bts,
			Header:     resp.Header,
			LatencyDur: dur,
		}, nil
	}
	if lastErr == nil {
		lastErr = errors.New("request failed after retries")
	}
	return nil, lastErr
}

// isRetriable returns true only for connection refused / EOF (transient transport errors).
func isRetriable(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	if strings.Contains(s, "connection refused") {
		return true
	}
	if strings.Contains(s, "EOF") {
		return true
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return false
}

// AuthHeaders builds Headers with Authorization and (optional) DPoP for the given method+url.
func (c *Client) AuthHeaders(ctx context.Context, method, path, bearer string) (Headers, error) {
	h := Headers{}
	if bearer != "" {
		h["Authorization"] = "Bearer " + bearer
	}
	if c.Prover != nil {
		proof, err := c.Prover.Sign(method, c.resolve(path), SignOpts{})
		if err != nil {
			return nil, err
		}
		h["DPoP"] = proof
	}
	return h, nil
}
