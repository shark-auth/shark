# `cmd/shark/cmd/agent_revoke_tokens.go`

## Purpose

Implements `shark agent revoke-tokens <id>` — revokes all active OAuth tokens for an agent. Use this to force re-authentication after a credential compromise or permission change without fully deleting the agent.

## Command shape

`shark agent revoke-tokens <id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `POST /api/v1/agents/{id}/tokens/revoke-all` — invalidate all active tokens for the agent

## Example

`shark agent revoke-tokens agt_abc123`
