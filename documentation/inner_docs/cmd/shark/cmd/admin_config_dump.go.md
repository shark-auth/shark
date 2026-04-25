# `cmd/shark/cmd/admin_config_dump.go`

## Purpose

Implements `shark admin config dump` — fetches and pretty-prints the live admin configuration. Always emits JSON because the config payload is inherently structured. Useful for auditing runtime settings without touching the database directly.

## Command shape

`shark admin config dump [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/admin/config` — retrieve the full admin configuration object

## Example

`shark admin config dump`
