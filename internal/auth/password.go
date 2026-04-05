package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"

	"github.com/sharkauth/sharkauth/internal/config"
)

var (
	ErrInvalidHash         = errors.New("invalid encoded hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

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

	keyLength := uint32(len(hash))
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
