package vault_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
	"github.com/sharkauth/sharkauth/internal/vault"
)

// The server secret must be >= 32 chars (see auth.NewFieldEncryptor).
const testServerSecret = "test-server-secret-0123456789-abcdef"

// mockTokenResponse models a canned /token endpoint reply used across the
// tests. Fields map straight onto the JSON payload the oauth2 library
// expects from a standards-compliant token endpoint.
type mockTokenResponse struct {
	status       int           // HTTP status — defaults to 200 when zero
	accessToken  string        // access_token
	refreshToken string        // refresh_token (empty = omit)
	tokenType    string        // token_type (defaults to "Bearer")
	expiresIn    time.Duration // expires_in (defaults to 1h when zero & status=200)
	scope        string        // optional "scope" field the provider echoes back
	errBody      string        // when set, body is literally this string
}

// mockOAuthServer spins up an httptest server emulating an OAuth 2.0 token
// endpoint. Callers swap the active response between test phases by mutating
// the returned pointer.
type mockOAuthServer struct {
	server   *httptest.Server
	response *mockTokenResponse
	// lastForm captures the most recently received form body so tests can
	// assert grant_type / refresh_token values.
	lastForm url.Values
}

// newMockOAuthServer returns a server that replies with the pointed-to
// response. Tests re-point `.response` between calls.
func newMockOAuthServer(t *testing.T, resp *mockTokenResponse) *mockOAuthServer {
	t.Helper()
	m := &mockOAuthServer{response: resp}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("mock oauth server: parse form: %v", err)
		}
		m.lastForm = r.PostForm

		cur := m.response
		if cur == nil {
			// Default to a success so unconfigured tests don't hang the
			// oauth2 client behind an empty-body parse error.
			cur = &mockTokenResponse{accessToken: "default"}
		}
		if cur.errBody != "" {
			w.WriteHeader(cur.status)
			_, _ = w.Write([]byte(cur.errBody))
			return
		}

		status := cur.status
		if status == 0 {
			status = http.StatusOK
		}
		tokenType := cur.tokenType
		if tokenType == "" {
			tokenType = "Bearer"
		}
		expiresIn := cur.expiresIn
		if expiresIn == 0 && status == http.StatusOK {
			expiresIn = time.Hour
		}

		payload := map[string]any{
			"access_token": cur.accessToken,
			"token_type":   tokenType,
			"expires_in":   int(expiresIn / time.Second),
		}
		if cur.refreshToken != "" {
			payload["refresh_token"] = cur.refreshToken
		}
		if cur.scope != "" {
			payload["scope"] = cur.scope
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Errorf("mock oauth server: encode payload: %v", err)
		}
	}))
	t.Cleanup(m.server.Close)
	return m
}

// setupManager wires a fresh in-memory store + FieldEncryptor + Manager with
// a stub clock. Returns everything the tests need to drive scenarios.
func setupManager(t *testing.T, nowFn func() time.Time) (*vault.Manager, *auth.FieldEncryptor, storage.Store) {
	t.Helper()
	store := testutil.NewTestDB(t)

	enc, err := auth.NewFieldEncryptor(testServerSecret)
	if err != nil {
		t.Fatalf("new field encryptor: %v", err)
	}
	m := vault.NewManagerWithClock(store, enc, nowFn)
	return m, enc, store
}

// seedUser inserts a minimal user row (FK target for vault_connections).
func seedUser(t *testing.T, store storage.Store, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	u := &storage.User{
		ID:        id,
		Email:     id + "@example.com",
		HashType:  "argon2id",
		Metadata:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

// seedProvider pushes a provider into the store via the Manager so the
// client secret is properly encrypted. Returns the provider ID.
func seedProvider(t *testing.T, m *vault.Manager, authURL, tokenURL string) string {
	t.Helper()
	p := &storage.VaultProvider{
		Name:        "mock",
		DisplayName: "Mock Provider",
		AuthURL:     authURL,
		TokenURL:    tokenURL,
		ClientID:    "client-123",
		Scopes:      []string{"read", "write"},
		Active:      true,
	}
	if err := m.CreateProvider(context.Background(), p, "super-secret"); err != nil {
		t.Fatalf("create provider: %v", err)
	}
	return p.ID
}

// ctxWithHTTPClient returns ctx with the oauth2 HTTP client override in place
// so exchanges + refreshes hit the mock server's TLS cert without panics.
func ctxWithHTTPClient(ctx context.Context, ts *httptest.Server) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, ts.Client())
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestExchangeAndStore_EncryptsTokens verifies that a successful code
// exchange persists encrypted tokens (never plaintext) and returns a
// connection with the right provider/user linkage.
func TestExchangeAndStore_EncryptsTokens(t *testing.T) {
	mock := newMockOAuthServer(t, &mockTokenResponse{
		accessToken:  "access-plain-123",
		refreshToken: "refresh-plain-456",
		scope:        "read write admin",
	})

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, enc, store := setupManager(t, func() time.Time { return now })

	seedUser(t, store, "usr_alice")
	providerID := seedProvider(t, m, "https://example.test/authorize", mock.server.URL)

	ctx := ctxWithHTTPClient(context.Background(), mock.server)
	conn, err := m.ExchangeAndStore(ctx, providerID, "usr_alice", "auth-code", "https://app.test/callback")
	if err != nil {
		t.Fatalf("ExchangeAndStore: %v", err)
	}

	if conn.ProviderID != providerID {
		t.Fatalf("provider id mismatch: got %q want %q", conn.ProviderID, providerID)
	}
	if conn.UserID != "usr_alice" {
		t.Fatalf("user id mismatch: got %q", conn.UserID)
	}
	if conn.AccessTokenEnc == "access-plain-123" {
		t.Fatalf("access token stored in plaintext")
	}
	if !strings.HasPrefix(conn.AccessTokenEnc, "enc::") {
		t.Fatalf("access token missing encryption prefix: %q", conn.AccessTokenEnc)
	}
	if !strings.HasPrefix(conn.RefreshTokenEnc, "enc::") {
		t.Fatalf("refresh token missing encryption prefix: %q", conn.RefreshTokenEnc)
	}

	// Decrypt round-trip — confirms the bytes we stored really are the
	// plaintext the mock server returned.
	accessPlain, err := enc.Decrypt(conn.AccessTokenEnc)
	if err != nil {
		t.Fatalf("decrypt access: %v", err)
	}
	if accessPlain != "access-plain-123" {
		t.Fatalf("decrypted access token: got %q want %q", accessPlain, "access-plain-123")
	}
	refreshPlain, err := enc.Decrypt(conn.RefreshTokenEnc)
	if err != nil {
		t.Fatalf("decrypt refresh: %v", err)
	}
	if refreshPlain != "refresh-plain-456" {
		t.Fatalf("decrypted refresh token: got %q want %q", refreshPlain, "refresh-plain-456")
	}

	if got := conn.Scopes; len(got) != 3 || got[0] != "read" || got[1] != "write" || got[2] != "admin" {
		t.Fatalf("granted scopes parse mismatch: %v", got)
	}

	// Ensure the mock received a proper authorization_code grant request.
	if grant := mock.lastForm.Get("grant_type"); grant != "authorization_code" {
		t.Fatalf("grant_type: got %q want authorization_code", grant)
	}
}

// TestGetFreshToken_ReturnsUnchangedWhenValid checks the happy path where
// the stored access token is still well inside its expiry window, so we
// return it without hitting the network.
func TestGetFreshToken_ReturnsUnchangedWhenValid(t *testing.T) {
	// The token endpoint must NOT be called in this test — point at a
	// server that errors on every request so we trip loudly if we do.
	panicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("mock server unexpectedly called for a valid-token scenario")
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	t.Cleanup(panicServer.Close)

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, _, store := setupManager(t, func() time.Time { return now })

	seedUser(t, store, "usr_bob")
	providerID := seedProvider(t, m, "https://example.test/authorize", panicServer.URL)

	// Hand-craft a connection with an access token expiring in 10 minutes.
	future := now.Add(10 * time.Minute)
	conn := mustSeedConnection(t, store, m, providerID, "usr_bob", "valid-access", "valid-refresh", &future)

	access, err := m.GetFreshToken(context.Background(), providerID, "usr_bob")
	if err != nil {
		t.Fatalf("GetFreshToken: %v", err)
	}
	if access != "valid-access" {
		t.Fatalf("access token: got %q want %q", access, "valid-access")
	}

	// Sanity: the stored ciphertext did not change.
	reloaded, err := store.GetVaultConnectionByID(context.Background(), conn.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.AccessTokenEnc != conn.AccessTokenEnc {
		t.Fatalf("access token unexpectedly rotated")
	}
}

// TestGetFreshToken_AutoRefreshesWhenExpired drives the refresh path end to
// end: stale token → manager swaps it for a fresh one via the mock server →
// stored ciphertext and the returned plaintext both reflect the new token.
func TestGetFreshToken_AutoRefreshesWhenExpired(t *testing.T) {
	mock := newMockOAuthServer(t, &mockTokenResponse{
		accessToken:  "refreshed-access",
		refreshToken: "rotated-refresh",
		expiresIn:    time.Hour,
	})

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, enc, store := setupManager(t, func() time.Time { return now })

	seedUser(t, store, "usr_carol")
	providerID := seedProvider(t, m, "https://example.test/authorize", mock.server.URL)

	pastExpiry := now.Add(-5 * time.Minute)
	conn := mustSeedConnection(t, store, m, providerID, "usr_carol", "old-access", "old-refresh", &pastExpiry)

	ctx := ctxWithHTTPClient(context.Background(), mock.server)
	access, err := m.GetFreshToken(ctx, providerID, "usr_carol")
	if err != nil {
		t.Fatalf("GetFreshToken: %v", err)
	}
	if access != "refreshed-access" {
		t.Fatalf("returned access: got %q want %q", access, "refreshed-access")
	}

	// Confirm the grant_type was refresh_token with the old refresh value.
	if grant := mock.lastForm.Get("grant_type"); grant != "refresh_token" {
		t.Fatalf("grant_type: got %q want refresh_token", grant)
	}
	if rt := mock.lastForm.Get("refresh_token"); rt != "old-refresh" {
		t.Fatalf("refresh_token sent: got %q want old-refresh", rt)
	}

	reloaded, err := store.GetVaultConnectionByID(context.Background(), conn.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.NeedsReauth {
		t.Fatalf("needs_reauth should be cleared after successful refresh")
	}
	accessPlain, err := enc.Decrypt(reloaded.AccessTokenEnc)
	if err != nil {
		t.Fatalf("decrypt stored access: %v", err)
	}
	if accessPlain != "refreshed-access" {
		t.Fatalf("stored access plaintext: got %q want %q", accessPlain, "refreshed-access")
	}
	refreshPlain, err := enc.Decrypt(reloaded.RefreshTokenEnc)
	if err != nil {
		t.Fatalf("decrypt stored refresh: %v", err)
	}
	if refreshPlain != "rotated-refresh" {
		t.Fatalf("stored refresh plaintext: got %q want %q", refreshPlain, "rotated-refresh")
	}
	if reloaded.ExpiresAt == nil || !reloaded.ExpiresAt.After(now) {
		t.Fatalf("expires_at not advanced: %+v", reloaded.ExpiresAt)
	}
}

// TestGetFreshToken_RefreshFailureMarksReauth verifies the error path: when
// the provider rejects the refresh token we surface ErrNeedsReauth AND
// flip the needs_reauth bit on the stored row.
func TestGetFreshToken_RefreshFailureMarksReauth(t *testing.T) {
	mock := newMockOAuthServer(t, &mockTokenResponse{
		status:  http.StatusBadRequest,
		errBody: `{"error":"invalid_grant","error_description":"refresh token revoked"}`,
	})

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, _, store := setupManager(t, func() time.Time { return now })

	seedUser(t, store, "usr_dave")
	providerID := seedProvider(t, m, "https://example.test/authorize", mock.server.URL)

	past := now.Add(-time.Hour)
	conn := mustSeedConnection(t, store, m, providerID, "usr_dave", "stale-access", "dead-refresh", &past)

	ctx := ctxWithHTTPClient(context.Background(), mock.server)
	_, err := m.GetFreshToken(ctx, providerID, "usr_dave")
	if !errors.Is(err, vault.ErrNeedsReauth) {
		t.Fatalf("expected ErrNeedsReauth, got %v", err)
	}

	reloaded, err := store.GetVaultConnectionByID(context.Background(), conn.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reloaded.NeedsReauth {
		t.Fatalf("needs_reauth flag not set after refresh failure")
	}

	// A second call should short-circuit on the needs_reauth flag without
	// hitting the mock server again (the handler would re-return 400, but
	// the manager should return ErrNeedsReauth before trying).
	_, err = m.GetFreshToken(context.Background(), providerID, "usr_dave")
	if !errors.Is(err, vault.ErrNeedsReauth) {
		t.Fatalf("second call should stay ErrNeedsReauth, got %v", err)
	}
}

// TestGetFreshToken_NoRefreshTokenReturnsSentinel covers the scenario where
// the access token has expired and we stored no refresh token (e.g. the user
// went through a one-shot consent). Manager returns ErrNoRefreshToken.
func TestGetFreshToken_NoRefreshTokenReturnsSentinel(t *testing.T) {
	// No network call is expected here; point at a would-panic server.
	panicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("mock server unexpectedly called when refresh token is absent")
	}))
	t.Cleanup(panicServer.Close)

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, _, store := setupManager(t, func() time.Time { return now })

	seedUser(t, store, "usr_eve")
	providerID := seedProvider(t, m, "https://example.test/authorize", panicServer.URL)

	past := now.Add(-time.Minute)
	mustSeedConnection(t, store, m, providerID, "usr_eve", "expired-access", "", &past)

	_, err := m.GetFreshToken(context.Background(), providerID, "usr_eve")
	if !errors.Is(err, vault.ErrNoRefreshToken) {
		t.Fatalf("expected ErrNoRefreshToken, got %v", err)
	}
}

// TestCreateProvider_EncryptsClientSecret verifies CreateProvider stores the
// client_secret under the encryption prefix and that it round-trips.
func TestCreateProvider_EncryptsClientSecret(t *testing.T) {
	m, enc, store := setupManager(t, nil)

	p := &storage.VaultProvider{
		Name:        "slack",
		DisplayName: "Slack",
		AuthURL:     "https://slack.com/oauth/v2/authorize",
		TokenURL:    "https://slack.com/api/oauth.v2.access",
		ClientID:    "slack-client-abc",
		Scopes:      []string{"channels:read"},
		Active:      true,
	}
	if err := m.CreateProvider(context.Background(), p, "slack-secret-xyz"); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	if p.ID == "" || !strings.HasPrefix(p.ID, "vp_") {
		t.Fatalf("provider id not populated/prefixed: %q", p.ID)
	}

	reloaded, err := store.GetVaultProviderByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("reload provider: %v", err)
	}
	if reloaded.ClientSecretEnc == "slack-secret-xyz" {
		t.Fatalf("client secret stored in plaintext")
	}
	if !strings.HasPrefix(reloaded.ClientSecretEnc, "enc::") {
		t.Fatalf("client secret missing encryption prefix: %q", reloaded.ClientSecretEnc)
	}
	plain, err := enc.Decrypt(reloaded.ClientSecretEnc)
	if err != nil {
		t.Fatalf("decrypt client secret: %v", err)
	}
	if plain != "slack-secret-xyz" {
		t.Fatalf("decrypted client secret: got %q want slack-secret-xyz", plain)
	}
}

// TestDisconnect_RemovesConnection exercises the disconnect path: row present
// → delete succeeds → subsequent Disconnect on the same id returns
// ErrConnectionNotFound.
func TestDisconnect_RemovesConnection(t *testing.T) {
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, _, store := setupManager(t, func() time.Time { return now })

	seedUser(t, store, "usr_frank")
	providerID := seedProvider(t, m, "https://example.test/authorize", "https://example.test/token")

	future := now.Add(time.Hour)
	conn := mustSeedConnection(t, store, m, providerID, "usr_frank", "acc", "ref", &future)

	if err := m.Disconnect(context.Background(), conn.ID); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	if _, err := store.GetVaultConnectionByID(context.Background(), conn.ID); err == nil {
		t.Fatalf("connection should be gone after Disconnect")
	}

	if err := m.Disconnect(context.Background(), conn.ID); !errors.Is(err, vault.ErrConnectionNotFound) {
		t.Fatalf("second Disconnect: got %v want ErrConnectionNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mustSeedConnection inserts a VaultConnection row directly via the store
// (skipping OAuth) so tests can set up specific expiry/token states. Tokens
// are encrypted with the same FieldEncryptor the manager uses.
func mustSeedConnection(t *testing.T, store storage.Store, m *vault.Manager, providerID, userID, accessPlain, refreshPlain string, expires *time.Time) *storage.VaultConnection {
	t.Helper()
	// Reach into a fresh encryptor with the same secret — keeps the helper
	// cohesive without needing to thread `enc` through every call site.
	enc, err := auth.NewFieldEncryptor(testServerSecret)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}

	accessEnc, err := enc.Encrypt(accessPlain)
	if err != nil {
		t.Fatalf("encrypt access: %v", err)
	}
	refreshEnc := ""
	if refreshPlain != "" {
		refreshEnc, err = enc.Encrypt(refreshPlain)
		if err != nil {
			t.Fatalf("encrypt refresh: %v", err)
		}
	}

	now := time.Now().UTC()
	id := fmt.Sprintf("vc_%s_%d", userID, time.Now().UnixNano())
	conn := &storage.VaultConnection{
		ID:              id,
		ProviderID:      providerID,
		UserID:          userID,
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		TokenType:       "Bearer",
		Scopes:          []string{"read"},
		ExpiresAt:       expires,
		Metadata:        map[string]any{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := store.CreateVaultConnection(context.Background(), conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	_ = m // parameter kept for symmetry with future assertions
	return conn
}
