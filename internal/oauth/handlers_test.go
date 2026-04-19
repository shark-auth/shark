package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// testServerSecret used for all handler tests. Must be >= 32 bytes.
const testServerSecret = "test-secret-must-be-at-least-32-bytes-long!!"

// newTestOAuthServer creates an OAuth Server backed by an in-memory SQLite DB
// with all migrations applied. Returns the server and the underlying store.
func newTestOAuthServer(t *testing.T) (*Server, storage.Store) {
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

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:    8080,
			Secret:  testServerSecret,
			BaseURL: "http://localhost:8080",
		},
		OAuthServer: config.OAuthServerConfig{
			Enabled:              true,
			AccessTokenLifetime:  "15m",
			RefreshTokenLifetime: "30d",
			AuthCodeLifetime:     "60s",
		},
	}

	srv, err := NewServer(db, cfg)
	if err != nil {
		t.Fatalf("creating OAuth server: %v", err)
	}

	return srv, db
}

// mountTestRouter sets up a chi router with the OAuth token and authorize endpoints.
func mountTestRouter(srv *Server) chi.Router {
	r := chi.NewRouter()
	r.Post("/oauth/token", srv.HandleToken)
	r.Get("/oauth/authorize", srv.HandleAuthorize)
	r.Post("/oauth/authorize", srv.HandleAuthorizeDecision)
	return r
}

// TestTokenEndpoint_ClientCredentials tests that a confidential client can
// obtain an access token via the client_credentials grant.
func TestTokenEndpoint_ClientCredentials(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "test-cc-client", false)

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {"openid"},
	}
	req, err := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test-cc-client", "test-secret")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if _, ok := tokenResp["access_token"]; !ok {
		t.Fatal("response missing access_token")
	}
	if tokenResp["token_type"] != "bearer" {
		t.Errorf("expected token_type=bearer, got %v", tokenResp["token_type"])
	}
	if _, ok := tokenResp["expires_in"]; !ok {
		t.Error("response missing expires_in")
	}
}

// TestTokenEndpoint_InvalidClient tests that an unknown client_id returns 401.
func TestTokenEndpoint_InvalidClient(t *testing.T) {
	srv, _ := newTestOAuthServer(t)

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type": {"client_credentials"},
	}
	req, err := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("nonexistent-client", "wrong-secret")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
}

// TestTokenEndpoint_WrongSecret tests that a wrong client secret returns 401.
func TestTokenEndpoint_WrongSecret(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "test-wrong-secret", false)

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type": {"client_credentials"},
	}
	req, err := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test-wrong-secret", "definitely-wrong-secret")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
}

// TestTokenEndpoint_AuthCode_MissingPKCE verifies that an authorization code
// exchange without a code_verifier fails when PKCE is enforced.
func TestTokenEndpoint_AuthCode_MissingPKCE(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "test-pkce-client", false)
	userID := seedUser(t, store, "pkce@example.com")

	// Manually create an authorization code in the store (bypassing the
	// authorize endpoint).
	codeHash := hashSignature("test-auth-code-123")
	authCode := &storage.OAuthAuthorizationCode{
		CodeHash:            codeHash,
		ClientID:            "test-pkce-client",
		UserID:              userID,
		RedirectURI:         "https://example.com/callback",
		Scope:               "openid",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", // SHA-256 of "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().UTC().Add(5 * time.Minute),
		CreatedAt:           time.Now().UTC(),
	}
	if err := store.CreateAuthorizationCode(context.Background(), authCode); err != nil {
		t.Fatalf("creating auth code: %v", err)
	}

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	// Try to exchange code WITHOUT code_verifier.
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {"test-auth-code-123"},
		"redirect_uri": {"https://example.com/callback"},
		"client_id":    {"test-pkce-client"},
	}
	req, err := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test-pkce-client", "test-secret")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	// fosite should reject with an error since code_verifier is missing.
	// The exact status code depends on fosite's implementation but it should
	// not be 200.
	if resp.StatusCode == http.StatusOK {
		t.Fatal("expected error for missing PKCE code_verifier, got 200")
	}
}

// TestAuthorizeEndpoint_NotLoggedIn tests that the authorize endpoint returns
// login_required when no session is present.
func TestAuthorizeEndpoint_NotLoggedIn(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "test-auth-client", false)

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	reqURL := ts.URL + "/oauth/authorize?" + url.Values{
		"response_type": {"code"},
		"client_id":     {"test-auth-client"},
		"redirect_uri":  {"https://example.com/callback"},
		"scope":         {"openid"},
		"state":         {"test-state"},
	}.Encode()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(reqURL)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if result["error"] != "login_required" {
		t.Errorf("expected error=login_required, got %q", result["error"])
	}
}

// TestAuthorizeEndpoint_ConsentRequired tests that the authorize endpoint
// returns consent info when a user is logged in but has no prior consent.
func TestAuthorizeEndpoint_ConsentRequired(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "test-consent-client", false)
	userID := seedUser(t, store, "consent@example.com")

	r := chi.NewRouter()
	// Simulate logged-in user via X-User-ID header.
	r.Get("/oauth/authorize", srv.HandleAuthorize)

	ts := httptest.NewServer(r)
	defer ts.Close()

	reqURL := ts.URL + "/oauth/authorize?" + url.Values{
		"response_type": {"code"},
		"client_id":     {"test-consent-client"},
		"redirect_uri":  {"https://example.com/callback"},
		"scope":         {"openid"},
		"state":         {"test-state"},
	}.Encode()

	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("X-User-ID", userID)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if result["type"] != "consent_required" {
		t.Errorf("expected type=consent_required, got %q", result["type"])
	}
}

// TestTokenEndpoint_ClientCredentials_ScopeGrant verifies that granted scopes
// appear in the token response.
func TestTokenEndpoint_ClientCredentials_ScopeGrant(t *testing.T) {
	srv, store := newTestOAuthServer(t)

	// Create agent with specific scopes.
	h := sha256.Sum256([]byte("test-secret"))
	agent := &storage.Agent{
		ID:               "agent_scope_test",
		Name:             "Scope Test Agent",
		Description:      "Tests scope granting",
		ClientID:         "scope-test-client",
		ClientSecretHash: hex.EncodeToString(h[:]),
		ClientType:       "confidential",
		AuthMethod:       "client_secret_basic",
		RedirectURIs:     []string{"https://example.com/callback"},
		GrantTypes:       []string{"client_credentials"},
		ResponseTypes:    []string{"code"},
		Scopes:           []string{"read", "write", "admin"},
		TokenLifetime:    900,
		Active:           true,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := store.CreateAgent(context.Background(), agent); err != nil {
		t.Fatalf("creating agent: %v", err)
	}

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {"read write"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("scope-test-client", "test-secret")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp) //nolint:errcheck

	scope, _ := tokenResp["scope"].(string)
	if !strings.Contains(scope, "read") || !strings.Contains(scope, "write") {
		t.Errorf("expected scope to contain read and write, got %q", scope)
	}
}

// TestNewServer_KeyIdempotency verifies that creating two OAuth servers
// against the same store reuses the existing ES256 key rather than generating
// a new one.
func TestNewServer_KeyIdempotency(t *testing.T) {
	db, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("creating test db: %v", err)
	}
	if err := storage.RunMigrations(db.DB(), testMigrationsFS, "testmigrations"); err != nil {
		db.Close()
		t.Fatalf("running migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		Server: config.ServerConfig{
			Secret:  testServerSecret,
			BaseURL: "http://localhost:8080",
		},
		OAuthServer: config.OAuthServerConfig{
			Enabled:              true,
			AccessTokenLifetime:  "15m",
			RefreshTokenLifetime: "30d",
			AuthCodeLifetime:     "60s",
		},
	}

	srv1, err := NewServer(db, cfg)
	if err != nil {
		t.Fatalf("creating first server: %v", err)
	}

	srv2, err := NewServer(db, cfg)
	if err != nil {
		t.Fatalf("creating second server: %v", err)
	}

	if srv1.SigningKeyID != srv2.SigningKeyID {
		t.Errorf("expected same signing key ID, got %q vs %q", srv1.SigningKeyID, srv2.SigningKeyID)
	}
}
