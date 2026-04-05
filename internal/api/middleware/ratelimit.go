package middleware

import (
	"net/http"
	"sync"
	"time"
)

// tokenBucket implements a simple token bucket rate limiter for a single client.
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(maxTokens, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

// rateLimiter holds per-IP token buckets.
type rateLimiter struct {
	buckets    map[string]*tokenBucket
	mu         sync.RWMutex
	maxTokens  float64
	refillRate float64
}

func newRateLimiter(maxTokens, refillRate float64) *rateLimiter {
	rl := &rateLimiter{
		buckets:    make(map[string]*tokenBucket),
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}

	// Background cleanup of stale buckets every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.RLock()
	bucket, ok := rl.buckets[ip]
	rl.mu.RUnlock()

	if !ok {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		bucket, ok = rl.buckets[ip]
		if !ok {
			bucket = newTokenBucket(rl.maxTokens, rl.refillRate)
			rl.buckets[ip] = bucket
		}
		rl.mu.Unlock()
	}

	return bucket.allow()
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, bucket := range rl.buckets {
		bucket.mu.Lock()
		if bucket.lastRefill.Before(cutoff) && bucket.tokens >= bucket.maxTokens {
			delete(rl.buckets, ip)
		}
		bucket.mu.Unlock()
	}
}

// RateLimit returns middleware that applies IP-based token bucket rate limiting.
// maxTokens is the burst capacity, refillRate is tokens added per second.
func RateLimit(maxTokens, refillRate float64) func(http.Handler) http.Handler {
	limiter := newRateLimiter(maxTokens, refillRate)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			if !limiter.allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate_limited","message":"Too many requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
