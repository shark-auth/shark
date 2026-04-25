# paywall.go

**Path:** `cmd/shark/cmd/paywall.go`
**Package:** `cmd`
**LOC:** 64
**Tests:** none direct

## Purpose
Implements `shark paywall preview <slug>` — fetches the rendered paywall HTML from `GET /paywall/{slug}?tier=...&return=...` and prints to stdout, or opens it in the default browser.

## Key types / functions
- `paywallCmd` (var, line 11) — parent.
- `paywallPreviewCmd` (var, line 16):
  - Requires `--tier`; optional `--return` (default `/`); `--open` to launch a browser instead.
  - Uses `adminDoRaw` because the response is HTML, not JSON.
  - Browser launch delegates to `openBrowser` (proxy_admin.go).

## Imports of note
- `net/url`

## Wired by / used by
- Registered on `root` in `init()` at line 58.

## Notes
- Lane E, milestone E2.
