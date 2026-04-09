package auth

import (
	"strings"
	"testing"
)

func TestAPIKeyGeneration(t *testing.T) {
	fullKey, keyHash, keyPrefix, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	// Must start with sk_live_ prefix
	if !strings.HasPrefix(fullKey, "sk_live_") {
		t.Errorf("expected key to start with sk_live_, got %q", fullKey)
	}

	// Key hash must not be empty
	if keyHash == "" {
		t.Error("expected non-empty key hash")
	}

	// Key prefix must be 8 chars
	if len(keyPrefix) != 8 {
		t.Errorf("expected key prefix length 8, got %d (%q)", len(keyPrefix), keyPrefix)
	}

	// Key prefix must be the first 8 chars after sk_live_
	afterPrefix := strings.TrimPrefix(fullKey, "sk_live_")
	if !strings.HasPrefix(afterPrefix, keyPrefix) {
		t.Errorf("key prefix %q does not match start of key body %q", keyPrefix, afterPrefix[:8])
	}

	// Full key should be reasonably long (sk_live_ = 8 chars + base62 encoded 32 bytes ~ 43 chars)
	if len(fullKey) < 40 {
		t.Errorf("expected key length >= 40, got %d", len(fullKey))
	}

	// Generate a second key and verify uniqueness
	fullKey2, keyHash2, _, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() second call error: %v", err)
	}
	if fullKey == fullKey2 {
		t.Error("two generated keys should not be identical")
	}
	if keyHash == keyHash2 {
		t.Error("two key hashes should not be identical")
	}
}

func TestAPIKeyHashRoundTrip(t *testing.T) {
	fullKey, keyHash, _, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	// Hashing the full key should produce the same hash
	computedHash := HashAPIKey(fullKey)
	if computedHash != keyHash {
		t.Errorf("expected hash %q, got %q", keyHash, computedHash)
	}

	// ValidateAPIKey should return true for the correct key
	if !ValidateAPIKey(fullKey, keyHash) {
		t.Error("ValidateAPIKey should return true for matching key and hash")
	}
}

func TestAPIKeyWrongKey(t *testing.T) {
	_, keyHash, _, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	// A different key should not validate against the stored hash
	wrongKey := "sk_live_thisisthewrongkeycompletelyXXXXXXXX"
	if ValidateAPIKey(wrongKey, keyHash) {
		t.Error("ValidateAPIKey should return false for wrong key")
	}

	// An empty key should not validate
	if ValidateAPIKey("", keyHash) {
		t.Error("ValidateAPIKey should return false for empty key")
	}
}

func TestHashAPIKeyDeterministic(t *testing.T) {
	key := "sk_live_testkey12345678901234567890"
	hash1 := HashAPIKey(key)
	hash2 := HashAPIKey(key)
	if hash1 != hash2 {
		t.Error("HashAPIKey should be deterministic")
	}
}

func TestCheckScope(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		action   string
		expected bool
	}{
		{"exact match", []string{"users:read", "users:write"}, "users:read", true},
		{"no match", []string{"users:read"}, "users:write", false},
		{"wildcard all", []string{"*"}, "anything:here", true},
		{"resource wildcard", []string{"users:*"}, "users:write", true},
		{"resource wildcard no match", []string{"roles:*"}, "users:write", false},
		{"empty scopes", []string{}, "users:read", false},
		{"nil scopes", nil, "users:read", false},
		{"empty action", []string{"users:read"}, "", false},
		{"multiple scopes second match", []string{"roles:read", "users:write"}, "users:write", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckScope(tt.scopes, tt.action)
			if result != tt.expected {
				t.Errorf("CheckScope(%v, %q) = %v, want %v", tt.scopes, tt.action, result, tt.expected)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	tb := NewTokenBucket()
	keyHash := "test-key-hash-for-rate-limiting"
	limit := 5 // 5 requests per hour

	// First 5 requests should be allowed (burst capacity)
	for i := 0; i < limit; i++ {
		if !tb.Allow(keyHash, limit) {
			t.Errorf("request %d should have been allowed", i+1)
		}
	}

	// Next request should be denied (bucket exhausted)
	if tb.Allow(keyHash, limit) {
		t.Error("request after exhausting limit should be denied")
	}

	// Another denied
	if tb.Allow(keyHash, limit) {
		t.Error("second request after exhausting limit should also be denied")
	}
}

func TestRateLimiterSeparateKeys(t *testing.T) {
	tb := NewTokenBucket()
	key1 := "key-hash-1"
	key2 := "key-hash-2"
	limit := 2

	// Exhaust key1
	tb.Allow(key1, limit)
	tb.Allow(key1, limit)

	// key1 should be denied
	if tb.Allow(key1, limit) {
		t.Error("key1 should be denied after exhausting limit")
	}

	// key2 should still be allowed (separate bucket)
	if !tb.Allow(key2, limit) {
		t.Error("key2 should be allowed (separate bucket)")
	}
}
