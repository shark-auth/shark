//go:build integration

package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// TestOrgRBAC_FullFlow exercises the complete custom org-role lifecycle:
//  1. Create org → 3 builtin roles seeded automatically
//  2. Create custom role `editor`
//  3. AttachOrgPermission(editor, "org", "update")
//  4. GrantOrgRole(userB, editor)
//  5. HasOrgPermission(userB, orgID, "org", "update") → true
//  6. PATCH org as userB → 200
//  7. RevokeOrgRole(userB, editor)
//  8. PATCH org as userB → 403
func TestOrgRBAC_FullFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	// ── Step 1: userA creates an org (seeds 3 builtin roles + grants owner to userA) ──
	userAID := loginFreshUser(t, ts, "owner-rbac@x.io")
	createResp := ts.PostJSON("/api/v1/organizations", map[string]string{
		"name": "RBACCorp", "slug": "rbac-corp",
	})
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create org: expected 201, got %d", createResp.StatusCode)
	}
	var org struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(createResp, &org)
	orgID := org.ID

	// Verify 3 builtin roles exist.
	store := ts.Store
	roles, err := store.GetOrgRolesByOrgID(ctx, orgID)
	if err != nil {
		t.Fatalf("GetOrgRolesByOrgID: %v", err)
	}
	if len(roles) != 3 {
		t.Fatalf("expected 3 builtin roles after org creation, got %d", len(roles))
	}

	// ── Step 2: userA creates a custom role `editor` ──
	createRoleResp := ts.PostJSON("/api/v1/organizations/"+orgID+"/roles", map[string]string{
		"name":        "editor",
		"description": "Can edit org settings",
	})
	if createRoleResp.StatusCode != http.StatusCreated {
		t.Fatalf("create custom role: expected 201, got %d", createRoleResp.StatusCode)
	}
	var editorRole struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(createRoleResp, &editorRole)
	editorRoleID := editorRole.ID

	if editorRoleID == "" {
		t.Fatal("editor role ID is empty")
	}

	// ── Step 3: AttachOrgPermission(editor, "org", "update") ──
	rbacMgr := rbac.NewRBACManager(store)
	if err := rbacMgr.AttachOrgPermission(ctx, editorRoleID, "org", "update"); err != nil {
		t.Fatalf("AttachOrgPermission: %v", err)
	}

	// ── Step 4: Create userB and add them to the org, grant editor role ──
	now := time.Now().UTC().Format(time.RFC3339)
	userBID := loginFreshUser(t, ts, "editor-rbac@x.io")

	// Add userB as member
	if err := store.CreateOrganizationMember(ctx, &storage.OrganizationMember{
		OrganizationID: orgID,
		UserID:         userBID,
		Role:           storage.OrgRoleMember,
		JoinedAt:       now,
	}); err != nil {
		t.Fatalf("add userB to org: %v", err)
	}

	// Grant editor role to userB (via rbacMgr)
	if err := rbacMgr.GrantOrgRole(ctx, orgID, userBID, editorRoleID, userAID); err != nil {
		t.Fatalf("GrantOrgRole: %v", err)
	}

	// ── Step 5: HasOrgPermission(userB, orgID, "org", "update") → true ──
	ok, err := rbacMgr.HasOrgPermission(ctx, userBID, orgID, "org", "update")
	if err != nil {
		t.Fatalf("HasOrgPermission: %v", err)
	}
	if !ok {
		t.Fatal("expected userB to have org:update after editor role grant, got false")
	}

	// ── Step 6: PATCH org as userB → 200 ──
	// userB is already the active session from loginFreshUser above.
	patchResp := ts.PatchJSON("/api/v1/organizations/"+orgID, map[string]string{
		"name": "RBACCorp Updated",
	})
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH org as editor: expected 200, got %d", patchResp.StatusCode)
	}
	patchResp.Body.Close()

	// ── Step 7: RevokeOrgRole(userB, editor) ──
	if err := rbacMgr.RevokeOrgRole(ctx, orgID, userBID, editorRoleID); err != nil {
		t.Fatalf("RevokeOrgRole: %v", err)
	}

	// ── Step 8: PATCH org as userB → 403 ──
	patchResp2 := ts.PatchJSON("/api/v1/organizations/"+orgID, map[string]string{
		"name": "RBACCorp Denied",
	})
	if patchResp2.StatusCode != http.StatusForbidden && patchResp2.StatusCode != http.StatusNotFound {
		t.Fatalf("PATCH org after revoke: expected 403 or 404, got %d", patchResp2.StatusCode)
	}
	patchResp2.Body.Close()
}
