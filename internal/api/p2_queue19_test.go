package api_test

// Tests for queue #19 P2 cleanup items:
//   - TestAdminListConsentsByUser   â€” ?user_id filter on /admin/oauth/consents
//   - TestUserEffectivePermissions  â€” GET /users/{id}/permissions returns deduped flat list
//   - TestDeleteOrgMemberAsAdmin    â€” DELETE /admin/organizations/{id}/members/{uid}

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)


// ---- TestAdminListConsentsByUser ----

func TestAdminListConsentsByUser(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Seed two users.
	for _, uid := range []string{"usr_consent_a", "usr_consent_b"} {
		if err := ts.Store.CreateUser(ctx, &storage.User{
			ID: uid, Email: uid + "@test.io",
			HashType: "argon2id", Metadata: "{}",
			CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Seed one consent per user (different client_id so the id can differ).
	consentA := &storage.OAuthConsent{
		ID: "consent_test_a", UserID: "usr_consent_a",
		ClientID: "client_a", Scope: "openid profile",
		GrantedAt: now,
	}
	consentB := &storage.OAuthConsent{
		ID: "consent_test_b", UserID: "usr_consent_b",
		ClientID: "client_b", Scope: "openid email",
		GrantedAt: now,
	}
	if err := ts.Store.CreateOAuthConsent(ctx, consentA); err != nil {
		t.Fatal(err)
	}
	if err := ts.Store.CreateOAuthConsent(ctx, consentB); err != nil {
		t.Fatal(err)
	}

	// Query with ?user_id=usr_consent_a â€” should only return consentA.
	resp := ts.GetWithAdminKey("/api/v1/admin/oauth/consents?user_id=usr_consent_a")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Data  []json.RawMessage `json:"data"`
		Total int               `json:"total"`
	}
	ts.DecodeJSON(resp, &body)

	if len(body.Data) != 1 {
		t.Fatalf("expected 1 consent for usr_consent_a, got %d", len(body.Data))
	}

	// Verify the returned consent belongs to user A.
	var consent struct {
		UserID string `json:"user_id"`
		ID     string `json:"id"`
	}
	if err := json.Unmarshal(body.Data[0], &consent); err != nil {
		t.Fatal(err)
	}
	if consent.UserID != "usr_consent_a" {
		t.Errorf("expected user_id=usr_consent_a, got %q", consent.UserID)
	}
	if consent.ID != "consent_test_a" {
		t.Errorf("expected id=consent_test_a, got %q", consent.ID)
	}

	// Without filter â€” both consents returned.
	resp2 := ts.GetWithAdminKey("/api/v1/admin/oauth/consents")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("unfiltered: expected 200, got %d", resp2.StatusCode)
	}
	var body2 struct {
		Data []json.RawMessage `json:"data"`
	}
	ts.DecodeJSON(resp2, &body2)
	if len(body2.Data) < 2 {
		t.Errorf("unfiltered: expected at least 2 consents, got %d", len(body2.Data))
	}
}

// ---- TestUserEffectivePermissions ----

func TestUserEffectivePermissions(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// Seed a user.
	userID := "usr_effperm"
	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: userID, Email: "effperm@test.io",
		HashType: "argon2id", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Create 2 roles with 2 permissions each, one permission shared.
	roleA := &storage.Role{ID: "role_ep_a", Name: "ep_roleA", CreatedAt: now, UpdatedAt: now}
	roleB := &storage.Role{ID: "role_ep_b", Name: "ep_roleB", CreatedAt: now, UpdatedAt: now}
	for _, r := range []*storage.Role{roleA, roleB} {
		if err := ts.Store.CreateRole(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	// Permissions: shared (users:read), unique to A (users:write), unique to B (agents:manage).
	permShared := &storage.Permission{ID: "perm_ep_shared", Action: "users", Resource: "read", CreatedAt: now}
	permA := &storage.Permission{ID: "perm_ep_a", Action: "users", Resource: "write", CreatedAt: now}
	permB := &storage.Permission{ID: "perm_ep_b", Action: "agents", Resource: "manage", CreatedAt: now}
	for _, p := range []*storage.Permission{permShared, permA, permB} {
		if err := ts.Store.CreatePermission(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	// Attach: roleA â†’ shared + permA; roleB â†’ shared + permB.
	for _, pair := range [][2]string{
		{"role_ep_a", "perm_ep_shared"},
		{"role_ep_a", "perm_ep_a"},
		{"role_ep_b", "perm_ep_shared"},
		{"role_ep_b", "perm_ep_b"},
	} {
		if err := ts.Store.AttachPermissionToRole(ctx, pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}

	// Assign both roles to user.
	for _, roleID := range []string{"role_ep_a", "role_ep_b"} {
		if err := ts.Store.AssignRoleToUser(ctx, userID, roleID); err != nil {
			t.Fatal(err)
		}
	}

	// GET /users/{id}/permissions â€” should return 3 unique permissions.
	resp := ts.GetWithAdminKey("/api/v1/users/" + userID + "/permissions")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var perms []struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(resp, &perms)

	if len(perms) != 3 {
		t.Errorf("expected 3 unique permissions (shared deduped), got %d", len(perms))
	}

	// Verify no duplicates.
	seen := map[string]bool{}
	for _, p := range perms {
		if seen[p.ID] {
			t.Errorf("duplicate permission id %q in response", p.ID)
		}
		seen[p.ID] = true
	}
}

// ---- TestDeleteOrgMemberAsAdmin ----

func TestDeleteOrgMemberAsAdmin(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// Seed org with two members (two owners so last-owner guard doesn't block).
	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_adm_rm", Name: "AdminRemove", Slug: "admin-rm", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	for _, uid := range []string{"usr_adm_rm_owner", "usr_adm_rm_member"} {
		email := uid + "@test.io"
		role := storage.OrgRoleMember
		if uid == "usr_adm_rm_owner" {
			role = storage.OrgRoleOwner
		}
		if err := ts.Store.CreateUser(ctx, &storage.User{
			ID: uid, Email: email,
			HashType: "argon2id", Metadata: "{}",
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		if err := ts.Store.CreateOrganizationMember(ctx, &storage.OrganizationMember{
			OrganizationID: "org_adm_rm", UserID: uid, Role: role, JoinedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Verify member exists before delete.
	members, err := ts.Store.ListOrganizationMembers(ctx, "org_adm_rm")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members before delete, got %d", len(members))
	}

	// Admin DELETE the member.
	resp := ts.DeleteWithAdminKey("/api/v1/admin/organizations/org_adm_rm/members/usr_adm_rm_member")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	// Verify member was removed.
	members, err = ts.Store.ListOrganizationMembers(ctx, "org_adm_rm")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member after delete, got %d", len(members))
	}
	if members[0].UserID != "usr_adm_rm_owner" {
		t.Errorf("expected remaining member to be usr_adm_rm_owner, got %q", members[0].UserID)
	}
}
