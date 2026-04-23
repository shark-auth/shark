package oauth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// mountDCRRouter sets up a chi router with all DCR endpoints.
func mountDCRRouter(srv *Server) chi.Router {
	r := chi.NewRouter()
	r.Post("/oauth/register", srv.HandleDCRRegister)
	r.Get("/oauth/register/{client_id}", srv.HandleDCRGet)
	r.Put("/oauth/register/{client_id}", srv.HandleDCRUpdate)
	r.Delete("/oauth/register/{client_id}", srv.HandleDCRDelete)
	return r
}

// postJSON sends a POST request with a JSON body.
func postJSON(t *testing.T, ts *httptest.Server, path string, body interface{}, authToken string) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	return resp
}

// getWithBearer sends a GET request with a Bearer token.
func getWithBearer(t *testing.T, ts *httptest.Server, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	return resp
}

// putJSON sends a PUT request with a JSON body and Bearer token.
func putJSON(t *testing.T, ts *httptest.Server, path string, body interface{}, token string) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPut, ts.URL+path, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	return resp
}

// deleteWithBearer sends a DELETE request with a Bearer token.
func deleteWithBearer(t *testing.T, ts *httptest.Server, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	return resp
}

// decodeBody decodes the response body into a map.
func decodeBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("decoding response body: %v (raw: %s)", err, body)
	}
	return m
}

// TestDCR_Register_Success verifies that a valid POST to /oauth/register
// returns 201 with all required fields per RFC 7591 §3.2.1.
func TestDCR_Register_Success(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Test Client",
		"redirect_uris": []string{"https://example.com/callback"},
		"grant_types":   []string{"authorization_code"},
		"scope":         "openid profile",
	}

	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// Verify required fields.
	for _, field := range []string{"client_id", "client_secret", "client_id_issued_at",
		"client_secret_expires_at", "registration_access_token", "registration_client_uri",
		"client_name", "grant_types", "response_types", "token_endpoint_auth_method"} {
		if _, ok := result[field]; !ok {
			t.Errorf("response missing required field %q", field)
		}
	}

	// Verify client_id has the expected prefix.
	clientID, _ := result["client_id"].(string)
	if len(clientID) < len("shark_dcr_") || clientID[:10] != "shark_dcr_" {
		t.Errorf("expected client_id to start with 'shark_dcr_', got %q", clientID)
	}

	// Verify registration_client_uri format.
	regURI, _ := result["registration_client_uri"].(string)
	if regURI == "" {
		t.Error("registration_client_uri is empty")
	}

	// Verify client_secret_expires_at is 0 (never).
	if exp, ok := result["client_secret_expires_at"].(float64); !ok || exp != 0 {
		t.Errorf("expected client_secret_expires_at=0, got %v", result["client_secret_expires_at"])
	}

	// Verify scope is passed through.
	if scope, _ := result["scope"].(string); scope != "openid profile" {
		t.Errorf("expected scope='openid profile', got %q", scope)
	}
}

// TestDCR_Register_ClientCredentials verifies registration with client_credentials grant
// (no redirect_uris required).
func TestDCR_Register_ClientCredentials(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name": "CC Client",
		"grant_types": []string{"client_credentials"},
	}

	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
}

// TestDCR_Register_MissingRedirectURI tests that registering authorization_code
// without redirect_uris returns 400 invalid_redirect_uri per RFC 7591.
func TestDCR_Register_MissingRedirectURI(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name": "Bad Client",
		"grant_types": []string{"authorization_code"},
		// redirect_uris intentionally omitted.
	}

	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "invalid_redirect_uri" {
		t.Errorf("expected error=invalid_redirect_uri, got %q", result["error"])
	}
}

// TestDCR_Register_InvalidRedirectURI tests that an http:// non-localhost URI is rejected.
func TestDCR_Register_InvalidRedirectURI(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Bad Redirect Client",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{"http://evil.com/steal"},
	}

	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "invalid_redirect_uri" {
		t.Errorf("expected error=invalid_redirect_uri, got %q", result["error"])
	}
}

// TestDCR_Register_LocalhostHTTPAllowed verifies that http://localhost is allowed.
func TestDCR_Register_LocalhostHTTPAllowed(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Native App",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{"http://localhost:9000/callback"},
	}

	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
}

// TestDCR_Register_InvalidGrantType tests that an unsupported grant type returns 400.
func TestDCR_Register_InvalidGrantType(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Bad Grant Client",
		"grant_types":   []string{"magic_unicorn"},
		"redirect_uris": []string{"https://example.com/callback"},
	}

	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "invalid_client_metadata" {
		t.Errorf("expected error=invalid_client_metadata, got %q", result["error"])
	}
}

// TestDCR_Register_MissingClientName verifies that omitting client_name → 400.
func TestDCR_Register_MissingClientName(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"redirect_uris": []string{"https://example.com/callback"},
		"grant_types":   []string{"authorization_code"},
	}

	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}
}

// TestDCR_Get_ValidToken verifies GET with a valid registration_access_token returns 200.
func TestDCR_Get_ValidToken(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	// Register first.
	payload := map[string]interface{}{
		"client_name":   "Get Test Client",
		"redirect_uris": []string{"https://example.com/callback"},
		"grant_types":   []string{"authorization_code"},
		"scope":         "openid",
	}
	regResp := postJSON(t, ts, "/oauth/register", payload, "")
	var regResult map[string]interface{}
	json.NewDecoder(regResp.Body).Decode(&regResult) //nolint:errcheck
	regResp.Body.Close()

	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("registration failed with status %d", regResp.StatusCode)
	}

	clientID := regResult["client_id"].(string)
	regToken := regResult["registration_access_token"].(string)

	// GET the registration.
	resp := getWithBearer(t, ts, "/oauth/register/"+clientID, regToken)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck

	if result["client_id"] != clientID {
		t.Errorf("expected client_id=%q, got %q", clientID, result["client_id"])
	}
	if result["client_name"] != "Get Test Client" {
		t.Errorf("expected client_name='Get Test Client', got %q", result["client_name"])
	}
	// GET should NOT return client_secret or registration_access_token.
	if _, ok := result["client_secret"]; ok {
		t.Error("GET response should not contain client_secret")
	}
}

// TestDCR_Get_InvalidToken verifies GET with wrong token returns 401.
func TestDCR_Get_InvalidToken(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	// Register first.
	payload := map[string]interface{}{
		"client_name":   "Invalid Token Test",
		"redirect_uris": []string{"https://example.com/cb"},
		"grant_types":   []string{"authorization_code"},
	}
	regResp := postJSON(t, ts, "/oauth/register", payload, "")
	var regResult map[string]interface{}
	json.NewDecoder(regResp.Body).Decode(&regResult) //nolint:errcheck
	regResp.Body.Close()

	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("registration failed with status %d", regResp.StatusCode)
	}

	clientID, _ := regResult["client_id"].(string)

	// Use a wrong token.
	resp := getWithBearer(t, ts, "/oauth/register/"+clientID, "wrongtoken")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "invalid_token" {
		t.Errorf("expected error=invalid_token, got %q", result["error"])
	}
}

// TestDCR_Get_MissingToken verifies GET with no token returns 401.
func TestDCR_Get_MissingToken(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	// Register first.
	payload := map[string]interface{}{
		"client_name":   "No Token Test",
		"redirect_uris": []string{"https://example.com/cb"},
		"grant_types":   []string{"authorization_code"},
	}
	regResp := postJSON(t, ts, "/oauth/register", payload, "")
	var regResult map[string]interface{}
	json.NewDecoder(regResp.Body).Decode(&regResult) //nolint:errcheck
	regResp.Body.Close()

	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("registration failed with status %d", regResp.StatusCode)
	}

	clientID, _ := regResult["client_id"].(string)

	resp := getWithBearer(t, ts, "/oauth/register/"+clientID, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
}

// TestDCR_Update_Success verifies PUT updates client metadata.
func TestDCR_Update_Success(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	// Register first.
	payload := map[string]interface{}{
		"client_name":   "Update Test Client",
		"redirect_uris": []string{"https://example.com/callback"},
		"grant_types":   []string{"authorization_code"},
		"scope":         "openid",
	}
	regResp := postJSON(t, ts, "/oauth/register", payload, "")
	var regResult map[string]interface{}
	json.NewDecoder(regResp.Body).Decode(&regResult) //nolint:errcheck
	regResp.Body.Close()

	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("registration failed with status %d", regResp.StatusCode)
	}

	clientID := regResult["client_id"].(string)
	regToken := regResult["registration_access_token"].(string)

	// Update the client.
	updatePayload := map[string]interface{}{
		"client_name":   "Updated Client Name",
		"redirect_uris": []string{"https://example.com/callback", "https://example.com/alt-cb"},
		"grant_types":   []string{"authorization_code", "refresh_token"},
		"scope":         "openid profile email",
	}
	resp := putJSON(t, ts, "/oauth/register/"+clientID, updatePayload, regToken)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck

	if result["client_name"] != "Updated Client Name" {
		t.Errorf("expected client_name='Updated Client Name', got %q", result["client_name"])
	}

	// Verify the agent was actually updated in the store.
	agent, err := store.GetAgentByClientID(context.Background(), clientID)
	if err != nil {
		t.Fatalf("fetching updated agent: %v", err)
	}
	if agent.Name != "Updated Client Name" {
		t.Errorf("agent name not updated in store, got %q", agent.Name)
	}
	if len(agent.RedirectURIs) != 2 {
		t.Errorf("expected 2 redirect URIs, got %d", len(agent.RedirectURIs))
	}

	// client_secret and registration_access_token must NOT be returned on update.
	if _, ok := result["client_secret"]; ok && result["client_secret"] != "" {
		t.Error("PUT response should not return client_secret")
	}
	if _, ok := result["registration_access_token"]; ok && result["registration_access_token"] != "" {
		t.Error("PUT response should not return registration_access_token")
	}
}

// TestDCR_Update_InvalidToken verifies PUT with wrong token returns 401.
func TestDCR_Update_InvalidToken(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Update Auth Test",
		"redirect_uris": []string{"https://example.com/cb"},
		"grant_types":   []string{"authorization_code"},
	}
	regResp := postJSON(t, ts, "/oauth/register", payload, "")
	var regResult map[string]interface{}
	json.NewDecoder(regResp.Body).Decode(&regResult) //nolint:errcheck
	regResp.Body.Close()

	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("registration failed with status %d", regResp.StatusCode)
	}

	clientID, _ := regResult["client_id"].(string)

	updatePayload := map[string]interface{}{
		"client_name":   "Hacked Name",
		"redirect_uris": []string{"https://evil.com/steal"},
		"grant_types":   []string{"authorization_code"},
	}
	resp := putJSON(t, ts, "/oauth/register/"+clientID, updatePayload, "wrongtoken")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
}

// TestDCR_Delete_Success verifies DELETE returns 204 and deactivates the client.
func TestDCR_Delete_Success(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	// Register first.
	payload := map[string]interface{}{
		"client_name":   "Delete Test Client",
		"redirect_uris": []string{"https://example.com/callback"},
		"grant_types":   []string{"authorization_code"},
	}
	regResp := postJSON(t, ts, "/oauth/register", payload, "")
	var regResult map[string]interface{}
	json.NewDecoder(regResp.Body).Decode(&regResult) //nolint:errcheck
	regResp.Body.Close()

	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("registration failed with status %d", regResp.StatusCode)
	}

	clientID := regResult["client_id"].(string)
	regToken := regResult["registration_access_token"].(string)

	// Delete the client.
	resp := deleteWithBearer(t, ts, "/oauth/register/"+clientID, regToken)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, body)
	}

	// Verify the agent is now inactive.
	agent, err := store.GetAgentByClientID(context.Background(), clientID)
	if err != nil {
		t.Fatalf("fetching deactivated agent: %v", err)
	}
	if agent.Active {
		t.Error("agent should be inactive after DELETE")
	}

	// Subsequent GET should fail (registration_access_token check against DCR record).
	resp2 := getWithBearer(t, ts, "/oauth/register/"+clientID, regToken)
	defer resp2.Body.Close()
	// After deletion of DCR record, the token verification should fail.
	if resp2.StatusCode == http.StatusOK {
		t.Error("GET after DELETE should not return 200")
	}
}

// TestDCR_Delete_InvalidToken verifies DELETE with wrong token returns 401.
func TestDCR_Delete_InvalidToken(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Delete Auth Test",
		"redirect_uris": []string{"https://example.com/cb"},
		"grant_types":   []string{"authorization_code"},
	}
	regResp := postJSON(t, ts, "/oauth/register", payload, "")
	var regResult map[string]interface{}
	json.NewDecoder(regResp.Body).Decode(&regResult) //nolint:errcheck
	regResp.Body.Close()

	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("registration failed with status %d", regResp.StatusCode)
	}

	clientID, _ := regResult["client_id"].(string)

	resp := deleteWithBearer(t, ts, "/oauth/register/"+clientID, "notmytoken")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
}

// TestDCR_RegistrationTokenSeparateFromClientSecret verifies that the
// registration_access_token is different from the client_secret.
func TestDCR_RegistrationTokenSeparateFromClientSecret(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Separation Test",
		"redirect_uris": []string{"https://example.com/cb"},
		"grant_types":   []string{"authorization_code"},
	}
	resp := postJSON(t, ts, "/oauth/register", payload, "")
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	resp.Body.Close()

	secret := result["client_secret"].(string)
	regToken := result["registration_access_token"].(string)

	if secret == regToken {
		t.Error("client_secret and registration_access_token must be different credentials")
	}
	if secret == "" {
		t.Error("client_secret must not be empty")
	}
	if regToken == "" {
		t.Error("registration_access_token must not be empty")
	}
}

// TestValidateRedirectURI covers the redirect URI validation logic directly.
func TestValidateRedirectURI(t *testing.T) {
	cases := []struct {
		uri   string
		valid bool
	}{
		{"https://example.com/callback", true},
		{"https://app.example.org/cb?foo=bar", true},
		{"http://localhost/callback", true},
		{"http://localhost:8080/callback", true},
		{"http://127.0.0.1:9000/callback", true},
		{"http://evil.com/steal", false},
		{"http://notlocalhost.com/callback", false},
		{"ftp://example.com/callback", false},
		{"not-a-url", false},
		{"", false},
	}

	for _, tc := range cases {
		err := validateRedirectURI(tc.uri)
		if tc.valid && err != nil {
			t.Errorf("URI %q: expected valid, got error: %v", tc.uri, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("URI %q: expected error, got nil", tc.uri)
		}
	}
}

// TestDCR_RegistrationTokenHash verifies the stored hash matches SHA-256 of the token.
func TestDCR_RegistrationTokenHash(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Hash Test Client",
		"redirect_uris": []string{"https://example.com/cb"},
		"grant_types":   []string{"authorization_code"},
	}
	resp := postJSON(t, ts, "/oauth/register", payload, "")
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	resp.Body.Close()

	clientID := result["client_id"].(string)
	regToken := result["registration_access_token"].(string)

	// Verify the hash stored in the DB matches SHA-256 of the token.
	dcr, err := store.GetDCRClient(context.Background(), clientID)
	if err != nil {
		t.Fatalf("fetching DCR client: %v", err)
	}

	h := sha256.Sum256([]byte(regToken))
	expectedHash := hex.EncodeToString(h[:])
	if dcr.RegistrationTokenHash != expectedHash {
		t.Errorf("stored hash mismatch: expected %q, got %q", expectedHash, dcr.RegistrationTokenHash)
	}
}

// TestDCR_DefaultGrantTypes verifies that omitting grant_types defaults to authorization_code.
func TestDCR_DefaultGrantTypes(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRouter(srv))
	defer ts.Close()

	payload := map[string]interface{}{
		"client_name":   "Defaults Test",
		"redirect_uris": []string{"https://example.com/cb"},
		// grant_types omitted — defaults to ["authorization_code"].
	}
	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck

	grantTypes, _ := result["grant_types"].([]interface{})
	if len(grantTypes) != 1 || grantTypes[0] != "authorization_code" {
		t.Errorf("expected default grant_types=[authorization_code], got %v", grantTypes)
	}

	responseTypes, _ := result["response_types"].([]interface{})
	if len(responseTypes) != 1 || responseTypes[0] != "code" {
		t.Errorf("expected default response_types=[code], got %v", responseTypes)
	}

	if result["token_endpoint_auth_method"] != "client_secret_basic" {
		t.Errorf("expected default token_endpoint_auth_method=client_secret_basic, got %v",
			result["token_endpoint_auth_method"])
	}
}
