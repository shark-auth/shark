// Package api â€” tests for PATCH /api/v1/admin/apps/{id} covering the
// A8 integration_mode + branding_override + proxy_login_fallback fields.
package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shark-auth/shark/internal/testutil"
)

type updateAppPayload struct {
	IntegrationMode       *string         `json:"integration_mode,omitempty"`
	BrandingOverride      json.RawMessage `json:"branding_override,omitempty"`
	ProxyLoginFallback    *string         `json:"proxy_login_fallback,omitempty"`
	ProxyLoginFallbackURL *string         `json:"proxy_login_fallback_url,omitempty"`
}

type updateAppResponse struct {
	ID                    string          `json:"id"`
	IntegrationMode       string          `json:"integration_mode"`
	BrandingOverride      json.RawMessage `json:"branding_override"`
	ProxyLoginFallback    string          `json:"proxy_login_fallback"`
	ProxyLoginFallbackURL string          `json:"proxy_login_fallback_url"`
}

func seededAppID(t *testing.T, ts *testutil.TestServer) string {
	t.Helper()
	app, err := ts.Store.GetDefaultApplication(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultApplication: %v", err)
	}
	return app.ID
}

func TestUpdateApplication_SetIntegrationMode(t *testing.T) {
	ts := testutil.NewTestServer(t)
	id := seededAppID(t, ts)

	mode := "hosted"
	resp := ts.PatchJSONWithAdminKey(
		"/api/v1/admin/apps/"+id,
		updateAppPayload{IntegrationMode: &mode},
	)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH: got %d, want 200", resp.StatusCode)
	}

	// GET confirms persistence.
	getResp := ts.GetWithAdminKey("/api/v1/admin/apps/" + id)
	defer getResp.Body.Close()
	var got updateAppResponse
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if got.IntegrationMode != "hosted" {
		t.Errorf("integration_mode: got %q, want hosted", got.IntegrationMode)
	}
}

func TestUpdateApplication_InvalidIntegrationMode(t *testing.T) {
	ts := testutil.NewTestServer(t)
	id := seededAppID(t, ts)

	mode := "bogus"
	resp := ts.PatchJSONWithAdminKey(
		"/api/v1/admin/apps/"+id,
		updateAppPayload{IntegrationMode: &mode},
	)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("PATCH bogus mode: got %d, want 400", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "invalid_request" {
		t.Errorf("error: got %q, want invalid_request", body["error"])
	}
}

func TestUpdateApplication_SetBrandingOverride(t *testing.T) {
	ts := testutil.NewTestServer(t)
	id := seededAppID(t, ts)

	override := json.RawMessage(`{"primary_color":"#f00"}`)
	resp := ts.PatchJSONWithAdminKey(
		"/api/v1/admin/apps/"+id,
		updateAppPayload{BrandingOverride: override},
	)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH: got %d, want 200", resp.StatusCode)
	}

	// ResolveBranding returns the override primary_color.
	branding, err := ts.Store.ResolveBranding(context.Background(), id)
	if err != nil {
		t.Fatalf("ResolveBranding: %v", err)
	}
	if branding.PrimaryColor != "#f00" {
		t.Errorf("primary_color: got %q, want #f00", branding.PrimaryColor)
	}
}

func TestUpdateApplication_InvalidProxyLoginFallback(t *testing.T) {
	// Chosen behavior: custom_url is accepted; we only enforce the enum at
	// the proxy_login_fallback level. Missing URL is accepted and stored
	// as empty â€” matches "store whatever set" per the task spec.
	ts := testutil.NewTestServer(t)
	id := seededAppID(t, ts)

	fb := "custom_url"
	resp := ts.PatchJSONWithAdminKey(
		"/api/v1/admin/apps/"+id,
		updateAppPayload{ProxyLoginFallback: &fb},
	)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH custom_url (no URL): got %d, want 200", resp.StatusCode)
	}

	// Now prove that an unknown enum value does fail.
	bogus := "nope"
	resp2 := ts.PatchJSONWithAdminKey(
		"/api/v1/admin/apps/"+id,
		updateAppPayload{ProxyLoginFallback: &bogus},
	)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("PATCH bogus fallback: got %d, want 400", resp2.StatusCode)
	}
}
