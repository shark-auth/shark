# session.go

**Path:** `internal/oauth/session.go`
**Package:** `oauth`
**LOC:** 93
**Tests:** indirectly via `jwt_access_token_test.go`, `handlers_test.go`

## Purpose
Defines `SharkSession`, the per-request fosite session that carries OIDC ID-token claims AND JWT access-token claims so fosite's `DefaultJWTStrategy` can sign RFC 7519 access tokens with our ES256 key.

## RFCs implemented
- RFC 7519 JWT — provides claims container fosite signs into the access token.
- OpenID Connect Core — via embedded `openid.DefaultSession`.

## Key types / functions
- `SharkSession` (struct, line 23) — embeds `*openid.DefaultSession`; adds `JWTClaims *jwt.JWTClaims` and `JWTHeader *jwt.Headers`. Satisfies `fosite.Session`, `openid.Session`, AND `oauth2.JWTSessionContainer`.
- `GetJWTClaims` (func, line 37) — returns the claims container fosite mutates pre-sign (scope, aud, exp).
- `GetJWTHeader` (func, line 45) — returns the header container (kid lives here).
- `Clone` (func, line 55) — deep copy required by fosite when stashing sessions. Defers to `DefaultSession.Clone()`, then deep-copies `Extra`, `Audience`, and `Scope` slices on the JWT claims so downstream mutations don't leak across cloned requests.

## Imports of note
- `github.com/ory/fosite`, `handler/openid`, `token/jwt`

## Wired by
- `handlers.go` — `s.newSession("")` returns a fresh `*SharkSession` per token request.
- `exchange.go` — token-exchange path constructs claims directly without going through fosite's session.

## Used by
- Every JWT access-token issuance path. `client_id`, `cnf.jkt`, `act`, and pinned `jti` are all written into `JWTClaims.Extra` from handlers.go before fosite signs.

## Notes
- Deep-copy in Clone is load-bearing: shallow-copying `Extra` map caused cross-request claim leakage in early prototypes.
