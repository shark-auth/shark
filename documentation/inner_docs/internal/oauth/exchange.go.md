# exchange.go

**Path:** `internal/oauth/exchange.go`
**Package:** `oauth`
**LOC:** 379
**Tests:** `exchange_test.go`

## Purpose
RFC 8693 Token Exchange handler — agent-to-agent delegation with nested `act` chains.

## RFCs implemented
- RFC 8693 OAuth 2.0 Token Exchange
- RFC 8707 resource indicator (audience parameter)

## Key types / functions
- Constants `grantTypeTokenExchange`, `tokenTypeAccessToken` (line 30) — URN identifiers.
- `HandleTokenExchange` (func, line 36) — full RFC 8693 flow: authenticates acting agent, parses subject_token, checks revoked-JTI, validates scope narrowing, enforces `may_act`, builds delegation chain, signs new ES256 JWT, persists to `oauth_tokens` with `delegation_subject` + `delegation_actor`.
- `parseSubjectJWT` (later) — verifies the subject token signature against this server's ES256 JWKS.
- `buildActClaim` (later) — constructs the nested `act` chain by wrapping the prior actor in a new `{sub, act}` shell.
- `isMayActAllowed` (later) — checks subject's `may_act` claim against the acting client_id.
- `writeExchangeError` — RFC 6749 §5.2 error envelope.

## Imports of note
- `github.com/golang-jwt/jwt/v5`
- `github.com/matoous/go-nanoid/v2` for jti
- `internal/storage`

## Wired by
- `internal/oauth/handlers.go:56` — `HandleToken` intercepts grant_type `urn:ietf:params:oauth:grant-type:token-exchange`.

## Used by
- Agent-to-agent delegation flows; MCP server-to-server calls.

## Notes
- `resolvedUserID` (line 177) carefully NULLs user_id when subject is a client_id (avoids users(id) FK fail for client_credentials chains).
- Atomic refresh-token rotation shipped 2026-04-24 in store.go affects refresh-grant siblings of exchanged tokens.
- DPoP replay cache is process-local — see SCALE.md.
