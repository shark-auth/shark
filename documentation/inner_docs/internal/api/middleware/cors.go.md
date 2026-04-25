# middleware/cors.go

**Path:** `internal/api/middleware/cors.go`
**Package:** `middleware`
**LOC:** 45
**Tests:** none direct

## Purpose
Cross-Origin Resource Sharing handler with allowlist semantics. Mounted only when `cfg.Server.CORSOrigins` is non-empty so the default zero-config posture is same-origin-only.

## Middleware exposed
- `CORS(allowedOrigins []string) func(http.Handler) http.Handler` (line 9) — builds a `map[string]bool` of allowed origins, with `*` recognised as "allow all". Behaviour:
  - Requests without an `Origin` header pass through untouched
  - Origins matching the set (or all) get: `Access-Control-Allow-Origin: <origin>`, `Allow-Credentials: true`, `Allow-Headers: Content-Type, Authorization, X-Admin-Key`, `Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS`, `Max-Age: 86400`, `Vary: Origin`
  - `OPTIONS` preflights short-circuit with `204 No Content`
  - Unmatched origins receive no CORS headers — browser blocks the response

## Key types
None.

## Imports of note
- `net/http` only

## Chain order
Mounted at `router.go:215` (global), AFTER `SecurityHeaders` + `RateLimit` + `MaxBodySize`. Conditional: only when `len(cfg.Server.CORSOrigins) > 0`.

## Wired by / used by
- `internal/api/router.go:213–216`

## Notes
- `Allow-Credentials: true` requires explicit origin echo (not `*`) per CORS spec — when `allowAll` is true with `*`, the middleware still echoes the request `Origin` so credentials work.
- `X-Admin-Key` is in `Allow-Headers` even though admin auth is now `Authorization: Bearer sk_live_*` — kept for legacy SDK back-compat.
