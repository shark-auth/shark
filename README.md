# sharkauth

### a self-hosted authentication platform that replaces Auth0

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
| session encryption | AES-256 + HMAC-SHA256 via gorilla/securecookie |
| API key storage | SHA-256 hash, key shown once at creation |
| MFA recovery codes | bcrypt cost 10, single-use |
| magic link tokens | SHA-256 hash of crypto/rand token |
| passkeys | FIDO2/WebAuthn with stored public keys |

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

### RBAC (admin — requires `X-Admin-Key`)

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
GET    /healthz                              # {"status": "ok"}
```

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
| `sk_live_` | API key (client-facing) |

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

api_keys        -- name, key_hash (SHA-256), key_prefix, scopes (JSON),
                -- rate_limit, expires_at, last_used_at, revoked_at

audit_logs      -- actor_id, actor_type, action, target_type, target_id,
                -- ip, user_agent, metadata (JSON), status, created_at
```

SQLite pragmas: `journal_mode=WAL`, `foreign_keys=ON`

---

## configuration

sharkauth loads config from YAML with environment variable interpolation (`${VAR_NAME}`).

```bash
./shark serve --config sharkauth.yaml    # default: sharkauth.yaml
```

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
  secret: "${SHARKAUTH_SECRET}"            # 32+ bytes, encrypts sessions
  base_url: "https://auth.example.com"     # determines Secure cookie flag
  cors_origins: []                         # empty = same-origin only

storage:
  path: "./data/sharkauth.db"

auth:
  session_lifetime: "30d"
  password_min_length: 8

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
  google:
    client_id: "${GOOGLE_CLIENT_ID}"
    client_secret: "${GOOGLE_CLIENT_SECRET}"
  github:
    client_id: "${GITHUB_CLIENT_ID}"
    client_secret: "${GITHUB_CLIENT_SECRET}"
  apple:
    client_id: "${APPLE_CLIENT_ID}"
    team_id: "${APPLE_TEAM_ID}"
    key_id: "${APPLE_KEY_ID}"
    private_key_path: "./apple_auth_key.p8"
  discord:
    client_id: "${DISCORD_CLIENT_ID}"
    client_secret: "${DISCORD_CLIENT_SECRET}"

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

admin:
  api_key: "${SHARKAUTH_ADMIN_KEY}"        # required for admin endpoints
```

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
| `weak_password` | password too short |
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
  -> RealIP             # extract client IP from X-Forwarded-For
  -> Logger             # HTTP request/response logging
  -> Recoverer          # panic recovery
  -> RateLimit          # 100 req/s per IP (token bucket)
  -> CORS               # if cors_origins configured
  -> [route matched]
  -> RequireSession     # validate shark_session cookie
  -> RequireMFA         # enforce mfa_passed=true
  -> AdminAPIKey        # validate X-Admin-Key header
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
