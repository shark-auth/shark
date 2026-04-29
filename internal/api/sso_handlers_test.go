package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

func TestSSOHandlers_CreateAndListConnections(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create connection (admin endpoint)
	body := map[string]interface{}{
		"type":               "oidc",
		"name":               "Test OIDC Provider",
		"domain":             "example.com",
		"oidc_issuer":        "https://idp.example.com",
		"oidc_client_id":     "client-123",
		"oidc_client_secret": "secret-456",
	}
	resp := ts.PostJSONWithAdminKey("/api/v1/sso/connections", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}

	var created storage.SSOConnection
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created connection should have an ID")
	}
	if created.Name != "Test OIDC Provider" {
		t.Fatalf("expected name %q, got %q", "Test OIDC Provider", created.Name)
	}

	// List connections (admin endpoint)
	resp2 := ts.GetWithAdminKey("/api/v1/sso/connections")
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", resp2.StatusCode)
	}

	var connections []*storage.SSOConnection
	if err := json.NewDecoder(resp2.Body).Decode(&connections); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}
}

func TestSSOHandlers_GetUpdateDeleteConnection(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create
	body := map[string]interface{}{
		"type":           "oidc",
		"name":           "Get-Update-Delete Test",
		"oidc_issuer":    "https://idp.test.com",
		"oidc_client_id": "client",
	}
	resp := ts.PostJSONWithAdminKey("/api/v1/sso/connections", body)
	defer resp.Body.Close()

	var created storage.SSOConnection
	json.NewDecoder(resp.Body).Decode(&created)

	// Get
	resp2 := ts.GetWithAdminKey("/api/v1/sso/connections/" + created.ID)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", resp2.StatusCode)
	}

	// Update
	updateBody := map[string]interface{}{
		"type":           "oidc",
		"name":           "Updated Name",
		"oidc_issuer":    "https://idp.test.com",
		"oidc_client_id": "client",
	}
	resp3 := ts.PutJSONWithAdminKey("/api/v1/sso/connections/"+created.ID, updateBody)
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("update: expected 200, got %d", resp3.StatusCode)
	}

	// Delete
	resp4 := ts.DeleteWithAdminKey("/api/v1/sso/connections/" + created.ID)
	defer resp4.Body.Close()

	if resp4.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", resp4.StatusCode)
	}

	// Get should now 404
	resp5 := ts.GetWithAdminKey("/api/v1/sso/connections/" + created.ID)
	defer resp5.Body.Close()

	if resp5.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", resp5.StatusCode)
	}
}

func TestSSOHandlers_AutoRoute(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create OIDC connection with domain (admin endpoint)
	body := map[string]interface{}{
		"type":           "oidc",
		"name":           "Corp SSO",
		"domain":         "bigcorp.com",
		"oidc_issuer":    "https://idp.bigcorp.com",
		"oidc_client_id": "client",
	}
	resp := ts.PostJSONWithAdminKey("/api/v1/sso/connections", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}

	// Auto-route by email (public endpoint)
	resp2 := ts.Get("/api/v1/auth/sso?email=employee@bigcorp.com")
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("auto-route: expected 200, got %d", resp2.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&result)

	if result["connection_type"] != "oidc" {
		t.Fatalf("expected connection_type oidc, got %v", result["connection_type"])
	}
	if result["connection_name"] != "Corp SSO" {
		t.Fatalf("expected connection_name %q, got %v", "Corp SSO", result["connection_name"])
	}

	// Auto-route with unknown domain should 404
	resp3 := ts.Get("/api/v1/auth/sso?email=user@unknown.com")
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusNotFound {
		t.Fatalf("auto-route unknown: expected 404, got %d", resp3.StatusCode)
	}

	// Auto-route without email should 400
	resp4 := ts.Get("/api/v1/auth/sso")
	defer resp4.Body.Close()

	if resp4.StatusCode != http.StatusBadRequest {
		t.Fatalf("auto-route no email: expected 400, got %d", resp4.StatusCode)
	}
}

func TestSSOHandlers_SAMLMetadata(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create SAML connection (admin endpoint)
	body := map[string]interface{}{
		"type":         "saml",
		"name":         "Okta Production",
		"domain":       "okta-corp.com",
		"saml_idp_url": "https://okta.example.com/sso",
	}
	resp := ts.PostJSONWithAdminKey("/api/v1/sso/connections", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create saml: expected 201, got %d", resp.StatusCode)
	}

	var created storage.SSOConnection
	json.NewDecoder(resp.Body).Decode(&created)

	// Get SAML metadata (public endpoint)
	resp2 := ts.Get("/api/v1/sso/saml/" + created.ID + "/metadata")
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("metadata: expected 200, got %d", resp2.StatusCode)
	}

	contentType := resp2.Header.Get("Content-Type")
	if contentType != "application/xml" {
		t.Fatalf("expected Content-Type application/xml, got %q", contentType)
	}
}

func TestSSOHandlers_CreateInvalidType(t *testing.T) {
	ts := testutil.NewTestServer(t)

	body := map[string]interface{}{
		"type": "invalid",
		"name": "Bad Connection",
	}
	resp := ts.PostJSONWithAdminKey("/api/v1/sso/connections", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid type: expected 400, got %d", resp.StatusCode)
	}
}
