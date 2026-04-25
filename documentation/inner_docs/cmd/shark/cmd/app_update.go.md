# app_update.go

**Path:** `cmd/shark/cmd/app_update.go`
**Package:** `cmd`
**LOC:** 132
**Tests:** app_test.go

## Purpose
Implements `shark app update <id-or-client-id>` — mutates name and adds/removes URLs from callbacks, logouts, origins lists.

## Key types / functions
- `appUpdateCmd` (var, line 23):
  - Validates added URLs via `validateCLIURLs`.
  - Resolves via `lookupApp`.
  - Mutates the app via `applyURLMutations` per list.
  - Calls `store.UpdateApplication` then re-fetches for output.
- `applyURLMutations` (func, line 89) — removes then adds, deduplicates, returns non-nil empty slice when result would be nil.

## Imports of note
- `internal/config`, `internal/storage`

## Wired by / used by
- Attached to `appCmd` in `init()` line 122.

## Notes
- Flags come in add/remove pairs: `--add-callback` / `--remove-callback`, etc.
- Removal happens before addition, so removing then re-adding the same URL preserves it.
