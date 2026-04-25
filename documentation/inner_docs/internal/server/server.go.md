# server.go

**Path:** `internal/server/server.go`  
**Package:** `server`  
**LOC:** 649  
**Tests:** `server_test.go`

## Purpose
Main server lifecycle: wires config, storage, migrations, HTTP, OAuth, webhook, telemetry into a runnable Bootstrap struct. Entry point for cobra subcommands (serve, dev-mode, tests).

## Key types / functions
- `Options` (struct, line 38) — configuration for server assembly
- `Bootstrap` (struct, line 74) — result of Build(); holds Config, Store, API, Dispatcher, AdminKey
- `Build(ctx, opts)` (func, line 101) — loads config, opens DB, runs migrations, wires API, persists admin key
- `yamlHasLegacyProxyRules(path)` (func) — v1.5 deprecation warning scanner
- `Run(ctx, opts)` (func) — starts HTTP listener; handles first-boot prompt and telemetry setup

## Imports of note
- `embed` — migrations loaded from filesystem
- `os/exec` — opens browser on first boot
- `internal/api` — wires API server
- `internal/config` — Config.Load()
- `internal/email` — email Sender selection
- `internal/oauth` — OAuth AS wiring
- `internal/proxy` — reverse proxy mount
- `internal/telemetry` — ping loop startup
- `internal/webhook` — Dispatcher startup

## Wired by
- `cmd/serve.go`, `cmd/dev.go` — cobra commands call Build/Run
- Integration tests call Build directly to set up test fixtures

## Notes
- DevMode enables in-db email capture (DevInboxSender), relaxed CORS, auto-secret, /admin/dev/* routes
- --proxy-upstream flag overrides cfg.Proxy at bootstrap time
- Admin key auto-generated on first boot if missing; printed to stdout once
- ProxyListeners holds W15 multi-listener set; legacy mode uses catch-all in api.Server
- Minimum YAML config (phase 2): server.base_url + (email.provider OR smtp.*)
- First-boot prompt suppressed with --no-prompt for CI/headless environments

