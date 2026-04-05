package rbac_test

import (
	"context"
	"testing"

	"github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// testSetup holds a store for test helpers.
type testSetup struct {
	Store storage.Store
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, ts *testSetup) string // returns userID
		action   string
		resource string
		want     bool
	}{
		{
			name: "admin with wildcard has all permissions",
			setup: func(t *testing.T, ts *testSetup) string {
				user := testutil.CreateUser(t, ts.Store, "admin@test.com", nil)
				role := testutil.CreateRole(t, ts.Store, "admin")
				perm := testutil.CreatePermission(t, ts.Store, "*", "*")
				ts.Store.AttachPermissionToRole(context.Background(), role.ID, perm.ID)
				ts.Store.AssignRoleToUser(context.Background(), user.ID, role.ID)
				return user.ID
			},
			action:   "delete",
			resource: "users",
			want:     true,
		},
		{
			name: "user with specific role can access matching action+resource",
			setup: func(t *testing.T, ts *testSetup) string {
				user := testutil.CreateUser(t, ts.Store, "reader@test.com", nil)
				role := testutil.CreateRole(t, ts.Store, "viewer")
				perm := testutil.CreatePermission(t, ts.Store, "read", "users")
				ts.Store.AttachPermissionToRole(context.Background(), role.ID, perm.ID)
				ts.Store.AssignRoleToUser(context.Background(), user.ID, role.ID)
				return user.ID
			},
			action:   "read",
			resource: "users",
			want:     true,
		},
		{
			name: "user with multiple roles: permissions merge",
			setup: func(t *testing.T, ts *testSetup) string {
				user := testutil.CreateUser(t, ts.Store, "multi@test.com", nil)
				role1 := testutil.CreateRole(t, ts.Store, "role-a")
				role2 := testutil.CreateRole(t, ts.Store, "role-b")
				perm1 := testutil.CreatePermission(t, ts.Store, "read", "users")
				perm2 := testutil.CreatePermission(t, ts.Store, "write", "roles")
				ts.Store.AttachPermissionToRole(context.Background(), role1.ID, perm1.ID)
				ts.Store.AttachPermissionToRole(context.Background(), role2.ID, perm2.ID)
				ts.Store.AssignRoleToUser(context.Background(), user.ID, role1.ID)
				ts.Store.AssignRoleToUser(context.Background(), user.ID, role2.ID)
				return user.ID
			},
			action:   "write",
			resource: "roles",
			want:     true,
		},
		{
			name: "user with no roles: no access",
			setup: func(t *testing.T, ts *testSetup) string {
				user := testutil.CreateUser(t, ts.Store, "norole@test.com", nil)
				return user.ID
			},
			action:   "read",
			resource: "users",
			want:     false,
		},
		{
			name: "user with role but wrong resource: denied",
			setup: func(t *testing.T, ts *testSetup) string {
				user := testutil.CreateUser(t, ts.Store, "wrongres@test.com", nil)
				role := testutil.CreateRole(t, ts.Store, "limited")
				perm := testutil.CreatePermission(t, ts.Store, "read", "users")
				ts.Store.AttachPermissionToRole(context.Background(), role.ID, perm.ID)
				ts.Store.AssignRoleToUser(context.Background(), user.ID, role.ID)
				return user.ID
			},
			action:   "read",
			resource: "roles",
			want:     false,
		},
		{
			name: "wildcard action matches any action",
			setup: func(t *testing.T, ts *testSetup) string {
				user := testutil.CreateUser(t, ts.Store, "wildaction@test.com", nil)
				role := testutil.CreateRole(t, ts.Store, "all-actions")
				perm := testutil.CreatePermission(t, ts.Store, "*", "users")
				ts.Store.AttachPermissionToRole(context.Background(), role.ID, perm.ID)
				ts.Store.AssignRoleToUser(context.Background(), user.ID, role.ID)
				return user.ID
			},
			action:   "delete",
			resource: "users",
			want:     true,
		},
		{
			name: "wildcard resource matches any resource",
			setup: func(t *testing.T, ts *testSetup) string {
				user := testutil.CreateUser(t, ts.Store, "wildres@test.com", nil)
				role := testutil.CreateRole(t, ts.Store, "read-all")
				perm := testutil.CreatePermission(t, ts.Store, "read", "*")
				ts.Store.AttachPermissionToRole(context.Background(), role.ID, perm.ID)
				ts.Store.AssignRoleToUser(context.Background(), user.ID, role.ID)
				return user.ID
			},
			action:   "read",
			resource: "anything",
			want:     true,
		},
		{
			name: "wrong action denied even with correct resource",
			setup: func(t *testing.T, ts *testSetup) string {
				user := testutil.CreateUser(t, ts.Store, "wrongaction@test.com", nil)
				role := testutil.CreateRole(t, ts.Store, "reader-only")
				perm := testutil.CreatePermission(t, ts.Store, "read", "users")
				ts.Store.AttachPermissionToRole(context.Background(), role.ID, perm.ID)
				ts.Store.AssignRoleToUser(context.Background(), user.ID, role.ID)
				return user.ID
			},
			action:   "write",
			resource: "users",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := testutil.NewTestDB(t)
			ts := &testSetup{Store: store}
			manager := rbac.NewRBACManager(store)

			userID := tt.setup(t, ts)
			got, err := manager.HasPermission(context.Background(), userID, tt.action, tt.resource)
			if err != nil {
				t.Fatalf("HasPermission returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("HasPermission(%q, %q, %q) = %v, want %v", userID, tt.action, tt.resource, got, tt.want)
			}
		})
	}
}

func TestGetEffectivePermissions(t *testing.T) {
	store := testutil.NewTestDB(t)
	manager := rbac.NewRBACManager(store)
	ctx := context.Background()

	user := testutil.CreateUser(t, store, "perms@test.com", nil)
	role1 := testutil.CreateRole(t, store, "role-x")
	role2 := testutil.CreateRole(t, store, "role-y")

	perm1 := testutil.CreatePermission(t, store, "read", "users")
	perm2 := testutil.CreatePermission(t, store, "write", "users")
	perm3 := testutil.CreatePermission(t, store, "read", "roles")

	// Attach perm1 and perm2 to role1, perm1 and perm3 to role2 (perm1 is shared)
	store.AttachPermissionToRole(ctx, role1.ID, perm1.ID)
	store.AttachPermissionToRole(ctx, role1.ID, perm2.ID)
	store.AttachPermissionToRole(ctx, role2.ID, perm1.ID)
	store.AttachPermissionToRole(ctx, role2.ID, perm3.ID)

	store.AssignRoleToUser(ctx, user.ID, role1.ID)
	store.AssignRoleToUser(ctx, user.ID, role2.ID)

	perms, err := manager.GetEffectivePermissions(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetEffectivePermissions returned error: %v", err)
	}

	// Should have 3 unique permissions (perm1 de-duplicated)
	if len(perms) != 3 {
		t.Fatalf("expected 3 permissions, got %d", len(perms))
	}
}

func TestSeedDefaultRoles(t *testing.T) {
	store := testutil.NewTestDB(t)
	manager := rbac.NewRBACManager(store)
	ctx := context.Background()

	// Seed once
	if err := manager.SeedDefaultRoles(ctx); err != nil {
		t.Fatalf("SeedDefaultRoles returned error: %v", err)
	}

	// Verify admin role exists
	admin, err := store.GetRoleByName(ctx, "admin")
	if err != nil {
		t.Fatalf("admin role not found: %v", err)
	}
	if admin.Name != "admin" {
		t.Fatalf("expected role name 'admin', got %q", admin.Name)
	}

	// Verify admin has wildcard permission
	perms, err := store.GetPermissionsByRoleID(ctx, admin.ID)
	if err != nil {
		t.Fatalf("GetPermissionsByRoleID returned error: %v", err)
	}
	if len(perms) != 1 {
		t.Fatalf("expected 1 permission for admin, got %d", len(perms))
	}
	if perms[0].Action != "*" || perms[0].Resource != "*" {
		t.Fatalf("expected wildcard permission, got %s/%s", perms[0].Action, perms[0].Resource)
	}

	// Verify member role exists
	member, err := store.GetRoleByName(ctx, "member")
	if err != nil {
		t.Fatalf("member role not found: %v", err)
	}
	if member.Name != "member" {
		t.Fatalf("expected role name 'member', got %q", member.Name)
	}

	// Seed again (idempotent)
	if err := manager.SeedDefaultRoles(ctx); err != nil {
		t.Fatalf("SeedDefaultRoles (second call) returned error: %v", err)
	}

	// Should still have exactly 2 roles
	roles, err := store.ListRoles(ctx)
	if err != nil {
		t.Fatalf("ListRoles returned error: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles after double seed, got %d", len(roles))
	}
}
