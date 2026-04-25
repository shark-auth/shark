# router.go

**Path:** `internal/api/router.go`
**Package:** `api`
**LOC:** 931
**Tests:** none (covered indirectly via handler integration tests)

## Purpose
Defines the `Server` struct (the API host's dependency container) and `NewServer` — the single function that wires every chi route, middleware, subsystem (proxy, OAuth AS, SSO, RBAC, audit, webhook dispatcher, vault, flow engine), and resolver onto one `chi.Router`.

## Handlers exposed
- `NewServer` (func, line 126) — constructor; mounts global middleware (`RequestID`, `RealIP`, `Logger`, `Recoverer`, `MaxBodySize 1MB`, `SecurityHeaders`, `RateLimit 100/100`, optional `CORS`) then every route tree
- `handleHealthz` (func, line 910) — `GET /healthz`; pings DB and returns `{status: ok|unhealthy}`
- `notImplemented` (func, line 924) — 501 stub used for `/api/v1/migrate/*`
- `initProxy` (method, line 791) — builds proxy Engine + Breaker + ReverseProxy + Manager when `Proxy.Enabled`
- `proxyAuthMiddleware` / `ProxyAuthMiddlewareFor` (line 867 / 875) — resolves identity (JWT or session) into `proxy.Identity` for the catch-all reverse-proxy mount

## Key types
- `Server` (struct, line 31) — holds Store, Config, SessionManager, PasskeyManager, OAuthManager, MagicLinkManager, JWTManager, RBAC, AuditLogger, RateLimiter, LockoutManager, FieldEncryptor, SSOHandlers, WebhookDispatcher, OAuthServer, VaultManager, FlowEngine, ProxyEngine/Breaker/Handler/Listeners/Manager, AppResolver
- `ServerOption` (type, line 81) — functional options: `WithProxyListeners`, `WithEmailSender`, `WithWebhookDispatcher`, `WithJWTManager`, `WithOAuthServer`

## Route surface (high level)
- Public: `/healthz`, `/.well-known/jwks.json`, `/.well-known/oauth-authorization-server`, `/assets/branding/*`, `/hosted/{app}/{page}`, `/paywall/{app}`
- `/api/v1/auth/*` — signup, login, logout, /me, email verify, oauth, passkey, magic-link, password, mfa, flow/mfa, sso, sessions, consents, revoke
- `/api/v1/organizations/*` — session-auth + RBAC permission gates
- `/api/v1/{roles,permissions,users,sso,api-keys,audit-logs,webhooks,agents,admin/apps,vault}` — admin-key gated
- `/api/v1/admin/*` — bootstrap consume (line 576), config, stats, sessions, branding, email-templates, proxy admin/lifecycle/rules, flows, organizations
- `/oauth/*` — OAuth 2.1 AS (token, authorize, register/DCR, introspect, revoke, device)
- `/admin*` — embedded React dashboard SPA
- Catch-all `/*` — reverse proxy (only when `ProxyHandler != nil`)

## Imports of note
- `github.com/go-chi/chi/v5` — router + standard middleware
- `internal/api/middleware` (aliased `mw`) — auth, ratelimit, CORS, security headers, admin API key
- `internal/auth`, `internal/auth/jwt`, `internal/oauth`, `internal/sso`, `internal/proxy`, `internal/authflow`, `internal/audit`, `internal/webhook`, `internal/vault`, `internal/rbac`, `internal/admin`

## Wired by / used by
- Constructed by `internal/server` startup; `Router` field exposed for `http.Server`

## Notes
- Route order matters: bootstrap `/admin/bootstrap/consume` (line 576) is mounted BEFORE the `/admin` group so `AdminAPIKeyFromStore` doesn't gate it.
- Reverse proxy catch-all is mounted last so chi trie routes static API paths first.
- Branding `/assets/branding/*` registers both GET and HEAD so email previews don't hit proxy 401s.
