package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// handleBrandingAsset serves content-addressed branding assets (logos, etc.)
// from data/assets/branding/{sha}.{ext}. Public — no auth required — because
// these URLs are embedded into outbound emails and external sites. Content
// addressing makes the bytes safe to cache forever.
func (s *Server) handleBrandingAsset(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/assets/branding/")
	// Path traversal guard: reject anything that could escape the dir or
	// reach into a subdirectory. Filenames are content-addressed SHAs with
	// a fixed extension, so "/" and ".." never appear in legitimate URLs.
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || filename == "" {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join("data", "assets", "branding", filename)
	f, err := os.Open(full) //#nosec G304 -- filename is validated above (no "/" or ".."); path is fixed-prefixed
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close() //#nosec G307 -- read-only file; close error is not actionable

	// Content-addressed → safe to cache forever.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	}

	info, err := f.Stat()
	var modTime time.Time
	if err == nil {
		modTime = info.ModTime()
	}
	http.ServeContent(w, r, filename, modTime, f)
}
