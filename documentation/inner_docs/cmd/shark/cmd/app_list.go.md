# app_list.go

**Path:** `cmd/shark/cmd/app_list.go`
**Package:** `cmd`
**LOC:** 107
**Tests:** app_test.go, json_output_test.go

## Purpose
Implements `shark app list` — opens the SQLite store and prints up to 100 applications via tabwriter (or JSON).

## Key types / functions
- `appListCmd` (var, line 15) — cobra command.
- `summarizeURLs` (func, line 66) — collapses long URL lists to "N urls".
- `appToJSON` (func, line 80) — canonical JSON shape for an Application; reused by app_create.go, app_show.go.
- `maybeJSONErr` (func, line 96) — package-wide helper that emits structured JSON errors when `--json` is set; used by most subcommands.

## Imports of note
- `text/tabwriter`
- `internal/config`, `internal/storage`

## Wired by / used by
- Attached to `appCmd` in `init()` line 103.
- `appToJSON` and `maybeJSONErr` are imported (by package) by every other app/admin subcommand.

## Notes
- Pagination is hard-coded to limit=100, offset=0.
