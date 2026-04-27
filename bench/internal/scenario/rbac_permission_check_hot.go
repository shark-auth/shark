package scenario

// rbac_permission_check_hot — E1 MARQUEE
//
// Setup: create 1 user (admin API), 1 role with 1 permission, assign role to user.
// Load: hammer POST /api/v1/auth/check with that user/action/resource combo.
// Pure read path — SQLite read concurrency allows high parallelism.
// Reports RPS and p99 latency at configured concurrency.

import (
	"context"
	"fmt"
	"time"

	"github.com/sharkauth/bench/internal/client"
	"github.com/sharkauth/bench/internal/fixtures"
	"github.com/sharkauth/bench/internal/metrics"
)

// RBACPermissionCheckHot hammers the permission check hot path.
type RBACPermissionCheckHot struct {
	userID     string
	roleID     string
	permID     string
	action     string
	resource   string
}

// NewRBACPermissionCheckHot constructs the scenario.
func NewRBACPermissionCheckHot() *RBACPermissionCheckHot {
	// Use runID to ensure unique action/resource per bench run.
	return &RBACPermissionCheckHot{
		action:   "read",
		resource: "bench:resource:" + runID,
	}
}

func (s *RBACPermissionCheckHot) Name() string { return "rbac_permission_check_hot" }

func (s *RBACPermissionCheckHot) Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error {
	if opts.AdminKey == "" {
		return fmt.Errorf("rbac_permission_check_hot: AdminKey required")
	}
	adminH := client.Headers{"Authorization": "Bearer " + opts.AdminKey}

	// 1. Create user.
	email := uniqueEmail("rbac-hot-user")
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
	s.userID = userResp.ID

	// 2. Create role.
	var roleResp struct {
		ID string `json:"id"`
	}
	r2, err := c.JSON(ctx, "POST", "/api/v1/roles", map[string]any{
		"name":        "bench-hot-role-" + runID,
		"description": "bench scenario role",
	}, adminH, &roleResp)
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}
	if r2.Status < 200 || r2.Status >= 300 {
		return fmt.Errorf("create role: status=%d body=%s", r2.Status, string(r2.Body))
	}
	s.roleID = roleResp.ID

	// 3. Create permission.
	var permResp struct {
		ID string `json:"id"`
	}
	r3, err := c.JSON(ctx, "POST", "/api/v1/permissions", map[string]any{
		"action":   s.action,
		"resource": s.resource,
	}, adminH, &permResp)
	if err != nil {
		return fmt.Errorf("create permission: %w", err)
	}
	if r3.Status < 200 || r3.Status >= 300 {
		return fmt.Errorf("create permission: status=%d body=%s", r3.Status, string(r3.Body))
	}
	s.permID = permResp.ID

	// 4. Attach permission to role.
	r4, err := c.JSON(ctx, "POST", "/api/v1/roles/"+s.roleID+"/permissions", map[string]any{
		"permission_id": s.permID,
	}, adminH, nil)
	if err != nil {
		return fmt.Errorf("attach permission: %w", err)
	}
	if r4.Status < 200 || r4.Status >= 300 {
		return fmt.Errorf("attach permission: status=%d body=%s", r4.Status, string(r4.Body))
	}

	// 5. Assign role to user.
	r5, err := c.JSON(ctx, "POST", "/api/v1/users/"+s.userID+"/roles", map[string]any{
		"role_id": s.roleID,
	}, adminH, nil)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	if r5.Status < 200 || r5.Status >= 300 {
		return fmt.Errorf("assign role: status=%d body=%s", r5.Status, string(r5.Body))
	}

	return nil
}

func (s *RBACPermissionCheckHot) Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result {
	if s.userID == "" {
		return Result{Name: s.Name(), Errors: 1, Extra: map[string]any{"error": "setup not run"}}
	}
	if opts.AdminKey == "" {
		return Result{Name: s.Name(), Errors: 1, Extra: map[string]any{"error": "AdminKey required"}}
	}

	adminH := client.Headers{"Authorization": "Bearer " + opts.AdminKey}
	hist := metrics.New()
	ok := metrics.NewCounter()
	errs := metrics.NewCounter()

	checkBody := map[string]any{
		"user_id":  s.userID,
		"action":   s.action,
		"resource": s.resource,
	}

	body := func(workerID int) (bool, time.Duration) {
		start := time.Now()
		r, err := c.JSON(ctx, "POST", "/api/v1/auth/check", checkBody, adminH, nil)
		lat := time.Since(start)
		if err != nil {
			return false, lat
		}
		return r.Status >= 200 && r.Status < 300, lat
	}

	start := time.Now()
	runWorkers(opts.Concurrency, opts.Duration, hist, ok, errs, body)
	dur := time.Since(start)

	extra := map[string]any{
		"user_id":  s.userID,
		"role_id":  s.roleID,
		"perm_id":  s.permID,
		"action":   s.action,
		"resource": s.resource,
	}
	return finalize(s.Name(), hist, ok, errs, dur, extra)
}

func (s *RBACPermissionCheckHot) Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error {
	return nil
}
