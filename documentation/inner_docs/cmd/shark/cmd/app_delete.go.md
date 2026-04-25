# app_delete.go

**Path:** `cmd/shark/cmd/app_delete.go`
**Package:** `cmd`
**LOC:** 75
**Tests:** app_test.go

## Purpose
Implements `shark app delete <id-or-client-id>` — confirms then removes the application, refusing to delete the default app.

## Key types / functions
- `appDeleteCmd` (var, line 18):
  - Resolves with `lookupApp`.
  - Refuses with exit 1 if `app.IsDefault`.
  - Interactive y/N prompt unless `--yes`.
  - Calls `store.DeleteApplication`.

## Imports of note
- `bufio`, `os`
- `internal/config`, `internal/storage`

## Wired by / used by
- Attached to `appCmd` in `init()` line 71.

## Notes
- Confirmation accepts only `y` or `yes` (case-insensitive).
