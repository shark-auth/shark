# `cmd/shark/cmd/api_key_rotate.go`

## Purpose

Implements `shark api-key rotate <id>` — replaces an existing API key's secret with a new one, invalidating the old value immediately. The new key value is printed once and is not retrievable afterward.

## Command shape

`shark api-key rotate <id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `POST /api/v1/api-keys/{id}/rotate` — generate a new key secret

## Example

`shark api-key rotate key_abc123`
