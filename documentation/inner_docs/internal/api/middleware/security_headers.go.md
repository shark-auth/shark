# middleware/security_headers.go

**Path:** `internal/api/middleware/security_headers.go`
**Package:** `middleware`
**LOC:** 36
**Tests:** none direct

## Purpose
Sets OWASP-recommended security response headers on every response. Loosens framing/CSP for the hosted SPA and the branding preview iframe so the in-dashboard preview pane renders.

## Middleware exposed
- `SecurityHeaders() func(http.Handler) http.Handler` (line 9) — sets:
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `X-XSS-Protection: 0` (modern browsers ignore the legacy auditor; explicit disable is safer)
  - `Permissions-Policy: camera=(), microphone=(), geolocation=()`
  - Path-conditional framing/CSP:
    - Path starts with `/hosted/` OR `?preview=true` → `X-Frame-Options: SAMEORIGIN`, `Content-Security-Policy: default-src 'self' 'unsafe-inline'; frame-ancestors 'self'`
    - Otherwise → `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'self'; frame-ancestors 'none'`
  - HSTS: `Strict-Transport-Security: max-age=63072000; includeSubDomains` only when `r.TLS != nil` or `X-Forwarded-Proto: https`

## Key types
None.

## Imports of note
- `net/http`, `strings` only — zero internal deps

## Chain order
Mounted at `router.go:210` globally, AFTER `MaxBodySize` and BEFORE `RateLimit` + `CORS`. Applies to every route.

## Wired by / used by
- `internal/api/router.go:210`

## Notes
- The framing exemption is intentionally path-prefixed (`/hosted/`) rather than query-param-only to be more robust against URL rewriting.
- HSTS is omitted on plain HTTP so local `http://localhost` dev doesn't pin clients to HTTPS forever.
- `'unsafe-inline'` in the hosted-page CSP is a known relaxation — the React bundle inlines styles. Tighten with nonces in a future revision.
