package api_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

// writeBrandingAsset creates a file at data/assets/branding/<name> relative
// to the test's cwd (which chdirTemp switches into). Returns the raw bytes
// written so callers can assert on the served body.
func writeBrandingAsset(t *testing.T, name string, body []byte) {
	t.Helper()
	dir := filepath.Join("data", "assets", "branding")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir branding dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil { //#nosec G306 -- test fixture
		t.Fatalf("write asset: %v", err)
	}
}

func TestHandleBrandingAsset_ServesFile(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	want := []byte("fake-png-bytes")
	writeBrandingAsset(t, "abc123.png", want)

	resp := ts.Get("/assets/branding/abc123.png")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("body mismatch: got %q, want %q", got, want)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type: got %q, want image/png", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("cache-control: got %q", cc)
	}
}

func TestHandleBrandingAsset_PathTraversal(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	resp := ts.Get("/assets/branding/..%2Fsecret")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestHandleBrandingAsset_NestedPath(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	resp := ts.Get("/assets/branding/foo/bar.png")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestHandleBrandingAsset_Missing(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	resp := ts.Get("/assets/branding/nonexistent-sha.png")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestHandleBrandingAsset_PNGContentType(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	writeBrandingAsset(t, "logo.png", []byte("x"))
	resp := ts.Get("/assets/branding/logo.png")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type: got %q, want image/png", ct)
	}
}

func TestHandleBrandingAsset_SVGContentType(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	writeBrandingAsset(t, "logo.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	resp := ts.Get("/assets/branding/logo.svg")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("content-type: got %q, want image/svg+xml", ct)
	}
}

func TestHandleBrandingAsset_CacheControl(t *testing.T) {
	chdirTemp(t)
	ts := testutil.NewTestServer(t)
	defer ts.Server.Close()

	writeBrandingAsset(t, "cached.png", []byte("y"))
	resp := ts.Get("/assets/branding/cached.png")
	defer resp.Body.Close()
	if cc := resp.Header.Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("cache-control: got %q, want public, max-age=31536000, immutable", cc)
	}
}
