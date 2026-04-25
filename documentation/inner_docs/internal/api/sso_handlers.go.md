# sso_handlers.go

**Path:** `internal/api/sso_handlers.go`
**Package:** `api`
**LOC:** 344
**Tests:** likely integration-tested

## Purpose
SSO (SAML + OIDC) handlers. Distinct from the rest of `api/` because SSO is a self-contained subsystem with its own handler struct (`SSOHandlers`) carrying an in-memory state cache for OIDC nonce/state values. SAML endpoints are public (per the spec); CRUD is admin-key gated.

## Handlers exposed
**Connection CRUD (admin)**
- `(*SSOHandlers).CreateConnection` (line 62) — POST `/sso/connections`
- `ListConnections` (line 78), `GetConnection` (line 88), `UpdateConnection` (line 105), `DeleteConnection` (line 128)

**SAML (public)**
- `SAMLMetadata` (line 146) — GET `/sso/saml/{connection_id}/metadata`. Returns SP XML metadata.
- `SAMLACS` (line 166) — POST `/sso/saml/{connection_id}/acs`. Assertion Consumer Service. Issues JWT (access+refresh or session JWT) when configured.

**OIDC (public)**
- `OIDCAuth` (line 204) — start.
- `OIDCCallback` (line 231) — exchange code, validate state/nonce, issue JWT.

**Auto-routing**
- `SSOAutoRoute` (line 304) — domain-based connection picker.

## Key types
- `SSOHandlers` (line 24) — `{manager, server, mu, stateStore}`. `stateStore` is `map[string]*ssoStateEntry` cleaned every 5 min by a background goroutine.
- `ssoStateEntry` (line 17) — `{connectionID, nonce, expiresAt}`

## Constructor
- `NewSSOHandlers` (line 32) — spawns a 5-minute cleanup ticker for the state store.

## Helpers
- `cleanupStates` (line 48), `ssoPathParam` (line 342)

## Imports of note
- `internal/sso` — `SSOManager` (CreateConnection, GenerateSPMetadata, HandleSAMLACS, etc.)
- `internal/storage` — `SSOConnection`
- Server's `JWTManager` for token issuance

## Wired by
- `internal/api/router.go:180-185` (`SSOManager` init); routes mounted via the `SSOHandlers.Routes()` helper or directly.

## Notes
- State store is in-memory only — multi-replica deployments need either sticky sessions or a shared cache for OIDC.
- JWT issuance branches on `Config.Auth.JWT.Mode`: `access_refresh` issues a pair, otherwise a single session JWT.
