package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

// TestCORSRelaxedPersisted verifies that PATCHing server.cors_relaxed=true
// is reflected in the subsequent GET /admin/config response.
func TestCORSRelaxedPersisted(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Enable relaxed CORS.
	resp := ts.PatchJSONWithAdminKey("/api/v1/admin/config", map[string]any{
		"server": map[string]any{"cors_relaxed": true},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH cors_relaxed=true: %d", resp.StatusCode)
	}

	// Verify it was stored.
	resp = ts.GetWithAdminKey("/api/v1/admin/config")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/config: %d", resp.StatusCode)
	}
	var cfg struct {
		Server struct {
			CORSRelaxed bool `json:"cors_relaxed"`
		} `json:"server"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if !cfg.Server.CORSRelaxed {
		t.Errorf("expected cors_relaxed=true in config response after PATCH true")
	}

	// Disable it.
	resp = ts.PatchJSONWithAdminKey("/api/v1/admin/config", map[string]any{
		"server": map[string]any{"cors_relaxed": false},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH cors_relaxed=false: %d", resp.StatusCode)
	}

	resp = ts.GetWithAdminKey("/api/v1/admin/config")
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Server.CORSRelaxed {
		t.Errorf("expected cors_relaxed=false in config response after PATCH false")
	}
}
