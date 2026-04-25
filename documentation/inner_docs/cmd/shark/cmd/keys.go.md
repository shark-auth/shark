# keys.go

**Path:** `cmd/shark/cmd/keys.go`
**Package:** `cmd`
**LOC:** 100
**Tests:** none direct

## Purpose
Implements `shark keys generate-jwt` — generates and stores an RS256 JWT signing keypair, with optional rotation of existing active keys.

## Key types / functions
- `keysCmd` (var, line 16) — parent cobra command.
- `keysGenerateJWTCmd` (var, line 21) — child command:
  - Loads YAML config, opens SQLite store, runs migrations.
  - Calls `jwtmgr.NewManager(...).GenerateAndStore(ctx, rotate)`.
  - Prints kid/algorithm/status (or JSON via `--json`).
- Flags: `--rotate`, `--config`, `--json`.

## Imports of note
- `internal/auth/jwt` (alias `jwtmgr`)
- `internal/config`, `internal/storage`

## Wired by / used by
- Registered in `cmd/shark/cmd/root.go:96`.

## Notes
- Without `--rotate`: insert fails if an active key already exists.
- With `--rotate`: retires all active keys, both old and new remain in JWKS until rotated_at + 2*access_token_ttl elapses.
