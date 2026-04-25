# app.go

**Path:** `cmd/shark/cmd/app.go`
**Package:** `cmd`
**LOC:** 14
**Tests:** app_test.go

## Purpose
Parent cobra command for all `shark app <subcommand>` operations.

## Key types / functions
- `appCmd` (var, line 6) — registered on `root` in init().

## Wired by / used by
- Children attached by `app_create.go`, `app_list.go`, `app_show.go`, `app_update.go`, `app_rotate.go`, `app_delete.go`.
