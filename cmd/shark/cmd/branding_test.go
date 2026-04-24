package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestBrandingSet_TokenPairs verifies that --token key=value pairs are sent
// correctly via the admin API.
func TestBrandingSet_TokenPairs(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && r.URL.Path == "/api/v1/admin/branding/design-tokens" {
			_ = json.NewDecoder(r.Body).Decode(&received)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"branding": map[string]any{}, "design_tokens": map[string]any{}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	t.Setenv("SHARK_URL", srv.URL)
	t.Setenv("SHARK_ADMIN_TOKEN", "test-token")

	if err := brandingSetCmd.Flags().Set("token", "colors.primary=#6366f1"); err != nil {
		t.Fatalf("set token flag: %v", err)
	}
	t.Cleanup(func() {
		// StringArray flags can't be reset via Set on the same key; recreate
		// via a direct flags lookup.
		f := brandingSetCmd.Flags().Lookup("token")
		if f != nil {
			f.Value.Set("") //nolint:errcheck
		}
	})

	stdout := &bytes.Buffer{}
	brandingSetCmd.SetOut(stdout)
	t.Cleanup(func() { brandingSetCmd.SetOut(nil) })

	if err := brandingSetCmd.RunE(brandingSetCmd, []string{"my-app"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if received == nil {
		t.Fatal("server never received a request")
	}
	tokens, ok := received["design_tokens"].(map[string]any)
	if !ok {
		t.Fatalf("design_tokens missing from payload: %+v", received)
	}
	if tokens["colors.primary"] != "#6366f1" {
		t.Errorf("wrong token value: %+v", tokens)
	}
}

// TestBrandingSet_FromFile verifies that --from-file reads a JSON file and
// sends its content as design_tokens.
func TestBrandingSet_FromFile(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "tokens.json")
	if err := os.WriteFile(tokenFile, []byte(`{"colors":{"primary":"#fff"}}`), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer srv.Close()

	t.Setenv("SHARK_URL", srv.URL)
	t.Setenv("SHARK_ADMIN_TOKEN", "test-token")

	if err := brandingSetCmd.Flags().Set("from-file", tokenFile); err != nil {
		t.Fatalf("set from-file: %v", err)
	}
	t.Cleanup(func() { _ = brandingSetCmd.Flags().Set("from-file", "") })

	if err := brandingSetCmd.RunE(brandingSetCmd, []string{"my-app"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	tokens, ok := received["design_tokens"].(map[string]any)
	if !ok {
		t.Fatalf("design_tokens missing: %+v", received)
	}
	colors, ok := tokens["colors"].(map[string]any)
	if !ok || colors["primary"] != "#fff" {
		t.Errorf("unexpected tokens: %+v", tokens)
	}
}

// TestBrandingSet_NoTokens asserts that providing neither --token nor
// --from-file returns an error.
func TestBrandingSet_NoTokens(t *testing.T) {
	t.Setenv("SHARK_ADMIN_TOKEN", "test-token")
	err := brandingSetCmd.RunE(brandingSetCmd, []string{"my-app"})
	if err == nil {
		t.Fatal("expected error when no tokens provided")
	}
}
