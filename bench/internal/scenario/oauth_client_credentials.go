package scenario

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

// OAuthClientCredentials registers one DCR client during setup, then hammers
// POST /oauth/token with grant_type=client_credentials.
type OAuthClientCredentials struct{}

// NewOAuthClientCredentials constructs the scenario.
func NewOAuthClientCredentials() *OAuthClientCredentials { return &OAuthClientCredentials{} }

func (s *OAuthClientCredentials) Name() string { return "oauth_client_credentials" }

func (s *OAuthClientCredentials) Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error {
	body := map[string]any{
		"client_name":                "bench-cc-" + runID,
		"grant_types":                []string{"client_credentials"},
		"token_endpoint_auth_method": "client_secret_basic",
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
	if resp.ClientID == "" || resp.ClientSecret == "" {
		return fmt.Errorf("oauth register: missing client_id/secret in response: %s", string(r.Body))
	}
	fx.AddClient(fixtures.OAuthClient{ClientID: resp.ClientID, ClientSecret: resp.ClientSecret})
	return nil
}

func (s *OAuthClientCredentials) Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result {
	hist := metrics.New()
	ok := metrics.NewCounter()
	errs := metrics.NewCounter()

	clients := fx.SnapshotClients()
	if len(clients) == 0 {
		return Result{Name: s.Name(), Errors: 1, Extra: map[string]any{"error": "no oauth client"}}
	}
	cli := clients[0]
	basic := base64.StdEncoding.EncodeToString([]byte(cli.ClientID + ":" + cli.ClientSecret))
	headers := client.Headers{
		"Authorization": "Basic " + basic,
		"Content-Type":  "application/x-www-form-urlencoded",
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	formBody := []byte(form.Encode())

	body := func(workerID int) (bool, time.Duration) {
		start := time.Now()
		// Re-shallow-copy the headers map per call so the client can't accidentally mutate caller state.
		h := client.Headers{}
		for k, v := range headers {
			h[k] = v
		}
		r, err := c.Post(ctx, "/oauth/token", formBody, h)
		lat := time.Since(start)
		if err != nil {
			return false, lat
		}
		if r.Status >= 200 && r.Status < 300 {
			return true, lat
		}
		return false, lat
	}

	start := time.Now()
	runWorkers(opts.Concurrency, opts.Duration, hist, ok, errs, body)
	dur := time.Since(start)

	extra := map[string]any{
		"client_id": cli.ClientID,
	}
	return finalize(s.Name(), hist, ok, errs, dur, extra)
}

func (s *OAuthClientCredentials) Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error {
	return nil
}
