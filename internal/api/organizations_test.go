package api_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

func loginFreshUser(t *testing.T, ts *testutil.TestServer, email string) string {
	t.Helper()
	uid := ts.SignupAndVerify(email, "Hunter2Hunter2", "")
	resp := ts.PostJSON("/api/v1/auth/login", map[string]string{
		"email": email, "password": "Hunter2Hunter2",
	})
	resp.Body.Close()
	return uid
}

func TestCreateOrganizationAndListMine(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_ = loginFreshUser(t, ts, "founder@x.io")

	badResp := ts.PostJSON("/api/v1/organizations", map[string]string{
		"name": "Acme", "slug": "Bad Slug!",
	})
	if badResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad slug: %d", badResp.StatusCode)
	}

	resp := ts.PostJSON("/api/v1/organizations", map[string]string{
		"name": "Acme", "slug": "acme",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d", resp.StatusCode)
	}
	var org struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	ts.DecodeJSON(resp, &org)
	if !strings.HasPrefix(org.ID, "org_") || org.Slug != "acme" {
		t.Fatalf("bad response: %+v", org)
	}

	dupResp := ts.PostJSON("/api/v1/organizations", map[string]string{
		"name": "Acme2", "slug": "acme",
	})
	if dupResp.StatusCode != http.StatusConflict {
		t.Fatalf("dup slug: %d", dupResp.StatusCode)
	}

	listResp := ts.Get("/api/v1/organizations")
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	ts.DecodeJSON(listResp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("list: %d", len(list.Data))
	}
}

func TestOrganizationNonMemberGet404(t *testing.T) {
	ts := testutil.NewTestServer(t)

	_ = loginFreshUser(t, ts, "owner@x.io")
	createResp := ts.PostJSON("/api/v1/organizations", map[string]string{
		"name": "Private", "slug": "private",
	})
	var org struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(createResp, &org)

	ts.PostJSON("/api/v1/auth/logout", nil).Body.Close()
	_ = loginFreshUser(t, ts, "stranger@x.io")

	resp := ts.Get("/api/v1/organizations/" + org.ID)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for non-member, got %d", resp.StatusCode)
	}
}

func TestOrgOwnerCannotBeLastRemoved(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ownerID := loginFreshUser(t, ts, "lastowner@x.io")

	createResp := ts.PostJSON("/api/v1/organizations", map[string]string{
		"name": "Solo", "slug": "solo",
	})
	var org struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(createResp, &org)

	demoteResp := ts.PatchJSON("/api/v1/organizations/"+org.ID+"/members/"+ownerID,
		map[string]string{"role": "member"})
	if demoteResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 last_owner on demote, got %d", demoteResp.StatusCode)
	}

	removeResp := ts.Delete("/api/v1/organizations/" + org.ID + "/members/" + ownerID)
	if removeResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 last_owner on remove, got %d", removeResp.StatusCode)
	}
}

func TestOrgInvitationFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_ = loginFreshUser(t, ts, "boss@x.io")

	createResp := ts.PostJSON("/api/v1/organizations", map[string]string{
		"name": "Team", "slug": "team",
	})
	var org struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(createResp, &org)

	inviteResp := ts.PostJSON("/api/v1/organizations/"+org.ID+"/invitations", map[string]string{
		"email": "recruit@x.io", "role": "member",
	})
	if inviteResp.StatusCode != http.StatusCreated {
		t.Fatalf("invite: %d", inviteResp.StatusCode)
	}

	if ts.EmailSender == nil {
		t.Skip("memory email sender not wired; invitation email not captured")
	}
	// Wait briefly for the fire-and-forget email goroutine.
	time.Sleep(100 * time.Millisecond)
	messages := ts.EmailSender.MessagesTo("recruit@x.io")
	if len(messages) == 0 {
		t.Fatal("expected invitation email captured")
	}

	var rawToken string
	for _, msg := range messages {
		idx := strings.Index(msg.HTML, "/organizations/invitations/")
		if idx < 0 {
			continue
		}
		tail := msg.HTML[idx+len("/organizations/invitations/"):]
		end := strings.Index(tail, "/accept")
		if end > 0 {
			rawToken = tail[:end]
			break
		}
	}
	if rawToken == "" {
		t.Fatal("could not extract invitation token from email HTML")
	}

	ts.PostJSON("/api/v1/auth/logout", nil).Body.Close()
	_ = loginFreshUser(t, ts, "recruit@x.io")

	acceptResp := ts.PostJSON("/api/v1/organizations/invitations/"+rawToken+"/accept", nil)
	if acceptResp.StatusCode != http.StatusOK {
		t.Fatalf("accept: %d", acceptResp.StatusCode)
	}

	listResp := ts.Get("/api/v1/organizations")
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	ts.DecodeJSON(listResp, &list)
	if len(list.Data) != 1 || list.Data[0].ID != org.ID {
		t.Fatalf("recruit orgs: %+v", list)
	}
}

func TestOrgInvitationEmailMismatchRejected(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	seedUserForOrgAPI(t, ts.Store, "usr_boss", "boss@x.io")
	if err := ts.Store.CreateOrganization(ctx, &storage.Organization{
		ID: "org_t", Name: "T", Slug: "t", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	_ = ts.Store.CreateOrganizationMember(ctx, &storage.OrganizationMember{
		OrganizationID: "org_t", UserID: "usr_boss",
		Role: storage.OrgRoleOwner, JoinedAt: now,
	})
	rawToken := "a" + strings.Repeat("b", 63)
	tokenHash := sha256HexHelper(rawToken)
	invitedBy := "usr_boss"
	_ = ts.Store.CreateOrganizationInvitation(ctx, &storage.OrganizationInvitation{
		ID: "inv_t", OrganizationID: "org_t",
		Email: "target@x.io", Role: storage.OrgRoleMember, TokenHash: tokenHash,
		InvitedBy: &invitedBy,
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		CreatedAt: now,
	})

	_ = loginFreshUser(t, ts, "someone-else@x.io")
	resp := ts.PostJSON("/api/v1/organizations/invitations/"+rawToken+"/accept", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 email mismatch, got %d", resp.StatusCode)
	}
}

func seedUserForOrgAPI(t *testing.T, store storage.Store, id, email string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := store.CreateUser(context.Background(), &storage.User{
		ID: id, Email: email, HashType: "argon2id", Metadata: "{}",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed user %s: %v", id, err)
	}
}

// sha256HexHelper mirrors hashInvitationToken in organization_handlers.go
// so tests can craft invitations directly.
func sha256HexHelper(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
