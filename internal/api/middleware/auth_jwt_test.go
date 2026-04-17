package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/auth"
	jwtpkg "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

const (
	mwTestSecret  = "test-server-secret-that-is-at-least-32-bytes-long"
	mwTestBaseURL = "http://localhost:8080"
)

func newMWTestJWTCfg(checkPerRequest bool) *config.JWTConfig {
	return &config.JWTConfig{
		Enabled:         true,
		Mode:            "session",
		Audience:        "shark",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "30d",
		ClockSkew:       "1s", // minimal skew so expired tokens fail fast
		Revocation: config.JWTRevocationConfig{
			CheckPerRequest: checkPerRequest,
		},
	}
}

func newMWTestJWTManager(t *testing.T, store *storage.SQLiteStore, checkPerRequest bool) *jwtpkg.Manager {
	t.Helper()
	cfg := newMWTestJWTCfg(checkPerRequest)
	mgr := jwtpkg.NewManager(cfg, store, mwTestBaseURL, mwTestSecret)
	if err := mgr.EnsureActiveKey(context.Background()); err != nil {
		t.Fatalf("EnsureActiveKey: %v", err)
	}
	return mgr
}

// mwTestUser returns a minimal storage.User for token issuance.
func mwTestUser() *storage.User {
	return &storage.User{ID: "usr_mwtest", Email: "mwtest@example.com"}
}

// captureHandler is a simple handler that records what the middleware set in context.
type captureHandler struct {
	userID     string
	sessionID  string
	authMethod string
}

func (c *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.userID = mw.GetUserID(r.Context())
	c.sessionID = mw.GetSessionID(r.Context())
	c.authMethod = mw.GetAuthMethod(r.Context())
	w.WriteHeader(http.StatusOK)
}

// makeSessionManager returns a real SessionManager backed by the test store.
func makeSessionManager(t *testing.T, store storage.Store, cfg *config.Config) *auth.SessionManager {
	t.Helper()
	lifetime := 24 * time.Hour
	return auth.NewSessionManager(store, cfg.Server.Secret, lifetime, cfg.Server.BaseURL)
}

// makeEncryptedSessionCookie creates a real session in the store and returns
// the securecookie-encoded cookie value (as the middleware would read it).
func makeEncryptedSessionCookie(t *testing.T, sm *auth.SessionManager, store *storage.SQLiteStore, userID string) (sessionID string, cookieValue string) {
	t.Helper()

	sess, err := sm.CreateSessionWithMFA(context.Background(), userID, "127.0.0.1", "test-ua", "password", false)
	if err != nil {
		t.Fatalf("CreateSessionWithMFA: %v", err)
	}

	// Capture the encoded cookie value using SetSessionCookie on a recorder.
	rec := httptest.NewRecorder()
	sm.SetSessionCookie(rec, sess.ID)

	result := rec.Result()
	cookies := result.Cookies()
	for _, c := range cookies {
		if c.Name == "shark_session" {
			return sess.ID, c.Value
		}
	}
	t.Fatal("SetSessionCookie did not set a shark_session cookie")
	return "", ""
}

// buildExpiredRS256Token generates an RSA keypair, inserts the public key into
// the store (as a retired key), and returns a signed expired token.
// This is a self-contained version of what export_test.go provides, usable from
// external test packages since export_test.go functions are not accessible externally.
func buildExpiredRS256Token(t *testing.T, store *storage.SQLiteStore) string {
	t.Helper()
	ctx := context.Background()

	// Generate RSA-2048 keypair.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	// Compute kid = base64url(SHA-256(DER public key))[:16] (same convention as keys.go).
	derBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	hash := sha256.Sum256(derBytes)
	kid := base64.RawURLEncoding.EncodeToString(hash[:])[:16]

	// Encode public key to PEM.
	pubPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}))

	// Insert into store as "retired" so kid lookup passes.
	// Validate only reads public_key_pem (plaintext column) — private_key_pem is unused here.
	if err := store.InsertSigningKey(ctx, &storage.SigningKey{
		KID:           kid,
		Algorithm:     "RS256",
		PublicKeyPEM:  pubPEM,
		PrivateKeyPEM: "dummy-placeholder", // Validate never decrypts retired keys
		Status:        "retired",
	}); err != nil {
		t.Fatalf("InsertSigningKey: %v", err)
	}

	// Build expired token signed with the private key.
	now := time.Now().UTC()
	claims := gojwt.MapClaims{
		"iss":        mwTestBaseURL,
		"sub":        "usr_expired",
		"aud":        gojwt.ClaimStrings{"shark"},
		"exp":        gojwt.NewNumericDate(now.Add(-2 * time.Hour)), // 2h in the past
		"nbf":        gojwt.NewNumericDate(now.Add(-3 * time.Hour)),
		"iat":        gojwt.NewNumericDate(now.Add(-3 * time.Hour)),
		"jti":        "expired-test-jti",
		"token_type": "session",
		"session_id": "",
		"mfa_passed": false,
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	return signed
}

// TestRequireSession_BearerValid: valid access JWT bearer → 200, AuthMethod="jwt",
// SessionID="" (access token, token_type="access", per §2.2 has no SessionID).
func TestRequireSession_BearerValid(t *testing.T) {
	store := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	jwtMgr := newMWTestJWTManager(t, store, false)
	sm := makeSessionManager(t, store, cfg)

	user := mwTestUser()

	// Issue an access token (token_type="access") — SessionID must be empty per §2.2.
	accessToken, _, err := jwtMgr.IssueAccessRefreshPair(context.Background(), user, "sess_bearer_valid", false)
	if err != nil {
		t.Fatalf("IssueAccessRefreshPair: %v", err)
	}

	capture := &captureHandler{}
	handler := mw.RequireSessionFunc(sm, jwtMgr)(capture)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capture.authMethod != "jwt" {
		t.Errorf("expected AuthMethod=jwt, got %q", capture.authMethod)
	}
	if capture.userID != user.ID {
		t.Errorf("expected UserID=%s, got %q", user.ID, capture.userID)
	}
	// access token → SessionID must be "" (§2.2: empty when token_type=="access")
	if capture.sessionID != "" {
		t.Errorf("expected SessionID='' for access token, got %q", capture.sessionID)
	}
}

// TestRequireSession_BearerInvalid: malformed bearer → 401 + WWW-Authenticate.
// Cookie MUST be ignored even if a valid cookie is also present (no fallthrough per §2.1).
func TestRequireSession_BearerInvalid(t *testing.T) {
	store := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	jwtMgr := newMWTestJWTManager(t, store, false)
	sm := makeSessionManager(t, store, cfg)

	// Create a valid user and session.
	user := mwTestUser()
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	// Get an encoded cookie value we can attach to the request.
	_, cookieVal := makeEncryptedSessionCookie(t, sm, store, user.ID)

	capture := &captureHandler{}
	handler := mw.RequireSessionFunc(sm, jwtMgr)(capture)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	// Also set a valid cookie — middleware must NOT fall through to cookie path.
	req.AddCookie(&http.Cookie{Name: "shark_session", Value: cookieVal})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header on invalid bearer rejection")
	}
	// Handler must NOT have been called (captureHandler.authMethod remains empty).
	if capture.authMethod != "" {
		t.Errorf("capture handler should not have been called; authMethod=%q", capture.authMethod)
	}
}

// TestRequireSession_BearerExpired: expired JWT bearer → 401 with WWW-Authenticate
// containing error="invalid_token".
func TestRequireSession_BearerExpired(t *testing.T) {
	store := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	sm := makeSessionManager(t, store, cfg)
	jwtMgr := newMWTestJWTManager(t, store, false)

	expiredToken := buildExpiredRS256Token(t, store)

	capture := &captureHandler{}
	handler := mw.RequireSessionFunc(sm, jwtMgr)(capture)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
	wwwAuth := w.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("expected WWW-Authenticate header on expired bearer")
	}
	if !strings.Contains(wwwAuth, "invalid_token") {
		t.Errorf("expected invalid_token in WWW-Authenticate, got: %s", wwwAuth)
	}
	if capture.authMethod != "" {
		t.Errorf("handler should not have been called; authMethod=%q", capture.authMethod)
	}
}

// TestRequireSession_BearerRefreshToken: refresh token used as bearer → 401.
// The WWW-Authenticate header includes error_description mentioning refresh (§2.3).
// Checked via the response body which contains the full message.
func TestRequireSession_BearerRefreshToken(t *testing.T) {
	store := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	jwtMgr := newMWTestJWTManager(t, store, false)
	sm := makeSessionManager(t, store, cfg)

	user := mwTestUser()
	_, refreshToken, err := jwtMgr.IssueAccessRefreshPair(context.Background(), user, "sess_refresh_test", false)
	if err != nil {
		t.Fatalf("IssueAccessRefreshPair: %v", err)
	}

	capture := &captureHandler{}
	handler := mw.RequireSessionFunc(sm, jwtMgr)(capture)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when using refresh token as bearer, got %d", w.Code)
	}
	// §2.3: the rejection must communicate that a refresh token was used.
	// Check both WWW-Authenticate and body for the "refresh" mention.
	wwwAuth := w.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("expected WWW-Authenticate header")
	}
	body := w.Body.String()
	mentionsRefresh := strings.Contains(strings.ToLower(wwwAuth), "refresh") ||
		strings.Contains(strings.ToLower(body), "refresh")
	if !mentionsRefresh {
		t.Errorf("response should mention refresh token rejection; WWW-Authenticate=%q body=%q", wwwAuth, body)
	}
	if capture.authMethod != "" {
		t.Errorf("handler should not have been called; authMethod=%q", capture.authMethod)
	}
}

// TestRequireSession_CookieValid: cookie only (no bearer) → 200, AuthMethod="cookie",
// SessionID non-empty.
func TestRequireSession_CookieValid(t *testing.T) {
	store := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	jwtMgr := newMWTestJWTManager(t, store, false)
	sm := makeSessionManager(t, store, cfg)

	user := mwTestUser()
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	sessID, cookieVal := makeEncryptedSessionCookie(t, sm, store, user.ID)

	capture := &captureHandler{}
	handler := mw.RequireSessionFunc(sm, jwtMgr)(capture)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "shark_session", Value: cookieVal})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid cookie, got %d", w.Code)
	}
	if capture.authMethod != "cookie" {
		t.Errorf("expected AuthMethod=cookie, got %q", capture.authMethod)
	}
	if capture.userID != user.ID {
		t.Errorf("expected UserID=%s, got %q", user.ID, capture.userID)
	}
	if capture.sessionID != sessID {
		t.Errorf("expected SessionID=%s, got %q", sessID, capture.sessionID)
	}
}

// TestRequireSession_NoAuth: no credentials → 401 with WWW-Authenticate header.
func TestRequireSession_NoAuth(t *testing.T) {
	store := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	jwtMgr := newMWTestJWTManager(t, store, false)
	sm := makeSessionManager(t, store, cfg)

	capture := &captureHandler{}
	handler := mw.RequireSessionFunc(sm, jwtMgr)(capture)

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with no credentials, got %d", w.Code)
	}
	wwwAuth := w.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("expected WWW-Authenticate header when no credentials provided")
	}
	if capture.authMethod != "" {
		t.Errorf("handler should not have been reached; authMethod=%q", capture.authMethod)
	}
}

// TestRequireSession_BothPresent: valid bearer + valid cookie → 200, AuthMethod="jwt"
// (bearer wins per §2.1 decision tree, cookie is ignored).
func TestRequireSession_BothPresent(t *testing.T) {
	store := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	jwtMgr := newMWTestJWTManager(t, store, false)
	sm := makeSessionManager(t, store, cfg)

	user := mwTestUser()
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	// Get encoded session cookie.
	sessID, cookieVal := makeEncryptedSessionCookie(t, sm, store, user.ID)
	_ = sessID

	// Issue a valid session JWT (token_type="session").
	token, err := jwtMgr.IssueSessionJWT(context.Background(), user, sessID, false)
	if err != nil {
		t.Fatalf("IssueSessionJWT: %v", err)
	}

	capture := &captureHandler{}
	handler := mw.RequireSessionFunc(sm, jwtMgr)(capture)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.AddCookie(&http.Cookie{Name: "shark_session", Value: cookieVal})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capture.authMethod != "jwt" {
		t.Errorf("expected AuthMethod=jwt (bearer wins), got %q", capture.authMethod)
	}
	if capture.userID != user.ID {
		t.Errorf("expected UserID=%s, got %q", user.ID, capture.userID)
	}
}
