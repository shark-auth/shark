package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

const testIssuer = "https://auth.example.com"

func TestMetadataEndpoint(t *testing.T) {
	handler := MetadataHandler(testIssuer)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	// Status 200
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	// Content-Type
	ct := res.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	// Cache-Control
	cc := res.Header.Get("Cache-Control")
	if cc != "public, max-age=3600" {
		t.Errorf("expected Cache-Control 'public, max-age=3600', got %q", cc)
	}

	// Decode body
	var meta map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	// Helper: read a string field
	getString := func(field string) string {
		t.Helper()
		raw, ok := meta[field]
		if !ok {
			t.Errorf("missing field %q", field)
			return ""
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("field %q is not a string: %v", field, err)
		}
		return s
	}

	// Helper: read a []string field
	getStrings := func(field string) []string {
		t.Helper()
		raw, ok := meta[field]
		if !ok {
			t.Errorf("missing field %q", field)
			return nil
		}
		var ss []string
		if err := json.Unmarshal(raw, &ss); err != nil {
			t.Errorf("field %q is not a []string: %v", field, err)
		}
		return ss
	}

	// --- RFC 8414 required fields ---
	if v := getString("issuer"); v != testIssuer {
		t.Errorf("issuer: want %q, got %q", testIssuer, v)
	}
	if v := getString("authorization_endpoint"); v != testIssuer+"/oauth/authorize" {
		t.Errorf("authorization_endpoint: got %q", v)
	}
	if v := getString("token_endpoint"); v != testIssuer+"/oauth/token" {
		t.Errorf("token_endpoint: got %q", v)
	}
	if v := getString("jwks_uri"); v != testIssuer+"/.well-known/jwks.json" {
		t.Errorf("jwks_uri: got %q", v)
	}

	// --- MCP required ---
	if v := getString("registration_endpoint"); v != testIssuer+"/oauth/register" {
		t.Errorf("registration_endpoint: got %q", v)
	}

	// --- OAuth 2.1 endpoints ---
	if v := getString("revocation_endpoint"); v != testIssuer+"/oauth/revoke" {
		t.Errorf("revocation_endpoint: got %q", v)
	}
	if v := getString("introspection_endpoint"); v != testIssuer+"/oauth/introspect" {
		t.Errorf("introspection_endpoint: got %q", v)
	}
	// device_authorization_endpoint intentionally absent — device flow hidden in v0.1.
	if _, ok := meta["device_authorization_endpoint"]; ok {
		t.Errorf("device_authorization_endpoint must not be advertised in v0.1 (device flow hidden)")
	}
	if v := getString("service_documentation"); v != "https://sharkauth.com/docs" {
		t.Errorf("service_documentation: got %q", v)
	}

	// --- code_challenge_methods_supported must include S256 ---
	ccm := getStrings("code_challenge_methods_supported")
	if !slices.Contains(ccm, "S256") {
		t.Errorf("code_challenge_methods_supported does not contain S256; got %v", ccm)
	}

	// --- grant_types_supported must include the v0.1 grant types ---
	// device_code (RFC 8628) intentionally excluded — device flow hidden in v0.1.
	wantGrants := []string{
		"authorization_code",
		"client_credentials",
		"refresh_token",
		"urn:ietf:params:oauth:grant-type:token-exchange",
	}
	grants := getStrings("grant_types_supported")
	for _, g := range wantGrants {
		if !slices.Contains(grants, g) {
			t.Errorf("grant_types_supported missing %q; got %v", g, grants)
		}
	}
	if slices.Contains(grants, "urn:ietf:params:oauth:grant-type:device_code") {
		t.Errorf("grant_types_supported must not advertise device_code in v0.1 (device flow hidden); got %v", grants)
	}

	// --- response_types_supported should only contain "code" (no implicit) ---
	rt := getStrings("response_types_supported")
	if !slices.Contains(rt, "code") {
		t.Errorf("response_types_supported missing 'code'; got %v", rt)
	}
	if slices.Contains(rt, "token") {
		t.Errorf("response_types_supported must not contain 'token' (implicit flow forbidden by OAuth 2.1)")
	}

	// --- token_endpoint_auth_methods_supported ---
	wantAuthMethods := []string{
		"client_secret_basic",
		"client_secret_post",
		"private_key_jwt",
		"none",
	}
	authMethods := getStrings("token_endpoint_auth_methods_supported")
	for _, m := range wantAuthMethods {
		if !slices.Contains(authMethods, m) {
			t.Errorf("token_endpoint_auth_methods_supported missing %q; got %v", m, authMethods)
		}
	}

	// --- scopes_supported ---
	scopes := getStrings("scopes_supported")
	for _, s := range []string{"openid", "profile", "email"} {
		if !slices.Contains(scopes, s) {
			t.Errorf("scopes_supported missing %q; got %v", s, scopes)
		}
	}

	// --- dpop_signing_alg_values_supported ---
	dpop := getStrings("dpop_signing_alg_values_supported")
	for _, alg := range []string{"ES256", "RS256"} {
		if !slices.Contains(dpop, alg) {
			t.Errorf("dpop_signing_alg_values_supported missing %q; got %v", alg, dpop)
		}
	}
}

// TestMetadataHandlerReuse verifies the handler can be called multiple times
// and returns consistent output (confirming the payload is pre-marshaled once).
func TestMetadataHandlerReuse(t *testing.T) {
	handler := MetadataHandler(testIssuer)

	var bodies [2][]byte
	for i := range bodies {
		req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		bodies[i] = rec.Body.Bytes()
	}

	if string(bodies[0]) != string(bodies[1]) {
		t.Error("handler returned different bodies across calls; payload must be pre-marshaled")
	}
}
