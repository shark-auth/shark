# `cmd/shark/cmd/api_key_revoke.go`

## Purpose

Implements `shark api-key revoke <id>` — permanently deletes an API key, immediately invalidating any requests that use it. This action is irreversible.

## Command shape

`shark api-key revoke <id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `DELETE /api/v1/api-keys/{id}` — delete and invalidate the key

## Example

`shark api-key revoke key_abc123`
