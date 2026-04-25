# consent_handlers.go

**Path:** `internal/api/consent_handlers.go`
**Package:** `api`
**LOC:** 221
**Tests:** likely integration-tested

## Purpose
OAuth consent grants — both user-facing (manage your own consents) and admin (cross-user view + revoke). Revoking a consent cascades into best-effort revocation of every OAuth token issued to that client for that user.

## Handlers exposed
- `handleListConsents` (line 26) — GET `/api/v1/auth/consents`. Session-scoped to caller; enriches each row with `agent_name` via `GetAgentByClientID`.
- `handleRevokeConsent` (line 59) — DELETE `.../consents/{id}`. Verifies ownership (IDOR guard via list-and-find), revokes consent, then `RevokeOAuthTokensByClientID` (best-effort), audits `consent.revoked` as user actor.
- `handleAdminListConsents` (line 133) — GET `/admin/oauth/consents`. Cross-user view with optional `?user_id=` filter. Adds `user_id` to wire shape.
- `handleAdminRevokeConsent` (line 174) — DELETE `/admin/oauth/consents/{id}`. Skips ownership check; audits as admin actor; same token-cascade revocation.

## Key types
- `consentResponse` (line 14) — user-facing wire shape with `agent_name`.
- `adminConsentResponse` (line 124) — embeds `consentResponse` + `user_id`.

## Imports of note
- `internal/api/middleware` (`mw.GetUserID`)
- `internal/storage` — `ListConsentsByUserID`, `ListAllConsents`, `RevokeOAuthConsent`, `RevokeOAuthTokensByClientID`, `GetAgentByClientID`

## Wired by
- `internal/api/router.go:352-353` (user-facing)
- `internal/api/router.go:609-610` (admin)

## Notes
- Token cascade failures are non-fatal — the consent is already revoked by the time we attempt token cleanup.
- Agent-name resolution is best-effort; a missing agent silently leaves `agent_name` empty.
