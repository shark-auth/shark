# middleware/ratelimit.go

**Path:** `internal/api/middleware/ratelimit.go`
**Package:** `middleware`
**LOC:** 131
**Tests:** `ratelimit_test.go`

## Purpose
Per-IP token-bucket rate limiter for the global router. Designed to throttle anonymous/unauthenticated traffic before it reaches handlers — API-key requests have their own per-key rate limiting in `apikey.go`.

## Middleware exposed
- `RateLimit(maxTokens, refillRate float64) func(http.Handler) http.Handler` (line 108) — wraps every request, picks a client IP (prefers `X-Real-Ip` set by chi `RealIP`, falls back to `RemoteAddr`), looks up or creates the per-IP `tokenBucket`, calls `allow()`. On rejection emits `Retry-After: 1` + 429 with body `{"error":"rate_limited","message":"Too many requests"}`.

## Internal types
- `tokenBucket` (struct, line 10) — `tokens`, `maxTokens`, `refillRate` (tokens/sec), `lastRefill`, `mu`
  - `newTokenBucket(maxTokens, refillRate)` (line 18)
  - `allow()` (line 27) — refills lazily based on elapsed wall time
- `rateLimiter` (struct, line 47) — concurrent map of `ip → *tokenBucket`
  - `newRateLimiter(maxTokens, refillRate)` (line 54) — starts a 5-min ticker goroutine for `cleanup`
  - `allow(ip)` (line 73) — double-checked lock to create-on-miss
  - `cleanup()` (line 92) — drops buckets idle > 10 min and currently full

## Imports of note
- `net/http`, `sync`, `time` only — zero internal deps

## Chain order
Mounted at `router.go:211` globally with `RateLimit(100, 100)` (100 burst, 100 tokens/sec refill) AFTER `SecurityHeaders` and BEFORE `CORS`.

## Wired by / used by
- `internal/api/router.go:211` only — no per-route override.

## Notes
- Pure in-memory state per process — multi-replica deployments do NOT share buckets. Front a load balancer with shared rate limiting if you need cluster-wide throttling.
- Cleanup is conservative (10 min idle + bucket full) to avoid evicting actively-throttled IPs prematurely.
- Per-IP keying makes NAT collateral damage possible; pair with `RealIP` to get the real client when behind a proxy.
