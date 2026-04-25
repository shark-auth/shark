# `cmd/shark/cmd/agent_delete.go`

## Purpose

Implements `shark agent delete <id>` — deactivates (soft-deletes) an agent by ID. Prompts for confirmation unless `--yes` is passed, guarding against accidental deletion in scripts.

## Command shape

`shark agent delete <id> [--yes] [--json]`

## Flags

- `--yes` — skip the interactive confirmation prompt
- `--json` — emit raw JSON output

## API endpoint(s) called

- `DELETE /api/v1/agents/{id}` — deactivate the agent

## Example

`shark agent delete agt_abc123 --yes`
