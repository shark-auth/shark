# apikey_handlers.go

**Path:** `internal/api/apikey_handlers.go`
**Package:** `api`
**LOC:** 481
**Tests:** likely integration-tested

## Purpose
M2M API key CRUD. Keys are issued once on create/rotate (full plaintext returned exactly once); list/get returns masked display strings + prefix/suffix only. Backs `/api/v1/admin/api-keys/*`.

## Handlers exposed
- `handleCreateAPIKey` (line 65) — POST. Requires name + at least one scope; optional `rate_limit` (falls back to config `DefaultRateLimit`); optional `expires_at` (RFC3339 validated). Mints via `auth.GenerateAPIKey` and audits `api_key.created`.
- `handleListAPIKeys` (line 175) — GET. Wraps result under `{api_keys: [...]}`.
- `handleGetAPIKey` (line 196) — GET `/{id}`.
- `handleUpdateAPIKey` (line 219) — PATCH (name, scopes, rate_limit, expires_at).
- `handleRevokeAPIKey` (line 297) — DELETE (sets revoked_at).
- `handleRotateAPIKey` (line 365) — POST `/{id}/rotate`. Generates a new key + hash, preserves metadata, returns fresh plaintext.

## Key types
- `createAPIKeyRequest` (line 19), `createAPIKeyResponse` (line 27, includes full `Key`)
- `apiKeyResponse` (line 40, masked `KeyDisplay`)
- `updateAPIKeyRequest` (line 55)

## Helpers
- `toAPIKeyResponse` (line 462) — DB row → masked wire shape.

## Imports of note
- `internal/auth` — `GenerateAPIKey`
- `internal/storage` — APIKey CRUD + AuditLog
- `gonanoid` — id suffix (`key_*`)

## Wired by
- `internal/api/router.go:469-474`

## Notes
- Scopes are stored as a JSON-encoded string in the DB column.
- `KeyDisplay` is the masked human-readable form (e.g. `sk_live_AbCd...xK9f`).
