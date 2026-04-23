# feat(proxy): multi-listener reverse proxy (transparent protection on real app port)

**Labels:** `feature`, `proxy`, `architecture`, `P1`, `phase-7`

## TL;DR

Today shark proxy works one of two incomplete ways. Neither delivers "one binary, transparent protection." Ship multi-listener support + finish JWT verify in standalone mode. One shark binary protects N apps on their native ports, no external reverse proxy needed.

## Current state (both modes broken for the target use case)

### Embedded mode (`shark serve --proxy-upstream http://localhost:3001`)
- Single listener on main port (`:8080`)
- `/*` catch-all proxies to upstream
- User must hit `localhost:8080` — not their app's real port
- Admin dashboard shares same port (fine — /admin/* wins route match)
- Auth enforcement works (session cookie + JWT + rules engine)

### Standalone mode (`shark proxy --port 3000 --upstream http://localhost:3001 --auth http://localhost:8080`)
- Dedicated port — user hits `localhost:3000` directly
- BUT: JWT verify is not wired. Per `shark proxy --help`:
  > "MVP scope: this standalone mode treats all requests as anonymous because JWT verification via the auth server's JWKS endpoint is not wired yet. Rules with require:authenticated will therefore deny. For the full-featured proxy use the embedded mode."
- Every `require:authenticated` rule → 401. Useless for real protection.

## Gap

To get Caddy-like transparent protection — user hits `:3000`, shark intercepts, auth enforced, proxies to real app on `:3001` — you must run Caddy/nginx/Traefik in front of shark. Three processes. Three configs. Three restart workflows. Shark's pitch is "one binary." This breaks the pitch for the reverse-proxy use case.

Shark has 90% of the code already:
- `internal/proxy/ReverseProxy` — working
- Rules engine with default-deny — working (commit 6363fe8)
- Circuit breaker with LRU cache — working (commit 718e1e1)
- Session cookie + JWT auth middleware — working
- JWKS fetch infrastructure — working (used by OAuth)

Missing: secondary listener lifecycle + JWKS cache in standalone mode.

## Target config shape

```yaml
proxy:
  listeners:
    - bind: ":3000"
      upstream: "http://localhost:3001"
      session_cookie_domain: localhost
      trusted_headers: ["X-Shark-User-ID", "X-Shark-Email", "X-Shark-Roles"]
      strip_incoming: true
      rules:
        - path: /login
          allow: anonymous
        - path: /_next/*
          allow: anonymous
        - path: /public/*
          allow: anonymous
        - path: /*
          require: authenticated
    - bind: ":3100"
      upstream: "http://localhost:3101"
      rules:
        - path: /*
          require: authenticated
          scopes: ["admin"]
```

Each listener = own port, upstream, rules, trusted headers. Admin/auth stay on `:8080`.

## Legacy compat

Existing config with top-level `enabled/upstream/rules` must keep working unchanged. `Resolve()` compiles legacy fields into `listeners[0]` on main port. No user-facing breaking change.

## User flow (target)

1. React dev server on `:3001`
2. Shark config binds `:3000` listener with upstream `:3001`
3. User browser hits `http://localhost:3000`
4. No session → 302 to `http://localhost:8080/admin/login?return_to=http://localhost:3000/whatever`
5. User logs in, session cookie set on `.localhost` parent domain
6. Browser hits `http://localhost:3000` again → shark validates session → proxies to `:3001` with `X-Shark-User-ID: usr_abc` injected
7. React app reads header in SSR middleware, renders authed page
8. User never sees port `:8080` except during login

## Files + scope

### `internal/config/config.go`
- New `ProxyListenerConfig` struct: `Bind`, `Upstream`, `SessionCookieDomain`, `TrustedHeaders`, `StripIncoming *bool`, `Rules []ProxyRule`, `TimeoutSeconds int`
- `ProxyConfig.Listeners []ProxyListenerConfig`
- `Resolve()`: if `Listeners` empty AND legacy `Enabled/Upstream/Rules` set → synthesize single listener bound to main port
- If both set → log warn, listeners win

### `internal/proxy/engine.go`
- Per-listener engine instance (rules compiled once per listener)
- Shared circuit breaker? Or per-listener? Per-listener — lets a dead upstream not tank the others

### `internal/proxy/listener.go` (new)
- `type Listener struct { server *http.Server; engine *Engine; breaker *Breaker; ... }`
- `(l *Listener) Start(ctx) error`
- `(l *Listener) Shutdown(ctx) error`
- Shares auth middleware + store + RBAC from parent server

### `internal/server/server.go`
- `Build()` iterates `cfg.Proxy.Listeners` → instantiates `Listener` per entry → stored in `s.ProxyListeners []*proxy.Listener`
- `Run()` spawns each listener via goroutine + context-cancel shutdown
- If any listener fails to bind (port in use etc) → fatal, don't start server half-bound
- Graceful shutdown: fan-out shutdown to all listeners + main server

### `cmd/shark/cmd/serve.go`
- Keep `--proxy-upstream` as sugar (compiles to first listener on main port, for dev ergonomics)
- Or deprecate with warning. Decide on ship.

### `cmd/shark/cmd/proxy.go` (standalone mode JWT finish)
- On start, fetch `{--auth}/.well-known/jwks.json` → cache in-memory, refresh every 15 min (or on kid miss)
- For every request with `Authorization: Bearer <jwt>`: verify signature + exp + aud + iss against cached JWKS
- Drop from session cookie path (standalone has no direct DB access)
- `require:authenticated` rules now actually work
- Smoke: standalone with valid JWT = 200, invalid = 401, missing = 401

### `admin/src/components/proxy_config.tsx`
- Page becomes "Protected apps" list (N cards, one per listener)
- Each card: port, upstream, bound status, req/s, circuit state, rules count, "Manage rules" button
- Global status widget now per-listener

### `admin/src/components/proxy_wizard.tsx`
- Wizard Step 1: "Which port should shark listen on to protect this app?" — default to `upstream_port - 1000` (e.g. upstream :3000 → shark :2000). Editable.
- Generates the listener entry, not just rules

### `smoke_test.sh`
- New section 72: multi-listener
  - Spawn toy HTTP server on `:9001`
  - Config shark with listener on `:9000` → `:9001`
  - Hit `:9000` unauthed → 401
  - Hit `:9000` with valid session → 200 + `X-Shark-User-ID` injected to upstream
- New section 73: standalone JWT
  - Start `shark serve` on :8080
  - Start `shark proxy --auth http://localhost:8080 --upstream http://localhost:9001 --port 9002` on :9002
  - No auth → 401
  - Mint JWT via shark oauth token endpoint → send as `Authorization: Bearer <jwt>` to :9002 → 200
  - Tampered JWT → 401
  - Expired JWT → 401

## Acceptance

1. Yaml with ≥2 listeners boots without error, binds all ports, admin port still works
2. Unauthed hit on protected listener → 401 (or 302 to configured login URL)
3. Authed hit → proxied with identity headers injected
4. Legacy single-port config (`enabled/upstream/rules` at top level) works unchanged
5. Standalone mode JWT verify works against shark JWKS
6. Dashboard shows per-listener health + rules + simulator
7. Graceful shutdown: SIGTERM fan-out shuts all listeners, drains connections
8. Port-in-use on any listener → fatal startup error, not partial bind
9. All smoke sections pass

## Out of scope

- TLS termination (document: use Caddy for TLS in production, shark behind it — acceptable trade for single-binary dev/staging experience)
- HTTP/2 upstream (Go stdlib http.Transport default)
- WebSocket upgrades through proxy (current ReverseProxy handles, verify in smoke)
- Multi-binary distribution (single-port standalone mode still useful for edge deployments)

## Why this matters

Without multi-listener, every self-hoster needing transparent protection deploys:
- shark (for auth)
- Caddy/nginx/Traefik (for reverse proxy + forward_auth callback to shark)
- Their app

That's 3 processes, 3 configs, 3 restart cycles. Competing closed-source auth products (Clerk, WorkOS) don't ship reverse proxy at all — too much infra. Shark shipping transparent reverse proxy in the SAME binary = real distribution advantage. The `curl | sh` install still ends with a working single process. Launch-ready differentiator.

## Estimate

4-6h CC.

## Related

- Tracked as **W15** in `FRONTEND_WIRING_GAPS.md`
- Depends on: none
- Unblocks: transparent proxy use case for all self-hosters
- Phase: 7 (SDK phase) — or squeeze into launch if wall-clock allows

## Decision requested

A) Ship W15 into launch scope (pre April 27)
B) Defer to Phase 7 alongside SDK, document manual Caddy workaround in docs
C) Ship W15c (standalone JWT) only — quicker partial win, still requires standalone-mode awareness

Recommend **A** if wall-clock allows — the "one binary" pitch is the launch narrative. Partial delivery undermines it.
