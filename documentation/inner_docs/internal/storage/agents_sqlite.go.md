# agents_sqlite.go

**Path:** `internal/storage/agents_sqlite.go`
**Package:** `storage`
**LOC:** 295
**Tests:** `agents_sqlite_test.go`

## Purpose
SQLite implementation of every Agent-domain method on the `Store` interface: CRUD plus secret + DCR-secret rotation.

## Interface methods implemented
- `CreateAgent` (12) — JSON-encodes slices/metadata, NULL-safes `created_by`
- `GetAgentByID` (43)
- `GetAgentByClientID` (48) — used by fosite token-endpoint client auth
- `ListAgents` (53) — search + active filter + count + paginated results
- `UpdateAgent`, `UpdateAgentSecret`, `DeactivateAgent`
- `RotateDCRClientSecret` — writes new + previous hash with grace window expiry (F4.3)

## Tables touched
- agents

## Imports of note
- `database/sql`, `encoding/json`, `time`

## Used by
- `internal/api/agents.go` admin handlers
- `internal/api/dcr.go` DCR registration + rotation
- `internal/oauth/store.go` fosite client lookup

## Notes
- All slice + metadata fields go through `json.Marshal`/`Unmarshal` at the boundary.
- `created_by` is NULL-safed because empty string would violate the FK to `users(id)`.
- Timestamps stored as RFC3339 UTC strings (consistent with the rest of storage).
