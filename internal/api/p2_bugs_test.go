package api_test

// Tests for the three P2 dashboard UX bug-fixes:
//   Task 1 â€” GET /api/v1/webhooks/events (TestAdminWebhookEventsEndpoint)
//   Task 2 â€” GET /api/v1/admin/vault/connections?provider_id (TestAdminVaultConnectionsByProvider)
//   Task 3 â€” GET /api/v1/admin/permissions/batch-usage (TestAdminPermissionsBatchUsage)

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/api"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// ---------------------------------------------------------------------------
// Task 1 â€” webhook events catalogue endpoint
// ---------------------------------------------------------------------------

func TestAdminWebhookEventsEndpoint(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.GetWithAdminKey("/api/v1/webhooks/events")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Events []string `json:"events"`
	}
	ts.DecodeJSON(resp, &body)

	if len(body.Events) == 0 {
		t.Fatal("expected non-empty events list")
	}

	// Must contain all entries from the server's KnownWebhookEvents map.
	for ev := range api.KnownWebhookEvents {
		if !slices.Contains(body.Events, ev) {
			t.Errorf("events list is missing %q", ev)
		}
	}

	// webhook.test must be present (was missing from COMMON_EVENTS before fix).
	if !slices.Contains(body.Events, "webhook.test") {
		t.Error("events list must include webhook.test")
	}

	// Response should be sorted for stable clients.
	sorted := make([]string, len(body.Events))
	copy(sorted, body.Events)
	slices.Sort(sorted)
	for i, ev := range sorted {
		if body.Events[i] != ev {
			t.Errorf("events not sorted: index %d got %q want %q", i, body.Events[i], ev)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 2 â€” vault connections provider_id filter
// ---------------------------------------------------------------------------

func TestAdminVaultConnectionsByProvider(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	// Seed two providers.
	p1 := &storage.VaultProvider{
		ID:          "vp_test_p1",
		Name:        "testprov1",
		DisplayName: "Test Provider 1",
		AuthURL:     "https://provider1.example.com/auth",
		TokenURL:    "https://provider1.example.com/token",
		Active:      true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	p2 := &storage.VaultProvider{
		ID:          "vp_test_p2",
		Name:        "testprov2",
		DisplayName: "Test Provider 2",
		AuthURL:     "https://provider2.example.com/auth",
		TokenURL:    "https://provider2.example.com/token",
		Active:      true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := ts.Store.CreateVaultProvider(ctx, p1); err != nil {
		t.Fatalf("seed p1: %v", err)
	}
	if err := ts.Store.CreateVaultProvider(ctx, p2); err != nil {
		t.Fatalf("seed p2: %v", err)
	}

	// Create real users (vault_connections has FK on users.id).
	ua := testutil.CreateUser(t, ts.Store, "vault_ua@test.com", nil)
	ub := testutil.CreateUser(t, ts.Store, "vault_ub@test.com", nil)
	uc := testutil.CreateUser(t, ts.Store, "vault_uc@test.com", nil)

	// Seed connections: 2 for p1, 1 for p2.
	seedConn := func(providerID, userID string) {
		c := &storage.VaultConnection{
			ID:         "vc_" + providerID[len(providerID)-2:] + "_" + userID[len(userID)-2:],
			ProviderID: providerID,
			UserID:     userID,
			Scopes:     []string{"read"},
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		if err := ts.Store.CreateVaultConnection(ctx, c); err != nil {
			t.Fatalf("seed conn prov=%s user=%s: %v", providerID, userID, err)
		}
	}
	seedConn(p1.ID, ua.ID)
	seedConn(p1.ID, ub.ID)
	seedConn(p2.ID, uc.ID)

	// Unfiltered â€” returns all 3 connections.
	all := ts.GetWithAdminKey("/api/v1/admin/vault/connections")
	if all.StatusCode != http.StatusOK {
		t.Fatalf("unfiltered: %d", all.StatusCode)
	}
	var allBody struct {
		Data  []map[string]any `json:"data"`
		Total int              `json:"total"`
	}
	ts.DecodeJSON(all, &allBody)
	if allBody.Total < 3 {
		t.Errorf("expected at least 3 connections, got %d", allBody.Total)
	}

	// Filtered to p1 â€” must return exactly 2.
	p1Resp := ts.GetWithAdminKey("/api/v1/admin/vault/connections?provider_id=" + p1.ID)
	if p1Resp.StatusCode != http.StatusOK {
		t.Fatalf("filtered p1: %d", p1Resp.StatusCode)
	}
	var p1Body struct {
		Data  []map[string]any `json:"data"`
		Total int              `json:"total"`
	}
	ts.DecodeJSON(p1Resp, &p1Body)
	if p1Body.Total != 2 {
		t.Errorf("expected 2 connections for p1, got %d", p1Body.Total)
	}
	for _, c := range p1Body.Data {
		if c["provider_id"] != p1.ID {
			t.Errorf("connection has wrong provider_id: %v", c["provider_id"])
		}
	}

	// Filtered to p2 â€” must return exactly 1.
	p2Resp := ts.GetWithAdminKey("/api/v1/admin/vault/connections?provider_id=" + p2.ID)
	if p2Resp.StatusCode != http.StatusOK {
		t.Fatalf("filtered p2: %d", p2Resp.StatusCode)
	}
	var p2Body struct {
		Data  []map[string]any `json:"data"`
		Total int              `json:"total"`
	}
	ts.DecodeJSON(p2Resp, &p2Body)
	if p2Body.Total != 1 {
		t.Errorf("expected 1 connection for p2, got %d", p2Body.Total)
	}
}

// ---------------------------------------------------------------------------
// Task 3 â€” permissions batch-usage endpoint
// ---------------------------------------------------------------------------

func TestAdminPermissionsBatchUsage(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	// Create 3 permissions directly in storage.
	now := time.Now().UTC().Format(time.RFC3339)
	perms := []*storage.Permission{
		{ID: "perm_batch_a", Action: "read", Resource: "batch_res_a", CreatedAt: now},
		{ID: "perm_batch_b", Action: "write", Resource: "batch_res_b", CreatedAt: now},
		{ID: "perm_batch_c", Action: "delete", Resource: "batch_res_c", CreatedAt: now},
	}
	for _, p := range perms {
		if err := ts.Store.CreatePermission(ctx, p); err != nil {
			t.Fatalf("seed perm %s: %v", p.ID, err)
		}
	}

	// Create 2 roles.
	roles := []*storage.Role{
		{ID: "role_batch_1", Name: "batch_role_1", CreatedAt: now, UpdatedAt: now},
		{ID: "role_batch_2", Name: "batch_role_2", CreatedAt: now, UpdatedAt: now},
	}
	for _, r := range roles {
		if err := ts.Store.CreateRole(ctx, r); err != nil {
			t.Fatalf("seed role %s: %v", r.ID, err)
		}
	}

	// Attach: role1 â†’ perm_a + perm_b; role2 â†’ perm_b + perm_c.
	attachments := [][2]string{
		{"role_batch_1", "perm_batch_a"},
		{"role_batch_1", "perm_batch_b"},
		{"role_batch_2", "perm_batch_b"},
		{"role_batch_2", "perm_batch_c"},
	}
	for _, a := range attachments {
		if err := ts.Store.AttachPermissionToRole(ctx, a[0], a[1]); err != nil {
			t.Fatalf("attach %v: %v", a, err)
		}
	}

	// Create 3 users and assign them roles.
	users := []struct{ email, roleID string }{
		{"batch_u1@example.com", "role_batch_1"},
		{"batch_u2@example.com", "role_batch_1"},
		{"batch_u3@example.com", "role_batch_2"},
	}
	for _, u := range users {
		user := testutil.CreateUser(t, ts.Store, u.email, nil)
		if err := ts.Store.AssignRoleToUser(ctx, user.ID, u.roleID); err != nil {
			t.Fatalf("assign role for %s: %v", u.email, err)
		}
	}

	// Hit the batch-usage endpoint with all 3 permission IDs.
	ids := strings.Join([]string{"perm_batch_a", "perm_batch_b", "perm_batch_c"}, ",")
	resp := ts.GetWithAdminKey("/api/v1/admin/permissions/batch-usage?ids=" + ids)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("batch-usage: %d", resp.StatusCode)
	}

	var body map[string]struct {
		Roles int `json:"roles"`
		Users int `json:"users"`
	}
	ts.DecodeJSON(resp, &body)

	// perm_a: 1 role (role1), 2 users (u1, u2).
	if body["perm_batch_a"].Roles != 1 {
		t.Errorf("perm_a roles: want 1 got %d", body["perm_batch_a"].Roles)
	}
	if body["perm_batch_a"].Users != 2 {
		t.Errorf("perm_a users: want 2 got %d", body["perm_batch_a"].Users)
	}

	// perm_b: 2 roles (role1+role2), 3 distinct users (u1, u2, u3).
	if body["perm_batch_b"].Roles != 2 {
		t.Errorf("perm_b roles: want 2 got %d", body["perm_batch_b"].Roles)
	}
	if body["perm_batch_b"].Users != 3 {
		t.Errorf("perm_b users: want 3 got %d", body["perm_batch_b"].Users)
	}

	// perm_c: 1 role (role2), 1 user (u3).
	if body["perm_batch_c"].Roles != 1 {
		t.Errorf("perm_c roles: want 1 got %d", body["perm_batch_c"].Roles)
	}
	if body["perm_batch_c"].Users != 1 {
		t.Errorf("perm_c users: want 1 got %d", body["perm_batch_c"].Users)
	}

	// Empty ids param returns empty object, not error.
	emptyResp := ts.GetWithAdminKey("/api/v1/admin/permissions/batch-usage")
	if emptyResp.StatusCode != http.StatusOK {
		t.Fatalf("empty ids: %d", emptyResp.StatusCode)
	}
}
