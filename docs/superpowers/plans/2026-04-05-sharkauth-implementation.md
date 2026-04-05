# SharkAuth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build SharkAuth — a single Go binary auth server with passkeys, MFA, OAuth, magic links, RBAC, SSO, M2M API keys, audit logs, admin dashboard, and TypeScript SDK.

**Architecture:** Go HTTP server (stdlib net/http + chi router) with SQLite storage (mattn/go-sqlite3), embedded Svelte dashboard (go:embed), and a zero-dependency TypeScript SDK. All features in one binary. Server-side sessions via encrypted cookies.

**Tech Stack:** Go 1.22+, SQLite, Svelte 5 + SvelteKit, TypeScript, Argon2id, go-webauthn, pquerna/otp, chi router, gorilla/securecookie, pressly/goose (migrations).

**Spec:** `shark-auth-launch-sprint-v2.md` is the source of truth for all data models, endpoints, flows, and config.

---

## Execution Waves

```
Wave 1: Foundation (sequential - 1 agent)
  └── go.mod, config, SQLite, migrations, router skeleton, testutil

Wave 2: Core Auth (sequential - 1 agent, depends on Wave 1)
  └── password hashing, sessions, signup/login/logout/me, basic integration tests

Wave 3: Feature Fan-Out (parallel - 8 agents simultaneously, depend on Wave 2)
  ├── Agent A: OAuth (Google, GitHub, Apple, Discord)
  ├── Agent B: MFA/TOTP + recovery codes
  ├── Agent C: Passkeys/WebAuthn
  ├── Agent D: Magic Links + SMTP email
  ├── Agent E: RBAC (roles, permissions, middleware)
  ├── Agent F: SSO (OIDC + SAML)
  ├── Agent G: M2M API Keys (scoped, rate-limited)
  └── Agent H: Audit Logs (engine, handlers, middleware)

Wave 4: Client-Side (parallel - 2 agents, depend on Wave 3)
  ├── Agent I: Admin Dashboard (Svelte, embedded)
  └── Agent J: TypeScript SDK (@sharkauth/js)

Wave 5: DevOps (1 agent)
  └── Agent K: Docker + CI + Makefile
```

---

## File Structure

```
sharkauth/
├── main.go                              # CLI entry point (cobra: serve, migrate, init)
├── go.mod
├── go.sum
├── sharkauth.yaml                       # Default config
├── shark.toml.example                   # Example config (alternate format)
├── Makefile                             # build, test, embed-dashboard, docker
├── Dockerfile                           # Multi-stage: node build → go build → alpine
├── docker-compose.yml                   # One-command dev setup with Mailpit
├── .github/workflows/ci.yml            # lint → test-go → test-sdk → build
├── .golangci.yml                        # gosec + standard linters
│
├── cmd/
│   └── shark/
│       └── main.go                      # cobra root + serve/migrate/init commands
│
├── internal/
│   ├── config/
│   │   └── config.go                    # YAML + env var loader (koanf or viper)
│   │
│   ├── storage/
│   │   ├── storage.go                   # Storage interface (all DB methods)
│   │   ├── sqlite.go                    # SQLite implementation
│   │   └── migrations.go               # Embedded SQL migrations runner
│   │
│   ├── auth/
│   │   ├── password.go                  # Argon2id hash + verify + bcrypt compat
│   │   ├── password_test.go             # Hash round-trip, bcrypt rehash, wrong pw
│   │   ├── session.go                   # Create, validate, revoke, cookie r/w
│   │   ├── session_test.go              # Expiry, rotation, MFA gating
│   │   ├── oauth.go                     # Generic OAuth handler + provider registry
│   │   ├── providers/
│   │   │   ├── google.go
│   │   │   ├── github.go
│   │   │   ├── apple.go                 # JWT client_secret
│   │   │   └── discord.go
│   │   ├── mfa.go                       # TOTP enroll, verify, challenge, recovery
│   │   ├── mfa_test.go                  # TOTP tolerance, recovery one-time use
│   │   ├── passkey.go                   # WebAuthn register + login (go-webauthn)
│   │   ├── magiclink.go                 # Token gen, hash, verify, email trigger
│   │   └── apikey.go                    # M2M key gen, hash, validate, rate limit
│   │
│   ├── rbac/
│   │   ├── rbac.go                      # Role/permission CRUD + resolution
│   │   ├── rbac_test.go                 # Table-driven permission resolution
│   │   └── middleware.go                # RequirePermission("action", "resource")
│   │
│   ├── sso/
│   │   ├── saml.go                      # SAML SP: metadata, ACS, assertion parsing
│   │   ├── oidc.go                      # OIDC client: redirect, callback, exchange
│   │   └── connection.go                # Connection CRUD + domain routing
│   │
│   ├── audit/
│   │   ├── audit.go                     # Log(), Query(), Cleanup() engine
│   │   ├── audit_test.go               # Event capture, query filters, retention
│   │   └── middleware.go                # HTTP middleware for auto-logging
│   │
│   ├── email/
│   │   ├── sender.go                    # SMTP client (net/smtp + STARTTLS)
│   │   └── templates/
│   │       ├── magic_link.html
│   │       └── verify_email.html
│   │
│   ├── migrate/
│   │   └── auth0.go                     # Auth0 JSON import + rehash
│   │
│   ├── user/
│   │   └── user.go                      # User model + CRUD helpers
│   │
│   ├── testutil/
│   │   ├── db.go                        # NewTestDB (in-memory SQLite)
│   │   ├── server.go                    # TestServer (httptest + cookiejar)
│   │   ├── config.go                    # TestConfig with safe defaults
│   │   ├── factories.go                 # CreateUser, CreateRole, etc.
│   │   └── email.go                     # MemoryEmailSender
│   │
│   └── api/
│       ├── router.go                    # Chi router, mount all routes
│       ├── auth_handlers.go             # Signup, login, logout, me
│       ├── auth_handlers_test.go        # Full flow integration tests
│       ├── oauth_handlers.go            # OAuth redirect + callback
│       ├── passkey_handlers.go          # WebAuthn begin/finish
│       ├── magiclink_handlers.go        # Send + verify
│       ├── mfa_handlers.go              # Enroll, verify, challenge, recovery
│       ├── rbac_handlers.go             # Roles, permissions, assignment, check
│       ├── sso_handlers.go              # Connections, SAML ACS, OIDC callback
│       ├── apikey_handlers.go           # M2M API key CRUD + rotate
│       ├── audit_handlers.go            # Audit log list, detail, export
│       ├── user_handlers.go             # Admin user endpoints
│       ├── migrate_handlers.go          # Migration endpoints
│       └── middleware/
│           ├── auth.go                  # Session validation
│           ├── admin.go                 # Admin API key check
│           └── ratelimit.go             # IP-based + key-based rate limiting
│
├── migrations/
│   └── 001_init.sql                     # All 18 tables in one file
│
├── dashboard/                           # Svelte app
│   ├── package.json
│   ├── svelte.config.js
│   ├── vite.config.ts
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +layout.svelte
│   │   │   ├── +page.svelte             # Overview
│   │   │   ├── users/+page.svelte
│   │   │   ├── users/[id]/+page.svelte
│   │   │   ├── sessions/+page.svelte
│   │   │   ├── roles/+page.svelte
│   │   │   ├── roles/[id]/+page.svelte
│   │   │   ├── sso/+page.svelte
│   │   │   ├── sso/[id]/+page.svelte
│   │   │   ├── api-keys/+page.svelte
│   │   │   ├── audit/+page.svelte
│   │   │   └── migrations/+page.svelte
│   │   └── lib/
│   │       ├── api.ts                   # Fetch wrapper
│   │       └── components/              # Shared components
│   └── static/
│
├── sdk/                                 # @sharkauth/js
│   ├── package.json
│   ├── tsconfig.json
│   ├── tsup.config.ts
│   ├── vitest.config.ts
│   ├── src/
│   │   ├── index.ts
│   │   ├── client.ts
│   │   ├── auth.ts
│   │   ├── passkey.ts
│   │   ├── magic-link.ts
│   │   ├── oauth.ts
│   │   ├── mfa.ts
│   │   └── types.ts
│   └── src/__tests__/
│       ├── auth.test.ts
│       ├── passkey.test.ts
│       └── mfa.test.ts
│
└── examples/
    ├── nextjs/
    └── react-spa/
```

---

## Task 1: Foundation (Wave 1)

**Agent:** Sequential, single agent
**Depends on:** Nothing
**Files:** go.mod, cmd/shark/main.go, internal/config/config.go, internal/storage/*, internal/user/user.go, internal/api/router.go, internal/api/middleware/*, internal/testutil/*, migrations/001_init.sql, sharkauth.yaml

### Subtasks:

- [ ] **1.1: Initialize Go module**

```bash
cd /path/to/shark
# Remove old empty scaffolding files
rm -rf cmd/shark/main.go internal/ migrations/ web/ sqlc.yaml shark.toml.example Makefile

go mod init github.com/sharkauth/sharkauth
```

Add dependencies:
```bash
go get github.com/go-chi/chi/v5
go get github.com/mattn/go-sqlite3
go get github.com/go-webauthn/webauthn
go get github.com/pquerna/otp
go get github.com/gorilla/securecookie
go get golang.org/x/crypto
go get golang.org/x/oauth2
go get github.com/coreos/go-oidc/v3
go get github.com/golang-jwt/jwt/v5
go get github.com/crewjam/saml
go get github.com/knadh/koanf/v2
go get github.com/knadh/koanf/parsers/yaml
go get github.com/knadh/koanf/providers/file
go get github.com/knadh/koanf/providers/env
go get github.com/matoous/go-nanoid/v2
go get github.com/pressly/goose/v3
```

- [ ] **1.2: Write database schema migration (all 18 tables)**

Create `migrations/001_init.sql` with the complete schema from the spec — all tables including audit_logs + indices. Exact SQL is in `shark-auth-launch-sprint-v2.md` lines 92-256.

- [ ] **1.3: Implement config loader**

`internal/config/config.go` — struct matching the spec's YAML config (lines 456-521). Use koanf for YAML parsing + env var overrides. Every field maps to the spec.

- [ ] **1.4: Implement SQLite storage layer**

`internal/storage/storage.go` — interface with all DB methods.
`internal/storage/sqlite.go` — SQLite implementation with WAL mode, foreign keys enabled.
`internal/storage/migrations.go` — goose-based migration runner (pressly/goose) with go:embed. Migrations in `migrations/*.sql` as goose-format SQL files (-- +goose Up / -- +goose Down).

- [ ] **1.5: Implement user model**

`internal/user/user.go` — User struct, Create, GetByID, GetByEmail, List (paginated), Update, Delete.

- [ ] **1.6: Implement base router + middleware**

`internal/api/router.go` — chi router mounting all route groups.
`internal/api/middleware/auth.go` — session validation middleware.
`internal/api/middleware/admin.go` — admin API key check.
`internal/api/middleware/ratelimit.go` — IP-based rate limiter (in-memory token bucket).

- [ ] **1.7: Implement testutil package**

`internal/testutil/db.go` — NewTestDB (in-memory SQLite with migrations).
`internal/testutil/server.go` — TestServer (httptest.Server + http.Client with cookiejar).
`internal/testutil/config.go` — TestConfig with reduced argon2id params.
`internal/testutil/factories.go` — CreateUser, CreateUserWithRole, CreateAndLogin, CreateAPIKey.
`internal/testutil/email.go` — MemoryEmailSender implementing EmailSender interface.

- [ ] **1.8: CLI entry point**

`cmd/shark/main.go` — cobra CLI with `serve`, `migrate`, `init` commands. `serve` loads config, opens DB, runs migrations, starts HTTP server.

- [ ] **1.9: Default config file**

`sharkauth.yaml` — copy of the spec's config example (lines 456-521).

- [ ] **1.10: Commit foundation**

```bash
git add -A && git commit -m "feat: project foundation — go.mod, config, SQLite, router, testutil"
```

---

## Task 2: Core Auth (Wave 2)

**Agent:** Sequential, single agent
**Depends on:** Task 1
**Files:** internal/auth/password.go, internal/auth/session.go, internal/api/auth_handlers.go + tests

### Subtasks:

- [ ] **2.1: Password hashing**

`internal/auth/password.go`:
- `HashPassword(password string) (string, error)` — Argon2id with params: 64MB memory, 3 iterations, 2 parallelism
- `VerifyPassword(password, hash string) (bool, error)` — detect hash type ($argon2id$ or $2a$/$2b$), verify accordingly
- `NeedsRehash(hash string) bool` — returns true for non-argon2id hashes

`internal/auth/password_test.go`:
- Test argon2id round-trip
- Test wrong password fails
- Test bcrypt hash verifies correctly
- Test NeedsRehash detects bcrypt
- Test hash uniqueness (random salt)

- [ ] **2.2: Session management**

`internal/auth/session.go`:
- Session struct matching spec's sessions table
- `CreateSession(userID, ip, userAgent, authMethod string) (*Session, error)`
- `ValidateSession(token string) (*Session, error)` — check expiry, return session
- `RevokeSession(sessionID string) error`
- `SetSessionCookie(w, session)` / `GetSessionFromCookie(r)` — gorilla/securecookie
- `UpgradeMFA(sessionID string) error` — set mfa_passed=1

`internal/auth/session_test.go`:
- Test session create + validate round-trip
- Test expired session rejected
- Test revoked session rejected
- Test cookie set + read

- [ ] **2.3: Auth API handlers**

`internal/api/auth_handlers.go`:
- `POST /api/v1/auth/signup` — validate input, hash password, create user, create session, set cookie
- `POST /api/v1/auth/login` — verify password, check MFA enabled, create session (partial if MFA)
- `POST /api/v1/auth/logout` — revoke session, clear cookie
- `GET /api/v1/auth/me` — return current user from session

`internal/api/auth_handlers_test.go`:
- Integration test: signup → login → me → logout → me=401
- Test duplicate email on signup → 409
- Test wrong password → 401
- Test session cookie is set on signup/login

- [ ] **2.4: Commit core auth**

```bash
git add -A && git commit -m "feat: core auth — password hashing, sessions, signup/login/logout/me"
```

---

## Task 3: OAuth (Wave 3, Agent A)

**Depends on:** Task 2
**Files:** internal/auth/oauth.go, internal/auth/providers/*, internal/api/oauth_handlers.go

Implement the generic OAuth handler pattern + 4 providers (Google, GitHub, Apple, Discord). Each provider implements a `Provider` interface with `AuthURL()`, `Exchange()`, `GetUser()`. Apple requires JWT client_secret generation from .p8 key.

Spec reference: lines 271-278 (endpoints), lines 493-505 (config), lines 633-637 (providers).

One integration test with mock GitHub provider using httptest.Server.

---

## Task 4: MFA/TOTP (Wave 3, Agent B)

**Depends on:** Task 2
**Files:** internal/auth/mfa.go, internal/auth/mfa_test.go, internal/api/mfa_handlers.go

Implement TOTP enrollment, verification, challenge flow, recovery codes (10 per user, bcrypt-hashed, one-time use). MFA login flow: login → mfa_required response → POST /mfa/challenge with TOTP code → session upgraded.

Spec reference: lines 349-371 (endpoints + flow).

Unit tests: TOTP verify, ±1 step tolerance, reject old codes, recovery code uniqueness.
Integration test: enroll → verify → logout → login → mfa_required → challenge → /me works.

---

## Task 5: Passkeys/WebAuthn (Wave 3, Agent C)

**Depends on:** Task 2
**Files:** internal/auth/passkey.go, internal/api/passkey_handlers.go

Use `go-webauthn/webauthn` library. Registration begin/finish, login begin/finish, credential CRUD (list, delete, rename). Discoverable credential flow (no email required) + non-discoverable fallback. Passkey login sets mfa_passed=1 automatically.

Spec reference: lines 280-322 (endpoints + flows), lines 918-943 (implementation notes).

Integration test: begin endpoints return well-formed PublicKeyCredentialCreationOptions.

---

## Task 6: Magic Links + Email (Wave 3, Agent D)

**Depends on:** Task 2
**Files:** internal/email/sender.go, internal/email/templates/*, internal/auth/magiclink.go, internal/api/magiclink_handlers.go

SMTP sender with STARTTLS. HTML email templates (inline CSS). Token generation (32 bytes → base64url), SHA-256 hash storage, 10-minute expiry. Send endpoint with rate limit (1/email/60s). Verify endpoint creates session + redirects. Create-on-first-use flow.

Spec reference: lines 326-347 (endpoints + flow).

Integration test with MemoryEmailSender: send → capture email → extract token → verify → session active → second verify = 400.

---

## Task 7: RBAC (Wave 3, Agent E)

**Depends on:** Task 2
**Files:** internal/rbac/rbac.go, internal/rbac/rbac_test.go, internal/rbac/middleware.go, internal/api/rbac_handlers.go

Role CRUD, permission CRUD, role-permission attachment, user-role assignment, permission resolution (user → roles → permissions), `POST /auth/check` endpoint, `RequirePermission` middleware. Seed default roles (admin, member) on first boot.

Spec reference: lines 373-394 (endpoints).

Unit tests: table-driven permission resolution (admin wildcard, multi-role merge, no roles = no access).
Integration test: user with role → 200, user without role → 403.

---

## Task 8: SSO (Wave 3, Agent F)

**Depends on:** Task 2
**Files:** internal/sso/saml.go, internal/sso/oidc.go, internal/sso/connection.go, internal/api/sso_handlers.go

OIDC client flow (redirect → callback → token exchange → user creation/linking). SAML SP (metadata XML generation, ACS endpoint, assertion parsing, signature verification). Connection CRUD + domain-based auto-routing.

Spec reference: lines 396-413 (endpoints).

Integration test with mock OIDC provider.

---

## Task 9: M2M API Keys (Wave 3, Agent G)

**Depends on:** Task 2
**Files:** internal/auth/apikey.go, internal/api/apikey_handlers.go

Key generation (`sk_live_` + 32 random bytes base62). SHA-256 hash storage. CRUD endpoints (create returns full key ONCE, list shows prefix only). Rotate endpoint (atomic: create new, revoke old). Auth middleware (Bearer token → hash → lookup → scope check → rate limit). In-memory token bucket rate limiter per key.

Spec reference: lines 416-439 (endpoints + auth flow).

Unit tests: key generation, hash round-trip, constant-time comparison.
Integration test: create → use → scope enforcement → rate limit.

---

## Task 10: Audit Logs (Wave 3, Agent H)

**Depends on:** Task 2
**Files:** internal/audit/audit.go, internal/audit/audit_test.go, internal/audit/middleware.go, internal/api/audit_handlers.go

Audit engine: `Log(event)` writes to DB, `Query(filters)` with cursor-based pagination, `Cleanup(retention)` background goroutine. 33 event types per spec. HTTP middleware for auto-logging auth events. API handlers: list (filterable), detail, per-user, export (JSON/CSV).

Spec reference: audit_logs table + events table in the spec.

Integration test: perform login → query audit logs → verify event recorded.

---

## Task 11: Admin Dashboard (Wave 4, Agent I)

**Depends on:** Tasks 3-10 (backend complete)
**Files:** dashboard/* (Svelte app)

Svelte 5 + SvelteKit static adapter. 12 views: overview, users list, user detail, sessions, roles list, role detail, SSO connections list, SSO connection detail, API keys, audit logs, migrations. Fetch wrapper for internal API. Embedded in Go binary via go:embed.

Build: `npm run build` → static output → Go embeds `dashboard/build/`.

---

## Task 12: TypeScript SDK (Wave 4, Agent J)

**Depends on:** Tasks 3-10 (API surface stable)
**Files:** sdk/*

`@sharkauth/js` — zero-dependency, fetch-based, isomorphic (Node/browser/edge). Modules: auth, passkey (ArrayBuffer↔base64url helpers), magic-link, oauth, mfa, types. Build with tsup (ESM + CJS + .d.ts). Tests with vitest + MSW.

Spec reference: lines 639-798 (SDK design).

---

## Task 13: Docker + CI + Makefile (Wave 5, Agent K)

**Depends on:** All backend tasks
**Files:** Dockerfile, docker-compose.yml, .github/workflows/ci.yml, .golangci.yml, Makefile

Multi-stage Dockerfile: node build dashboard → go build → alpine (<30MB). Docker-compose with Mailpit for dev SMTP. GitHub Actions: lint (golangci-lint + gosec) → test-go (-race, 60% coverage) → test-sdk (typecheck + vitest) → build. Makefile targets: build, test, embed-dashboard, docker, lint.

