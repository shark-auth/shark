# server.go

**Path:** `internal/server/server.go`  
**Package:** `server`  
**LOC:** 649  
**Tests:** `server_test.go`

## Purpose
Main server lifecycle: wires config, storage, migrations, HTTP, OAuth, webhook, telemetry into a runnable Bootstrap struct. Entry point for cobra subcommands (serve, tests).

## Key types / functions
- `Options` (struct) — configuration for server assembly: MigrationsFS, MigrationsDir, NoPrompt, ProxyUpstream. **ConfigPath removed Phase H.**
- `Bootstrap` (struct) — result of Build(); holds Config, Store, API, Dispatcher, AdminKey
- `Build(ctx, opts)` (func) — loads config (env vars only), opens DB, runs migrations, wires API, persists admin key
- `Run(ctx, opts)` (func) — starts HTTP listener; handles first-boot prompt and telemetry setup

## Imports of note
- `embed` — migrations loaded from filesystem
- `os/exec` — opens browser on first boot
- `internal/api` — wires API server (NewServer takes 2 args — ConfigPath removed)
- `internal/config` — Config.Load() (env vars only, no YAML)
- `internal/email` — email Sender selection
- `internal/oauth` — OAuth AS wiring
- `internal/proxy` — reverse proxy mount
- `internal/telemetry` — ping loop startup
- `internal/webhook` — Dispatcher startup

## Wired by
- `cmd/serve.go` — cobra command calls Build/Run
- Integration tests call Build directly to set up test fixtures

## Notes
- **ConfigPath field REMOVED in Phase H.** No YAML config file loaded at any point.
- **yamlHasLegacyProxyRules() DELETED in Phase H.** No YAML scan on startup.
- First boot detected by absence of admin key in DB; interactive prompt opens browser, prints magic-link sign-in URL.
- `--no-prompt` suppresses browser-open for CI/headless.
- Admin key auto-generated on first boot if missing; printed to stdout once.
- ProxyUpstream overrides cfg.Proxy.Upstream at bootstrap time.
- ProxyListeners holds W15 multi-listener set; legacy mode uses catch-all in api.Server.

