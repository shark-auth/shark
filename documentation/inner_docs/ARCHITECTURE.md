# SharkAuth — High-Level Architecture

This document is the entry point to the per-file inner documentation under `documentation/inner_docs/`. It describes how the pieces fit together: the layers, the surfaces, the data flows, and the boundaries that matter for operation and contribution.

For a per-file reference, see the matching `.md` next to any source path (e.g. `internal/oauth/handlers.go` is documented at `documentation/inner_docs/internal/oauth/handlers.go.md`).

---

## One-paragraph summary

SharkAuth is a single-binary Go service that ships an OAuth 2.1 authorization server, a full identity stack (passwords, passkeys, MFA, magic links, SSO, social login), a multi-tenant org/RBAC model, an admin dashboard (React/Vite SPA embedded in the binary), a reverse-proxy mode that adds auth to any upstream HTTP service with zero code changes, and a token Vault that stores third-party OAuth credentials so agents can act on behalf of users. SQLite (WAL mode) is the only persistence layer in v0.9.x. The product is designed to be `go install`-able, `docker run`-able, and operationally self-contained.

---

## Top-level repository layout

```
shark/
├── cmd/                       — CLI entry points (cobra subcommands)
│   └── shark/
│       ├── main.go            — embeds migrations, calls cmd.Execute
│       └── cmd/               — 25 subcommands (serve, init, app, agent, proxy, branding, paywall, ...)
├── internal/                  — server-side Go packages (private)
│   ├── api/                   — HTTP router + handlers (~50 files, ~200 routes)
│   │   └── middleware/        — auth, RBAC, rate-limit, CORS, security headers, body-limit
│   ├── auth/                  — identity primitives (password, MFA, passkey, magiclink, sessions, lockout, apikey, fieldcrypt, providers)
│   │   └── jwt/               — RS256 JWT manager + key encryption + revocation
│   ├── oauth/                 — OAuth 2.1 AS (fosite-based): handlers, store, exchange, dpop, dcr, device, introspect, revoke, metadata, audience, consent, session, errors
│   ├── storage/               — SQLite store + Store interface + goose migrations runner
│   ├── proxy/                 — reverse-proxy mode (rules engine, lifecycle state machine, circuit breaker, LRU)
│   ├── sso/                   — upstream SAML SP + OIDC client logic
│   ├── rbac/                  — role/permission/wildcard matcher + org-scoped HTTP middleware
│   ├── webhook/               — outbound event dispatcher (HMAC + retry + dead-letter)
│   ├── audit/                 — audit log emitter + query + cleanup
│   ├── server/                — top-level Server struct, first-boot bootstrap
│   ├── config/                — koanf YAML + env-var schema
│   ├── email/                 — Sender interface, SMTP, Resend, dev inbox, templates
│   ├── vault/                 — third-party OAuth credential store + provider templates
│   ├── authflow/              — visual flow engine (login/signup/MFA/paywall page templates)
│   ├── telemetry/             — anonymous install ping
│   ├── admin/                 — embedded admin SPA file server
│   ├── identity/              — request principal struct
│   ├── user/                  — user CRUD service wrapper
│   └── testutil/              — test helpers (not production)
├── admin/src/                 — React admin dashboard (TypeScript, Vite, ~85 files)
│   ├── components/            — page components + shared widgets
│   ├── design/                — primitives (Button, Input, Card) + tokens
│   ├── hosted/                — hosted login/signup/MFA/paywall pages (separately bundled)
│   └── lib/                   — API client, hooks, utilities
├── packages/                  — npm-published frontend SDKs
│   └── shark-auth-react/      — React SDK (@shark-auth/react)
├── sdk/                       — non-npm SDKs
│   ├── typescript/            — Node SDK
│   └── python/                — Python SDK
├── migrations/                — 22+ goose SQL migrations (embedded via go:embed)
├── docs/                      — user-facing developer docs
├── documentation/             — internal references (this file + api_reference + inner_docs)
└── gstack/                    — launch session artifacts
```

---

## Five HTTP surfaces

SharkAuth exposes five distinct HTTP surfaces. Each has its own root path, error envelope, and authentication model. The full route table lives in `internal/api/router.go` (219 routes registered).

| Surface | Root path | Owner package | Error envelope | Documented in |
|---|---|---|---|---|
| **OAuth 2.1 Authorization Server** | `/oauth/*`, `/.well-known/oauth-authorization-server`, `/.well-known/jwks.json` | `internal/oauth` | RFC 6749 §5.2 (`{error, error_description, error_uri}`) | `documentation/api_reference/sections/oauth.yaml` |
| **Authentication & Identity** | `/api/v1/auth/*` | `internal/api` (handlers) + `internal/auth` (primitives) | Extended (`{error, code, message, docs_url, details}`) | `documentation/api_reference/sections/auth.yaml` |
| **Platform** | `/api/v1/{organizations,roles,permissions,users,sso,api-keys,audit-logs,webhooks,agents}` | `internal/api` + `internal/storage` + `internal/rbac` | Extended | `documentation/api_reference/sections/platform.yaml` |
| **Admin Operations** | `/api/v1/admin/*`, `/api/v1/vault/*` | `internal/api` (admin_*.go + vault_handlers.go) | Extended | `documentation/api_reference/sections/admin.yaml` |
| **System** | `/healthz`, `/assets/branding/*`, `/hosted/*`, `/paywall/*` | `internal/api` + `internal/admin` | none (HTML / image) | `documentation/api_reference/sections/system.yaml` |

The split exists because each surface has a different audience (developer vs end-user vs operator vs MCP client) and a different security model. Mixing them would muddy both the public API contract and the per-surface error catalog.

---

## Architectural layers

```
                ┌──────────────────────────────────────────────────────┐
                │  Client tier                                         │
                │  - admin/src React SPA (embedded in binary)          │
                │  - SDKs (TS, Python, React)                          │
                │  - MCP clients (any RFC-compliant OAuth client)      │
                │  - Proxy upstreams (any HTTP service)                │
                └──────────────────────────────────────────────────────┘
                                       │ HTTP
                                       ▼
                ┌──────────────────────────────────────────────────────┐
                │  Router (internal/api/router.go)                     │
                │  - chi.Router; 219 routes                            │
                │  - Middleware chain: cors → security headers →        │
                │    body-limit → rate-limit → auth → rbac             │
                └──────────────────────────────────────────────────────┘
                                       │
                                       ▼
                ┌──────────────────────────────────────────────────────┐
                │  Handlers (internal/api/*_handlers.go)                │
                │  - Per-surface handler files                          │
                │  - Validate request, call services, emit audit       │
                └──────────────────────────────────────────────────────┘
                                       │
                                       ▼
                ┌──────────────────────────────────────────────────────┐
                │  Services / domain logic                             │
                │  - internal/auth (password, mfa, session, lockout)   │
                │  - internal/oauth (fosite-backed AS)                 │
                │  - internal/proxy (reverse proxy, rules engine)      │
                │  - internal/sso (upstream IdP)                       │
                │  - internal/rbac (role/permission decisions)         │
                │  - internal/webhook (outbound event fanout)          │
                │  - internal/audit (event log)                        │
                │  - internal/vault (3rd-party OAuth tokens)           │
                │  - internal/email (SMTP / Resend / dev inbox)        │
                │  - internal/authflow (hosted page flows)             │
                └──────────────────────────────────────────────────────┘
                                       │
                                       ▼
                ┌──────────────────────────────────────────────────────┐
                │  Storage interface (internal/storage/storage.go)     │
                │  ~120 methods grouped by domain                      │
                └──────────────────────────────────────────────────────┘
                                       │
                                       ▼
                ┌──────────────────────────────────────────────────────┐
                │  SQLite (internal/storage/sqlite*.go)                │
                │  - WAL mode + foreign keys                           │
                │  - 22+ goose migrations embedded via go:embed         │
                └──────────────────────────────────────────────────────┘
```

Strict layer rule: handlers may not touch SQLite directly. They go through the `Store` interface so the storage backend can be swapped (Postgres planned for Q3 2026).

---

## Concern separation

| Concern | Owner |
|---|---|
| HTTP routing | `internal/api/router.go` |
| Request validation, response shaping | `internal/api/*_handlers.go` |
| OAuth protocol logic (RFCs) | `internal/oauth/*` (composes fosite) |
| Identity primitives (password, MFA, sessions) | `internal/auth/*` |
| RBAC decisions | `internal/rbac/*` |
| Persistence | `internal/storage/*` |
| Reverse-proxy decisions | `internal/proxy/*` |
| Audit log writes | `internal/audit/*` |
| Webhook fanout | `internal/webhook/*` |
| Email delivery | `internal/email/*` |
| Configuration | `internal/config/*` |
| First-boot bootstrap | `internal/server/*` + `cmd/shark/cmd/init.go` |
| CLI surface | `cmd/shark/cmd/*` |
| Admin SPA | `admin/src/*` (built to `admin/dist/`, served by `internal/admin`) |
| Hosted pages | `admin/src/hosted/*` (separately bundled) + `internal/api/hosted_handlers.go` |

Cross-cutting concerns (audit emission, RBAC checks, rate limiting) are wired in via middleware in `internal/api/middleware/` so handlers stay declarative.

---

## Key data flows

### 1. Password login (browser)

```
Browser
  │ POST /api/v1/auth/login {email, password}
  ▼
router.go → middleware (cors, rate-limit) → auth_handlers.handleLogin
  │
  ├─→ internal/auth/lockout.IsLocked(email)         (process-local map)
  ├─→ internal/storage.GetUserByEmail               (SQLite)
  ├─→ internal/auth/password.VerifyPassword         (argon2id, bcrypt fallback)
  ├─→ if MFA enabled: return MFA-required, set pending challenge
  ├─→ internal/auth/session.NewSession              (securecookie AES-GCM)
  ├─→ internal/audit.Log(actor=user, action=login, status=success)
  └─→ internal/webhook.Dispatch(user.login)          (async fanout)
  │
  ▼
Set-Cookie: shark_session=...
Response: {user, session_expires_at}
```

### 2. OAuth 2.1 authorization-code flow with DCR + PKCE + DPoP

```
MCP client (cold start)
  │ GET /.well-known/oauth-authorization-server     (RFC 8414 discovery)
  ├─→ oauth/metadata.go MetadataHandler             (cached JSON)
  │
  │ POST /oauth/register {redirect_uris, ...}        (RFC 7591 DCR)
  ├─→ oauth/dcr.go HandleDCRRegister
  │   - issues client_id + client_secret + registration_access_token
  │   - persists via storage.CreateOAuthDCRClient
  │
  │ GET /oauth/authorize?response_type=code&...&code_challenge=...   (PKCE S256)
  ├─→ oauth/handlers.go HandleAuthorize
  │   - looks up session, redirects to consent if needed
  │   - persists PKCE challenge separately (sanitization workaround)
  │
  │ POST /oauth/token (grant_type=authorization_code, code, code_verifier, DPoP header)
  ├─→ oauth/handlers.go HandleToken
  │   - Validates DPoP proof (oauth/dpop.go ValidateDPoPProof)
  │     - signature check
  │     - jti replay check (in-memory cache — process-local!)
  │     - jkt thumbprint extraction
  │   - fosite issues access_token (JWT, ES256, includes cnf.jkt)
  │   - fosite issues refresh_token (opaque, family-tracked)
  │   - audience binding via RFC 8707 resource indicators
  │
  ▼
Returns {access_token, refresh_token, expires_in, token_type=DPoP}
```

### 3. RFC 8693 agent-to-agent delegation chain

```
Agent A holds a token. Wants to call resource as Agent B on behalf of itself.

POST /oauth/token
  grant_type=urn:ietf:params:oauth:grant-type:token-exchange
  subject_token=<agent A's access token>
  subject_token_type=urn:ietf:params:oauth:token-type:access_token
  actor_token=<agent A's identity token>
  resource=https://billing.example.com
  │
  ▼
oauth/handlers.go HandleToken → intercepts grant_type → oauth/exchange.go HandleTokenExchange
  │
  ├─→ Validates subject_token (signature + cnf.jkt + may_act constraints)
  ├─→ Validates actor_token
  ├─→ Builds nested `act` claim chain:
  │     {sub: B, act: {sub: A, act: {sub: original_user}}}
  │   so the resource server can audit the full delegation depth
  ├─→ Issues new access_token bound to agent B with chain in claims
  └─→ internal/audit.Log(actor=A, action=delegate_to, target=B)
```

### 4. Reverse-proxy request

```
Curl/MCP client request to upstream:
  GET https://my-app.example.com/api/billing → arrives at SharkAuth proxy listener (port 8080)
  │
  ▼
internal/proxy/listener.go (single-listener model in v0.9.x)
  │
  ├─→ proxy/rules.go Match(request)                  (first-match-wins)
  │   - matches on path + method + required-scope + required-tier + required-mfa
  │
  ├─→ if rule.Action == paywall: redirect to /paywall/{app_slug}
  ├─→ if rule.Action == block: 403
  ├─→ if rule.Action == pass:
  │   - validates Bearer token (introspect locally; verify DPoP if cnf.jkt present)
  │   - injects X-Shark-User, X-Shark-Scopes, X-Shark-Trace-ID headers
  │   - circuit breaker check (proxy/circuit.go — per-instance state!)
  │   - reverse-proxies to upstream
  │
  ├─→ internal/audit.Log(action=proxy.allow, target=upstream)
  └─→ Forwards response (with optional response-header rewrites)
```

### 5. Webhook fanout

```
Any handler emits an event:
  internal/webhook.Dispatcher.Dispatch(event_type, payload, org_id)
  │
  ▼
- Loads matching webhook subscriptions from storage
- For each subscription:
  - HMAC-SHA256 sign payload with subscription's secret
  - POST to subscription.url with signature header
  - On non-2xx or timeout, schedule retry: [1m, 5m, 30m, 2h, 12h]
  - After 5 attempts: dead-letter
  - All attempts persisted for replay via /api/v1/webhooks/{id}/deliveries/{deliveryId}/replay
  - Real-time emission to admin SSE stream so dashboard shows deliveries live
```

---

## Identity model

```
        ┌─────────────────────┐
        │   Organization      │  multi-tenant root
        └─────────┬───────────┘
                  │ has many
        ┌─────────┴───────────┐
        │ OrganizationMember  │  (user_id ↔ org_id, with roles[])
        └─────────┬───────────┘
                  │
        ┌─────────┴───────────┐         ┌──────────────┐
        │       User          │ has many │   Session    │
        └─────────┬───────────┘─────────▶└──────────────┘
                  │ may have
                  ├─→ MFASettings (TOTP secret + recovery codes)
                  ├─→ PasskeyCredential[] (WebAuthn)
                  ├─→ OAuthAccount[] (Google/GitHub social link)
                  ├─→ Consent[] (granted scopes per Application)
                  └─→ VaultConnection[] (3rd-party OAuth tokens)

        ┌─────────────────────┐
        │     Application     │  OAuth client (admin-managed, has slug + branding)
        └─────────────────────┘

        ┌─────────────────────┐
        │       Agent         │  OAuth client (programmatic, identity for M2M / agentic flows)
        └─────────┬───────────┘
                  │ holds
                  └─→ AgentTokens (issued via OAuth; can sub-delegate via RFC 8693)

        ┌─────────────────────┐
        │   DCR Client        │  RFC 7591 dynamically-registered OAuth client
        └─────────────────────┘  (different from Application — usually an MCP server)
```

OAuth tokens carry:
- `sub` — user_id or agent_id
- `act` — nested delegation chain (RFC 8693)
- `cnf.jkt` — DPoP key thumbprint binding (RFC 9449)
- `aud` — audience-restricted to a specific resource (RFC 8707)
- `family_id` — refresh token family for reuse detection
- `request_id` — fosite's correlation across rotation chain

---

## Token lifecycle

| Token type | Format | Lifetime | Rotation | Revocation |
|---|---|---|---|---|
| Access token | JWT, ES256 signed | 1h default | Reissue via refresh | Revoke by `jti` (admin or user); checked on every protected request |
| Refresh token | Opaque hash | 30d default | One-shot rotation; family-tracked; reuse triggers family revocation | Revoke endpoint or family revocation on reuse |
| Authorization code | Opaque hash | 10m | n/a — one-shot | Consumed at /oauth/token |
| Device code | Opaque hash + user_code | 5m default | n/a | Approve/deny at /oauth/device/verify |
| DCR registration token | Bearer | until rotated | Rotate via `/oauth/register/{client_id}/registration-token` | Revoke endpoint |
| Magic link token | SHA-256 hash | configurable (default 15m) | n/a — one-shot | Consumed on verify |
| Passkey challenge | random | 5m | n/a | TTL expiration |

Atomic rotation invariant (shipped 2026-04-24): `RotateRefreshToken` uses a single `UPDATE ... WHERE revoked_at IS NULL RETURNING id` to ensure exactly one caller wins under concurrent refresh attempts. See `internal/storage/oauth_sqlite.go:191` and `internal/oauth/store.go:255`.

---

## Cross-cutting concerns

### Audit

Every state-changing handler calls `internal/audit.Log(actor, action, target, status, metadata)`. Audit entries are persisted in `audit_logs` table and emitted in real-time to webhook subscribers. Background cleanup goroutine purges entries older than the configured retention. See `documentation/inner_docs/internal/audit/audit.go.md`.

### RBAC

All admin-scoped + many platform endpoints go through `internal/rbac/org_middleware.go` which extracts org_id from path or header, calls `RBAC.HasPermission(user, resource, action, org_id)`, and 403s on deny. RBAC supports wildcards (`billing:*`, `*:read`) and roles can be org-scoped. Default seeds populate root + admin + member roles on first boot.

### Rate limiting

Three rate limiters live in process memory (per-replica state — see SCALE.md):
- Global router-level token bucket (`router.go:211`, 100 rps burst)
- Per-email magic-link cooldown (`magiclink_handlers.go:30-69`, 60s)
- Per-API-key bucket (`auth/apikey.go:121`, configurable)

DPoP nonce/replay cache is also process-local (`oauth/dpop.go:48-78`). Same scaling caveat.

### Account lockout

`internal/auth/lockout.go` tracks failed logins per email in an in-memory map. After 5 fails, locks for 15 minutes. Cleanup goroutine sweeps stale entries every 10 minutes. Per-replica state.

### Configuration

`internal/config/config.go` uses koanf. Loads from `sharkauth.yaml` then overrides with env vars (`${SHARK_*}` interpolation). Secret fields (server.secret, smtp.password, signing keys) are loaded once at startup and never re-read.

---

## Frontend architecture (admin SPA)

The admin dashboard at `admin/src/` is a React 18 + TypeScript + Vite SPA bundled to `admin/dist/` and served by `internal/admin` via the Go binary's embedded file system.

```
admin/src/
├── main.tsx               — bootstrap; ReactDOM.render
├── App.tsx                — root router; auth state; renders <Layout> + active page
├── components/            — page components (one per top-nav item) + shared widgets
│   ├── overview.tsx       — dashboard with SSE-driven live activity stream
│   ├── users.tsx          — user CRUD table
│   ├── organizations.tsx  — split-pane org browser (members + roles + invitations)
│   ├── rbac.tsx           — role/permission matrix
│   ├── proxy_config.tsx   — proxy rules + lifecycle controls
│   ├── ...                — 30+ more (one per feature)
│   ├── api.tsx            — fetch-based HTTP client with bearer auth + 401 redirect
│   ├── toast.tsx          — notification system
│   ├── shared.tsx         — icon set, Avatar, CopyField, Sparkline, Donut
│   └── CommandPalette.tsx — Cmd+K global navigation
├── design/                — design system primitives + tokens
│   ├── primitives/        — Button, Input, Card
│   └── tokens.ts          — color, spacing, typography, motion
└── hosted/                — separately-bundled hosted login/signup/MFA/paywall pages
    ├── App.tsx            — auth-flow router
    └── routes/            — login, signup, mfa, etc.
```

Conventions:
- One file per page component, named after the nav-item slug (users.tsx, sessions.tsx, audit.tsx, etc.).
- Hooks live alongside components when single-use, in `lib/hooks/` when shared.
- API calls go through `components/api.tsx` — never raw fetch in components.
- All state is local + URL-synced; no global Redux/Zustand — `useURLParams.tsx` syncs filters to query string.
- Toasts and command-palette are global widgets mounted at App level.

The hosted pages are a separate Vite entry (`hosted-entry.tsx`) so end-user-facing pages don't ship the admin bundle. They render server-side via `internal/api/hosted_handlers.go` reading the prebuilt HTML and injecting the active Application's branding.

---

## Single-binary distribution model

```
go build -o sharkauth ./cmd/shark
```

Produces a single binary that contains:
- All Go server code
- Embedded migrations (`migrations/*.sql` via `go:embed`)
- Embedded admin SPA (`admin/dist/*` via `go:embed`)
- Embedded hosted pages
- Embedded email templates (with overrides via DB)
- Embedded consent screen HTML

`shark serve --dev` launches with:
- Ephemeral SQLite DB (file or in-memory)
- Random server secret
- SMTP captured to dev inbox (`internal/email/devinbox.go`)
- Admin key printed to stdout
- Auto-opens browser to admin dashboard

`shark serve --no-prompt` is the production launch mode used in containers. Reads `sharkauth.yaml` + env vars; no first-boot prompt; emits one-time bootstrap token to stdout for first admin creation.

---

## External integrations

| External | Direction | Where | Notes |
|---|---|---|---|
| SMTP | outbound | `internal/email/smtp.go` | TLS/STARTTLS, sync send (no queue) |
| Resend | outbound | `internal/email/resend.go` | HTTP API alternative to SMTP |
| Google OAuth | inbound (social login) | `internal/auth/oauth.go` + `auth_handlers.go` | Standard OAuth 2.0 client |
| GitHub OAuth | inbound (social login) | same | same |
| Upstream OIDC IdPs | inbound (SSO) | `internal/sso/oidc.go` | Per-connection IdP config; nonce replay protection |
| Upstream SAML IdPs | inbound (SSO) | `internal/sso/saml.go` | SP mode; URI-qualified attribute fallback |
| Vault providers | outbound | `internal/vault/providers.go` | Stores OAuth tokens for Google/GitHub/Slack/Microsoft/Notion/Linear/Jira on user's behalf |
| Webhook URLs | outbound | `internal/webhook/dispatcher.go` | HMAC signed; retry with backoff; dead-letter |
| Reverse-proxy upstreams | bidirectional | `internal/proxy/listener.go` | Drop-in auth in front of any HTTP service |
| Anonymous telemetry | outbound | `internal/telemetry/ping.go` | Optional; sends install ID + version every 24h |

There is no Redis, no Postgres, no Kafka, no message queue, no separate worker process. v0.9.x is single-process by design.

---

## Scale boundaries (read SCALE.md before deploying)

In v0.9.x, four pieces of state live in process memory and split-brain across replicas:

1. DPoP replay cache (`internal/oauth/dpop.go`)
2. Device-flow rate limiter (`internal/oauth/device.go`)
3. Per-IP rate limiter (`internal/api/middleware/ratelimit.go`)
4. Per-API-key rate limiter (`internal/auth/apikey.go`)

Plus: SQLite WAL lock means multiple processes writing to the same `dev.db` file is unsafe. Plus: proxy circuit breaker state (`internal/proxy/circuit.go`) is per-instance.

For single-replica deployments (the supported model), none of these matter. For horizontal scale (planned Q3 2026), the roadmap is:

1. Atomic refresh-token rotation — **shipped 2026-04-24** (closes a single-replica race too)
2. PostgreSQL driver alongside SQLite
3. Move the four in-memory caches to either Postgres-backed tables with time-windowed indexes or an optional Redis driver
4. Move proxy circuit-breaker state to shared cache
5. Helm chart for Kubernetes horizontal pod autoscaling

---

## Building and developing

```bash
# Build (Go 1.25+)
make build               # produces ./shark

# Run with dev defaults
./shark serve --dev      # in-memory DB, dev inbox, opens browser

# Run with production config
./shark serve --config /etc/sharkauth.yaml

# Run tests
make test                # unit + integration
make smoke               # end-to-end smoke

# Build admin SPA only (when iterating on frontend)
cd admin && pnpm install && pnpm dev    # http://localhost:5173 with API proxy

# Generate OpenAPI bundle
npx @redocly/cli bundle documentation/api_reference/openapi.yaml --ext yaml --output _bundled.yaml
```

---

## Conventions worth knowing

- **No interface-driven OOP**. Storage and email/sender are interfaces because they have multiple implementations or need test mocks. Most code is plain functions and structs.
- **Errors flow up unchanged**. Handlers translate to the appropriate envelope at the boundary (RFC 6749 §5.2 for /oauth/*, extended for everything else). Internal error wrapping uses standard `fmt.Errorf("...: %w", err)`.
- **slog is the logger**. `slog.Info`, `slog.Error`. Default JSON handler. No third-party logger.
- **No reflection-based config**. koanf parses YAML into a typed struct; env-var overrides via `${VAR}` interpolation.
- **Migrations are append-only**. New schema changes get a new file under `migrations/`. Never edit a shipped migration.
- **SQL written by hand**. No ORM. `database/sql` + `modernc.org/sqlite` driver (CGO-free).
- **Tests prefer real SQLite**. Most tests run against a fresh SQLite instance, not a mock store. Slower but catches contract drift.
- **Audit on the way out**. Every state-changing handler emits an audit log entry before returning. Don't return success without auditing.

---

## Where to dig next

- For a specific HTTP route → `internal/api/router.go` line, then the handler file's per-file MD.
- For OAuth-protocol questions → `internal/oauth/` per-file MDs (each names the RFC it implements).
- For DB schema → `internal/storage/storage.go.md` (master interface + entity types).
- For the admin dashboard surface → `admin/src/components/` per-file MDs.
- For machine-readable API spec → `documentation/api_reference/`.
- For launch + planning context → `gstack/`.

---

## Generated

This document was generated as part of the SharkAuth launch sprint on 2026-04-24. It accompanies 217 per-file MDs under `documentation/inner_docs/` produced by parallel doc agents. Update this file when adding a new top-level package, new HTTP surface, or new external integration.
