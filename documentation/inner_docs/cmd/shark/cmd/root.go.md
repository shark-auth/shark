# root.go

**Path:** `cmd/shark/cmd/root.go`
**Package:** `cmd`
**LOC:** 97
**Tests:** none (covered indirectly by other `*_test.go` in the package)

## Purpose
Defines the top-level cobra `root` command, JSON output helpers, embedded migrations injection point, and persistent verbose-logging flag.

## Key types / functions
- `SetMigrations` (func, line 18) — main injects the embed.FS at startup.
- `Execute` (func, line 39) — runs `root.Execute()`.
- `root` (var, line 25) — cobra base command, `SilenceUsage`/`SilenceErrors` true.
- `configureLogger` (func, line 44) — slog text handler at INFO/DEBUG.
- `jsonFlag`, `writeJSON`, `writeJSONError`, `addJSONFlag` (lines 54-87) — shared `--json` plumbing.
- `init` (line 90) — registers persistent `--verbose/-v`, plus `serveCmd`, `initCmd`, `healthCmd`, `versionCmd`, `keysCmd`.

## Imports of note
- `github.com/spf13/cobra`
- `embed`, `log/slog`

## Wired by / used by
- Called from `cmd/shark/main.go`.
- Other subcommand files attach to `root` from their own `init()` blocks.

## Notes
- `migrationsFS` is package-level state set exactly once before `Execute`.
- Persistent `--url`/`--token` flags are added by `adminapi.go`'s init for admin-API subcommands.
