# middleware.go

**Path:** `internal/audit/middleware.go`
**Package:** `audit`
**LOC:** 38
**Tests:** none in same file

## Purpose
Helper to extract client IP and user-agent from inbound HTTP requests for audit logging — proxy-aware (X-Forwarded-For / X-Real-Ip) with a RemoteAddr fallback.

## Key types / functions
- `ExtractRequestInfo(r *http.Request) (ip, userAgent string)` — public entry point. Returns `(ip, userAgent)`.
- `extractIP(r *http.Request) string` — unexported. Order:
  1. `X-Forwarded-For` first comma-separated entry.
  2. `X-Real-Ip`.
  3. `net.SplitHostPort(r.RemoteAddr)` host-only; raw RemoteAddr if split fails.

## Imports
- `net`, `net/http`, `strings` (stdlib only).

## Wired by
- audit-logging middleware in `internal/api/middleware/audit*.go` and any handler that records IP context (login, password change, admin actions).

## Used by
- Anywhere an audit event needs request-origin metadata.

## Notes
- No trusted-proxy configuration — XFF is honoured unconditionally. Sufficient for single-tenant deployments behind a known reverse proxy; review before exposing directly to the internet.
- File is misnamed (`middleware.go`) — it does not return an `http.Handler`; it's a request-info extractor.
