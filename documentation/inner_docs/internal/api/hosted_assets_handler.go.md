# hosted_assets_handler.go

**Path:** `internal/api/hosted_assets_handler.go`
**Package:** `api`
**LOC:** 51
**Tests:** none colocated

## Purpose
Public, no-auth handler that serves content-addressed branding assets (logos) from `data/assets/branding/{sha}{ext}`. Public because these URLs are embedded into outbound emails and external sites; content addressing makes the bytes safe to cache forever.

## Handlers exposed
- `handleBrandingAsset` (line 15) — `GET/HEAD /assets/branding/*`. Path-traversal guard rejects any filename containing `..` or `/`; opens the fixed-prefixed path; sets `Cache-Control: public, max-age=31536000, immutable`; sets Content-Type from extension (`.png|.svg|.jpg|.jpeg`); serves via `http.ServeContent` (handles ETags + Range).

## Key types
None.

## Imports of note
- `os`, `path/filepath`, `net/http` — stdlib only.

## Wired by
- `internal/api/router.go:234-235` (mounted before auth middleware so the asset is publicly served).

## Notes
- No auth — branding logos are intentionally public.
- Content-addressed URLs (the SHA is in the path) → safe to set immutable cache for 1 year.
- Note: `handleHostedAssets` (the SPA bundle) is a separate handler in `hosted_handlers.go` despite the name overlap.
