# errors.go

**Path:** `internal/oauth/errors.go`
**Package:** `oauth`
**LOC:** 62
**Tests:** `errors_test.go`

## Purpose
Canonical RFC 6749 §5.2 error envelope used by every `/oauth/**` endpoint. Disjoint from the SharkAuth admin-API envelope in `internal/api/errors.go` because OAuth client libraries assume the standard shape.

## RFCs implemented
- RFC 6749 §5.2 error response body
- Constants for OIDC, RFC 7009, RFC 7662, RFC 8628, RFC 9449 extension errors.

## Key types / functions
- `OAuthError` (struct, line 19) — `{error, error_description?, error_uri?}` only. Comment forbids extra top-level fields.
- `WriteOAuthError` (func, line 28) — sets Content-Type, `Cache-Control: no-store`, `Pragma: no-cache`, writes status, encodes JSON.
- `NewOAuthError` (func, line 37) — convenience builder.
- Error-code constants (line 43-62): `ErrInvalidRequest`, `ErrInvalidClient`, `ErrInvalidGrant`, `ErrInvalidDPoPProof`, `ErrAuthorizationPending`, `ErrSlowDown`, etc.

## Imports of note
- `encoding/json`, `net/http` only.

## Wired by
- All sibling files in this package use the constants for error codes.

## Used by
- `dcr.go`, `device.go`, `exchange.go`, `introspect.go`, `revoke.go`, `handlers.go`.

## Notes
- Comment at top is load-bearing: do NOT swap in `internal/api/errors.go` — AppAuth/oauth2-proxy/Authlib will break.
- Shark-specific extensions must piggyback on `error_uri` or a separate response header, never a new JSON field.
