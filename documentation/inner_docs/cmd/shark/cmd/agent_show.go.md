# agent_show.go + agent_update.go + agent_delete.go + agent_rotate_secret.go + agent_revoke_tokens.go

**Path:** `cmd/shark/cmd/agent_*.go`
**Package:** `cmd`

## Purpose
Extends the existing `agentCmd` (from `agent_register.go`) with show/update/delete/rotate-secret/revoke-tokens subcommands.

## Subcommands (new)
- `shark agent show <id>` — GET /api/v1/agents/{id}
- `shark agent update <id> [--name --description]` — PATCH /api/v1/agents/{id}
- `shark agent delete <id> [--yes]` — DELETE /api/v1/agents/{id}
- `shark agent rotate-secret <id>` — POST /api/v1/agents/{id}/rotate-secret
- `shark agent revoke-tokens <id>` — POST /api/v1/agents/{id}/tokens/revoke-all
