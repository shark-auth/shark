package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const fieldEncryptionPrefix = "enc::"

// FieldEncryptor provides AES-256-GCM encryption for sensitive database fields.
// It derives a dedicated key from the server secret to isolate field encryption
// from session encryption.
type FieldEncryptor struct {
	gcm cipher.AEAD
}

// NewFieldEncryptor creates a FieldEncryptor from the server secret.
// It derives a separate 256-bit key using SHA-256(secret + "field-encryption").
func NewFieldEncryptor(serverSecret string) (*FieldEncryptor, error) {
	if len(serverSecret) < 32 {
		return nil, errors.New("server secret must be at least 32 characters")
	}

	// Derive a dedicated key for field encryption
	h := sha256.New()
	h.Write([]byte(serverSecret))
	h.Write([]byte("field-encryption"))
	key := h.Sum(nil) // 32 bytes = AES-256

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	return &FieldEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts a plaintext string and returns a prefixed base64 string.
// Returns empty string for empty input.
func (fe *FieldEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, fe.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := fe.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return fieldEncryptionPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a prefixed encrypted string back to plaintext.
// If the value doesn't have the encryption prefix, it's returned as-is
// (supports transparent migration of existing unencrypted data).
func (fe *FieldEncryptor) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	// If not encrypted, return as-is (migration path for existing data)
	if !strings.HasPrefix(encrypted, fieldEncryptionPrefix) {
		return encrypted, nil
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, fieldEncryptionPrefix))
	if err != nil {
		return "", fmt.Errorf("decoding base64: %w", err)
	}

	nonceSize := fe.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := fe.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted returns true if the value has the encryption prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, fieldEncryptionPrefix)
}
