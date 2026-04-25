# admin_oauth_handlers.go

**Path:** `internal/api/admin_oauth_handlers.go`
**Package:** `api`
**LOC:** 137
**Tests:** none colocated

## Purpose
Admin-key surface for the OAuth device-flow approval queue — lets dashboard operators list, approve, or deny pending device codes (cross-user override of the user-facing verify flow).

## Handlers exposed
- `handleAdminListDeviceCodes` (line 34) — GET `/api/v1/admin/oauth/device-codes`. Default = pending + non-expired; `?status=all` returns every row. Enriches each with `agent_name` resolved via `GetAgentByClientID` (cached per request).
- `handleAdminApproveDeviceCode` (line 72) — POST `.../device-codes/{user_code}/approve`. Delegates to `adminDecideDeviceCode("approved")`.
- `handleAdminDenyDeviceCode` (line 77) — POST `.../device-codes/{user_code}/deny`.
- `adminDecideDeviceCode` (line 81, internal) — shared core: 404 on unknown, 410 on expired, 409 on non-pending; optional `{user_id}` body binds the issued token to a specific user; emits audit `oauth.device.{approved|denied}`.

## Key types
- `adminDeviceCodeResponse` (line 16) — wire shape mirroring `OAuthDeviceCode` minus the internal hash; resolves `agent_name`.

## Imports of note
- `github.com/go-chi/chi/v5` — URL params
- `internal/storage` — `ListPendingDeviceCodes`, `GetDeviceCodeByUserCode`, `UpdateDeviceCodeStatus`, `GetAgentByClientID`, AuditLog

## Wired by
- `internal/api/router.go:615-617`

## Notes
- Optional `{user_id}` body lets admin bind token to a specific user; defaults to the existing `dc.UserID` if omitted.
- All audits are best-effort (`_ = AuditLogger.Log(...)`); failures don't block the response.
