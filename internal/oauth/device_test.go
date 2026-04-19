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

	"github.com/sharkauth/sharkauth/internal/storage"
)

// mountDeviceRouter adds device flow routes to a chi router.
func mountDeviceRouter(srv *Server) chi.Router {
	r := chi.NewRouter()
	r.Post("/oauth/token", srv.HandleToken)
	r.Post("/oauth/device", srv.HandleDeviceAuthorization)
	r.Get("/oauth/device/verify", srv.HandleDeviceVerify)
	r.Post("/oauth/device/verify", srv.HandleDeviceApprove)
	return r
}

// seedDeviceAgent creates an agent that supports device_code grant.
func seedDeviceAgent(t *testing.T, store storage.Store, clientID string) *storage.Agent {
	t.Helper()
	h := sha256.Sum256([]byte("test-secret"))
	agent := &storage.Agent{
		ID:               "agent_" + clientID,
		Name:             "Device Test Agent",
		Description:      "Tests device flow",
		ClientID:         clientID,
		ClientSecretHash: hex.EncodeToString(h[:]),
		ClientType:       "confidential",
		AuthMethod:       "client_secret_basic",
		RedirectURIs:     []string{"https://example.com/callback"},
		GrantTypes:       []string{"authorization_code", "client_credentials", "urn:ietf:params:oauth:grant-type:device_code"},
		ResponseTypes:    []string{"code"},
		Scopes:           []string{"openid", "profile", "read"},
		TokenLifetime:    900,
		Active:           true,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := store.CreateAgent(context.Background(), agent); err != nil {
		t.Fatalf("seedDeviceAgent: %v", err)
	}
	return agent
}

// createPendingDeviceCode creates a device code directly in the store (bypasses HTTP).
func createPendingDeviceCode(t *testing.T, store storage.Store, clientID, scope string, expiresIn time.Duration) (plainCode, userCode, hash string) {
	t.Helper()
	var err error
	plainCode, hash, err = generateDeviceCode()
	if err != nil {
		t.Fatalf("generateDeviceCode: %v", err)
	}
	userCode, err = generateUserCode()
	if err != nil {
		t.Fatalf("generateUserCode: %v", err)
	}
	dc := &storage.OAuthDeviceCode{
		DeviceCodeHash: hash,
		UserCode:       userCode,
		ClientID:       clientID,
		Scope:          scope,
		Status:         "pending",
		PollInterval:   5,
		ExpiresAt:      time.Now().UTC().Add(expiresIn),
		CreatedAt:      time.Now().UTC(),
	}
	if err := store.CreateDeviceCode(context.Background(), dc); err != nil {
		t.Fatalf("CreateDeviceCode: %v", err)
	}
	return plainCode, userCode, hash
}

// ---------------------------------------------------------------------------
// TestGenerateUserCode — validates format and charset
// ---------------------------------------------------------------------------

func TestGenerateUserCode(t *testing.T) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	validChars := make(map[rune]bool, len(charset))
	for _, c := range charset {
		validChars[c] = true
	}

	for i := 0; i < 50; i++ {
		code, err := generateUserCode()
		if err != nil {
			t.Fatalf("generateUserCode: %v", err)
		}
		// Must be exactly XXXX-XXXX (9 chars with dash at index 4).
		if len(code) != 9 {
			t.Errorf("expected len 9, got %d for %q", len(code), code)
		}
		if code[4] != '-' {
			t.Errorf("expected dash at index 4, got %q", code)
		}
		// All non-dash chars must be in valid charset.
		for _, c := range code {
			if c == '-' {
				continue
			}
			if !validChars[c] {
				t.Errorf("invalid char %q in user_code %q", c, code)
			}
		}
		// Ambiguous chars must be absent.
		for _, bad := range []rune{'I', 'O', '0', '1'} {
			if strings.ContainsRune(code, bad) {
				t.Errorf("ambiguous char %q found in user_code %q", bad, code)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Authorization_Success
// ---------------------------------------------------------------------------

func TestDevice_Authorization_Success(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedDeviceAgent(t, store, "device-client")

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{
		"client_id": {"device-client"},
		"scope":     {"openid read"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/device", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result deviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.DeviceCode == "" {
		t.Error("missing device_code")
	}
	if result.UserCode == "" {
		t.Error("missing user_code")
	}
	if len(result.UserCode) != 9 || result.UserCode[4] != '-' {
		t.Errorf("unexpected user_code format: %q", result.UserCode)
	}
	if result.VerificationURI == "" {
		t.Error("missing verification_uri")
	}
	if result.VerificationURIComplete == "" {
		t.Error("missing verification_uri_complete")
	}
	if result.ExpiresIn != 900 {
		t.Errorf("expected expires_in=900, got %d", result.ExpiresIn)
	}
	if result.Interval != 5 {
		t.Errorf("expected interval=5, got %d", result.Interval)
	}
}

// TestDevice_Authorization_UnknownClient
func TestDevice_Authorization_UnknownClient(t *testing.T) {
	srv, _ := newTestOAuthServer(t)

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{"client_id": {"nonexistent"}, "scope": {"openid"}}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/device", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Token_Pending — poll before user approves
// ---------------------------------------------------------------------------

func TestDevice_Token_Pending(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedDeviceAgent(t, store, "device-pending-client")

	plainCode, _, _ := createPendingDeviceCode(t, store, "device-pending-client", "openid", 15*time.Minute)

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {plainCode},
		"client_id":   {"device-pending-client"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "authorization_pending" {
		t.Errorf("expected authorization_pending, got %q", result["error"])
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Token_SlowDown — rapid polling triggers slow_down
// ---------------------------------------------------------------------------

func TestDevice_Token_SlowDown(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedDeviceAgent(t, store, "device-slowdown-client")

	plainCode, _, hash := createPendingDeviceCode(t, store, "device-slowdown-client", "openid", 15*time.Minute)

	// Pre-set last_polled_at to "just now" so the next poll within 5s triggers slow_down.
	if err := store.UpdateDeviceCodePolledAt(context.Background(), hash); err != nil {
		t.Fatalf("UpdateDeviceCodePolledAt: %v", err)
	}

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {plainCode},
		"client_id":   {"device-slowdown-client"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "slow_down" {
		t.Errorf("expected slow_down, got %q", result["error"])
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Token_Expired
// ---------------------------------------------------------------------------

func TestDevice_Token_Expired(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedDeviceAgent(t, store, "device-expired-client")

	// Create a device code already past its expiry.
	plainCode, _, _ := createPendingDeviceCode(t, store, "device-expired-client", "openid", -1*time.Minute)

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {plainCode},
		"client_id":   {"device-expired-client"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "expired_token" {
		t.Errorf("expected expired_token, got %q", result["error"])
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Token_Denied — user denied
// ---------------------------------------------------------------------------

func TestDevice_Token_Denied(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedDeviceAgent(t, store, "device-denied-client")

	plainCode, _, hash := createPendingDeviceCode(t, store, "device-denied-client", "openid", 15*time.Minute)

	// Simulate user denying.
	if err := store.UpdateDeviceCodeStatus(context.Background(), hash, "denied", ""); err != nil {
		t.Fatalf("UpdateDeviceCodeStatus: %v", err)
	}

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {plainCode},
		"client_id":   {"device-denied-client"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "access_denied" {
		t.Errorf("expected access_denied, got %q", result["error"])
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Token_Approved — full happy path
// ---------------------------------------------------------------------------

func TestDevice_Token_Approved(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedDeviceAgent(t, store, "device-approved-client")
	userID := seedUser(t, store, "deviceuser@example.com")

	plainCode, _, hash := createPendingDeviceCode(t, store, "device-approved-client", "openid read", 15*time.Minute)

	// Simulate user approving.
	if err := store.UpdateDeviceCodeStatus(context.Background(), hash, "approved", userID); err != nil {
		t.Fatalf("UpdateDeviceCodeStatus: %v", err)
	}

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {plainCode},
		"client_id":   {"device-approved-client"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if _, ok := result["access_token"]; !ok {
		t.Error("missing access_token")
	}
	if _, ok := result["refresh_token"]; !ok {
		t.Error("missing refresh_token")
	}
	if result["token_type"] != "bearer" {
		t.Errorf("expected bearer, got %v", result["token_type"])
	}
	if _, ok := result["expires_in"]; !ok {
		t.Error("missing expires_in")
	}
	if result["scope"] != "openid read" {
		t.Errorf("expected scope 'openid read', got %v", result["scope"])
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Verify_GET — renders code entry form
// ---------------------------------------------------------------------------

func TestDevice_Verify_GET(t *testing.T) {
	srv, _ := newTestOAuthServer(t)

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/oauth/device/verify")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Enter Device Code") {
		t.Error("expected 'Enter Device Code' heading in response")
	}
	if !strings.Contains(bodyStr, "user_code") {
		t.Error("expected user_code input field in response")
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Approve_POST_Denied — user submits denied decision via POST
// ---------------------------------------------------------------------------

func TestDevice_Approve_POST_Denied(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedDeviceAgent(t, store, "device-approve-denied-client")
	userID := seedUser(t, store, "approveuser@example.com")

	_, userCode, _ := createPendingDeviceCode(t, store, "device-approve-denied-client", "openid", 15*time.Minute)

	r := chi.NewRouter()
	r.Post("/oauth/device/verify", srv.HandleDeviceApprove)

	ts := httptest.NewServer(r)
	defer ts.Close()

	form := url.Values{
		"user_code": {userCode},
		"approved":  {"false"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/device/verify", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-User-ID", userID)

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	// Should show error page (denied) with 200.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Denied") && !strings.Contains(bodyStr, "denied") && !strings.Contains(bodyStr, "Deny") {
		t.Errorf("expected denial message in response, got: %.300s", bodyStr)
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Token_InvalidCode — unknown device code returns error
// ---------------------------------------------------------------------------

func TestDevice_Token_InvalidCode(t *testing.T) {
	srv, _ := newTestOAuthServer(t)

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {"totally-made-up-device-code"},
		"client_id":   {"nobody"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %q", result["error"])
	}
}

// ---------------------------------------------------------------------------
// TestDevice_Authorization_MissingClientID
// ---------------------------------------------------------------------------

func TestDevice_Authorization_MissingClientID(t *testing.T) {
	srv, _ := newTestOAuthServer(t)

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{"scope": {"openid"}}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/device", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "invalid_request" {
		t.Errorf("expected invalid_request, got %q", result["error"])
	}
}

// TestDevice_Token_Approved_StoresTokens verifies that tokens end up in the DB.
func TestDevice_Token_Approved_StoresTokens(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedDeviceAgent(t, store, "device-store-client")
	userID := seedUser(t, store, "storeuser@example.com")

	plainCode, _, hash := createPendingDeviceCode(t, store, "device-store-client", "openid", 15*time.Minute)
	if err := store.UpdateDeviceCodeStatus(context.Background(), hash, "approved", userID); err != nil {
		t.Fatalf("UpdateDeviceCodeStatus: %v", err)
	}

	ts := httptest.NewServer(mountDeviceRouter(srv))
	defer ts.Close()

	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {plainCode},
		"client_id":   {"device-store-client"},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck

	// Verify that the access token can be found in the DB by its hash.
	accessToken, _ := result["access_token"].(string)
	if accessToken == "" {
		t.Fatal("no access_token in response")
	}
	h := sha256.Sum256([]byte(accessToken))
	tokenHash := hex.EncodeToString(h[:])

	tok, err := store.GetOAuthTokenByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatalf("token not found in DB: %v", err)
	}
	if tok.UserID != userID {
		t.Errorf("expected UserID=%q, got %q", userID, tok.UserID)
	}
	if tok.ClientID != "device-store-client" {
		t.Errorf("expected ClientID=device-store-client, got %q", tok.ClientID)
	}
	if tok.TokenType != "access" {
		t.Errorf("expected TokenType=access, got %q", tok.TokenType)
	}
}
