# `cmd/shark/cmd/api_key_create.go`

## Purpose

Implements `shark api-key create` — creates a named admin API key. `--name` is required. The raw key value is printed once on creation and is not retrievable afterward; treat it like a password.

## Command shape

`shark api-key create --name <name> [--json]`

## Flags

- `--name` — human-readable label for the key (required)
- `--json` — emit raw JSON output

## API endpoint(s) called

- `POST /api/v1/api-keys` — create a new API key

## Example

`shark api-key create --name "CI pipeline"`
