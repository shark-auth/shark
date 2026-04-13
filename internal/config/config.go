package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds all SharkAuth configuration.
type Config struct {
	Server    ServerConfig    `koanf:"server"`
	Storage   StorageConfig   `koanf:"storage"`
	Auth          AuthConfig          `koanf:"auth"`
	Passkeys      PasskeyConfig       `koanf:"passkeys"`
	MagicLink     MagicLinkConfig     `koanf:"magic_link"`
	PasswordReset PasswordResetConfig `koanf:"password_reset"`
	SMTP          SMTPConfig          `koanf:"smtp"`
	MFA       MFAConfig       `koanf:"mfa"`
	Social    SocialConfig    `koanf:"social"`
	SSO       SSOConfig       `koanf:"sso"`
	APIKeys   APIKeysConfig   `koanf:"api_keys"`
	Audit     AuditConfig     `koanf:"audit"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port        int      `koanf:"port"`
	Secret      string   `koanf:"secret"`
	BaseURL     string   `koanf:"base_url"`
	CORSOrigins []string `koanf:"cors_origins"`
}

// StorageConfig holds database settings.
type StorageConfig struct {
	Path string `koanf:"path"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	SessionLifetime   string        `koanf:"session_lifetime"`
	PasswordMinLength int           `koanf:"password_min_length"`
	Argon2id          Argon2idConfig `koanf:"argon2id"`
}

// Argon2idConfig holds Argon2id password hashing parameters.
type Argon2idConfig struct {
	Memory      uint32 `koanf:"memory"`      // memory in KiB (default 65536 = 64MB)
	Iterations  uint32 `koanf:"iterations"`   // number of iterations (default 3)
	Parallelism uint8  `koanf:"parallelism"`  // parallelism factor (default 2)
	SaltLength  uint32 `koanf:"salt_length"`  // salt length in bytes (default 16)
	KeyLength   uint32 `koanf:"key_length"`   // key length in bytes (default 32)
}

// SessionLifetimeDuration parses the session lifetime string into a time.Duration.
func (a *AuthConfig) SessionLifetimeDuration() time.Duration {
	return parseDuration(a.SessionLifetime, 30*24*time.Hour)
}

// PasskeyConfig holds WebAuthn/passkey settings.
type PasskeyConfig struct {
	RPName           string `koanf:"rp_name"`
	RPID             string `koanf:"rp_id"`
	Origin           string `koanf:"origin"`
	Attestation      string `koanf:"attestation"`
	ResidentKey      string `koanf:"resident_key"`
	UserVerification string `koanf:"user_verification"`
}

// MagicLinkConfig holds magic link settings.
type MagicLinkConfig struct {
	TokenLifetime string `koanf:"token_lifetime"`
	RedirectURL   string `koanf:"redirect_url"`
}

// TokenLifetimeDuration parses the token lifetime string into a time.Duration.
func (m *MagicLinkConfig) TokenLifetimeDuration() time.Duration {
	return parseDuration(m.TokenLifetime, 10*time.Minute)
}

// PasswordResetConfig holds password reset settings.
type PasswordResetConfig struct {
	RedirectURL string `koanf:"redirect_url"`
}

// SMTPConfig holds email sending settings.
type SMTPConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	From     string `koanf:"from"`
	FromName string `koanf:"from_name"`
}

// MFAConfig holds TOTP/MFA settings.
type MFAConfig struct {
	Issuer        string `koanf:"issuer"`
	RecoveryCodes int    `koanf:"recovery_codes"`
}

// SocialConfig holds OAuth provider settings.
type SocialConfig struct {
	RedirectURL string        `koanf:"redirect_url"` // Post-OAuth redirect to frontend
	Google      GoogleConfig  `koanf:"google"`
	GitHub      GitHubConfig  `koanf:"github"`
	Apple       AppleConfig   `koanf:"apple"`
	Discord     DiscordConfig `koanf:"discord"`
}

// GoogleConfig holds Google OAuth settings.
type GoogleConfig struct {
	ClientID     string   `koanf:"client_id"`
	ClientSecret string   `koanf:"client_secret"`
	Scopes       []string `koanf:"scopes"` // Optional: override default scopes
}

// GitHubConfig holds GitHub OAuth settings.
type GitHubConfig struct {
	ClientID     string   `koanf:"client_id"`
	ClientSecret string   `koanf:"client_secret"`
	Scopes       []string `koanf:"scopes"` // Optional: override default scopes
}

// AppleConfig holds Apple OAuth settings.
type AppleConfig struct {
	ClientID       string `koanf:"client_id"`
	TeamID         string `koanf:"team_id"`
	KeyID          string `koanf:"key_id"`
	PrivateKeyPath string `koanf:"private_key_path"`
}

// DiscordConfig holds Discord OAuth settings.
type DiscordConfig struct {
	ClientID     string   `koanf:"client_id"`
	ClientSecret string   `koanf:"client_secret"`
	Scopes       []string `koanf:"scopes"` // Optional: override default scopes
}

// SSOConfig holds SSO settings.
type SSOConfig struct {
	SAML SAMLConfig `koanf:"saml"`
	OIDC OIDCConfig `koanf:"oidc"`
}

// SAMLConfig holds SAML service provider settings.
type SAMLConfig struct {
	SPEntityID string `koanf:"sp_entity_id"`
}

// OIDCConfig is a placeholder for OIDC settings configured per-connection via API.
type OIDCConfig struct{}

// APIKeysConfig holds M2M API key settings.
type APIKeysConfig struct {
	DefaultRateLimit int    `koanf:"default_rate_limit"`
	KeyMaxLifetime   string `koanf:"key_max_lifetime"`
}

// AuditConfig holds audit log settings.
type AuditConfig struct {
	Retention       string `koanf:"retention"`
	CleanupInterval string `koanf:"cleanup_interval"`
}

// CleanupIntervalDuration parses the cleanup interval string into a time.Duration.
func (a *AuditConfig) CleanupIntervalDuration() time.Duration {
	return parseDuration(a.CleanupInterval, 1*time.Hour)
}

// envVarPattern matches ${VAR_NAME} patterns in config values.
var envVarPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// interpolateEnvVars walks all koanf keys and replaces ${VAR} patterns
// with actual environment variable values.
func interpolateEnvVars(k *koanf.Koanf) {
	for _, key := range k.Keys() {
		val := k.String(key)
		if val == "" {
			continue
		}
		replaced := envVarPattern.ReplaceAllStringFunc(val, func(match string) string {
			varName := envVarPattern.FindStringSubmatch(match)[1]
			if envVal, ok := os.LookupEnv(varName); ok {
				return envVal
			}
			return match // leave unresolved if env var not set
		})
		if replaced != val {
			k.Set(key, replaced)
		}
	}
}

// Load reads configuration from a YAML file and applies environment variable overrides.
// Environment variables use the prefix SHARKAUTH_ and replace dots with underscores.
// For example, server.port becomes SHARKAUTH_SERVER_PORT.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	// Set defaults
	defaults := map[string]interface{}{
		"server.port":              8080,
		"server.base_url":         "http://localhost:8080",
		"storage.path":            "./data/sharkauth.db",
		"auth.session_lifetime":    "30d",
		"auth.password_min_length": 8,
		"auth.argon2id.memory":      65536,
		"auth.argon2id.iterations":  3,
		"auth.argon2id.parallelism": 2,
		"auth.argon2id.salt_length": 16,
		"auth.argon2id.key_length":  32,
		"passkeys.rp_name":         "SharkAuth",
		"passkeys.attestation":     "none",
		"passkeys.resident_key":    "preferred",
		"passkeys.user_verification": "preferred",
		"magic_link.token_lifetime": "10m",
		"magic_link.redirect_url":      "http://localhost:3000/auth/callback",
		"password_reset.redirect_url":  "http://localhost:3000/auth/reset-password",
		"smtp.port":               587,
		"smtp.from_name":          "SharkAuth",
		"mfa.issuer":              "SharkAuth",
		"mfa.recovery_codes":      10,
		"api_keys.default_rate_limit": 1000,
		"api_keys.key_max_lifetime":   "365d",
		"audit.retention":         "0",
		"audit.cleanup_interval":  "1h",
	}
	for key, val := range defaults {
		if err := k.Set(key, val); err != nil {
			return nil, fmt.Errorf("setting default %s: %w", key, err)
		}
	}

	// Load YAML file if it exists
	if path != "" {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file %s: %w", path, err)
		}
	}

	// Interpolate ${VAR_NAME} patterns in YAML values with actual env vars
	interpolateEnvVars(k)

	// Load environment variable overrides with SHARKAUTH_ prefix.
	// Nesting uses double-underscore: SHARKAUTH_SMTP__FROM_NAME -> smtp.from_name
	// Single underscores are preserved as literal underscores in key names.
	if err := k.Load(env.Provider("SHARKAUTH_", ".", func(s string) string {
		key := strings.TrimPrefix(s, "SHARKAUTH_")
		key = strings.ToLower(key)
		// Double underscore is the nesting separator
		key = strings.ReplaceAll(key, "__", ".")
		return key
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// parseDuration parses a duration string that supports "d" suffix for days,
// in addition to standard Go duration strings.
func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" || s == "0" {
		return fallback
	}
	// Handle "Nd" format (days)
	if strings.HasSuffix(s, "d") {
		trimmed := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(trimmed, "%d", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	// Try standard Go duration
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
