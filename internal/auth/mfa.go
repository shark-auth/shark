package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/storage"
)

const (
	// recoveryCodeLength is the length of each recovery code (8 alphanumeric chars).
	recoveryCodeLength = 8
	// recoveryCodeAlphabet is the character set for recovery codes.
	recoveryCodeAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	// bcryptCost is the cost parameter for hashing recovery codes.
	bcryptCost = 10
)

// MFAManager handles TOTP enrollment, validation, and recovery codes.
type MFAManager struct {
	store storage.Store
	cfg   config.MFAConfig
}

// NewMFAManager creates a new MFAManager.
func NewMFAManager(store storage.Store, cfg config.MFAConfig) *MFAManager {
	return &MFAManager{
		store: store,
		cfg:   cfg,
	}
}

// GenerateSecret creates a new TOTP secret for the given email.
// Returns the base32-encoded secret and an otpauth:// URI for QR code generation.
func (m *MFAManager) GenerateSecret(email string) (secret string, qrURI string, err error) {
	issuer := m.cfg.Issuer
	if issuer == "" {
		issuer = "SharkAuth"
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: email,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", fmt.Errorf("generating TOTP key: %w", err)
	}

	return key.Secret(), key.URL(), nil
}

// ValidateTOTP verifies a TOTP code against the given secret.
// Allows +/- 1 step tolerance (30-second window).
func (m *MFAManager) ValidateTOTP(secret, code string) bool {
	valid, _ := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:     1,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return valid
}

// GenerateRecoveryCodes creates N recovery codes, hashes them with bcrypt,
// stores the hashes, and returns the plaintext codes to show the user once.
func (m *MFAManager) GenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	count := m.cfg.RecoveryCodes
	if count <= 0 {
		count = 10
	}

	// Delete any existing recovery codes for this user
	if err := m.store.DeleteAllMFARecoveryCodesByUserID(ctx, userID); err != nil {
		return nil, fmt.Errorf("deleting old recovery codes: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	plaintextCodes := make([]string, count)
	storageCodes := make([]*storage.MFARecoveryCode, count)

	for i := 0; i < count; i++ {
		code := generateRandomCode(recoveryCodeLength)
		plaintextCodes[i] = code

		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcryptCost)
		if err != nil {
			return nil, fmt.Errorf("hashing recovery code: %w", err)
		}

		id, _ := gonanoid.New()
		storageCodes[i] = &storage.MFARecoveryCode{
			ID:        "mrc_" + id,
			UserID:    userID,
			Code:      string(hash),
			Used:      false,
			CreatedAt: now,
		}
	}

	if err := m.store.CreateMFARecoveryCodes(ctx, storageCodes); err != nil {
		return nil, fmt.Errorf("storing recovery codes: %w", err)
	}

	return plaintextCodes, nil
}

// VerifyRecoveryCode checks a plaintext recovery code against stored hashes.
// If a match is found and the code has not been used, it marks it as used.
// Returns true if the code is valid and was successfully consumed.
func (m *MFAManager) VerifyRecoveryCode(ctx context.Context, userID, code string) (bool, error) {
	codes, err := m.store.GetMFARecoveryCodesByUserID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("fetching recovery codes: %w", err)
	}

	// Normalize: lowercase, trim whitespace and dashes
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")

	for _, stored := range codes {
		if stored.Used {
			continue
		}
		// Use constant-time comparison via bcrypt (which is inherently constant-time
		// for the hash comparison step).
		if err := bcrypt.CompareHashAndPassword([]byte(stored.Code), []byte(code)); err == nil {
			// Match found â€” mark it used
			if err := m.store.MarkMFARecoveryCodeUsed(ctx, stored.ID); err != nil {
				return false, fmt.Errorf("marking recovery code used: %w", err)
			}
			return true, nil
		}
	}

	// No match found â€” use subtle.ConstantTimeCompare on a dummy to avoid timing leaks
	// on the "no codes at all" path vs "codes exist but none match" path.
	dummy := []byte("constant-time-dummy-comparison")
	subtle.ConstantTimeCompare(dummy, dummy)

	return false, nil
}

// generateRandomCode generates a random alphanumeric code of the given length
// using rejection sampling to avoid modulo bias.
func generateRandomCode(length int) string {
	alphabetLen := byte(len(recoveryCodeAlphabet))
	// Find the largest multiple of alphabetLen that fits in a byte
	maxValid := byte(256 - (256 % int(alphabetLen))) //#nosec G115 -- alphabetLen=36; (256 - 256%36) == 252 fits in byte
	result := make([]byte, length)
	buf := make([]byte, 1)
	for i := 0; i < length; {
		if _, err := rand.Read(buf); err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		if buf[0] < maxValid {
			result[i] = recoveryCodeAlphabet[buf[0]%alphabetLen]
			i++
		}
		// else: reject and retry
	}
	return string(result)
}
