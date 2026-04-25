# branding.go

**Path:** `cmd/shark/cmd/branding.go`
**Package:** `cmd`
**LOC:** 111
**Tests:** branding_test.go

## Purpose
Implements `shark branding {set,get}` — manages design tokens via the admin API (`PATCH /api/v1/admin/branding/design-tokens`, `GET /api/v1/admin/branding`).

## Key types / functions
- `brandingCmd` (var, line 13) — parent.
- `brandingSetCmd` (var, line 18) — accepts repeated `--token key=value` pairs OR `--from-file tokens.json`.
  - Sends `{design_tokens: {...}}` payload.
- `brandingGetCmd` (var, line 84) — returns the full branding JSON.

## Imports of note
- `encoding/json`, `os` for file reads.
- Uses `adminDo`, `apiError`, `maybeJSONErr`.

## Wired by / used by
- Registered on `root` in `init()` at line 103.

## Notes
- Lane E, milestone E3.
- The `<slug>` positional arg is reserved for future per-app scoping; today the endpoint is global.
