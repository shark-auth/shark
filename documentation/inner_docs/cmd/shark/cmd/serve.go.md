# serve.go

**Path:** `cmd/shark/cmd/serve.go`
**Package:** `cmd`
**LOC:** 64
**Tests:** none direct (covered by E2E harness `internal/testutil/cli`)

## Purpose
Implements `shark serve` — boots the SharkAuth HTTP server using the embedded migrations FS and the loaded YAML config.

## Key types / functions
- `serveCmd` (var, line 22) — cobra command with `RunE`.
  - Sets up SIGINT/SIGTERM signal context.
  - In `--dev`, tolerates a missing config file by clearing `ConfigPath`.
  - Builds `server.Options{ConfigPath, MigrationsFS, MigrationsDir, NoPrompt}` and calls `server.Serve`.
- Flags (line 58): `--config`, `--dev`, `--reset`, `--proxy-upstream`, `--no-prompt`.

## Imports of note
- `os/signal`, `syscall`
- `github.com/sharkauth/sharkauth/internal/server`

## Wired by / used by
- Registered in `cmd/shark/cmd/root.go:92`.
- `--dev` delegates to `applyDevMode` (dev.go).
- `--no-prompt` is forwarded to `server.Options.NoPrompt` (H7 first-boot prompt).
- `--proxy-upstream` is the legacy direct upstream URL (proxy v1.5 prefers DB-backed rules).

## Notes
- Default config path is `sharkauth.yaml`.
