# admin_bootstrap_handlers.go

**Path:** `internal/api/admin_bootstrap_handlers.go`
**Package:** `api`
**LOC:** 205
**Tests:** none colocated (covered indirectly via server boot tests)

## Purpose
First-boot bootstrap token (T15) — mints a one-time URL on a fresh install that lets the operator exchange the printed token for a real admin API key without copy-pasting `sk_live_...` from the server log.

## Handlers exposed
- `MintBootstrapToken` (method, line 68) — called by `server.Serve` at startup. Probes audit_logs for any `admin.*` entry; if absent, generates a 32-byte hex token, stores its SHA-256 hash + 10-min expiry in memory, and returns the raw token for stdout.
- `handleBootstrapConsume` (line 114) — POST `/api/v1/admin/bootstrap/consume`. No auth middleware (the token IS the credential); validates token (constant-time hash compare, single-use, expiry), then mints a `key_*` admin API key with scopes `["*"]` and audits `admin.bootstrap.consumed`.

## Key types
- `bootstrapToken` (struct, line 45) — in-memory token: hash + expiresAt + consumed flag.
- `bootstrapConsumeRequest` (line 102), `bootstrapConsumeResponse` (line 107)

## Imports of note
- `crypto/rand`, `crypto/sha256`, `crypto/subtle` — token generation + constant-time compare
- `internal/auth` — `GenerateAPIKey`
- `internal/storage` — APIKey + AuditLog persistence

## Wired by
- `internal/api/router.go:576` (`POST /admin/bootstrap/consume`)

## Notes
- Token TTL = 10 min (`bootstrapTokenTTL`). Server restart wipes state.
- `consumed=true` flips under the same mutex as the hash compare so concurrent requests cannot both win.
- Returns `("", nil)` from `MintBootstrapToken` when an admin has already acted (so subsequent boots don't print a new URL).
