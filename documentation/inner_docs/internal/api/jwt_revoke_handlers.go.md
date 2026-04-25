# jwt_revoke_handlers.go

**Path:** `internal/api/jwt_revoke_handlers.go`
**Package:** `api`
**LOC:** 137
**Tests:** likely integration-tested

## Purpose
JWT JTI revocation — a user-facing endpoint that lets the caller revoke their own JWT (current or explicit body token), and an admin endpoint that lets operators revoke any JTI. Revocation tracks `(jti, expires_at)` until the token would naturally expire.

## Handlers exposed
- `handleUserRevoke` (line 30) — POST `/api/v1/auth/revoke`. Session-gated. If body has `{token}`, validates it and ensures `claims.Subject == GetUserID(ctx)` (403 otherwise) before revoking. If body is empty and the auth method is `jwt`, revokes the JTI from context. Always returns 200 with `{message: "Token revoked"}`.
- `handleAdminRevokeJTI` (line 94) — POST `/api/v1/admin/auth/revoke-jti`. Admin-key gated. Body `{jti, expires_at}`. Revokes any JTI for any user. Errors when `JWTManager` is nil (501 not_configured) or fields missing.

## Key types
- `revokeRequest` (line 12) — `{token?}` for user route.
- `adminRevokeJTIRequest` (line 18) — `{jti, expires_at}` for admin route.

## Imports of note
- `internal/api/middleware` (`mw.GetUserID`, `mw.GetClaims`, `mw.GetAuthMethod`)
- `s.JWTManager` (field on Server) — `Validate`, `RevokeJTI`

## Wired by
- `internal/api/router.go:359` (user-facing `/auth/revoke`)
- `internal/api/router.go:536` (admin-key `/admin/auth/revoke-jti`)

## Notes
- Cookie sessions are unaffected unless the caller also holds a JWT — this only revokes JWT JTIs, not sessions.
- 403 ownership check on body-supplied tokens prevents one user revoking another user's JWT.
