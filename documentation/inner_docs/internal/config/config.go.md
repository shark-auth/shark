# config.go

**Path:** `internal/config/config.go`  
**Package:** `config`  
**LOC:** 493  
**Tests:** `config_test.go`

## Purpose
Koanf-based environment-variable configuration schema. Defines the typed Config struct hierarchy with time-duration parsing and proxy legacy-field handling. Bootstrap-time env binding only — runtime config (session lifetimes, email provider, password rules, etc.) lives in SQLite `system_config` and is mutated via the admin API.

## Key types / functions
- `Config` (struct) — root config: Server, Storage, Auth, Passkeys, Email, SMTP, MFA, Social, SSO, OAuthServer, Proxy, Telemetry, etc.
- `ServerConfig` (struct) — port, secret, base_url, CORS origins
- `StorageConfig` (struct) — SQLite path
- `EmailConfig` (struct) — provider (resend/smtp/shark), API key, from address
- `SMTPConfig` (struct) — host, port, username, password
- `ProxyConfig` (struct) — enabled, upstream, timeout, trusted headers, listeners (W15)
- `ProxyListenerConfig` (struct) — per-listener bind, upstream, session cookie domain, timeout
- `TelemetryConfig` (struct) — enabled, endpoint
- `ProxyConfig.Resolve()` (func) — synthesizes implicit listener from legacy fields if Listeners empty

## Imports of note
- `koanf/v2` — config loading library
- `koanf/providers/env` — environment variable source only

## Wired by
- `server.Build()` calls `config.Load()`

## Notes
- **YAML file loading REMOVED in Phase H.** `Config.Save()` deleted; `koanf/parsers/yaml`, `koanf/providers/file`, `gopkg.in/yaml.v3` removed from go.mod.
- `yaml:` struct tags stripped; only `koanf:` tags remain for env binding (`SHARK_*`).
- Time-duration fields parsed via helper funcs (AccessTokenTTLDuration, etc.); defaults if invalid.
- ProxyRule type retained for in-memory use; YAML import handler (`handleImportYAMLRules`) deleted Phase H.
- W15: ProxyListenerConfig.Bind="" means inherit main listener (legacy fallback).
- StripIncoming defaults to true if nil; TimeoutDuration defaults to 30s if ≤0.

