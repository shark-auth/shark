# branding.go

**Path:** `internal/storage/branding.go`
**Package:** `storage`
**LOC:** 146
**Tests:** `branding_test.go`

## Purpose
SQLite implementation of branding CRUD plus `ResolveBranding(appID)` — the merge of global branding row + per-application override JSON. The `BrandingConfig` struct itself lives in `storage.go`.

## Interface methods implemented
- `GetBranding` (14) — fetch a single branding row by id (typically `"global"`)
- `UpdateBranding` (33) — partial update with allowlist (primary/secondary color, font_family, footer_text, email_from_name/address, **design_tokens** JSON blob from PROXYV1_5 §4.11)
- `SetBrandingLogo` (79), `ClearBrandingLogo` (87)
- `ResolveBranding` (96) — merges `applications.branding_override` JSON over the global row; falls back to global on any error or empty override

## Tables touched
- branding (read/write)
- applications (read-only join to fetch `branding_override`)

## Imports of note
- `database/sql`, `encoding/json`, `errors`, `strings`, `time`

## Used by
- `internal/api/branding.go` admin CRUD
- `internal/api/hosted` — hosted login pages for theming
- `internal/email` — email-from name/address + footer
- Dashboard design-tokens editor (H4)

## Notes
- Allowlist on `UpdateBranding` silently drops unknown fields — safer than 400-ing on forward-compat columns.
- `design_tokens` accepts a nested JSON object from the dashboard; this file stringifies it before INSERT so the column (TEXT NOT NULL) always holds valid JSON.
- `ResolveBranding` is "best-effort merge": malformed override JSON returns the global row rather than erroring, so a corrupted app row never breaks login UX.
