# agent_register.go

**Path:** `cmd/shark/cmd/agent_register.go`
**Package:** `cmd`
**LOC:** 86
**Tests:** none direct

## Purpose
Implements `shark agent register` — creates a new agent identity via `POST /api/v1/agents`, returning the client_id and one-time client_secret.

## Key types / functions
- `agentCmd` (var, line 15) — parent cobra command.
- `agentRegisterCmd` (var, line 20) — child:
  - Builds payload `{name, description?, metadata.app_slug?}`.
  - Posts via `adminDo`.
  - Maps 409 → `conflict` error code; prints `agent registered\n  id ...  client_id ...  client_secret ...`.
- Flags: `--app`, `--name` (required), `--description`, `--json`.

## Imports of note
- Uses `adminDo`, `apiError`, `extractData`, `maybeJSONErr` from sibling files.

## Wired by / used by
- Registered on `root` in `init()` at line 78.
- Backed by `internal/api/agent_handlers.go:handleCreateAgent`.

## Notes
- Lane E, milestone E5.
- Secret is shown only once.
