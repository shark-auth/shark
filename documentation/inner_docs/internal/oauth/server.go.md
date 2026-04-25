# server.go

**Path:** `internal/oauth/server.go`
**Package:** `oauth`
**LOC:** 267
**Tests:** indirectly via `handlers_test.go`, `jwt_access_token_test.go`

## Purpose
Constructs the fosite-backed OAuth 2.1 Authorization Server, owning ES256 signing-key lifecycle and the in-memory DPoP replay cache.

## RFCs implemented
- RFC 6749 (OAuth 2.0 Framework — composed via fosite)
- RFC 7519 JWT access tokens (DX1 — `AccessTokenIssuer` + JWT strategy)
- RFC 7636 PKCE (`EnforcePKCE: true`)
- RFC 9449 DPoP (replay cache field)

## Key types / functions
- `Server` (struct, line 26) — bundles `fosite.OAuth2Provider`, `FositeStore`, ES256 priv key, `DPoPJTICache`, `Issuer`.
- `NewServer` (func, line 40) — wires fosite: ensures ES256 key, builds HMAC + JWT strategies, composes grant factories (authorize_explicit, client_credentials, refresh, PKCE, revocation, introspection).
- `ensureES256Key` (func, line 131) — loads or generates the active ES256 key from `jwt_signing_keys`; transactional retire-and-replace when server.secret rotates.

## Imports of note
- `github.com/ory/fosite` + `compose`, `handler/openid`, `token/jwt`
- `github.com/go-jose/go-jose/v4`
- `internal/auth/jwt` (key encoding + AES-GCM key encryption)
- `internal/storage`, `internal/config`

## Wired by
- `internal/server/server.go` — instantiates one `*Server` and mounts handlers under `/oauth/*`.

## Used by
- `handlers.go`, `exchange.go`, `dcr.go`, `device.go`, `introspect.go`, `revoke.go` — all bind methods on `*Server`.

## Notes
- Refresh tokens + auth codes stay opaque (HMAC); only access tokens are JWTs.
- `SHA256Hasher` (defined in store.go) replaces fosite's bcrypt hasher.
- DPoP cache is process-local — see SCALE.md before horizontally scaling.
- Stale-key retirement uses an explicit tx so /oauth/* never breaks if the insert fails.
