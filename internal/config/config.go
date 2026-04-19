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
	Email         EmailConfig         `koanf:"email"`
	MFA       MFAConfig       `koanf:"mfa"`
	Social    SocialConfig    `koanf:"social"`
	SSO       SSOConfig       `koanf:"sso"`
	APIKeys     APIKeysConfig     `koanf:"api_keys"`
	Audit       AuditConfig       `koanf:"audit"`
	OAuthServer OAuthServerConfig `koanf:"oauth_server"`
	Proxy       ProxyConfig       `koanf:"proxy"`
}

// ProxyConfig holds reverse-proxy settings consumed by internal/proxy. Koanf
// unmarshals directly into this struct; internal/proxy.Config is built from
// it at wiring time (Task P4) so this type stays free of non-serializable
// fields like time.Duration or *slog.Logger.
//
// Timeout is a plain int in seconds (not a duration string) so operators
// configuring via YAML or env vars don't have to remember a unit suffix
// for a value that will almost always be single-digit seconds.
//
// StripIncoming is a pointer so YAML can distinguish "unset" (nil, treated
// as the safe default of true) from an explicit `false`. Resolve applies
// that default — code reading cfg.Proxy.StripIncoming must either go
// through Resolve first or handle nil themselves.
type ProxyConfig struct {
	Enabled        bool        `koanf:"enabled"`
	Upstream       string      `koanf:"upstream"`
	Timeout        int         `koanf:"timeout_seconds"`
	TrustedHeaders []string    `koanf:"trusted_headers"`
	StripIncoming  *bool       `koanf:"strip_incoming"`
	Rules          []ProxyRule `koanf:"rules"`
}

// ProxyRule is a single route-level authorization rule consumed by the
// proxy's rules engine (internal/proxy). Kept in the config package (not
// in internal/proxy) so the YAML schema lives next to the rest of the
// user-facing configuration.
//
// Exactly one of Require or Allow must be set. Allow is a readability
// sugar that currently only accepts "anonymous"; it is equivalent to
// Require="anonymous" but reads more naturally in "allow public paths"
// scenarios. Scopes is an AND-constraint applied on top of Require.
type ProxyRule struct {
	Path    string   `koanf:"path"`
	Methods []string `koanf:"methods"`
	Require string   `koanf:"require"`
	Allow   string   `koanf:"allow"`
	Scopes  []string `koanf:"scopes"`
}

// TimeoutDuration returns the configured timeout as a time.Duration,
// falling back to 30s when the field is zero/unset.
func (p *ProxyConfig) TimeoutDuration() time.Duration {
	if p.Timeout <= 0 {
		return 30 * time.Second
	}
	return time.Duration(p.Timeout) * time.Second
}

// StripIncomingOrDefault returns the effective StripIncoming value,
// defaulting to true when the pointer is nil. Safe to call on a
// zero-valued ProxyConfig.
func (p *ProxyConfig) StripIncomingOrDefault() bool {
	if p.StripIncoming == nil {
		return true
	}
	return *p.StripIncoming
}

// EmailProvider enum. `dev` is auto-selected by --dev when unset.
const (
	EmailProviderShark  = "shark"
	EmailProviderResend = "resend"
	EmailProviderSMTP   = "smtp"
	EmailProviderDev    = "dev"
)

// EmailConfig holds provider-agnostic email settings. Introduced in phase 2;
// coexists with the legacy SMTPConfig which is kept as a deprecated alias so
// existing deployments don't break.
//
// Resolve() fills in the provider (and copies legacy smtp.* if email.* is empty)
// and is called once during Load.
type EmailConfig struct {
	Provider string `koanf:"provider"` // shark | resend | smtp | dev
	APIKey   string `koanf:"api_key"`
	From     string `koanf:"from"`
	FromName string `koanf:"from_name"`

	// SMTP-only fields (only read when Provider=="smtp").
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port        int      `koanf:"port"`
	Secret      string   `koanf:"secret"`
	BaseURL     string   `koanf:"base_url"`
	CORSOrigins []string `koanf:"cors_origins"`

	// DevMode is set at runtime via `shark serve --dev`. It enables the dev
	// inbox routes and relaxed defaults. Never loaded from YAML.
	DevMode bool `koanf:"-"`
}

// StorageConfig holds database settings.
type StorageConfig struct {
	Path string `koanf:"path"`
}

// JWTRevocationConfig holds JWT revocation settings.
type JWTRevocationConfig struct {
	CheckPerRequest bool `koanf:"check_per_request"`
}

// JWTConfig holds JWT issuance and validation settings.
// Duration fields are stored as strings and parsed via accessor methods,
// following the same pattern as SessionLifetime.
type JWTConfig struct {
	Enabled         bool                `koanf:"enabled"`
	Mode            string              `koanf:"mode"`
	Issuer          string              `koanf:"issuer"`
	Audience        string              `koanf:"audience"`
	AccessTokenTTL  string              `koanf:"access_token_ttl"`
	RefreshTokenTTL string              `koanf:"refresh_token_ttl"`
	ClockSkew       string              `koanf:"clock_skew"`
	Revocation      JWTRevocationConfig `koanf:"revocation"`
}

// AccessTokenTTLDuration parses the access token TTL string into a time.Duration.
func (j *JWTConfig) AccessTokenTTLDuration() time.Duration {
	return parseDuration(j.AccessTokenTTL, 15*time.Minute)
}

// RefreshTokenTTLDuration parses the refresh token TTL string into a time.Duration.
func (j *JWTConfig) RefreshTokenTTLDuration() time.Duration {
	return parseDuration(j.RefreshTokenTTL, 30*24*time.Hour)
}

// ClockSkewDuration parses the clock skew string into a time.Duration.
func (j *JWTConfig) ClockSkewDuration() time.Duration {
	return parseDuration(j.ClockSkew, 30*time.Second)
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	SessionLifetime   string         `koanf:"session_lifetime"`
	PasswordMinLength int            `koanf:"password_min_length"`
	Argon2id          Argon2idConfig `koanf:"argon2id"`
	JWT               JWTConfig      `koanf:"jwt"`
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
	// Deprecated: migrated to default application's allowed_callback_urls on first boot.
	// Removal target: Phase 6 (/oauth/authorize landing).
	RedirectURL string `koanf:"redirect_url"`
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
	// Deprecated: migrated to default application's allowed_callback_urls on first boot.
	// Removal target: Phase 6 (/oauth/authorize landing).
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

// OAuthServerConfig holds OAuth 2.1 Authorization Server settings.
type OAuthServerConfig struct {
	Enabled              bool   `koanf:"enabled"`
	Issuer               string `koanf:"issuer"`                  // defaults to server.base_url
	SigningAlgorithm     string `koanf:"signing_algorithm"`       // ES256 (default) | RS256
	AccessTokenLifetime  string `koanf:"access_token_lifetime"`   // default: 15m
	RefreshTokenLifetime string `koanf:"refresh_token_lifetime"`  // default: 30d
	AuthCodeLifetime     string `koanf:"auth_code_lifetime"`      // default: 60s
	DeviceCodeLifetime   string `koanf:"device_code_lifetime"`    // default: 15m
	ConsentTemplate      string `koanf:"consent_template"`        // path to custom template dir
	RequireDPoP          bool   `koanf:"require_dpop"`            // require DPoP for all clients
}

func (o *OAuthServerConfig) AccessTokenLifetimeDuration() time.Duration {
	return parseDuration(o.AccessTokenLifetime, 15*time.Minute)
}

func (o *OAuthServerConfig) RefreshTokenLifetimeDuration() time.Duration {
	return parseDuration(o.RefreshTokenLifetime, 30*24*time.Hour)
}

func (o *OAuthServerConfig) AuthCodeLifetimeDuration() time.Duration {
	return parseDuration(o.AuthCodeLifetime, 60*time.Second)
}

func (o *OAuthServerConfig) DeviceCodeLifetimeDuration() time.Duration {
	return parseDuration(o.DeviceCodeLifetime, 15*time.Minute)
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
			k.Set(key, replaced) //#nosec G104 -- koanf.Set only errors on invalid keys; key came from k.All() and is known-valid
		}
	}
}

// Load reads configuration from a YAML file and applies environment variable overrides.
// Environment variables use the prefix SHARKAUTH_ and replace dots with underscores.
// For example, server.port becomes SHARKAUTH_SERVER_PORT.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	// Set defaults
	//#nosec G101 -- default config values (ports, URLs, durations, issuer names), no secrets
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
		// JWT defaults
		"auth.jwt.enabled":                   true,
		"auth.jwt.mode":                      "session",
		"auth.jwt.audience":                  "shark",
		"auth.jwt.access_token_ttl":          "15m",
		"auth.jwt.refresh_token_ttl":         "30d",
		"auth.jwt.clock_skew":                "30s",
		"auth.jwt.revocation.check_per_request": false,
		// OAuth server defaults
		"oauth_server.enabled":                true,
		"oauth_server.signing_algorithm":      "ES256",
		"oauth_server.access_token_lifetime":  "15m",
		"oauth_server.refresh_token_lifetime": "30d",
		"oauth_server.auth_code_lifetime":     "60s",
		"oauth_server.device_code_lifetime":   "15m",
		"oauth_server.require_dpop":           false,
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

	cfg.Email.Resolve(&cfg.SMTP)

	return &cfg, nil
}

// Resolve fills in missing email fields from the legacy smtp: block so existing
// deployments keep working without changes. Picks a provider when absent:
//   - smtp.host == "smtp.resend.com" -> provider=resend
//   - smtp.host set                  -> provider=smtp
//   - everything else                -> left empty (startup validator refuses)
//
// Callers can then switch on cfg.Email.Provider.
func (e *EmailConfig) Resolve(legacy *SMTPConfig) {
	if e.Provider == "" {
		switch {
		case legacy != nil && legacy.Host == "smtp.resend.com":
			e.Provider = EmailProviderResend
			if e.APIKey == "" {
				e.APIKey = legacy.Password
			}
		case legacy != nil && legacy.Host != "":
			e.Provider = EmailProviderSMTP
			if e.Host == "" {
				e.Host = legacy.Host
			}
			if e.Port == 0 {
				e.Port = legacy.Port
			}
			if e.Username == "" {
				e.Username = legacy.Username
			}
			if e.Password == "" {
				e.Password = legacy.Password
			}
		}
	}
	if legacy != nil {
		if e.From == "" {
			e.From = legacy.From
		}
		if e.FromName == "" {
			e.FromName = legacy.FromName
		}
	}
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
