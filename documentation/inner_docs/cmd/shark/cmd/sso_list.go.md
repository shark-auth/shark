# `cmd/shark/cmd/sso_list.go`

## Purpose

Implements `shark sso list` — retrieves all SSO connections and renders them in a table showing ID, name, type, domain, and enabled status. Useful for auditing configured identity providers.

## Command shape

`shark sso list [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/sso/connections` — list all SSO connections

## Example

`shark sso list`
