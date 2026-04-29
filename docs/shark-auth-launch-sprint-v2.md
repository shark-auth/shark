# Shark Auth â€” Launch Sprint Spec (v2)

**Ship date:** Sunday, April 27, 2026
**Time budget:** ~17 days (evenings + three weekends)
**Rule:** If it's not on this list, it doesn't exist yet.

> **What changed from v1:** Market research showed passkeys are approaching table stakes, REST-only is a dealbreaker for most devs, single-provider OAuth limits addressable market, and M2M tokens are increasingly expected. This spec adds passkeys/WebAuthn, multi-provider social OAuth (Google, GitHub, Apple, Discord), magic links, M2M API keys, and a TypeScript SDK â€” while keeping the single-binary philosophy intact.

---

## What ships at launch

A single Go binary that handles signup, login, sessions, OAuth, MFA, passkeys, magic links, RBAC, SSO, M2M tokens â€” with a working Auth0 migration CLI, an embedded admin dashboard, and a TypeScript SDK. No "shipping soon." Every feature on the landing page works. No separate frontend deploy. `shark serve` and visit `:8080/admin`.

### The binary does exactly this:

1. **Email/password signup + login** (Argon2id hashing)
2. **Passkey/WebAuthn login** (FIDO2-compliant, resident credentials, platform + cross-platform authenticators)
3. **Magic link login** (email-based passwordless â€” token via SMTP, click to authenticate)
4. **Server-side sessions** (encrypted cookies, no JWT complexity)
5. **Social OAuth login** â€” Google, GitHub, Apple, Discord (four providers, generic handler pattern)
6. **MFA â€” TOTP** (Google Authenticator / Authy compatible, recovery codes)
7. **RBAC** (roles, permissions, role assignment, middleware enforcement)
8. **SSO â€” OIDC provider** (SharkAuth as an OIDC IdP)
9. **SSO â€” SAML SP** (beta â€” connect to enterprise IdPs like Okta/Azure AD)
10. **M2M API keys** (service-to-service auth â€” scoped, rotatable, rate-limit-aware)
11. **Auth0 user import** (read their export JSON, verify bcrypt hashes, rehash on first login)
12. **Audit logs** (every auth event recorded â€” login, signup, MFA, role changes, SSO, API keys. Filterable, exportable, retention-configurable)
13. **REST API** for everything above
14. **TypeScript SDK** (`@sharkauth/js`) â€” fetch-based, zero-dependency, works in Node/browser/edge
15. **YAML config** with env var overrides
16. **SQLite storage** (embedded, zero-config)
17. **Docker image** (one container, done)
18. **Admin dashboard** (Svelte, embedded in the binary â€” user management, sessions, MFA, passkeys, RBAC, SSO connections, API keys, audit logs, migration status)
19. **Automated test suite** (Go unit + integration tests with 60%+ coverage, SDK tests with Vitest, CI pipeline with gosec linting)

### What does NOT ship at launch:

- No organizations/multi-tenancy (post-launch â€” highest priority after launch)
- No OIDC client federation (post-launch â€” this is "Login with X enterprise IdP via OIDC")
- No agent identity / MCP auth (later â€” emerging standard, not yet table stakes)
- No Clerk/Firebase/Cognito migration (later)
- No Postgres mode (later)
- No React/Next.js component library (later â€” SDK ships first, pre-built UI components follow)

---

## Pricing (ships on sharkauth.com)

### Philosophy

The binary is the product. Cloud is a convenience layer, not a feature gate. Self-hosted is the free tier. Every auth feature ships in the binary for $0. Cloud sells operational burden â€” managed infra, SLA, support â€” not features.

### Tiers

| Tier | Price | MAU | Key Value |
|------|-------|-----|-----------|
| **Self-Hosted** | $0 forever | Unlimited | Full feature parity. Your infra, your data. Community support (Discord). |
| **Starter** | $19/mo | 50,000 | Managed cloud. Custom domain. White-label. Email support. |
| **Growth** | $49/mo | 150,000 | Priority support (12h). Migration help. 90-day audit retention. Webhooks. |
| **Scale** | $149/mo | 500,000 | Dedicated support + Slack. 99.9% SLA. 1-year audit retention. HA/multi-region. |

Overage: $0.003/MAU past tier limit.

14-day free cloud trial on Starter. No credit card required.

### What's included on EVERY tier (no exceptions, no add-ons):

- Passkeys / WebAuthn
- Magic links
- MFA / TOTP
- SSO (SAML + OIDC)
- RBAC
- M2M API keys
- Audit logs
- Session control
- Unlimited seats
- Migration engine
- TypeScript SDK

### Competitive Reality at Scale

| MAU | Clerk Pro | Auth0 | SharkAuth Cloud | Self-Hosted |
|------|-----------|-------|-----------------|-------------|
| 50K | $25 | ~$3,500 | $19 | $0 |
| 100K | $1,025 | ~$6,300 | $19 | $0 |
| 200K | $2,825 | ~$13,500 | $49 | $0 |
| 500K | $8,225 | ~$35,000 | $149 | $0 |

---

## Data Model (SQLite)

```sql
-- Core tables

CREATE TABLE users (
    id            TEXT PRIMARY KEY,     -- "usr_" + nanoid
    email         TEXT UNIQUE NOT NULL,
    email_verified INTEGER DEFAULT 0,
    password_hash TEXT,                 -- null for OAuth-only / passkey-only users
    hash_type     TEXT DEFAULT 'argon2id',
    name          TEXT,
    avatar_url    TEXT,
    mfa_enabled   INTEGER DEFAULT 0,   -- whether user has MFA active
    mfa_secret    TEXT,                 -- TOTP shared secret (encrypted)
    mfa_verified  INTEGER DEFAULT 0,   -- whether MFA setup was completed
    metadata      TEXT DEFAULT '{}',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip         TEXT,
    user_agent TEXT,
    mfa_passed INTEGER DEFAULT 0,      -- session has completed MFA challenge
    auth_method TEXT DEFAULT 'password', -- "password", "passkey", "magic_link", "oauth", "sso"
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE oauth_accounts (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,          -- "google", "github", "apple", "discord"
    provider_id TEXT NOT NULL,
    email       TEXT,
    access_token  TEXT,
    refresh_token TEXT,
    created_at  TEXT NOT NULL,
    UNIQUE(provider, provider_id)
);

CREATE TABLE migrations (
    id          TEXT PRIMARY KEY,
    source      TEXT NOT NULL,
    status      TEXT NOT NULL,
    users_total INTEGER DEFAULT 0,
    users_imported INTEGER DEFAULT 0,
    errors      TEXT DEFAULT '[]',
    created_at  TEXT NOT NULL,
    completed_at TEXT
);

-- Passkey / WebAuthn tables

CREATE TABLE passkey_credentials (
    id              TEXT PRIMARY KEY,       -- "pk_" + nanoid
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id   BLOB NOT NULL UNIQUE,   -- raw credential ID from authenticator
    public_key      BLOB NOT NULL,          -- COSE-encoded public key
    aaguid          TEXT,                   -- authenticator attestation GUID
    sign_count      INTEGER DEFAULT 0,      -- replay attack protection
    name            TEXT,                   -- user-assigned label ("MacBook Touch ID")
    transports      TEXT DEFAULT '[]',      -- JSON array: ["internal","usb","ble","nfc"]
    backed_up       INTEGER DEFAULT 0,      -- BS flag from authenticator
    created_at      TEXT NOT NULL,
    last_used_at    TEXT
);

-- Magic link tables

CREATE TABLE magic_link_tokens (
    id         TEXT PRIMARY KEY,
    email      TEXT NOT NULL,
    token_hash TEXT NOT NULL,               -- SHA-256 of the token sent via email
    used       INTEGER DEFAULT 0,
    expires_at TEXT NOT NULL,               -- short-lived: 10 minutes
    created_at TEXT NOT NULL
);

-- MFA tables

CREATE TABLE mfa_recovery_codes (
    id      TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code    TEXT NOT NULL,              -- hashed recovery code
    used    INTEGER DEFAULT 0,
    created_at TEXT NOT NULL
);

-- RBAC tables

CREATE TABLE roles (
    id          TEXT PRIMARY KEY,       -- "role_" + nanoid
    name        TEXT UNIQUE NOT NULL,   -- "admin", "editor", "viewer"
    description TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE TABLE permissions (
    id       TEXT PRIMARY KEY,          -- "perm_" + nanoid
    action   TEXT NOT NULL,             -- "read", "write", "delete"
    resource TEXT NOT NULL,             -- "users", "posts", "billing"
    created_at TEXT NOT NULL,
    UNIQUE(action, resource)
);

CREATE TABLE role_permissions (
    role_id       TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- SSO tables

CREATE TABLE sso_connections (
    id           TEXT PRIMARY KEY,      -- "sso_" + nanoid
    type         TEXT NOT NULL,         -- "saml" or "oidc"
    name         TEXT NOT NULL,         -- "Okta Production", "Azure AD"
    domain       TEXT,                  -- email domain for auto-routing
    -- SAML fields
    saml_idp_url       TEXT,            -- IdP SSO URL
    saml_idp_cert      TEXT,            -- IdP x509 certificate
    saml_sp_entity_id  TEXT,            -- our entity ID
    saml_sp_acs_url    TEXT,            -- our ACS callback
    -- OIDC fields
    oidc_issuer        TEXT,            -- IdP issuer URL
    oidc_client_id     TEXT,
    oidc_client_secret TEXT,
    enabled    INTEGER DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE sso_identities (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id   TEXT NOT NULL REFERENCES sso_connections(id) ON DELETE CASCADE,
    provider_sub    TEXT NOT NULL,      -- subject ID from IdP
    created_at      TEXT NOT NULL,
    UNIQUE(connection_id, provider_sub)
);

-- M2M API key tables

CREATE TABLE api_keys (
    id           TEXT PRIMARY KEY,      -- "key_" + nanoid
    name         TEXT NOT NULL,         -- "backend-service", "cron-worker"
    key_hash     TEXT NOT NULL UNIQUE,  -- SHA-256 of the full key (prefix "sk_" + random)
    key_prefix   TEXT NOT NULL,         -- first 8 chars for identification in logs/dashboard
    scopes       TEXT DEFAULT '[]',     -- JSON array: ["users:read","users:write","roles:read"]
    rate_limit   INTEGER DEFAULT 1000,  -- requests per hour, 0 = unlimited
    expires_at   TEXT,                  -- null = never expires
    last_used_at TEXT,
    created_at   TEXT NOT NULL,
    revoked_at   TEXT                   -- soft-delete: set timestamp to revoke
);

-- Audit log tables

CREATE TABLE audit_logs (
    id         TEXT PRIMARY KEY,          -- "aud_" + nanoid
    actor_id   TEXT,                      -- user or API key ID (null for system events)
    actor_type TEXT DEFAULT 'user',       -- "user", "api_key", "system"
    action     TEXT NOT NULL,             -- "user.login", "user.signup", "mfa.enabled", etc.
    target_type TEXT,                     -- "user", "role", "session", "sso_connection", "api_key"
    target_id  TEXT,                      -- ID of the affected resource
    ip         TEXT,
    user_agent TEXT,
    metadata   TEXT DEFAULT '{}',        -- JSON: provider name, error reason, old/new values, etc.
    status     TEXT DEFAULT 'success',   -- "success" or "failure"
    created_at TEXT NOT NULL
);

CREATE INDEX idx_audit_logs_actor   ON audit_logs(actor_id, created_at);
CREATE INDEX idx_audit_logs_action  ON audit_logs(action, created_at);
CREATE INDEX idx_audit_logs_target  ON audit_logs(target_id, created_at);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
```

**Audit log events captured:**

| Action | Trigger |
|--------|---------|
| `user.signup` | New account created |
| `user.login` | Successful password login |
| `user.login_failed` | Bad password / unknown email |
| `user.logout` | Session destroyed |
| `user.deleted` | Admin deletes user |
| `oauth.login` | OAuth login (metadata: provider) |
| `passkey.registered` | New passkey added |
| `passkey.login` | Passkey authentication |
| `passkey.deleted` | Passkey removed |
| `magic_link.sent` | Magic link email sent |
| `magic_link.verified` | Magic link used |
| `mfa.enrolled` | TOTP setup started |
| `mfa.enabled` | TOTP verified / active |
| `mfa.disabled` | MFA turned off |
| `mfa.challenge_passed` | Correct TOTP code |
| `mfa.challenge_failed` | Wrong TOTP code |
| `mfa.recovery_used` | Recovery code consumed |
| `role.created` | New role |
| `role.updated` | Role permissions changed |
| `role.deleted` | Role removed |
| `role.assigned` | Role assigned to user |
| `role.unassigned` | Role removed from user |
| `sso.connection_created` | SAML/OIDC connection added |
| `sso.connection_updated` | Connection config changed |
| `sso.connection_deleted` | Connection removed |
| `sso.login` | SSO authentication (metadata: connection_id) |
| `api_key.created` | New M2M key |
| `api_key.rotated` | Key rotated |
| `api_key.revoked` | Key revoked |
| `migration.started` | Auth0 import kicked off |
| `migration.completed` | Import finished (metadata: counts) |
| `session.revoked` | Admin or user revoked a session |

**What is NOT logged:** passwords, tokens, secrets, full API keys, request/response bodies, PII beyond actor/target IDs.

**Retention:** Self-hosted = unlimited (user controls their DB). Cloud Starter = 30 days. Cloud Growth = 90 days. Cloud Scale = 1 year. Retention enforced by a background goroutine that runs `DELETE FROM audit_logs WHERE created_at < ?` every hour.

---

## API Endpoints

### Auth (core)

```
POST   /api/v1/auth/signup              â€” Create user + session
POST   /api/v1/auth/login               â€” Verify password, return session (or MFA challenge)
POST   /api/v1/auth/logout              â€” Destroy session
GET    /api/v1/auth/me                  â€” Current user from session
```

### Social OAuth (generic pattern)

```
GET    /api/v1/auth/oauth/:provider              â€” Redirect to provider (google, github, apple, discord)
GET    /api/v1/auth/oauth/:provider/callback     â€” Handle OAuth callback, create/link user + session
```

**Supported providers:** `google`, `github`, `apple`, `discord`

### Passkeys / WebAuthn

```
POST   /api/v1/auth/passkey/register/begin    â€” Generate registration options (PublicKeyCredentialCreationOptions)
POST   /api/v1/auth/passkey/register/finish   â€” Verify attestation, store credential
POST   /api/v1/auth/passkey/login/begin       â€” Generate authentication options (PublicKeyCredentialRequestOptions)
POST   /api/v1/auth/passkey/login/finish      â€” Verify assertion, create session
GET    /api/v1/auth/passkey/credentials        â€” List user's registered passkeys
DELETE /api/v1/auth/passkey/credentials/:id    â€” Remove a passkey
PATCH  /api/v1/auth/passkey/credentials/:id    â€” Rename a passkey
```

**Passkey registration flow:**

```
1. User is logged in (via password, OAuth, or magic link)
2. POST /passkey/register/begin
   â†’ Server generates challenge, returns PublicKeyCredentialCreationOptions
   â†’ Options include: rp.id (from config), user.id, user.name, user.displayName
   â†’ Options include: pubKeyCredParams [ES256, RS256], authenticatorSelection
3. Client calls navigator.credentials.create() with those options
4. POST /passkey/register/finish with attestation response
   â†’ Server verifies origin, challenge, attestation
   â†’ Stores credential_id, public_key, sign_count, aaguid, transports
   â†’ Returns { "credential_id": "pk_xxx", "name": "Unnamed passkey" }
```

**Passkey login flow:**

```
1. POST /passkey/login/begin (optionally with { "email": "..." } for non-discoverable)
   â†’ Server generates challenge, returns PublicKeyCredentialRequestOptions
   â†’ If email provided: includes allowCredentials for that user's passkeys
   â†’ If no email: empty allowCredentials (discoverable/resident key flow)
2. Client calls navigator.credentials.get() with those options
3. POST /passkey/login/finish with assertion response
   â†’ Server looks up credential by credential_id
   â†’ Verifies signature against stored public_key
   â†’ Verifies sign_count > stored value (replay protection)
   â†’ Updates sign_count + last_used_at
   â†’ Creates session with auth_method="passkey", mfa_passed=1 (passkeys satisfy MFA)
   â†’ Returns session cookie
```

### Magic Links

```
POST   /api/v1/auth/magic-link/send       â€” Send magic link to email
GET    /api/v1/auth/magic-link/verify      â€” Verify token from email link, create session
```

**Magic link flow:**

```
1. POST /magic-link/send with { "email": "user@example.com" }
   â†’ Generate 32-byte random token
   â†’ Store SHA-256(token) in magic_link_tokens with 10-minute expiry
   â†’ Send email with link: https://yourapp.com/auth/magic?token=<token>
   â†’ If user doesn't exist: create account with email_verified=1
   â†’ Always return 200 (don't leak whether email exists)
2. GET /magic-link/verify?token=<token>
   â†’ Hash token, look up in magic_link_tokens
   â†’ Verify not expired, not used
   â†’ Mark as used
   â†’ Create or find user by email, set email_verified=1
   â†’ Create session with auth_method="magic_link"
   â†’ Redirect to configured success URL with session cookie set
```

### MFA

```
POST   /api/v1/auth/mfa/enroll          â€” Generate TOTP secret + QR URI
POST   /api/v1/auth/mfa/verify          â€” Confirm setup with first code
POST   /api/v1/auth/mfa/challenge       â€” Submit TOTP code during login
POST   /api/v1/auth/mfa/recovery        â€” Use a recovery code
DELETE /api/v1/auth/mfa                  â€” Disable MFA (requires current code)
GET    /api/v1/auth/mfa/recovery-codes   â€” Regenerate recovery codes
```

**MFA login flow:**

```
1. POST /auth/login with email + password
2. If MFA enabled â†’ 200 { "mfa_required": true, "session_token": "partial_xxx" }
   Session is created with mfa_passed=0 (can only hit /mfa/challenge)
3. POST /auth/mfa/challenge with TOTP code
4. If valid â†’ session upgraded to mfa_passed=1, full access granted
5. If user lost device â†’ POST /auth/mfa/recovery with recovery code
```

**Note:** Passkey login bypasses MFA â€” passkeys are phishing-resistant and satisfy AAL2. Sessions created via passkey have mfa_passed=1 automatically.

### RBAC

```
POST   /api/v1/roles                    â€” Create role
GET    /api/v1/roles                    â€” List roles
GET    /api/v1/roles/:id                â€” Get role with permissions
PUT    /api/v1/roles/:id                â€” Update role
DELETE /api/v1/roles/:id                â€” Delete role

POST   /api/v1/permissions              â€” Create permission
GET    /api/v1/permissions              â€” List permissions

POST   /api/v1/roles/:id/permissions    â€” Attach permissions to role
DELETE /api/v1/roles/:id/permissions/:pid â€” Detach permission from role

POST   /api/v1/users/:id/roles          â€” Assign role to user
DELETE /api/v1/users/:id/roles/:rid      â€” Remove role from user
GET    /api/v1/users/:id/roles           â€” List user's roles
GET    /api/v1/users/:id/permissions     â€” List user's effective permissions (resolved)

POST   /api/v1/auth/check               â€” Check permission: { "user_id", "action", "resource" } â†’ { "allowed": bool }
```

### SSO

```
POST   /api/v1/sso/connections           â€” Create SSO connection (SAML or OIDC config)
GET    /api/v1/sso/connections           â€” List connections
GET    /api/v1/sso/connections/:id       â€” Get connection details
PUT    /api/v1/sso/connections/:id       â€” Update connection
DELETE /api/v1/sso/connections/:id       â€” Delete connection

GET    /api/v1/sso/saml/:connection_id/metadata  â€” SP metadata XML (for IdP setup)
POST   /api/v1/sso/saml/:connection_id/acs       â€” SAML ACS endpoint (receives assertion)

GET    /api/v1/sso/oidc/:connection_id/auth      â€” OIDC authorization redirect
GET    /api/v1/sso/oidc/:connection_id/callback   â€” OIDC callback

GET    /api/v1/auth/sso?email=user@corp.com      â€” Auto-route: looks up domain â†’ redirect to correct IdP
```

### M2M API Keys

```
POST   /api/v1/api-keys                 â€” Create API key (returns full key ONCE, then only prefix shown)
GET    /api/v1/api-keys                 â€” List API keys (prefix + metadata only, never full key)
GET    /api/v1/api-keys/:id             â€” Get API key details
PATCH  /api/v1/api-keys/:id             â€” Update name, scopes, rate_limit, expires_at
DELETE /api/v1/api-keys/:id             â€” Revoke API key (soft-delete, sets revoked_at)
POST   /api/v1/api-keys/:id/rotate     â€” Rotate: generate new key, revoke old one atomically
```

**API key authentication:**
```
Authorization: Bearer sh_xxx(32 random bytes base62-encoded)

â†’ Server hashes the key with SHA-256
â†’ Looks up key_hash in api_keys table
â†’ Verifies not revoked (revoked_at IS NULL)
â†’ Verifies not expired (expires_at IS NULL OR expires_at > now)
â†’ Checks scope against requested action
â†’ Enforces rate limit (in-memory token bucket per key_hash)
â†’ Updates last_used_at
â†’ Request proceeds with the key's scopes as the permission set
```

**Key format:** `sk_live_` + 32 random bytes base62-encoded (48 chars total). The `sk_live_` prefix makes keys easily identifiable in logs and secret scanners.

### Audit Logs

```
GET    /api/v1/audit-logs                â€” List audit logs (paginated, filterable)
GET    /api/v1/audit-logs/:id            â€” Get single audit log entry
GET    /api/v1/users/:id/audit-logs      â€” Audit logs for a specific user (as actor or target)
POST   /api/v1/audit-logs/export         â€” Export logs as JSON/CSV (date range required)
```

**Query parameters for `GET /api/v1/audit-logs`:**

| Param | Example | Description |
|-------|---------|-------------|
| `action` | `user.login` | Filter by event type (supports comma-separated: `user.login,user.signup`) |
| `actor_id` | `usr_abc123` | Filter by who did it |
| `target_id` | `usr_xyz789` | Filter by what was affected |
| `status` | `failure` | `success` or `failure` |
| `ip` | `192.168.1.1` | Filter by IP address |
| `from` | `2026-04-01T00:00:00Z` | Start of date range |
| `to` | `2026-04-07T23:59:59Z` | End of date range |
| `limit` | `50` | Page size (default 50, max 200) |
| `cursor` | `aud_xxx` | Cursor-based pagination (ID of last item) |

### Admin + Migration

```
GET    /api/v1/users                     â€” List users (paginated)
GET    /api/v1/users/:id                 â€” Get user
POST   /api/v1/migrate/auth0             â€” Upload Auth0 export JSON
GET    /api/v1/migrate/:id               â€” Migration status
GET    /healthz                          â€” Health check
```

---

## Config (expanded)

```yaml
# sharkauth.yaml
server:
  port: 8080
  secret: "${SHARKAUTH_SECRET}"
  base_url: "https://auth.myapp.com"    # required for passkeys (rp.id) and magic links

storage:
  path: "./data/sharkauth.db"

auth:
  session_lifetime: "30d"
  password_min_length: 8

passkeys:
  rp_name: "My App"                     # relying party display name
  rp_id: ""                             # defaults to hostname from base_url
  origin: ""                            # defaults to base_url
  attestation: "none"                   # "none", "indirect", "direct" (none = most compatible)
  resident_key: "preferred"             # "discouraged", "preferred", "required"
  user_verification: "preferred"        # "discouraged", "preferred", "required"

magic_link:
  token_lifetime: "10m"                 # how long magic link tokens are valid
  redirect_url: "http://localhost:3000/auth/callback"  # where to redirect after verification

smtp:
  host: "${SMTP_HOST}"
  port: 587
  username: "${SMTP_USER}"
  password: "${SMTP_PASS}"
  from: "auth@myapp.com"
  from_name: "My App"

mfa:
  issuer: "SharkAuth"                    # shows in authenticator app
  recovery_codes: 10                     # number of codes generated

social:
  google:
    client_id: "${GOOGLE_CLIENT_ID}"
    client_secret: "${GOOGLE_CLIENT_SECRET}"
  github:
    client_id: "${GITHUB_CLIENT_ID}"
    client_secret: "${GITHUB_CLIENT_SECRET}"
  apple:
    client_id: "${APPLE_CLIENT_ID}"         # Services ID
    team_id: "${APPLE_TEAM_ID}"
    key_id: "${APPLE_KEY_ID}"
    private_key_path: "./apple_auth_key.p8" # or APPLE_PRIVATE_KEY env var
  discord:
    client_id: "${DISCORD_CLIENT_ID}"
    client_secret: "${DISCORD_CLIENT_SECRET}"

sso:
  saml:
    sp_entity_id: "https://auth.myapp.com"
  oidc:
    # Per-connection config via API, not static config

api_keys:
  default_rate_limit: 1000              # requests per hour for new keys
  key_max_lifetime: "365d"              # max allowed expires_at (0 = unlimited)

audit:
  retention: "0"                          # "0" = keep forever (self-hosted default)
  cleanup_interval: "1h"                  # how often to purge expired logs
  # Cloud overrides: Starter=30d, Growth=90d, Scale=365d

admin:
  api_key: "${SHARKAUTH_ADMIN_KEY}"
```

---

## Project Structure (updated)

```
sharkauth/
â”œâ”€â”€ main.go
â”œâ”€â”€ cmd/
â”‚   â”œâ”€â”€ serve.go
â”‚   â””â”€â”€ migrate.go
â”œâ”€â”€ internal/
â”‚   â”œâ”€â”€ config/
â”‚   â”‚   â””â”€â”€ config.go
â”‚   â”œâ”€â”€ storage/
â”‚   â”‚   â”œâ”€â”€ storage.go                   # Storage interface
â”‚   â”‚   â””â”€â”€ sqlite.go                    # SQLite implementation
â”‚   â”œâ”€â”€ auth/
â”‚   â”‚   â”œâ”€â”€ password.go                  # Argon2id + multi-hash
â”‚   â”‚   â”œâ”€â”€ session.go                   # Sessions + MFA-aware gating
â”‚   â”‚   â”œâ”€â”€ oauth.go                     # Generic OAuth handler + provider registry
â”‚   â”‚   â”œâ”€â”€ providers/                   # OAuth provider implementations
â”‚   â”‚   â”‚   â”œâ”€â”€ google.go
â”‚   â”‚   â”‚   â”œâ”€â”€ github.go
â”‚   â”‚   â”‚   â”œâ”€â”€ apple.go                 # Apple uses JWT client_secret
â”‚   â”‚   â”‚   â””â”€â”€ discord.go
â”‚   â”‚   â”œâ”€â”€ mfa.go                       # TOTP generation, verification, recovery codes
â”‚   â”‚   â”œâ”€â”€ passkey.go                   # WebAuthn registration + authentication (go-webauthn/webauthn)
â”‚   â”‚   â”œâ”€â”€ magiclink.go                 # Token generation, email sending, verification
â”‚   â”‚   â””â”€â”€ apikey.go                    # M2M API key CRUD, hashing, validation, rate limiting
â”‚   â”œâ”€â”€ email/
â”‚   â”‚   â”œâ”€â”€ sender.go                    # SMTP client wrapper
â”‚   â”‚   â””â”€â”€ templates/                   # HTML email templates
â”‚   â”‚       â”œâ”€â”€ magic_link.html
â”‚   â”‚       â””â”€â”€ verify_email.html
â”‚   â”œâ”€â”€ rbac/
â”‚   â”‚   â”œâ”€â”€ rbac.go                      # Role/permission CRUD
â”‚   â”‚   â””â”€â”€ middleware.go                # RequirePermission("action", "resource")
â”‚   â”œâ”€â”€ sso/
â”‚   â”‚   â”œâ”€â”€ saml.go                      # SAML SP: metadata, ACS, assertion parsing
â”‚   â”‚   â”œâ”€â”€ oidc.go                      # OIDC client: auth redirect, callback, token exchange
â”‚   â”‚   â””â”€â”€ connection.go                # SSO connection CRUD + domain routing
â”‚   â”œâ”€â”€ audit/
â”‚   â”‚   â”œâ”€â”€ audit.go                     # Log(), Query(), Cleanup() â€” core audit engine
â”‚   â”‚   â””â”€â”€ middleware.go                # HTTP middleware that auto-logs requests
â”‚   â”œâ”€â”€ user/
â”‚   â”‚   â””â”€â”€ user.go
â”‚   â”œâ”€â”€ migrate/
â”‚   â”‚   â””â”€â”€ auth0.go
â”‚   â”œâ”€â”€ testutil/                        # Shared test infrastructure
â”‚   â”‚   â”œâ”€â”€ db.go                        # NewTestDB (in-memory SQLite)
â”‚   â”‚   â”œâ”€â”€ server.go                    # TestServer (httptest + cookiejar)
â”‚   â”‚   â”œâ”€â”€ config.go                    # TestConfig with safe defaults
â”‚   â”‚   â”œâ”€â”€ factories.go                 # CreateUser, CreateRole, CreateAPIKey, etc.
â”‚   â”‚   â””â”€â”€ email.go                     # MemoryEmailSender (captures sent emails)
â”‚   â””â”€â”€ api/
â”‚       â”œâ”€â”€ router.go
â”‚       â”œâ”€â”€ auth_handlers.go             # Signup, login, logout, me
â”‚       â”œâ”€â”€ oauth_handlers.go            # Generic OAuth redirect + callback
â”‚       â”œâ”€â”€ passkey_handlers.go          # WebAuthn register/login begin/finish
â”‚       â”œâ”€â”€ magiclink_handlers.go        # Magic link send + verify
â”‚       â”œâ”€â”€ mfa_handlers.go              # Enroll, verify, challenge, recovery
â”‚       â”œâ”€â”€ rbac_handlers.go             # Roles, permissions, assignment, check
â”‚       â”œâ”€â”€ sso_handlers.go              # Connections, SAML ACS, OIDC callback
â”‚       â”œâ”€â”€ apikey_handlers.go           # M2M API key CRUD + rotate
â”‚       â”œâ”€â”€ audit_handlers.go            # Audit log list, detail, export
â”‚       â”œâ”€â”€ user_handlers.go             # Admin user endpoints
â”‚       â””â”€â”€ migrate_handlers.go          # Migration endpoints
â”œâ”€â”€ dashboard/                           # Svelte app, embedded in binary
â”‚   â”œâ”€â”€ src/
â”‚   â”‚   â”œâ”€â”€ routes/
â”‚   â”‚   â”‚   â”œâ”€â”€ +page.svelte             # Dashboard overview (user count, active sessions, auth method breakdown)
â”‚   â”‚   â”‚   â”œâ”€â”€ users/
â”‚   â”‚   â”‚   â”‚   â”œâ”€â”€ +page.svelte         # User list (search, paginate, filter by auth method)
â”‚   â”‚   â”‚   â”‚   â””â”€â”€ [id]/+page.svelte    # User detail (sessions, roles, MFA, passkeys, OAuth accounts)
â”‚   â”‚   â”‚   â”œâ”€â”€ sessions/+page.svelte    # Active sessions (revoke, per-device view, auth method column)
â”‚   â”‚   â”‚   â”œâ”€â”€ roles/
â”‚   â”‚   â”‚   â”‚   â”œâ”€â”€ +page.svelte         # Role list + create
â”‚   â”‚   â”‚   â”‚   â””â”€â”€ [id]/+page.svelte    # Role detail (edit permissions, assigned users)
â”‚   â”‚   â”‚   â”œâ”€â”€ sso/
â”‚   â”‚   â”‚   â”‚   â”œâ”€â”€ +page.svelte         # SSO connections list
â”‚   â”‚   â”‚   â”‚   â””â”€â”€ [id]/+page.svelte    # Connection config (SAML cert upload, OIDC fields)
â”‚   â”‚   â”‚   â”œâ”€â”€ api-keys/
â”‚   â”‚   â”‚   â”‚   â””â”€â”€ +page.svelte         # API key list, create, revoke, rotate, usage stats
â”‚   â”‚   â”‚   â”œâ”€â”€ audit/
â”‚   â”‚   â”‚   â”‚   â””â”€â”€ +page.svelte         # Audit log stream (filter by action/user/date, live tail)
â”‚   â”‚   â”‚   â””â”€â”€ migrations/
â”‚   â”‚   â”‚       â””â”€â”€ +page.svelte         # Migration history + trigger new import
â”‚   â”‚   â””â”€â”€ lib/
â”‚   â”‚       â”œâ”€â”€ api.ts                   # Fetch wrapper for internal API
â”‚   â”‚       â””â”€â”€ components/              # Shared UI components
â”‚   â”œâ”€â”€ package.json
â”‚   â””â”€â”€ svelte.config.js
â”œâ”€â”€ sdk/                                 # TypeScript SDK
â”‚   â”œâ”€â”€ src/
â”‚   â”‚   â”œâ”€â”€ index.ts                     # Main export: createSharkAuth(config)
â”‚   â”‚   â”œâ”€â”€ client.ts                    # HTTP client (fetch-based, works everywhere)
â”‚   â”‚   â”œâ”€â”€ auth.ts                      # signup, login, logout, me
â”‚   â”‚   â”œâ”€â”€ passkey.ts                   # registerPasskey, loginWithPasskey (wraps navigator.credentials)
â”‚   â”‚   â”œâ”€â”€ magic-link.ts               # sendMagicLink
â”‚   â”‚   â”œâ”€â”€ oauth.ts                     # getOAuthURL, handleCallback
â”‚   â”‚   â”œâ”€â”€ mfa.ts                       # enrollMFA, verifyMFA, challengeMFA
â”‚   â”‚   â””â”€â”€ types.ts                     # All TypeScript interfaces
â”‚   â”œâ”€â”€ package.json                     # @sharkauth/js
â”‚   â”œâ”€â”€ tsconfig.json
â”‚   â””â”€â”€ README.md                        # SDK-specific docs with code examples
â”œâ”€â”€ examples/
â”‚   â”œâ”€â”€ nextjs/                          # Next.js App Router example
â”‚   â”‚   â”œâ”€â”€ middleware.ts                # Auth middleware (session check)
â”‚   â”‚   â”œâ”€â”€ app/
â”‚   â”‚   â”‚   â”œâ”€â”€ login/page.tsx           # Login form (password + passkey + magic link + OAuth buttons)
â”‚   â”‚   â”‚   â””â”€â”€ dashboard/page.tsx       # Protected page
â”‚   â”‚   â””â”€â”€ lib/sharkauth.ts            # SDK initialization
â”‚   â””â”€â”€ react-spa/                       # Vite + React SPA example
â”‚       â”œâ”€â”€ src/
â”‚       â”‚   â”œâ”€â”€ auth/
â”‚       â”‚   â”‚   â”œâ”€â”€ LoginForm.tsx         # Full login component
â”‚       â”‚   â”‚   â””â”€â”€ PasskeyButton.tsx     # Passkey-specific component
â”‚       â”‚   â””â”€â”€ App.tsx
â”‚       â””â”€â”€ package.json
â”œâ”€â”€ sharkauth.yaml
â”œâ”€â”€ Dockerfile
â”œâ”€â”€ docker-compose.yml
â”œâ”€â”€ README.md
â””â”€â”€ go.mod
```

---

## TypeScript SDK Design (`@sharkauth/js`)

### API surface

```typescript
import { createSharkAuth } from '@sharkauth/js';

const auth = createSharkAuth({
  baseURL: 'https://auth.myapp.com',  // or http://localhost:8080
  // No API key needed for client-side operations â€” session cookies handle auth
});

// Email/password
await auth.signup({ email, password, name });
await auth.login({ email, password });
await auth.logout();
const user = await auth.me();

// Passkeys
const credential = await auth.passkey.register();       // wraps navigator.credentials.create()
const session = await auth.passkey.login();              // wraps navigator.credentials.get()
const passkeys = await auth.passkey.list();              // list user's passkeys
await auth.passkey.remove(credentialId);                 // delete a passkey
await auth.passkey.rename(credentialId, 'MacBook Pro');  // rename

// Magic links
await auth.magicLink.send({ email });                   // trigger email

// OAuth
const url = auth.oauth.getURL('google');                // returns redirect URL
const url = auth.oauth.getURL('github');

// MFA
const { secret, qrUri } = await auth.mfa.enroll();
await auth.mfa.verify({ code });                        // confirm setup
await auth.mfa.challenge({ code });                     // during login
await auth.mfa.useRecoveryCode({ code });

// Check auth state (middleware helper)
const { authenticated, user } = await auth.check();
```

### Design constraints

- **Zero dependencies** â€” uses native `fetch` only
- **Isomorphic** â€” works in browser, Node.js 18+, Deno, Bun, Cloudflare Workers, Vercel Edge
- **Cookie-based** â€” relies on `credentials: 'include'` for session management, no token handling
- **Passkey helpers** â€” wraps WebAuthn browser API with proper ArrayBufferâ†”base64url conversion
- **Tree-shakeable** â€” ESM-first with named exports
- **Type-safe** â€” full TypeScript types for all request/response shapes

### Build + publish

```bash
cd sdk/
npm run build    # tsup â†’ ESM + CJS + .d.ts
npm publish      # â†’ @sharkauth/js on npm
```

---

## Testing Strategy

### Philosophy

Test as you build â€” not as a separate phase. Every feature gets at minimum one integration test before moving to the next. Target 60% coverage by launch. Focus on flows that would cause the most damage if broken.

### Test Infrastructure (`internal/testutil/`)

Built on Day 1 (Saturday morning of Weekend 1), used everywhere:

- **`NewTestDB(t)`** â€” in-memory SQLite (`:memory:?_foreign_keys=on`). Each test gets its own DB. No cleanup. Tests run in parallel.
- **`TestServer`** â€” wraps `httptest.Server` + `http.Client` with `cookiejar.Jar`. Maintains session cookies across requests automatically. Helpers: `PostJSON()`, `Get()`, `DecodeJSON[T]()`.
- **`TestConfig`** â€” safe defaults for all config sections. Reduced Argon2id params (16MB memory, 1 iteration) so tests don't block on hashing.
- **Factory functions** â€” `CreateUser()`, `CreateUserWithRole()`, `CreateAndLogin()`, `CreateAPIKey()`, etc. One-liners that set up test state.
- **`MemoryEmailSender`** â€” implements `EmailSender` interface, captures sent emails in a slice. Used to test magic link flows without SMTP.
- **`Clock` interface** â€” injectable time source. Tests use `TestClock` with `Advance(d)` to test expiry without `time.Sleep`.

### What Gets Tested (and How)

| Area | Type | Approach |
|------|------|----------|
| Password hashing (argon2id + bcrypt compat) | Unit | Real crypto, reduced params. Test round-trip, wrong password, bcryptâ†’argon2id rehash. |
| TOTP generation/verification | Unit | Real `pquerna/otp`. Generate code at known time, verify. Test Â±1 step tolerance, reject old codes. |
| Recovery codes | Unit | Verify uniqueness, one-time use, hash comparison. |
| Session create/validate/expire/revoke | Integration | Real SQLite. Test cookie setting, expiry via Clock, rotation on login. |
| Signup â†’ login â†’ me â†’ logout flow | Integration | `TestServer` + cookiejar. Full HTTP round-trip. |
| MFA enroll â†’ verify â†’ login â†’ challenge â†’ upgrade | Integration | Generate real TOTP code from enrolled secret, verify partial session gates `/me`. |
| OAuth callback | Integration | Mock provider (Google/GitHub) with `httptest.Server` returning fake tokens. Override `TokenURL`/`UserInfoURL` in config. Test one provider, trust the generic handler pattern for the rest. |
| Passkey register/login begin | Integration | Test that `begin` endpoints return well-formed `PublicKeyCredentialCreationOptions`/`RequestOptions`. For `finish` endpoints, test error cases (bad challenge, expired). Trust `go-webauthn` library for crypto verification. Full passkey flow tested manually in browser. |
| Magic link send â†’ verify â†’ session | Integration | `MemoryEmailSender` captures email, extract token, verify endpoint creates session. Test one-time use (second verify = 400). Test expiry via Clock. |
| RBAC permission resolution | Unit | Table-driven tests. Admin wildcard, multiple roles merge, no roles = no access, specific action+resource matching. |
| `RequirePermission` middleware | Integration | Create user with role, hit protected endpoint. Create user without role, verify 403. |
| `POST /auth/check` | Integration | Exercise the permission check endpoint with various user/action/resource combos. |
| API key hash + validate | Unit | Generate key, hash, verify. Wrong key fails. Constant-time comparison (`crypto/subtle`). |
| API key scope enforcement | Integration | Key with `users:read` can GET users, gets 403 on POST roles. |
| API key rate limiting | Integration | Exhaust rate limit, verify 429 on next request. |
| Auth0 migration import | Integration | Load fixture JSON, verify users imported, bcrypt hashes verified, rehash on first login. |
| Audit log capture | Integration | Perform an action (login, role change), query audit log endpoint, verify event recorded with correct actor/target/action. |
| Session fixation | Security | Verify session token changes on login (not reused from signup). |
| Login brute force | Security | 10 wrong passwords â†’ 11th attempt (even correct) returns 429. |
| Token expiry enforcement | Security | Magic link + session expiry via Clock. Expired = rejected. |
| SSO OIDC callback | Integration | Mock OIDC provider. Test token exchange + user creation. |
| SSO SAML ACS | Integration | Test with pre-built SAML assertion XML (from Okta dev account). Verify user created/linked. |

### What Does NOT Get Tested (in the sprint)

- **Dashboard Svelte components** â€” internal admin tool, manual QA is enough. Add Playwright post-launch.
- **SAML XML parsing edge cases** â€” trust the SAML library, test your integration only.
- **Each OAuth provider separately** â€” all 4 use the generic handler. Test GitHub mock, trust the pattern.
- **Cookie encryption internals** â€” trust `gorilla/securecookie`. Test at HTTP level only.
- **Email HTML template rendering** â€” visual, not functional. Manual check.
- **Passkey `finish` crypto verification** â€” go-webauthn is conformance-tested.

### SDK Testing (TypeScript)

Stack: **Vitest + MSW (Mock Service Worker)**. MSW intercepts `fetch` at the network level â€” no real server needed.

- Test request formatting (correct URLs, methods, JSON bodies)
- Test response parsing (happy path + one error case per method)
- Test passkey helpers (`base64url â†” ArrayBuffer` conversion â€” pure functions)
- Test error handling (network errors, 4xx/5xx, expired sessions)
- `npm run typecheck` (tsc --noEmit) catches contract drift

### CI Pipeline (GitHub Actions)

```yaml
# .github/workflows/ci.yml
jobs:
  lint:        # golangci-lint with gosec (catches weak crypto, SQL injection)
  test-go:     # go test -race -coverprofile ./... â€” fail if <60% coverage
  test-sdk:    # npm run typecheck && npm test (vitest)
  build:       # compile Svelte â†’ embed in Go â†’ verify binary starts
```

### Estimated Effort

~8 hours total across 17 days, woven into each feature's implementation slot. Not a separate phase.

---

## 17-Day Sprint Plan

### Weekend 1 (April 11â€“12) â€” Core auth + OAuth

**Saturday â€” Scaffold + Auth + OAuth**

Morning (4h):
- [ ] `go mod init`, project scaffold
- [ ] Config loader (YAML + env vars, including new passkey/smtp/social sections)
- [ ] SQLite storage layer (connect, migrate full schema including new tables, CRUD)
- [ ] User model + create/get/list
- [ ] **`internal/testutil/` package** â€” `NewTestDB`, `TestServer`, `TestConfig`, factories, `MemoryEmailSender`

Afternoon (4h):
- [ ] Password hashing (Argon2id) + multi-hash verification (bcrypt compat)
- [ ] **Unit tests: password hash round-trip, wrong password, bcrypt compat, rehash detection**
- [ ] Session management (create, validate, revoke, cookie handling, auth_method tracking)
- [ ] Signup + login + logout + me API handlers
- [ ] **Integration test: signup â†’ login â†’ me â†’ logout â†’ login flow (TestServer + cookiejar)**
- [ ] **Audit log engine** (`internal/audit/audit.go`) â€” `Log()` writes to DB, `Query()` with filters, `Cleanup()` goroutine

Evening (3h):
- [ ] Generic OAuth handler: provider registry pattern, redirect + callback
- [ ] Google OAuth provider implementation
- [ ] GitHub OAuth provider implementation
- [ ] Admin endpoints behind API key
- [ ] **Audit middleware** â€” auto-log `user.signup`, `user.login`, `user.login_failed`, `user.logout`

**Sunday â€” More OAuth + Migration + MFA**

Morning (4h):
- [ ] Apple OAuth provider (JWT client_secret from .p8 key, id_token parsing)
- [ ] Discord OAuth provider
- [ ] **Integration test: OAuth callback with mock GitHub provider (httptest.Server)**
- [ ] Test all four OAuth flows end-to-end
- [ ] Error handling, input validation, HTTP status codes

Afternoon (4h):
- [ ] Auth0 migration: JSON parser, CLI command, API endpoint
- [ ] Transparent bcryptâ†’argon2id rehash on first login
- [ ] **Integration test: Auth0 import fixture â†’ user created â†’ bcrypt login â†’ rehash verified**
- [ ] MFA: TOTP secret generation (crypto/rand â†’ base32)
- [ ] MFA: QR URI builder (otpauth://totp/...)

Evening (3h):
- [ ] MFA: TOTP code validation (HMAC-SHA1, 30s window, Â±1 step tolerance)
- [ ] **Unit tests: TOTP verify, Â±1 step tolerance, reject 5-min-old code, recovery code uniqueness**
- [ ] MFA: Enroll + verify + challenge endpoints
- [ ] MFA: Recovery codes (generate 10, hash with bcrypt, one-time use)
- [ ] MFA: Login flow integration (partial session â†’ challenge â†’ upgrade)
- [ ] MFA: Disable endpoint (require current code to turn off)
- [ ] **Integration test: MFA enroll â†’ verify â†’ logout â†’ login â†’ mfa_required â†’ challenge â†’ /me works**
- [ ] Audit: wire `mfa.enrolled`, `mfa.enabled`, `mfa.disabled`, `mfa.challenge_passed/failed` events

### Weeknights (April 13â€“17) â€” RBAC + SSO + Passkeys

**Monday evening (3h):**
- [ ] RBAC: Roles + permissions CRUD
- [ ] RBAC: Role-permission attachment
- [ ] RBAC: User-role assignment
- [ ] Audit: wire `role.created`, `role.updated`, `role.deleted`, `role.assigned`, `role.unassigned`

**Tuesday evening (3h):**
- [ ] RBAC: Permission resolution (user â†’ roles â†’ permissions)
- [ ] **Unit tests: RBAC permission resolution â€” table-driven (admin wildcard, multi-role merge, no roles = no access)**
- [ ] RBAC: `POST /auth/check` endpoint
- [ ] RBAC: `RequirePermission` middleware
- [ ] **Integration test: user with role hits protected endpoint (200), user without role (403)**
- [ ] RBAC: Seed default roles (admin, member) on first boot

**Wednesday evening (3h):**
- [ ] SSO: Connection model CRUD
- [ ] SSO: OIDC client flow (redirect â†’ callback â†’ token exchange â†’ user creation/linking)
- [ ] SSO: Domain-based auto-routing (`GET /auth/sso?email=...`)

**Thursday evening (3h):**
- [ ] SSO: SAML SP metadata generation (XML)
- [ ] SSO: SAML ACS endpoint (receive + parse assertion)
- [ ] SSO: SAML signature verification (x509 cert from IdP)
- [ ] SSO: User creation/linking from SAML assertion
- [ ] Test SAML with a free Okta dev account
- [ ] Audit: wire `sso.connection_created`, `sso.login` events

**Friday evening (3h):**
- [ ] Passkeys: Add `go-webauthn/webauthn` dependency
- [ ] Passkeys: Registration begin/finish endpoints (attestation verification)
- [ ] Passkeys: Login begin/finish endpoints (assertion verification, sign_count update)
- [ ] Passkeys: Credential CRUD (list, delete, rename)
- [ ] Passkeys: Discoverable credential flow (no email required) + non-discoverable fallback
- [ ] **Integration test: passkey `begin` endpoints return well-formed options (rp.id, challenge, user info)**
- [ ] Audit: wire `passkey.registered`, `passkey.login`, `passkey.deleted`

### Weekend 2 (April 18â€“19) â€” Magic Links + M2M + Dashboard

**Saturday â€” Magic Links + M2M API Keys**

Morning (4h):
- [ ] Email: SMTP sender (net/smtp with STARTTLS)
- [ ] Email: HTML templates (magic link, email verification â€” minimal, inline CSS)
- [ ] Magic links: Token generation (32 bytes â†’ base64url), SHA-256 hash storage
- [ ] Magic links: Send endpoint (rate limit: 1 per email per 60s)
- [ ] Magic links: Verify endpoint (hash token, check expiry, create session, redirect)
- [ ] Magic links: Create-on-first-use flow (new user gets account + email_verified=1)
- [ ] **Integration test: magic link send â†’ MemoryEmailSender captures email â†’ extract token â†’ verify â†’ session active â†’ second verify = 400 (one-time use)**
- [ ] Audit: wire `magic_link.sent`, `magic_link.verified`

Afternoon (4h):
- [ ] M2M API keys: Key generation (sk_live_ + 32 random bytes base62)
- [ ] M2M API keys: Create endpoint (return full key ONCE, store SHA-256 hash)
- [ ] M2M API keys: List/get/update/revoke endpoints
- [ ] M2M API keys: Rotate endpoint (atomic: create new, revoke old)
- [ ] M2M API keys: Auth middleware (Bearer token â†’ hash â†’ lookup â†’ scope check)
- [ ] M2M API keys: In-memory token bucket rate limiter per key
- [ ] **Unit tests: API key generation (sk_live_ prefix), hash round-trip, wrong key fails, constant-time comparison**
- [ ] **Integration test: create key â†’ API call with Bearer â†’ scope enforcement (403 on wrong scope) â†’ rate limit (429)**
- [ ] Audit: wire `api_key.created`, `api_key.rotated`, `api_key.revoked`

Evening (3h):
- [ ] Integration testing: passkey register â†’ passkey login â†’ verify MFA skipped
- [ ] Integration testing: magic link send â†’ verify â†’ session created
- [ ] Integration testing: M2M key create â†’ API call with key â†’ scope enforcement
- [ ] Test passkey flow in browser (need minimal HTML test page)
- [ ] **Security tests: session fixation (token rotates on login), login brute force (10 fails â†’ 429), token expiry (Clock-based)**
- [ ] **Integration test: audit log â€” perform login + role change â†’ GET /audit-logs â†’ verify events recorded**

**Sunday â€” Dashboard**

Morning (4h):
- [ ] Dashboard: MFA views (per-user MFA status, enable/disable toggle, recovery code regen)
- [ ] Dashboard: RBAC views (role list, create/edit role, attach permissions, assign roles to users)
- [ ] Dashboard: SSO views (connection list, create SAML/OIDC connection form, cert upload, test button)
- [ ] Dashboard: Migration view (upload Auth0 JSON, progress bar, history table)

Afternoon (4h):
- [ ] Dashboard: Passkey views (per-user passkey list, last used, delete, device info from aaguid)
- [ ] Dashboard: API key views (create key modal, show full key once, list with prefix, revoke, rotate, usage stats)
- [ ] Dashboard: User detail page â€” show MFA status, passkeys, assigned roles, SSO identities, OAuth accounts, active sessions, auth method breakdown
- [ ] Dashboard: Overview page â€” auth method distribution chart (password vs passkey vs magic link vs OAuth vs SSO)

Evening (3h):
- [ ] Dashboard: **Audit log view** (event stream with filters: action, user, date range, status. Cursor-based pagination. Export button.)
- [ ] Audit log API handlers (`audit_handlers.go`) â€” list, detail, per-user, export
- [ ] Integration testing: all dashboard views against live API
- [ ] Fix edge cases, error messages, validation gaps
- [ ] Passkey: Test with multiple authenticator types (platform + cross-platform)

### Weeknights (April 20â€“24) â€” SDK + Examples

**Monday evening (3h):**
- [ ] SDK: Project scaffold (tsup, tsconfig, package.json for @sharkauth/js)
- [ ] SDK: HTTP client (fetch wrapper with credentials: 'include', error handling)
- [ ] SDK: auth module (signup, login, logout, me, check)
- [ ] SDK: types.ts (all request/response interfaces)

**Tuesday evening (3h):**
- [ ] SDK: passkey module (ArrayBufferâ†”base64url helpers, register, login, list, remove, rename)
- [ ] SDK: magic-link module (send)
- [ ] SDK: oauth module (getURL helper)
- [ ] SDK: mfa module (enroll, verify, challenge, useRecoveryCode)

**Wednesday evening (3h):**
- [ ] SDK: Build + test (ESM + CJS output, verify types)
- [ ] **SDK tests: Vitest + MSW â€” auth happy path, login error, passkey base64url helpers, type checking (tsc --noEmit)**
- [ ] SDK: README with full code examples (all auth methods)
- [ ] Example: Next.js App Router project (middleware.ts, login page, protected page)

**Thursday evening (3h):**
- [ ] Example: React SPA with Vite (login form with all auth methods, passkey button component)
- [ ] Test SDK against live SharkAuth server end-to-end
- [ ] SDK: Edge cases (expired sessions, network errors, passkey not supported fallback)

**Friday evening (3h) â€” buffer / polish:**
- [ ] Fix any SDK or API issues found during testing
- [ ] SDK: Add JSDoc comments to all public methods
- [ ] Publish @sharkauth/js to npm (v0.1.0)

### Weekend 3 (April 26â€“27) â€” Package + Ship

**Saturday â€” Docker + Docs**

Morning (4h):
- [ ] Dockerfile (multi-stage build: compile Svelte â†’ embed in Go â†’ Alpine, <30MB)
- [ ] docker-compose.yml for one-command startup (includes SMTP for dev via Mailpit)
- [ ] `sharkauth init` command (generates config + secret + first admin key)
- [ ] Verify dashboard serves at `:8080/admin` from single binary

Afternoon (4h):
- [ ] README.md: quickstart, full API reference, config guide
- [ ] README.md: Passkey setup guide (config, SDK usage, browser support notes)
- [ ] README.md: Magic link setup guide (SMTP config, email templates)
- [ ] README.md: MFA setup guide with dashboard screenshots
- [ ] README.md: RBAC guide (create roles, assign, check â€” show dashboard + API)
- [ ] README.md: SSO setup guide (SAML with Okta walkthrough, OIDC example)
- [ ] README.md: M2M API key guide (create, scope, rotate, use in service)
- [ ] README.md: Auth0 migration guide (CLI + dashboard upload)
- [ ] README.md: SDK quickstart (npm install, Next.js example, React example)

Evening (2h):
- [ ] End-to-end testing of Docker image
- [ ] **GitHub Actions CI: lint (golangci-lint + gosec) â†’ test-go (race + 60% coverage gate) â†’ test-sdk (typecheck + vitest) â†’ build (Svelte + Go + verify binary starts)**
- [ ] `go test ./... -race -cover` â€” verify 60%+ coverage, fix any gaps
- [ ] Tag v0.1.0

**Sunday â€” Launch**

Morning (3h):
- [ ] Update sharkauth.com: remove all "shipping soon" labels
- [ ] Update sharkauth.com: new pricing (self-hosted free, $19/$49/$149 cloud)
- [ ] Update sharkauth.com: add 14-day free trial CTA
- [ ] Update sharkauth.com: add dashboard screenshots to features section
- [ ] Update sharkauth.com: add passkey + magic link to features (differentiation vs Auth0 free tier)
- [ ] Update sharkauth.com: add SDK install snippet prominently
- [ ] Update comparison table with real 2026 numbers

Afternoon (2h):
- [ ] Final full-flow test on a fresh $5 VPS (prove the "3 minutes" claim)
- [ ] Record demo GIF: install â†’ `shark serve` â†’ dashboard â†’ create user â†’ passkey register â†’ passkey login â†’ magic link â†’ OAuth â†’ MFA â†’ assign role â†’ create API key â†’ migrate Auth0 users
- [ ] Push everything

Evening (2h):
- [ ] Post to r/selfhosted, r/golang, r/nextjs, Hacker News
- [ ] Post on X/Twitter
- [ ] Announce on BuildersMTY Discord
- [ ] Respond to every comment for 48 hours

---

## Auth0 Migration

### Password hash flow:

```
Login request comes in
  â†’ Look up user by email
  â†’ Check hash_type
  â†’ If "bcrypt": verify with bcrypt, if valid â†’ rehash with argon2id, update hash_type
  â†’ If "argon2id": verify with argon2id (normal path)
  â†’ If "scrypt": verify with scrypt, rehash (for Firebase imports later)
  â†’ Return session (or MFA challenge if enabled)
```

---

## WebAuthn Implementation Notes

### Library

Use `github.com/go-webauthn/webauthn` â€” the standard Go WebAuthn library. It handles CBOR decoding, attestation verification, and assertion verification.

### Key decisions

- **Attestation:** `none` by default (maximum authenticator compatibility). Configurable for enterprise deployments that need device attestation.
- **Resident keys:** `preferred` â€” enables discoverable credential flow (login without typing email) but doesn't exclude security keys that don't support it.
- **User verification:** `preferred` â€” uses biometric/PIN when available, doesn't fail on security keys without it.
- **Algorithms:** Support ES256 (preferred) and RS256 (fallback for Windows Hello on older versions).
- **Challenge storage:** Store in sessions table with 5-minute expiry. Clean up expired challenges on a timer.
- **Sign count validation:** Verify sign_count increases on each authentication. Log warning if it doesn't (cloned authenticator risk) but don't block â€” some authenticators (like iCloud Keychain synced passkeys) always return 0.
- **MFA bypass:** Passkey authentication sets mfa_passed=1 automatically. Passkeys are phishing-resistant AAL2 by definition (NIST SP 800-63-4).

### Browser detection for SDK

```typescript
// In @sharkauth/js passkey module
export function isPasskeySupported(): boolean {
  return typeof window !== 'undefined'
    && typeof window.PublicKeyCredential !== 'undefined'
    && typeof window.PublicKeyCredential.isConditionalMediationAvailable === 'function';
}
```

---

## README Structure (ships with the code)

```markdown
# ðŸ¦ˆ Shark Auth

Single-binary authentication server.
Passkeys, MFA, SSO, RBAC, magic links, API keys, admin dashboard â€” all in one binary. Self-host in three minutes.

## Quickstart

# Download binary from GitHub Releases: https://github.com/shark-auth/shark/releases
./shark serve

# Dashboard at http://localhost:8080/admin
# API at http://localhost:8080/api/v1

## SDK

npm install @sharkauth/js

import { createSharkAuth } from '@sharkauth/js';
const auth = createSharkAuth({ baseURL: 'http://localhost:8080' });

// Password login
await auth.login({ email: 'user@example.com', password: 'secret' });

// Passkey login (one tap, no password)
await auth.passkey.login();

// Magic link (passwordless email)
await auth.magicLink.send({ email: 'user@example.com' });

## Migrate from Auth0

sharkauth migrate auth0 --file export.json
# 4,218 users imported. Passwords work immediately.
# Or upload via dashboard: Admin â†’ Migrations â†’ Import

## Features

- Email/password + passkeys + magic links
- Social login: Google, GitHub, Apple, Discord
- MFA (TOTP) with recovery codes
- RBAC (roles + permissions + enforcement middleware)
- SSO (SAML + OIDC)
- M2M API keys (scoped, rotatable)
- Admin dashboard (embedded Svelte UI â€” no separate deploy)
- TypeScript SDK (@sharkauth/js)
- Server-side sessions
- Auth0 migration (CLI or dashboard)
- SQLite default, zero dependencies
- <30MB Docker image

## API Reference

[Full endpoint docs: auth, passkeys, magic links, OAuth, MFA, RBAC, SSO, API keys, admin, migration]

## Guides

- [Passkey Setup](docs/passkeys.md)
- [Magic Links](docs/magic-links.md)
- [MFA Setup](docs/mfa.md)
- [RBAC Configuration](docs/rbac.md)
- [SSO with Okta](docs/sso-okta.md)
- [M2M API Keys](docs/api-keys.md)
- [Auth0 Migration](docs/migration-auth0.md)
- [SDK Quickstart](docs/sdk.md)
- [Next.js Integration](docs/nextjs.md)
- [Dashboard Overview](docs/dashboard.md)

## Pricing

Self-hosted: $0 forever. Every feature. No limits.
Cloud: Starting at $19/mo. Same features, we run it.

## Why Shark Auth?

- Single binary. Dashboard included. Runs on a $5 VPS.
- Passkeys, MFA, SSO, RBAC on every plan. No add-on tax.
- TypeScript SDK â€” npm install and go.
- Auth0 migration in one command. No password resets.
- 98% cheaper than Clerk/Auth0 at every scale.
- Open source. Self-host forever.
```

---

## Launch Post Template

**Title:** "Shark Auth: Single binary auth with passkeys, MFA, SSO, RBAC, and a TypeScript SDK."

**Body:**

I built Shark Auth because auth providers charge you per heartbeat
and gate basic security features behind enterprise plans.

Shark Auth is a single Go binary. Self-host it in 3 minutes.
Run `shark serve` and you get an API + an admin dashboard. No separate frontend. No npm install. Everything in one binary.

What's included:

- Email/password + passkeys + magic links
- Social login: Google, GitHub, Apple, Discord
- MFA (TOTP) with recovery codes
- RBAC â€” roles, permissions, enforcement
- SSO â€” SAML + OIDC
- M2M API keys â€” scoped, rotatable, rate-limited
- Admin dashboard â€” manage everything from a browser
- TypeScript SDK â€” `npm install @sharkauth/js`
- Auth0 migration: one command, passwords work immediately
- SQLite default, <30MB Docker image

Pricing: Self-hosted is free forever with every feature.
Cloud: $19/mo for 50K MAU. No per-user pricing. Ever.

At 100K users, Clerk costs $1,025/mo. Auth0 costs ~$6,300/mo.
Shark Auth costs $19. Or $0 if you self-host.

Passkeys on every tier (Auth0 gates them behind paid plans).
MFA on every tier (Clerk charges $100/mo extra).
SSO on every tier (WorkOS charges $125/connection).

[screenshot of dashboard]

GitHub: [link]
SDK: `npm install @sharkauth/js`
Site: sharkauth.com
Docs: docs.sharkauth.com

What migration source should I build next?
