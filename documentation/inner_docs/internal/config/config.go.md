# config.go

**Path:** `internal/config/config.go`  
**Package:** `config`  
**LOC:** 493  
**Tests:** `config_test.go`

## Purpose
Koanf-based YAML + environment variable configuration schema. Defines all server config struct hierarchy with time-duration parsing and proxy legacy-field handling.

## Key types / functions
- `Config` (struct, line 18) — root config: Server, Storage, Auth, Passkeys, Email, SMTP, MFA, Social, SSO, OAuthServer, Proxy, Telemetry, etc.
- `ServerConfig` (struct, line 159) — port, secret, base_url, CORS origins, dev mode
- `StorageConfig` (struct, line 167) — SQLite path
- `EmailConfig` (struct, line 148) — provider (resend/smtp/shark), API key, from address
- `SMTPConfig` (struct) — host, port, username, password (legacy; reused for Resend API key)
- `ProxyConfig` (struct, line 65) — enabled, upstream, timeout, trusted headers, listeners (W15)
- `ProxyListenerConfig` (struct, line 78) — per-listener bind, upstream, session cookie domain, timeout
- `TelemetryConfig` (struct, line 54) — enabled, endpoint
- `ProxyConfig.Resolve()` (func, line 101) — synthesizes implicit listener from legacy fields if Listeners empty
- `Config.Save(path)` (func, line 38) — persists YAML to disk

## Imports of note
- `koanf/v2` — config loading library
- `koanf/parsers/yaml` — YAML unmarshaling
- `koanf/providers/file`, `env` — file and environment sources
- `gopkg.in/yaml.v3` — YAML encoding for Save

## Wired by
- `server.Build()` calls config.Load(opts.ConfigPath)
- Admin API reads/writes config via Config.Save() for dynamic reconfig

## Notes
- koanf tags control YAML/env binding: `koanf:"field_name" yaml:"field_name"`
- Environment variables interpolate via koanf providers
- Time-duration fields parsed via helper funcs (AccessTokenTTLDuration, etc.); defaults if invalid
- ProxyRule type deprecated v1.5 (rules moved to DB); retained for backward compat
- W15: ProxyListenerConfig.Bind="" means inherit main listener (legacy fallback)
- StripIncoming defaults to true if nil
- TimeoutDuration defaults to 30s if ≤0

