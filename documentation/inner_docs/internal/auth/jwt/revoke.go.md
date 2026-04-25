# revoke.go

**Path:** `internal/auth/jwt/revoke.go`
**Package:** `jwt`
**LOC:** 15
**Tests:** covered by `manager_test.go`

## Purpose
JTI revocation insert with lazy pruning of expired rows.

## Key types / functions
- `RevokeJTI` (method on `*Manager`, line 10) — calls `store.PruneExpiredRevokedJTI` (best-effort, error ignored) then `store.InsertRevokedJTI(jti, expiresAt)`.

## Imports of note
- `context`, `time` — only.

## Used by
- `internal/api/auth_handlers.go` — logout path explicitly revokes the active token's JTI.
- `internal/auth/jwt/manager.go` — refresh-token rotation revokes the old refresh JTI (line 375 of manager.go).

## Notes
- Pruning runs on every revoke call to keep the table bounded without a separate background sweeper.
- `expiresAt` matches the token's `exp` so pruning is safe — once the underlying token is expired, the revocation row is no longer needed.
