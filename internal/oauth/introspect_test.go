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
	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/shark-auth/shark/internal/storage"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// mountIntrospectRevokeRouter mounts /oauth/introspect and /oauth/revoke.
func mountIntrospectRevokeRouter(srv *Server) chi.Router {
	r := chi.NewRouter()
	r.Post("/oauth/token", srv.HandleToken)
	r.Post("/oauth/introspect", srv.HandleIntrospect)
	r.Post("/oauth/revoke", srv.HandleRevoke)
	return r
}

// seedAPIKey creates a test admin API key (scope "*") and returns the raw key.
func seedAPIKey(t *testing.T, store storage.Store) string {
	t.Helper()
	return seedAPIKeyWithScopes(t, store, "key_test_admin", "sk"+"_live_testadminkey0000000000000000", []string{"*"})
}

func seedAPIKeyWithScopes(t *testing.T, store storage.Store, id, rawKey string, scopes []string) string {
	t.Helper()
	keyHash := sha256.Sum256([]byte(rawKey))
	scopeJSONBytes, err := json.Marshal(scopes)
	if err != nil {
		t.Fatalf("marshalling scopes: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	key := &storage.APIKey{
		ID:        id,
		Name:      "Test Admin Key",
		KeyHash:   hex.EncodeToString(keyHash[:]),
		KeyPrefix: "testadmi",
		KeySuffix: "0000",
		Scopes:    string(scopeJSONBytes),
		RateLimit: 1000,
		CreatedAt: now,
	}
	if err := store.CreateAPIKey(context.Background(), key); err != nil {
		t.Fatalf("seeding admin api key: %v", err)
	}
	return rawKey
}

func extractTestJTI(t *testing.T, tokenStr string) string {
	t.Helper()
	parser := gojwt.NewParser(gojwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenStr, gojwt.MapClaims{})
	if err != nil {
		t.Fatalf("parsing JWT for test: %v", err)
	}
	claims, ok := token.Claims.(gojwt.MapClaims)
	if !ok {
		t.Fatal("unexpected claims type")
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		t.Fatal("expected JWT jti")
	}
	return jti
}

// obtainAccessToken performs client_credentials grant and returns the raw access_token.
func obtainAccessToken(t *testing.T, ts *httptest.Server, clientID, clientSecret string) string {
	t.Helper()
	form := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {"openid"},
	}
	req, err := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("building token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending token request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 for token, got %d: %s", resp.StatusCode, body)
	}
	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp) //nolint:errcheck
	tok, _ := tokenResp["access_token"].(string)
	if tok == "" {
		t.Fatal("no access_token in response")
	}
	return tok
}

// doIntrospect sends a POST /oauth/introspect and decodes the JSON response.
func doIntrospect(t *testing.T, ts *httptest.Server, token string, authFn func(*http.Request)) (int, map[string]interface{}) {
	t.Helper()
	form := url.Values{"token": {token}}
	req, err := http.NewRequest("POST", ts.URL+"/oauth/introspect", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("building introspect request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if authFn != nil {
		authFn(req)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending introspect request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result) //nolint:errcheck
	return resp.StatusCode, result
}

// basicAuth returns a request-modifier that sets HTTP Basic auth.
func basicAuth(clientID, secret string) func(*http.Request) {
	return func(r *http.Request) {
		r.SetBasicAuth(clientID, secret)
	}
}

// bearerAuth returns a request-modifier that sets a Bearer token header.
func bearerAuth(token string) func(*http.Request) {
	return func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	}
}

// doRevoke sends a POST /oauth/revoke and returns the HTTP status code.
func doRevoke(t *testing.T, ts *httptest.Server, token string, authFn func(*http.Request)) int {
	t.Helper()
	form := url.Values{"token": {token}}
	req, err := http.NewRequest("POST", ts.URL+"/oauth/revoke", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("building revoke request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if authFn != nil {
		authFn(req)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending revoke request: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// ---------------------------------------------------------------------------
// Introspection tests
// ---------------------------------------------------------------------------

// TestIntrospect_ActiveToken verifies that a valid, active access token returns
// full claims with active:true.
func TestIntrospect_ActiveToken(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "introspect-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "introspect-client", "test-secret")

	status, result := doIntrospect(t, ts, accessToken, basicAuth("introspect-client", "test-secret"))

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if result["active"] != true {
		t.Errorf("expected active=true, got %v", result["active"])
	}
	if result["client_id"] != "introspect-client" {
		t.Errorf("expected client_id=introspect-client, got %v", result["client_id"])
	}
	if result["token_type"] != "Bearer" {
		t.Errorf("expected token_type=Bearer, got %v", result["token_type"])
	}
	if _, hasJTI := result["jti"]; !hasJTI {
		t.Error("expected jti field in response")
	}
	if _, hasExp := result["exp"]; !hasExp {
		t.Error("expected exp field in response")
	}
	if result["iss"] != srv.Issuer {
		t.Errorf("expected iss=%s, got %v", srv.Issuer, result["iss"])
	}
}

// TestIntrospect_RevokedToken verifies that a revoked token returns {active:false}
// with no other claims.
func TestIntrospect_RevokedToken(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "revoke-intros-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "revoke-intros-client", "test-secret")

	// Revoke it.
	revokeStatus := doRevoke(t, ts, accessToken, basicAuth("revoke-intros-client", "test-secret"))
	if revokeStatus != http.StatusOK {
		t.Fatalf("expected 200 from revoke, got %d", revokeStatus)
	}

	// Introspect the revoked token.
	status, result := doIntrospect(t, ts, accessToken, basicAuth("revoke-intros-client", "test-secret"))
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if result["active"] != false {
		t.Errorf("expected active=false for revoked token, got %v", result["active"])
	}
	// Per RFC 7662 Â§2.2, no other fields should be present when active=false.
	if _, hasScope := result["scope"]; hasScope {
		t.Error("scope should not be present in inactive response")
	}
	if _, hasSub := result["sub"]; hasSub {
		t.Error("sub should not be present in inactive response")
	}
}

// TestIntrospect_ExpiredToken verifies that an expired token returns {active:false}.
func TestIntrospect_ExpiredToken(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "expired-intros-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	// Manually insert a token that is already expired.
	expiredToken := &storage.OAuthToken{
		ID:        "tok_expired001",
		JTI:       "jti-expired-test-001",
		ClientID:  "expired-intros-client",
		TokenType: "access",
		TokenHash: hashSignature("fake-expired-token-sig"),
		Scope:     "openid",
		ExpiresAt: time.Now().UTC().Add(-10 * time.Minute), // already expired
		CreatedAt: time.Now().UTC().Add(-25 * time.Minute),
	}
	if err := store.CreateOAuthToken(context.Background(), expiredToken); err != nil {
		t.Fatalf("inserting expired token: %v", err)
	}

	// Use JTI lookup via a synthetic JWT-style lookup by searching by JTI.
	// Build a fake opaque token whose hash matches the stored record.
	fakeRawToken := "fake-expired-token-sig"

	status, result := doIntrospect(t, ts, fakeRawToken, basicAuth("expired-intros-client", "test-secret"))
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if result["active"] != false {
		t.Errorf("expected active=false for expired token, got %v", result["active"])
	}
}

// TestIntrospect_UnknownToken verifies that a token not in the DB returns {active:false}.
func TestIntrospect_UnknownToken(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "unknown-intros-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	status, result := doIntrospect(t, ts, "this.token.does.not.exist.at.all", basicAuth("unknown-intros-client", "test-secret"))
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if result["active"] != false {
		t.Errorf("expected active=false for unknown token, got %v", result["active"])
	}
}

func TestIntrospect_ForgedJWTWithKnownJTIInactive(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "forged-jti-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "forged-jti-client", "test-secret")
	jti := extractTestJTI(t, accessToken)

	forged := gojwt.NewWithClaims(gojwt.SigningMethodHS256, gojwt.MapClaims{
		"iss": srv.Issuer,
		"sub": "attacker",
		"jti": jti,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	forgedToken, err := forged.SignedString([]byte("wrong-signing-key"))
	if err != nil {
		t.Fatalf("signing forged JWT: %v", err)
	}

	status, result := doIntrospect(t, ts, forgedToken, basicAuth("forged-jti-client", "test-secret"))
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, result)
	}
	if result["active"] != false {
		t.Fatalf("expected forged JWT to be inactive, got %v", result)
	}
}

// TestIntrospect_NoAuth verifies that a request without client credentials returns 401.
func TestIntrospect_NoAuth(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "noauth-intros-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	// Obtain a real token so we have something to introspect.
	accessToken := obtainAccessToken(t, ts, "noauth-intros-client", "test-secret")

	// Introspect without any auth.
	status, result := doIntrospect(t, ts, accessToken, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %v", status, result)
	}
}

// TestIntrospect_InvalidClient verifies that wrong credentials return 401.
func TestIntrospect_InvalidClient(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "invalid-client-intros", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "invalid-client-intros", "test-secret")

	// Use the wrong secret.
	status, result := doIntrospect(t, ts, accessToken, basicAuth("invalid-client-intros", "wrong-secret"))
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %v", status, result)
	}
}

// TestIntrospect_AdminAuth verifies that an admin API key can introspect any token.
func TestIntrospect_AdminAuth(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "admin-intros-client", false)
	adminKey := seedAPIKey(t, store)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "admin-intros-client", "test-secret")

	// Use admin Bearer key.
	status, result := doIntrospect(t, ts, accessToken, bearerAuth(adminKey))
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, result)
	}
	if result["active"] != true {
		t.Errorf("expected active=true with admin key, got %v", result["active"])
	}
	if result["client_id"] != "admin-intros-client" {
		t.Errorf("expected client_id=admin-intros-client, got %v", result["client_id"])
	}
}

func TestIntrospect_BearerAPIKeyRequiresAdminScope(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "nonadmin-intros-client", false)
	nonAdminKey := seedAPIKeyWithScopes(t, store, "key_nonadmin", "sk"+"_live_nonadminkey0000000000000000", []string{"tokens:read"})
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "nonadmin-intros-client", "test-secret")

	status, result := doIntrospect(t, ts, accessToken, bearerAuth(nonAdminKey))
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-admin API key, got %d: %v", status, result)
	}
}

// ---------------------------------------------------------------------------
// Revocation tests
// ---------------------------------------------------------------------------

// TestRevoke_AccessToken verifies that a revoked access token is then inactive.
func TestRevoke_AccessToken(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "revoke-access-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "revoke-access-client", "test-secret")

	// Confirm active before revocation.
	_, beforeResult := doIntrospect(t, ts, accessToken, basicAuth("revoke-access-client", "test-secret"))
	if beforeResult["active"] != true {
		t.Fatalf("expected token to be active before revocation, got %v", beforeResult["active"])
	}

	// Revoke.
	revokeStatus := doRevoke(t, ts, accessToken, basicAuth("revoke-access-client", "test-secret"))
	if revokeStatus != http.StatusOK {
		t.Fatalf("expected 200 from revoke, got %d", revokeStatus)
	}

	// Confirm inactive after revocation.
	_, afterResult := doIntrospect(t, ts, accessToken, basicAuth("revoke-access-client", "test-secret"))
	if afterResult["active"] != false {
		t.Errorf("expected active=false after revocation, got %v", afterResult["active"])
	}
}

// TestRevoke_RefreshToken verifies that a refresh token revocation marks it inactive.
func TestRevoke_RefreshToken(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "revoke-refresh-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	// Manually insert a refresh token record.
	familyID := "test-family-001"
	refreshSig := "refresh-token-raw-sig-for-revoke-test"
	refreshHash := hashSignature(refreshSig)
	refreshToken := &storage.OAuthToken{
		ID:        "tok_refresh001",
		JTI:       "jti-refresh-revoke-001",
		ClientID:  "revoke-refresh-client",
		TokenType: "refresh",
		TokenHash: refreshHash,
		Scope:     "openid offline_access",
		FamilyID:  familyID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateOAuthToken(context.Background(), refreshToken); err != nil {
		t.Fatalf("inserting refresh token: %v", err)
	}

	// Revoke via the raw opaque token string.
	revokeStatus := doRevoke(t, ts, refreshSig, basicAuth("revoke-refresh-client", "test-secret"))
	if revokeStatus != http.StatusOK {
		t.Fatalf("expected 200 from revoke, got %d", revokeStatus)
	}

	// Verify the token is now revoked in the DB.
	tok, err := store.GetOAuthTokenByJTI(context.Background(), "jti-refresh-revoke-001")
	if err != nil {
		t.Fatalf("looking up refresh token: %v", err)
	}
	if tok.RevokedAt == nil {
		t.Error("expected refresh token to be revoked")
	}
}

// TestRevoke_UnknownToken verifies that revoking an unknown token still returns 200.
func TestRevoke_UnknownToken(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "revoke-unknown-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	revokeStatus := doRevoke(t, ts, "completely-unknown-token-xyz", basicAuth("revoke-unknown-client", "test-secret"))
	if revokeStatus != http.StatusOK {
		t.Fatalf("expected 200 per RFC 7009 for unknown token, got %d", revokeStatus)
	}
}

// TestRevoke_WrongClient verifies that a client cannot revoke another client's token.
// Per RFC 7009, it still returns 200 but the token is NOT revoked (no-op).
func TestRevoke_WrongClient(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "owner-client", false)
	seedAgent(t, store, "other-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	// Obtain a token for owner-client.
	accessToken := obtainAccessToken(t, ts, "owner-client", "test-secret")

	// Attempt to revoke it as other-client.
	revokeStatus := doRevoke(t, ts, accessToken, basicAuth("other-client", "test-secret"))
	// Per RFC 7009 Â§2.2, must return 200 even for unauthorized attempts.
	if revokeStatus != http.StatusOK {
		t.Fatalf("expected 200 per RFC 7009, got %d", revokeStatus)
	}

	// Token must still be active (revocation was a no-op).
	_, result := doIntrospect(t, ts, accessToken, basicAuth("owner-client", "test-secret"))
	if result["active"] != true {
		t.Errorf("expected token to remain active after wrong-client revoke attempt, got %v", result["active"])
	}
}

func TestRevoke_ForgedJWTWithKnownJTIDoesNotRevoke(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "forged-revoke-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "forged-revoke-client", "test-secret")
	jti := extractTestJTI(t, accessToken)
	forged := gojwt.NewWithClaims(gojwt.SigningMethodHS256, gojwt.MapClaims{
		"iss": srv.Issuer,
		"sub": "attacker",
		"jti": jti,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	forgedToken, err := forged.SignedString([]byte("wrong-signing-key"))
	if err != nil {
		t.Fatalf("signing forged JWT: %v", err)
	}

	revokeStatus := doRevoke(t, ts, forgedToken, basicAuth("forged-revoke-client", "test-secret"))
	if revokeStatus != http.StatusOK {
		t.Fatalf("expected 200 per RFC 7009, got %d", revokeStatus)
	}

	_, result := doIntrospect(t, ts, accessToken, basicAuth("forged-revoke-client", "test-secret"))
	if result["active"] != true {
		t.Fatalf("expected real token to remain active, got %v", result)
	}
}

// TestRevoke_NoAuth verifies that revocation without client auth returns 401.
func TestRevoke_NoAuth(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "revoke-noauth-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	accessToken := obtainAccessToken(t, ts, "revoke-noauth-client", "test-secret")
	revokeStatus := doRevoke(t, ts, accessToken, nil)
	if revokeStatus != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", revokeStatus)
	}
}
