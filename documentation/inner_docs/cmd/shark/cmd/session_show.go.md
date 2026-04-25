# `cmd/shark/cmd/session_show.go`

## Purpose

Implements `shark session show <id>` — displays details of a single session including auth method, IP address, creation, and expiry. Internally queries the admin sessions list endpoint with a `session_id` filter since there is no dedicated single-session GET route.

## Command shape

`shark session show <id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/admin/sessions?limit=1&session_id={id}` — fetch the specific session via list filter

## Example

`shark session show sess_abc123`
