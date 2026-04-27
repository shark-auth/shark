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

// TestListConnections_ReturnsUserConnections verifies that ListConnections
// scopes results to the requested user: two providers + two connections
// belong to user_a while a third belongs to user_b. Only the first two
// should come back when we query as user_a.
func TestListConnections_ReturnsUserConnections(t *testing.T) {
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, _, store := setupManager(t, func() time.Time { return now })

	seedUser(t, store, "usr_a")
	seedUser(t, store, "usr_b")

	// Two providers — use a second call to seedProvider-like logic since
	// the helper hard-codes Name="mock".
	providerA := seedProvider(t, m, "https://a.test/authorize", "https://a.test/token")
	pB := &storage.VaultProvider{
		Name:        "mock-b",
		DisplayName: "Mock B",
		AuthURL:     "https://b.test/authorize",
		TokenURL:    "https://b.test/token",
		ClientID:    "client-b",
		Scopes:      []string{"read"},
		Active:      true,
	}
	if err := m.CreateProvider(context.Background(), pB, "secret-b"); err != nil {
		t.Fatalf("create provider B: %v", err)
	}

	future := now.Add(time.Hour)
	mustSeedConnection(t, store, m, providerA, "usr_a", "acc-a1", "ref-a1", &future)
	mustSeedConnection(t, store, m, pB.ID, "usr_a", "acc-a2", "ref-a2", &future)
	mustSeedConnection(t, store, m, providerA, "usr_b", "acc-b1", "ref-b1", &future)

	got, err := m.ListConnections(context.Background(), "usr_a")
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 connections for usr_a, got %d", len(got))
	}
	for _, c := range got {
		if c.UserID != "usr_a" {
			t.Fatalf("ListConnections leaked a row for %q", c.UserID)
		}
	}

	gotB, err := m.ListConnections(context.Background(), "usr_b")
	if err != nil {
		t.Fatalf("ListConnections usr_b: %v", err)
	}
	if len(gotB) != 1 {
		t.Fatalf("expected 1 connection for usr_b, got %d", len(gotB))
	}
}

// TestBuildAuthURL_IncludesScopesAndOffline verifies the authorize URL has
// the pieces we rely on: response_type=code, the provider's client_id, the
// space-joined scope list, and access_type=offline so the first consent
// yields a refresh token.
func TestBuildAuthURL_IncludesScopesAndOffline(t *testing.T) {
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, _, _ := setupManager(t, func() time.Time { return now })

	// seedProvider uses scopes = ["read", "write"] and client_id "client-123".
	providerID := seedProvider(t, m, "https://example.test/authorize", "https://example.test/token")

	raw, err := m.BuildAuthURL(context.Background(), providerID, "state-abc", "https://app.test/cb", nil)
	if err != nil {
		t.Fatalf("BuildAuthURL: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}
	q := u.Query()
	if got := q.Get("response_type"); got != "code" {
		t.Fatalf("response_type: got %q want %q", got, "code")
	}
	if got := q.Get("client_id"); got != "client-123" {
		t.Fatalf("client_id: got %q want %q", got, "client-123")
	}
	if got := q.Get("scope"); got != "read write" {
		t.Fatalf("scope: got %q want %q", got, "read write")
	}
	if got := q.Get("access_type"); got != "offline" {
		t.Fatalf("access_type: got %q want %q (offline access requests refresh tokens)", got, "offline")
	}
	if got := q.Get("state"); got != "state-abc" {
		t.Fatalf("state echo mismatch: got %q", got)
	}
}

// TestExchangeAndStore_PreservesRefreshTokenWhenOmittedOnReExchange covers
// the regression flagged by I2: if an upstream provider omits refresh_token
// on a re-exchange (user re-consented but provider didn't re-issue), we
// must NOT overwrite the ciphertext with Encrypt(""). The original refresh
// token ciphertext should remain decryptable to the original plaintext.
func TestExchangeAndStore_PreservesRefreshTokenWhenOmittedOnReExchange(t *testing.T) {
	// First response includes a refresh token; second one omits it.
	resp := &mockTokenResponse{
		accessToken:  "access-first",
		refreshToken: "refresh-original",
		expiresIn:    time.Hour,
		scope:        "read write",
	}
	mock := newMockOAuthServer(t, resp)

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	m, enc, store := setupManager(t, func() time.Time { return now })

	seedUser(t, store, "usr_reissue")
	providerID := seedProvider(t, m, "https://example.test/authorize", mock.server.URL)

	ctx := ctxWithHTTPClient(context.Background(), mock.server)
	first, err := m.ExchangeAndStore(ctx, providerID, "usr_reissue", "code-1", "https://app.test/cb")
	if err != nil {
		t.Fatalf("first ExchangeAndStore: %v", err)
	}
	originalRefreshEnc := first.RefreshTokenEnc
	if originalRefreshEnc == "" {
		t.Fatalf("setup: first exchange produced empty refresh ciphertext")
	}
	if plain, _ := enc.Decrypt(originalRefreshEnc); plain != "refresh-original" {
		t.Fatalf("setup: first refresh plaintext got %q want refresh-original", plain)
	}

	// Second exchange: access token rotates but refresh_token is omitted.
	mock.response = &mockTokenResponse{
		accessToken:  "access-second",
		refreshToken: "", // upstream did NOT rotate the refresh token
		expiresIn:    time.Hour,
		scope:        "read write",
	}

	second, err := m.ExchangeAndStore(ctx, providerID, "usr_reissue", "code-2", "https://app.test/cb")
	if err != nil {
		t.Fatalf("second ExchangeAndStore: %v", err)
	}

	if second.RefreshTokenEnc != originalRefreshEnc {
		t.Fatalf("refresh ciphertext was overwritten on re-exchange; want preserved original")
	}

	// Authoritative check: load from store and decrypt.
	reloaded, err := store.GetVaultConnectionByID(context.Background(), second.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	refreshPlain, err := enc.Decrypt(reloaded.RefreshTokenEnc)
	if err != nil {
		t.Fatalf("decrypt preserved refresh: %v", err)
	}
	if refreshPlain != "refresh-original" {
		t.Fatalf("preserved refresh plaintext: got %q want refresh-original", refreshPlain)
	}

	// Access token should have rotated.
	accessPlain, err := enc.Decrypt(reloaded.AccessTokenEnc)
	if err != nil {
		t.Fatalf("decrypt access: %v", err)
	}
	if accessPlain != "access-second" {
		t.Fatalf("access plaintext: got %q want access-second", accessPlain)
	}
}

// markReauthErrStore wraps a real Store and forces
// MarkVaultConnectionNeedsReauth to return a sentinel error. Everything
// else delegates to the embedded store so the rest of the manager keeps
// working. Method set is satisfied via interface embedding — Go picks up
// the real implementation for every method we don't redeclare.
type markReauthErrStore struct {
	storage.Store
	err error
}

func (s *markReauthErrStore) MarkVaultConnectionNeedsReauth(_ context.Context, _ string, _ bool) error {
	return s.err
}

// TestGetFreshToken_RefreshFailureWithMarkReauthError verifies I3: when the
// refresh fails AND the subsequent flag-flip also fails, the returned error
// surfaces the storage failure instead of silently falling through to the
// ErrNeedsReauth sentinel.
func TestGetFreshToken_RefreshFailureWithMarkReauthError(t *testing.T) {
	mock := newMockOAuthServer(t, &mockTokenResponse{
		status:  http.StatusBadRequest,
		errBody: `{"error":"invalid_grant"}`,
	})

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	// Build the normal stack first — seeding providers + connections needs
	// the real store — then wrap the store and build a second Manager that
	// shares the same DB but reroutes MarkVaultConnectionNeedsReauth.
	baseStore := testutil.NewTestDB(t)
	enc, err := auth.NewFieldEncryptor(testServerSecret)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	seedM := vault.NewManagerWithClock(baseStore, enc, func() time.Time { return now })

	seedUser(t, baseStore, "usr_mark")
	providerID := seedProvider(t, seedM, "https://example.test/authorize", mock.server.URL)
	past := now.Add(-time.Hour)
	mustSeedConnection(t, baseStore, seedM, providerID, "usr_mark", "stale", "dead-refresh", &past)

	// The wrapped store returns a sentinel on MarkVaultConnectionNeedsReauth.
	markErr := errors.New("boom: storage offline")
	wrapped := &markReauthErrStore{Store: baseStore, err: markErr}
	m := vault.NewManagerWithClock(wrapped, enc, func() time.Time { return now })

	ctx := ctxWithHTTPClient(context.Background(), mock.server)
	_, err = m.GetFreshToken(ctx, providerID, "usr_mark")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if errors.Is(err, vault.ErrNeedsReauth) {
		t.Fatalf("expected wrapped storage error, got ErrNeedsReauth (storage failure was swallowed)")
	}
	if !errors.Is(err, markErr) {
		t.Fatalf("expected wrapped markErr, got %v", err)
	}
}

// TestUpdateProviderSecret exercises the secret-rotation path: create a
// provider with secret A, rotate to secret B, then reload + decrypt and
// confirm we see B.
func TestUpdateProviderSecret(t *testing.T) {
	m, enc, store := setupManager(t, nil)

	p := &storage.VaultProvider{
		Name:        "rotate-me",
		DisplayName: "Rotate Me",
		AuthURL:     "https://rotate.test/authorize",
		TokenURL:    "https://rotate.test/token",
		ClientID:    "client-rot",
		Scopes:      []string{"read"},
		Active:      true,
	}
	if err := m.CreateProvider(context.Background(), p, "secret-A"); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	if err := m.UpdateProviderSecret(context.Background(), p.ID, "secret-B"); err != nil {
		t.Fatalf("UpdateProviderSecret: %v", err)
	}

	reloaded, err := store.GetVaultProviderByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("reload provider: %v", err)
	}
	if reloaded.ClientSecretEnc == "secret-B" {
		t.Fatalf("client secret stored in plaintext after rotation")
	}
	plain, err := enc.Decrypt(reloaded.ClientSecretEnc)
	if err != nil {
		t.Fatalf("decrypt rotated secret: %v", err)
	}
	if plain != "secret-B" {
		t.Fatalf("rotated secret plaintext: got %q want secret-B", plain)
	}
}

// TestBuildAuthURL_LinearExtraParams verifies that BuildAuthURL injects
// prompt=consent into the authorize URL for a provider named "linear", relying
// on the template registry — not a hard-coded branch in the manager.
func TestBuildAuthURL_LinearExtraParams(t *testing.T) {
	now := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	m, _, _ := setupManager(t, func() time.Time { return now })

	// Seed a provider with name "linear" so the template lookup finds the extras.
	p := &storage.VaultProvider{
		Name:        "linear",
		DisplayName: "Linear",
		AuthURL:     "https://linear.app/oauth/authorize",
		TokenURL:    "https://api.linear.app/oauth/token",
		ClientID:    "linear-client",
		Scopes:      []string{"read", "write"},
		Active:      true,
	}
	if err := m.CreateProvider(context.Background(), p, "linear-secret"); err != nil {
		t.Fatalf("create linear provider: %v", err)
	}

	raw, err := m.BuildAuthURL(context.Background(), p.ID, "state-xyz", "https://app.test/cb", nil)
	if err != nil {
		t.Fatalf("BuildAuthURL: %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	q := u.Query()
	if got := q.Get("prompt"); got != "consent" {
		t.Errorf("prompt: got %q, want consent (Linear requires prompt=consent)", got)
	}
}

// TestBuildAuthURL_JiraExtraParams verifies both audience and prompt are injected.
func TestBuildAuthURL_JiraExtraParams(t *testing.T) {
	now := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	m, _, _ := setupManager(t, func() time.Time { return now })

	p := &storage.VaultProvider{
		Name:        "jira",
		DisplayName: "Jira Cloud",
		AuthURL:     "https://auth.atlassian.com/authorize",
		TokenURL:    "https://auth.atlassian.com/oauth/token",
		ClientID:    "jira-client",
		Scopes:      []string{"read:jira-work"},
		Active:      true,
	}
	if err := m.CreateProvider(context.Background(), p, "jira-secret"); err != nil {
		t.Fatalf("create jira provider: %v", err)
	}

	raw, err := m.BuildAuthURL(context.Background(), p.ID, "state-jira", "https://app.test/cb", nil)
	if err != nil {
		t.Fatalf("BuildAuthURL: %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	q := u.Query()
	if got := q.Get("audience"); got != "api.atlassian.com" {
		t.Errorf("audience: got %q, want api.atlassian.com", got)
	}
	if got := q.Get("prompt"); got != "consent" {
		t.Errorf("prompt: got %q, want consent", got)
	}
}

// TestExchangeAndStore_SlackV2_OkFalse verifies that a Slack v2 token
// endpoint returning HTTP 200 with {ok:false, error:"invalid_code"} is
// surfaced as an error, not a silent empty-token success.
func TestExchangeAndStore_SlackV2_OkFalse(t *testing.T) {
	// Slack returns HTTP 200 even for errors.
	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_code"}`))
	}))
	t.Cleanup(slackServer.Close)

	now := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	m, _, store := setupManager(t, func() time.Time { return now })
	seedUser(t, store, "usr_slack_err")

	// Use token URL pointing at our mock but name the provider "slack" so
	// the slack_v2 branch activates.
	p := &storage.VaultProvider{
		Name:        "slack",
		DisplayName: "Slack",
		AuthURL:     "https://slack.com/oauth/v2/authorize",
		TokenURL:    slackServer.URL,
		ClientID:    "slack-client",
		Scopes:      []string{"chat:write"},
		Active:      true,
	}
	if err := m.CreateProvider(context.Background(), p, "slack-secret"); err != nil {
		t.Fatalf("create slack provider: %v", err)
	}

	ctx := ctxWithHTTPClient(context.Background(), slackServer)
	_, err := m.ExchangeAndStore(ctx, p.ID, "usr_slack_err", "bad-code", "https://app.test/cb")
	if err == nil {
		t.Fatal("expected error from Slack ok:false response, got nil")
	}
	if !strings.Contains(err.Error(), "invalid_code") {
		t.Errorf("expected error to contain Slack's error field, got: %v", err)
	}
}

// TestExchangeAndStore_SlackV2_XoxpPreferred verifies that when both bot
// (xoxb) and user (xoxp) tokens are present we prefer the user token.
func TestExchangeAndStore_SlackV2_XoxpPreferred(t *testing.T) {
	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"ok": true,
			"access_token": "xoxb-bot-token",
			"token_type": "bot",
			"scope": "chat:write",
			"authed_user": {
				"access_token": "xoxp-user-token",
				"scope": "identity.basic",
				"token_type": "user"
			}
		}`))
	}))
	t.Cleanup(slackServer.Close)

	now := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	m, enc, store := setupManager(t, func() time.Time { return now })
	seedUser(t, store, "usr_slack_xoxp")

	p := &storage.VaultProvider{
		Name:        "slack",
		DisplayName: "Slack",
		AuthURL:     "https://slack.com/oauth/v2/authorize",
		TokenURL:    slackServer.URL,
		ClientID:    "slack-client",
		Scopes:      []string{"chat:write"},
		Active:      true,
	}
	if err := m.CreateProvider(context.Background(), p, "slack-secret"); err != nil {
		t.Fatalf("create slack provider: %v", err)
	}

	ctx := ctxWithHTTPClient(context.Background(), slackServer)
	conn, err := m.ExchangeAndStore(ctx, p.ID, "usr_slack_xoxp", "good-code", "https://app.test/cb")
	if err != nil {
		t.Fatalf("ExchangeAndStore: %v", err)
	}

	plain, err := enc.Decrypt(conn.AccessTokenEnc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plain != "xoxp-user-token" {
		t.Errorf("expected xoxp user token, got %q", plain)
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
	id := fmt.Sprintf("vc_%s_%s_%d", userID, providerID, time.Now().UnixNano())
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
