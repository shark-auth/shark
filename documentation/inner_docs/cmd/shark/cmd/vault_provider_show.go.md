# `cmd/shark/cmd/vault_provider_show.go`

## Purpose

Implements `shark vault provider show <name-or-id>` — fetches a vault provider by ID or name and displays its id, name, type, and creation date. Returns a structured 404 error if the provider does not exist.

## Command shape

`shark vault provider show <name-or-id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/vault/providers/{id}` — retrieve a single vault provider

## Example

`shark vault provider show default`
