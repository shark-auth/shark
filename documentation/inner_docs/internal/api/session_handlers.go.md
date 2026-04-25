# session_handlers.go

**Path:** `internal/api/session_handlers.go`
**Package:** `api`
**LOC:** 332
**Tests:** `session_agent_p1_test.go`

## Purpose
Session management surface — self-service ("my sessions") for end users plus admin-scoped listing/revocation/purge. Also hosts shared helpers (`ipOf`, `uaOf`, `actorID`, `auditSessionRevoke`, `effectiveLimit`).

## Handlers exposed
### Self-service (`/api/v1/auth/sessions/*`)
- `handleListMySessions` (func, line 50) — `GET`; lists current user's sessions, marks `current=true` for the caller's session
- `handleRevokeMySession` (func, line 82) — `DELETE /{id}`; ownership-checked (404 not 403 to avoid existence leak), audits `session.revoke` (actor_type=user), emits `WebhookEventSessionRevoked`

### Admin (`/api/v1/admin/sessions/*`)
- `handleAdminListSessions` (func, line 115) — `GET`; supports `user_id`, `auth_method`, `mfa_passed`, `limit`, `cursor` query params; emits `next_cursor=created_at|id` when page is full
- `handleAdminDeleteSession` (func, line 171) — `DELETE /{id}`; audits `actor_type=admin`
- `handleListUserSessions` (func, line 199) — `GET /api/v1/users/{id}/sessions`
- `handleRevokeUserSessions` (func, line 224) — `DELETE /api/v1/users/{id}/sessions`; one audit + webhook per revoked session
- `handleAdminRevokeAllSessions` (func, line 253) — `DELETE /api/v1/admin/sessions`; bulk delete + single summary audit; returns `{revoked: N}`
- `handlePurgeExpiredSessions` (func, line 295) — `POST /api/v1/admin/sessions/purge-expired`; returns `{deleted: N}`

## Key types
- `sessionResponse` (struct, line 19), `adminSessionResponse` (line 32 — embeds + `user_email`, `jti`)
- `sessionListResponse` / `adminSessionListResponse` (lines 38, 43) — `{data, next_cursor}`

## Helpers
- `auditSessionRevoke` (line 272), `effectiveLimit` (line 306; default 50, max 200), `ipOf` (line 316; X-Forwarded-For fallback), `uaOf` (line 323), `actorID` (line 327; APIKeyID || UserID)

## Imports of note
- `internal/api/middleware` — `GetUserID`, `GetSessionID`, `APIKeyIDKey`
- `internal/storage` — `ListSessionsOpts`, `WebhookEventSessionRevoked`

## Wired by / used by
- Self-service routes: `internal/api/router.go:343–347`; admin routes: `router.go:587–590`; per-user admin routes: `router.go:434–435`

## Notes
- Self-service revoke returns 404 (not 403) when the session belongs to another user — prevents session-id existence enumeration.
- `last_activity_at` currently aliases `created_at` (TODO).
