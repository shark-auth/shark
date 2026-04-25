# factories.go

**Path:** `internal/testutil/factories.go`
**Package:** `testutil`
**LOC:** 150
**Tests:** none (test helper).

## Purpose
Domain-object factories — `CreateUser`, `CreateRole`, `CreatePermission`, `CreateSession`, `CreateAPIKey`, etc. — that produce IDs, RFC3339 timestamps, and call the corresponding `storage.Store` methods, failing the test on error. Test helper — **not for production runtime.**

## Functions
- `CreateUser(t, store, email, passwordHash)` (line 15) — fresh `usr_<nanoid>`, optional argon2id hash
- `CreateUserWithRole(t, store, email, hash, roleName)` (line 45) — finds-or-creates role, assigns
- `CreateRole(t, store, name)` (line 63)
- `CreatePermission(t, store, action, resource)` (line 84)
- `CreateSession(t, store, userID)` (line 104) — 24h TTL, IP `127.0.0.1`, UA `TestClient/1.0`
- `CreateAPIKey(t, store, name)` (line 128) — `sk_live_t...` prefix, suffix `test`, scopes `users:read,users:write`

## Imports of note
- `gonanoid` for ID generation
- `internal/storage`

## Used by
- Storage-layer tests (`*_test.go` in `internal/storage`)
- API handler tests in `internal/api`

## Notes
- Test helper — **not for production runtime.**
- Test users get `hashType="argon2id"` only when a password hash is supplied; otherwise empty so the user looks like a passwordless account (passkey/magic-link/SSO).
- IDs are real nanoids per the production prefix convention so any string-format assertions stay realistic.
