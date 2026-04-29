package testutil

import (
	"github.com/shark-auth/shark/internal/config"
)

// TestConfig returns a Config with safe defaults suitable for testing.
// Uses reduced argon2id parameters, test secrets, and in-memory storage.
func TestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:    0, // random port
			Secret:  "test-secret-must-be-at-least-32-bytes-long!!",
			BaseURL: "http://localhost:8080",
		},
		Storage: config.StorageConfig{
			Path: ":memory:",
		},
		Auth: config.AuthConfig{
			SessionLifetime:   "24h",
			PasswordMinLength: 8,
			Argon2id: config.Argon2idConfig{
				Memory:      16384, // 16MB (reduced from 64MB for fast tests)
				Iterations:  1,     // 1 iteration (reduced from 3 for fast tests)
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   32,
			},
			JWT: config.JWTConfig{
				Enabled:         true,
				Mode:            "session",
				Audience:        "shark",
				AccessTokenTTL:  "15m",
				RefreshTokenTTL: "30d",
				ClockSkew:       "30s",
				Revocation: config.JWTRevocationConfig{
					CheckPerRequest: false,
				},
			},
		},
		Passkeys: config.PasskeyConfig{
			RPName:           "SharkAuth Test",
			RPID:             "localhost",
			Origin:           "http://localhost:8080",
			Attestation:      "none",
			ResidentKey:      "preferred",
			UserVerification: "preferred",
		},
		MagicLink: config.MagicLinkConfig{
			TokenLifetime: "10m",
			RedirectURL:   "http://localhost:3000/auth/callback",
		},
		SMTP: config.SMTPConfig{
			Host:     "localhost",
			Port:     1025,
			From:     "test@sharkauth.local",
			FromName: "SharkAuth Test",
		},
		MFA: config.MFAConfig{
			Issuer:        "SharkAuth Test",
			RecoveryCodes: 10,
		},
		APIKeys: config.APIKeysConfig{
			DefaultRateLimit: 1000,
			KeyMaxLifetime:   "365d",
		},
		Audit: config.AuditConfig{
			Retention:       "0",
			CleanupInterval: "1h",
		},
	}
}
