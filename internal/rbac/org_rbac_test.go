package rbac_test

import (
	"context"
	"errors"
	"testing"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// createOrg creates a test organization directly via the store.
func createOrg(t *testing.T, store storage.Store) *storage.Organization {
	t.Helper()
	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	org := &storage.Organization{
		ID:        "org_" + id,
		Name:      "Test Org " + id,
		Slug:      "test-org-" + id,
		Metadata:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.CreateOrganization(context.Background(), org); err != nil {
		t.Fatalf("creating test org: %v", err)
	}
	return org
}

// createMember adds a user to an org as "member" role so FK constraints pass.
func createMember(t *testing.T, store storage.Store, orgID, userID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	m := &storage.OrganizationMember{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           storage.OrgRoleMember,
		JoinedAt:       now,
	}
	if err := store.CreateOrganizationMember(context.Background(), m); err != nil {
		t.Fatalf("creating org member: %v", err)
	}
}

// TestHasOrgPermission_Wildcard verifies that the owner role with (*,*) grants
// any permission check.
func TestHasOrgPermission_Wildcard(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := rbac.NewRBACManager(store)
	ctx := context.Background()

	org := createOrg(t, store)
	user := testutil.CreateUser(t, store, "owner@test.com", nil)
	createMember(t, store, org.ID, user.ID)

	// Seed the three builtin roles.
	if err := mgr.SeedOrgRoles(ctx, org.ID); err != nil {
		t.Fatalf("SeedOrgRoles: %v", err)
	}

	// Find the owner role and grant it to the user.
	ownerRole, err := store.GetOrgRoleByName(ctx, org.ID, "owner")
	if err != nil {
		t.Fatalf("GetOrgRoleByName(owner): %v", err)
	}
	if err := store.GrantOrgRole(ctx, org.ID, user.ID, ownerRole.ID, user.ID); err != nil {
		t.Fatalf("GrantOrgRole: %v", err)
	}

	// Owner with (*,*) should pass any permission check.
	ok, err := mgr.HasOrgPermission(ctx, user.ID, org.ID, "delete", "everything")
	if err != nil {
		t.Fatalf("HasOrgPermission returned error: %v", err)
	}
	if !ok {
		t.Error("expected owner with (*,*) to have any permission, got false")
	}

	ok, err = mgr.HasOrgPermission(ctx, user.ID, org.ID, "org", "delete")
	if err != nil {
		t.Fatalf("HasOrgPermission returned error: %v", err)
	}
	if !ok {
		t.Error("expected owner to have org:delete permission, got false")
	}
}

// TestHasOrgPermission_NoRole verifies that a user with no org roles gets ErrNotMember.
func TestHasOrgPermission_NoRole(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := rbac.NewRBACManager(store)
	ctx := context.Background()

	org := createOrg(t, store)
	user := testutil.CreateUser(t, store, "norole@test.com", nil)
	// Do NOT create member or grant any org role.

	ok, err := mgr.HasOrgPermission(ctx, user.ID, org.ID, "org", "read")
	if !errors.Is(err, rbac.ErrNotMember) {
		t.Errorf("expected ErrNotMember, got err=%v ok=%v", err, ok)
	}
	if ok {
		t.Error("expected ok=false for non-member")
	}
}

// TestHasOrgPermission_CustomRole verifies that a user with a custom role gets
// exactly the permissions attached to that role.
func TestHasOrgPermission_CustomRole(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := rbac.NewRBACManager(store)
	ctx := context.Background()

	org := createOrg(t, store)
	user := testutil.CreateUser(t, store, "editor@test.com", nil)
	createMember(t, store, org.ID, user.ID)

	// Create a custom role with a single permission.
	customRole, err := mgr.CreateOrgRole(ctx, org.ID, "editor", "Can edit content")
	if err != nil {
		t.Fatalf("CreateOrgRole: %v", err)
	}
	if err := mgr.AttachOrgPermission(ctx, customRole.ID, "org", "update"); err != nil {
		t.Fatalf("AttachOrgPermission: %v", err)
	}
	if err := store.GrantOrgRole(ctx, org.ID, user.ID, customRole.ID, user.ID); err != nil {
		t.Fatalf("GrantOrgRole: %v", err)
	}

	// Should have org:update.
	ok, err := mgr.HasOrgPermission(ctx, user.ID, org.ID, "org", "update")
	if err != nil {
		t.Fatalf("HasOrgPermission(org,update): %v", err)
	}
	if !ok {
		t.Error("expected org:update to be granted, got false")
	}

	// Should NOT have org:delete.
	ok, err = mgr.HasOrgPermission(ctx, user.ID, org.ID, "org", "delete")
	if err != nil {
		t.Fatalf("HasOrgPermission(org,delete): %v", err)
	}
	if ok {
		t.Error("expected org:delete to be denied, got true")
	}
}

// TestSeedOrgRoles_Idempotent verifies that calling SeedOrgRoles twice produces
// exactly 3 org_roles rows (no duplicates).
func TestSeedOrgRoles_Idempotent(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := rbac.NewRBACManager(store)
	ctx := context.Background()

	org := createOrg(t, store)

	if err := mgr.SeedOrgRoles(ctx, org.ID); err != nil {
		t.Fatalf("SeedOrgRoles first call: %v", err)
	}
	if err := mgr.SeedOrgRoles(ctx, org.ID); err != nil {
		t.Fatalf("SeedOrgRoles second call: %v", err)
	}

	roles, err := store.GetOrgRolesByOrgID(ctx, org.ID)
	if err != nil {
		t.Fatalf("GetOrgRolesByOrgID: %v", err)
	}
	if len(roles) != 3 {
		t.Errorf("expected exactly 3 org roles after double seed, got %d", len(roles))
	}

	// Verify all three names are present.
	names := make(map[string]bool)
	for _, r := range roles {
		names[r.Name] = true
		if !r.IsBuiltin {
			t.Errorf("expected role %q to be builtin, got IsBuiltin=false", r.Name)
		}
	}
	for _, expected := range []string{"owner", "admin", "member"} {
		if !names[expected] {
			t.Errorf("missing expected builtin role %q", expected)
		}
	}
}

// TestCreateOrgRole_DuplicateName verifies that creating two roles with the same
// name in the same org surfaces a (wrapped) error from the UNIQUE constraint.
func TestCreateOrgRole_DuplicateName(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := rbac.NewRBACManager(store)
	ctx := context.Background()

	org := createOrg(t, store)

	if _, err := mgr.CreateOrgRole(ctx, org.ID, "editor", "First"); err != nil {
		t.Fatalf("first CreateOrgRole: %v", err)
	}
	if _, err := mgr.CreateOrgRole(ctx, org.ID, "editor", "Duplicate"); err == nil {
		t.Error("expected error on duplicate role name, got nil")
	}
}

// TestDeleteOrgRole_Builtin_Refused verifies that deleting a builtin role returns
// ErrBuiltinRole.
func TestDeleteOrgRole_Builtin_Refused(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := rbac.NewRBACManager(store)
	ctx := context.Background()

	org := createOrg(t, store)
	if err := mgr.SeedOrgRoles(ctx, org.ID); err != nil {
		t.Fatalf("SeedOrgRoles: %v", err)
	}

	ownerRole, err := store.GetOrgRoleByName(ctx, org.ID, "owner")
	if err != nil {
		t.Fatalf("GetOrgRoleByName(owner): %v", err)
	}

	err = mgr.DeleteOrgRole(ctx, org.ID, ownerRole.ID)
	if !errors.Is(err, rbac.ErrBuiltinRole) {
		t.Errorf("expected ErrBuiltinRole, got %v", err)
	}
}

// TestDeleteOrgRole_Custom_OK verifies that deleting a non-builtin role succeeds.
func TestDeleteOrgRole_Custom_OK(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := rbac.NewRBACManager(store)
	ctx := context.Background()

	org := createOrg(t, store)

	customRole, err := mgr.CreateOrgRole(ctx, org.ID, "temp-role", "Temporary")
	if err != nil {
		t.Fatalf("CreateOrgRole: %v", err)
	}

	if err := mgr.DeleteOrgRole(ctx, org.ID, customRole.ID); err != nil {
		t.Fatalf("DeleteOrgRole: %v", err)
	}

	// Confirm it's gone.
	if _, err := store.GetOrgRoleByID(ctx, customRole.ID); err == nil {
		t.Error("expected error fetching deleted role, got nil")
	}
}

// TestGrantRevokeOrgRole_RoundTrip verifies the grant → check → revoke → check cycle.
func TestGrantRevokeOrgRole_RoundTrip(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := rbac.NewRBACManager(store)
	ctx := context.Background()

	org := createOrg(t, store)
	userA := testutil.CreateUser(t, store, "granter@test.com", nil)
	userB := testutil.CreateUser(t, store, "grantee@test.com", nil)
	createMember(t, store, org.ID, userA.ID)
	createMember(t, store, org.ID, userB.ID)

	// Create a custom role with a permission.
	role, err := mgr.CreateOrgRole(ctx, org.ID, "viewer", "Read-only access")
	if err != nil {
		t.Fatalf("CreateOrgRole: %v", err)
	}
	if err := mgr.AttachOrgPermission(ctx, role.ID, "org", "read"); err != nil {
		t.Fatalf("AttachOrgPermission: %v", err)
	}

	// Grant the role to userB.
	if err := mgr.GrantOrgRole(ctx, org.ID, userB.ID, role.ID, userA.ID); err != nil {
		t.Fatalf("GrantOrgRole: %v", err)
	}

	// userB should now have org:read.
	ok, err := mgr.HasOrgPermission(ctx, userB.ID, org.ID, "org", "read")
	if err != nil {
		t.Fatalf("HasOrgPermission after grant: %v", err)
	}
	if !ok {
		t.Error("expected org:read after grant, got false")
	}

	// Revoke the role.
	if err := mgr.RevokeOrgRole(ctx, org.ID, userB.ID, role.ID); err != nil {
		t.Fatalf("RevokeOrgRole: %v", err)
	}

	// userB should now get ErrNotMember (no roles left).
	_, err = mgr.HasOrgPermission(ctx, userB.ID, org.ID, "org", "read")
	if !errors.Is(err, rbac.ErrNotMember) {
		t.Errorf("expected ErrNotMember after revoke, got %v", err)
	}
}
