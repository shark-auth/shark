# `cmd/shark/cmd/sso_delete.go`

## Purpose

Implements `shark sso delete <id>` — permanently removes an SSO connection. Prompts for confirmation unless `--yes` is passed, preventing accidental deletion of production identity providers.

## Command shape

`shark sso delete <id> [--yes] [--json]`

## Flags

- `--yes` — skip the interactive confirmation prompt
- `--json` — emit raw JSON output

## API endpoint(s) called

- `DELETE /api/v1/sso/connections/{id}` — delete the SSO connection

## Example

`shark sso delete sso_abc123 --yes`
