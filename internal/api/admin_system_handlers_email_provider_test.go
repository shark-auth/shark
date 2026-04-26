package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

// TestEmailProviderDevToggle verifies:
//   - Switching to provider=dev auto-saves previous_provider.
//   - Switching back to the previous_provider restores it.
func TestEmailProviderDevToggle(t *testing.T) {
	ts := testutil.NewTestServer(t)

	patchProvider := func(t *testing.T, provider string) {
		t.Helper()
		resp := ts.PatchJSONWithAdminKey("/api/v1/admin/config", map[string]any{
			"email": map[string]any{"provider": provider},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("PATCH email.provider=%s: %d", provider, resp.StatusCode)
		}
	}

	getEmail := func(t *testing.T) (provider, previous string) {
		t.Helper()
		resp := ts.GetWithAdminKey("/api/v1/admin/config")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /admin/config: %d", resp.StatusCode)
		}
		var cfg struct {
			Email struct {
				Provider         string `json:"provider"`
				PreviousProvider string `json:"previous_provider"`
			} `json:"email"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return cfg.Email.Provider, cfg.Email.PreviousProvider
	}

	// Seed a production provider.
	patchProvider(t, "resend")
	p, _ := getEmail(t)
	if p != "resend" {
		t.Fatalf("expected provider=resend after seed, got %q", p)
	}

	// Switch to dev — should auto-capture previous_provider=resend.
	patchProvider(t, "dev")
	p, prev := getEmail(t)
	if p != "dev" {
		t.Errorf("expected provider=dev, got %q", p)
	}
	if prev != "resend" {
		t.Errorf("expected previous_provider=resend after switching to dev, got %q", prev)
	}

	// Restore using the captured previous_provider value.
	patchProvider(t, prev)
	p, _ = getEmail(t)
	if p != "resend" {
		t.Errorf("expected provider=resend after restoring from dev, got %q", p)
	}
}
