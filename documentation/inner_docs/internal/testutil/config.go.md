# config.go

**Path:** `internal/testutil/config.go`
**Package:** `testutil`
**LOC:** 72
**Tests:** none (test helper).

## Purpose
Returns a `*config.Config` with safe defaults suitable for tests: in-memory storage, reduced argon2id parameters (16MB / 1 iter, vs production 64MB / 3 iter), test secrets, dev SMTP host. Test helper — **not for production runtime.**

## Functions
- `TestConfig()` (line 9) — returns the fully populated config.

## Sections seeded
- Server (port 0 random, 32-byte secret, base URL `localhost:8080`)
- Storage (`:memory:`)
- Auth (24h sessions, password min 8, reduced argon2id, JWT enabled in session mode)
- Passkeys (`SharkAuth Test`, RPID `localhost`)
- MagicLink (10m TTL)
- SMTP (localhost:1025 — mailpit/mailhog convention)
- MFA, APIKeys, Audit

## Imports of note
- `internal/config`

## Used by
- `internal/testutil/server.go::NewTestServer` and other test entry points

## Notes
- Test helper — **not for production runtime.**
- Argon2id deliberately weakened so the unit suite stays under a few seconds.
