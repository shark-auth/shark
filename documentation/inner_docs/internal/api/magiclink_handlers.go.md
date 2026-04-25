# magiclink_handlers.go

**Path:** `internal/api/magiclink_handlers.go`
**Package:** `api`
**LOC:** 234
**Tests:** `magiclink_handlers_test.go`

## Purpose
Handlers for the email magic-link login flow: send link (rate-limited per email) and verify link (consumes token, mints session, runs auth-flow hook, validates `redirect_uri` against the default application's allowlist).

## Handlers exposed
- `handleMagicLinkSend` (func, line 80) — `POST /api/v1/auth/magic-link/send`; normalises email, applies per-email cooldown (60s), always returns 200 with stable success message to avoid leaking account existence; logs send failures
- `handleMagicLinkVerify` (func, line 123) — `GET /api/v1/auth/magic-link/verify?token=...&redirect_uri=...`; returns 501 when `MagicLinkManager` is nil; surfaces `ErrMagicLinkNotFound|Used|Expired` as distinct error codes; fires `AuthFlowTriggerMagicLink`; sets session cookie; validates redirect against `defaultApp.AllowedCallbackURLs` via `redirect.Validate`

## Key types
- `magicLinkSendRequest` (struct, line 26)
- `magicLinkRateLimiter` (struct, line 31) — per-email cooldown map with goroutine cleanup every 5 min
- Re-exports: `ErrMagicLinkNotFound`, `ErrMagicLinkUsed`, `ErrMagicLinkExpired` (lines 19–23)

## Functions
- `newMagicLinkRateLimiter(cooldown)` (line 37) — starts background cleanup goroutine
- `(rl) allow(email)` (line 54) — true when caller exceeds cooldown
- `(rl) cleanup()` (line 67) — drops entries older than 2× cooldown

## Imports of note
- `internal/auth` — `MagicLinkManager`, error sentinels
- `internal/auth/redirect` — callback URL allowlist validation
- `internal/storage` — `AuthFlowTriggerMagicLink`, `GetDefaultApplication`

## Wired by / used by
- Routes registered in `internal/api/router.go:295–298`
- `magicLinkRL` is owned by `Server` (`internal/api/router.go:142`)

## Notes
- Send always returns the same 200 message regardless of email existence or rate-limit hit — caller can't distinguish.
- Token verification consumes the magic link before flow hook runs; a flow block leaves the user without a session and they must request a fresh link.
- `redirect_uri` defaults to `cfg.MagicLink.RedirectURL`; missing default app returns 500 server_error.
