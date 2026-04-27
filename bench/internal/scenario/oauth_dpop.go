package scenario

// oauth_dpop — C1 MARQUEE
//
// Registers one DCR client that accepts DPoP-bound tokens, then hammers
// POST /oauth/token using DPoP proof headers. Compares p99 vs the bearer
// baseline (oauth_client_credentials) to measure DPoP overhead.
//
// Two sub-runs within Run:
//   1. Bearer (no DPoP): straight client_credentials, same client.
//   2. DPoP  (batched cache mode): same client but with DPoP proof on every request.
//
// Overhead delta = DPoP p99 - Bearer p99.

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/sharkauth/bench/internal/client"
	"github.com/sharkauth/bench/internal/fixtures"
	"github.com/sharkauth/bench/internal/metrics"
)

// OAuthDPoP exercises the DPoP token issuance path.
type OAuthDPoP struct {
	clientID     string
	clientSecret string
	basic        string
}

// NewOAuthDPoP constructs the scenario.
func NewOAuthDPoP() *OAuthDPoP { return &OAuthDPoP{} }

func (s *OAuthDPoP) Name() string { return "oauth_dpop" }

func (s *OAuthDPoP) Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error {
	body := map[string]any{
		"client_name":                "bench-dpop-" + runID,
		"grant_types":                []string{"client_credentials"},
		"token_endpoint_auth_method": "client_secret_basic",
		// DPoP-bound clients can still use the token endpoint; DPoP binding is
		// opt-in per request. We test overhead by sending vs not sending DPoP header.
	}
	var resp struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	r, err := c.JSON(ctx, "POST", "/oauth/register", body, nil, &resp)
	if err != nil {
		return fmt.Errorf("oauth register: %w", err)
	}
	if r.Status < 200 || r.Status >= 300 {
		return fmt.Errorf("oauth register: status=%d body=%s", r.Status, string(r.Body))
	}
	if resp.ClientID == "" {
		return fmt.Errorf("oauth register: empty client_id: %s", string(r.Body))
	}
	s.clientID = resp.ClientID
	s.clientSecret = resp.ClientSecret
	s.basic = base64.StdEncoding.EncodeToString([]byte(resp.ClientID + ":" + resp.ClientSecret))
	return nil
}

func (s *OAuthDPoP) Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result {
	// Build a fresh DPoP prover for this scenario (resign mode to measure real overhead).
	prover, err := client.NewProver("resign")
	if err != nil {
		return Result{Name: s.Name(), Errors: 1, Extra: map[string]any{"error": "prover: " + err.Error()}}
	}

	histBearer := metrics.New()
	histDPoP := metrics.New()
	okBearer := metrics.NewCounter()
	okDPoP := metrics.NewCounter()
	errBearer := metrics.NewCounter()
	errDPoP := metrics.NewCounter()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	formBody := []byte(form.Encode())

	// Sub-run 1: bearer (no DPoP header) — half the duration.
	halfDur := opts.Duration / 2
	if halfDur <= 0 {
		halfDur = 15 * time.Second
	}
	bearerHeaders := client.Headers{
		"Authorization": "Basic " + s.basic,
		"Content-Type":  "application/x-www-form-urlencoded",
	}
	bearerFn := func(workerID int) (bool, time.Duration) {
		h := copyHeaders(bearerHeaders)
		start := time.Now()
		r, err := c.Post(ctx, "/oauth/token", formBody, h)
		lat := time.Since(start)
		if err != nil {
			return false, lat
		}
		return r.Status >= 200 && r.Status < 300, lat
	}
	runWorkers(opts.Concurrency, halfDur, histBearer, okBearer, errBearer, bearerFn)

	// Sub-run 2: DPoP (resign mode = fresh proof per request) — same half duration.
	tokenURL := c.BaseURL + "/oauth/token"
	dpopFn := func(workerID int) (bool, time.Duration) {
		proof, perr := prover.Sign("POST", tokenURL, client.SignOpts{})
		if perr != nil {
			return false, 0
		}
		h := copyHeaders(bearerHeaders)
		h["DPoP"] = proof
		start := time.Now()
		r, err := c.Post(ctx, "/oauth/token", formBody, h)
		lat := time.Since(start)
		if err != nil {
			return false, lat
		}
		return r.Status >= 200 && r.Status < 300, lat
	}
	runWorkers(opts.Concurrency, halfDur, histDPoP, okDPoP, errDPoP, dpopFn)

	bearerP99 := histBearer.Quantile(0.99)
	dpopP99 := histDPoP.Quantile(0.99)
	overhead := dpopP99 - bearerP99

	extra := map[string]any{
		"bearer_p50":      histBearer.Quantile(0.50).String(),
		"bearer_p99":      bearerP99.String(),
		"dpop_p50":        histDPoP.Quantile(0.50).String(),
		"dpop_p99":        dpopP99.String(),
		"overhead_p99":    overhead.String(),
		"overhead_p99_ms": overhead.Milliseconds(),
		"bearer_ok":       okBearer.Load(),
		"dpop_ok":         okDPoP.Load(),
		"dpop_mode":       "resign",
	}

	// Primary result uses DPoP latency numbers.
	totalOK := okBearer.Load() + okDPoP.Load()
	dur := opts.Duration
	if dur == 0 {
		dur = time.Second
	}
	return Result{
		Name:       s.Name(),
		OK:         totalOK,
		Errors:     errBearer.Load() + errDPoP.Load(),
		LatencyP50: histDPoP.Quantile(0.50),
		LatencyP95: histDPoP.Quantile(0.95),
		LatencyP99: dpopP99,
		Throughput: float64(totalOK) / dur.Seconds(),
		Extra:      extra,
	}
}

func (s *OAuthDPoP) Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error {
	return nil
}

// copyHeaders returns a shallow copy.
func copyHeaders(h client.Headers) client.Headers {
	out := make(client.Headers, len(h))
	for k, v := range h {
		out[k] = v
	}
	return out
}
