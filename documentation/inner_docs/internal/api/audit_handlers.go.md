# audit_handlers.go

**Path:** `internal/api/audit_handlers.go`
**Package:** `api`
**LOC:** 385
**Tests:** likely integration-tested

## Purpose
Audit log query, single-record fetch, per-user merged view, CSV export, and admin purge. Wraps `internal/audit.Logger` queries with cursor-based pagination + RFC3339 date validation.

## Handlers exposed
- `handleListAuditLogs` (line 31) — GET `/admin/audit-logs`. Supports filters: `action`, `actor_id`, `actor_type`, `target_id`, `org_id`, `session_id`, `resource_type`, `resource_id`, `status`, `ip`, `from`, `to`, `cursor`, `limit` (default 50, max 200). Returns `{data, next_cursor, has_more}`.
- `handleGetAuditLog` (line 112) — GET `/{id}`.
- `handleUserAuditLogs` (line 136) — GET `/users/{id}/audit-logs`. Queries logs where user is actor OR target, dedupes, sorts by created_at desc.
- `handleExportAuditLogs` (line 230) — POST `/admin/audit-logs/export`. CSV writer over RFC3339 from/to range with optional `action` filter.
- `handlePurgeAuditLogs` (line 336) — POST `/admin/audit-logs/purge`. Deletes rows older than the retention window.

## Key types
- `auditListResponse` (line 17) — `{data, next_cursor, has_more}`
- `auditExportRequest` (line 24) — `{from, to, action?}`

## Helpers
- `strPtrVal` (line 322), `sortAuditLogsByCreatedAtDesc` (line 374), `newAuditLogger` (line 383, package factory).

## Imports of note
- `internal/audit` — `Logger` (Query, GetByID)
- `internal/storage` — AuditLog + AuditLogQuery
- `encoding/csv` — export

## Wired by
- `internal/api/router.go:480-482`, `:591` (purge), `:433` (per-user)

## Notes
- Pagination uses opaque cursor = last row's ID; the handler fetches `limit+1` to detect `has_more`.
- Export streams CSV directly to `w` with Content-Type `text/csv`.
