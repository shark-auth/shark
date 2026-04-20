package api_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// ---- DELETE /admin/sessions (revoke-all) ----

func TestAdminRevokeAllSessions(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Seed one user and 3 active sessions.
	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: "usr_ra1", Email: "ra1@x.io", HashType: "argon2id", Metadata: "{}",
		CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"sess_ra1", "sess_ra2", "sess_ra3"} {
		if err := ts.Store.CreateSession(ctx, &storage.Session{
			ID: id, UserID: "usr_ra1", AuthMethod: "password",
			ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
			CreatedAt: now.Format(time.RFC3339),
		}); err != nil {
			t.Fatal(err)
		}
	}

	resp := ts.DeleteWithAdminKey("/api/v1/admin/sessions")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	var body struct {
		Revoked int64 `json:"revoked"`
	}
	ts.DecodeJSON(resp, &body)
	if body.Revoked != 3 {
		t.Errorf("revoked: got %d, want 3", body.Revoked)
	}

	// Verify sessions are gone.
	left, _ := ts.Store.GetSessionsByUserID(ctx, "usr_ra1")
	if len(left) != 0 {
		t.Fatalf("expected 0 sessions left, got %d", len(left))
	}

	// Verify audit log created.
	audit, err := ts.Store.QueryAuditLogs(ctx, storage.AuditLogQuery{Action: "session.revoke", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) == 0 {
		t.Error("expected at least one audit entry for revoke-all")
	}
}

func TestAdminRevokeAllSessions_RequiresAdminKey(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.Delete("/api/v1/admin/sessions")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// ---- POST /agents/{id}/rotate-secret ----

func TestAgentRotateSecret(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	// Create agent via API.
	createResp := ts.PostJSONWithAdminKey("/api/v1/agents", map[string]interface{}{
		"name":        "rotate-test-agent",
		"grant_types": []string{"client_credentials"},
	})
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent status=%d", createResp.StatusCode)
	}
	var created struct {
		ID           string `json:"id"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	ts.DecodeJSON(createResp, &created)

	// Rotate the secret.
	rotateResp := ts.PostJSONWithAdminKey("/api/v1/agents/"+created.ID+"/rotate-secret", nil)
	if rotateResp.StatusCode != http.StatusOK {
		t.Fatalf("rotate status=%d", rotateResp.StatusCode)
	}
	var rotated struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Message      string `json:"message"`
	}
	ts.DecodeJSON(rotateResp, &rotated)

	if rotated.ClientSecret == "" {
		t.Fatal("expected new plaintext secret in response")
	}
	if rotated.ClientSecret == created.ClientSecret {
		t.Error("expected new secret to differ from old secret")
	}
	if rotated.ClientID != created.ClientID {
		t.Errorf("client_id mismatch: got %q, want %q", rotated.ClientID, created.ClientID)
	}

	// Verify the new hash is stored correctly by checking old hash no longer matches.
	oldHash := sha256.Sum256([]byte(created.ClientSecret))
	newHash := sha256.Sum256([]byte(rotated.ClientSecret))
	agent, err := ts.Store.GetAgentByID(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedHash := agent.ClientSecretHash
	if storedHash == hex.EncodeToString(oldHash[:]) {
		t.Error("expected stored hash to differ from old secret hash after rotation")
	}
	if storedHash != hex.EncodeToString(newHash[:]) {
		t.Error("stored hash does not match new secret hash")
	}

	// Verify audit log for rotation.
	audit, err := ts.Store.QueryAuditLogs(ctx, storage.AuditLogQuery{Action: "agent.secret.rotated", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) == 0 {
		t.Error("expected audit entry for agent.secret.rotated")
	}
}

func TestAgentRotateSecret_RequiresAdminKey(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Seed agent directly.
	agent := &storage.Agent{
		ID: "agent_nokey", Name: "no-key-test",
		ClientID: "shark_agent_nokey", ClientSecretHash: "fakehash",
		ClientType: "confidential", AuthMethod: "client_secret_basic",
		RedirectURIs: []string{}, GrantTypes: []string{}, ResponseTypes: []string{},
		Scopes: []string{}, TokenLifetime: 3600, Metadata: map[string]any{},
		Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := ts.Store.CreateAgent(ctx, agent); err != nil {
		t.Fatal(err)
	}

	resp := ts.PostJSON("/api/v1/agents/agent_nokey/rotate-secret", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}
