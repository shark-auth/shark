package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"

	"github.com/shark-auth/shark/internal/config"
)

var (
	ErrInvalidHash         = errors.New("invalid encoded hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

// Hasher manages password hashing and verification with a concurrency limit
// to prevent CPU saturation (and scheduler starvation) during high-concurrency
// login/signup bursts. It also maintains a short-lived cache of successful
// verifications to skip Argon2id during multi-step auth flows (e.g., MFA).
type Hasher struct {
	sem chan struct{}

	// Short-lived cache for repeat password verifications
	cacheMu sync.RWMutex
	cache   map[string]time.Time
}

// NewHasher creates a new Hasher with a semaphore limited to the number of
// CPU cores (or the provided limit if > 0).
func NewHasher(limit int) *Hasher {
	if limit <= 0 {
		limit = runtime.NumCPU()
	}
	h := &Hasher{
		sem:   make(chan struct{}, limit),
		cache: make(map[string]time.Time),
	}
	go h.cleanupLoop()
	return h
}

func (h *Hasher) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	for range ticker.C {
		h.cacheMu.Lock()
		now := time.Now()
		for k, v := range h.cache {
			if now.After(v) {
				delete(h.cache, k)
			}
		}
		h.cacheMu.Unlock()
	}
}

// cacheKey generates a fast, deterministic key for the CryptoCache.
func cacheKey(password, encodedHash string) string {
	sum := sha256.Sum256([]byte(password + ":" + encodedHash))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

// Hash hashes a password using Argon2id, respecting the concurrency limit.
func (h *Hasher) Hash(password string, cfg config.Argon2idConfig) (string, error) {
	h.sem <- struct{}{}
	defer func() { <-h.sem }()
	return HashPassword(password, cfg)
}

// Verify checks a password against a hash, respecting the concurrency limit
// and leveraging a 60-second success cache.
func (h *Hasher) Verify(password, encodedHash string) (bool, error) {
	ck := cacheKey(password, encodedHash)

	// Check cache (fast path)
	h.cacheMu.RLock()
	expiry, found := h.cache[ck]
	h.cacheMu.RUnlock()
	if found && time.Now().Before(expiry) {
		return true, nil
	}

	// Wait for CPU capacity
	h.sem <- struct{}{}
	match, err := VerifyPassword(password, encodedHash)
	<-h.sem

	// On success, cache for 60 seconds
	if match && err == nil {
		h.cacheMu.Lock()
		h.cache[ck] = time.Now().Add(60 * time.Second)
		h.cacheMu.Unlock()
	}

	return match, err
}

// HashPassword hashes a password using Argon2id with the given config params.
// Returns the encoded hash string: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
func HashPassword(password string, cfg config.Argon2idConfig) (string, error) {
	salt := make([]byte, cfg.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, cfg.Iterations, cfg.Memory, cfg.Parallelism, cfg.KeyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, cfg.Memory, cfg.Iterations, cfg.Parallelism, b64Salt, b64Hash)

	return encoded, nil
}

// VerifyPassword checks a password against an encoded hash.
// Supports both $argon2id$ and $2a$/$2b$ (bcrypt) formats for Auth0 migration.
func VerifyPassword(password, encodedHash string) (bool, error) {
	if isBcryptHash(encodedHash) {
		err := bcrypt.CompareHashAndPassword([]byte(encodedHash), []byte(password))
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}

	// Parse argon2id hash
	memory, iterations, parallelism, salt, hash, err := parseArgon2idHash(encodedHash)
	if err != nil {
		return false, err
	}

	keyLength := uint32(len(hash)) //#nosec G115 -- hash is a fixed-size argon2id output (â‰¤64 bytes), always fits in uint32
	otherHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

// NeedsRehash returns true if the hash is not argon2id (e.g., bcrypt from Auth0 import).
func NeedsRehash(encodedHash string) bool {
	return !strings.HasPrefix(encodedHash, "$argon2id$")
}

// isBcryptHash returns true if the hash looks like a bcrypt hash.
func isBcryptHash(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$")
}

// parseArgon2idHash extracts the parameters, salt, and hash from an encoded argon2id string.
func parseArgon2idHash(encodedHash string) (memory, iterations uint32, parallelism uint8, salt, hash []byte, err error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return 0, 0, 0, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return 0, 0, 0, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return 0, 0, 0, nil, nil, ErrIncompatibleVersion
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return 0, 0, 0, nil, nil, ErrInvalidHash
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("decoding salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("decoding hash: %w", err)
	}

	return memory, iterations, parallelism, salt, hash, nil
}
