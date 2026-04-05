# Shark Auth — Launch Sprint Spec (v2)

**Ship date:** Sunday, April 27, 2026
**Time budget:** ~17 days (evenings + three weekends)
**Rule:** If it's not on this list, it doesn't exist yet.

> **What changed from v1:** Market research showed passkeys are approaching table stakes, REST-only is a dealbreaker for most devs, single-provider OAuth limits addressable market, and M2M tokens are increasingly expected. This spec adds passkeys/WebAuthn, multi-provider social OAuth (Google, GitHub, Apple, Discord), magic links, M2M API keys, and a TypeScript SDK — while keeping the single-binary philosophy intact.

---

## What ships at launch

A single Go binary that handles signup, login, sessions, OAuth, MFA, passkeys, magic links, RBAC, SSO, M2M tokens — with a working Auth0 migration CLI, an embedded admin dashboard, and a TypeScript SDK. No "shipping soon." Every feature on the landing page works. No separate frontend deploy. `shark serve` and visit `:8080/admin`.

### The binary does exactly this:

1. **Email/password signup + login** (Argon2id hashing)
2. **Passkey/WebAuthn login** (FIDO2-compliant, resident credentials, platform + cross-platform authenticators)
3. **Magic link login** (email-based passwordless — token via SMTP, click to authenticate)
4. **Server-side sessions** (encrypted cookies, no JWT complexity)
5. **Social OAuth login** — Google, GitHub, Apple, Discord (four providers, generic handler pattern)
6. **MFA — TOTP** (Google Authenticator / Authy compatible, recovery codes)
7. **RBAC** (roles, permissions, role assignment, middleware enforcement)
8. **SSO — OIDC provider** (SharkAuth as an OIDC IdP)
9. **SSO — SAML SP** (beta — connect to enterprise IdPs like Okta/Azure AD)
10. **M2M API keys** (service-to-service auth — scoped, rotatable, rate-limit-aware)
11. **Auth0 user import** (read their export JSON, verify bcrypt hashes, rehash on first login)
12. **REST API** for everything above
13. **TypeScript SDK** (`@sharkauth/js`) — fetch-based, zero-dependency, works in Node/browser/edge
14. **YAML config** with env var overrides
15. **SQLite storage** (embedded, zero-config)
16. **Docker image** (one container, done)
17. **Admin dashboard** (Svelte, embedded in the binary — user management, sessions, MFA, passkeys, RBAC, SSO connections, API keys, migration status)

### What does NOT ship at launch:

- No organizations/multi-tenancy (post-launch — highest priority after launch)
- No OIDC client federation (post-launch — this is "Login with X enterprise IdP via OIDC")
- No agent identity / MCP auth (later — emerging standard, not yet table stakes)
- No Clerk/Firebase/Cognito migration (later)
- No Postgres mode (later)
- No React/Next.js component library (later — SDK ships first, pre-built UI components follow)

---

## Pricing (ships on sharkauth.com)

### Philosophy

The binary is the product. Cloud is a convenience layer, not a feature gate. Self-hosted is the free tier. Every auth feature ships in the binary for $0. Cloud sells operational burden — managed infra, SLA, support — not features.

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
```

---

## API Endpoints

### Auth (core)

```
POST   /api/v1/auth/signup              — Create user + session
POST   /api/v1/auth/login               — Verify password, return session (or MFA challenge)
POST   /api/v1/auth/logout              — Destroy session
GET    /api/v1/auth/me                  — Current user from session
```

### Social OAuth (generic pattern)

```
GET    /api/v1/auth/oauth/:provider              — Redirect to provider (google, github, apple, discord)
GET    /api/v1/auth/oauth/:provider/callback     — Handle OAuth callback, create/link user + session
```

**Supported providers:** `google`, `github`, `apple`, `discord`

### Passkeys / WebAuthn

```
POST   /api/v1/auth/passkey/register/begin    — Generate registration options (PublicKeyCredentialCreationOptions)
POST   /api/v1/auth/passkey/register/finish   — Verify attestation, store credential
POST   /api/v1/auth/passkey/login/begin       — Generate authentication options (PublicKeyCredentialRequestOptions)
POST   /api/v1/auth/passkey/login/finish      — Verify assertion, create session
GET    /api/v1/auth/passkey/credentials        — List user's registered passkeys
DELETE /api/v1/auth/passkey/credentials/:id    — Remove a passkey
PATCH  /api/v1/auth/passkey/credentials/:id    — Rename a passkey
```

**Passkey registration flow:**

```
1. User is logged in (via password, OAuth, or magic link)
2. POST /passkey/register/begin
   → Server generates challenge, returns PublicKeyCredentialCreationOptions
   → Options include: rp.id (from config), user.id, user.name, user.displayName
   → Options include: pubKeyCredParams [ES256, RS256], authenticatorSelection
3. Client calls navigator.credentials.create() with those options
4. POST /passkey/register/finish with attestation response
   → Server verifies origin, challenge, attestation
   → Stores credential_id, public_key, sign_count, aaguid, transports
   → Returns { "credential_id": "pk_xxx", "name": "Unnamed passkey" }
```

**Passkey login flow:**

```
1. POST /passkey/login/begin (optionally with { "email": "..." } for non-discoverable)
   → Server generates challenge, returns PublicKeyCredentialRequestOptions
   → If email provided: includes allowCredentials for that user's passkeys
   → If no email: empty allowCredentials (discoverable/resident key flow)
2. Client calls navigator.credentials.get() with those options
3. POST /passkey/login/finish with assertion response
   → Server looks up credential by credential_id
   → Verifies signature against stored public_key
   → Verifies sign_count > stored value (replay protection)
   → Updates sign_count + last_used_at
   → Creates session with auth_method="passkey", mfa_passed=1 (passkeys satisfy MFA)
   → Returns session cookie
```

### Magic Links

```
POST   /api/v1/auth/magic-link/send       — Send magic link to email
GET    /api/v1/auth/magic-link/verify      — Verify token from email link, create session
```

**Magic link flow:**

```
1. POST /magic-link/send with { "email": "user@example.com" }
   → Generate 32-byte random token
   → Store SHA-256(token) in magic_link_tokens with 10-minute expiry
   → Send email with link: https://yourapp.com/auth/magic?token=<token>
   → If user doesn't exist: create account with email_verified=1
   → Always return 200 (don't leak whether email exists)
2. GET /magic-link/verify?token=<token>
   → Hash token, look up in magic_link_tokens
   → Verify not expired, not used
   → Mark as used
   → Create or find user by email, set email_verified=1
   → Create session with auth_method="magic_link"
   → Redirect to configured success URL with session cookie set
```

### MFA

```
POST   /api/v1/auth/mfa/enroll          — Generate TOTP secret + QR URI
POST   /api/v1/auth/mfa/verify          — Confirm setup with first code
POST   /api/v1/auth/mfa/challenge       — Submit TOTP code during login
POST   /api/v1/auth/mfa/recovery        — Use a recovery code
DELETE /api/v1/auth/mfa                  — Disable MFA (requires current code)
GET    /api/v1/auth/mfa/recovery-codes   — Regenerate recovery codes
```

**MFA login flow:**

```
1. POST /auth/login with email + password
2. If MFA enabled → 200 { "mfa_required": true, "session_token": "partial_xxx" }
   Session is created with mfa_passed=0 (can only hit /mfa/challenge)
3. POST /auth/mfa/challenge with TOTP code
4. If valid → session upgraded to mfa_passed=1, full access granted
5. If user lost device → POST /auth/mfa/recovery with recovery code
```

**Note:** Passkey login bypasses MFA — passkeys are phishing-resistant and satisfy AAL2. Sessions created via passkey have mfa_passed=1 automatically.

### RBAC

```
POST   /api/v1/roles                    — Create role
GET    /api/v1/roles                    — List roles
GET    /api/v1/roles/:id                — Get role with permissions
PUT    /api/v1/roles/:id                — Update role
DELETE /api/v1/roles/:id                — Delete role

POST   /api/v1/permissions              — Create permission
GET    /api/v1/permissions              — List permissions

POST   /api/v1/roles/:id/permissions    — Attach permissions to role
DELETE /api/v1/roles/:id/permissions/:pid — Detach permission from role

POST   /api/v1/users/:id/roles          — Assign role to user
DELETE /api/v1/users/:id/roles/:rid      — Remove role from user
GET    /api/v1/users/:id/roles           — List user's roles
GET    /api/v1/users/:id/permissions     — List user's effective permissions (resolved)

POST   /api/v1/auth/check               — Check permission: { "user_id", "action", "resource" } → { "allowed": bool }
```

### SSO

```
POST   /api/v1/sso/connections           — Create SSO connection (SAML or OIDC config)
GET    /api/v1/sso/connections           — List connections
GET    /api/v1/sso/connections/:id       — Get connection details
PUT    /api/v1/sso/connections/:id       — Update connection
DELETE /api/v1/sso/connections/:id       — Delete connection

GET    /api/v1/sso/saml/:connection_id/metadata  — SP metadata XML (for IdP setup)
POST   /api/v1/sso/saml/:connection_id/acs       — SAML ACS endpoint (receives assertion)

GET    /api/v1/sso/oidc/:connection_id/auth      — OIDC authorization redirect
GET    /api/v1/sso/oidc/:connection_id/callback   — OIDC callback

GET    /api/v1/auth/sso?email=user@corp.com      — Auto-route: looks up domain → redirect to correct IdP
```

### M2M API Keys

```
POST   /api/v1/api-keys                 — Create API key (returns full key ONCE, then only prefix shown)
GET    /api/v1/api-keys                 — List API keys (prefix + metadata only, never full key)
GET    /api/v1/api-keys/:id             — Get API key details
PATCH  /api/v1/api-keys/:id             — Update name, scopes, rate_limit, expires_at
DELETE /api/v1/api-keys/:id             — Revoke API key (soft-delete, sets revoked_at)
POST   /api/v1/api-keys/:id/rotate     — Rotate: generate new key, revoke old one atomically
```

**API key authentication:**
```
Authorization: Bearer sk_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

→ Server hashes the key with SHA-256
→ Looks up key_hash in api_keys table
→ Verifies not revoked (revoked_at IS NULL)
→ Verifies not expired (expires_at IS NULL OR expires_at > now)
→ Checks scope against requested action
→ Enforces rate limit (in-memory token bucket per key_hash)
→ Updates last_used_at
→ Request proceeds with the key's scopes as the permission set
```

**Key format:** `sk_live_` + 32 random bytes base62-encoded (48 chars total). The `sk_live_` prefix makes keys easily identifiable in logs and secret scanners.

### Admin + Migration

```
GET    /api/v1/users                     — List users (paginated)
GET    /api/v1/users/:id                 — Get user
POST   /api/v1/migrate/auth0             — Upload Auth0 export JSON
GET    /api/v1/migrate/:id               — Migration status
GET    /healthz                          — Health check
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

admin:
  api_key: "${SHARKAUTH_ADMIN_KEY}"
```

---

## Project Structure (updated)

```
sharkauth/
├── main.go
├── cmd/
│   ├── serve.go
│   └── migrate.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── storage/
│   │   ├── storage.go                   # Storage interface
│   │   └── sqlite.go                    # SQLite implementation
│   ├── auth/
│   │   ├── password.go                  # Argon2id + multi-hash
│   │   ├── session.go                   # Sessions + MFA-aware gating
│   │   ├── oauth.go                     # Generic OAuth handler + provider registry
│   │   ├── providers/                   # OAuth provider implementations
│   │   │   ├── google.go
│   │   │   ├── github.go
│   │   │   ├── apple.go                 # Apple uses JWT client_secret
│   │   │   └── discord.go
│   │   ├── mfa.go                       # TOTP generation, verification, recovery codes
│   │   ├── passkey.go                   # WebAuthn registration + authentication (go-webauthn/webauthn)
│   │   ├── magiclink.go                 # Token generation, email sending, verification
│   │   └── apikey.go                    # M2M API key CRUD, hashing, validation, rate limiting
│   ├── email/
│   │   ├── sender.go                    # SMTP client wrapper
│   │   └── templates/                   # HTML email templates
│   │       ├── magic_link.html
│   │       └── verify_email.html
│   ├── rbac/
│   │   ├── rbac.go                      # Role/permission CRUD
│   │   └── middleware.go                # RequirePermission("action", "resource")
│   ├── sso/
│   │   ├── saml.go                      # SAML SP: metadata, ACS, assertion parsing
│   │   ├── oidc.go                      # OIDC client: auth redirect, callback, token exchange
│   │   └── connection.go                # SSO connection CRUD + domain routing
│   ├── user/
│   │   └── user.go
│   ├── migrate/
│   │   └── auth0.go
│   └── api/
│       ├── router.go
│       ├── auth_handlers.go             # Signup, login, logout, me
│       ├── oauth_handlers.go            # Generic OAuth redirect + callback
│       ├── passkey_handlers.go          # WebAuthn register/login begin/finish
│       ├── magiclink_handlers.go        # Magic link send + verify
│       ├── mfa_handlers.go              # Enroll, verify, challenge, recovery
│       ├── rbac_handlers.go             # Roles, permissions, assignment, check
│       ├── sso_handlers.go              # Connections, SAML ACS, OIDC callback
│       ├── apikey_handlers.go           # M2M API key CRUD + rotate
│       ├── user_handlers.go             # Admin user endpoints
│       └── migrate_handlers.go          # Migration endpoints
├── dashboard/                           # Svelte app, embedded in binary
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +page.svelte             # Dashboard overview (user count, active sessions, auth method breakdown)
│   │   │   ├── users/
│   │   │   │   ├── +page.svelte         # User list (search, paginate, filter by auth method)
│   │   │   │   └── [id]/+page.svelte    # User detail (sessions, roles, MFA, passkeys, OAuth accounts)
│   │   │   ├── sessions/+page.svelte    # Active sessions (revoke, per-device view, auth method column)
│   │   │   ├── roles/
│   │   │   │   ├── +page.svelte         # Role list + create
│   │   │   │   └── [id]/+page.svelte    # Role detail (edit permissions, assigned users)
│   │   │   ├── sso/
│   │   │   │   ├── +page.svelte         # SSO connections list
│   │   │   │   └── [id]/+page.svelte    # Connection config (SAML cert upload, OIDC fields)
│   │   │   ├── api-keys/
│   │   │   │   └── +page.svelte         # API key list, create, revoke, rotate, usage stats
│   │   │   └── migrations/
│   │   │       └── +page.svelte         # Migration history + trigger new import
│   │   └── lib/
│   │       ├── api.ts                   # Fetch wrapper for internal API
│   │       └── components/              # Shared UI components
│   ├── package.json
│   └── svelte.config.js
├── sdk/                                 # TypeScript SDK
│   ├── src/
│   │   ├── index.ts                     # Main export: createSharkAuth(config)
│   │   ├── client.ts                    # HTTP client (fetch-based, works everywhere)
│   │   ├── auth.ts                      # signup, login, logout, me
│   │   ├── passkey.ts                   # registerPasskey, loginWithPasskey (wraps navigator.credentials)
│   │   ├── magic-link.ts               # sendMagicLink
│   │   ├── oauth.ts                     # getOAuthURL, handleCallback
│   │   ├── mfa.ts                       # enrollMFA, verifyMFA, challengeMFA
│   │   └── types.ts                     # All TypeScript interfaces
│   ├── package.json                     # @sharkauth/js
│   ├── tsconfig.json
│   └── README.md                        # SDK-specific docs with code examples
├── examples/
│   ├── nextjs/                          # Next.js App Router example
│   │   ├── middleware.ts                # Auth middleware (session check)
│   │   ├── app/
│   │   │   ├── login/page.tsx           # Login form (password + passkey + magic link + OAuth buttons)
│   │   │   └── dashboard/page.tsx       # Protected page
│   │   └── lib/sharkauth.ts            # SDK initialization
│   └── react-spa/                       # Vite + React SPA example
│       ├── src/
│       │   ├── auth/
│       │   │   ├── LoginForm.tsx         # Full login component
│       │   │   └── PasskeyButton.tsx     # Passkey-specific component
│       │   └── App.tsx
│       └── package.json
├── sharkauth.yaml
├── Dockerfile
├── docker-compose.yml
├── README.md
└── go.mod
```

---

## TypeScript SDK Design (`@sharkauth/js`)

### API surface

```typescript
import { createSharkAuth } from '@sharkauth/js';

const auth = createSharkAuth({
  baseURL: 'https://auth.myapp.com',  // or http://localhost:8080
  // No API key needed for client-side operations — session cookies handle auth
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

- **Zero dependencies** — uses native `fetch` only
- **Isomorphic** — works in browser, Node.js 18+, Deno, Bun, Cloudflare Workers, Vercel Edge
- **Cookie-based** — relies on `credentials: 'include'` for session management, no token handling
- **Passkey helpers** — wraps WebAuthn browser API with proper ArrayBuffer↔base64url conversion
- **Tree-shakeable** — ESM-first with named exports
- **Type-safe** — full TypeScript types for all request/response shapes

### Build + publish

```bash
cd sdk/
npm run build    # tsup → ESM + CJS + .d.ts
npm publish      # → @sharkauth/js on npm
```

---

## 17-Day Sprint Plan

### Weekend 1 (April 11–12) — Core auth + OAuth

**Saturday — Scaffold + Auth + OAuth**

Morning (4h):
- [ ] `go mod init`, project scaffold
- [ ] Config loader (YAML + env vars, including new passkey/smtp/social sections)
- [ ] SQLite storage layer (connect, migrate full schema including new tables, CRUD)
- [ ] User model + create/get/list

Afternoon (4h):
- [ ] Password hashing (Argon2id) + multi-hash verification (bcrypt compat)
- [ ] Session management (create, validate, revoke, cookie handling, auth_method tracking)
- [ ] Signup + login + logout + me API handlers
- [ ] Test with curl

Evening (3h):
- [ ] Generic OAuth handler: provider registry pattern, redirect + callback
- [ ] Google OAuth provider implementation
- [ ] GitHub OAuth provider implementation
- [ ] Admin endpoints behind API key

**Sunday — More OAuth + Migration + MFA**

Morning (4h):
- [ ] Apple OAuth provider (JWT client_secret from .p8 key, id_token parsing)
- [ ] Discord OAuth provider
- [ ] Test all four OAuth flows end-to-end
- [ ] Error handling, input validation, HTTP status codes

Afternoon (4h):
- [ ] Auth0 migration: JSON parser, CLI command, API endpoint
- [ ] Transparent bcrypt→argon2id rehash on first login
- [ ] MFA: TOTP secret generation (crypto/rand → base32)
- [ ] MFA: QR URI builder (otpauth://totp/...)

Evening (3h):
- [ ] MFA: TOTP code validation (HMAC-SHA1, 30s window, ±1 step tolerance)
- [ ] MFA: Enroll + verify + challenge endpoints
- [ ] MFA: Recovery codes (generate 10, hash with bcrypt, one-time use)
- [ ] MFA: Login flow integration (partial session → challenge → upgrade)
- [ ] MFA: Disable endpoint (require current code to turn off)
- [ ] Test full MFA flow end-to-end

### Weeknights (April 13–17) — RBAC + SSO + Passkeys

**Monday evening (3h):**
- [ ] RBAC: Roles + permissions CRUD
- [ ] RBAC: Role-permission attachment
- [ ] RBAC: User-role assignment

**Tuesday evening (3h):**
- [ ] RBAC: Permission resolution (user → roles → permissions)
- [ ] RBAC: `POST /auth/check` endpoint
- [ ] RBAC: `RequirePermission` middleware
- [ ] RBAC: Seed default roles (admin, member) on first boot

**Wednesday evening (3h):**
- [ ] SSO: Connection model CRUD
- [ ] SSO: OIDC client flow (redirect → callback → token exchange → user creation/linking)
- [ ] SSO: Domain-based auto-routing (`GET /auth/sso?email=...`)

**Thursday evening (3h):**
- [ ] SSO: SAML SP metadata generation (XML)
- [ ] SSO: SAML ACS endpoint (receive + parse assertion)
- [ ] SSO: SAML signature verification (x509 cert from IdP)
- [ ] SSO: User creation/linking from SAML assertion
- [ ] Test SAML with a free Okta dev account

**Friday evening (3h):**
- [ ] Passkeys: Add `go-webauthn/webauthn` dependency
- [ ] Passkeys: Registration begin/finish endpoints (attestation verification)
- [ ] Passkeys: Login begin/finish endpoints (assertion verification, sign_count update)
- [ ] Passkeys: Credential CRUD (list, delete, rename)
- [ ] Passkeys: Discoverable credential flow (no email required) + non-discoverable fallback

### Weekend 2 (April 18–19) — Magic Links + M2M + Dashboard

**Saturday — Magic Links + M2M API Keys**

Morning (4h):
- [ ] Email: SMTP sender (net/smtp with STARTTLS)
- [ ] Email: HTML templates (magic link, email verification — minimal, inline CSS)
- [ ] Magic links: Token generation (32 bytes → base64url), SHA-256 hash storage
- [ ] Magic links: Send endpoint (rate limit: 1 per email per 60s)
- [ ] Magic links: Verify endpoint (hash token, check expiry, create session, redirect)
- [ ] Magic links: Create-on-first-use flow (new user gets account + email_verified=1)

Afternoon (4h):
- [ ] M2M API keys: Key generation (sk_live_ + 32 random bytes base62)
- [ ] M2M API keys: Create endpoint (return full key ONCE, store SHA-256 hash)
- [ ] M2M API keys: List/get/update/revoke endpoints
- [ ] M2M API keys: Rotate endpoint (atomic: create new, revoke old)
- [ ] M2M API keys: Auth middleware (Bearer token → hash → lookup → scope check)
- [ ] M2M API keys: In-memory token bucket rate limiter per key

Evening (3h):
- [ ] Integration testing: passkey register → passkey login → verify MFA skipped
- [ ] Integration testing: magic link send → verify → session created
- [ ] Integration testing: M2M key create → API call with key → scope enforcement
- [ ] Test passkey flow in browser (need minimal HTML test page)

**Sunday — Dashboard**

Morning (4h):
- [ ] Dashboard: MFA views (per-user MFA status, enable/disable toggle, recovery code regen)
- [ ] Dashboard: RBAC views (role list, create/edit role, attach permissions, assign roles to users)
- [ ] Dashboard: SSO views (connection list, create SAML/OIDC connection form, cert upload, test button)
- [ ] Dashboard: Migration view (upload Auth0 JSON, progress bar, history table)

Afternoon (4h):
- [ ] Dashboard: Passkey views (per-user passkey list, last used, delete, device info from aaguid)
- [ ] Dashboard: API key views (create key modal, show full key once, list with prefix, revoke, rotate, usage stats)
- [ ] Dashboard: User detail page — show MFA status, passkeys, assigned roles, SSO identities, OAuth accounts, active sessions, auth method breakdown
- [ ] Dashboard: Overview page — auth method distribution chart (password vs passkey vs magic link vs OAuth vs SSO)

Evening (3h):
- [ ] Integration testing: all dashboard views against live API
- [ ] Fix edge cases, error messages, validation gaps
- [ ] Passkey: Test with multiple authenticator types (platform + cross-platform)

### Weeknights (April 20–24) — SDK + Examples

**Monday evening (3h):**
- [ ] SDK: Project scaffold (tsup, tsconfig, package.json for @sharkauth/js)
- [ ] SDK: HTTP client (fetch wrapper with credentials: 'include', error handling)
- [ ] SDK: auth module (signup, login, logout, me, check)
- [ ] SDK: types.ts (all request/response interfaces)

**Tuesday evening (3h):**
- [ ] SDK: passkey module (ArrayBuffer↔base64url helpers, register, login, list, remove, rename)
- [ ] SDK: magic-link module (send)
- [ ] SDK: oauth module (getURL helper)
- [ ] SDK: mfa module (enroll, verify, challenge, useRecoveryCode)

**Wednesday evening (3h):**
- [ ] SDK: Build + test (ESM + CJS output, verify types)
- [ ] SDK: README with full code examples (all auth methods)
- [ ] Example: Next.js App Router project (middleware.ts, login page, protected page)

**Thursday evening (3h):**
- [ ] Example: React SPA with Vite (login form with all auth methods, passkey button component)
- [ ] Test SDK against live SharkAuth server end-to-end
- [ ] SDK: Edge cases (expired sessions, network errors, passkey not supported fallback)

**Friday evening (3h) — buffer / polish:**
- [ ] Fix any SDK or API issues found during testing
- [ ] SDK: Add JSDoc comments to all public methods
- [ ] Publish @sharkauth/js to npm (v0.1.0)

### Weekend 3 (April 26–27) — Package + Ship

**Saturday — Docker + Docs**

Morning (4h):
- [ ] Dockerfile (multi-stage build: compile Svelte → embed in Go → Alpine, <30MB)
- [ ] docker-compose.yml for one-command startup (includes SMTP for dev via Mailpit)
- [ ] `sharkauth init` command (generates config + secret + first admin key)
- [ ] Verify dashboard serves at `:8080/admin` from single binary

Afternoon (4h):
- [ ] README.md: quickstart, full API reference, config guide
- [ ] README.md: Passkey setup guide (config, SDK usage, browser support notes)
- [ ] README.md: Magic link setup guide (SMTP config, email templates)
- [ ] README.md: MFA setup guide with dashboard screenshots
- [ ] README.md: RBAC guide (create roles, assign, check — show dashboard + API)
- [ ] README.md: SSO setup guide (SAML with Okta walkthrough, OIDC example)
- [ ] README.md: M2M API key guide (create, scope, rotate, use in service)
- [ ] README.md: Auth0 migration guide (CLI + dashboard upload)
- [ ] README.md: SDK quickstart (npm install, Next.js example, React example)

Evening (2h):
- [ ] End-to-end testing of Docker image
- [ ] GitHub Actions: build + test + publish image to ghcr.io
- [ ] Tag v0.1.0

**Sunday — Launch**

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
- [ ] Record demo GIF: install → `shark serve` → dashboard → create user → passkey register → passkey login → magic link → OAuth → MFA → assign role → create API key → migrate Auth0 users
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
  → Look up user by email
  → Check hash_type
  → If "bcrypt": verify with bcrypt, if valid → rehash with argon2id, update hash_type
  → If "argon2id": verify with argon2id (normal path)
  → If "scrypt": verify with scrypt, rehash (for Firebase imports later)
  → Return session (or MFA challenge if enabled)
```

---

## WebAuthn Implementation Notes

### Library

Use `github.com/go-webauthn/webauthn` — the standard Go WebAuthn library. It handles CBOR decoding, attestation verification, and assertion verification.

### Key decisions

- **Attestation:** `none` by default (maximum authenticator compatibility). Configurable for enterprise deployments that need device attestation.
- **Resident keys:** `preferred` — enables discoverable credential flow (login without typing email) but doesn't exclude security keys that don't support it.
- **User verification:** `preferred` — uses biometric/PIN when available, doesn't fail on security keys without it.
- **Algorithms:** Support ES256 (preferred) and RS256 (fallback for Windows Hello on older versions).
- **Challenge storage:** Store in sessions table with 5-minute expiry. Clean up expired challenges on a timer.
- **Sign count validation:** Verify sign_count increases on each authentication. Log warning if it doesn't (cloned authenticator risk) but don't block — some authenticators (like iCloud Keychain synced passkeys) always return 0.
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
# 🦈 Shark Auth

Single-binary authentication server.
Passkeys, MFA, SSO, RBAC, magic links, API keys, admin dashboard — all in one binary. Self-host in three minutes.

## Quickstart

curl -sSL get.sharkauth.dev | sh
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
# Or upload via dashboard: Admin → Migrations → Import

## Features

- Email/password + passkeys + magic links
- Social login: Google, GitHub, Apple, Discord
- MFA (TOTP) with recovery codes
- RBAC (roles + permissions + enforcement middleware)
- SSO (SAML + OIDC)
- M2M API keys (scoped, rotatable)
- Admin dashboard (embedded Svelte UI — no separate deploy)
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
- TypeScript SDK — npm install and go.
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
- RBAC — roles, permissions, enforcement
- SSO — SAML + OIDC
- M2M API keys — scoped, rotatable, rate-limited
- Admin dashboard — manage everything from a browser
- TypeScript SDK — `npm install @sharkauth/js`
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
Docs: docs.sharkauth.dev

What migration source should I build next?
