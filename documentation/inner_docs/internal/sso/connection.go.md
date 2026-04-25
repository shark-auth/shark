# connection.go

**Path:** `internal/sso/connection.go`
**Package:** `sso`
**LOC:** 239
**Tests:** see `connection_test.go`, `oidc_test.go`, `saml_test.go`

## Purpose
SSO connection management surface: CRUD, email-domain routing, and the find-or-create-user/link-identity flow shared by OIDC and SAML completion handlers. Persists `storage.SSOConnection` and `storage.SSOIdentity` rows.

## Key types
- `SessionCreator` interface — `CreateSession(ctx, userID, ip, ua, authMethod) (*storage.Session, error)`. Decouples package from concrete `auth.SessionManager`.
- `SSOManager` struct — fields: `store storage.Store`, `sessions SessionCreator`, `cfg *config.Config`.
- `NewSSOManager(store, sessions, cfg) *SSOManager`.

## Key methods
- `CreateConnection(ctx, conn)` — validates `Type` is `"saml"|"oidc"`, generates `sso_*` ID, stamps timestamps, defaults `Enabled=true`, and for SAML auto-fills `SAMLSPEntityID` (from `cfg.SSO.SAML.SPEntityID`) and `SAMLSPAcsURL` (`{BaseURL}/api/v1/sso/saml/{id}/acs`) when omitted.
- `GetConnection(ctx, id)` — wraps `sql.ErrNoRows` as "connection not found".
- `ListConnections(ctx)`.
- `UpdateConnection(ctx, conn)` — preserves `CreatedAt`, refreshes `UpdatedAt`.
- `DeleteConnection(ctx, id)` — existence-checked.
- `RouteByEmail(ctx, email)` — splits `@`, lowercases domain, looks up via `GetSSOConnectionByDomain`; rejects disabled connections.
- `findOrCreateUser(ctx, connectionID, providerSub, email, name)` — three-arm logic:
  1. Existing `SSOIdentity` for (connection, sub) → return its user.
  2. User exists by email → reuse + link identity.
  3. Else create user (`usr_*`, `EmailVerified=true` because SSO-verified emails are trusted) + link identity.
- (helpers for `generateID`, `strPtr`, `ptrEmpty` live in this package; `findOrCreateUser` continues past line 200 with identity row creation.)

## Imports
- `context`, `crypto/rand`, `database/sql`, `encoding/hex`, `errors`, `fmt`, `strings`, `time`
- `internal/config`, `internal/storage`.

## Wired by
- Server bootstrap registers `NewSSOManager` and routes (`internal/api/sso_*.go`, `internal/sso/oidc*.go`, `internal/sso/saml*.go`).

## Used by
- OIDC + SAML callback handlers (calling `findOrCreateUser` then `sessions.CreateSession`).
- Admin SSO connection CRUD endpoints.

## Notes
- `SSO-verified emails are trusted` — `EmailVerified=true` short-circuits any verify-email flow; rely on IdP for proof of email ownership.
- `Enabled=true` on create; toggling lives via `UpdateConnection`.
- File continues past 200 lines with the SSO identity link insert and helper functions.
