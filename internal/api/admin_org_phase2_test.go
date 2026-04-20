package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// ---- GET /admin/organizations/{id}/roles ----

func TestAdminListOrgRoles_Empty(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_re", Name: "RolesEmpty", Slug: "roles-empty", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/organizations/org_re/roles")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Roles []any `json:"roles"`
	}
	ts.DecodeJSON(resp, &body)
	if body.Roles == nil {
		t.Error("roles field should be non-nil (empty array expected)")
	}
	if len(body.Roles) != 0 {
		t.Errorf("expected 0 roles, got %d", len(body.Roles))
	}
}

func TestAdminListOrgRoles_Populated(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// Org A: 2 custom roles.
	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_ra", Name: "OrgA", Slug: "org-roles-a", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ts.Store.CreateOrgRole(ctx, "org_ra", "role_ra1", "editor", "Editor role", false); err != nil {
		t.Fatal(err)
	}
	if err := ts.Store.CreateOrgRole(ctx, "org_ra", "role_ra2", "viewer", "Viewer role", false); err != nil {
		t.Fatal(err)
	}

	// Org B: 1 custom role — should not bleed into org A results.
	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_rb", Name: "OrgB", Slug: "org-roles-b", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ts.Store.CreateOrgRole(ctx, "org_rb", "role_rb1", "other", "Other org role", false); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/organizations/org_ra/roles")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Roles []struct {
			ID    string `json:"id"`
			OrgID string `json:"organization_id"`
		} `json:"roles"`
	}
	ts.DecodeJSON(resp, &body)
	if len(body.Roles) != 2 {
		t.Fatalf("expected 2 roles for org_ra, got %d", len(body.Roles))
	}
	for _, role := range body.Roles {
		if role.OrgID != "org_ra" {
			t.Errorf("role %s belongs to wrong org %s (bleed)", role.ID, role.OrgID)
		}
	}
}

func TestAdminListOrgRoles_RequiresAdminKey(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_rk", Name: "KeyCheck", Slug: "roles-key-check", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	resp := ts.Get("/api/v1/admin/organizations/org_rk/roles")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAdminListOrgRoles_UnknownOrg(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.GetWithAdminKey("/api/v1/admin/organizations/org_does_not_exist/roles")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---- GET /admin/organizations/{id}/invitations ----

func TestAdminListOrgInvitations_Empty(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_ie", Name: "InvEmpty", Slug: "inv-empty", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/organizations/org_ie/invitations")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Invitations []any `json:"invitations"`
	}
	ts.DecodeJSON(resp, &body)
	if body.Invitations == nil {
		t.Error("invitations field should be non-nil (empty array expected)")
	}
	if len(body.Invitations) != 0 {
		t.Errorf("expected 0 invitations, got %d", len(body.Invitations))
	}
}

func TestAdminListOrgInvitations_FiltersExpired(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_if", Name: "InvFilter", Slug: "inv-filter", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Pending invitation — should appear.
	if err := ts.Store.CreateOrganizationInvitation(ctx, &storage.OrganizationInvitation{
		ID: "inv_pending", OrganizationID: "org_if",
		Email: "pending@x.io", Role: storage.OrgRoleMember,
		TokenHash: "hash_pending",
		ExpiresAt: time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339),
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Expired invitation — should be filtered out.
	if err := ts.Store.CreateOrganizationInvitation(ctx, &storage.OrganizationInvitation{
		ID: "inv_expired", OrganizationID: "org_if",
		Email: "expired@x.io", Role: storage.OrgRoleMember,
		TokenHash: "hash_expired",
		ExpiresAt: time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Accepted invitation — should be filtered out.
	if err := ts.Store.CreateOrganizationInvitation(ctx, &storage.OrganizationInvitation{
		ID: "inv_accepted", OrganizationID: "org_if",
		Email: "accepted@x.io", Role: storage.OrgRoleMember,
		TokenHash: "hash_accepted",
		ExpiresAt: time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339),
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ts.Store.MarkOrganizationInvitationAccepted(ctx, "inv_accepted", now); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/organizations/org_if/invitations")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Invitations []struct {
			ID string `json:"id"`
		} `json:"invitations"`
	}
	ts.DecodeJSON(resp, &body)
	if len(body.Invitations) != 1 {
		t.Fatalf("expected 1 pending invitation, got %d", len(body.Invitations))
	}
	if body.Invitations[0].ID != "inv_pending" {
		t.Errorf("expected inv_pending, got %s", body.Invitations[0].ID)
	}
}

func TestAdminListOrgInvitations_RequiresAdminKey(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_ik", Name: "InvKeyCheck", Slug: "inv-key-check", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	resp := ts.Get("/api/v1/admin/organizations/org_ik/invitations")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}
