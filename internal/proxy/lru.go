package proxy

import (
	"container/list"
	"sync"
	"time"
)

// lruCache is a small thread-safe LRU map from cookie-hash to Identity with
// a per-entry TTL. O(1) get/put backed by container/list. The cache is
// intentionally typed to Identity (rather than a generic value) because the
// circuit breaker only ever stores one kind of value — both the positive
// (authed) and negative (known-bad) caches share this implementation, with
// the negative cache simply storing a zero Identity as a presence sentinel.
//
// TTL is enforced lazily: expired entries are removed on get() rather than
// by a sweeper goroutine. That keeps the implementation dependency-free and
// avoids a second goroutine to reason about under Stop().
//
// The `now` function is injectable so tests can advance virtual time without
// sleeping — real production code always uses time.Now.
type lruCache struct {
	mu      sync.Mutex
	size    int
	list    *list.List
	entries map[string]*list.Element
	ttl     time.Duration
	now     func() time.Time
}

// lruEntry is the element stored in the doubly-linked list. storedAt lets
// get() compute the age returned to callers (for X-Shark-Cache-Age).
type lruEntry struct {
	key       string
	value     Identity
	storedAt  time.Time
	expiresAt time.Time
}

// newLRU constructs an LRU with the given capacity and per-entry TTL. A
// size <= 0 is clamped to 1 so the cache is always capable of holding at
// least one entry (preventing silent data-loss on misconfiguration).
func newLRU(size int, ttl time.Duration) *lruCache {
	if size <= 0 {
		size = 1
	}
	return &lruCache{
		size:    size,
		list:    list.New(),
		entries: make(map[string]*list.Element, size),
		ttl:     ttl,
		now:     time.Now,
	}
}

// get returns (value, age, true) when key is present and unexpired.
// A hit moves the entry to the front of the recency list. An expired entry
// is evicted and get reports a miss. age is always >= 0 — if wall-clock
// skew somehow produced a negative duration we clamp to zero.
func (c *lruCache) get(key string) (Identity, time.Duration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		return Identity{}, 0, false
	}
	entry := elem.Value.(*lruEntry)
	now := c.now()
	if !entry.expiresAt.IsZero() && !now.Before(entry.expiresAt) {
		// Expired — evict and report a miss. This is the only place we
		// clean up expired entries; combined with capacity eviction it's
		// enough to keep memory bounded.
		c.list.Remove(elem)
		delete(c.entries, key)
		return Identity{}, 0, false
	}
	c.list.MoveToFront(elem)
	age := now.Sub(entry.storedAt)
	if age < 0 {
		age = 0
	}
	return entry.value, age, true
}

// put inserts or overwrites key=value. On overwrite the entry is bumped to
// the front. On insert, if the cache is at capacity the least-recently-used
// entry is evicted. The TTL applied is the cache-level ttl captured at
// newLRU — per-entry TTL overrides are not currently needed.
func (c *lruCache) put(key string, value Identity) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	var expires time.Time
	if c.ttl > 0 {
		expires = now.Add(c.ttl)
	}

	if elem, ok := c.entries[key]; ok {
		entry := elem.Value.(*lruEntry)
		entry.value = value
		entry.storedAt = now
		entry.expiresAt = expires
		c.list.MoveToFront(elem)
		return
	}

	entry := &lruEntry{
		key:       key,
		value:     value,
		storedAt:  now,
		expiresAt: expires,
	}
	elem := c.list.PushFront(entry)
	c.entries[key] = elem

	// Evict oldest entries until we're at or below capacity. We use a loop
	// because a concurrent caller could have grown the map between the
	// check-and-evict, though the outer mutex should already prevent that.
	for c.list.Len() > c.size {
		oldest := c.list.Back()
		if oldest == nil {
			break
		}
		c.list.Remove(oldest)
		delete(c.entries, oldest.Value.(*lruEntry).key)
	}
}

// len returns the current number of entries in the cache. Includes expired
// entries that haven't yet been lazily evicted.
func (c *lruCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.list.Len()
}
