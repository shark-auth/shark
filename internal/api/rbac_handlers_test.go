package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/shark-auth/shark/internal/testutil"
)

// TestAuthCheckIDORPrevention is the regression suite for the cross-tenant IDOR
// fix in handleAuthCheck. Three cases:
//
//  1. Session caller supplies a foreign user_id in the body â†’ silently clamped to
//     caller's own ID (silent-ignore behaviour). The response reflects the caller's
//     permissions, not the victim's.
//
//  2. Admin key caller supplies a foreign user_id â†’ accepted as-is (legit
//     backend-to-backend cross-user check).
//
//  3. Session caller omits user_id â†’ evaluates caller's own permissions.
func TestAuthCheckIDORPrevention(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	// --- Shared fixtures -------------------------------------------------------

	// Role + permission that userA will hold, userB will not.
	role := testutil.CreateRole(t, ts.Store, "idor-test-role")
	perm := testutil.CreatePermission(t, ts.Store, "read", "idor-resource")
	if err := ts.Store.AttachPermissionToRole(ctx, role.ID, perm.ID); err != nil {
		t.Fatalf("attach perm to role: %v", err)
	}

	userA := testutil.CreateUser(t, ts.Store, "idor-usera@example.com", nil)
	userB := testutil.CreateUser(t, ts.Store, "idor-userb@example.com", nil)

	// Assign role only to userA.
	if err := ts.Store.AssignRoleToUser(ctx, userA.ID, role.ID); err != nil {
		t.Fatalf("assign role to userA: %v", err)
	}

	// Mint a JWT for userA (session path).
	sessA := testutil.CreateSession(t, ts.Store, userA.ID)
	accessToken, _, err := ts.APIServer.JWTManager.IssueAccessRefreshPair(ctx, userA, sessA.ID, false)
	if err != nil {
		t.Fatalf("mint JWT for userA: %v", err)
	}

	// --- Case 1: Session caller cannot probe another user's permissions ---------
	t.Run("session_path_user_id_clamped_to_caller", func(t *testing.T) {
		// userA is logged in and supplies userB's ID in the body.
		// Handler must silently clamp to userA â€” result is allowed=true (userA HAS
		// the perm), NOT whatever userB would return (userB does NOT have the perm).
		// If IDOR were present the result for userB would be allowed=false, which is
		// the wrong result evaluated for the wrong user.
		resp := ts.PostJSONWithBearer("/api/v1/auth/check", map[string]string{
			"user_id":  userB.ID, // attacker-supplied foreign ID
			"action":   "read",
			"resource": "idor-resource",
		}, accessToken)
		if resp.StatusCode != http.StatusOK {
			body := readBody(t, resp)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}
		var result map[string]interface{}
		ts.DecodeJSON(resp, &result)
		// Must return userA's result (allowed=true), not userB's (allowed=false).
		if result["allowed"] != true {
			t.Fatalf("IDOR: expected allowed=true (userA's result), got %v â€” body user_id was not clamped", result["allowed"])
		}
	})

	// --- Case 2: Admin key can still query any user ----------------------------
	t.Run("admin_path_cross_user_check_still_works", func(t *testing.T) {
		// Admin checks userA's permissions â€” should be allowed=true.
		resp := ts.PostJSONWithAdminKey("/api/v1/auth/check", map[string]string{
			"user_id":  userA.ID,
			"action":   "read",
			"resource": "idor-resource",
		})
		if resp.StatusCode != http.StatusOK {
			body := readBody(t, resp)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}
		var result map[string]interface{}
		ts.DecodeJSON(resp, &result)
		if result["allowed"] != true {
			t.Fatalf("admin cross-user check broken: expected allowed=true for userA, got %v", result["allowed"])
		}

		// Admin checks userB's permissions â€” should be allowed=false.
		resp2 := ts.PostJSONWithAdminKey("/api/v1/auth/check", map[string]string{
			"user_id":  userB.ID,
			"action":   "read",
			"resource": "idor-resource",
		})
		if resp2.StatusCode != http.StatusOK {
			body2 := readBody(t, resp2)
			t.Fatalf("expected 200, got %d: %s", resp2.StatusCode, body2)
		}
		var result2 map[string]interface{}
		ts.DecodeJSON(resp2, &result2)
		if result2["allowed"] != false {
			t.Fatalf("admin cross-user check broken: expected allowed=false for userB, got %v", result2["allowed"])
		}
	})

	// --- Case 3: Session caller without body user_id checks own permissions ----
	t.Run("session_path_no_user_id_uses_caller", func(t *testing.T) {
		resp := ts.PostJSONWithBearer("/api/v1/auth/check", map[string]string{
			"action":   "read",
			"resource": "idor-resource",
			// no user_id field
		}, accessToken)
		if resp.StatusCode != http.StatusOK {
			body := readBody(t, resp)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}
		var result map[string]interface{}
		ts.DecodeJSON(resp, &result)
		if result["allowed"] != true {
			t.Fatalf("expected allowed=true for session caller's own perm check, got %v", result["allowed"])
		}
	})
}

func TestRBACIntegration(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	// 1. Create a role via API
	resp := ts.PostJSONWithAdminKey("/api/v1/roles", map[string]string{
		"name":        "editor",
		"description": "Can edit content",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201 creating role, got %d: %s", resp.StatusCode, body)
	}
	var roleResult map[string]interface{}
	ts.DecodeJSON(resp, &roleResult)
	roleID, ok := roleResult["id"].(string)
	if !ok || roleID == "" {
		t.Fatal("expected role ID in response")
	}

	// 2. Create a permission via API
	resp = ts.PostJSONWithAdminKey("/api/v1/permissions", map[string]interface{}{
		"action":   "write",
		"resource": "articles",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201 creating permission, got %d: %s", resp.StatusCode, body)
	}
	var permResult map[string]interface{}
	ts.DecodeJSON(resp, &permResult)
	permID, ok := permResult["id"].(string)
	if !ok || permID == "" {
		t.Fatal("expected permission ID in response")
	}

	// 3. Attach permission to role via API
	resp = ts.PostJSONWithAdminKey("/api/v1/roles/"+roleID+"/permissions", map[string]string{
		"permission_id": permID,
	})
	if resp.StatusCode != http.StatusNoContent {
		body := readBody(t, resp)
		t.Fatalf("expected 204 attaching permission, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 4. Create a test user
	user := testutil.CreateUser(t, ts.Store, "rbac-test@example.com", nil)

	// 5. Assign role to user via API
	resp = ts.PostJSONWithAdminKey("/api/v1/users/"+user.ID+"/roles", map[string]string{
		"role_id": roleID,
	})
	if resp.StatusCode != http.StatusNoContent {
		body := readBody(t, resp)
		t.Fatalf("expected 204 assigning role, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 6. Verify /auth/check returns allowed=true
	resp = ts.PostJSONWithAdminKey("/api/v1/auth/check", map[string]string{
		"user_id":  user.ID,
		"action":   "write",
		"resource": "articles",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /auth/check, got %d: %s", resp.StatusCode, body)
	}
	var checkResult map[string]interface{}
	ts.DecodeJSON(resp, &checkResult)
	if checkResult["allowed"] != true {
		t.Fatalf("expected allowed=true, got %v", checkResult["allowed"])
	}

	// 7. Verify unassigned user gets allowed=false
	unassigned := testutil.CreateUser(t, ts.Store, "no-role@example.com", nil)
	resp = ts.PostJSONWithAdminKey("/api/v1/auth/check", map[string]string{
		"user_id":  unassigned.ID,
		"action":   "write",
		"resource": "articles",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /auth/check (unassigned), got %d: %s", resp.StatusCode, body)
	}
	ts.DecodeJSON(resp, &checkResult)
	if checkResult["allowed"] != false {
		t.Fatalf("expected allowed=false for unassigned user, got %v", checkResult["allowed"])
	}

	// 8. List user roles
	resp = ts.GetWithAdminKey("/api/v1/users/" + user.ID + "/roles")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 listing user roles, got %d: %s", resp.StatusCode, body)
	}
	var rolesResult []map[string]interface{}
	ts.DecodeJSON(resp, &rolesResult)
	if len(rolesResult) != 1 {
		t.Fatalf("expected 1 role, got %d", len(rolesResult))
	}
	if rolesResult[0]["name"] != "editor" {
		t.Fatalf("expected role name 'editor', got %v", rolesResult[0]["name"])
	}

	// 9. List user effective permissions
	resp = ts.GetWithAdminKey("/api/v1/users/" + user.ID + "/permissions")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 listing user permissions, got %d: %s", resp.StatusCode, body)
	}
	var permsResult []map[string]interface{}
	ts.DecodeJSON(resp, &permsResult)
	if len(permsResult) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(permsResult))
	}
	if permsResult[0]["action"] != "write" || permsResult[0]["resource"] != "articles" {
		t.Fatalf("expected write/articles permission, got %v/%v", permsResult[0]["action"], permsResult[0]["resource"])
	}

	// 10. Get role with permissions
	resp = ts.GetWithAdminKey("/api/v1/roles/" + roleID)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 getting role, got %d: %s", resp.StatusCode, body)
	}
	var roleDetail map[string]interface{}
	ts.DecodeJSON(resp, &roleDetail)
	if roleDetail["name"] != "editor" {
		t.Fatalf("expected role name 'editor', got %v", roleDetail["name"])
	}
	rolePerms, ok := roleDetail["permissions"].([]interface{})
	if !ok || len(rolePerms) != 1 {
		t.Fatalf("expected 1 permission in role detail, got %v", roleDetail["permissions"])
	}

	// 11. Remove role from user, then verify denied
	resp = ts.DeleteWithAdminKey("/api/v1/users/" + user.ID + "/roles/" + roleID)
	if resp.StatusCode != http.StatusNoContent {
		body := readBody(t, resp)
		t.Fatalf("expected 204 removing role, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	resp = ts.PostJSONWithAdminKey("/api/v1/auth/check", map[string]string{
		"user_id":  user.ID,
		"action":   "write",
		"resource": "articles",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /auth/check after role removal, got %d: %s", resp.StatusCode, body)
	}
	ts.DecodeJSON(resp, &checkResult)
	if checkResult["allowed"] != false {
		t.Fatalf("expected allowed=false after role removal, got %v", checkResult["allowed"])
	}

	// Suppress unused import warning
	_ = ctx
}

func TestRBACRolesCRUD(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create role
	resp := ts.PostJSONWithAdminKey("/api/v1/roles", map[string]string{
		"name":        "test-role",
		"description": "A test role",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	var role map[string]interface{}
	ts.DecodeJSON(resp, &role)
	roleID := role["id"].(string)

	// List roles
	resp = ts.GetWithAdminKey("/api/v1/roles")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	var roles []map[string]interface{}
	ts.DecodeJSON(resp, &roles)
	if len(roles) < 1 {
		t.Fatal("expected at least 1 role")
	}

	// Update role
	resp = ts.PutJSONWithAdminKey("/api/v1/roles/"+roleID, map[string]string{
		"name":        "updated-role",
		"description": "Updated description",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	var updated map[string]interface{}
	ts.DecodeJSON(resp, &updated)
	if updated["name"] != "updated-role" {
		t.Fatalf("expected name 'updated-role', got %v", updated["name"])
	}

	// Delete role
	resp = ts.DeleteWithAdminKey("/api/v1/roles/" + roleID)
	if resp.StatusCode != http.StatusNoContent {
		body := readBody(t, resp)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Verify deleted
	resp = ts.GetWithAdminKey("/api/v1/roles/" + roleID)
	if resp.StatusCode != http.StatusNotFound {
		body := readBody(t, resp)
		t.Fatalf("expected 404 after delete, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}

func TestRBACDuplicateRole(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/roles", map[string]string{
		"name": "dupe-role",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	resp = ts.PostJSONWithAdminKey("/api/v1/roles", map[string]string{
		"name": "dupe-role",
	})
	if resp.StatusCode != http.StatusConflict {
		body := readBody(t, resp)
		t.Fatalf("expected 409 for duplicate, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}

func TestRBACAdminKeyRequired(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Request without admin key should fail
	resp := ts.PostJSON("/api/v1/roles", map[string]string{
		"name": "sneaky",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		body := readBody(t, resp)
		t.Fatalf("expected 401 without admin key, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}
