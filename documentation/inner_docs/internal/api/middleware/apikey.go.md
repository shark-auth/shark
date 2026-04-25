# middleware/apikey.go

**Path:** `internal/api/middleware/apikey.go`
**Package:** `middleware`
**LOC:** 159
**Tests:** none direct

## Purpose
Bearer API-key authentication for non-admin scopes — supports per-scope authorization, per-key rate limiting via `TokenBucket`, and async `last_used_at` updates. Also defines the API-key context keys + accessors and the shared `writeJSONError` helper used by all middleware files in this package (avoids an `internal/api` import cycle).

## Middleware exposed
- `RequireAPIKey(store, rateLimiter, requiredScope) func(http.Handler) http.Handler` (line 50)
  1. Parses `Authorization: Bearer sk_live_xxx`
  2. Hashes with SHA-256 (`auth.HashAPIKey`)
  3. Looks up `api_keys` by hash; rejects revoked / expired
  4. Verifies `requiredScope` against JSON-encoded `scopes` array via `auth.CheckScope`
  5. Applies token-bucket rate limit per key hash (returns 429 + `Retry-After: 60`)
  6. Spawns 5s-bounded goroutine to bump `last_used_at` (detached from request ctx so caller drop doesn't cancel)
  7. Stashes API key ID, scopes, name on the request context

## Context accessors
- `GetAPIKeyID(ctx) string` (line 24)
- `GetAPIKeyScopes(ctx) []string` (line 32)
- Context keys: `APIKeyIDKey`, `APIKeyScopesKey`, `APIKeyNameKey` (lines 14–21)

## Helpers
- `writeJSONError(w, status, errCode, message)` (line 150) — emits W18 envelope `{error, message, code, docs_url}` (`docs_url = https://docs.shark-auth.com/errors/<code>`); duplicated here because middleware can't import `internal/api`

## Imports of note
- `internal/auth` — `HashAPIKey`, `CheckScope`, `TokenBucket`
- `internal/storage` — `Store.GetAPIKeyByKeyHash`, `Store.UpdateAPIKey`

## Chain order
Currently no router site uses `RequireAPIKey` directly — admin endpoints use the dedicated `AdminAPIKeyFromStore`. This middleware exists for future per-scope keys (e.g. `users:read`, `webhooks:emit`) that don't grant full admin.

## Wired by / used by
- `writeJSONError` is consumed by `admin.go`, `auth.go` (the JWT/cookie middleware), and itself.

## Notes
- Bounded `last_used_at` goroutine uses `context.Background()` + 5s `WithTimeout` so it can't leak across shutdown.
- Rate limiter is keyed by `keyHash` (not key ID) so a leaked key is throttled regardless of key ID enumeration.
