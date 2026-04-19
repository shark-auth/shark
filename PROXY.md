# Shark Proxy

> Zero-code auth. Put Shark in front of your backend. Clients authenticate with Shark; your backend reads identity from injected headers. Circuit breaker keeps the proxy running even when the auth server hiccups.

**Status:** Shipped in Phase 6 (`claude/admin-vendor-assets-fix` branch, commits `50ef326..d728114`).

---

## TL;DR

```yaml
# sharkauth.yaml
proxy:
  enabled: true
  upstream: "http://localhost:3000"
  rules:
    - { path: "/api/admin/*", require: "role:admin" }
    - { path: "/api/public/*", allow: "anonymous" }
    - { path: "/webhooks/*", require: "agent", scopes: ["webhooks:write"] }
    - { path: "/api/*", require: "authenticated" }
```

```bash
shark serve                          # embedded (recommended)
shark serve --proxy-upstream http://localhost:3000   # one-liner override
shark proxy --upstream ... --auth ...                # standalone (anonymous-only MVP)
```

Your backend now reads `X-User-ID`, `X-User-Email`, `X-User-Roles`, `X-Agent-ID`, `X-Auth-Method` directly from request headers. Every auth check, role gate, and agent scope lives in one place: Shark's YAML.

---

## Why Proxy

Every web app repeats the same lines:

```js
const session = await validateSessionFromCookie(req);
const user = await db.users.find(session.user_id);
if (!user.roles.includes('admin')) return res.status(403);
const roles = await db.roles.byUser(user.id);
```

Shark Proxy moves that into config:

```yaml
rules:
  - { path: "/api/admin/*", require: "role:admin" }
```

Upstream handler reads `X-User-ID`, trusts it (Shark stripped any spoofed version), and does its actual work. Auth logic doesn't live in app code anymore.

---

## Two Modes

### Embedded (recommended)

Same binary. Shark serves both auth routes (`/api/v1/*`, `/oauth/*`, `/admin/*`, `/.well-known/*`) AND proxies everything else to your upstream.

```bash
shark serve --proxy-upstream http://localhost:3000
# or set proxy.enabled + proxy.upstream in sharkauth.yaml
```

### Standalone (MVP)

Separate `shark proxy` process. Connects to a Shark auth instance via its base URL.

```bash
shark proxy --upstream http://localhost:3000 --auth http://auth.internal:8080
```

**Caveat:** MVP standalone is anonymous-only. JWT verification requires fetching JWKS from the auth server — deferred to follow-up (`TODO(P4.1)`). Rules with `require: authenticated` or anything requiring identity will 403 in standalone mode today. Use embedded if you need that.

---

## Rules

First match wins. No match → 403 (default deny). Rule order is authoritative.

```yaml
rules:
  - path: "/api/admin/*"           # wildcards: /foo/*, /foo/*/bar, /foo/{id}
    methods: ["GET", "POST"]       # optional; omit = any
    require: "role:admin"
    # scopes: ["admin:read"]       # extra scope constraint (AND'd with require)

  - path: "/api/public/*"
    allow: "anonymous"             # alias for require: anonymous

  - path: "/webhooks/*"
    require: "agent"
    scopes: ["webhooks:write"]

  - path: "/api/*"
    require: "authenticated"
```

### Requirement types

| Value | Meaning |
|---|---|
| `anonymous` | Always allow (no auth needed) |
| `authenticated` | `user_id` OR `agent_id` present |
| `role:X` | User has role `X` |
| `permission:X:Y` | User has permission `Y` on resource `X` — **stubbed, returns 403 until Phase 6.5 wires RBAC** |
| `agent` | Agent identity (OAuth agent token) |
| `scope:X` | Token scope includes `X` |

### Path matching

- `/api/foo` — exact
- `/api/foo/*` — prefix match (`/api/foo`, `/api/foo/bar`, `/api/foo/bar/baz`)
- `/api/*/deep` — single-segment wildcard
- `/api/{id}` — param placeholder (single-segment wildcard, no capture in MVP)
- Case-sensitive. Must begin with `/`.

### Method filter

`methods: ["GET", "POST"]` narrows the rule. A PUT request against a GET-only rule is treated as no-match (falls through to the next rule), NOT an explicit deny.

### Default deny

Empty rules list → every request returns 403. This is the default posture. Add `allow: anonymous` rules to open paths.

---

## Identity headers

Shark strips these headers from every inbound request (anti-spoofing), then injects the real values before forwarding to upstream:

| Header | Value |
|---|---|
| `X-User-ID` | User's `usr_xxx` ID |
| `X-User-Email` | User's email |
| `X-User-Roles` | Comma-joined role names (e.g. `admin,user`) |
| `X-Agent-ID` | Agent's `ag_xxx` ID (when agent-authed) |
| `X-Agent-Name` | Agent's display name |
| `X-Auth-Method` | `jwt` / `session-live` / `session-cached` / `anonymous` / `anonymous-degraded` |
| `X-Shark-Auth-Mode` | Same as `X-Auth-Method` (duplicate for upstream clarity) |
| `X-Shark-Cache-Age` | Seconds since last live verification (only set when > 0) |

Upstream code reads these as ground truth — Shark already stripped any spoofed variants from the client.

### Trusted header allowlist

If you need to pass through a header that matches the stripped prefixes (`X-User-*`, `X-Agent-*`, `X-Shark-*`) but is NOT an identity header (e.g. `X-User-Feature-Flag`), add it to `proxy.trusted_headers`:

```yaml
proxy:
  trusted_headers: ["X-User-Feature-Flag"]
```

Use sparingly. Every trusted header is a potential spoofing surface.

---

## Circuit Breaker

The proxy survives auth-server outages. Four layers:

| Layer | Mechanism | Protects |
|---|---|---|
| **L1 JWT local** | Stateless ES256 signature check via cached JWKS | Agents + OAuth flows — never goes down |
| **L2 Session cache** | LRU map `SHA-256(cookie) → identity`, 5-minute TTL | Human sessions |
| **L3 Health monitor** | Pings `/api/v1/admin/health` every 10s; 3 fails → open | Detection |
| **L4 Degraded mode** | `X-Shark-Auth-Mode: session-cached` + `X-Shark-Cache-Age` headers | Upstream awareness |

### States

```
closed ──(3 fails)──> open ──(success)──> half-open ──(success)──> closed
                                                  └──(fail)────────> open
```

**closed** (healthy): every request hits the auth server live, identity returned + cached.

**open** (auth server down):
- JWT tokens → verified locally from cached JWKS, zero degradation
- Cookie with cache hit → served from cache, `X-Shark-Cache-Age: N` tells upstream how stale
- Cookie with cache miss → `miss_behavior: reject` (default) returns 401; `allow_readonly` permits GET/HEAD as anonymous

**half-open**: after one success during open, next probe decides — success closes, failure re-opens.

### Negative cache

Known-bad tokens (last auth-server lookup returned 401) get cached with 30s TTL. Prevents hammering the auth server when an attacker replays expired tokens.

### Why agents never go down

Agent OAuth access tokens are JWTs signed with the server's ES256 key. Verification is pure math: parse, check expiry, verify signature against a cached public key. Zero database calls, zero auth-server calls. Even during a full auth-server outage, agent traffic flows unaffected.

---

## Admin API

All endpoints require admin API key (Bearer token in `Authorization` header).

### `GET /api/v1/admin/proxy/status`

Current circuit state snapshot.

```json
{
  "data": {
    "state": "closed",           // closed | open | half-open
    "cache_size": 42,
    "neg_cache_size": 3,
    "failures": 0,
    "last_check": "2026-04-19T20:00:00Z",
    "last_latency_ns": 12400000,
    "last_status": 200,
    "health_url": "http://localhost:8080/api/v1/admin/health"
  }
}
```

Returns **404** when proxy is disabled in config.

### `GET /api/v1/admin/proxy/status/stream`

Server-Sent Events, 2s tick. Each event is the full `status` JSON. Client disconnect closes stream cleanly.

**Note:** EventSource native API doesn't support custom headers, so dashboard polls `/status` every 2s instead. SSE endpoint works with `curl` or a custom client that can set `Authorization`.

### `GET /api/v1/admin/proxy/rules`

Compiled rule list. Use this for dashboard rule preview.

```json
{
  "data": [
    { "path": "/api/admin/*", "methods": ["GET","POST"], "require": "role:admin", "scopes": [] },
    { "path": "/api/public/*", "methods": [], "require": "anonymous", "scopes": [] }
  ]
}
```

### `POST /api/v1/admin/proxy/simulate`

The money endpoint. Paste a URL + identity → see what would happen.

**Request:**
```json
{
  "method": "GET",
  "path": "/api/users",
  "identity": {
    "user_id": "usr_abc",
    "roles": ["admin"],
    "scopes": ["admin:read"],
    "auth_method": "session-live"
  }
}
```

**Response:**
```json
{
  "matched_rule": { "path": "/api/*", "require": "authenticated" },
  "decision": "allow",
  "reason": "",
  "injected_headers": {
    "X-User-ID": "usr_abc",
    "X-User-Roles": "admin",
    "X-Auth-Method": "session-live"
  },
  "eval_us": 12
}
```

Used by the dashboard's simulator hero. Backend evaluates the actual `proxy.Engine.Evaluate` — no mocks, no drift.

---

## Dashboard

Navigate to `/admin/proxy` (keyboard: `g p`).

- **Circuit strip**: three live gauges (L1 JWT local · L2 Session cache · L3 Upstream health). Green pulse when closed, amber when half-open, red when open. Polls `/status` every 2s.
- **Test URL simulator**: method + path input, optional identity overrides, click Simulate → result card shows matched rule, decision, injected headers, eval time in µs.
- **Rules table**: read-only list. Each row has a `Test` button that seeds the simulator with that rule's path.

Editing rules in the dashboard is **deferred to P5.1** — edit `sharkauth.yaml` and reload for now. Full inline editor with drag-reorder planned.

---

## CLI

```bash
shark serve --proxy-upstream http://localhost:3000
```

Mounts the proxy as a catch-all on Shark's serve process. All `/api/v1/*`, `/oauth/*`, `/admin/*`, `/.well-known/*` routes win via chi trie precedence — only unmatched paths are proxied.

```bash
shark proxy --upstream http://localhost:3000 --auth http://auth.internal:8080 --port 8081 [--rules rules.yaml]
```

Standalone mode. MVP is anonymous-only until standalone JWKS fetch lands (TODO P4.1).

---

## Operational notes

- **Panic recovery**: upstream panics are caught, logged, return 503. Shark never crashes because of an upstream bug.
- **Timeout**: `proxy.timeout_seconds` (default 30). Upstream responses past the deadline return 502 with `"upstream unreachable"` (no error details leak).
- **Anti-spoofing**: inbound `X-User-ID`, `X-Agent-*`, `X-Shark-*` are stripped unconditionally. Even if `strip_incoming: false`, the canonical identity headers are overwritten at inject-time (double-layer defense).
- **Graceful shutdown**: the circuit breaker's health-monitor goroutine exits when the process receives SIGTERM.
- **Thread safety**: rule engine is immutable after construction. Config reload requires a new engine instance (not hot-swap).

---

## Roadmap

**P5.1** — inline rule editor in dashboard (currently YAML-only).
**P5.1** — drag-reorder rules with optimistic mutations.
**P4.1** — standalone proxy JWT verification via JWKS fetch.
**Phase 6.5** — RBAC wiring for `require: permission:X:Y` (currently stubbed to deny).

---

## Testing

```bash
# Unit tests — 79 pass (core + rules + circuit)
go test ./internal/proxy/... -count=1 -v

# HTTP handlers — 20+ pass
go test ./internal/api/... -run Proxy -count=1 -v

# Smoke tests
./smoke_test.sh   # sections 49 cover proxy admin endpoints
```

---

## Reference

- Plan: `docs/superpowers/plans/2026-04-18-proxy.md`
- Code: `internal/proxy/` (core, rules, circuit, LRU), `internal/api/proxy_handlers.go` (admin API), `cmd/shark/cmd/proxy.go` (standalone), `admin/src/components/proxy_config.tsx` (dashboard)
- Tests: `internal/proxy/*_test.go`, `internal/api/proxy_handlers_test.go`
