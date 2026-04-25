# lru.go

**Path:** `internal/proxy/lru.go`
**Package:** `proxy`
**LOC:** 137
**Tests:** `lru_test.go`

## Purpose
Thread-safe LRU cache from cookie-hash to Identity with per-entry TTL and lazy expiration. Used by circuit breaker for both positive (authed) and negative (known-bad) caches.

## Key types / functions
- `lruCache` (struct, line 22) — mutex-protected container/list, entries map, TTL duration, injected now() func
- `lruEntry` (struct, line 33) — key, value (Identity), storedAt, expiresAt timestamps
- `newLRU()` (line 43) — constructs cache with capacity + TTL; clamps size to min 1
- `lruCache.get()` (line 60) — returns (Identity, age, bool); moves hit to front, evicts expired entries lazily
- `lruCache.put()` (line 90) — inserts or overwrites, moves to front, evicts oldest on capacity overflow
- `lruCache.len()` (line 133) — returns current entry count (includes expired entries not yet evicted)

## Imports of note
- `container/list` — O(1) doubly-linked list for recency tracking

## Wired by
- `internal/proxy/circuit.go` (Breaker constructs two caches: positive + negative, line 184-185)

## Used by
- Circuit breaker's Lookup/Miss methods for session identity caching

## Notes
- TTL enforced lazily on get() rather than by sweeper goroutine — dependency-free, avoids second goroutine under Stop() (line 70).
- Injected now() func enables time-travel in tests without sleeping (line 52).
- LRU eviction by capacity: when at max capacity, oldest entry (list.Back) is removed (line 121).
- Negative cache stores zero Identity as presence sentinel (line 14).
- Age clamped to zero if wall-clock skew produces negative duration (line 80).
