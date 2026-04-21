package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

// brandingResponse mirrors the GET /admin/branding envelope.
type brandingResponse struct {
	Branding struct {
		LogoURL          string `json:"logo_url"`
		LogoSHA          string `json:"logo_sha"`
		PrimaryColor     string `json:"primary_color"`
		SecondaryColor   string `json:"secondary_color"`
		FontFamily       string `json:"font_family"`
		FooterText       string `json:"footer_text"`
		EmailFromName    string `json:"email_from_name"`
		EmailFromAddress string `json:"email_from_address"`
	} `json:"branding"`
	Fonts []string `json:"fonts"`
}

// chdirData temporarily switches into a fresh temp dir so the handler's
// relative data/assets/branding writes land in an isolated location and
// clean themselves up after the test.
func chdirTemp(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prev)
	})
}

// fetchBranding GETs /api/v1/admin/branding with the admin key and decodes.
func fetchBranding(t *testing.T, ts *testutil.TestServer) brandingResponse {
	t.Helper()
	resp := ts.GetWithAdminKey("/api/v1/admin/branding")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET branding: expected 200, got %d", resp.StatusCode)
	}
	var out brandingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode branding: %v", err)
	}
	return out
}

func TestGetBranding_Defaults(t *testing.T) {
	ts := testutil.NewTestServer(t)
	out := fetchBranding(t, ts)

	if out.Branding.PrimaryColor != "#7c3aed" {
		t.Errorf("primary_color: want #7c3aed, got %q", out.Branding.PrimaryColor)
	}
	if out.Branding.SecondaryColor != "#1a1a1a" {
		t.Errorf("secondary_color: want #1a1a1a, got %q", out.Branding.SecondaryColor)
	}
	if out.Branding.FontFamily != "manrope" {
		t.Errorf("font_family: want manrope, got %q", out.Branding.FontFamily)
	}
	if out.Branding.EmailFromName != "SharkAuth" {
		t.Errorf("email_from_name: want SharkAuth, got %q", out.Branding.EmailFromName)
	}
	wantFonts := map[string]bool{"manrope": true, "inter": true, "ibm_plex": true}
	if len(out.Fonts) != len(wantFonts) {
		t.Fatalf("fonts: want %d entries, got %d (%v)", len(wantFonts), len(out.Fonts), out.Fonts)
	}
	for _, f := range out.Fonts {
		if !wantFonts[f] {
			t.Errorf("unexpected font slug: %q", f)
		}
	}
}

func TestPatchBranding_PrimaryColor(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PatchJSONWithAdminKey("/api/v1/admin/branding", map[string]any{
		"primary_color": "#ff0000",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH: expected 200, got %d", resp.StatusCode)
	}
	var patched brandingResponse
	if err := json.NewDecoder(resp.Body).Decode(&patched); err != nil {
		t.Fatalf("decode patch response: %v", err)
	}
	if patched.Branding.PrimaryColor != "#ff0000" {
		t.Errorf("patch response primary_color: want #ff0000, got %q", patched.Branding.PrimaryColor)
	}

	// GET again to confirm persistence (not just echo).
	out := fetchBranding(t, ts)
	if out.Branding.PrimaryColor != "#ff0000" {
		t.Errorf("re-GET primary_color: want #ff0000, got %q", out.Branding.PrimaryColor)
	}
}

// uploadLogo builds a multipart form, POSTs with admin key, and returns the
// response. filename + contents are passed straight through so tests can
// exercise extension + SVG-script-scan paths.
func uploadLogo(t *testing.T, ts *testutil.TestServer, filename string, contents []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("logo", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL("/api/v1/admin/branding/logo"), &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+ts.AdminKey)

	resp, err := ts.Client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return resp
}

// tinyPNG is a 1x1 transparent PNG, enough to round-trip through the
// extension check + hash + disk write without embedding real image data.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func TestUploadLogo_HappyPath_PNG(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)

	resp := uploadLogo(t, ts, "logo.png", tinyPNG)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload: expected 200, got %d: %s", resp.StatusCode, body)
	}

	var out struct {
		LogoURL string `json:"logo_url"`
		LogoSHA string `json:"logo_sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(out.LogoURL, "/assets/branding/") {
		t.Errorf("logo_url: want /assets/branding/ prefix, got %q", out.LogoURL)
	}
	if !strings.HasSuffix(out.LogoURL, ".png") {
		t.Errorf("logo_url: want .png suffix, got %q", out.LogoURL)
	}
	if len(out.LogoSHA) != 64 {
		t.Errorf("logo_sha: want 64-hex, got %d chars (%q)", len(out.LogoSHA), out.LogoSHA)
	}

	// File was written to data/assets/branding/{sha}.png (under the temp cwd).
	wantPath := filepath.Join("data", "assets", "branding", out.LogoSHA+".png")
	if _, err := os.Stat(wantPath); err != nil {
		t.Errorf("expected file at %s: %v", wantPath, err)
	}

	// DB row was updated too.
	b, err := ts.Store.GetBranding(context.Background(), "global")
	if err != nil {
		t.Fatalf("GetBranding: %v", err)
	}
	if b.LogoURL != out.LogoURL || b.LogoSHA != out.LogoSHA {
		t.Errorf("DB: want (%q,%q), got (%q,%q)", out.LogoURL, out.LogoSHA, b.LogoURL, b.LogoSHA)
	}
}

func TestUploadLogo_RejectsSVGWithScript(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	resp := uploadLogo(t, ts, "evil.svg", svg)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errBody.Error != "invalid_svg" {
		t.Errorf("want error=invalid_svg, got %q", errBody.Error)
	}
}

func TestUploadLogo_TooLarge(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)

	big := bytes.Repeat([]byte{0x41}, 2<<20) // 2 MiB
	resp := uploadLogo(t, ts, "big.png", big)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errBody.Error != "file_too_large" {
		t.Errorf("want error=file_too_large, got %q", errBody.Error)
	}
}

func TestDeleteLogo_Clears(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	// Seed a logo pointer directly via the store so we don't depend on
	// upload-side plumbing (covered by its own test).
	if err := ts.Store.SetBrandingLogo(ctx, "global", "/assets/branding/deadbeef.png", "deadbeef"); err != nil {
		t.Fatalf("seed SetBrandingLogo: %v", err)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL("/api/v1/admin/branding/logo"), nil)
	req.Header.Set("Authorization", "Bearer "+ts.AdminKey)
	resp, err := ts.Client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	b, err := ts.Store.GetBranding(ctx, "global")
	if err != nil {
		t.Fatalf("GetBranding: %v", err)
	}
	if b.LogoURL != "" || b.LogoSHA != "" {
		t.Errorf("expected cleared logo pointer, got url=%q sha=%q", b.LogoURL, b.LogoSHA)
	}
}
