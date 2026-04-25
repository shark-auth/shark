# circuit.go

**Path:** `internal/proxy/circuit.go`
**Package:** `proxy`
**LOC:** 637
**Tests:** `circuit_test.go`

## Purpose
Session cache + background health monitor implementing per-instance circuit breaker (closed → open → half-open state machine) to gracefully degrade when auth server is unhealthy.

## Key types / functions
- `BreakerState` (enum, line 22) — Closed (healthy, hit auth live), Open (cache fallback), HalfOpen (probing)
- `BreakerConfig` (struct, line 77) — HealthURL, HealthInterval (10s default), FailureThreshold (3), CacheSize (10k), CacheTTL (5m), NegativeTTL (30s), MissBehavior (reject|allow_readonly)
- `Breaker` (struct, line 142) — circuit state machine with positive/negative LRU caches, background probe goroutine
- `CachedIdentity` (struct, line 168) — Identity + CachedAt timestamp for age reporting
- `Breaker.Start()` (line 198) — launches health monitor goroutine with context cancellation
- `Breaker.Lookup()` — positive cache hit, returns (Identity, age, bool)
- `Breaker.probe()` — background goroutine polls HealthURL, updates state on failures
- `BreakerResolver.Resolve()` — session → (cached Identity, age) or error if open and no cache

## Imports of note
- `crypto/sha256` — hashing session cookies for cache keys
- Standard http.Client for health probes

## Wired by
- `internal/proxy/listener.go` (NewListener constructs Breaker with config)
- Auth middleware that wraps requests to check cache before hitting auth server

## Used by
- Reverse proxy request flow when identity resolution is needed
- Dashboard to read circuit state via Breaker.Stats()

## Notes
- Health probes run on HealthInterval (default 10s) and time out at HealthTimeout (3s default).
- FailureThreshold (3 consecutive) opens circuit; any success in HalfOpen closes it.
- Positive cache (CacheTTL=5m) stores authed identities; negative cache (NegativeTTL=30s) stores known-bad tokens.
- MissBehavior="allow_readonly" permits GET/HEAD with anonymous-degraded identity when open and no cache (line 58, 106).
- Per-instance state — each Listener has its own Breaker, no shared state across listeners (SCALE.md).
- Lazy TTL expiration on get() avoids sweeper goroutine dependency (line 70).
