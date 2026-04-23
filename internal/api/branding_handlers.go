// Package api — admin-key-authenticated branding CRUD + logo upload handlers.
//
// Part of Phase A (task A5) of the branding + hosted-components plan. The
// /admin/branding subroute exposes GET/PATCH for the single "global" branding
// row and POST/DELETE for the logo asset. Logo uploads are written to
// data/assets/branding/{sha}{ext} and served by a sibling handler (A6).
//
// SVG uploads are rejected when they contain obviously-scripted content. The
// scan is intentionally a byte-level substring check, not a real XML parse —
// a full sanitizer is far too easy to bypass, so the strategy is "reject the
// obvious stuff and rely on the asset-serve CSP as the real guard rail".
package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// brandingFonts lists the allowed font-family slugs exposed by the admin UI.
// Kept as a package-level var to match the list wired into the dashboard
// font picker; the storage layer's allowlist filter does not constrain
// values by shape, so the UI is the source of truth.
var brandingFonts = []string{"manrope", "inter", "ibm_plex"}

// handleGetBranding handles GET /admin/branding. Returns the global branding
// row plus the available font-family slugs so the dashboard can render a
// dropdown without a second round-trip.
func (s *Server) handleGetBranding(w http.ResponseWriter, r *http.Request) {
	b, err := s.Store.GetBranding(r.Context(), "global")
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"branding": b,
		"fonts":    brandingFonts,
	})
}

// handlePatchBranding handles PATCH /admin/branding. The request body is an
// arbitrary field->value map; the storage layer applies its own allowlist so
// unknown keys are silently dropped. Returns the freshly-read branding on
// success so the caller never has to re-GET.
func (s *Server) handlePatchBranding(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if err := s.Store.UpdateBranding(r.Context(), "global", body); err != nil {
		internal(w, err)
		return
	}
	s.handleGetBranding(w, r)
}

// handleUploadLogo handles POST /admin/branding/logo. Reads the "logo"
// multipart field, caps the total body at 1MiB, validates extension + SVG
// script content, hashes the bytes, writes to data/assets/branding/{sha}{ext},
// and persists the pointer + sha into the branding row.
func (s *Server) handleUploadLogo(w http.ResponseWriter, r *http.Request) {
	// MaxBytesReader wraps r.Body so ParseMultipartForm will surface a size
	// error without buffering the full payload into memory first.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("file_too_large", "Logo must be 1MB or smaller"))
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Missing 'logo' file field"))
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".png", ".svg", ".jpg", ".jpeg":
		// allowed
	default:
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_file_type", "Logo must be .png, .svg, .jpg, or .jpeg"))
		return
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		// Most likely the MaxBytesReader firing mid-stream.
		writeJSON(w, http.StatusBadRequest, errPayload("file_too_large", "Logo must be 1MB or smaller"))
		return
	}

	if ext == ".svg" {
		lower := bytes.ToLower(buf.Bytes())
		if bytes.Contains(lower, []byte("<script")) ||
			bytes.Contains(lower, []byte("<foreignobject")) ||
			bytes.Contains(lower, []byte("javascript:")) {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_svg", "SVG contains disallowed scripting content"))
			return
		}
	}

	sum := sha256.Sum256(buf.Bytes())
	sha := hex.EncodeToString(sum[:])
	filename := sha + ext

	dir := filepath.Join("data", "assets", "branding")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		internal(w, err)
		return
	}
	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, buf.Bytes(), 0o644); err != nil {
		internal(w, err)
		return
	}

	url := "/assets/branding/" + filename
	if err := s.Store.SetBrandingLogo(r.Context(), "global", url, sha); err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"logo_url": url,
		"logo_sha": sha,
	})
}

// handleDeleteLogo handles DELETE /admin/branding/logo. Clears the DB pointer
// and returns 204; the on-disk asset is left in place (content-addressed
// storage means old SHAs can safely linger, and a future GC pass can sweep
// orphans once per-app overrides land).
func (s *Server) handleDeleteLogo(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.ClearBrandingLogo(r.Context(), "global"); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
