# audit_export.go

**Path:** `cmd/shark/cmd/audit_export.go`
**Package:** `cmd`

## Purpose
Implements `shark audit export` — exports audit logs as CSV via POST /api/v1/audit-logs/export.

## Key flags
- `--since` — start date (RFC3339 or YYYY-MM-DD)
- `--until` — end date
- `--output / -o` — write CSV to file instead of stdout
