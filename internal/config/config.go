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
	yamlv3 "gopkg.in/yaml.v3"
)

// Config holds all SharkAuth configuration.
type Config struct {
	Server        ServerConfig        `koanf:"server" yaml:"server"`
	Storage       StorageConfig       `koanf:"storage" yaml:"storage"`
	Auth          AuthConfig          `koanf:"auth" yaml:"auth"`
	Passkeys      PasskeyConfig       `koanf:"passkeys" yaml:"passkeys"`
	MagicLink     MagicLinkConfig     `koanf:"magic_link" yaml:"magic_link"`
	PasswordReset PasswordResetConfig `koanf:"password_reset" yaml:"password_reset"`
	SMTP          SMTPConfig          `koanf:"smtp" yaml:"smtp"`
	Email         EmailConfig         `koanf:"email" yaml:"email"`
	MFA           MFAConfig           `koanf:"mfa" yaml:"mfa"`
	Social        SocialConfig        `koanf:"social" yaml:"social"`
	SSO           SSOConfig           `koanf:"sso" yaml:"sso"`
	APIKeys       APIKeysConfig       `koanf:"api_keys" yaml:"api_keys"`
	Audit         AuditConfig         `koanf:"audit" yaml:"audit"`
	OAuthServer   OAuthServerConfig   `koanf:"oauth_server" yaml:"oauth_server"`
	Proxy         ProxyConfig         `koanf:"proxy" yaml:"proxy"`
	Telemetry     TelemetryConfig     `koanf:"telemetry" yaml:"telemetry"`
}

// Save persists the current configuration to the specified YAML file path.
func (c *Config) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("config: create file: %w", err)
	}
	defer f.Close()

	enc := yamlv3.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("config: encode yaml: %w", err)
	}
	return nil
}

// TelemetryConfig holds anonymous install-ping settings.
type TelemetryConfig struct {
	Enabled  bool   `koanf:"enabled" yaml:"enabled"`
	Endpoint string `koanf:"endpoint" yaml:"endpoint"`
}

// ProxyConfig holds reverse-proxy settings consumed by internal/proxy.
//
// Deprecated: the `rules` YAML field was removed in v1.5. Proxy rules
// now live in the DB and are managed via the Admin API
// (`/api/v1/admin/proxy/rules`). The legacy `rules:` YAML block is
// ignored on load; a warning is emitted at server startup when it is
// still present on disk. See docs/proxy_v1_5/migration/yaml_deprecation.md.
type ProxyConfig struct {
	Enabled        bool                  `koanf:"enabled" yaml:"enabled"`
	Upstream       string                `koanf:"upstream" yaml:"upstream"`
	Timeout        int                   `koanf:"timeout_seconds" yaml:"timeout_seconds"`
	TrustedHeaders []string              `koanf:"trusted_headers" yaml:"trusted_headers"`
	StripIncoming  *bool                 `koanf:"strip_incoming" yaml:"strip_incoming"`
	Listeners      []ProxyListenerConfig `koanf:"listeners" yaml:"listeners"`
}

// ProxyListenerConfig is one reverse-proxy listener in the W15 multi-listener design.
//
// The legacy `rules:` sub-field was removed in v1.5; per-listener rules
// are now sourced from the DB via the shared proxy engine.
type ProxyListenerConfig struct {
	Bind                string   `koanf:"bind" yaml:"bind"`
	Upstream            string   `koanf:"upstream" yaml:"upstream"`
	SessionCookieDomain string   `koanf:"session_cookie_domain" yaml:"session_cookie_domain"`
	TrustedHeaders      []string `koanf:"trusted_headers" yaml:"trusted_headers"`
	StripIncoming       *bool    `koanf:"strip_incoming" yaml:"strip_incoming"`
	Timeout             int      `koanf:"timeout_seconds" yaml:"timeout_seconds"`
}

func (l *ProxyListenerConfig) TimeoutDuration() time.Duration {
	if l.Timeout <= 0 {
		return 30 * time.Second
	}
	return time.Duration(l.Timeout) * time.Second
}

func (l *ProxyListenerConfig) StripIncomingOrDefault() bool {
	if l.StripIncoming == nil {
		return true
	}
	return *l.StripIncoming
}

func (p *ProxyConfig) Resolve() {
	if len(p.Listeners) > 0 {
		return
	}
	if !p.Enabled || p.Upstream == "" {
		return
	}
	p.Listeners = []ProxyListenerConfig{{
		Bind:           "",
		Upstream:       p.Upstream,
		TrustedHeaders: p.TrustedHeaders,
		StripIncoming:  p.StripIncoming,
		Timeout:        p.Timeout,
	}}
}

// ProxyRule is the historical rule shape that used to be loaded from the
// `proxy.rules:` YAML section.
//
// Deprecated: v1.5 moved proxy rules into the DB (table `proxy_rules`,
// managed via `/api/v1/admin/proxy/rules`). This struct is retained for
// backward compatibility with callers that still import the type name
// (e.g. the legacy YAML import handler) but is no longer populated by
// config.Load. Any new code should use `storage.ProxyRule` or
// `proxy.RuleSpec` directly.
type ProxyRule struct {
	Path    string   `koanf:"path" yaml:"path"`
	Methods []string `koanf:"methods" yaml:"methods"`
	Require string   `koanf:"require" yaml:"require"`
	Allow   string   `koanf:"allow" yaml:"allow"`
	Scopes  []string `koanf:"scopes" yaml:"scopes"`
}

func (p *ProxyConfig) TimeoutDuration() time.Duration {
	if p.Timeout <= 0 {
		return 30 * time.Second
	}
	return time.Duration(p.Timeout) * time.Second
}

func (p *ProxyConfig) StripIncomingOrDefault() bool {
	if p.StripIncoming == nil {
		return true
	}
	return *p.StripIncoming
}

type EmailConfig struct {
	Provider string `koanf:"provider" yaml:"provider"`
	APIKey   string `koanf:"api_key" yaml:"api_key"`
	From     string `koanf:"from" yaml:"from"`
	FromName string `koanf:"from_name" yaml:"from_name"`
	Host     string `koanf:"host" yaml:"host"`
	Port     int    `koanf:"port" yaml:"port"`
	Username string `koanf:"username" yaml:"username"`
	Password string `koanf:"password" yaml:"password"`
}

type ServerConfig struct {
	Port        int      `koanf:"port" yaml:"port"`
	Secret      string   `koanf:"secret" yaml:"secret"`
	BaseURL     string   `koanf:"base_url" yaml:"base_url"`
	CORSOrigins []string `koanf:"cors_origins" yaml:"cors_origins"`
	DevMode     bool     `koanf:"-" yaml:"-"`
}

type StorageConfig struct {
	Path string `koanf:"path" yaml:"path"`
}

type JWTRevocationConfig struct {
	CheckPerRequest bool `koanf:"check_per_request" yaml:"check_per_request"`
}

type JWTConfig struct {
	Enabled         bool                `koanf:"enabled" yaml:"enabled"`
	Mode            string              `koanf:"mode" yaml:"mode"`
	Issuer          string              `koanf:"issuer" yaml:"issuer"`
	Audience        string              `koanf:"audience" yaml:"audience"`
	AccessTokenTTL  string              `koanf:"access_token_ttl" yaml:"access_token_ttl"`
	RefreshTokenTTL string              `koanf:"refresh_token_ttl" yaml:"refresh_token_ttl"`
	ClockSkew       string              `koanf:"clock_skew" yaml:"clock_skew"`
	Revocation      JWTRevocationConfig `koanf:"revocation" yaml:"revocation"`
}

func (j *JWTConfig) AccessTokenTTLDuration() time.Duration {
	return parseDuration(j.AccessTokenTTL, 15*time.Minute)
}

func (j *JWTConfig) RefreshTokenTTLDuration() time.Duration {
	return parseDuration(j.RefreshTokenTTL, 30*24*time.Hour)
}

func (j *JWTConfig) ClockSkewDuration() time.Duration {
	return parseDuration(j.ClockSkew, 30*time.Second)
}

type AuthConfig struct {
	SessionLifetime   string         `koanf:"session_lifetime" yaml:"session_lifetime"`
	PasswordMinLength int            `koanf:"password_min_length" yaml:"password_min_length"`
	Argon2id          Argon2idConfig `koanf:"argon2id" yaml:"argon2id"`
	JWT               JWTConfig      `koanf:"jwt" yaml:"jwt"`
}

type Argon2idConfig struct {
	Memory      uint32 `koanf:"memory" yaml:"memory"`
	Iterations  uint32 `koanf:"iterations" yaml:"iterations"`
	Parallelism uint8  `koanf:"parallelism" yaml:"parallelism"`
	SaltLength  uint32 `koanf:"salt_length" yaml:"salt_length"`
	KeyLength   uint32 `koanf:"key_length" yaml:"key_length"`
}

func (a *AuthConfig) SessionLifetimeDuration() time.Duration {
	return parseDuration(a.SessionLifetime, 30*24*time.Hour)
}

type PasskeyConfig struct {
	RPName           string `koanf:"rp_name" yaml:"rp_name"`
	RPID             string `koanf:"rp_id" yaml:"rp_id"`
	Origin           string `koanf:"origin" yaml:"origin"`
	Attestation      string `koanf:"attestation" yaml:"attestation"`
	ResidentKey      string `koanf:"resident_key" yaml:"resident_key"`
	UserVerification string `koanf:"user_verification" yaml:"user_verification"`
}

type MagicLinkConfig struct {
	TokenLifetime string `koanf:"token_lifetime" yaml:"token_lifetime"`
	RedirectURL   string `koanf:"redirect_url" yaml:"redirect_url"`
}

func (m *MagicLinkConfig) TokenLifetimeDuration() time.Duration {
	return parseDuration(m.TokenLifetime, 10*time.Minute)
}

type PasswordResetConfig struct {
	RedirectURL   string `koanf:"redirect_url" yaml:"redirect_url"`
	TokenLifetime string `koanf:"token_lifetime" yaml:"token_lifetime"`
}

type SMTPConfig struct {
	Host     string `koanf:"host" yaml:"host"`
	Port     int    `koanf:"port" yaml:"port"`
	Username string `koanf:"username" yaml:"username"`
	Password string `koanf:"password" yaml:"password"`
	From     string `koanf:"from" yaml:"from"`
	FromName string `koanf:"from_name" yaml:"from_name"`
}

type MFAConfig struct {
	Issuer        string `koanf:"issuer" yaml:"issuer"`
	RecoveryCodes int    `koanf:"recovery_codes" yaml:"recovery_codes"`
}

type SocialConfig struct {
	RedirectURL string        `koanf:"redirect_url" yaml:"redirect_url"`
	Google      GoogleConfig  `koanf:"google" yaml:"google"`
	GitHub      GitHubConfig  `koanf:"github" yaml:"github"`
	Apple       AppleConfig   `koanf:"apple" yaml:"apple"`
	Discord     DiscordConfig `koanf:"discord" yaml:"discord"`
}

type GoogleConfig struct {
	ClientID     string   `koanf:"client_id" yaml:"client_id"`
	ClientSecret string   `koanf:"client_secret" yaml:"client_secret"`
	Scopes       []string `koanf:"scopes" yaml:"scopes"`
}

type GitHubConfig struct {
	ClientID     string   `koanf:"client_id" yaml:"client_id"`
	ClientSecret string   `koanf:"client_secret" yaml:"client_secret"`
	Scopes       []string `koanf:"scopes" yaml:"scopes"`
}

type AppleConfig struct {
	ClientID       string `koanf:"client_id" yaml:"client_id"`
	TeamID         string `koanf:"team_id" yaml:"team_id"`
	KeyID          string `koanf:"key_id" yaml:"key_id"`
	PrivateKeyPath string `koanf:"private_key_path" yaml:"private_key_path"`
}

type DiscordConfig struct {
	ClientID     string   `koanf:"client_id" yaml:"client_id"`
	ClientSecret string   `koanf:"client_secret" yaml:"client_secret"`
	Scopes       []string `koanf:"scopes" yaml:"scopes"`
}

type SSOConfig struct {
	SAML SAMLConfig `koanf:"saml" yaml:"saml"`
	OIDC OIDCConfig `koanf:"oidc" yaml:"oidc"`
}

type SAMLConfig struct {
	SPEntityID string `koanf:"sp_entity_id" yaml:"sp_entity_id"`
}

type OIDCConfig struct{}

type APIKeysConfig struct {
	DefaultRateLimit int    `koanf:"default_rate_limit" yaml:"default_rate_limit"`
	KeyMaxLifetime   string `koanf:"key_max_lifetime" yaml:"key_max_lifetime"`
}

type AuditConfig struct {
	Retention       string `koanf:"retention" yaml:"retention"`
	CleanupInterval string `koanf:"cleanup_interval" yaml:"cleanup_interval"`
}

func (a *AuditConfig) CleanupIntervalDuration() time.Duration {
	return parseDuration(a.CleanupInterval, 1*time.Hour)
}

type OAuthServerConfig struct {
	Enabled              bool   `koanf:"enabled" yaml:"enabled"`
	Issuer               string `koanf:"issuer" yaml:"issuer"`
	SigningAlgorithm     string `koanf:"signing_algorithm" yaml:"signing_algorithm"`
	AccessTokenLifetime  string `koanf:"access_token_lifetime" yaml:"access_token_lifetime"`
	RefreshTokenLifetime string `koanf:"refresh_token_lifetime" yaml:"refresh_token_lifetime"`
	AuthCodeLifetime     string `koanf:"auth_code_lifetime" yaml:"auth_code_lifetime"`
	DeviceCodeLifetime   string `koanf:"device_code_lifetime" yaml:"device_code_lifetime"`
	ConsentTemplate      string `koanf:"consent_template" yaml:"consent_template"`
	RequireDPoP          bool   `koanf:"require_dpop" yaml:"require_dpop"`
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

var envVarPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

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
			return match
		})
		if replaced != val {
			k.Set(key, replaced)
		}
	}
}

func Load(path string) (*Config, error) {
	k := koanf.New(".")
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
		"password_reset.token_lifetime": "30m",
		"password_reset.redirect_url":  "http://localhost:3000/auth/reset-password",
		"smtp.port":               587,
		"smtp.from_name":          "SharkAuth",
		"mfa.issuer":              "SharkAuth",
		"mfa.recovery_codes":      10,
		"api_keys.default_rate_limit": 1000,
		"api_keys.key_max_lifetime":   "365d",
		"audit.retention":         "0",
		"audit.cleanup_interval":  "1h",
		"auth.jwt.enabled":                   true,
		"auth.jwt.mode":                      "session",
		"auth.jwt.audience":                  "shark",
		"auth.jwt.access_token_ttl":          "15m",
		"auth.jwt.refresh_token_ttl":         "30d",
		"auth.jwt.clock_skew":                "30s",
		"auth.jwt.revocation.check_per_request": false,
		"telemetry.enabled":  true,
		"telemetry.endpoint": "https://telemetry.shark-auth.com/v1/ping",
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
	if path != "" {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file %s: %w", path, err)
		}
	}
	interpolateEnvVars(k)
	if err := k.Load(env.Provider("SHARKAUTH_", ".", func(s string) string {
		key := strings.TrimPrefix(s, "SHARKAUTH_")
		key = strings.ToLower(key)
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
	cfg.Proxy.Resolve()
	return &cfg, nil
}

// EmailProvider enum. `dev` is auto-selected by --dev when unset.
const (
	EmailProviderShark  = "shark"
	EmailProviderResend = "resend"
	EmailProviderSMTP   = "smtp"
	EmailProviderDev    = "dev"
)

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

func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" || s == "0" {
		return fallback
	}
	if strings.HasSuffix(s, "d") {
		trimmed := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(trimmed, "%d", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
