<p align="center">
  <img src="https://www.sharkauth.com/shark_whitebg_text_logo.svg" alt="Shark" height="180" />
</p>

<p align="center">
  <strong>The identity platform for the agent era.</strong><br/>
  One binary. Every auth feature. Humans and AI agents. Free forever.
</p>

<p align="center">
  <a href="https://github.com/shark-auth/shark/releases"><img src="https://img.shields.io/github/v/release/shark-auth/shark?style=flat-square&color=0066ff" alt="Release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License" /></a>
  <a href="https://github.com/shark-auth/shark/actions"><img src="https://img.shields.io/github/actions/workflow/status/shark-auth/shark/ci.yml?style=flat-square" alt="CI" /></a>
  <a href="https://discord.gg/sharkauth"><img src="https://img.shields.io/discord/000000000?style=flat-square&label=discord&color=5865F2" alt="Discord" /></a>
  <a href="https://sharkauth.com/docs"><img src="https://img.shields.io/badge/docs-sharkauth.com-black?style=flat-square" alt="Docs" /></a>
</p>

---

Shark is the first open-source identity platform built for AI agents as first-class citizens. MCP-native OAuth 2.1. DPoP-bound tokens. Agent-to-agent delegation with `act` chains. A managed Token Vault so agents never touch raw third-party credentials. Plus every human-auth primitive you expect: password, passkeys, magic links, MFA, SSO, RBAC, organizations, audit logs, webhooks, admin dashboard. One Go binary. ~20MB. SQLite. No external dependencies.

Self-hosted is free forever with zero feature gates. [Shark Cloud](https://sharkauth.com/pricing) starts at $0.

```bash
curl -fsSL https://sharkauth.com/install | sh
shark init
shark serve
```

Open `http://localhost:8080/admin` and you're running.

---

## Why Shark

Every other OSS auth system was built before AI agents existed. We built for agents first, then made sure humans got the same polish.

- **Agents as first-class identities.** `agent_` entities with their own lifecycle, audit trail, consent grants, and delegation chains. Not an add-on, not a paid feature. [Auth0 charges 50% extra for this](https://auth0.com/ai). We include it.
- **MCP-native OAuth 2.1.** `/.well-known/oauth-authorization-server`, dynamic client registration (RFC 7591), resource indicators (RFC 8707), device flow (RFC 8628), DPoP (RFC 9449), token exchange (RFC 8693). MCP clients auto-discover everything.
- **Token Vault.** Managed OAuth tokens for Google, Slack, GitHub, Notion, Linear, Jira, Microsoft. Agents request fresh tokens via bearer delegation. Shark handles refresh, encryption (AES-256-GCM), rotation. Agents never see a refresh token.
- **One binary, zero config.** `shark serve` boots a full auth system with an admin dashboard. No Postgres, no Redis, no Docker compose files.
- **Every feature, free forever.** SSO (SAML + OIDC), organizations, webhooks, RBAC, audit logs, impersonation, migration tools — all in the binary. No enterprise paywalls.
- **Any language, any framework.** REST API + auth proxy with agent scope enforcement. Go, Python, Ruby, PHP, Java, Rust. If it speaks HTTP, it works with Shark.

---

## Quick Start

### Install

```bash
# macOS / Linux
curl -fsSL https://sharkauth.com/install | sh

# Or with Go
go install github.com/shark-auth/shark/cmd/shark@latest

# Or with Docker
docker run -p 8080:8080 -v shark-data:/data ghcr.io/shark-auth/shark
```

### Initialize

```bash
shark init
```

Generates `sharkauth.yaml` and prints your admin API key. **One question** (base URL), done. Email defaults to the shark.email testing tier so the server boots end-to-end with zero extra setup.

> shark.email is **testing-only** (rate-limited, sender locked to `noreply@shark.email`, no SLA). Switch to your own provider before any user-facing flow: `shark email setup`, edit `email:` in `sharkauth.yaml`, or use Settings → Email in the dashboard.

### Run

```bash
shark serve
```

Dashboard at `localhost:8080/admin`. API at `localhost:8080/api/v1/`.

### Dev Mode

```bash
shark serve --dev
```

No email config needed. Magic links print to stdout. Verification emails appear in the dashboard Dev Inbox. In-memory database. Perfect for local development.

### Integrate (TypeScript)

```bash
npm install @sharkauth/js
```

```typescript
import { createSharkClient } from "@sharkauth/js";

const shark = createSharkClient({ baseUrl: "http://localhost:8080" });

// Sign up
await shark.signUp({ email: "alice@example.com", password: "securepass" });

// Sign in
await shark.signIn({ email: "alice@example.com", password: "securepass" });

// Get session
const user = await shark.getUser(); // null if not authenticated
```

### Integrate (Zero Code — Proxy Mode)

```bash
shark proxy --upstream http://localhost:3000
```

Every request to your app gets authenticated automatically. Shark injects headers:

```
X-Shark-User-ID: usr_abc123
X-Shark-User-Email: alice@example.com
X-Shark-User-Roles: admin,editor
X-Shark-Org-ID: org_xyz789
```

Your backend reads headers. No SDK. No auth code. Any language.

---

## Features

### Human Authentication

| Feature                  | Description                                                                        |
| ------------------------ | ---------------------------------------------------------------------------------- |
| **Email / password**     | Argon2id hashing. Password policy enforcement. Bcrypt migration for Auth0 imports. |
| **OAuth / social login** | Google, GitHub, Apple, Discord — extensible provider registry.                     |
| **Passkeys (WebAuthn)**  | FIDO2-compliant passwordless auth. Multiple credentials per user.                  |
| **Magic links**          | Passwordless email login. Rate-limited, anti-enumeration.                          |
| **MFA (TOTP)**           | Google Authenticator compatible. 10 single-use recovery codes.                     |
| **SSO (SAML + OIDC)**    | Enterprise single sign-on. Domain-based auto-routing. Free on every plan.          |
| **Sessions**             | Server-side encrypted cookies (default) or JWT mode. Configurable lifetime.        |

### AI Agent Authentication

| Feature               | Description                                                                                               |
| --------------------- | --------------------------------------------------------------------------------------------------------- |
| **Agent identities**  | First-class `agent_` entities — not users, not service accounts. Own lifecycle, permissions, audit trail. |
| **OAuth 2.1 server**  | Authorization code + PKCE, client credentials, device flow, token exchange. MCP-native.                   |
| **MCP compatibility** | `/.well-known/oauth-authorization-server`, resource indicators, dynamic client registration, CIMD.        |
| **Token Vault**       | Managed OAuth tokens for third-party APIs (Google, Slack, GitHub). Agents never touch raw credentials.    |
| **Delegation chains** | Agent A delegates to Agent B on behalf of User Alice. Full chain tracked in audit log.                    |
| **Device flow**       | Headless agents authenticate without a browser (RFC 8628).                                                |
| **DPoP**              | Proof-of-possession tokens. Stolen agent tokens are useless without the private key.                      |

### Authorization & Management

| Feature             | Description                                                                                  |
| ------------------- | -------------------------------------------------------------------------------------------- |
| **Organizations**   | Multi-tenancy for B2B SaaS. Org-level roles, invitations, SSO enforcement per org.           |
| **RBAC**            | Roles + permissions with wildcard matching (`users:*`). Global and org-scoped.               |
| **M2M API keys**    | `sk_live_` prefixed. Scoped, rate-limited, rotatable. Auto-generated admin key on first run. |
| **Audit logs**      | Every auth event logged. Filterable, exportable (CSV), configurable retention. Agent-aware.  |
| **Webhooks**        | Event-driven notifications. HMAC-SHA256 signed. Exponential retry. Delivery logs.            |
| **Impersonation**   | Time-limited admin sessions as any user. All actions flagged in audit trail.                 |
| **User management** | Admin CRUD, search, metadata, self-deletion (GDPR). Dashboard + CLI + API.                   |

### Platform

| Feature                 | Description                                                                                        |
| ----------------------- | -------------------------------------------------------------------------------------------------- |
| **Admin dashboard**     | Svelte SPA embedded in the binary. Users, sessions, roles, SSO, audit, settings.                   |
| **Auth proxy**          | `shark proxy --upstream` — zero-code auth for any backend via header injection.                    |
| **OIDC provider**       | Shark as an identity provider. "Sign in with [YourApp]." Federation between instances.             |
| **Pre-built UI**        | `<shark-sign-in>`, `<shark-user-button>` — web components that work in any framework.              |
| **Visual flow builder** | Drag-and-drop auth flow customization. Export as YAML.                                             |
| **Compliance toolkit**  | GDPR export, right to erasure, SOC2 access review reports, session geography.                      |
| **Migration tools**     | `shark migrate auth0`, `shark migrate clerk`, `shark migrate supabase`. Password hashes preserved. |
| **shark.email**         | Free transactional email relay. 1,000 emails/mo. No SMTP config needed to get started.             |

### Developer Experience

| Feature             | Description                                                                      |
| ------------------- | -------------------------------------------------------------------------------- |
| **10-second start** | `shark init && shark serve` — running with a dashboard in seconds.               |
| **Dev mode**        | `shark serve --dev` — emails captured in Dev Inbox, no SMTP required.            |
| **CLI**             | `shark users list`, `shark keys rotate`, `shark migrate auth0 export.json`       |
| **< 5KB SDK**       | Cookie-based sessions = no JWT parsing, no token refresh. Tiny client.           |
| **Error docs**      | Every error includes a `docs_url` for instant troubleshooting.                   |
| **Any language**    | REST API. Proxy mode. Go, Python, Ruby, PHP, Java, Rust, .NET — all first-class. |

---

## Competitive Comparison

### Feature Matrix

|                         | **Shark**             | Auth0         | Clerk      | WorkOS           | Ory              | Zitadel        | better-auth  |
| ----------------------- | --------------------- | ------------- | ---------- | ---------------- | ---------------- | -------------- | ------------ |
| Self-hosted             | **Yes (MIT)**         | No            | No         | No               | Yes (Apache)     | Yes (AGPL)     | Yes (MIT)    |
| Single binary           | **Yes**               | --            | --         | --               | No (2+ svcs)     | Yes (needs PG) | No (library) |
| SQLite / zero-config DB | **Yes**               | No            | No         | No               | Yes              | No             | Yes          |
| Framework-agnostic      | **Any language**      | Yes           | JS-first   | Yes              | Yes              | Yes            | JS only      |
| Auth proxy (zero-code)  | **Yes**               | No            | No         | Yes              | Yes (Oathkeeper) | No             | No           |
| Passkeys                | **Yes**               | Yes           | Yes (paid) | Yes              | Yes              | Yes            | Yes          |
| SAML SSO                | **Free**              | $240+ min     | $75/conn   | $125/conn        | Enterprise       | Free           | Free         |
| Organizations           | **Yes**               | Yes           | Yes        | Yes              | Partial          | Yes            | Yes          |
| Webhooks                | **Yes**               | Yes           | Yes        | Yes              | Yes              | Yes            | Partial      |
| Audit logs              | **Yes**               | Yes (2-30d)   | No         | Yes ($125+)      | Yes              | Yes            | Yes          |
| RBAC                    | **Yes (wildcards)**   | Yes           | Yes        | Yes (FGA)        | Yes (Zanzibar)   | Yes            | Yes          |
| Agent / MCP auth        | **Native**            | 50% add-on    | No         | Yes              | No               | No             | Plugin       |
| OAuth 2.1 AS            | **Yes**               | Yes           | Partial    | Yes              | Yes (Hydra)      | Yes            | Yes          |
| Token Vault             | **Yes**               | Yes (30 apps) | No         | No               | No               | No             | No           |
| Device flow             | **Yes**               | Yes           | No         | No               | Yes              | Yes            | Yes          |
| DPoP                    | **Yes**               | Yes           | No         | No               | No               | No             | No           |
| Token exchange          | **Yes**               | Yes           | No         | No               | Yes              | Yes            | No           |
| OIDC provider           | **Yes**               | Yes           | Partial    | Yes              | Yes              | Yes            | Yes          |
| Impersonation           | **Yes**               | Deprecated    | Yes        | Yes              | Partial          | Yes            | Yes          |
| Pre-built UI            | **Web components**    | Hosted page   | React only | Hosted page      | Reference        | Hosted page    | None         |
| CLI                     | **Yes**               | Yes           | No         | Yes              | Yes              | Partial        | Yes          |
| Cookie + JWT modes      | **Configurable**      | Both          | Both       | Both             | Both             | Both           | Both         |
| Migration tools         | **Auth0/Clerk/Supa**  | Import/export | Import     | Migration guides | Partial          | Yes            | Partial      |
| Email built-in          | **Yes + shark.email** | Yes           | Yes        | Yes              | Yes              | Yes            | No           |

### Pricing

| MAU          | **Shark Cloud** | **Shark Self-Hosted** | Auth0      | Clerk          | WorkOS    | Supabase   |
| ------------ | --------------- | --------------------- | ---------- | -------------- | --------- | ---------- |
| 5,000        | **$0**          | **$0**                | $0         | $0             | $0        | $0         |
| 10,000       | **$0**          | **$0**                | $35+       | $0             | $0        | $0         |
| 50,000       | **$49**         | **$0**                | $200+      | $25 + overages | $0        | $25        |
| 100,000      | **$49**         | **$0**                | $2,000+    | ~$1,825        | $0 (+SSO) | ~$25-630   |
| 200,000      | **$249**        | **$0**                | $5,000+    | ~$3,825        | $0 (+SSO) | ~$350      |
| 500,000      | **$249**        | **$0**                | $12,000+   | ~$9,825        | $0 (+SSO) | ~$1,000    |
| + SSO        | **Included**    | **Included**          | $240+ min  | $75/conn       | $125/conn | $0.015/MAU |
| + Agent auth | **Included**    | **Included**          | 50% add-on | N/A            | Included  | N/A        |
| + Orgs       | **Included**    | **Included**          | Included   | Included       | Included  | N/A        |
| + Audit logs | **Included**    | **Included**          | Included   | N/A            | $125+/mo  | N/A        |

### Cloud Tiers

|                           | **Starter** | **Pro**       | **Business**  | **Enterprise**  |
| ------------------------- | ----------- | ------------- | ------------- | --------------- |
| **Price**                 | Free        | $49/mo        | $249/mo       | Custom          |
| **MAU**                   | 5,000       | 50,000        | 500,000       | Unlimited       |
| **SSO connections**       | 1 OIDC      | 3 (SAML+OIDC) | Unlimited     | Unlimited       |
| **Organizations**         | 3           | 50            | Unlimited     | Unlimited       |
| **Webhooks**              | 1 endpoint  | 10 endpoints  | Unlimited     | Unlimited       |
| **Audit retention**       | 7 days      | 90 days       | 1 year        | Custom          |
| **Agent identities**      | 5           | 100           | Unlimited     | Unlimited       |
| **Token Vault providers** | 3           | 20            | Unlimited     | Unlimited       |
| **Dashboard seats**       | 1           | 5             | 20            | Unlimited       |
| **Custom domain**         | --          | Yes           | Yes           | Yes             |
| **Impersonation**         | --          | --            | Yes           | Yes             |
| **Compliance reports**    | --          | --            | SOC2 evidence | Full            |
| **Support**               | Community   | Email (48h)   | Priority (4h) | Dedicated + SLA |
| **Uptime SLA**            | --          | 99.5%         | 99.9%         | 99.99%          |

Self-hosted is always free, always unlimited, always full-featured.

---

## Architecture

```
shark serve
    |
    +-- HTTP Server (chi router)
    |     +-- Global middleware (rate limit, CORS, security headers, logging)
    |     +-- /api/v1/* (REST API — 60+ endpoints)
    |     +-- /oauth/* (OAuth 2.1 authorization server)
    |     +-- /admin/* (embedded Svelte dashboard)
    |     +-- /.well-known/* (OIDC/OAuth discovery, JWKS)
    |
    +-- Auth Engine
    |     +-- Password (Argon2id, bcrypt migration)
    |     +-- OAuth (Google, GitHub, Apple, Discord)
    |     +-- Passkeys (WebAuthn/FIDO2)
    |     +-- Magic Links (SHA-256 tokens, rate-limited)
    |     +-- MFA (TOTP + recovery codes)
    |     +-- SSO (SAML 2.0 SP + OIDC client)
    |     +-- Sessions (encrypted cookies or JWT, configurable)
    |
    +-- Agent Auth (OAuth 2.1)
    |     +-- Client credentials, auth code + PKCE, device flow
    |     +-- Token exchange (RFC 8693, delegation chains)
    |     +-- DPoP (RFC 9449, proof-of-possession)
    |     +-- Dynamic client registration (RFC 7591)
    |     +-- JWT signing (ES256, RS256)
    |     +-- Token Vault (managed third-party OAuth)
    |
    +-- Storage (SQLite WAL, embedded)
    |     +-- Users, sessions, roles, permissions, orgs
    |     +-- Agents, OAuth tokens, consents, device codes
    |     +-- Audit logs, webhooks, vault connections
    |     +-- Field encryption (AES-256-GCM)
    |
    +-- Webhook Dispatcher (async, HMAC-signed, retry with backoff)
    +-- Audit Logger (every auth event, filterable, exportable)
    +-- Email (SMTP, Resend, shark.email, dev inbox)
```

Single process. Single file database. Everything embedded.

---

## Security

| Layer              | Implementation                                     |
| ------------------ | -------------------------------------------------- |
| Password hashing   | Argon2id (64MB, 3 iterations, 2 threads)           |
| Session encryption | AES-256 + HMAC-SHA256                              |
| API key storage    | SHA-256 (never plaintext)                          |
| MFA recovery codes | bcrypt (cost 10, single-use)                       |
| Field encryption   | AES-256-GCM (MFA secrets, OAuth tokens, SSO creds) |
| Token binding      | DPoP (RFC 9449) for agent tokens                   |
| PKCE               | S256 mandatory on all OAuth flows                  |
| Anti-enumeration   | Constant-time comparisons, generic error messages  |
| Rate limiting      | Per-IP, per-API-key, per-endpoint                  |
| Account lockout    | 5 failures = 15-minute lockout                     |
| Security headers   | CSP, HSTS, X-Frame-Options, X-Content-Type-Options |
| Refresh tokens     | Rotation with family-based reuse detection         |
| Request limits     | MaxBytesReader on all JSON endpoints               |

---

## SDKs

### Client SDKs

```bash
npm install @sharkauth/js        # Core (< 5KB gzipped, zero deps)
npm install @sharkauth/react     # React hooks + components
npm install @sharkauth/svelte    # Svelte stores + components
npm install @sharkauth/vue       # Vue composables + components
```

### Pre-Built UI Components

```html
<!-- Works in any framework — React, Vue, Svelte, Angular, vanilla HTML -->
<script src="https://cdn.sharkauth.com/ui.js"></script>

<shark-sign-in
  base-url="https://auth.example.com"
  providers="google,github,passkey"
  theme="dark"
>
</shark-sign-in>
```

Customize in the dashboard visual editor. Click "Copy Code." The code is yours — not a locked component.

### Admin SDKs

```bash
npm install @sharkauth/node      # Node.js admin SDK
pip install git+https://github.com/sharkauth/sharkauth#subdirectory=sdk/python  # Python SDK (PyPI release coming after dogfood validation)
go get github.com/shark-auth/go   # Go admin SDK
```

### Or Just Use the API

Every feature is available via REST. No SDK required.

```bash
# Sign up a user
curl -X POST https://auth.example.com/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com", "password": "securepass"}'

# Get current user
curl https://auth.example.com/api/v1/auth/me \
  -H "Cookie: shark_session=..."

# Create an agent
curl -X POST https://auth.example.com/api/v1/agents \
  -H "Authorization: Bearer sk_live_..." \
  -d '{"name": "DeployBot", "scopes": ["deploy:create"], "grant_types": ["client_credentials"]}'
```

---

## CLI

```bash
shark init                              # Interactive setup
shark serve                             # Start server
shark serve --dev                       # Dev mode (no email config needed)
shark health                            # Check running instance

shark users list                        # List users
shark users create --email alice@co.io  # Create user
shark keys create --scopes "users:*"    # Create API key
shark keys rotate key_abc123            # Rotate key

shark migrate auth0 export.json         # Import from Auth0
shark migrate clerk export.json         # Import from Clerk
shark migrate supabase export.json      # Import from Supabase

shark proxy --upstream http://localhost:3000  # Auth proxy mode
```

---

## Configuration

```yaml
server:
  port: 8080
  secret: "${SHARKAUTH_SECRET}" # 32+ bytes, required
  base_url: "https://auth.example.com"

auth:
  session_mode: "cookie" # cookie | jwt
  session_lifetime: "30d"
  password_min_length: 8

email:
  provider: "shark" # shark | resend | sendgrid | ses | postmark | mailgun | smtp
  api_key: "${SHARK_EMAIL_KEY}"

social:
  google:
    client_id: "${GOOGLE_CLIENT_ID}"
    client_secret: "${GOOGLE_CLIENT_SECRET}"
  github:
    client_id: "${GITHUB_CLIENT_ID}"
    client_secret: "${GITHUB_CLIENT_SECRET}"

passkeys:
  rp_name: "MyApp"
  rp_id: "auth.example.com"

mfa:
  issuer: "MyApp"

audit:
  retention: "90d" # 0 = forever
```

Every config value can be overridden with environment variables: `SHARKAUTH_SERVER__PORT=9000`.

---

## Cloud

[Shark Cloud](https://sharkauth.com) runs the same binary you self-host. Each tenant gets their own SQLite database — not a row filter, a real physical database. Your data never touches anyone else's.

```bash
# Migrate from self-hosted to cloud (or back) in one command
shark cloud migrate --to cloud
shark cloud migrate --to self-hosted
```

Cloud sells operational convenience, not features. Every auth capability in the binary is available to self-hosted users at $0 forever.

---

## Agent Auth (MCP-Native)

Shark is the first open-source identity platform with native AI agent authentication.

### Register an Agent

```bash
curl -X POST https://auth.example.com/api/v1/agents \
  -H "Authorization: Bearer sk_live_admin..." \
  -d '{
    "name": "DeployBot",
    "grant_types": ["client_credentials", "authorization_code"],
    "scopes": ["deploy:create", "deploy:status"],
    "redirect_uris": ["http://localhost:8080/callback"]
  }'
```

### Agent Gets a Token (Client Credentials)

```bash
curl -X POST https://auth.example.com/oauth/token \
  -d "grant_type=client_credentials&client_id=...&client_secret=...&scope=deploy:create"
```

### Agent Acts on Behalf of a User (Auth Code + PKCE)

The user approves via consent screen. The agent receives a scoped, time-limited token.

### Agent Delegates to Another Agent (Token Exchange)

```bash
curl -X POST https://auth.example.com/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange\
  &subject_token=eyJ...agent_a_token\
  &subject_token_type=urn:ietf:params:oauth:token-type:access_token\
  &scope=deploy:create"
```

The resulting token carries a delegation chain: `User Alice -> Agent A -> Agent B`.

### Headless Agent (Device Flow)

```bash
# Agent requests device code
curl -X POST https://auth.example.com/oauth/device \
  -d "client_id=...&scope=deploy:create"

# Response: { "user_code": "SHARK-ABCD", "verification_uri": "https://auth.example.com/oauth/device/verify" }
# User visits URL, enters code, approves. Agent polls token endpoint.
```

### Token Vault (Third-Party API Access)

```bash
# Agent requests a fresh Google Calendar token for the current user
curl https://auth.example.com/api/v1/vault/google_calendar/token \
  -H "Authorization: Bearer eyJ...agent_token"

# Shark handles OAuth, storage, refresh. Agent never sees the refresh token.
```

### MCP Discovery

```bash
curl https://auth.example.com/.well-known/oauth-authorization-server
# Returns RFC 8414 metadata — MCP clients auto-discover everything
```

---

## Data Model

| Prefix              | Entity             |
| ------------------- | ------------------ |
| `usr_`              | User               |
| `sess_`             | Session            |
| `org_`              | Organization       |
| `inv_`              | Invitation         |
| `role_`             | Role               |
| `perm_`             | Permission         |
| `pk_`               | Passkey credential |
| `mlt_`              | Magic link token   |
| `mrc_`              | MFA recovery code  |
| `key_` / `sk_live_` | API key            |
| `aud_`              | Audit log entry    |
| `sso_`              | SSO connection     |
| `agent_`            | Agent identity     |
| `token_`            | OAuth token        |
| `wh_`               | Webhook            |
| `vc_`               | Vault connection   |
| `vp_`               | Vault provider     |

---

## API Reference

Full reference at [sharkauth.com/docs/api](https://sharkauth.com/docs/api).

<details>
<summary><strong>Authentication</strong> (15 endpoints)</summary>

| Method | Endpoint                           | Description             |
| ------ | ---------------------------------- | ----------------------- |
| POST   | `/api/v1/auth/signup`              | Create account          |
| POST   | `/api/v1/auth/login`               | Sign in                 |
| POST   | `/api/v1/auth/logout`              | Sign out                |
| GET    | `/api/v1/auth/me`                  | Current user            |
| POST   | `/api/v1/auth/password/change`     | Change password         |
| POST   | `/api/v1/auth/password/reset/send` | Send reset link         |
| POST   | `/api/v1/auth/password/reset`      | Reset with token        |
| GET    | `/api/v1/auth/sessions`            | List my sessions        |
| DELETE | `/api/v1/auth/sessions/{id}`       | Revoke session          |
| DELETE | `/api/v1/auth/me`                  | Delete my account       |
| POST   | `/api/v1/auth/email/verify/send`   | Send verification       |
| GET    | `/api/v1/auth/email/verify`        | Verify email            |
| GET    | `/api/v1/auth/sso`                 | SSO auto-route by email |
| GET    | `/api/v1/auth/consents`            | My agent consents       |
| DELETE | `/api/v1/auth/consents/{id}`       | Revoke agent consent    |

</details>

<details>
<summary><strong>OAuth / Social</strong> (2 endpoints per provider)</summary>

| Method | Endpoint                                 | Description      |
| ------ | ---------------------------------------- | ---------------- |
| GET    | `/api/v1/auth/oauth/{provider}/start`    | Begin OAuth flow |
| GET    | `/api/v1/auth/oauth/{provider}/callback` | OAuth callback   |

Providers: `google`, `github`, `apple`, `discord`.

</details>

<details>
<summary><strong>Passkeys</strong> (6 endpoints)</summary>

| Method | Endpoint                                | Description           |
| ------ | --------------------------------------- | --------------------- |
| POST   | `/api/v1/auth/passkeys/register/begin`  | Start registration    |
| POST   | `/api/v1/auth/passkeys/register/finish` | Complete registration |
| POST   | `/api/v1/auth/passkeys/login/begin`     | Start login           |
| POST   | `/api/v1/auth/passkeys/login/finish`    | Complete login        |
| GET    | `/api/v1/auth/passkeys`                 | List credentials      |
| DELETE | `/api/v1/auth/passkeys/{id}`            | Remove credential     |

</details>

<details>
<summary><strong>Magic Links</strong> (2 endpoints)</summary>

| Method | Endpoint                         | Description      |
| ------ | -------------------------------- | ---------------- |
| POST   | `/api/v1/auth/magic-link/send`   | Send magic link  |
| GET    | `/api/v1/auth/magic-link/verify` | Verify and login |

</details>

<details>
<summary><strong>MFA</strong> (6 endpoints)</summary>

| Method | Endpoint                          | Description           |
| ------ | --------------------------------- | --------------------- |
| POST   | `/api/v1/auth/mfa/enroll`         | Start TOTP enrollment |
| POST   | `/api/v1/auth/mfa/verify`         | Verify and enable MFA |
| POST   | `/api/v1/auth/mfa/challenge`      | Verify TOTP code      |
| POST   | `/api/v1/auth/mfa/recovery`       | Use recovery code     |
| DELETE | `/api/v1/auth/mfa`                | Disable MFA           |
| GET    | `/api/v1/auth/mfa/recovery-codes` | View recovery codes   |

</details>

<details>
<summary><strong>Organizations</strong> (10 endpoints)</summary>

| Method | Endpoint                                           | Description        |
| ------ | -------------------------------------------------- | ------------------ |
| POST   | `/api/v1/organizations`                            | Create org         |
| GET    | `/api/v1/organizations`                            | List my orgs       |
| GET    | `/api/v1/organizations/{id}`                       | Get org            |
| PATCH  | `/api/v1/organizations/{id}`                       | Update org         |
| DELETE | `/api/v1/organizations/{id}`                       | Delete org         |
| GET    | `/api/v1/organizations/{id}/members`               | List members       |
| POST   | `/api/v1/organizations/{id}/invitations`           | Invite member      |
| DELETE | `/api/v1/organizations/{id}/members/{uid}`         | Remove member      |
| PATCH  | `/api/v1/organizations/{id}/members/{uid}`         | Update member role |
| POST   | `/api/v1/organizations/invitations/{token}/accept` | Accept invite      |

</details>

<details>
<summary><strong>SSO</strong> (9 endpoints)</summary>

| Method | Endpoint                         | Description             |
| ------ | -------------------------------- | ----------------------- |
| POST   | `/api/v1/sso/connections`        | Create connection       |
| GET    | `/api/v1/sso/connections`        | List connections        |
| GET    | `/api/v1/sso/connections/{id}`   | Get connection          |
| PUT    | `/api/v1/sso/connections/{id}`   | Update connection       |
| DELETE | `/api/v1/sso/connections/{id}`   | Delete connection       |
| GET    | `/api/v1/sso/saml/{id}/metadata` | SAML SP metadata        |
| POST   | `/api/v1/sso/saml/{id}/acs`      | SAML assertion consumer |
| GET    | `/api/v1/sso/oidc/{id}/auth`     | OIDC auth redirect      |
| GET    | `/api/v1/sso/oidc/{id}/callback` | OIDC callback           |

</details>

<details>
<summary><strong>RBAC</strong> (10 endpoints)</summary>

| Method | Endpoint                               | Description           |
| ------ | -------------------------------------- | --------------------- |
| POST   | `/api/v1/roles`                        | Create role           |
| GET    | `/api/v1/roles`                        | List roles            |
| GET    | `/api/v1/roles/{id}`                   | Get role              |
| PUT    | `/api/v1/roles/{id}`                   | Update role           |
| DELETE | `/api/v1/roles/{id}`                   | Delete role           |
| POST   | `/api/v1/roles/{id}/permissions`       | Attach permission     |
| DELETE | `/api/v1/roles/{id}/permissions/{pid}` | Detach permission     |
| POST   | `/api/v1/users/{id}/roles`             | Assign role to user   |
| DELETE | `/api/v1/users/{id}/roles/{rid}`       | Remove role from user |
| GET    | `/api/v1/auth/check`                   | Check permission      |

</details>

<details>
<summary><strong>Agents & OAuth 2.1</strong> (15+ endpoints)</summary>

| Method | Endpoint                                  | Description                   |
| ------ | ----------------------------------------- | ----------------------------- |
| POST   | `/api/v1/agents`                          | Register agent                |
| GET    | `/api/v1/agents`                          | List agents                   |
| GET    | `/api/v1/agents/{id}`                     | Get agent                     |
| PATCH  | `/api/v1/agents/{id}`                     | Update agent                  |
| DELETE | `/api/v1/agents/{id}`                     | Deactivate agent              |
| GET    | `/oauth/authorize`                        | Authorization endpoint        |
| POST   | `/oauth/token`                            | Token endpoint (all grants)   |
| POST   | `/oauth/revoke`                           | Revoke token                  |
| POST   | `/oauth/introspect`                       | Introspect token              |
| POST   | `/oauth/register`                         | Dynamic client registration   |
| POST   | `/oauth/device`                           | Device authorization          |
| GET    | `/oauth/device/verify`                    | Device code verification page |
| GET    | `/.well-known/oauth-authorization-server` | AS metadata                   |
| GET    | `/.well-known/jwks.json`                  | Signing keys                  |

</details>

<details>
<summary><strong>Token Vault</strong> (7 endpoints)</summary>

| Method | Endpoint                            | Description             |
| ------ | ----------------------------------- | ----------------------- |
| POST   | `/api/v1/vault/providers`           | Register OAuth provider |
| GET    | `/api/v1/vault/providers`           | List providers          |
| GET    | `/api/v1/vault/connect/{provider}`  | Start OAuth connect     |
| GET    | `/api/v1/vault/callback/{provider}` | OAuth callback          |
| GET    | `/api/v1/vault/{provider}/token`    | Get fresh token         |
| GET    | `/api/v1/vault/connections`         | List my connections     |
| DELETE | `/api/v1/vault/connections/{id}`    | Disconnect              |

</details>

<details>
<summary><strong>Webhooks</strong> (7 endpoints)</summary>

| Method | Endpoint                           | Description     |
| ------ | ---------------------------------- | --------------- |
| POST   | `/api/v1/webhooks`                 | Create webhook  |
| GET    | `/api/v1/webhooks`                 | List webhooks   |
| GET    | `/api/v1/webhooks/{id}`            | Get webhook     |
| PATCH  | `/api/v1/webhooks/{id}`            | Update webhook  |
| DELETE | `/api/v1/webhooks/{id}`            | Delete webhook  |
| POST   | `/api/v1/webhooks/{id}/test`       | Send test event |
| GET    | `/api/v1/webhooks/{id}/deliveries` | Delivery log    |

</details>

<details>
<summary><strong>Admin</strong> (10+ endpoints)</summary>

| Method | Endpoint                                 | Description         |
| ------ | ---------------------------------------- | ------------------- |
| GET    | `/api/v1/admin/stats`                    | Overview metrics    |
| GET    | `/api/v1/admin/sessions`                 | All active sessions |
| DELETE | `/api/v1/admin/sessions/{id}`            | Revoke any session  |
| POST   | `/api/v1/admin/users/{id}/impersonate`   | Impersonate user    |
| GET    | `/api/v1/admin/compliance/access-review` | SOC2 access review  |
| GET    | `/api/v1/users`                          | List users          |
| GET    | `/api/v1/users/{id}`                     | Get user            |
| PATCH  | `/api/v1/users/{id}`                     | Update user         |
| DELETE | `/api/v1/users/{id}`                     | Delete user         |
| GET    | `/api/v1/users/{id}/export`              | GDPR data export    |
| DELETE | `/api/v1/users/{id}/erasure`             | Right to erasure    |

</details>

---

## Self-Hosting

### Requirements

- Any machine that runs Go binaries (Linux, macOS, Windows)
- 64MB RAM minimum
- No external databases, no Redis, no message queues

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -v shark-data:/data \
  -e SHARKAUTH_SECRET=your-32-byte-secret \
  ghcr.io/shark-auth/shark
```

### Binary

```bash
curl -fsSL https://sharkauth.com/install | sh
shark init
shark serve
```

### Kubernetes

Helm chart available at `shark-auth/helm-charts`.

---

## Contributing

Shark is MIT-licensed and open to contributions.

```bash
git clone https://github.com/shark-auth/shark.git
cd shark
make dev    # Runs in dev mode with hot reload
make test   # Run all tests
make lint   # Run linters
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

[MIT](LICENSE) — use Shark however you want. No AGPL, no CLA, no strings.

---

<p align="center">
  <a href="https://sharkauth.com">Website</a> &nbsp;|&nbsp;
  <a href="https://sharkauth.com/docs">Docs</a> &nbsp;|&nbsp;
  <a href="https://discord.gg/sharkauth">Discord</a> &nbsp;|&nbsp;
  <a href="https://github.com/shark-auth/shark">GitHub</a> &nbsp;|&nbsp;
  <a href="https://sharkauth.com/pricing">Pricing</a>
</p>
