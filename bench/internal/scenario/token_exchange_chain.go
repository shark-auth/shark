package scenario

// token_exchange_chain — C2 MARQUEE
//
// Setup: create 3 agents (A, B, C). A gets client_credentials token.
// B does depth-1 exchange (subject=A's token). C does depth-2 exchange
// (subject=B's exchanged token), giving us depth-3 total delegation.
//
// Load: Run each depth level 100× sequentially to get stable p50/p99.
// Reports p50/p99 per depth in Extra. Uses cascade profile (1 conc, runs to completion).

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

const tokenExchangeGrantType = "urn:ietf:params:oauth:grant-type:token-exchange"
const tokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"

// tokenExchangeAgent holds DCR credentials for one agent in the chain.
type tokenExchangeAgent struct {
	ClientID     string
	ClientSecret string
	basic        string
}

func newTEAgent(clientID, clientSecret string) tokenExchangeAgent {
	return tokenExchangeAgent{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		basic:        base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret)),
	}
}

// TokenExchangeChain is the depth-3 delegation chain latency scenario.
type TokenExchangeChain struct {
	agents [3]tokenExchangeAgent // 0=A,1=B,2=C
}

// NewTokenExchangeChain constructs the scenario.
func NewTokenExchangeChain() *TokenExchangeChain { return &TokenExchangeChain{} }

func (s *TokenExchangeChain) Name() string { return "token_exchange_chain" }

func (s *TokenExchangeChain) Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error {
	if opts.AdminKey == "" {
		return fmt.Errorf("token_exchange_chain: AdminKey required")
	}
	adminH := client.Headers{"Authorization": "Bearer " + opts.AdminKey}

	names := []string{"bench-te-a", "bench-te-b", "bench-te-c"}
	for i, name := range names {
		// Use admin agents API — DCR rejects token-exchange grant type.
		body := map[string]any{
			"name":        name + "-" + runID,
			"grant_types": []string{"client_credentials", tokenExchangeGrantType},
			"scopes":      []string{"delegate"},
		}
		var resp struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
		r, err := c.JSON(ctx, "POST", "/api/v1/agents", body, adminH, &resp)
		if err != nil {
			return fmt.Errorf("create agent %s: %w", name, err)
		}
		if r.Status < 200 || r.Status >= 300 {
			return fmt.Errorf("create agent %s: status=%d body=%s", name, r.Status, string(r.Body))
		}
		if resp.ClientID == "" {
			return fmt.Errorf("create agent %s: empty client_id", name)
		}
		s.agents[i] = newTEAgent(resp.ClientID, resp.ClientSecret)
	}
	return nil
}

// issueCC issues a fresh client_credentials token for the given agent.
func (s *TokenExchangeChain) issueCC(ctx context.Context, c *client.Client, ag tokenExchangeAgent) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "delegate")
	h := client.Headers{
		"Authorization": "Basic " + ag.basic,
		"Content-Type":  "application/x-www-form-urlencoded",
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	r, err := c.JSON(ctx, "POST", "/oauth/token", []byte(form.Encode()), h, &resp)
	if err != nil {
		return "", err
	}
	if r.Status != 200 {
		return "", fmt.Errorf("cc token status=%d body=%s", r.Status, string(r.Body))
	}
	return resp.AccessToken, nil
}

// exchange performs one token exchange: actor=ag authenticates, subject=subjectToken.
func (s *TokenExchangeChain) exchange(ctx context.Context, c *client.Client, ag tokenExchangeAgent, subjectToken string) (string, time.Duration, error) {
	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrantType)
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", tokenTypeAccessToken)
	form.Set("scope", "delegate")
	h := client.Headers{
		"Authorization": "Basic " + ag.basic,
		"Content-Type":  "application/x-www-form-urlencoded",
	}
	start := time.Now()
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	r, err := c.JSON(ctx, "POST", "/oauth/token", []byte(form.Encode()), h, &resp)
	lat := time.Since(start)
	if err != nil {
		return "", lat, err
	}
	if r.Status != 200 {
		return "", lat, fmt.Errorf("exchange status=%d body=%s", r.Status, string(r.Body))
	}
	return resp.AccessToken, lat, nil
}

func (s *TokenExchangeChain) Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result {
	const iterations = 100

	histD1 := metrics.New()
	histD2 := metrics.New()
	histD3 := metrics.New()
	var errCount int64

	for i := 0; i < iterations; i++ {
		if ctx.Err() != nil {
			break
		}

		// Depth 1: A issues CC token, B exchanges it.
		tokenA, err := s.issueCC(ctx, c, s.agents[0])
		if err != nil {
			errCount++
			continue
		}
		_, d1Lat, err := s.exchange(ctx, c, s.agents[1], tokenA)
		histD1.Record(int64(d1Lat))
		if err != nil {
			errCount++
			continue
		}

		// Depth 2: A issues CC, B exchanges → tokenB, C exchanges tokenB.
		tokenA2, err := s.issueCC(ctx, c, s.agents[0])
		if err != nil {
			errCount++
			continue
		}
		tokenB, _, err := s.exchange(ctx, c, s.agents[1], tokenA2)
		if err != nil {
			errCount++
			continue
		}
		_, d2Lat, err := s.exchange(ctx, c, s.agents[2], tokenB)
		histD2.Record(int64(d2Lat))
		if err != nil {
			errCount++
			continue
		}

		// Depth 3: full chain — measure just the final hop's latency.
		tokenA3, err := s.issueCC(ctx, c, s.agents[0])
		if err != nil {
			errCount++
			continue
		}
		tokenB2, _, err := s.exchange(ctx, c, s.agents[1], tokenA3)
		if err != nil {
			errCount++
			continue
		}
		_, d3Lat, err := s.exchange(ctx, c, s.agents[2], tokenB2)
		histD3.Record(int64(d3Lat))
		if err != nil {
			errCount++
		}
	}

	extra := map[string]any{
		"iterations":   iterations,
		"depth1_p50":   histD1.Quantile(0.50).String(),
		"depth1_p99":   histD1.Quantile(0.99).String(),
		"depth2_p50":   histD2.Quantile(0.50).String(),
		"depth2_p99":   histD2.Quantile(0.99).String(),
		"depth3_p50":   histD3.Quantile(0.50).String(),
		"depth3_p99":   histD3.Quantile(0.99).String(),
	}

	// Use depth-3 histogram as the primary latency stat for the summary table.
	totalOK := int64(histD3.Count())
	dur := time.Duration(iterations) * histD3.Quantile(0.50) // approximate
	if dur == 0 {
		dur = time.Second
	}
	return Result{
		Name:       s.Name(),
		OK:         totalOK,
		Errors:     errCount,
		LatencyP50: histD3.Quantile(0.50),
		LatencyP95: histD3.Quantile(0.95),
		LatencyP99: histD3.Quantile(0.99),
		Throughput: float64(totalOK) / dur.Seconds(),
		Extra:      extra,
	}
}

func (s *TokenExchangeChain) Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error {
	return nil
}
