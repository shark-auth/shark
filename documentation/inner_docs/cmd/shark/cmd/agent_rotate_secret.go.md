# `cmd/shark/cmd/agent_rotate_secret.go`

## Purpose

Implements `shark agent rotate-secret <id>` — generates a new client secret for an agent and invalidates the old one. The new secret is printed once to stdout; it cannot be retrieved again.

## Command shape

`shark agent rotate-secret <id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `POST /api/v1/agents/{id}/rotate-secret` — issue a new client secret

## Example

`shark agent rotate-secret agt_abc123`
