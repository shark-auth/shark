# store.go

**Path:** `internal/oauth/store.go`
**Package:** `oauth`
**LOC:** 535
**Tests:** `store_test.go`

## Purpose
Adapter that satisfies fosite's `ClientManager`, `CoreStorage`, `TokenRevocationStorage`, and `PKCERequestStorage` against SharkAuth's SQLite-backed `storage.Store`.

## RFCs implemented
- RFC 6749 client + token storage
- RFC 7636 PKCE storage (`pkce.PKCERequestStorage`)
- RFC 7009 revocation storage
- RFC 8707 resource-indicator persistence (line 150, 195)

## Key types / functions
- `FositeStore` (struct, line 32) — wraps a `storage.Store`.
- `SHA256Hasher` (struct, line 47) — replaces bcrypt; agents store SHA-256 hex.
- `agentToClient` (func, line 88) — maps `storage.Agent` → `fosite.DefaultOpenIDConnectClient`; honours rotation grace by emitting `RotatedSecrets` while `OldSecretExpiresAt` is in the future (F4.3).
- `GetClient`, `ClientAssertionJWTValid`, `SetClientAssertionJWT` — `fosite.ClientManager` impl.
- `CreateAuthorizeCodeSession` / `GetAuthorizeCodeSession` (line 132 / 170) — auth-code persistence with PKCE + resource fields.
- Refresh-token rotation atomicity patched here on 2026-04-24 (per memory).

## Imports of note
- `github.com/ory/fosite`, `handler/oauth2`, `handler/pkce`
- `internal/storage`
- `github.com/google/uuid`

## Wired by
- `server.go:55` — `NewFositeStore(store)` injected into fosite compose.

## Used by
- All fosite grant handlers (transparent).

## Notes
- Compile-time interface assertions at line 24 catch fosite API breaks early.
- Auth code is stored under `sha256(signature)` so leaked DB rows can't replay codes.
- Revoked-JTI table doubles as the client-assertion replay cache.
- `setSessionSubject` helper bridges fosite's session abstraction to our string user_id.
