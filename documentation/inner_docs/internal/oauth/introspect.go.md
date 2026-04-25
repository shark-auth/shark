# introspect.go

**Path:** `internal/oauth/introspect.go`
**Package:** `oauth`
**LOC:** 267
**Tests:** `introspect_test.go`

## Purpose
RFC 7662 Token Introspection endpoint plus shared 3-tier bearer-lookup helper used by other packages.

## RFCs implemented
- RFC 7662 OAuth 2.0 Token Introspection

## Key types / functions
- `introspectResponse` (struct, line 28) — RFC 7662 §2.2 fields including `agent_id` extension.
- `HandleIntrospect` (func, line 45) — authenticates caller (admin Bearer or client creds), looks up token, returns `{active:false}` for missing/revoked/expired tokens, otherwise authoritative DB-derived claims.
- `LookupBearer` (func, line 125) — public wrapper around `findTokenInDB` so vault/agent-auth can share canonical lookup.
- `findTokenInDB` (func, line 138) — 3-tier lookup: (1) JWT `jti` claim → `GetOAuthTokenByJTI`; (2) opaque `key.sig` form → `sha256(sig)` → `GetOAuthTokenByHash`; (3) full-token sha256 fallback.
- `extractJTIFromJWT` (func, line 171) — unverified JWT parse to pull `jti`; DB lookup is the actual trust boundary.
- `authenticateClient` (func, line 188) — admin-key (sk_live_*) or client-credential auth.

## Imports of note
- `github.com/golang-jwt/jwt/v5` (unverified parse only)
- `internal/auth` (HashAPIKey)
- `internal/storage`

## Wired by
- `internal/server/server.go` — mounts `POST /oauth/introspect`.

## Used by
- `revoke.go` (shares `findTokenInDB` + `authenticateClient`).
- `internal/vault`, `internal/api/middleware` (via `LookupBearer`).

## Notes
- Cache-Control: no-store on every response.
- Missing/empty `token` returns `{active:false}` not 4xx (RFC 7662 §2.1).
- Username enrichment is best-effort (silent on user lookup failure).
