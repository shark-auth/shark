# serve.go

**Path:** `cmd/shark/cmd/serve.go`
**Package:** `cmd`
**LOC:** 43 (W17 Phase H)
**Tests:** none direct (covered by E2E harness `internal/testutil/cli`)

## Purpose
Implements `shark serve` â€” boots the SharkAuth HTTP server using the embedded migrations FS. No config file needed: all runtime config lives in SQLite and is mutated via the admin API or Settings dashboard.

## Key types / functions
- `serveCmd` (var, line 18) â€” cobra command with `RunE`.
  - Sets up SIGINT/SIGTERM signal context.
  - Builds `server.Options{MigrationsFS, MigrationsDir, NoPrompt}` and calls `server.Serve`.
- Flags (line 39): `--proxy-upstream`, `--no-prompt`.

## Imports of note
- `os/signal`, `syscall`
- `github.com/shark-auth/shark/internal/server`

## Wired by / used by
- Registered in `cmd/shark/cmd/root.go`.
- `--no-prompt` forwarded to `server.Options.NoPrompt` â€” skips first-boot browser-open prompt for CI/headless.
- `--proxy-upstream` mounts reverse proxy to the given upstream URL at bootstrap.

## Notes
- **`--config` flag REMOVED in Phase H.** Source of truth = SQLite.
- **`--dev` and `--reset` flags REMOVED in Phase D/H.**
- First boot detected automatically by `server.Build`; interactive prompt opens browser and prints magic-link admin sign-in URL.
- `--no-prompt` is the production/container mode (equivalent to old `--dev` for headless, but without ephemeral DB).
