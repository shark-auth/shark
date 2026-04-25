# middleware/admin.go

**Path:** `internal/api/middleware/admin.go`
**Package:** `middleware`
**LOC:** 88
**Tests:** none direct (covered via `admin_*_handlers_test.go`)

## Purpose
Admin-only authentication middleware: validates a `Bearer sk_live_*` API key and requires it to carry the `*` (wildcard/admin) scope. Distinct from `RequireAPIKey` because admin endpoints skip the rate limiter and update `last_used_at` synchronously to avoid SQLite lock contention on bursty dashboard traffic.

## Middleware exposed
- `AdminAPIKeyFromStore(store, rateLimiter) func(http.Handler) http.Handler` (line 18)
  1. Reads `Authorization: Bearer sk_live_xxx` (or `?token=` for SSE/EventSource which can't set headers)
  2. Format guard: must start with `sk_live_`
  3. Hashes via `auth.HashAPIKey` and looks up in `api_keys` table
  4. Rejects when `revoked_at` is set or `expires_at` is in the past
  5. Parses JSON `scopes` array, requires `*` via `auth.CheckScope` → 403 `forbidden` if missing
  6. Updates `last_used_at` synchronously (admin volume is low) — no goroutine, no rate limiter
  7. Calls `next.ServeHTTP`

## Key types
None. Uses `storage.Store` + `auth.TokenBucket` (the rate limiter param is currently unused — kept for signature parity with `RequireAPIKey`).

## Imports of note
- `internal/auth` — `HashAPIKey`, `CheckScope`, `TokenBucket`
- `internal/storage` — `Store.GetAPIKeyByKeyHash`, `Store.UpdateAPIKey`

## Chain order
Sits at the top of every admin route group in `router.go` (e.g. `r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))` inside `/admin`, `/users`, `/roles`, `/permissions`, `/audit-logs`, `/api-keys`, `/webhooks`, `/agents`, `/admin/apps`, `/sso/connections`, `/vault/providers`, `/migrate`).

## Wired by / used by
- Every admin-key gated route group in `internal/api/router.go`

## Notes
- Error envelopes use `writeJSONError` from `apikey.go` (W18 shape: `{error, message, code, docs_url}`).
- The `?token=` query fallback exists specifically so the dashboard's SSE log stream (`/admin/logs/stream`) can authenticate without `Authorization` header support in `EventSource`.
