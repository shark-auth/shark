package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

func TestPasskeyRegisterBegin(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Signup to get a session
	resp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    "passkey@example.com",
		"password": "securepassword123",
		"name":     "Passkey User",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201 for signup, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// POST /passkey/register/begin (authenticated)
	resp = ts.PostJSON("/api/v1/auth/passkey/register/begin", nil)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for register/begin, got %d: %s", resp.StatusCode, body)
	}

	var beginResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&beginResp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	resp.Body.Close()

	// Verify challenge key is returned
	challengeKey, ok := beginResp["challengeKey"].(string)
	if !ok || challengeKey == "" {
		t.Fatal("expected non-empty challengeKey in response")
	}

	// Verify publicKey options are returned
	publicKey, ok := beginResp["publicKey"].(map[string]interface{})
	if !ok {
		t.Fatal("expected publicKey object in response")
	}

	// Verify rp.id
	rp, ok := publicKey["rp"].(map[string]interface{})
	if !ok {
		t.Fatal("expected rp object in publicKey")
	}
	if rp["id"] != "localhost" {
		t.Fatalf("expected rp.id = localhost, got %v", rp["id"])
	}
	if rp["name"] != "SharkAuth Test" {
		t.Fatalf("expected rp.name = SharkAuth Test, got %v", rp["name"])
	}

	// Verify challenge exists and is non-empty
	challenge, ok := publicKey["challenge"].(string)
	if !ok || challenge == "" {
		t.Fatal("expected non-empty challenge in publicKey")
	}

	// Verify user info
	user, ok := publicKey["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user object in publicKey")
	}
	if user["name"] != "passkey@example.com" {
		t.Fatalf("expected user.name = passkey@example.com, got %v", user["name"])
	}
	if user["displayName"] != "Passkey User" {
		t.Fatalf("expected user.displayName = Passkey User, got %v", user["displayName"])
	}

	// Verify pubKeyCredParams includes ES256 and RS256
	params, ok := publicKey["pubKeyCredParams"].([]interface{})
	if !ok || len(params) == 0 {
		t.Fatal("expected non-empty pubKeyCredParams")
	}

	algSet := make(map[float64]bool)
	for _, p := range params {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if alg, ok := pm["alg"].(float64); ok {
			algSet[alg] = true
		}
	}
	if !algSet[-7] {
		t.Fatal("expected ES256 (alg=-7) in pubKeyCredParams")
	}
	if !algSet[-257] {
		t.Fatal("expected RS256 (alg=-257) in pubKeyCredParams")
	}
}

func TestPasskeyRegisterBeginRequiresAuth(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// POST without being logged in
	resp := ts.PostJSON("/api/v1/auth/passkey/register/begin", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		body := readBody(t, resp)
		t.Fatalf("expected 401 for unauthenticated register/begin, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}

func TestPasskeyCredentialCRUD(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Signup to get a session
	resp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    "passkey-crud@example.com",
		"password": "securepassword123",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201 for signup, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// List credentials (should be empty)
	resp = ts.Get("/api/v1/auth/passkey/credentials")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for credentials list, got %d: %s", resp.StatusCode, body)
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	resp.Body.Close()

	creds, ok := listResp["credentials"].([]interface{})
	if !ok {
		t.Fatal("expected credentials array in response")
	}
	if len(creds) != 0 {
		t.Fatalf("expected 0 credentials, got %d", len(creds))
	}

	// Delete non-existent credential -> 404
	req, err := http.NewRequest("DELETE", ts.URL("/api/v1/auth/passkey/credentials/pk_nonexistent"), nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	resp, err = ts.Client.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		body := readBody(t, resp)
		t.Fatalf("expected 404 for deleting non-existent cred, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Rename non-existent credential -> 404
	resp = ts.PatchJSON("/api/v1/auth/passkey/credentials/pk_nonexistent", map[string]string{
		"name": "My Passkey",
	})
	if resp.StatusCode != http.StatusNotFound {
		body := readBody(t, resp)
		t.Fatalf("expected 404 for renaming non-existent cred, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}

func TestPasskeyLoginBeginDiscoverable(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// POST /passkey/login/begin with empty body (discoverable flow)
	resp := ts.PostJSON("/api/v1/auth/passkey/login/begin", map[string]string{})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for login/begin discoverable, got %d: %s", resp.StatusCode, body)
	}

	var loginResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	resp.Body.Close()

	challengeKey, ok := loginResp["challengeKey"].(string)
	if !ok || challengeKey == "" {
		t.Fatal("expected non-empty challengeKey")
	}

	publicKey, ok := loginResp["publicKey"].(map[string]interface{})
	if !ok {
		t.Fatal("expected publicKey object")
	}

	// Discoverable flow should have rpId but no allowCredentials
	if publicKey["rpId"] != "localhost" {
		t.Fatalf("expected rpId = localhost, got %v", publicKey["rpId"])
	}

	challenge, ok := publicKey["challenge"].(string)
	if !ok || challenge == "" {
		t.Fatal("expected non-empty challenge")
	}
}
