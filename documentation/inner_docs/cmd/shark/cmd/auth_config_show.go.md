# `cmd/shark/cmd/auth_config_show.go`

## Purpose

Implements `shark auth config show` — fetches and displays the current authentication configuration. In human mode, key-value pairs are printed; with `--json` the raw response is emitted. Also registered directly as `authConfigShowCmd` for internal aliasing.

## Command shape

`shark auth config show [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/admin/config` — retrieve the full auth configuration

## Example

`shark auth config show --json`
