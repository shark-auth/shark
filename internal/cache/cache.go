package cache

import (
	"sync"
	"time"
)

// AuthDecision represents a precomputed state for a user/session.
// By caching this, we avoid hitting the database for standard auth/RBAC checks.
type AuthDecision struct {
	UserID        string
	SessionID     string
	EmailVerified bool
	MFAPassed     bool
	Tier          string
	// EffectivePermissions could be a map or slice. Using a simple struct for now.
	// In a full implementation, this holds the flattened RBAC graph.
	Permissions   map[string]bool 
	ExpiresAt     time.Time
}

// Cache is a simple thread-safe, TTL-based memory cache for AuthDecisions.
type Cache struct {
	mu    sync.RWMutex
	items map[string]AuthDecision
	ttl   time.Duration
}

// New creates a new AuthDecision cache.
func New(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]AuthDecision),
		ttl:   ttl,
	}
	go c.cleanupLoop()
	return c
}

// Set stores a decision in the cache.
func (c *Cache) Set(key string, decision AuthDecision) {
	c.mu.Lock()
	defer c.mu.Unlock()
	decision.ExpiresAt = time.Now().Add(c.ttl)
	c.items[key] = decision
}

// Get retrieves a decision from the cache if it exists and is not expired.
func (c *Cache) Get(key string) (AuthDecision, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	if !found {
		return AuthDecision{}, false
	}
	if time.Now().After(item.ExpiresAt) {
		return AuthDecision{}, false
	}
	return item, true
}

// Delete removes an item from the cache (e.g., on logout or permission change).
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// cleanupLoop periodically removes expired items to prevent unbounded growth.
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.ExpiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
