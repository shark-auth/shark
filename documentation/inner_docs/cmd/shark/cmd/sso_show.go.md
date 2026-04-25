# `cmd/shark/cmd/sso_show.go`

## Purpose

Implements `shark sso show <id>` — fetches a single SSO connection and displays its id, name, type, domain, enabled flag, and creation date. Returns a structured 404 error if the connection does not exist.

## Command shape

`shark sso show <id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/sso/connections/{id}` — retrieve a single SSO connection

## Example

`shark sso show sso_abc123`
