# migrations.go

**Path:** `internal/storage/migrations.go`
**Package:** `storage`
**LOC:** 26
**Tests:** none direct; covered transitively via `testutil.NewTestDB` and `cmd/shark/cmd` integration tests.

## Purpose
Single function `RunMigrations(db, migrationsFS, dirPath)` that runs goose migrations against the supplied `*sql.DB` using an `embed.FS` source. Decouples this package from any specific migrations directory so callers (production server, CLI, test harness) embed their own copy.

## Function
- `RunMigrations` (line 14) — calls `goose.SetBaseFS(migrationsFS)`, `goose.SetDialect("sqlite3")`, then `goose.Up(db, dirPath)`.

## Imports of note
- `database/sql`
- `embed`
- `github.com/pressly/goose/v3`

## Used by
- `internal/server/build.go` (production boot, with `cmd/shark/migrations.FS`)
- `internal/testutil/db.go` (in-memory tests, with `testutil/migrations` embed)
- `internal/testutil/cli/harness.go` (CLI E2E harness)
- Various per-package `testmigrations` setups (oauth, etc.)

## Migrations on disk
Canonical migration set lives at `cmd/shark/migrations/` — 26 SQL files (goose up/down format). Names from `00001_init.sql` through `00025_branding_design_tokens.sql`. The same set is mirrored under `internal/testutil/migrations/` so test binaries don't depend on the cmd package.

## Notes
- Dialect is hard-coded to `sqlite3` — migrating to Postgres would require parameterizing.
- No down/rollback exposed; goose down is invoked manually if needed.
- Errors are wrapped with `fmt.Errorf` for a useful upstream stack.
