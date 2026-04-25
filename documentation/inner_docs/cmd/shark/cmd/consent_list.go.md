# `cmd/shark/cmd/consent_list.go`

## Purpose

Implements `shark consents list` — retrieves OAuth consents and renders them in a table with ID, user, client, scopes, and grant date. Optionally filters to a single user via `--user`, which appends a `user_id` query parameter.

## Command shape

`shark consents list [--user <id>] [--json]`

## Flags

- `--user`, `-u` — filter results to a specific user ID
- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/admin/oauth/consents` — list all consents (optionally filtered by `?user_id=`)

## Example

`shark consents list --user usr_abc123`
