# oauth_sqlite.go

**Path:** `internal/storage/oauth_sqlite.go`
**Package:** `storage`
**LOC:** 623
**Tests:** `oauth_sqlite_test.go`

## Purpose
SQLite implementation of every OAuth-domain method on the `Store` interface: authorization codes, PKCE sessions, tokens (access + refresh), consents, device codes, and DCR clients.

## Interface methods implemented
- Authorization codes: `CreateAuthorizationCode` (12), `GetAuthorizationCode` (26), `DeleteAuthorizationCode` (48), `DeleteExpiredAuthorizationCodes` (53)
- PKCE sessions: `CreatePKCESession` (64) with `INSERT OR REPLACE`, `GetPKCESession` (79), `DeletePKCESession` (98), `DeleteExpiredPKCESessions` (103)
- OAuth tokens:
  - `CreateOAuthToken` (114) — NULL-safes `agent_id`/`user_id`/`request_id` so empty strings don't trip the FK
  - `GetActiveOAuthTokenByRequestIDAndType` (148) — latest non-revoked row for fosite request ID + type
  - `GetOAuthTokenByJTI` (158), `GetOAuthTokenByHash` (166)
  - `RevokeOAuthToken` (174), `RevokeActiveOAuthTokenByRequestID` (191) — atomic UPDATE-with-subselect that closes the concurrent-refresh race
  - `RevokeOAuthTokensByClientID`, `RevokeOAuthTokenFamily`, `ListOAuthTokensByAgentID`, `DeleteExpiredOAuthTokens`, `UpdateOAuthTokenDPoPJKT`
- Consents: `CreateOAuthConsent`, `GetActiveConsent`, `ListConsentsByUserID`, `ListAllConsents`, `RevokeOAuthConsent`
- Device codes: `CreateDeviceCode`, `GetDeviceCodeByUserCode`, `GetDeviceCodeByHash`, `ListPendingDeviceCodes`, `UpdateDeviceCodeStatus`, `UpdateDeviceCodePolledAt`, `DeleteExpiredDeviceCodes`
- DCR: `CreateDCRClient`, `GetDCRClient`, `UpdateDCRClient`, `DeleteDCRClient`, `RotateDCRRegistrationToken`

## Tables touched
- oauth_authorization_codes
- oauth_pkce_sessions
- oauth_tokens
- oauth_consents
- oauth_device_codes
- oauth_dcr_clients

## Imports of note
- `database/sql`
- `time` — every timestamp serialized as RFC3339 UTC

## Used by
- `internal/oauth/store.go` (FositeStore wrapper)
- `internal/oauth/handlers.go` indirectly via the wrapper

## Notes
- `RevokeActiveOAuthTokenByRequestID` (line 191) is the atomic refresh-rotation primitive: single UPDATE with a subselect picking the freshest active row plus an outer `revoked_at IS NULL` predicate. SQLite WAL serializes per-table writes, so concurrent refresh attempts deterministically resolve — only one row gets revoked, others see zero rows affected and back off. Closes the read-then-write race that existed when `RotateRefreshToken` did two statements.
- `INSERT OR REPLACE` on `oauth_pkce_sessions` is intentional — fosite may resubmit the same signature_hash during retries.
