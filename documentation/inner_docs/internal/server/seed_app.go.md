# seed_app.go

**Path:** `internal/server/seed_app.go`  
**Package:** `server`  
**LOC:** 171  
**Tests:** `seed_app_test.go`

## Purpose
First-boot default OAuth application initialization. Idempotent seeding of a single "Default Application" with auto-generated client ID and secret on first start.

## Key types / functions
- `seedDefaultApplication(ctx, store, cfg)` (func, line 76) — creates default app if none exists
- `generateClientSecret()` (func, line 66) — 32-byte random → base62 (~43 chars)
- `base62Encode(b)` (func, line 24) — big-endian byte array → base62 string
- `divmod(n, d)` (func, line 55) — long division in-place for base62
- `printDefaultAppBanner(clientID, secret)` (func, line 163) — stdout banner with credentials

## Imports of note
- `crypto/rand` — secure random generation
- `crypto/sha256` — client secret hashing
- `database/sql` — idempotency check via GetDefaultApplication
- `internal/storage` — Application persist

## Wired by
- `Build()` in server.go calls seedDefaultApplication after migrations
- Tests inject MemoryEmailSender to sidestep actual SMTP

## Notes
- Client ID: `shark_app_` + nanoid (21 chars)
- Secret: base62-encoded 32 bytes (~43 chars)
- Secret stored as: SHA256 hash + 8-char prefix for display
- Callback URLs sourced from cfg.Social.RedirectURL + cfg.MagicLink.RedirectURL (deduplicated)
- Race-safe: concurrent first-boot processes detect existing app via re-fetch
- Banner printed to stdout once; callers should capture and store the secret

