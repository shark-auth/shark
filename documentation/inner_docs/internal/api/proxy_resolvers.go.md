# proxy_resolvers.go

**Path:** `internal/api/proxy_resolvers.go`
**Package:** `api`
**LOC:** 292
**Tests:** likely integration-tested

## Purpose
Auth resolvers wired into the proxy engine. Translates the incoming HTTP request (Bearer JWT or session cookie) into a `proxy.Identity` the rule engine can evaluate. Two implementations: `JWTResolver` (stateless, never blocked by breaker) and `LiveResolver` (cookie → session lookup, breaker-protected). Also provides `DBAppResolver` for host → app mapping.

## Handlers / methods
- `(*JWTResolver).Resolve` (line 44) — extracts Bearer, ignores `sk_*` admin keys (so dashboard requests fall through to LiveResolver), validates JWT via `jwt.Manager`, bakes Identity from claims (Tier + Scopes always, Roles best-effort from store with fallback to `claims.Roles`).
- `(*LiveResolver).Resolve` (line 118) — extracts session cookie via `auth.SessionManager`, validates session, hydrates email + roles + `tier` from `users.metadata` via `tierFromMetadata`.
- `(*DBAppResolver).ResolveApp` (line 209) — host → `proxy.ResolvedApp` lookup via storage.

## Key types
- `JWTResolver` (line 35) — `{JWT, Store}`
- `LiveResolver` (line 108) — `{Sessions, Store, RBAC}`
- `DBAppResolver` (declared elsewhere; method here)

## Helpers
- `tierFromMetadata` (line 174) — parses `users.metadata` JSON → `tier` field. Mirrors what `jwt.Manager` bakes into Claims so JWT and session paths agree.
- `extractBearer` (line 190) — case-insensitive `Bearer ` extractor.

## Package state
- `proxySessionCookieName` (line 23, const = `"shark_session"`) — kept in sync with `internal/auth/session.go`.
- `ErrNoCredentials` (line 30) — sentinel for no-credential requests.

## Imports of note
- `internal/auth`, `internal/auth/jwt`, `internal/proxy`, `internal/rbac`, `internal/storage`

## Wired by
- Constructed in `internal/api/router.go` `initProxy` (line ~791) and passed to the proxy engine.

## Notes
- JWT path bypasses the breaker (stateless verification doesn't depend on auth-server health).
- Email/role lookup failures inside JWT path are non-fatal — the JWT already proved identity.
- Skipping `sk_*` in JWTResolver lets admin API keys + session cookies coexist on the same request.
