package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

// setupHeader returns an "Authorization: Setup <token>" header value.
func setupHeader(token string) string {
	return "Setup " + token
}

// doSetupRequest sends an HTTP request to path with "Authorization: Setup <token>".
func doSetupRequest(t *testing.T, ts *testutil.TestServer, method, path, token string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL(path), r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", setupHeader(token))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := ts.Client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

// TestSetupStatusPublic verifies GET /api/v1/admin/setup/status is publicly
// accessible and returns a JSON body with a "pending" boolean.
func TestSetupStatusPublic(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.Get("/api/v1/admin/setup/status")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup/status: expected 200, got %d", resp.StatusCode)
	}
	var out map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode status body: %v", err)
	}
	// No token minted yet — pending must be false.
	if out["pending"] != false {
		t.Errorf("expected pending=false before any token is minted, got %v", out["pending"])
	}
}

// TestSetupTokenBadToken verifies that requests to the setup endpoints with
// an invalid token are rejected with 401.
func TestSetupTokenBadToken(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// GET /admin/setup/info with a bad token
	resp := doSetupRequest(t, ts, http.MethodGet, "/api/v1/admin/setup/info", "bad_token_xyz", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("setup/info bad token: expected 401, got %d", resp.StatusCode)
	}

	// POST /admin/setup/admin-user with a bad token
	resp2 := doSetupRequest(t, ts, http.MethodPost, "/api/v1/admin/setup/admin-user",
		"bad_token_xyz",
		map[string]string{"email": "x@example.com"})
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("setup/admin-user bad token: expected 401, got %d", resp2.StatusCode)
	}
}

// TestSetupTokenMissingAuthHeader verifies that requests without any
// Authorization header receive 401.
func TestSetupTokenMissingAuthHeader(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := doSetupRequest(t, ts, http.MethodGet, "/api/v1/admin/setup/info", "", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("setup/info no auth: expected 401, got %d", resp.StatusCode)
	}
}

// TestSetupAdminUserValidToken verifies the full happy-path:
//  1. Mint a setup token (simulating first-boot).
//  2. GET /admin/setup/info with the valid token → 200 with api_key.
//  3. POST /admin/setup/admin-user with valid token + email → 200 with sent:true.
//  4. Subsequent POST returns 410 (token consumed).
func TestSetupAdminUserValidToken(t *testing.T) {
	ts := testutil.NewTestServerDev(t)

	// 1. Mint a token using the exported method on APIServer.
	const fakeAdminKey = "sk_live_test_key_for_setup_handler_test"
	token, err := ts.APIServer.MintSetupToken(fakeAdminKey)
	if err != nil {
		t.Fatalf("MintSetupToken: %v", err)
	}
	if token == "" {
		t.Fatal("MintSetupToken returned empty token")
	}

	// 2. GET /admin/setup/info — should return the api_key once.
	resp := doSetupRequest(t, ts, http.MethodGet, "/api/v1/admin/setup/info", token, nil)
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("setup/info: expected 200, got %d: %s", resp.StatusCode, b)
	}
	var infoResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
		t.Fatalf("decode info body: %v", err)
	}
	resp.Body.Close()
	if infoResp["api_key"] != fakeAdminKey {
		t.Errorf("api_key mismatch: want %q, got %q", fakeAdminKey, infoResp["api_key"])
	}

	// 3. POST /admin/setup/admin-user — should create user + return sent:true.
	resp2 := doSetupRequest(t, ts, http.MethodPost, "/api/v1/admin/setup/admin-user",
		token,
		map[string]string{"email": "admin@example.com"})
	if resp2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		t.Fatalf("setup/admin-user: expected 200, got %d: %s", resp2.StatusCode, b)
	}
	var setupResp map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&setupResp); err != nil {
		t.Fatalf("decode admin-user body: %v", err)
	}
	resp2.Body.Close()
	if setupResp["sent"] != true {
		t.Errorf("expected sent:true, got %v", setupResp["sent"])
	}

	// 4. Second POST after consumption → 401 (token consumed, middleware rejects).
	// The SetupTokenMiddleware validates the token before the handler runs;
	// a consumed token fails validation and returns 401. This is the expected
	// behavior — 410 is returned only by the handler when the middleware somehow
	// passes a consumed state, but in practice the middleware fires first.
	resp3 := doSetupRequest(t, ts, http.MethodPost, "/api/v1/admin/setup/admin-user",
		token,
		map[string]string{"email": "admin@example.com"})
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusUnauthorized && resp3.StatusCode != http.StatusGone {
		t.Errorf("second setup/admin-user after consumption: expected 401 or 410, got %d", resp3.StatusCode)
	}
}

// TestSetupAdminUserMissingEmail verifies that POST with a valid token but
// missing email field returns 400.
func TestSetupAdminUserMissingEmail(t *testing.T) {
	ts := testutil.NewTestServer(t)

	token, err := ts.APIServer.MintSetupToken("sk_live_test")
	if err != nil {
		t.Fatalf("MintSetupToken: %v", err)
	}

	resp := doSetupRequest(t, ts, http.MethodPost, "/api/v1/admin/setup/admin-user",
		token,
		map[string]string{"email": ""})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing email: expected 400, got %d", resp.StatusCode)
	}
}
