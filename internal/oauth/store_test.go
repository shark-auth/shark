package oauth

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ory/fosite"

	"github.com/sharkauth/sharkauth/internal/storage"
)

//go:embed testmigrations/*.sql
var testMigrationsFS embed.FS

// newTestFositeStore creates a FositeStore backed by an in-memory SQLite DB
// with all migrations applied.
func newTestFositeStore(t *testing.T) (*FositeStore, storage.Store) {
	t.Helper()
	db, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("creating test db: %v", err)
	}
	if err := storage.RunMigrations(db.DB(), testMigrationsFS, "testmigrations"); err != nil {
		db.Close()
		t.Fatalf("running migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewFositeStore(db), db
}

// seedAgent inserts a test agent and returns it.
func seedAgent(t *testing.T, store storage.Store, clientID string, public bool) *storage.Agent {
	t.Helper()

	secretHash := ""
	clientType := "confidential"
	authMethod := "client_secret_basic"
	if public {
		clientType = "public"
		authMethod = "none"
	} else {
		// SHA-256 hash of "test-secret"
		h := sha256.Sum256([]byte("test-secret"))
		secretHash = hex.EncodeToString(h[:])
	}

	agent := &storage.Agent{
		ID:            "agent_" + clientID,
		Name:          "Test Agent " + clientID,
		Description:   "A test agent",
		ClientID:      clientID,
		ClientSecretHash: secretHash,
		ClientType:    clientType,
		AuthMethod:    authMethod,
		RedirectURIs:  []string{"https://example.com/callback"},
		GrantTypes:    []string{"authorization_code", "client_credentials"},
		ResponseTypes: []string{"code"},
		Scopes:        []string{"openid", "profile"},
		TokenLifetime: 900,
		Active:        true,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := store.CreateAgent(context.Background(), agent); err != nil {
		t.Fatalf("seeding agent: %v", err)
	}
	return agent
}

// seedUser inserts a minimal test user and returns the ID.
func seedUser(t *testing.T, store storage.Store, email string) string {
	t.Helper()
	id := "user_" + strings.ReplaceAll(email, "@", "_")
	now := time.Now().UTC().Format(time.RFC3339)
	user := &storage.User{
		ID:        id,
		Email:     email,
		Metadata:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("seeding user: %v", err)
	}
	return id
}

// ---------------------------------------------------------------------------
// TestGetClient
// ---------------------------------------------------------------------------

func TestGetClient(t *testing.T) {
	fs, db := newTestFositeStore(t)
	seedAgent(t, db, "my-agent", false)

	client, err := fs.GetClient(context.Background(), "my-agent")
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}

	if client.GetID() != "my-agent" {
		t.Errorf("expected ID %q, got %q", "my-agent", client.GetID())
	}
	if client.IsPublic() {
		t.Error("expected confidential client, got public")
	}
	if len(client.GetRedirectURIs()) != 1 || client.GetRedirectURIs()[0] != "https://example.com/callback" {
		t.Errorf("unexpected redirect URIs: %v", client.GetRedirectURIs())
	}
	if len(client.GetGrantTypes()) != 2 {
		t.Errorf("expected 2 grant types, got %d", len(client.GetGrantTypes()))
	}
	if len(client.GetScopes()) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(client.GetScopes()))
	}

	// Verify hashed secret is populated.
	if len(client.GetHashedSecret()) == 0 {
		t.Error("expected hashed secret to be populated for confidential client")
	}
}

func TestGetClient_Public(t *testing.T) {
	fs, db := newTestFositeStore(t)
	seedAgent(t, db, "public-agent", true)

	client, err := fs.GetClient(context.Background(), "public-agent")
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}

	if !client.IsPublic() {
		t.Error("expected public client")
	}
}

func TestGetClient_NotFound(t *testing.T) {
	fs, _ := newTestFositeStore(t)

	_, err := fs.GetClient(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown client_id")
	}

	// Should be a fosite.ErrNotFound or wrap it.
	rfcErr, ok := err.(*fosite.RFC6749Error)
	if !ok {
		t.Fatalf("expected *fosite.RFC6749Error, got %T: %v", err, err)
	}
	if rfcErr.CodeField != 404 {
		t.Errorf("expected HTTP 404 code, got %d", rfcErr.CodeField)
	}
}

func TestGetClient_Inactive(t *testing.T) {
	fs, db := newTestFositeStore(t)
	seedAgent(t, db, "inactive-agent", false)

	// Deactivate the agent.
	if err := db.DeactivateAgent(context.Background(), "agent_inactive-agent"); err != nil {
		t.Fatalf("deactivating agent: %v", err)
	}

	_, err := fs.GetClient(context.Background(), "inactive-agent")
	if err == nil {
		t.Fatal("expected error for inactive client")
	}
}

// ---------------------------------------------------------------------------
// TestAuthCodeRoundTrip
// ---------------------------------------------------------------------------

func TestAuthCodeRoundTrip(t *testing.T) {
	fs, db := newTestFositeStore(t)
	seedAgent(t, db, "code-agent", false)
	userID := seedUser(t, db, "user@example.com")

	ctx := context.Background()
	code := "test-auth-code-signature-12345"

	session := &fosite.DefaultSession{
		Subject: userID,
		ExpiresAt: map[fosite.TokenType]time.Time{
			fosite.AuthorizeCode: time.Now().UTC().Add(10 * time.Minute),
		},
	}

	client, _ := fs.GetClient(ctx, "code-agent")

	req := &fosite.Request{
		ID:             "req-123",
		RequestedAt:    time.Now().UTC(),
		Client:         client,
		RequestedScope: fosite.Arguments{"openid", "profile"},
		GrantedScope:   fosite.Arguments{"openid", "profile"},
		Session:        session,
		Form: url.Values{
			"redirect_uri":          {"https://example.com/callback"},
			"code_challenge":        {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
			"code_challenge_method": {"S256"},
			"nonce":                 {"test-nonce"},
		},
	}

	// Create
	if err := fs.CreateAuthorizeCodeSession(ctx, code, req); err != nil {
		t.Fatalf("CreateAuthorizeCodeSession: %v", err)
	}

	// Get
	retrieved, err := fs.GetAuthorizeCodeSession(ctx, code, &fosite.DefaultSession{})
	if err != nil {
		t.Fatalf("GetAuthorizeCodeSession: %v", err)
	}

	if retrieved.GetClient().GetID() != "code-agent" {
		t.Errorf("expected client_id %q, got %q", "code-agent", retrieved.GetClient().GetID())
	}

	scopes := retrieved.GetRequestedScopes()
	if len(scopes) != 2 || scopes[0] != "openid" {
		t.Errorf("unexpected scopes: %v", scopes)
	}

	form := retrieved.GetRequestForm()
	if form.Get("code_challenge") != "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk" {
		t.Errorf("unexpected code_challenge: %q", form.Get("code_challenge"))
	}
	if form.Get("nonce") != "test-nonce" {
		t.Errorf("unexpected nonce: %q", form.Get("nonce"))
	}

	// Invalidate
	if err := fs.InvalidateAuthorizeCodeSession(ctx, code); err != nil {
		t.Fatalf("InvalidateAuthorizeCodeSession: %v", err)
	}

	// Get after invalidation should fail.
	_, err = fs.GetAuthorizeCodeSession(ctx, code, &fosite.DefaultSession{})
	if err == nil {
		t.Fatal("expected error after invalidation")
	}
}

// ---------------------------------------------------------------------------
// TestAccessTokenRoundTrip
// ---------------------------------------------------------------------------

func TestAccessTokenRoundTrip(t *testing.T) {
	fs, db := newTestFositeStore(t)
	seedAgent(t, db, "token-agent", false)
	userID := seedUser(t, db, "token@example.com")

	ctx := context.Background()
	sig := "access-token-signature-abc123"

	session := &fosite.DefaultSession{
		Subject: userID,
		ExpiresAt: map[fosite.TokenType]time.Time{
			fosite.AccessToken: time.Now().UTC().Add(time.Hour),
		},
	}

	client, _ := fs.GetClient(ctx, "token-agent")

	req := &fosite.Request{
		ID:             "access-req-1",
		RequestedAt:    time.Now().UTC(),
		Client:         client,
		RequestedScope: fosite.Arguments{"openid"},
		GrantedScope:   fosite.Arguments{"openid"},
		Session:        session,
		Form:           url.Values{},
	}

	// Create
	if err := fs.CreateAccessTokenSession(ctx, sig, req); err != nil {
		t.Fatalf("CreateAccessTokenSession: %v", err)
	}

	// Get
	retrieved, err := fs.GetAccessTokenSession(ctx, sig, &fosite.DefaultSession{})
	if err != nil {
		t.Fatalf("GetAccessTokenSession: %v", err)
	}

	if retrieved.GetClient().GetID() != "token-agent" {
		t.Errorf("expected client_id %q, got %q", "token-agent", retrieved.GetClient().GetID())
	}
	if retrieved.GetID() != "access-req-1" {
		t.Errorf("expected request ID %q, got %q", "access-req-1", retrieved.GetID())
	}

	// Delete
	if err := fs.DeleteAccessTokenSession(ctx, sig); err != nil {
		t.Fatalf("DeleteAccessTokenSession: %v", err)
	}

	// Get after delete should fail.
	_, err = fs.GetAccessTokenSession(ctx, sig, &fosite.DefaultSession{})
	if err == nil {
		t.Fatal("expected error after deleting access token")
	}
}

// ---------------------------------------------------------------------------
// TestRefreshTokenRoundTrip
// ---------------------------------------------------------------------------

func TestRefreshTokenRoundTrip(t *testing.T) {
	fs, db := newTestFositeStore(t)
	seedAgent(t, db, "refresh-agent", false)
	userID := seedUser(t, db, "refresh@example.com")

	ctx := context.Background()
	sig := "refresh-token-signature-xyz789"
	accessSig := "access-token-for-refresh"

	session := &fosite.DefaultSession{
		Subject: userID,
		ExpiresAt: map[fosite.TokenType]time.Time{
			fosite.RefreshToken: time.Now().UTC().Add(24 * time.Hour),
		},
	}

	client, _ := fs.GetClient(ctx, "refresh-agent")

	req := &fosite.Request{
		ID:             "refresh-req-1",
		RequestedAt:    time.Now().UTC(),
		Client:         client,
		RequestedScope: fosite.Arguments{"openid", "offline_access"},
		GrantedScope:   fosite.Arguments{"openid", "offline_access"},
		Session:        session,
		Form:           url.Values{},
	}

	// Create
	if err := fs.CreateRefreshTokenSession(ctx, sig, accessSig, req); err != nil {
		t.Fatalf("CreateRefreshTokenSession: %v", err)
	}

	// Get
	retrieved, err := fs.GetRefreshTokenSession(ctx, sig, &fosite.DefaultSession{})
	if err != nil {
		t.Fatalf("GetRefreshTokenSession: %v", err)
	}

	if retrieved.GetClient().GetID() != "refresh-agent" {
		t.Errorf("expected client_id %q, got %q", "refresh-agent", retrieved.GetClient().GetID())
	}

	// Revoke via RevokeRefreshToken (by request ID)
	if err := fs.RevokeRefreshToken(ctx, "refresh-req-1"); err != nil {
		t.Fatalf("RevokeRefreshToken: %v", err)
	}

	// Get after revocation should return ErrInactiveToken.
	_, err = fs.GetRefreshTokenSession(ctx, sig, &fosite.DefaultSession{})
	if err == nil {
		t.Fatal("expected error after revoking refresh token")
	}
}

// ---------------------------------------------------------------------------
// TestSHA256Hasher
// ---------------------------------------------------------------------------

func TestSHA256Hasher_HashAndCompare(t *testing.T) {
	h := &SHA256Hasher{}
	ctx := context.Background()
	data := []byte("my-client-secret")

	// Hash
	hashed, err := h.Hash(ctx, data)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	// Should be a valid hex string.
	if len(hashed) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("expected 64 char hex hash, got %d chars: %q", len(hashed), string(hashed))
	}

	// Compare should succeed with matching data.
	if err := h.Compare(ctx, hashed, data); err != nil {
		t.Fatalf("Compare with correct data: %v", err)
	}

	// Compare should fail with wrong data.
	if err := h.Compare(ctx, hashed, []byte("wrong-secret")); err == nil {
		t.Fatal("Compare should fail with wrong data")
	}
}

func TestSHA256Hasher_CompareWithPrecomputedHash(t *testing.T) {
	h := &SHA256Hasher{}
	ctx := context.Background()

	// Precompute the hash of "test-secret" (same as in seedAgent).
	sum := sha256.Sum256([]byte("test-secret"))
	hash := []byte(hex.EncodeToString(sum[:]))

	if err := h.Compare(ctx, hash, []byte("test-secret")); err != nil {
		t.Fatalf("Compare with precomputed hash: %v", err)
	}

	if err := h.Compare(ctx, hash, []byte("not-test-secret")); err == nil {
		t.Fatal("Compare should fail with wrong secret")
	}
}

// ---------------------------------------------------------------------------
// TestPKCERequestStorage
// ---------------------------------------------------------------------------

func TestPKCERequestStorage(t *testing.T) {
	fs, db := newTestFositeStore(t)
	seedAgent(t, db, "pkce-agent", true)
	userID := seedUser(t, db, "pkce@example.com")

	ctx := context.Background()
	code := "pkce-code-signature-abcdef"

	session := &fosite.DefaultSession{
		Subject: userID,
		ExpiresAt: map[fosite.TokenType]time.Time{
			fosite.AuthorizeCode: time.Now().UTC().Add(10 * time.Minute),
		},
	}

	client, _ := fs.GetClient(ctx, "pkce-agent")

	req := &fosite.Request{
		ID:             "pkce-req-1",
		RequestedAt:    time.Now().UTC(),
		Client:         client,
		RequestedScope: fosite.Arguments{"openid"},
		GrantedScope:   fosite.Arguments{"openid"},
		Session:        session,
		Form: url.Values{
			"redirect_uri":          {"https://example.com/callback"},
			"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
			"code_challenge_method": {"S256"},
		},
	}

	// First create the auth code session (which stores PKCE data).
	if err := fs.CreateAuthorizeCodeSession(ctx, code, req); err != nil {
		t.Fatalf("CreateAuthorizeCodeSession: %v", err)
	}

	// CreatePKCERequestSession is a no-op since data is in the auth code.
	if err := fs.CreatePKCERequestSession(ctx, code, req); err != nil {
		t.Fatalf("CreatePKCERequestSession: %v", err)
	}

	// GetPKCERequestSession should return the same data.
	pkceReq, err := fs.GetPKCERequestSession(ctx, code, &fosite.DefaultSession{})
	if err != nil {
		t.Fatalf("GetPKCERequestSession: %v", err)
	}

	if pkceReq.GetRequestForm().Get("code_challenge") != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Errorf("unexpected code_challenge: %q", pkceReq.GetRequestForm().Get("code_challenge"))
	}

	// DeletePKCERequestSession is also a no-op.
	if err := fs.DeletePKCERequestSession(ctx, code); err != nil {
		t.Fatalf("DeletePKCERequestSession: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestRevokeAccessToken
// ---------------------------------------------------------------------------

func TestRevokeAccessToken(t *testing.T) {
	fs, db := newTestFositeStore(t)
	seedAgent(t, db, "revoke-agent", false)
	userID := seedUser(t, db, "revoke@example.com")

	ctx := context.Background()
	sig := "revoke-access-sig-111"

	session := &fosite.DefaultSession{
		Subject: userID,
		ExpiresAt: map[fosite.TokenType]time.Time{
			fosite.AccessToken: time.Now().UTC().Add(time.Hour),
		},
	}

	client, _ := fs.GetClient(ctx, "revoke-agent")

	req := &fosite.Request{
		ID:             "revoke-req-1",
		RequestedAt:    time.Now().UTC(),
		Client:         client,
		RequestedScope: fosite.Arguments{"openid"},
		GrantedScope:   fosite.Arguments{"openid"},
		Session:        session,
		Form:           url.Values{},
	}

	if err := fs.CreateAccessTokenSession(ctx, sig, req); err != nil {
		t.Fatalf("CreateAccessTokenSession: %v", err)
	}

	// Revoke by request ID.
	if err := fs.RevokeAccessToken(ctx, "revoke-req-1"); err != nil {
		t.Fatalf("RevokeAccessToken: %v", err)
	}

	// Should be inactive now.
	_, err := fs.GetAccessTokenSession(ctx, sig, &fosite.DefaultSession{})
	if err == nil {
		t.Fatal("expected error after revoking access token")
	}
}
