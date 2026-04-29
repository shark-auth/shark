# dev.go

**Path:** `cmd/shark/cmd/dev.go`
**Package:** `cmd`
**LOC:** 43
**Tests:** none

## Purpose
Helper that mutates `server.Options` to enable developer mode for `shark serve --dev`: writes to a local `dev.db`, auto-generates a 32-byte hex secret, optionally wipes the DB.

## Key types / functions
- `applyDevMode` (func, line 15) â€” sets `DevMode`, optionally removes `dev.db{,-wal,-shm}` when `reset` is true, generates `SecretOverride` and `StoragePathOverride`.
- `randomHex` (func, line 37) â€” `crypto/rand` â†’ hex.

## Imports of note
- `crypto/rand`, `encoding/hex`
- `github.com/shark-auth/shark/internal/server`

## Wired by / used by
- Called from `serve.go` when `--dev` is passed.

## Notes
- Treats missing dev.db files as success; only non-NotExist errors propagate.
