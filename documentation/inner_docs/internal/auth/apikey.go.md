# apikey.go

**Path:** `internal/auth/apikey.go`
**Package:** `auth`
**LOC:** 197
**Tests:** `apikey_test.go`

## Purpose
API key generation, hashing, scope checking, and per-key in-memory token-bucket rate limiting.

## Key types / functions
- Constants (line 15-27): `apiKeyPrefix="sk_live_"`, `base62Chars`, `randomBytesLen=32`, `keyPrefixLen=8`, `keySuffixLen=4`.
- `GenerateAPIKey` (func, line 33) — 32 random bytes → base62; returns `(fullKey, sha256Hash, displayPrefix, displaySuffix)` (Stripe-style mask `sk_live_AbCd...xK9f`).
- `HashAPIKey` (func, line 49) — hex(SHA-256(key)).
- `ValidateAPIKey` (func, line 56) — constant-time compare of hex hashes.
- `CheckScope` (func, line 63) — supports `"*"` global, exact match, and `"resource:*"` wildcards.
- `base62Encode` (func, line 82) — big.Int division loop with leading-zero padding.
- `bucket` (type, line 113) — single token bucket (tokens, max, refillRate, lastRefill).
- `TokenBucket` (type, line 121) — mutex-guarded `map[keyHash]*bucket` rate limiter.
- `NewTokenBucket` (func, line 127) — spawns 5-min cleanup goroutine.
- `Allow` (func, line 146) — refills based on elapsed seconds × (limit/3600 per-second rate); decrements; returns false when tokens<1.
- `cleanup` (func, line 187) — drops buckets idle >10 min and currently full.

## Imports of note
- `crypto/sha256`, `crypto/subtle` — hashing + CT compare.
- `math/big` — base62 conversion.
- `sync`, `time` — bucket bookkeeping.

## Used by
- `internal/api/admin_apikey_handlers.go` — issue/list/revoke.
- `internal/api/middleware.go` — request validation + rate limiting.

## Notes
- **PROCESS-LOCAL rate limiter** (see `SCALE.md`) — buckets are per-replica, so total rate scales with instance count. Move to Redis for accurate cluster-wide limits.
- Refill rate is `limit/3600` tokens/sec; bursting up to `limit` is allowed.
- DB stores only the SHA-256 hash; full key is shown to user once at issue time.
