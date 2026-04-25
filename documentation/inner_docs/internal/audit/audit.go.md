# audit.go

**Path:** `internal/audit/audit.go`
**Package:** `audit`
**LOC:** 100
**Tests:** `audit_test.go`

## Purpose
Audit event logging: Log/Query/GetByID + real-time webhook emission + background cleanup goroutine.

## Key types / functions
- `Logger` (struct, line 14) — wraps storage.Store + optional webhook.Dispatcher
- `NewLogger()` (line 20) — construct from store
- `Logger.SetDispatcher()` (line 25) — wire webhook dispatcher for real-time emission
- `Logger.Log()` (line 30) — records audit event; assigns ID + timestamp if missing; defaults Status="success", ActorType="user"; emits webhook event
- `Logger.Query()` (line 60) — retrieves audit logs with filters + cursor pagination
- `Logger.GetByID()` (line 65) — single entry lookup
- `Logger.DeleteBefore()` (line 70) — deletes entries older than cutoff
- `Logger.StartCleanup()` (line 76) — background goroutine deletes old logs on interval (respects retention duration)

## Imports of note
- `github.com/matoous/go-nanoid/v2` — event ID generation
- `internal/webhook` — Dispatcher for real-time emission
- `log/slog` — structured logging for cleanup errors

## Wired by
- Server.Build() constructs Logger, calls SetDispatcher
- Audit event emission sites (auth, role changes, etc.)

## Used by
- Admin API audit log endpoints (query, list, export)
- Dashboard for audit trail display
- Real-time webhook subscribers
- Compliance/retention policies

## Notes
- Log() auto-generates ID with "aud_" prefix if missing (line 33).
- Timestamp defaults to time.Now().UTC() in RFC3339 format (line 36).
- Metadata defaults to "{}" (empty JSON object) if missing (line 38).
- Status and ActorType given defaults for common cases (line 41, 44).
- Real-time emission only if Dispatcher wired (line 52).
- StartCleanup is idempotent: retention <= 0 or interval <= 0 disables cleanup (line 77).
- Cleanup runs on ticker interval; logs errors via slog but continues (line 93).
- DeleteBefore query filters by created_at < cutoff; count returned (line 70).
