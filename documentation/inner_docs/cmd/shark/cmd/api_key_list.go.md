# `cmd/shark/cmd/api_key_list.go`

## Purpose

Implements `shark api-key list` — retrieves all admin API keys and renders them in a tabular view showing ID, name, key prefix, and creation date. The raw key value is never returned by the list endpoint.

## Command shape

`shark api-key list [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/api-keys` — list all API keys

## Example

`shark api-key list`
