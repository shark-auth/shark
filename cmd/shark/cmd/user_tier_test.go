package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestUserTier_InvalidTier verifies that an unrecognised tier value returns an
// error before making any HTTP request.
func TestUserTier_InvalidTier(t *testing.T) {
	t.Setenv("SHARK_ADMIN_TOKEN", "test-token")
	err := userTierCmd.RunE(userTierCmd, []string{"user123", "enterprise"})
	if err == nil {
		t.Fatal("expected error for invalid tier 'enterprise'")
	}
}

// TestUserTier_ByID verifies that an opaque ID skips email resolution and
// patches the tier directly.
func TestUserTier_ByID(t *testing.T) {
	var patchedUserID string
	var patchedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			// /api/v1/admin/users/{id}/tier
			patchedUserID = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&patchedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"user": map[string]any{"id": "user_abc"}, "tier": "pro"},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	t.Setenv("SHARK_URL", srv.URL)
	t.Setenv("SHARK_ADMIN_TOKEN", "test-token")

	stdout := &bytes.Buffer{}
	userTierCmd.SetOut(stdout)
	t.Cleanup(func() { userTierCmd.SetOut(nil) })

	if err := userTierCmd.RunE(userTierCmd, []string{"user_abc", "pro"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if patchedUserID != "/api/v1/admin/users/user_abc/tier" {
		t.Errorf("unexpected patch path: %s", patchedUserID)
	}
	if patchedBody["tier"] != "pro" {
		t.Errorf("unexpected body: %+v", patchedBody)
	}
}

// TestUserTier_JSONOutput verifies the --json flag emits the API response.
func TestUserTier_JSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"user": map[string]any{"id": "u1"}, "tier": "free"},
		})
	}))
	defer srv.Close()

	t.Setenv("SHARK_URL", srv.URL)
	t.Setenv("SHARK_ADMIN_TOKEN", "test-token")

	if err := userTierCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set json: %v", err)
	}
	t.Cleanup(func() { _ = userTierCmd.Flags().Set("json", "false") })

	stdout := &bytes.Buffer{}
	userTierCmd.SetOut(stdout)
	t.Cleanup(func() { userTierCmd.SetOut(nil) })

	if err := userTierCmd.RunE(userTierCmd, []string{"u1", "free"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	if _, ok := parsed["data"]; !ok {
		t.Errorf("missing 'data' key: %+v", parsed)
	}
}

// TestResolveUserID_OpqueID verifies that a non-email ref is returned as-is.
func TestResolveUserID_OpaqueID(t *testing.T) {
	// No HTTP server needed — resolveUserID should short-circuit on no '@'.
	id, err := resolveUserID(userTierCmd, "user_abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "user_abc123" {
		t.Errorf("expected 'user_abc123', got %q", id)
	}
}
