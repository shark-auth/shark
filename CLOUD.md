# SharkAuth Cloud — Architecture Analysis & Roadmap

**Date:** 2026-04-13
**Status:** Planning — not yet started
**Prerequisite:** Self-hosted v0.1.0 ships first (April 27 target)

---

## What is Shark Cloud

A multi-tenant SaaS fork of SharkAuth. Same auth engine, different deployment model:

| | Self-Hosted | Cloud |
|---|---|---|
| Dashboard | Svelte (embedded in binary) | Next.js (separate app) |
| Database | SQLite (embedded, zero-config) | Postgres (managed, multi-tenant) |
| OAuth/Email | Customer configures | Preconfigured per tenant |
| Pricing | $0 forever | $19/$49/$149/mo tiers |
| Scaling | Single instance | Multi-instance behind LB |
| Tenant model | Single tenant | Multi-tenant, isolated |

The core value proposition: "You're paying for ops, not features." Every feature available in self-hosted must also be available in Cloud. Cloud sells operational convenience, not capabilities.

---

## Cloud Readiness Verdict

**Not on track. Fixable, but there are 2 critical blockers and 2 moderate risks.**

The self-hosted codebase has a clean foundation — the storage interface is properly abstracted, the REST API is comprehensive, and the config system is extensible. But the architecture assumes a single process with a single SQLite file, which breaks fundamentally in a cloud deployment.

---

## Critical Blocker 1: In-Memory State

**Severity:** Critical — breaks multi-instance deployment
**Effort:** 3-5 days
**Must fix before:** Any cloud deployment

Cloud means multiple instances behind a load balancer. Five components hold state exclusively in Go maps protected by mutexes. None of this state survives process restarts or is visible to other instances.

### Components affected

| Component | File | State | TTL | What breaks in multi-instance |
|-----------|------|-------|-----|-------------------------------|
| Account lockout | `internal/auth/lockout.go` | `map[string]*lockoutEntry` | 15 min | User fails 3x on instance A, retries on instance B — lockout bypassed entirely |
| WebAuthn challenges | `internal/auth/passkey.go:42-110` | `map[string]*challengeEntry` | 5 min | Passkey registration/login starts on A, browser callback hits B — challenge not found, ceremony fails |
| Magic link rate limiter | `internal/api/magiclink_handlers.go:26-73` | `map[string]time.Time` | 60 sec | Per-email rate limit is per-instance. User can spam magic links by hitting different instances |
| IP rate limiter | `internal/api/middleware/ratelimit.go:47-104` | `map[string]*tokenBucket` | 10 min | Rate limit buckets are per-instance. Attacker gets N * bucket_size requests across N instances |
| SSO state store | `internal/api/sso_handlers.go:23-44` | `map[string]*ssoStateEntry` | 10 min | SSO initiated on A, IdP callback lands on B — state mismatch, auth fails |

### Fix: Cache interface

Add a `Cache` interface that abstracts key-value storage with TTL:

```go
type Cache interface {
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}
```

Implementations:
- `internal/cache/memory.go` — in-process map (for self-hosted, maintains current behavior)
- `internal/cache/redis.go` — Redis/Valkey client (for cloud)

All 5 components migrate from direct map access to the Cache interface. Self-hosted users see no change. Cloud deployments configure Redis via `SHARKAUTH_CACHE__URL=redis://...`.

### Impact on self-hosted

None. The memory cache implementation preserves current behavior exactly. Self-hosted users never need Redis.

---

## Critical Blocker 2: Zero Tenant Isolation

**Severity:** Critical — data leaks between customers
**Effort:** 1-2 weeks
**Must fix before:** Any multi-tenant deployment

Not a single table has a `tenant_id` column. Every query is global. Two customers on the same Shark Cloud instance share everything — users, roles, API keys, audit logs, SSO connections.

### Schema changes required

New table:
```sql
CREATE TABLE tenants (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,      -- subdomain: {slug}.sharkauth.cloud
    plan        TEXT DEFAULT 'free',
    config      TEXT DEFAULT '{}',          -- per-tenant JSON config
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);
```

Every existing table gets a new column:
```sql
ALTER TABLE users ADD COLUMN tenant_id TEXT NOT NULL REFERENCES tenants(id);
ALTER TABLE sessions ADD COLUMN tenant_id TEXT NOT NULL REFERENCES tenants(id);
ALTER TABLE roles ADD COLUMN tenant_id TEXT NOT NULL REFERENCES tenants(id);
ALTER TABLE permissions ADD COLUMN tenant_id TEXT NOT NULL REFERENCES tenants(id);
ALTER TABLE api_keys ADD COLUMN tenant_id TEXT NOT NULL REFERENCES tenants(id);
ALTER TABLE sso_connections ADD COLUMN tenant_id TEXT NOT NULL REFERENCES tenants(id);
ALTER TABLE audit_logs ADD COLUMN tenant_id TEXT NOT NULL REFERENCES tenants(id);
-- (repeat for all 12 tables)
```

Unique constraints change:
```sql
-- Before: UNIQUE(email)
-- After:  UNIQUE(tenant_id, email)
-- Same user email can exist in different tenants

-- Before: UNIQUE(name) on roles
-- After:  UNIQUE(tenant_id, name) on roles
```

### Store interface changes

Every method that queries or writes data needs a tenant context. Two approaches:

**Option A: Add tenantID parameter to every method**
```go
GetUserByEmail(ctx context.Context, tenantID, email string) (*User, error)
ListUsers(ctx context.Context, tenantID string, opts ListUsersOpts) ([]*User, error)
```
- Pro: Explicit, impossible to forget
- Con: 40+ method signatures change, massive diff

**Option B: Extract tenantID from context**
```go
// Middleware sets tenant in context
ctx = context.WithValue(ctx, TenantIDKey, tenantID)

// Store reads from context
func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
    tenantID := TenantFromContext(ctx)
    // SELECT ... WHERE tenant_id = ? AND email = ?
}
```
- Pro: Interface stays the same, self-hosted passes empty context (no tenant filtering)
- Con: Implicit, easy to forget in new queries

**Recommendation:** Option B. It preserves the interface for self-hosted (where tenant context is always empty/ignored) and avoids a 40-method interface rewrite.

### Tenant extraction middleware

Cloud requests carry tenant identity via:
1. **Subdomain:** `acme.sharkauth.cloud` → tenant slug = `acme`
2. **Header:** `X-Tenant-ID: tenant_abc123` (for API-to-API calls)
3. **API key scope:** Each API key is scoped to a tenant

Middleware extracts tenant, validates it exists, and puts it in context. Every downstream query is automatically scoped.

### Impact on self-hosted

None if using Option B. Self-hosted builds skip the tenant middleware. Context has no tenant ID. Store queries have no `WHERE tenant_id = ?` clause. Zero behavioral change.

---

## Moderate Risk: Dashboard Auth Model

**Severity:** Moderate — awkward but functional
**Effort:** 2-3 days

Current admin auth uses M2M API keys (`sk_live_*` with `["*"]` scope). This works for:
- Programmatic access (server-to-server)
- Self-hosted admins who copy-paste a key

It does NOT work well for:
- Cloud dashboard where humans log in interactively
- Per-tenant admin roles (owner, editor, viewer)
- Audit trail that shows "Jane from Acme Corp" not "key_abc123"

### What Cloud needs

1. **Admin user login** — session-based auth for dashboard, not API keys
2. **Dashboard roles** — owner (full access), editor (CRUD users/roles), viewer (read-only)
3. **Tenant-scoped keys** — an API key sees only its tenant's data
4. **Invite flow** — owner invites team members to the dashboard

### Implementation

- Extend `SessionManager` to support admin sessions (same cookie, different `auth_method`)
- Add `dashboard_users` table (or reuse `users` with a `is_dashboard_admin` flag per tenant)
- New endpoints: `POST /api/v1/admin/login`, `POST /api/v1/admin/invite`
- RBAC already exists — create dashboard-specific roles (`dashboard:admin`, `dashboard:editor`, `dashboard:viewer`)

### Impact on self-hosted

Self-hosted keeps API key auth. The admin login flow is cloud-only.

---

## Moderate Risk: SQLite → Postgres Migration

**Severity:** Low-moderate — contained work
**Effort:** 2-3 days

The storage interface (`storage.Store`) is properly abstracted. A Postgres implementation is a new file (`internal/storage/postgres.go`) that satisfies the same interface.

### SQLite-specific constructs to adapt

| SQLite pattern | Postgres equivalent | Files affected |
|----------------|-------------------|----------------|
| `INTEGER DEFAULT 0` for booleans | `BOOLEAN DEFAULT FALSE` | Schema, all scan functions |
| `TEXT` for timestamps | `TIMESTAMPTZ` | Schema, all scan functions |
| `email LIKE ?` (case-sensitive) | `email ILIKE ?` | `sqlite.go:92` (user search) |
| `PRAGMA journal_mode=WAL` | Not needed (Postgres uses WAL by default) | `sqlite.go:33` |
| `PRAGMA foreign_keys=ON` | Not needed (Postgres enforces by default) | `sqlite.go:35` |
| `goose.SetDialect("sqlite3")` | `goose.SetDialect("postgres")` | `storage/migrations.go:17` |
| `scopes LIKE '%"*"%'` (JSON in TEXT) | `scopes @> '["*"]'::jsonb` | `sqlite.go:879` |
| `modernc.org/sqlite` driver | `github.com/lib/pq` or `pgx` | `sqlite.go` imports |

### Migration path

1. Create `internal/storage/postgres.go` implementing `Store`
2. Create `cmd/shark/migrations/postgres/` with Postgres-native migrations
3. Config chooses driver: `storage.driver: "sqlite"` (default) or `storage.driver: "postgres"`
4. Factory function: `storage.New(driver, dsn)` returns the right implementation

### Impact on self-hosted

None. SQLite remains the default. Postgres is opt-in via config.

---

## Low Risk: Hard-Coded Paths & Embedded Assets

**Severity:** Low
**Effort:** A few hours

- `go:embed` for migrations and email templates is fine — Cloud can use the same embedded assets or load from a configurable path
- Default `storage.path: "./data/sharkauth.db"` is relative — already overridable via env var `SHARKAUTH_STORAGE__PATH`
- Admin dashboard route (`/admin/*`) serves a placeholder — Cloud replaces this with Next.js on a separate domain

No action needed for Cloud. These are self-hosted concerns only.

---

## Per-Tenant Configuration

Cloud tenants need their own settings. The current YAML config is global. Cloud needs per-tenant overrides stored in the database.

### What's tenant-specific

| Setting | Self-hosted | Cloud |
|---------|-------------|-------|
| OAuth providers (Google, GitHub, etc.) | YAML config | Per-tenant DB config |
| SMTP / email sender | YAML config | Shared (SharkAuth sends from our domain) or per-tenant |
| CORS origins | YAML config | Per-tenant (each tenant's frontend origin) |
| Session lifetime | YAML config | Per-tenant |
| Password policy | YAML config | Per-tenant |
| MFA issuer name | YAML config | Per-tenant (shows tenant's app name in authenticator) |
| Passkey RP ID/origin | YAML config | Per-tenant (each tenant's domain) |
| Branding (logo, colors) | N/A | Per-tenant |

### Implementation

The `tenants.config` JSON column stores overrides. At request time:
1. Load global config from YAML (defaults)
2. Load tenant config from DB
3. Merge: tenant values override global defaults
4. Pass merged config to handlers

This is the `TenantConfig` pattern — widely used in multi-tenant SaaS.

---

## API Surface for Cloud

The current REST API is sufficient for a Next.js dashboard to consume. Both dashboards (Svelte self-hosted, Next.js cloud) call the same `/api/v1/` endpoints.

### Endpoints Cloud needs that don't exist yet

| Endpoint | Purpose | Depends on |
|----------|---------|------------|
| `POST /api/v1/tenants` | Create tenant | Tenant isolation |
| `GET /api/v1/tenants` | List tenants (super-admin) | Tenant isolation |
| `GET /api/v1/tenants/{id}` | Get tenant details | Tenant isolation |
| `PATCH /api/v1/tenants/{id}` | Update tenant config | Tenant isolation |
| `DELETE /api/v1/tenants/{id}` | Delete tenant + all data | Tenant isolation |
| `POST /api/v1/admin/login` | Dashboard admin login | Dashboard auth model |
| `POST /api/v1/admin/invite` | Invite team member | Dashboard auth model |
| `GET /api/v1/admin/stats` | Overview metrics | Already in DASHBOARD.md spec |
| `GET /api/v1/admin/health` | System diagnostics | Already in DASHBOARD.md spec |
| `GET /api/v1/admin/sessions` | All active sessions | Already in DASHBOARD.md spec |

### Endpoints that work for both today

Everything else — user CRUD, RBAC, API keys, audit logs, SSO connections, auth flows. Once tenant isolation is added, these automatically scope to the requesting tenant.

---

## Strategic Decision: Fork vs. Monorepo

### Option A: Fork (separate repositories)
- `shark-auth/shark` — self-hosted (Go + Svelte + SQLite)
- `shark-auth/shark-cloud` — cloud (Go + Next.js + Postgres)

**Pro:** Each product moves independently. No feature flags.
**Con:** Bug fixes must be cherry-picked between repos. Divergence over time.

### Option B: Monorepo with build tags
- `shark-auth/shark` — single repo
- `go build -tags cloud` includes Postgres driver, Redis cache, tenant middleware
- `go build` (default) produces the self-hosted binary

**Pro:** One codebase, one test suite, fixes apply everywhere.
**Con:** Build complexity. Risk of cloud concerns leaking into self-hosted code.

### Recommendation: Monorepo until Next.js diverges

Keep everything in one repo. The Go backend can serve both models via build tags or config. Fork only when the Next.js dashboard becomes its own significant codebase (at which point it's a separate frontend repo, not a backend fork).

```
shark-auth/shark/           ← Go backend (both modes)
  internal/storage/sqlite.go
  internal/storage/postgres.go
  internal/cache/memory.go
  internal/cache/redis.go
  admin/                    ← Svelte dashboard (self-hosted)

shark-auth/shark-dashboard/ ← Next.js dashboard (cloud)
```

The backend is shared. The dashboards are separate repos. This avoids the "two Go codebases" maintenance trap.

---

## Implementation Roadmap

### Phase 1: Ship self-hosted v0.1.0 (now → April 27)
- Svelte dashboard
- Current SQLite backend
- All issues resolved
- **No cloud work yet**

### Phase 2: Cache abstraction (post-launch, ~1 week)
- `Cache` interface + memory implementation (preserves self-hosted behavior)
- Redis implementation
- Migrate all 5 in-memory stores
- **Unblocks:** multi-instance deployment

### Phase 3: Postgres store (1 week)
- `internal/storage/postgres.go`
- Postgres migrations
- Config-driven driver selection
- **Unblocks:** managed database hosting

### Phase 4: Tenant isolation (1-2 weeks)
- `tenants` table + CRUD API
- `tenant_id` on all tables
- Context-based tenant extraction middleware
- Per-tenant config storage
- **Unblocks:** multi-tenant Cloud

### Phase 5: Cloud dashboard auth (1 week)
- Admin user login (session-based)
- Dashboard roles (owner/editor/viewer)
- Tenant-scoped API keys
- Team invite flow
- **Unblocks:** interactive Cloud dashboard

### Phase 6: Next.js dashboard (2-3 weeks)
- Consumes same REST API as Svelte dashboard
- Adds: tenant management, billing integration, onboarding flow
- **Ships:** Shark Cloud beta

### Total: ~7-9 weeks from self-hosted launch to Cloud beta

---

## Decisions to Make Now (Before Dashboard)

These decisions affect the Svelte dashboard work happening right now:

1. **Dashboard must talk to the API via REST only.** No direct DB queries, no server-side Go rendering of dashboard pages. The Next.js dashboard needs the same API surface.

2. **Build the `GET /admin/stats` and `GET /admin/health` endpoints now.** Both dashboards need them. Don't hardcode dashboard metrics with direct SQL.

3. **Don't embed tenant-specific assumptions in the Svelte dashboard.** It should work with a single global namespace (self-hosted) today. Cloud adds tenant scoping later.

4. **API key auth for the Svelte dashboard is fine for v0.1.0.** Cloud will add session-based admin login later. Don't over-engineer the auth model for the first release.

---

*This document should be reviewed after v0.1.0 ships and before Cloud development begins.*
