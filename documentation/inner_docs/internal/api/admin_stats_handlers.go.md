# admin_stats_handlers.go

**Path:** `internal/api/admin_stats_handlers.go`
**Package:** `api`
**LOC:** 205
**Tests:** none colocated

## Purpose
Lightweight dashboard counters + 30/90-day trend series. The cheap snapshot endpoint stays fast (bounded COUNT(*)s on indexed columns); heavier GROUP-BY queries live on the trends endpoint. Also hosts the shared `internal()` 500-writer helper.

## Handlers exposed
- `handleAdminStats` (line 40) — GET `/admin/stats`. Aggregates: total users, users created last 7d, active sessions, MFA adoption %, failed logins (24h), active API keys, expiring API keys (7d), SSO connection counts (total/enabled/per-connection user counts).
- `handleAdminStatsTrends` (line 141) — GET `/admin/stats/trends?days=N` (default 30, max 90). Returns `signups_by_day` (zero-filled) + `auth_methods` GROUP BY since cutoff.

## Key types
- `statsResponse` (line 15) — nested overview shape (Users, Sessions, MFA, FailedLogins24h, APIKeys, SSOConnections).
- `trendsResponse` (line 125), `dayBucket` (line 131), `methodBreakdown` (line 136)

## Helpers
- `fillDailyGaps` (line 182) — fills missing days with count=0 so the frontend chart is contiguous.
- `internal` (line 197) — package-wide 500 writer used by every handler in `internal/api/*`.

## Imports of note
- `internal/storage` — `CountUsers`, `CountActiveSessions`, `CountMFAEnabled`, `GroupUsersCreatedByDay`, `GroupSessionsByAuthMethodSince`, `CountSSOIdentitiesByConnection`

## Wired by
- `internal/api/router.go:585-586`

## Notes
- Active API key count is computed in-memory from `ListAPIKeys` because no `CountActiveAPIKeys` (scope-agnostic) helper exists yet.
