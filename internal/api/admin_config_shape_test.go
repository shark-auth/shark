package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

// TestAdminConfigShape asserts that GET /admin/config returns all the nested
// fields the dashboard reads. Any field missing here means operators will see
// fabricated values in the UI.
func TestAdminConfigShape(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.GetWithAdminKey("/api/v1/admin/config")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/config: status=%d", resp.StatusCode)
	}

	// Decode into a generic map so we can assert on exact key presence without
	// being coupled to the Go struct shape.
	var body map[string]json.RawMessage
	ts.DecodeJSON(resp, &body)

	// --- top-level scalar fields ---
	topLevel := []string{
		"dev_mode",
		"smtp_configured",
		"session_mode",
		"session_lifetime",
		"jwt_mode",
		"base_url",
		"social_providers",
	}
	for _, k := range topLevel {
		if _, ok := body[k]; !ok {
			t.Errorf("missing top-level field %q", k)
		}
	}

	// social_providers must be an array (not null).
	if raw, ok := body["social_providers"]; ok {
		var arr []interface{}
		if err := json.Unmarshal(raw, &arr); err != nil {
			t.Errorf("social_providers is not a JSON array: %v", err)
		}
	}

	// dev_mode must be a bool.
	if raw, ok := body["dev_mode"]; ok {
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			t.Errorf("dev_mode is not a bool: %v", err)
		}
	}

	// smtp_configured must be a bool.
	if raw, ok := body["smtp_configured"]; ok {
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			t.Errorf("smtp_configured is not a bool: %v", err)
		}
	}

	// --- passkey nested object ---
	passkeyRaw, ok := body["passkey"]
	if !ok {
		t.Fatal("missing top-level field \"passkey\"")
	}
	var passkey map[string]json.RawMessage
	if err := json.Unmarshal(passkeyRaw, &passkey); err != nil {
		t.Fatalf("passkey is not an object: %v", err)
	}
	passkeyFields := []string{"enabled", "rp_id", "rp_name", "origin", "user_verification", "attestation"}
	for _, k := range passkeyFields {
		if _, ok := passkey[k]; !ok {
			t.Errorf("missing passkey.%s", k)
		}
	}
	// passkey.enabled must be a bool.
	if raw, ok := passkey["enabled"]; ok {
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			t.Errorf("passkey.enabled is not a bool: %v", err)
		}
	}
	// passkey.rp_id must be a string and match what we set in TestConfig.
	if raw, ok := passkey["rp_id"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("passkey.rp_id is not a string: %v", err)
		}
		if s != "localhost" {
			t.Errorf("passkey.rp_id = %q, want %q", s, "localhost")
		}
	}

	// --- password_policy nested object ---
	ppRaw, ok := body["password_policy"]
	if !ok {
		t.Fatal("missing top-level field \"password_policy\"")
	}
	var pp map[string]json.RawMessage
	if err := json.Unmarshal(ppRaw, &pp); err != nil {
		t.Fatalf("password_policy is not an object: %v", err)
	}
	ppFields := []string{"min_length", "require_upper", "require_lower", "require_digit", "require_symbol"}
	for _, k := range ppFields {
		if _, ok := pp[k]; !ok {
			t.Errorf("missing password_policy.%s", k)
		}
	}
	// password_policy.min_length must be numeric ≥ 1.
	if raw, ok := pp["min_length"]; ok {
		var n float64
		if err := json.Unmarshal(raw, &n); err != nil {
			t.Errorf("password_policy.min_length is not a number: %v", err)
		}
		if n < 1 {
			t.Errorf("password_policy.min_length = %v, want ≥1", n)
		}
	}

	// --- jwt nested object ---
	jwtRaw, ok := body["jwt"]
	if !ok {
		t.Fatal("missing top-level field \"jwt\"")
	}
	var jwt map[string]json.RawMessage
	if err := json.Unmarshal(jwtRaw, &jwt); err != nil {
		t.Fatalf("jwt is not an object: %v", err)
	}
	jwtFields := []string{"algorithm", "lifetime", "active_keys"}
	for _, k := range jwtFields {
		if _, ok := jwt[k]; !ok {
			t.Errorf("missing jwt.%s", k)
		}
	}
	// jwt.algorithm must be a non-empty string.
	if raw, ok := jwt["algorithm"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("jwt.algorithm is not a string: %v", err)
		}
		if s == "" {
			t.Errorf("jwt.algorithm is empty")
		}
	}

	// --- magic_link nested object ---
	mlRaw, ok := body["magic_link"]
	if !ok {
		t.Fatal("missing top-level field \"magic_link\"")
	}
	var ml map[string]json.RawMessage
	if err := json.Unmarshal(mlRaw, &ml); err != nil {
		t.Fatalf("magic_link is not an object: %v", err)
	}
	if _, ok := ml["ttl"]; !ok {
		t.Error("missing magic_link.ttl")
	}
	if raw, ok := ml["ttl"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("magic_link.ttl is not a string: %v", err)
		}
		if s == "" {
			t.Errorf("magic_link.ttl is empty")
		}
	}

	// --- session_mode must be "cookie" or "jwt" ---
	if raw, ok := body["session_mode"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("session_mode is not a string: %v", err)
		}
		if s != "cookie" && s != "jwt" {
			t.Errorf("session_mode = %q, want \"cookie\" or \"jwt\"", s)
		}
	}
}
