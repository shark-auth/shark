# middleware/bodylimit.go

**Path:** `internal/api/middleware/bodylimit.go`
**Package:** `middleware`
**LOC:** 16
**Tests:** `bodylimit_test.go`

## Purpose
Caps request body size to a configurable byte limit so handlers can decode JSON without worrying about RAM exhaustion from oversized uploads. Sized to 1 MB by default at the global router (`internal/api/router.go:209`).

## Middleware exposed
- `MaxBodySize(maxBytes int64) func(http.Handler) http.Handler` (line 9) — wraps `r.Body` in `http.MaxBytesReader(w, r.Body, maxBytes)`. Subsequent decoders that exceed `maxBytes` will see `*http.MaxBytesError`, and `MaxBytesReader` itself emits a 413 Request Entity Too Large response automatically once the threshold is crossed during read.

## Key types
None.

## Imports of note
- `net/http` only — zero internal deps

## Chain order
Mounted as a global middleware at `router.go:209`, before `SecurityHeaders`, `RateLimit`, and `CORS`. Applies to every route in the router.

## Wired by / used by
- `internal/api/router.go:209` → `r.Use(mw.MaxBodySize(1 << 20))` (1 MiB)

## Notes
- 413 response shape is plaintext from `http.MaxBytesReader` (no JSON envelope) — handlers that decode `r.Body` should error-check decoder failures and surface a JSON error themselves.
- Bypass any future endpoint that legitimately needs > 1 MB (e.g. logo uploads — currently within 1 MB) by mounting with a larger limit at that route group instead of at the router.
