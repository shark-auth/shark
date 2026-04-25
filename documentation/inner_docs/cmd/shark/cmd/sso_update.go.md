# `cmd/shark/cmd/sso_update.go`

## Purpose

Implements `shark sso update <id>` — patches mutable fields (`name`, `domain`, `enabled`) on an existing SSO connection via PUT. At least one flag must be changed; supplying none returns an error.

## Command shape

`shark sso update <id> [--name <name>] [--domain <domain>] [--enabled <bool>] [--json]`

## Flags

- `--name` — new display name for the connection
- `--domain` — new email domain for auto-routing
- `--enabled` — enable (`true`) or disable (`false`) the connection
- `--json` — emit raw JSON output

## API endpoint(s) called

- `PUT /api/v1/sso/connections/{id}` — replace mutable fields on the connection

## Example

`shark sso update sso_abc123 --enabled=false`
