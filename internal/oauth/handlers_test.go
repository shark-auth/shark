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
	"github.com/ory/fosite"

	mw "github.com/shark-auth/shark/internal/api/middleware"
	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/storage"
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

// testUserMW reads the test-only "X-Test-Auth-User" header and injects it
// as the authenticated user into the request context. Production code paths
// never mount this middleware; real auth comes from the session middleware.
func testUserMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if uid := req.Header.Get("X-Test-Auth-User"); uid != "" {
			req = req.WithContext(context.WithValue(req.Context(), mw.UserIDKey, uid))
		}
		next.ServeHTTP(w, req)
	})
}

// mountTestRouter sets up a chi router with the OAuth token and authorize endpoints.
func mountTestRouter(srv *Server) chi.Router {
	r := chi.NewRouter()
	r.Use(testUserMW)
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

// TestAuthorizeEndpoint_NotLoggedIn verifies that an unauthenticated
// GET /oauth/authorize 302-redirects to the hosted login page for the
// client's application, carrying the original authorize URL in
// return_to so post-login the caller resumes the OAuth flow.
//
// This is the current product behaviour (see HandleAuthorize: when
// getUserIDFromRequest returns "", we resolve the app by client_id
// and redirect to /hosted/<slug>/login?...&return_to=<orig>). Replaces
// the earlier 401 JSON expectation which pre-dates the hosted-login
// UX. See LANE_D_SCOPE.md Â§D8.
func TestAuthorizeEndpoint_NotLoggedIn(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "test-auth-client", false)

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	originalQuery := url.Values{
		"response_type": {"code"},
		"client_id":     {"test-auth-client"},
		"redirect_uri":  {"https://example.com/callback"},
		"scope":         {"openid"},
		"state":         {"test-state"},
	}.Encode()
	reqURL := ts.URL + "/oauth/authorize?" + originalQuery

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

	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 302, got %d: %s", resp.StatusCode, body)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatalf("expected Location header on 302, got none")
	}
	if !strings.Contains(loc, "/hosted/") {
		t.Errorf("expected Location to point at /hosted/<slug>/login, got %q", loc)
	}
	if !strings.Contains(loc, "/login") {
		t.Errorf("expected Location to land on /login, got %q", loc)
	}
	if !strings.Contains(loc, "return_to=") {
		t.Errorf("expected return_to= in Location, got %q", loc)
	}
	// The authorize URL should be embedded (URL-encoded) in return_to so the
	// login page can bounce the caller back. Cheap substring check on the
	// client_id is enough â€” if return_to is correctly populated, this
	// substring will appear (URL-encoded form of the original query).
	if !strings.Contains(loc, "test-auth-client") {
		t.Errorf("expected return_to to carry original authorize URL params (client_id), got %q", loc)
	}
}

// TestAuthorizeEndpoint_ConsentRequired tests that the authorize endpoint
// renders the HTML consent page when a user is logged in but has no prior consent.
func TestAuthorizeEndpoint_ConsentRequired(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "test-consent-client", false)
	userID := seedUser(t, store, "consent@example.com")

	r := chi.NewRouter()
	r.Use(testUserMW)
	// Simulate logged-in user via X-Test-Auth-User header.
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
	req.Header.Set("X-Test-Auth-User", userID)

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

	// Verify the response is an HTML consent page.
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("expected Content-Type text/html, got %q", contentType)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "test-consent-client") {
		t.Errorf("expected consent page to contain client ID, body snippet: %.200s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Authorize") {
		t.Errorf("expected consent page to contain Authorize button, body snippet: %.200s", bodyStr)
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

// TestDPoPTokenEndpointURL verifies the canonical HTU builder normalizes
// scheme/host regardless of where the data came from.
func TestDPoPTokenEndpointURL(t *testing.T) {
	// Plain HTTP: scheme derived from TLS==nil.
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	r.Host = "auth.example.com"
	got := dpopTokenEndpointURL(r)
	if got != "http://auth.example.com/oauth/token" {
		t.Errorf("http path: got %q", got)
	}

	// Request with explicit scheme on URL.
	r2 := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	r2.URL.Scheme = "https"
	r2.Host = "auth.example.com"
	got2 := dpopTokenEndpointURL(r2)
	if got2 != "https://auth.example.com/oauth/token" {
		t.Errorf("https path: got %q", got2)
	}

	// Fallback when r.Host is empty â€” should pull from r.URL.Host.
	r3 := httptest.NewRequest(http.MethodPost, "http://fallback.example.com/oauth/token", nil)
	r3.Host = ""
	got3 := dpopTokenEndpointURL(r3)
	if got3 != "http://fallback.example.com/oauth/token" {
		t.Errorf("fallback path: got %q", got3)
	}
}

// TestHandleAuthorizeDecision_Approved posts a valid consent decision and
// verifies the handler redirects with an authorization code + persists the
// consent in storage.
func TestHandleAuthorizeDecision_Approved(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "decision-approve-client", false)
	userID := seedUser(t, store, "approve@example.com")

	r := chi.NewRouter()
	r.Use(testUserMW)
	r.Post("/oauth/authorize", srv.HandleAuthorizeDecision)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// The challenge field is the original authorize query string.
	challenge := url.Values{
		"response_type":         {"code"},
		"client_id":             {"decision-approve-client"},
		"redirect_uri":          {"https://example.com/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-approved-123"},
		"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"code_challenge_method": {"S256"},
	}.Encode()

	form := url.Values{
		"challenge": {challenge},
		"approved":  {"true"},
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Test-Auth-User", userID)

	// Don't follow redirects â€” we want the 302 itself.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("posting decision: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 302/303 redirect, got %d: %s", resp.StatusCode, body)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatal("expected Location header on approve redirect")
	}
	// The redirect must go to the client's redirect_uri and carry a code.
	if !strings.HasPrefix(loc, "https://example.com/callback") {
		t.Errorf("redirect target unexpected: %q", loc)
	}
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if parsed.Query().Get("code") == "" {
		t.Errorf("expected code param in redirect: %q", loc)
	}
	if parsed.Query().Get("state") != "state-approved-123" {
		t.Errorf("expected state=state-approved-123 in redirect: %q", parsed.Query().Get("state"))
	}

	// Consent should have been persisted.
	consent, err := store.GetActiveConsent(context.Background(), userID, "decision-approve-client")
	if err != nil {
		t.Fatalf("GetActiveConsent: %v", err)
	}
	if consent == nil {
		t.Fatal("expected active consent after approval")
	}
	if !strings.Contains(consent.Scope, "openid") {
		t.Errorf("expected consent scope to include openid, got %q", consent.Scope)
	}
}

// TestHandleAuthorizeDecision_Denied verifies the denial path writes an
// access_denied error to the redirect_uri (per fosite).
func TestHandleAuthorizeDecision_Denied(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "decision-deny-client", false)
	userID := seedUser(t, store, "deny@example.com")

	r := chi.NewRouter()
	r.Use(testUserMW)
	r.Post("/oauth/authorize", srv.HandleAuthorizeDecision)
	ts := httptest.NewServer(r)
	defer ts.Close()

	challenge := url.Values{
		"response_type":         {"code"},
		"client_id":             {"decision-deny-client"},
		"redirect_uri":          {"https://example.com/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-denied-123"},
		"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"code_challenge_method": {"S256"},
	}.Encode()

	form := url.Values{
		"challenge": {challenge},
		"approved":  {"false"},
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Test-Auth-User", userID)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("posting denial: %v", err)
	}
	defer resp.Body.Close()

	// Denial should redirect with an error param (NOT a code).
	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatalf("expected Location on deny; status %d body %s",
			resp.StatusCode, readBody(t, resp))
	}
	if !strings.HasPrefix(loc, "https://example.com/callback") {
		t.Errorf("unexpected deny redirect: %q", loc)
	}
	parsed, _ := url.Parse(loc)
	if parsed.Query().Get("code") != "" {
		t.Error("did not expect code in deny redirect")
	}
	if parsed.Query().Get("error") == "" {
		t.Errorf("expected error= in deny redirect: %q", loc)
	}

	// No consent should have been created.
	consent, _ := store.GetActiveConsent(context.Background(), userID, "decision-deny-client")
	if consent != nil {
		t.Errorf("expected no consent after denial, got %+v", consent)
	}
}

// TestHandleAuthorizeDecision_NoLogin verifies unauthenticated POSTs are
// rejected with login_required (exercises the early-return branch before the
// decision is evaluated).
func TestHandleAuthorizeDecision_NoLogin(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "decision-nologin-client", false)

	r := chi.NewRouter()
	r.Use(testUserMW)
	r.Post("/oauth/authorize", srv.HandleAuthorizeDecision)
	ts := httptest.NewServer(r)
	defer ts.Close()

	challenge := url.Values{
		"response_type":         {"code"},
		"client_id":             {"decision-nologin-client"},
		"redirect_uri":          {"https://example.com/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-nologin-123"},
		"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"code_challenge_method": {"S256"},
	}.Encode()

	form := url.Values{
		"challenge": {challenge},
		"approved":  {"true"},
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// NOTE: no X-User-ID header.

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("posting decision: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
	var result map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] != "login_required" {
		t.Errorf("expected login_required, got %q", result["error"])
	}
}

// TestAuthorizeEndpoint_ExistingConsent verifies the shortcut branch in
// HandleAuthorize: when consent already covers the requested scopes, the
// handler calls completeAuthorize and issues a code directly.
func TestAuthorizeEndpoint_ExistingConsent(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "existing-consent-client", false)
	userID := seedUser(t, store, "existing-consent@example.com")

	// Pre-populate consent that covers openid.
	if err := store.CreateOAuthConsent(context.Background(), &storage.OAuthConsent{
		ID:        "consent_existing",
		UserID:    userID,
		ClientID:  "existing-consent-client",
		Scope:     "openid",
		GrantedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed consent: %v", err)
	}

	r := chi.NewRouter()
	r.Use(testUserMW)
	r.Get("/oauth/authorize", srv.HandleAuthorize)
	ts := httptest.NewServer(r)
	defer ts.Close()

	reqURL := ts.URL + "/oauth/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {"existing-consent-client"},
		"redirect_uri":          {"https://example.com/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-auto-123"},
		"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"code_challenge_method": {"S256"},
	}.Encode()

	req, _ := http.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("X-Test-Auth-User", userID)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("doing request: %v", err)
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected redirect with code, got status %d body %s", resp.StatusCode, body)
	}
	parsed, _ := url.Parse(loc)
	if parsed.Query().Get("code") == "" {
		t.Errorf("expected code in auto-consent redirect: %q", loc)
	}
	if parsed.Query().Get("state") != "state-auto-123" {
		t.Errorf("expected state=state-auto-123 in redirect: %q", loc)
	}
}

// TestStoreDPoPJKT verifies that after HandleToken persists a token, the
// DPoP JKT column is populated on the most recent token row when the client
// supplied a DPoP proof. We call storeDPoPJKT directly because building a
// real DPoP-bound token request requires signing, and the end-to-end flow is
// covered by the DPoP unit tests elsewhere.
func TestStoreDPoPJKT(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "dpop-jkt-client", false)

	ctx := context.Background()
	// Insert a token row for this agent so storeDPoPJKT has something to
	// update. ListOAuthTokensByAgentID looks it up by "agent_"+clientID.
	tok := &storage.OAuthToken{
		ID:        "tok_dpop_jkt_1",
		JTI:       "jti-dpop-jkt-1",
		ClientID:  "dpop-jkt-client",
		AgentID:   "agent_dpop-jkt-client",
		TokenType: "access",
		TokenHash: "dpop-jkt-hash",
		Scope:     "openid",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateOAuthToken(ctx, tok); err != nil {
		t.Fatalf("CreateOAuthToken: %v", err)
	}

	// Build a fosite.AccessRequester stub. fosite.Request satisfies it.
	client, _ := srv.Store.GetClient(ctx, "dpop-jkt-client")
	ar := &fositeAccessRequest{
		Request: fosite.Request{
			Client: client,
		},
	}

	srv.storeDPoPJKT(ctx, ar, "my-jkt-thumbprint")

	got, err := store.GetOAuthTokenByJTI(ctx, "jti-dpop-jkt-1")
	if err != nil {
		t.Fatalf("GetOAuthTokenByJTI: %v", err)
	}
	if got.DPoPJKT != "my-jkt-thumbprint" {
		t.Errorf("expected DPoPJKT to be set, got %q", got.DPoPJKT)
	}

	// Also cover the "no tokens" branch â€” a second clientID with no rows.
	seedAgent(t, store, "dpop-jkt-empty", false)
	emptyClient, _ := srv.Store.GetClient(ctx, "dpop-jkt-empty")
	emptyAR := &fositeAccessRequest{
		Request: fosite.Request{Client: emptyClient},
	}
	// Must not panic. No assertion needed beyond "it returns".
	srv.storeDPoPJKT(ctx, emptyAR, "irrelevant-jkt")
}

// fositeAccessRequest is a minimal AccessRequester embedding fosite.Request so
// we can satisfy fosite.AccessRequester without building a full grant session.
type fositeAccessRequest struct {
	fosite.Request
}

func (f *fositeAccessRequest) GetGrantTypes() fosite.Arguments { return fosite.Arguments{} }

// readBody is a tiny test helper to drain a response body into a string.
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
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
