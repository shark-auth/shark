package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

func seedUserForOrg(t *testing.T, store *storage.SQLiteStore, id, email string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := store.CreateUser(context.Background(), &storage.User{
		ID: id, Email: email, HashType: "argon2id", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed user %s: %v", id, err)
	}
}

func TestOrganizationCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	o := &storage.Organization{
		ID: "org_acme", Name: "Acme", Slug: "acme", Metadata: `{"plan":"free"}`,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := store.CreateOrganization(ctx, o); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetOrganizationByID(ctx, "org_acme")
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "acme" {
		t.Errorf("slug: %q", got.Slug)
	}

	bySlug, err := store.GetOrganizationBySlug(ctx, "acme")
	if err != nil || bySlug.ID != "org_acme" {
		t.Errorf("by slug: %+v err=%v", bySlug, err)
	}

	got.Name = "Acme Corp"
	got.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := store.UpdateOrganization(ctx, got); err != nil {
		t.Fatal(err)
	}
	after, _ := store.GetOrganizationByID(ctx, "org_acme")
	if after.Name != "Acme Corp" {
		t.Errorf("update name: %q", after.Name)
	}

	if err := store.DeleteOrganization(ctx, "org_acme"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetOrganizationByID(ctx, "org_acme"); err == nil {
		t.Fatal("expected not-found after delete")
	}
}

func TestOrganizationMembershipAndListByUser(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	seedUserForOrg(t, store, "usr_alice", "alice@x.io")
	seedUserForOrg(t, store, "usr_bob", "bob@x.io")

	for i, slug := range []string{"acme", "initech"} {
		o := &storage.Organization{
			ID: "org_" + slug, Name: slug, Slug: slug, Metadata: "{}",
			CreatedAt: now, UpdatedAt: now,
		}
		if err := store.CreateOrganization(ctx, o); err != nil {
			t.Fatal(err)
		}
		if err := store.CreateOrganizationMember(ctx, &storage.OrganizationMember{
			OrganizationID: o.ID, UserID: "usr_alice",
			Role: storage.OrgRoleOwner, JoinedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if err := store.CreateOrganizationMember(ctx, &storage.OrganizationMember{
				OrganizationID: o.ID, UserID: "usr_bob",
				Role: storage.OrgRoleMember, JoinedAt: now,
			}); err != nil {
				t.Fatal(err)
			}
		}
	}

	alice, err := store.ListOrganizationsByUserID(ctx, "usr_alice")
	if err != nil || len(alice) != 2 {
		t.Fatalf("alice orgs: %d err=%v", len(alice), err)
	}
	bob, _ := store.ListOrganizationsByUserID(ctx, "usr_bob")
	if len(bob) != 1 || bob[0].Slug != "acme" {
		t.Fatalf("bob orgs: %+v", bob)
	}

	members, err := store.ListOrganizationMembers(ctx, "org_acme")
	if err != nil || len(members) != 2 {
		t.Fatalf("members: %d err=%v", len(members), err)
	}
	var seenBob bool
	for _, m := range members {
		if m.UserID == "usr_bob" && m.UserEmail == "bob@x.io" && m.Role == storage.OrgRoleMember {
			seenBob = true
		}
	}
	if !seenBob {
		t.Error("bob not present in joined member list")
	}

	n, _ := store.CountOrganizationMembers(ctx, "org_acme")
	if n != 2 {
		t.Errorf("count: %d", n)
	}

	owners, _ := store.CountOrganizationsByRole(ctx, "usr_alice", storage.OrgRoleOwner)
	if owners != 2 {
		t.Errorf("alice owner orgs: %d", owners)
	}
}

func TestOrganizationMemberRoleConstraint(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	seedUserForOrg(t, store, "usr_x", "x@x.io")
	if err := store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_x", Name: "x", Slug: "x", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	err := store.CreateOrganizationMember(ctx, &storage.OrganizationMember{
		OrganizationID: "org_x", UserID: "usr_x",
		Role: "superuser", JoinedAt: now, // invalid per CHECK
	})
	if err == nil {
		t.Fatal("expected CHECK constraint violation")
	}
}

func TestOrganizationInvitationLifecycle(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)
	expires := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)

	seedUserForOrg(t, store, "usr_admin", "admin@x.io")
	if err := store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_inv", Name: "Inv", Slug: "inv", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	invitedBy := "usr_admin"
	inv := &storage.OrganizationInvitation{
		ID: "inv_1", OrganizationID: "org_inv",
		Email: "new@x.io", Role: storage.OrgRoleMember,
		TokenHash: "hash_abc123", InvitedBy: &invitedBy,
		ExpiresAt: expires, CreatedAt: now,
	}
	if err := store.CreateOrganizationInvitation(ctx, inv); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetOrganizationInvitationByTokenHash(ctx, "hash_abc123")
	if err != nil || got.ID != "inv_1" || got.Email != "new@x.io" {
		t.Fatalf("get: %+v err=%v", got, err)
	}

	acceptedAt := time.Now().UTC().Format(time.RFC3339)
	if err := store.MarkOrganizationInvitationAccepted(ctx, "inv_1", acceptedAt); err != nil {
		t.Fatal(err)
	}
	after, _ := store.GetOrganizationInvitationByTokenHash(ctx, "hash_abc123")
	if after.AcceptedAt == nil || *after.AcceptedAt != acceptedAt {
		t.Error("accepted_at not set")
	}

	// Second accept is a no-op (AcceptedAt NOT NULL guard).
	if err := store.MarkOrganizationInvitationAccepted(ctx, "inv_1", "later"); err != nil {
		t.Fatal(err)
	}
	final, _ := store.GetOrganizationInvitationByTokenHash(ctx, "hash_abc123")
	if *final.AcceptedAt != acceptedAt {
		t.Errorf("accepted_at overwritten: %q", *final.AcceptedAt)
	}

	list, _ := store.ListOrganizationInvitationsByOrgID(ctx, "org_inv")
	if len(list) != 1 {
		t.Errorf("list: %d", len(list))
	}
}
