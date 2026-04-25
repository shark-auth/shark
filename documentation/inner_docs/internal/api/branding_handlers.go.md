# branding_handlers.go

**Path:** `internal/api/branding_handlers.go`
**Package:** `api`
**LOC:** 148
**Tests:** likely integration-tested

## Purpose
Admin-key branding CRUD + logo upload (Phase A task A5). The `/admin/branding` subroute exposes GET/PATCH for the single "global" branding row and POST/DELETE for the logo asset. Logo uploads are content-addressed under `data/assets/branding/{sha}{ext}`.

## Handlers exposed
- `handleGetBranding` (line 35) — GET `/admin/branding`. Returns `{branding, fonts}` so the dashboard renders the font picker without a second round-trip.
- `handlePatchBranding` (line 51) — PATCH. Free-form `map[string]any` body; storage applies its own field allowlist (unknown keys silently dropped). Returns the freshly-read row.
- `handleUploadLogo` (line 68) — POST `/admin/branding/logo`. Multipart `logo` field, max 1MiB (enforced via `http.MaxBytesReader`), allowed extensions `.png|.svg|.jpg|.jpeg`. SVG bytes are scanned for `<script`, `<foreignobject`, `javascript:` and rejected if found. Persists pointer + sha to branding row.
- `handleDeleteLogo` (line 142) — DELETE. Clears DB pointer; on-disk file is left in place (content-addressed → safe to leave).

## Key types
None — uses untyped `map[string]any` for PATCH bodies.

## Package state
- `brandingFonts` (line 30) — allowed font slugs: `manrope`, `inter`, `ibm_plex`.

## Imports of note
- `crypto/sha256`, `encoding/hex` — content addressing
- `internal/storage` — `GetBranding`, `UpdateBranding`, `SetBrandingLogo`, `ClearBrandingLogo`

## Wired by
- `internal/api/router.go:649-652`

## Notes
- SVG sanitisation is byte-substring only; the real defence is the asset-serve CSP.
- Logo URL form: `/assets/branding/{sha}.{ext}` (served by `hosted_assets_handler.go`).
