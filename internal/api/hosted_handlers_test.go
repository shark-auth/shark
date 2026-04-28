package api_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/api"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// createHostedApp inserts an application with the given slug and integration
// mode into the test store and returns it.
func createHostedApp(t *testing.T, store storage.Store, slug, integrationMode string) *storage.Application {
	t.Helper()
	nid, _ := gonanoid.New()
	appNid, _ := gonanoid.New()
	now := time.Now().UTC()
	app := &storage.Application{
		ID:                  "app_" + appNid,
		Name:                "Test App",
		Slug:                slug,
		ClientID:            "shark_app_" + nid,
		ClientSecretHash:    "deadbeef",
		ClientSecretPrefix:  "deadbeef",
		AllowedCallbackURLs: []string{},
		AllowedLogoutURLs:   []string{},
		AllowedOrigins:      []string{},
		IsDefault:           false,
		Metadata:            map[string]any{},
		IntegrationMode:     integrationMode,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := store.CreateApplication(context.Background(), app); err != nil {
		t.Fatalf("createHostedApp: %v", err)
	}
	return app
}

// TestHandleHostedPage_ValidAppValidPage_Returns200 checks that a hosted-mode
// app + valid page returns 200 with the SPA bootstrap script present.
func TestHandleHostedPage_ValidAppValidPage_Returns200(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "test", "hosted")

	resp := ts.Get("/hosted/test/login")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type: got %q, want text/html*", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	bs := string(body)
	if !strings.Contains(bs, "__SHARK_HOSTED") {
		t.Fatalf("body missing __SHARK_HOSTED: %s", bs[:min(len(bs), 500)])
	}
}

// TestHandleHostedPage_InvalidPage_Returns404 checks that unknown page names
// return 404.
func TestHandleHostedPage_InvalidPage_Returns404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "test", "hosted")

	resp := ts.Get("/hosted/test/bogus")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

// TestHandleHostedPage_UnknownSlug_Returns404 checks that an unregistered app
// slug returns 404.
func TestHandleHostedPage_UnknownSlug_Returns404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	resp := ts.Get("/hosted/nope/login")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

// TestHandleHostedPage_CustomIntegration_Returns404 checks that an app with
// integration_mode="custom" returns 404 with "hosted auth disabled" in the body.
func TestHandleHostedPage_CustomIntegration_Returns404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "myapp", "custom")

	resp := ts.Get("/hosted/myapp/login")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "hosted auth disabled") {
		t.Fatalf("body: got %q, want 'hosted auth disabled'", string(body))
	}
}

// TestHandleHostedPage_BrandingVarsInCSS checks that the app's resolved
// branding primary color appears in the inline <style> block.
func TestHandleHostedPage_BrandingVarsInCSS(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "brand-test", "hosted")

	// Set a custom primary color via the admin branding endpoint (admin key required).
	patchResp := ts.PatchJSONWithAdminKey("/api/v1/admin/branding",
		map[string]string{"primary_color": "#ff5733"})
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("patch branding status: %d", patchResp.StatusCode)
	}

	resp := ts.Get("/hosted/brand-test/login")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "#ff5733") {
		t.Fatalf("body missing branding color #ff5733; got: %s", string(body)[:min(len(string(body)), 2000)])
	}
}

// TestHandleHostedPage_OAuthParamsPassed checks that client_id, redirect_uri,
// and state forwarded on the query string are present in the __SHARK_HOSTED JSON.
func TestHandleHostedPage_OAuthParamsPassed(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "oauth-test", "hosted")

	resp := ts.Get("/hosted/oauth-test/login?client_id=abc&redirect_uri=https%3A%2F%2Fexample.com%2Fcb&state=xyz")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bs := string(body)
	for _, want := range []string{"abc", "https://example.com/cb", "xyz"} {
		if !strings.Contains(bs, want) {
			t.Fatalf("body missing %q in __SHARK_HOSTED", want)
		}
	}
}

// TestHandleHostedPage_ProxyModeAllowed verifies that proxy integration mode
// also serves the hosted shell (proxy-mode apps redirect unauthed users here).
func TestHandleHostedPage_ProxyModeAllowed(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "proxy-app", "proxy")

	resp := ts.Get("/hosted/proxy-app/login")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200 for proxy mode; got %d", resp.StatusCode, resp.StatusCode)
	}
}

// TestHandleHostedPage_NoCacheHeader verifies the no-store cache header is set.
func TestHandleHostedPage_NoCacheHeader(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "cache-test", "hosted")

	resp := ts.Get("/hosted/cache-test/login")
	defer resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if cc != "no-store" {
		t.Fatalf("Cache-Control: got %q, want no-store", cc)
	}
}

// TestFindHostedBundle verifies that the bundle resolver returns the real
// hosted-CzpeAm29.js filename from the embedded FS.
func TestFindHostedBundle(t *testing.T) {
	// findHostedBundle is unexported; test through the package-level init
	// side-effect by checking that NewTestServer builds and the handler
	// includes the bundle src in its HTML.
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "bundle-test", "hosted")

	resp := ts.Get("/hosted/bundle-test/login")
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bs := string(body)
	// The embedded bundle is hosted-CzpeAm29.js — check the path is present.
	if !strings.Contains(bs, "/admin/hosted/assets/hosted-") {
		t.Fatalf("body missing bundle script tag; got: %s", bs[:min(len(bs), 1000)])
	}
}

// TestHandleHostedAssets_ServesBundle checks that the embedded .js bundle is
// served from /admin/hosted/assets/ with immutable cache headers.
func TestHandleHostedAssets_ServesBundle(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	resp := ts.Get("/admin/hosted/assets/hosted-BDnwMcaL.js")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	cc := resp.Header.Get("Cache-Control")
	if !strings.Contains(cc, "max-age=31536000") {
		t.Fatalf("Cache-Control: got %q, want immutable", cc)
	}
}

// TestHandleHostedPage_AllPages verifies each valid page name returns 200.
func TestHandleHostedPage_AllPages(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "pages-test", "hosted")

	pages := []string{"login", "signup", "magic", "passkey", "mfa", "verify", "error"}
	for _, page := range pages {
		resp := ts.Get("/hosted/pages-test/" + page)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("page %q: got %d, want 200", page, resp.StatusCode)
		}
	}
}

// TestHandleHostedPage_BrandingVarsInCSS_WithAdminKey is a variant that
// confirms the branding PATCH endpoint requires an admin key (test infra).
// This verifies the ts.PatchJSON helper is using the admin key properly.
func TestHandleHostedPage_BrandingPatchRequiresAdminKey(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	// PATCH without the admin key should be rejected.
	resp, err := ts.Client.Do(mustReq(t, "PATCH", ts.URL("/api/v1/admin/branding"), strings.NewReader(`{"primary_color":"#123456"}`)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("expected rejection without admin key")
	}
}

// mustReq builds an http.Request or fatals the test.
func mustReq(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

// TestHandlePaywallPage_Renders402WithCSSVars checks that the paywall page
// returns 402 Payment Required and inlines the branding CSS custom
// properties.
func TestHandlePaywallPage_Renders402WithCSSVars(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "paywall-test", "hosted")

	// Set a custom primary color so we can assert the CSS var flowed through.
	patchResp := ts.PatchJSONWithAdminKey("/api/v1/admin/branding",
		map[string]string{"primary_color": "#ff00ee"})
	patchResp.Body.Close()

	resp := ts.Get("/paywall/paywall-test?tier=pro&return=%2Fapp%2Fdashboard")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("status: got %d, want 402", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type: got %q, want text/html*", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	bs := string(body)
	if !strings.Contains(bs, "--shark-primary: #ff00ee") {
		t.Fatalf("body missing --shark-primary css var; body=%s", bs[:min(len(bs), 1500)])
	}
	if !strings.Contains(bs, "Upgrade to pro") {
		t.Fatalf("body missing headline; body=%s", bs[:min(len(bs), 1500)])
	}
	if !strings.Contains(bs, "/app/dashboard?upgrade=pro") {
		t.Fatalf("body missing upgrade href; body=%s", bs[:min(len(bs), 1500)])
	}
}

// TestHandlePaywallPage_MissingTierReturns400 verifies the required
// ?tier= param produces a 400.
func TestHandlePaywallPage_MissingTierReturns400(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "no-tier", "hosted")

	resp := ts.Get("/paywall/no-tier")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

// TestHandlePaywallPage_UnknownSlug_Returns404 verifies an unregistered
// app slug falls through to 404.
func TestHandlePaywallPage_UnknownSlug_Returns404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	resp := ts.Get("/paywall/ghost?tier=pro")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

// TestHandlePaywallPage_MaliciousReturn_Sanitized verifies that a
// javascript: scheme in ?return= is replaced with "/" before embedding.
func TestHandlePaywallPage_MaliciousReturn_Sanitized(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "xss-test", "hosted")

	// URL-encoded javascript:alert(1) as the return value.
	resp := ts.Get("/paywall/xss-test?tier=pro&return=javascript%3Aalert%281%29")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("status: got %d, want 402", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bs := string(body)
	if strings.Contains(bs, "javascript:") {
		t.Fatalf("body must not contain javascript:, got: %s", bs[:min(len(bs), 1500)])
	}
	if !strings.Contains(bs, "/?upgrade=pro") {
		t.Fatalf("expected fallback upgrade href /?upgrade=pro, got body: %s", bs[:min(len(bs), 1500)])
	}
}

// TestHandlePaywallPage_MaliciousTier_Sanitized verifies that a tier
// value containing HTML-breaking chars is replaced with the fallback
// label.
func TestHandlePaywallPage_MaliciousTier_Sanitized(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	createHostedApp(t, ts.Store, "xss-tier", "hosted")

	resp := ts.Get("/paywall/xss-tier?tier=%3Cscript%3Ealert%281%29%3C%2Fscript%3E")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("status: got %d, want 402", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bs := string(body)
	if strings.Contains(bs, "<script>") {
		t.Fatalf("body must not contain <script>, got: %s", bs[:min(len(bs), 1500)])
	}
	if !strings.Contains(bs, "Upgrade to upgrade") {
		t.Fatalf("expected fallback tier label 'upgrade', got: %s", bs[:min(len(bs), 1500)])
	}
}

// Ensure api package is used (import cycle check).
var _ = api.Server{}
