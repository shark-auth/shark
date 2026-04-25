# app_show.go

**Path:** `cmd/shark/cmd/app_show.go`
**Package:** `cmd`
**LOC:** 89
**Tests:** app_test.go

## Purpose
Implements `shark app show <id-or-client-id>` — looks up by ID then by client_id, prints all fields.

## Key types / functions
- `appShowCmd` (var, line 16) — cobra command, exactly one positional arg.
- `lookupApp` (func, line 67) — try `GetApplicationByID`, fall back to `GetApplicationByClientID`; reused by app_show, app_update, app_delete, app_rotate.

## Imports of note
- `database/sql` (for `sql.ErrNoRows`)
- `internal/config`, `internal/storage`

## Wired by / used by
- Attached to `appCmd` in `init()` line 85.
- `lookupApp` is the canonical lookup helper for the entire `app` subcommand tree.

## Notes
- Pretty-prints URL slices via `json.MarshalIndent` for readability.
