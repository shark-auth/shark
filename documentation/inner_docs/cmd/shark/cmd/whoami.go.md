# whoami.go

**Path:** `cmd/shark/cmd/whoami.go`
**Package:** `cmd`
**LOC:** 97
**Tests:** none direct

## Purpose
Implements `shark whoami` — calls `GET /api/v1/auth/me` to introspect the configured token; falls back to `/api/v1/admin/stats` to detect when the token is an admin API key rather than a user JWT.

## Key types / functions
- `whoamiCmd` (var, line 17):
  - 401 from `/auth/me` triggers a probe of `/admin/stats`; if 200, reports `token_type: admin_api_key`.
  - Otherwise prints `id`, `email`, `tier` (or full JSON with `--json`).
- `extractNestedString` (func, line 75) — walks a nested map[string]any path.

## Imports of note
- Uses `adminDo`, `maybeJSONErr`, `apiError`.

## Wired by / used by
- Registered on `root` in `init()` at line 95.

## Notes
- Lane E, milestone E6.
- Tier is read from `metadata.tier` in the user response.
