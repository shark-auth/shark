package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

const (
	// apiKeyPrefix is prepended to all generated API keys.
	apiKeyPrefix = "sk_live_"
	// base62Chars is the character set used for base62 encoding.
	base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	// randomBytesLen is the number of random bytes used for key generation.
	randomBytesLen = 32
	// keyPrefixLen is how many characters after sk_live_ to store as the display prefix.
	keyPrefixLen = 8
)

// keySuffixLen is how many characters from the end of the encoded key to store for masked display.
const keySuffixLen = 4

// GenerateAPIKey creates a new API key: sk_live_ + 32 random bytes base62-encoded.
// Returns the full key (shown once to the user), its SHA-256 hash (stored in DB),
// a display prefix (first 8 chars after sk_live_), and a display suffix (last 4 chars)
// for masked display like OpenAI: sk_live_AbCd...xK9f
func GenerateAPIKey() (fullKey string, keyHash string, keyPrefix string, keySuffix string, err error) {
	randomBytes := make([]byte, randomBytesLen)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", "", fmt.Errorf("generating random bytes: %w", err)
	}

	encoded := base62Encode(randomBytes)
	fullKey = apiKeyPrefix + encoded
	keyHash = HashAPIKey(fullKey)
	keyPrefix = encoded[:keyPrefixLen]
	keySuffix = encoded[len(encoded)-keySuffixLen:]

	return fullKey, keyHash, keyPrefix, keySuffix, nil
}

// HashAPIKey returns the hex-encoded SHA-256 hash of the given API key.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// ValidateAPIKey hashes the provided key and compares it against the stored hash
// using constant-time comparison to prevent timing attacks.
func ValidateAPIKey(key, storedHash string) bool {
	computedHash := HashAPIKey(key)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}

// CheckScope verifies that the given action (e.g. "users:read") is allowed by
// the provided scopes list. Supports wildcard "*" scope for full access.
func CheckScope(scopes []string, action string) bool {
	for _, s := range scopes {
		if s == "*" {
			return true
		}
		if s == action {
			return true
		}
		// Support resource-level wildcards like "users:*"
		parts := strings.SplitN(s, ":", 2)
		actionParts := strings.SplitN(action, ":", 2)
		if len(parts) == 2 && len(actionParts) == 2 && parts[0] == actionParts[0] && parts[1] == "*" {
			return true
		}
	}
	return false
}

// base62Encode encodes raw bytes into a base62 string.
func base62Encode(data []byte) string {
	n := new(big.Int).SetBytes(data)
	base := big.NewInt(int64(len(base62Chars)))
	zero := big.NewInt(0)
	mod := new(big.Int)

	var result []byte
	for n.Cmp(zero) > 0 {
		n.DivMod(n, base, mod)
		result = append(result, base62Chars[mod.Int64()])
	}

	// Pad with leading zeros for any leading zero bytes in input
	for _, b := range data {
		if b != 0 {
			break
		}
		result = append(result, base62Chars[0])
	}

	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// --- Token Bucket Rate Limiter ---

// bucket represents a single token bucket for one API key.
type bucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second (derived from requests per hour)
	lastRefill time.Time
}

// TokenBucket is an in-memory per-API-key rate limiter using the token bucket algorithm.
type TokenBucket struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

// NewTokenBucket creates a new TokenBucket rate limiter.
func NewTokenBucket() *TokenBucket {
	tb := &TokenBucket{
		buckets: make(map[string]*bucket),
	}

	// Background cleanup of stale buckets every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			tb.cleanup()
		}
	}()

	return tb
}

// Allow checks if a request is allowed under the rate limit for the given key hash.
// The limit parameter is the maximum number of requests per hour for this key.
func (tb *TokenBucket) Allow(keyHash string, limit int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	maxTokens := float64(limit)
	refillRate := float64(limit) / 3600.0 // tokens per second

	b, ok := tb.buckets[keyHash]
	if !ok {
		b = &bucket{
			tokens:     maxTokens,
			maxTokens:  maxTokens,
			refillRate: refillRate,
			lastRefill: time.Now(),
		}
		tb.buckets[keyHash] = b
	}

	// Update rate limit if it changed
	if b.maxTokens != maxTokens {
		b.maxTokens = maxTokens
		b.refillRate = refillRate
	}

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// cleanup removes stale buckets that haven't been used in 10 minutes.
func (tb *TokenBucket) cleanup() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for key, b := range tb.buckets {
		if b.lastRefill.Before(cutoff) && b.tokens >= b.maxTokens {
			delete(tb.buckets, key)
		}
	}
}
