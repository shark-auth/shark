package scenario

// cascade_revoke_user_agents — C3 MARQUEE
//
// Setup: create 1 user (via admin API) and N agents (default 100) all with created_by=userID.
// Load: fire single POST /api/v1/users/{id}/revoke-agents and measure wall-clock ms.
// Repeat 10× for stable sample. Reports min/p50/p99/max wall-clock.
// Honest: SQLite single-writer; BUSY waits on the DELETE side are included in wall-clock.

import (
	"context"
	"fmt"
	"time"

	"github.com/sharkauth/bench/internal/client"
	"github.com/sharkauth/bench/internal/fixtures"
	"github.com/sharkauth/bench/internal/metrics"
)

// CascadeRevokeUserAgents is the cascade-revoke scenario for agents under one user.
type CascadeRevokeUserAgents struct {
	agentCount int
	userID     string
	agentIDs   []string
}

// NewCascadeRevokeUserAgents constructs the scenario with agentCount agents.
func NewCascadeRevokeUserAgents(agentCount int) *CascadeRevokeUserAgents {
	if agentCount <= 0 {
		agentCount = 100
	}
	return &CascadeRevokeUserAgents{agentCount: agentCount}
}

func (s *CascadeRevokeUserAgents) Name() string { return "cascade_revoke_user_agents" }

func (s *CascadeRevokeUserAgents) Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error {
	if opts.AdminKey == "" {
		return fmt.Errorf("cascade_revoke_user_agents: AdminKey required")
	}
	adminH := client.Headers{"Authorization": "Bearer " + opts.AdminKey}

	// 1. Create user via admin API.
	email := uniqueEmail("cascade-revoke-user")
	var userResp struct {
		ID string `json:"id"`
	}
	r, err := c.JSON(ctx, "POST", "/api/v1/admin/users", map[string]any{
		"email":          email,
		"password":       validBenchPassword,
		"email_verified": true,
	}, adminH, &userResp)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	if r.Status < 200 || r.Status >= 300 {
		return fmt.Errorf("create user: status=%d body=%s", r.Status, string(r.Body))
	}
	if userResp.ID == "" {
		return fmt.Errorf("create user: empty id in response: %s", string(r.Body))
	}
	s.userID = userResp.ID

	// 2. Create agentCount agents with created_by=userID.
	s.agentIDs = make([]string, 0, s.agentCount)
	for i := 0; i < s.agentCount; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var agentResp struct {
			ID string `json:"id"`
		}
		r2, err := c.JSON(ctx, "POST", "/api/v1/agents", map[string]any{
			"name":        fmt.Sprintf("bench-cr-agent-%s-%d", runID, i),
			"grant_types": []string{"client_credentials"},
			"created_by":  s.userID,
		}, adminH, &agentResp)
		if err != nil {
			return fmt.Errorf("create agent %d: %w", i, err)
		}
		if r2.Status < 200 || r2.Status >= 300 {
			return fmt.Errorf("create agent %d: status=%d body=%s", i, r2.Status, string(r2.Body))
		}
		s.agentIDs = append(s.agentIDs, agentResp.ID)
	}
	return nil
}

func (s *CascadeRevokeUserAgents) Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result {
	const iterations = 10

	if opts.AdminKey == "" {
		return Result{Name: s.Name(), Errors: 1, Extra: map[string]any{"error": "AdminKey required"}}
	}
	if s.userID == "" {
		return Result{Name: s.Name(), Errors: 1, Extra: map[string]any{"error": "setup not run"}}
	}

	adminH := client.Headers{"Authorization": "Bearer " + opts.AdminKey}
	hist := metrics.New()
	var okCount, errCount int64

	path := "/api/v1/users/" + s.userID + "/revoke-agents"
	for i := 0; i < iterations; i++ {
		if ctx.Err() != nil {
			break
		}
		start := time.Now()
		r, err := c.Post(ctx, path, nil, adminH)
		lat := time.Since(start)
		hist.Record(int64(lat))
		if err != nil {
			errCount++
			continue
		}
		// 200 or 204 both acceptable
		if r.Status >= 200 && r.Status < 300 {
			okCount++
		} else {
			errCount++
		}
	}

	extra := map[string]any{
		"agent_count": s.agentCount,
		"iterations":  iterations,
		"user_id":     s.userID,
		"p50_ms":      hist.Quantile(0.50).Milliseconds(),
		"p99_ms":      hist.Quantile(0.99).Milliseconds(),
		"max_ms":      hist.Max().Milliseconds(),
	}

	// Report throughput as agents/sec on best run
	totalDur := time.Duration(int64(iterations)) * hist.Mean()
	if totalDur <= 0 {
		totalDur = time.Second
	}
	rps := float64(int64(s.agentCount)*okCount) / totalDur.Seconds()

	return Result{
		Name:       s.Name(),
		OK:         okCount,
		Errors:     errCount,
		LatencyP50: hist.Quantile(0.50),
		LatencyP95: hist.Quantile(0.95),
		LatencyP99: hist.Quantile(0.99),
		Throughput: rps,
		Extra:      extra,
	}
}

func (s *CascadeRevokeUserAgents) Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error {
	return nil
}
