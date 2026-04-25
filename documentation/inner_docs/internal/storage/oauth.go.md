# oauth.go

**Path:** `internal/storage/oauth.go`
**Package:** `storage`
**LOC:** 91
**Tests:** indirectly via `oauth_sqlite_test.go` and `internal/oauth/store_test.go`.

## Purpose
Type-only file: declares the OAuth-domain entity structs that `oauth_sqlite.go` round-trips and that `internal/oauth` consumes via the `Store` interface.

## Types defined
- `OAuthAuthorizationCode` (line 6) — short-lived authorization code (PKCE + RAR + nonce fields).
- `OAuthToken` (line 22) — access/refresh token row with JTI, `RequestID` (fosite request ID, may repeat across rotations), DPoP `JKT`, delegation subject/actor, `FamilyID` for refresh-rotation lineage.
- `OAuthConsent` (line 44) — user's consent grant for an agent, with optional `RevokedAt`/`ExpiresAt`.
- `OAuthDeviceCode` (line 56) — RFC 8628 device authorization (status: pending/approved/denied/expired).
- `OAuthDCRClient` (line 71) — RFC 7591 dynamically registered client.
- `OAuthPKCESession` (line 84) — persists PKCE challenge separately from auth code because fosite calls `CreatePKCERequestSession` with the unsanitized challenge after `CreateAuthorizeCodeSession` has stripped form values.

## Used by
- `internal/storage/oauth_sqlite.go` — implementation.
- `internal/oauth/store.go` — fosite adapter wrapping `Store`.
- `internal/oauth/handlers.go` indirectly via the store.

## Notes
- `RequestID` on `OAuthToken` is JSON-hidden (`-`) because it's a fosite-internal correlation key, not part of the wire model.
- All hash/secret fields use `json:"-"` to prevent leaking into API responses.
