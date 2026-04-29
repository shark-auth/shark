# main.go

**Path:** `cmd/shark/main.go`
**Package:** `main`
**LOC:** 22
**Tests:** none

## Purpose
Process entry point for the `shark` binary; embeds SQL migrations and dispatches to the cobra command tree.

## Key types / functions
- `main` (func, line 15) â€” wires migrationsFS into the cmd package via `cmd.SetMigrations`, calls `cmd.Execute`, logs and exits 1 on error.
- `migrationsFS` (var, line 13) â€” `embed.FS` bound by `//go:embed migrations/*.sql`.

## Imports of note
- `embed` â€” for SQL migration embedding.
- `log/slog` â€” structured error log on failure.
- `github.com/shark-auth/shark/cmd/shark/cmd` â€” the cobra root.

## Wired by / used by
- Compiled by `go build ./cmd/shark`; also embedded SQL is re-exported via `cmd/shark/migrations/fs.go` for non-main consumers (test harness).

## Notes
- Errors print "Error: ..." to stderr and exit code 1 â€” the cobra root has `SilenceErrors: true` so the duplicate prefix is intentional here.
