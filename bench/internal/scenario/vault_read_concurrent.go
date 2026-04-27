package scenario

// vault_read_concurrent — D1 MARQUEE
//
// Setup:
//   1. Create a vault provider (github template, fake credentials).
//   2. Create N users via admin API.
//   3. Seed a demo vault connection per user via POST /api/v1/admin/vault/connections/_seed_demo.
//   4. For each user, obtain a user-bound OAuth access token via:
//      a. Register an agent with token-exchange grant.
//      b. Log the user in to get a session JWT (access_token from login).
//      c. The agent token-exchanges the user JWT → user-bound OAuth token.
//      NOTE: This requires JWT issued by JWTManager to be accepted by the
//            OAuth token-exchange endpoint. If parseSubjectJWT rejects RS256
//            (JWTManager) tokens, setup will fail and this scenario is SKIPPED.
//
// Load: N goroutines each hammer GET /api/v1/vault/{provider}/token with
//       their pre-obtained user-bound bearer.
//
// Honest: if setup fails (e.g. JWT signing mismatch), marks scenario as SKIP
//         and returns an informative error in Extra rather than faking numbers.

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/sharkauth/bench/internal/client"
	"github.com/sharkauth/bench/internal/fixtures"
	"github.com/sharkauth/bench/internal/metrics"
)

const vaultUserCount = 20 // N concurrent vault readers; keep small for write-heavy setup

// VaultReadConcurrent benchmarks vault token retrieval at N concurrent.
type VaultReadConcurrent struct {
	providerID   string
	providerName string
	bearers      []string // user-bound OAuth access tokens, one per user
	skipReason   string   // non-empty → skip Run
}

// NewVaultReadConcurrent constructs the scenario.
func NewVaultReadConcurrent() *VaultReadConcurrent {
	return &VaultReadConcurrent{
		providerName: "github",
	}
}

func (s *VaultReadConcurrent) Name() string { return "vault_read_concurrent" }

func (s *VaultReadConcurrent) Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error {
	if opts.AdminKey == "" {
		s.skipReason = "AdminKey required"
		return nil // soft skip
	}
	adminH := client.Headers{"Authorization": "Bearer " + opts.AdminKey}

	// 1. Create vault provider.
	var vpResp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	r, err := c.JSON(ctx, "POST", "/api/v1/vault/providers", map[string]any{
		"template":      "github",
		"client_id":     "bench-vault-fake-id-" + runID,
		"client_secret": "bench-vault-fake-secret-" + runID,
	}, adminH, &vpResp)
	if err != nil {
		s.skipReason = "vault provider create: " + err.Error()
		return nil
	}
	if r.Status < 200 || r.Status >= 300 {
		s.skipReason = fmt.Sprintf("vault provider create status=%d body=%s", r.Status, string(r.Body))
		return nil
	}
	s.providerID = vpResp.ID
	if vpResp.Name != "" {
		s.providerName = vpResp.Name
	}

	// 2. Register one agent for token exchange (shared for all users).
	var agentResp struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	r2, err := c.JSON(ctx, "POST", "/api/v1/agents", map[string]any{
		"name":        "bench-vault-agent-" + runID,
		"grant_types": []string{"client_credentials", tokenExchangeGrantType},
		"scopes":      []string{"vault:read"},
	}, adminH, &agentResp)
	if err != nil {
		s.skipReason = "create vault agent: " + err.Error()
		return nil
	}
	if r2.Status < 200 || r2.Status >= 300 {
		s.skipReason = fmt.Sprintf("create vault agent status=%d body=%s", r2.Status, string(r2.Body))
		return nil
	}
	agentBasic := encodeBasic(agentResp.ClientID, agentResp.ClientSecret)

	// 3. Create users, seed vault connections, obtain user-bound tokens.
	s.bearers = make([]string, 0, vaultUserCount)
	for i := 0; i < vaultUserCount; i++ {
		if ctx.Err() != nil {
			break
		}
		email := uniqueEmail("vault-read-user")

		// Create user.
		var userResp struct {
			ID string `json:"id"`
		}
		r3, err := c.JSON(ctx, "POST", "/api/v1/admin/users", map[string]any{
			"email":          email,
			"password":       validBenchPassword,
			"email_verified": true,
		}, adminH, &userResp)
		if err != nil || r3.Status < 200 || r3.Status >= 300 {
			// Best-effort: skip this user.
			continue
		}
		userID := userResp.ID

		// Seed demo vault connection.
		r4, err := c.JSON(ctx, "POST", "/api/v1/admin/vault/connections/_seed_demo", map[string]any{
			"user_id":     userID,
			"provider_id": s.providerID,
			"scopes":      []string{"repo"},
		}, adminH, nil)
		if err != nil || r4.Status < 200 || r4.Status >= 300 {
			continue
		}

		// Login to get JWT (access_token).
		var loginResp struct {
			AccessToken string `json:"access_token"`
			Token       string `json:"token"` // session mode
		}
		r5, err := c.JSON(ctx, "POST", "/api/v1/auth/login", map[string]any{
			"email":    email,
			"password": validBenchPassword,
		}, nil, &loginResp)
		if err != nil || r5.Status < 200 || r5.Status >= 300 {
			continue
		}
		userJWT := loginResp.AccessToken
		if userJWT == "" {
			userJWT = loginResp.Token
		}
		if userJWT == "" {
			// JWT mode not enabled — can't get user-bound token via this path.
			if i == 0 {
				s.skipReason = "login did not return JWT (JWT mode may be disabled); vault_read_concurrent requires user-bound bearer — SKIP"
				return nil
			}
			continue
		}

		// Token exchange: agent exchanges user's JWT → user-bound OAuth token.
		form := url.Values{}
		form.Set("grant_type", tokenExchangeGrantType)
		form.Set("subject_token", userJWT)
		form.Set("subject_token_type", tokenTypeAccessToken)
		form.Set("scope", "vault:read")
		h := client.Headers{
			"Authorization": "Basic " + agentBasic,
			"Content-Type":  "application/x-www-form-urlencoded",
		}
		var exchResp struct {
			AccessToken string `json:"access_token"`
		}
		r6, err := c.JSON(ctx, "POST", "/oauth/token", []byte(form.Encode()), h, &exchResp)
		if err != nil || r6.Status != 200 || exchResp.AccessToken == "" {
			if i == 0 {
				// First user failed — likely a JWT signing mismatch (RS256 vs ES256).
				body := ""
				if r6 != nil {
					body = string(r6.Body)
				}
				s.skipReason = fmt.Sprintf("token exchange failed (JWT signing mismatch likely) status=%v body=%s — vault_read_concurrent SKIP", r6.Status, body)
				return nil
			}
			continue
		}
		s.bearers = append(s.bearers, exchResp.AccessToken)
	}

	if len(s.bearers) == 0 {
		s.skipReason = "no user-bound bearers obtained — vault_read_concurrent SKIP"
	}
	return nil
}

func (s *VaultReadConcurrent) Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result {
	if s.skipReason != "" {
		return Result{
			Name:   s.Name(),
			Errors: 1,
			Extra:  map[string]any{"skip": true, "reason": s.skipReason},
		}
	}

	hist := metrics.New()
	ok := metrics.NewCounter()
	errs := metrics.NewCounter()

	path := "/api/v1/vault/" + s.providerName + "/token"
	bearers := s.bearers
	n := len(bearers)

	var mu sync.Mutex
	workerFn := func(workerID int) (bool, time.Duration) {
		mu.Lock()
		bearer := bearers[workerID%n]
		mu.Unlock()
		h := client.Headers{"Authorization": "Bearer " + bearer}
		start := time.Now()
		r, err := c.Get(ctx, path, h)
		lat := time.Since(start)
		if err != nil {
			return false, lat
		}
		return r.Status >= 200 && r.Status < 300, lat
	}

	conc := opts.Concurrency
	if conc > n {
		conc = n
	}
	start := time.Now()
	runWorkers(conc, opts.Duration, hist, ok, errs, workerFn)
	dur := time.Since(start)

	extra := map[string]any{
		"vault_provider":   s.providerName,
		"vault_users":      n,
		"effective_concurrency": conc,
	}
	return finalize(s.Name(), hist, ok, errs, dur, extra)
}

func (s *VaultReadConcurrent) Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error {
	return nil
}

// encodeBasic returns base64(clientID:clientSecret) for HTTP Basic auth.
func encodeBasic(clientID, clientSecret string) string {
	return base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
}
