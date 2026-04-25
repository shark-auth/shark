# db.go

**Path:** `internal/testutil/db.go`
**Package:** `testutil`
**LOC:** 33
**Tests:** none (test helper).

## Purpose
Spins up an in-memory SQLite store with the embedded test migration set applied, registers cleanup, and returns it. Test helper — **not for production runtime.**

## Functions
- `NewTestDB(t)` (line 15) — opens `:memory:` `*storage.SQLiteStore`, runs migrations from the embedded `migrations/*.sql`, registers `t.Cleanup` to close.

## Embedded FS
- `//go:embed migrations/*.sql` (line 10) — local copy of all 25 goose migrations so tests don't depend on `cmd/shark/migrations`.

## Imports of note
- `embed`, `testing`
- `internal/storage`

## Used by
- Every storage-layer `*_test.go` in `internal/storage`
- `internal/api`, `internal/auth`, `internal/oauth` test suites

## Notes
- Test helper — **not for production runtime.**
- The embedded migration set must be kept in lockstep with `cmd/shark/migrations/` (any new migration goes in both directories).
- `:memory:` plus `MaxOpenConns=1` (set by `NewSQLiteStore`) ensures all goroutines share the same DB.
