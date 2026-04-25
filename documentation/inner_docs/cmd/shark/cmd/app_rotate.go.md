# app_rotate.go

**Path:** `cmd/shark/cmd/app_rotate.go`
**Package:** `cmd`
**LOC:** 80
**Tests:** app_test.go

## Purpose
Implements `shark app rotate-secret <id-or-client-id>` — generates a new client_secret and updates the row; old secret is immediately invalid.

## Key types / functions
- `appRotateCmd` (var, line 14):
  - Resolves via `lookupApp` (app_show.go).
  - Calls `generateCLISecret` (app_create.go) for the new secret + hash + prefix.
  - Calls `store.RotateApplicationSecret`.
  - Prints rotated_at timestamp; supports `--json`.

## Imports of note
- `internal/config`, `internal/storage`

## Wired by / used by
- Attached to `appCmd` in `init()` line 76.

## Notes
- Secret shown exactly once; emphasises old secret invalidation in human-readable output.
