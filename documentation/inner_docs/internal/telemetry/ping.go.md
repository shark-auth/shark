# ping.go

**Path:** `internal/telemetry/ping.go`  
**Package:** `telemetry`  
**LOC:** 173  
**Tests:** `ping_test.go`

## Purpose
Anonymous install-count telemetry. Optional outbound ping loop (24h cadence) that sends system/version info to tracking endpoint. Opt-out via config or environment variable.

## Key types / functions
- `Config` (struct, line 42) — Enabled, Endpoint, Version, InstallIDPath
- `Payload` (struct, line 102) — InstallID, Version, OS, Arch, UptimeS, StartedAt, GoVersion (wire format)
- `StartPingLoop(ctx, cfg, logger)` (func, line 67) — spawns ping goroutine; no-op if disabled
- `sendPing(ctx, client, cfg, installID, startedAt)` (func, line 112) — HTTP POST to endpoint
- `loadOrCreateInstallID(path)` (func, line 149) — read UUID from file, or generate + persist

## Imports of note
- `bytes`, `encoding/json` — request marshaling
- `net/http` — HTTP client
- `os`, `path/filepath` — file I/O for install_id
- `runtime` — GOOS, GOARCH, Version()
- `google/uuid` — v4 UUID generation

## Wired by
- `server.Run()` calls StartPingLoop after HTTP bind
- HTTP client: http.DefaultClient; 5s timeout per ping

## Notes
- Install ID: UUID v4, generated on first boot, persisted to data/install_id
- First ping: ~30s after boot; subsequent pings: every 24h
- What we send: install_id, version, OS, arch, uptime, started_at, Go version
- What we do NOT send: user counts, app counts, hostnames, IPs, secrets, emails
- Opt out: telemetry.enabled: false in YAML or SHARKAUTH_TELEMETRY__ENABLED=false
- Endpoint does not log client IPs server-side
- HTTP >=300 status treated as error; goroutine logs warning, continues (non-fatal)
- No retry loop; next ping attempt is 24h later

