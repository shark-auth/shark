# `cmd/shark/cmd/session_list.go`

## Purpose

Implements `shark session list` — lists active sessions with user email, IP address, creation date, and expiry. Supports filtering by user and controlling the result count with `--limit`.

## Command shape

`shark session list [--user <id>] [--limit <n>] [--json]`

## Flags

- `--user` — filter by user ID
- `--limit` — maximum number of sessions to return (default 50)
- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/admin/sessions` — list sessions (with `?limit=` and optional `&user_id=`)

## Example

`shark session list --user usr_abc123 --limit 10`
