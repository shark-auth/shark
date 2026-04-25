# `cmd/shark/cmd/session_revoke.go`

## Purpose

Implements `shark session revoke <id>` — immediately terminates a session by ID, forcing the user to re-authenticate. Useful for incident response or compliance-driven logouts.

## Command shape

`shark session revoke <id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `DELETE /api/v1/admin/sessions/{id}` — terminate the session

## Example

`shark session revoke sess_abc123`
