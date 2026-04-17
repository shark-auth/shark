# sharkauth

### a lightweight modern self-hosted authentication platform with no bullshit.  

[![version](https://img.shields.io/badge/version-v0.1.0-red?style=for-the-badge&logoColor=white)](https://github.com/sharkauth/sharkauth/releases/latest)
[![license](https://img.shields.io/badge/license-MIT-green?style=for-the-badge)](LICENSE)
[![platform](https://img.shields.io/badge/platform-Linux_|_macOS_|_Windows-orange?style=for-the-badge)](/)
[![status](https://img.shields.io/badge/status-pre--launch-yellow?style=for-the-badge)](/)

**built with**

![Go](https://img.shields.io/badge/Go_1.25-00ADD8?style=flat-square&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)
![Chi](https://img.shields.io/badge/Chi-router-black?style=flat-square)
![Argon2id](https://img.shields.io/badge/Argon2id-password_hashing-purple?style=flat-square)
![WebAuthn](https://img.shields.io/badge/WebAuthn-FIDO2_passkeys-ff69b4?style=flat-square)
![TOTP](https://img.shields.io/badge/TOTP-MFA-blue?style=flat-square)

---

Auth0 charges $23/mo for 1,000 users. Clerk charges $25/mo. Both lock you into their infrastructure, rate-limit your own users, and own your auth data.

**sharkauth is a single Go binary with an embedded SQLite database.** Deploy it anywhere. Own your data. Every feature Auth0 offers — OAuth, passkeys, MFA, SSO, RBAC, audit logs, M2M API keys — ships in one binary that runs on a $5 VPS.

> **sharkauth's purpose: production-grade authentication you actually control.**

---

## architecture

sharkauth follows a clean layered architecture. Every component is internal — no public Go packages, just one HTTP API surface.

```
HTTP Request
    |
    v
Global Middleware (request ID, rate limit, CORS, logging, panic recovery)
    |
    v
Chi Router (/api/v1/*)
    |
    v
Feature Middleware (session auth, admin key, MFA enforcement, API key + scopes)
    |
    v
Handler (JSON parsing, input validation, response encoding)
    |
    v
Service Layer (auth/, rbac/, sso/, audit/)
    |
    v
Storage Layer (SQLite + WAL mode)
    |
    v
Audit Logger (async event capture)
```

```
cmd/shark/
  main.go                         # entry point, migrations, graceful shutdown
  migrations/
    00001_init.sql                 # full schema

internal/
  api/
    router.go                     # chi route registration
    auth_handlers.go              # signup, login, logout, me, password reset
    passkey_handlers.go           # WebAuthn registration + login
    mfa_handlers.go               # TOTP enroll, challenge, recovery
    oauth_handlers.go             # OAuth start + callback
    magiclink_handlers.go         # magic link send + verify
    apikey_handlers.go            # M2M API key CRUD + rotation
    rbac_handlers.go              # role + permission management
    sso_handlers.go               # SSO connection CRUD, SAML/OIDC flows
    audit_handlers.go             # audit log queries + CSV export
    middleware/                   # auth, CORS, rate limit, admin key, API key

  auth/
    session.go                    # AES-256 encrypted cookies, server-side sessions
    password.go                   # Argon2id hashing, bcrypt migration path
    passkey.go                    # WebAuthn/FIDO2 manager
    mfa.go                        # TOTP generation, recovery codes
    oauth.go                      # OAuth provider orchestration
    magiclink.go                  # token generation, email dispatch
    apikey.go                     # sk_live_ key generation, SHA-256 hashing
    providers/                    # Google, GitHub, Apple, Discord

  storage/
    storage.go                    # Store interface (300+ methods)
    sqlite.go                     # SQLite implementation
    migrations.go                 # Goose migration runner

  config/                         # YAML + ${ENV_VAR} interpolation (koanf)
  email/                          # SMTP + Resend HTTP API, HTML templates
  rbac/                           # permission checking, wildcard matching
  sso/                            # SAML 2.0 + OIDC provider flows
  audit/                          # event logging, retention, middleware
  user/                           # user CRUD with nanoid generation
  testutil/                       # test server, in-memory DB, factories, mock email
```

---

## features

sharkauth ships everything you need to replace a managed auth provider:

- **password authentication** — Argon2id hashing with automatic bcrypt migration for Auth0 imports. no password reset required during migration.
- **OAuth / social login** — Google, GitHub, Apple, Discord. add more by implementing one interface.
- **passkeys / WebAuthn** — FIDO2-compliant passwordless authentication. register multiple credentials per user.
- **magic links** — passwordless email login with rate limiting and anti-enumeration.
- **MFA (TOTP)** — Google Authenticator compatible. 10 single-use recovery codes hashed with bcrypt.
- **SSO** — SAML 2.0 and OIDC enterprise connections. domain-based auto-routing for multi-tenant.
- **RBAC** — roles, permissions, wildcard matching (`users:*`). fine-grained access control.
- **M2M API keys** — `sk_live_` prefixed keys with scopes, per-key rate limits, rotation, and revocation.
- **audit logs** — every auth event captured automatically. cursor-based pagination, filtering, CSV export.
- **session management** — AES-256 encrypted cookies, server-side validation, MFA-aware sessions.

---

## security

### cryptographic standards

| feature | implementation |
|---------|---------------|
| password hashing | Argon2id (64MB memory, 3 iterations, 2 threads) |
| password complexity | uppercase + lowercase + digit + common password rejection |
| session encryption | AES-256 + HMAC-SHA256 via gorilla/securecookie |
| field encryption | AES-256-GCM for MFA secrets and SSO credentials at rest |
| API key storage | SHA-256 hash, key shown once at creation |
| MFA recovery codes | bcrypt cost 10, single-use |
| magic link tokens | SHA-256 hash of crypto/rand token |
| passkeys | FIDO2/WebAuthn with stored public keys |
| security headers | CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy |
| account lockout | 5 failed attempts = 15-minute lockout per email |

### timing attack prevention

every secret comparison uses constant-time operations:

- password verification: `subtle.ConstantTimeCompare` on Argon2id output
- admin key validation: `subtle.ConstantTimeCompare`
- API key validation: `subtle.ConstantTimeCompare` on SHA-256 hash
- OAuth state validation: `subtle.ConstantTimeCompare`
- recovery codes: `bcrypt.CompareHashAndPassword` (inherently constant-time)

### anti-enumeration

- login returns generic `"Invalid email or password"` — never reveals if email exists
- magic link always returns `"Check your email"` regardless of account existence
- password reset follows the same pattern

### rate limiting

- **global:** token bucket, 100 req/s burst per IP
- **magic links:** 1 per email per 60 seconds
- **API keys:** configurable per-key rate limit (default 1000 req/hour)

### session security

```
cookie: shark_session
flags:  HttpOnly=true, SameSite=Lax, Secure=<auto from base_url>
encryption: AES-256 block cipher
signing: HMAC-SHA256
storage: server-side (sessions table)
```

---

## API

**base path:** `/api/v1`

### authentication

```
POST   /auth/signup                          # create account
POST   /auth/login                           # password login
POST   /auth/logout                          # destroy session
GET    /auth/me                              # current user (requires session + MFA)
DELETE /auth/me                              # delete own account (GDPR self-deletion)
```

### email verification

```
POST   /auth/email/verify/send              # send verification email (authed)
GET    /auth/email/verify?token=...          # verify email with token
```

### password management

```
POST   /auth/password/send-reset-link        # send reset email
POST   /auth/password/reset                  # reset with token
POST   /auth/password/change                 # change (authed + MFA)
```

### passkeys (WebAuthn)

```
POST   /auth/passkey/register/begin          # start registration (authed)
POST   /auth/passkey/register/finish         # complete registration
POST   /auth/passkey/login/begin             # start login (public)
POST   /auth/passkey/login/finish            # complete login
GET    /auth/passkey/credentials             # list credentials (authed)
DELETE /auth/passkey/credentials/{id}        # delete credential
PATCH  /auth/passkey/credentials/{id}        # rename credential
```

### magic links

```
POST   /auth/magic-link/send                # send login link
GET    /auth/magic-link/verify?token=...     # verify + create session
```

### MFA (TOTP)

```
POST   /auth/mfa/enroll                      # generate secret + QR (authed + MFA)
POST   /auth/mfa/verify                      # confirm first code, enable MFA
POST   /auth/mfa/challenge                   # verify TOTP after login
POST   /auth/mfa/recovery                    # use recovery code
GET    /auth/mfa/recovery-codes              # view codes (shown once)
DELETE /auth/mfa                             # disable MFA
```

### OAuth

```
GET    /auth/oauth/{provider}                # redirect to provider
GET    /auth/oauth/{provider}/callback       # handle callback
```

providers: `google`, `github`, `apple`, `discord`

### SSO

```
POST   /sso/connections                      # create connection (admin)
GET    /sso/connections                      # list connections (admin)
GET    /sso/connections/{id}                 # get connection (admin)
PUT    /sso/connections/{id}                 # update connection (admin)
DELETE /sso/connections/{id}                 # delete connection (admin)

GET    /sso/saml/{id}/metadata               # SAML SP metadata (public)
POST   /sso/saml/{id}/acs                    # SAML assertion consumer (public)
GET    /sso/oidc/{id}/auth                   # OIDC authorization (public)
GET    /sso/oidc/{id}/callback               # OIDC callback (public)

GET    /auth/sso                             # auto-route by domain header
```

### RBAC (admin — requires `Authorization: Bearer sk_live_...`)

```
POST   /roles                                # create role
GET    /roles                                # list roles
GET    /roles/{id}                           # get role
PUT    /roles/{id}                           # update role
DELETE /roles/{id}                           # delete role
POST   /roles/{id}/permissions               # attach permission
DELETE /roles/{id}/permissions/{pid}         # detach permission

POST   /permissions                          # create permission
GET    /permissions                          # list permissions

GET    /users                                # list users (search, pagination)
GET    /users/{id}                           # get user details
PATCH  /users/{id}                           # update user (email, name, metadata)
DELETE /users/{id}                           # delete user (cascades)
POST   /users/{id}/roles                     # assign role to user
DELETE /users/{id}/roles/{rid}               # remove role from user
GET    /users/{id}/roles                     # list user's roles
GET    /users/{id}/permissions               # effective permissions
GET    /users/{id}/audit-logs                # user audit trail

POST   /auth/check                           # check permission {user_id, action, resource}
```

### M2M API keys (admin)

```
POST   /api-keys                             # create key (returns full key once)
GET    /api-keys                             # list keys
GET    /api-keys/{id}                        # get key details
PATCH  /api-keys/{id}                        # update key
DELETE /api-keys/{id}                        # revoke key
POST   /api-keys/{id}/rotate                 # rotate key
```

### audit logs (admin)

```
GET    /audit-logs                           # query logs (cursor pagination)
GET    /audit-logs/{id}                      # get single log
POST   /audit-logs/export                    # CSV export
```

query params: `limit`, `cursor`, `action`, `actor_id`, `target_id`, `status`, `ip`, `from`, `to`

### health

```
GET    /healthz                              # {"status": "ok"} (pings DB for readiness)
```

### sessions (self-service, session cookie)

```
GET    /auth/sessions                        # list own sessions (flags current: true)
DELETE /auth/sessions/{id}                   # revoke one of your own sessions
```

### admin stats (admin — requires `Authorization: Bearer sk_live_...`)

```
GET    /admin/stats                          # users, sessions, MFA, failed logins, keys, SSO
GET    /admin/stats/trends?days=30           # signups_by_day (zero-filled), auth_methods
```

### admin sessions (admin)

```
GET    /admin/sessions                       # all active, keyset cursor pagination
  query: limit, cursor, user_id, auth_method, mfa_passed
DELETE /admin/sessions/{id}                  # revoke any session
GET    /users/{id}/sessions                  # per-user list
DELETE /users/{id}/sessions                  # revoke all for a user (granular audit)
```

### dev inbox (admin, only when started with `--dev`)

```
GET    /admin/dev/emails                     # list captured outbound emails
GET    /admin/dev/emails/{id}                # full HTML/text
DELETE /admin/dev/emails                     # clear (204)
```

### organizations (session cookie; per-handler role gates)

```
POST   /organizations                        # creator becomes owner
GET    /organizations                        # caller's orgs
GET    /organizations/{id}                   # 404 for non-members
PATCH  /organizations/{id}                   # admin+
DELETE /organizations/{id}                   # owner only
GET    /organizations/{id}/members
PATCH  /organizations/{id}/members/{uid}     # admin+, last-owner guard
DELETE /organizations/{id}/members/{uid}     # admin+, last-owner guard
POST   /organizations/{id}/invitations       # email, SHA-256 hashed token
POST   /organizations/invitations/{token}/accept
```

### webhooks (admin; HMAC-SHA256 signed, 5-attempt retry over ~14h)

```
POST   /webhooks                             # returns secret ONCE
GET    /webhooks
GET    /webhooks/{id}
PATCH  /webhooks/{id}
DELETE /webhooks/{id}
POST   /webhooks/{id}/test                   # synthetic webhook.test event
GET    /webhooks/{id}/deliveries             # keyset cursor pagination
```

events: `user.created`, `user.deleted`, `session.revoked`, `organization.created`, `organization.member_added`

signature: `X-Shark-Signature: t=<unix>,v1=<hex(hmac_sha256(secret, t.body))>`

---

## CLI

```bash
shark init                          # interactive setup — writes sharkauth.yaml
shark serve                         # run the server (reads sharkauth.yaml)
shark serve --dev                   # dev mode: ./dev.db, auto secret, dev inbox, relaxed CORS
shark serve --dev --reset           # wipe ./dev.db before starting
shark health --url http://localhost:8080   # probe /healthz
shark version                       # print version (ldflags or module build-info)

# Application management (Phase 3)
shark app create --name "My App" [--callback URL] [--logout URL] [--origin URL]
shark app list                      # list all registered applications
shark app show <id>                 # show application details
shark app update <id> [--name ...] [--callback URL] [--logout URL] [--origin URL]
shark app rotate-secret <id>        # rotate client_secret (new secret shown once)
shark app delete <id>               # delete application

# JWT key management (Phase 3)
shark keys generate-jwt             # generate initial RS256 keypair
shark keys generate-jwt --rotate    # retire active key(s) and generate a new one
```

`shark init` asks one question (base URL — defaults to `http://localhost:8080`) and writes a ready-to-run `sharkauth.yaml`. The 32-byte `secret` is auto-generated and email defaults to the **shark.email testing tier** so the server boots end-to-end with zero extra setup. Requires an interactive terminal. On first `shark serve`, the admin API key is generated and printed to stdout — save it, it is not shown again.

> **shark.email is testing-only.** Rate-limited, no deliverability guarantees, sender = `noreply@shark.email`. Switch to your own provider before any user-facing flow: edit `email:` in `sharkauth.yaml`, run `shark email setup`, or use Settings → Email in the dashboard.

`shark serve --dev` needs **no** config at all. A 32-byte secret is generated each run, emails are captured in the dev inbox instead of being sent, and `/admin/dev/*` endpoints are mounted.

---

## data model

### entity ID prefixes

| prefix | entity |
|--------|--------|
| `usr_` | user |
| `sess_` | session |
| `pk_` | passkey credential |
| `mlt_` | magic link token |
| `mrc_` | MFA recovery code |
| `role_` | role |
| `perm_` | permission |
| `key_` | API key |
| `aud_` | audit log |
| `de_` | dev inbox email (dev mode only) |
| `org_` | organization |
| `inv_` | organization invitation |
| `wh_` | webhook |
| `whd_` | webhook delivery |
| `sk_live_` | API key (client-facing) |
| `whsec_` | webhook signing secret (client-facing) |
| `app_` | application |
| `shark_app_` | application client ID (client-facing) |
| `orgrole_` | org role |

### schema

```sql
users           -- id, email, password_hash, hash_type, name, avatar_url,
                -- mfa_enabled, mfa_secret, mfa_verified, metadata, timestamps

sessions        -- id, user_id, ip, user_agent, mfa_passed, auth_method, expires_at

oauth_accounts  -- user_id, provider, provider_id, email, access_token, refresh_token

passkey_credentials  -- user_id, credential_id (blob), public_key (blob),
                     -- aaguid, sign_count, name, transports, backed_up

magic_link_tokens    -- email, token_hash, used, expires_at

mfa_recovery_codes   -- user_id, code (bcrypt hash), used

roles           -- name (unique), description
permissions     -- action, resource (supports wildcards)
role_permissions     -- role_id, permission_id
user_roles      -- user_id, role_id

sso_connections -- type (saml/oidc), name, domain, config fields, enabled
sso_identities  -- user_id, connection_id, provider_sub

api_keys        -- name, key_hash (SHA-256), key_prefix, key_suffix, scopes (JSON),
                -- rate_limit, expires_at, last_used_at, revoked_at

audit_logs      -- actor_id, actor_type, action, target_type, target_id,
                -- ip, user_agent, metadata (JSON), status, created_at
```

SQLite pragmas: `journal_mode=WAL`, `foreign_keys=ON`

---

## configuration

sharkauth loads config from YAML with environment variable interpolation (`${VAR_NAME}`).

```bash
shark serve --config sharkauth.yaml      # default: sharkauth.yaml
shark serve --dev                        # skip YAML entirely, use ephemeral dev defaults
```

### minimum config — 1 question

`shark init` asks for `base_url` and writes everything else for you. The generated `sharkauth.yaml` is just two blocks:

```yaml
server:
  secret: "<auto-generated 32 bytes>"   # auto by `shark init`
  base_url: "http://localhost:8080"     # the only question asked

email:
  provider: "shark"                     # shark.email testing tier — switch before production
```

That's it. Server boots, email works (via shark.email), dashboard mounts. Everything else (storage path, session lifetime, password rules, CORS) falls back to defaults baked into `config.Load`.

**Production checklist** — before exposing to users, override:
- `email.provider` → `resend` | `smtp` | `ses` | `postmark` | `mailgun` (shark.email is rate-limited, sender locked to `noreply@shark.email`, no SLA)
- `server.base_url` → your real HTTPS URL (drives cookie `Secure` flag, OAuth callbacks, magic link URLs)
- `server.cors_origins` → your frontend origins if calling the API from the browser

Startup validates `secret` and `base_url` in production mode. `shark serve --dev` bypasses entirely (generates a secret, uses the dev inbox, allows any CORS origin).

The legacy `smtp:` block still works as a deprecated alias — existing deployments don't need to migrate immediately.

### environment overrides

pattern: `SHARKAUTH_<SECTION>__<KEY>` (double underscore for nesting)

```bash
SHARKAUTH_SERVER__PORT=9000
SHARKAUTH_AUTH__PASSWORD_MIN_LENGTH=12
SHARKAUTH_SECRET=<your-32-byte-secret>
```

### full config reference

```yaml
server:
  port: 8080
  secret: "${SHARKAUTH_SECRET}"            # REQUIRED: 32+ chars, encrypts sessions + field encryption
  base_url: "https://auth.example.com"     # determines Secure cookie flag
  cors_origins: []                         # empty = same-origin only (see CORS section below)

storage:
  path: "./data/sharkauth.db"

auth:
  session_lifetime: "30d"
  password_min_length: 8                   # also enforces uppercase + lowercase + digit

passkeys:
  rp_name: "MyApp"
  rp_id: "auth.example.com"
  origin: "https://auth.example.com"
  attestation: "none"                      # none | direct | enterprise
  resident_key: "preferred"                # preferred | required | discouraged
  user_verification: "preferred"           # preferred | required | discouraged

magic_link:
  token_lifetime: "10m"
  redirect_url: "https://app.example.com/auth/callback"

password_reset:
  redirect_url: "https://app.example.com/auth/reset-password"

smtp:
  host: "smtp.resend.com"                  # auto-detects Resend HTTP API
  port: 465
  username: "resend"
  password: "${RESEND_API_KEY}"
  from: "noreply@example.com"
  from_name: "MyApp"

mfa:
  issuer: "MyApp"                          # shown in authenticator apps
  recovery_codes: 10

social:
  redirect_url: ""                         # post-OAuth redirect to frontend (empty = return JSON)
  google:
    client_id: "${GOOGLE_CLIENT_ID}"
    client_secret: "${GOOGLE_CLIENT_SECRET}"
    scopes: []                             # empty = defaults (openid, email, profile)
  github:
    client_id: "${GITHUB_CLIENT_ID}"
    client_secret: "${GITHUB_CLIENT_SECRET}"
    scopes: []                             # empty = defaults (user:email)
  apple:
    client_id: "${APPLE_CLIENT_ID}"
    team_id: "${APPLE_TEAM_ID}"
    key_id: "${APPLE_KEY_ID}"
    private_key_path: "./apple_auth_key.p8"
  discord:
    client_id: "${DISCORD_CLIENT_ID}"
    client_secret: "${DISCORD_CLIENT_SECRET}"
    scopes: []                             # empty = defaults (identify, email)

sso:
  saml:
    sp_entity_id: "https://auth.example.com"
  oidc: {}

api_keys:
  default_rate_limit: 1000                 # requests per hour per key
  key_max_lifetime: "365d"

audit:
  retention: "0"                           # 0 = keep forever, "90d" = 90 days
  cleanup_interval: "1h"

# Admin API keys are now managed via the M2M key system.
# On first `shark serve`, an admin key (scope: *) is generated and printed to stdout.
# Use: Authorization: Bearer sk_live_...
```

### JWT configuration

Phase 3 ships RS256 JWT issuance. Enabled by default; mode defaults to `session`.

```yaml
auth:
  jwt:
    enabled: true
    mode: "session"              # "session" = one long-lived JWT per login
                                 # "access_refresh" = short-lived access + refresh pair
    # issuer: ""                 # defaults to server.base_url
    access_token_ttl: "15m"      # ignored in session mode
    refresh_token_ttl: "30d"
    clock_skew: "30s"
    revocation:
      check_per_request: false   # true = every Bearer request hits the DB; adds latency
```

`issuer` auto-derives from `server.base_url` when unset. `audience` defaults to `"shark"`.

Login responses include `token` (session mode) or `access_token` + `refresh_token` (access_refresh mode) alongside the existing `shark_session` cookie. Bearer is accepted on all authenticated endpoints.

`GET /.well-known/jwks.json` is exposed publicly (no auth) for resource servers and edge validators. A signing keypair is auto-generated on first `shark serve`; rotate with `shark keys generate-jwt --rotate`.

> **Deprecation:** `social.redirect_url` and `magic_link.redirect_url` are read at startup to populate the default application's `allowed_callback_urls`. They still work but will be removed in Phase 6. Prefer `shark app update` or the admin API to manage redirect allowlists going forward.

---

### CORS

by default (`cors_origins: []`), sharkauth only allows same-origin requests. the embedded admin dashboard at `/admin` works without CORS since it's served from the same binary.

**when to configure CORS:**

- your frontend app runs on a different origin (e.g., `https://app.example.com` calling `https://auth.example.com/api/v1/`)
- you're developing locally with a frontend dev server on a different port

```yaml
# development: allow your frontend dev server
server:
  cors_origins:
    - "http://localhost:3000"
    - "http://localhost:5173"

# production: allow your frontend domain(s)
server:
  cors_origins:
    - "https://app.example.com"
    - "https://www.example.com"

# NOT recommended for production:
server:
  cors_origins:
    - "*"    # allows any origin — only use for public APIs
```

**what's allowed:**
- methods: `GET, POST, PUT, PATCH, DELETE, OPTIONS`
- headers: `Content-Type, Authorization, X-Admin-Key`
- credentials: `true` (cookies are always sent)
- preflight cache: `86400s` (24 hours)

**important:** if your frontend needs to read the `shark_session` cookie (it shouldn't — the cookie is HttpOnly), you need the exact origin in `cors_origins`, not `*`. wildcard CORS does not allow credentials.

### secret rotation

see [SECRETS.md](SECRETS.md) for the full rotation procedure for `server.secret` and admin API keys.

---

## installation

### build from source

```bash
git clone https://github.com/sharkauth/sharkauth
cd sharkauth
go build -o shark ./cmd/shark
```

### run

```bash
# configure
cp sharkauth.local.yaml sharkauth.yaml
# edit sharkauth.yaml with your secrets

# start
./shark serve --config sharkauth.yaml
```

> requires **Go 1.25+**

---

## usage

```bash
./shark serve                              # start server (default: :8080)
./shark serve --config /etc/sharkauth.yaml # custom config path
```

```
SharkAuth starting on :8080
Health check: http://localhost:8080/healthz
```

### deploy behind a reverse proxy

sharkauth serves HTTP. terminate TLS at your reverse proxy.

```nginx
server {
    listen 443 ssl;
    server_name auth.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

---

## auth flows

### password login with MFA

```
Client                    SharkAuth
  |                          |
  |  POST /auth/login        |
  |  {email, password}       |
  |------------------------->|
  |                          |  verify Argon2id hash
  |                          |  create session (mfa_passed=false)
  |  200 + shark_session     |
  |  {user, mfa_required}    |
  |<-------------------------|
  |                          |
  |  POST /auth/mfa/challenge|
  |  {code: "123456"}        |
  |------------------------->|
  |                          |  validate TOTP (+-30s tolerance)
  |                          |  upgrade session: mfa_passed=true
  |  200 {user}              |
  |<-------------------------|
```

### OAuth flow

```
Client                    SharkAuth              Provider
  |                          |                      |
  |  GET /auth/oauth/google  |                      |
  |------------------------->|                      |
  |  302 -> google.com/auth  |                      |
  |<-------------------------|                      |
  |                          |                      |
  |  (user authorizes)       |                      |
  |                          |  GET /callback?code= |
  |                          |--------------------->|
  |                          |  exchange code        |
  |                          |  fetch user info      |
  |                          |<---------------------|
  |                          |  find-or-create user  |
  |                          |  create session       |
  |  302 + shark_session     |                      |
  |<-------------------------|                      |
```

### bcrypt migration (Auth0 import)

```
1. import Auth0 users with bcrypt hashes into users table
2. user logs in with existing password
3. sharkauth detects $2a$ / $2b$ prefix
4. verifies via bcrypt.CompareHashAndPassword
5. re-hashes with Argon2id, updates users.password_hash
6. next login uses Argon2id — transparent to user
```

---

## error format

```json
{
  "error": "invalid_request",
  "message": "Email is required"
}
```

| code | meaning |
|------|---------|
| `invalid_request` | malformed input |
| `invalid_email` | email doesn't pass validation |
| `email_taken` | account already exists |
| `weak_password` | password fails complexity requirements |
| `account_locked` | too many failed login attempts (429) |
| `unauthorized` | no valid session or API key |
| `forbidden` | valid auth, insufficient permissions |
| `mfa_required` | MFA verification needed |
| `rate_limited` | too many requests (429) |
| `internal_error` | server error |
| `not_implemented` | endpoint not yet built |

---

## testing

14 test files cover critical paths: auth flows, handlers, RBAC, SSO, audit, API keys.

```bash
go test ./internal/...                     # run all tests
go test ./internal/api -v                  # verbose handler tests
go test ./internal/auth -run TestArgon2id  # specific test
```

test infrastructure (`internal/testutil/`):
- `TestServer` — wraps httptest with automatic cookie handling
- in-memory SQLite with auto-migration
- test factories for users, sessions, roles
- mock email sender that captures outbound emails

---

## dependencies

| package | purpose |
|---------|---------|
| `go-chi/chi` | HTTP router |
| `modernc.org/sqlite` | pure Go SQLite (no CGO) |
| `gorilla/securecookie` | AES-256 + HMAC session encryption |
| `golang.org/x/crypto` | Argon2id, bcrypt |
| `golang.org/x/oauth2` | OAuth2 client |
| `go-webauthn/webauthn` | FIDO2 passkeys |
| `pquerna/otp` | TOTP (RFC 6238) |
| `coreos/go-oidc` | OIDC discovery + verification |
| `crewjam/saml` | SAML 2.0 service provider |
| `knadh/koanf` | config loading (YAML + env vars) |
| `pressly/goose` | SQL migrations |
| `matoous/go-nanoid` | URL-safe ID generation |

zero runtime dependencies beyond the binary itself.

---

## middleware stack

```
request
  -> RequestID          # unique ID per request
  -> RealIP             # extract client IP from X-Forwarded-For / X-Real-IP
  -> Logger             # HTTP request/response logging
  -> Recoverer          # panic recovery
  -> SecurityHeaders    # OWASP headers (CSP, HSTS, X-Frame-Options, etc.)
  -> RateLimit          # 100 req/s per IP (token bucket, uses real client IP)
  -> CORS               # if cors_origins configured
  -> [route matched]
  -> RequireSession     # validate shark_session cookie
  -> RequireMFA         # enforce mfa_passed=true
  -> AdminAPIKey        # validate Authorization: Bearer sk_live_* (admin scope)
  -> RequireAPIKey      # validate Bearer sk_live_* token + scopes
  -> handler
```

---

## audit events

every authentication action is captured automatically:

| action | trigger |
|--------|---------|
| `user:signup` | new account created |
| `user:login` | successful authentication |
| `user:logout` | session destroyed |
| `user:password_change` | password updated |
| `user:mfa_enroll` | MFA enrolled |
| `user:mfa_challenge` | MFA verified |
| `user:passkey_register` | passkey added |
| `role:assign` | role assigned to user |
| `permission:check` | permission verification |

each event records: actor, action, target, IP, user agent, metadata (JSON), status, timestamp.

---

## license

MIT. See [LICENSE](LICENSE) for details.

_sharkauth is pre-launch software. security audit in progress._
