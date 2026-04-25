# `cmd/shark/cmd/agent_update.go`

## Purpose

Implements `shark agent update <id>` — patches mutable fields (`name`, `description`) on an existing agent. At least one flag must be provided; supplying none returns an error.

## Command shape

`shark agent update <id> [--name <name>] [--description <desc>] [--json]`

## Flags

- `--name` — new display name for the agent
- `--description` — new description
- `--json` — emit raw JSON output

## API endpoint(s) called

- `PATCH /api/v1/agents/{id}` — apply partial update to the agent

## Example

`shark agent update agt_abc123 --name "Ingestion Bot"`
