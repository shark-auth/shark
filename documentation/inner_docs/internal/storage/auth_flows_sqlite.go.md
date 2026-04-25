# auth_flows_sqlite.go

**Path:** `internal/storage/auth_flows_sqlite.go`
**Package:** `storage`
**LOC:** 341
**Tests:** `auth_flows_sqlite_test.go`

## Purpose
SQLite implementation of auth-flow CRUD + run history. Steps and Conditions are JSON-encoded at the storage boundary so callers always see typed `[]FlowStep` / `map[string]any`.

## Interface methods implemented
- `CreateAuthFlow` (13) — marshals steps + conditions to JSON
- `GetAuthFlowByID` (35)
- `ListAuthFlows` (41) — ordered by `priority DESC, created_at ASC`
- `ListAuthFlowsByTrigger` (64) — same ordering, filtered by trigger
- `UpdateAuthFlow`, `DeleteAuthFlow`
- `CreateAuthFlowRun`, `ListAuthFlowRunsByFlowID`
- Internal scanner + JSON helpers (`marshalFlowSteps`, `scanAuthFlowFromRows`)

## Tables touched
- auth_flows
- auth_flow_runs

## Imports of note
- `database/sql`, `encoding/json`, `time`

## Used by
- `internal/api/auth_flows.go` admin CRUD
- `internal/flows` engine for trigger lookups + run logging

## Notes
- Recursive `FlowStep.ThenSteps`/`ElseSteps` round-trip through plain `json.Marshal/Unmarshal` — no schema migration needed when Step types evolve.
- Run rows are append-only; the dashboard "History" tab caps reads with the `limit` arg.
