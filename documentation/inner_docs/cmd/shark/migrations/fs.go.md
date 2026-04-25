# fs.go

**Path:** `cmd/shark/migrations/fs.go`
**Package:** `migrations`
**LOC:** 25
**Tests:** none

## Purpose
Re-exports the canonical SQL migrations directory as an `embed.FS` so non-main packages (notably `internal/testutil/cli`) can use the exact same migration set as `shark serve`.

## Key types / functions
- `FS` (var, line 25) — `embed.FS` bound by `//go:embed *.sql`.

## Imports of note
- `embed`

## Wired by / used by
- Imported by `internal/testutil/cli` for E2E test harness.
- Mirrors the embed in `cmd/shark/main.go` — kept intentionally in sync.

## Notes
- Existed as a deliberate fix for the v1.5 Lane D drift bug where a separate `testmigrations/` directory stopped at 00012, causing `TestE2EServeFlow` to fail with "no such column: proxy_public_domain".
