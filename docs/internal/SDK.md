# SharkAuth SDK & Config Strategy

**Date:** 2026-04-13  
**Updated:** 2026-04-25  
**Status:** v0.9.0 â€” core SDK surfaces shipped; framework adapters in progress  
**Goal:** Ship the most ergonomic auth SDK in the market

---

## Shipped SDK surfaces (v0.9.0)

### TypeScript (`sdk/typescript/src/`)

| Module | Class | What it covers |
|--------|-------|----------------|
| `users.ts` | `UsersClient` | CRUD: list, get, create, update, delete users; set tier |
| `oauth.ts` | `OAuthClient` | Token introspection (`POST /oauth/introspect`), revocation (`POST /oauth/revoke`) |
| `magicLink.ts` | `MagicLinkClient` | Send (`POST /auth/magic-link/send`), verify (`GET /auth/magic-link/verify`) |
| `agents.ts` | `AgentsClient` | Agent CRUD, secret rotation, token revocation |

All clients accept `{ baseUrl, token }` and are exported from `index.ts`.

### Python (`sdk/python/shark_auth/`)

| Module | Class | What it covers |
|--------|-------|----------------|
| `users.py` | `UsersClient` | list_users, get_user, create_user, update_user, delete_user, set_user_tier |
| `oauth.py` | `OAuthClient` | introspect, revoke |
| `magic_link.py` | `MagicLinkClient` | send, verify |
| `agents.py` | `AgentsClient` | get_agent (+ existing CRUD) |

All clients accept `base_url` + `token` kwargs.

---

---

## Table of Contents

1. [Competitive Analysis](#competitive-analysis)
2. [SharkAuth's Unfair Advantages](#sharkauths-unfair-advantages)
3. [Config Redesign](#config-redesign)
4. [SDK Architecture](#sdk-architecture)
5. [Package Breakdown](#package-breakdown)
6. [Client SDK (`@sharkauth/browser`)](#client-sdk-sharkauthbrowser)
7. [React SDK (`@sharkauth/react`)](#react-sdk-sharkauthreact)
8. [Next.js SDK (`@sharkauth/next`)](#nextjs-sdk-sharkauthnext)
9. [Svelte SDK (`@sharkauth/svelte`)](#svelte-sdk-sharkauthsvelte)
10. [Admin SDK â€” Node (`@sharkauth/node`)](#admin-sdk--node-sharkauthnode)
11. [Admin SDK â€” Python (`sharkauth-py`)](#admin-sdk--python-sharkauth-py)
12. [Admin SDK â€” Go (`sharkauth-go`)](#admin-sdk--go-sharkauth-go)
13. [Mobile Strategy](#mobile-strategy)
14. [Passkey/WebAuthn Handling](#passkeywebauthn-handling)
15. [Session Architecture & Cross-Tab Sync](#session-architecture--cross-tab-sync)
16. [SSR Cookie Forwarding](#ssr-cookie-forwarding)
17. [Error Handling](#error-handling)
18. [Bundle Size Budget](#bundle-size-budget)
19. [Implementation Roadmap](#implementation-roadmap)

---

## Competitive Analysis

### The Gaps We Attack

| Competitor | Their Weakness | Our Attack |
|-----------|---------------|------------|
| **Clerk** | $550/mo at 10K MAU, per-user pricing, JS-only ecosystem, no self-host | Free forever self-hosted, Go-native, every language gets an SDK |
| **Auth0** | Weeks to integrate, $10K+/mo at scale, terrible docs post-Okta, severe vendor lock-in | 3-line config, transparent pricing, readable docs, zero lock-in |
| **Supabase Auth** | Random logouts, JWT headaches, coupled to Supabase platform, no standalone use | Server-side sessions (no JWT), standalone binary, stable sessions |
| **Firebase** | 80KB+ bundle, Google lock-in, no RBAC/enterprise features on free tier | ~5KB core SDK, independent, enterprise features included |
| **Better Auth** | JS-only, no SCIM, many deps, no Go/Python SDKs | Go-native backend, multi-language SDKs, minimal deps |
| **Stack Auth** | Next.js-only, confusing architecture, no security audit | Framework-agnostic, clear architecture, security-first |
| **Lucia** | Deprecated, single-maintainer, no pre-built UI | Active development, sustainable project, optional UI components |

### Detailed Competitor Pain Points

#### Clerk
- **Pricing is the #1 complaint.** Per-user pricing becomes "downright predatory" at scale. MFA, custom roles, and user banning locked behind paid tiers.
- **JavaScript-only.** Go SDK requires manual JWK caching. No .NET, Java, or meaningful non-JS support.
- **No self-hosting.** Fully managed only. No data residency control. Deal-breaker for regulated industries.
- **Heavy bundle.** Pre-built UI components bundled with auth logic. 5-second polling, Web Worker timers, BroadcastChannel, SafeLock, partitioned cookies â€” all running in the client.
- **Dark patterns.** Fixed 7-day session duration only unlockable on paid plans.
- **Slow support.** "You won't hear from their support for days."

#### Auth0
- **Integration is a nightmare.** Takes weeks, not minutes. Actions, Rules, Hooks create overlapping abstraction layers.
- **Pricing horror.** 300% price hikes. 15x bill escalation with modest growth. M2M token limits create hidden costs. ~12.5x more expensive than Cognito.
- **Post-Okta decay.** Stagnated product, "bloated dashboard in 2026," frequent outages, template-reply support even for $250K/year customers.
- **Documentation is bad.** Scattered, contradictory, incorrect examples. Multiple developers independently confirm.
- **Vendor lock-in is severe.** Migration is a "complete disaster." Password hash incompatibility, social identity relinking required.
- **Performance.** 5x higher latency in Asia-Pacific (600ms vs 120ms).

#### Supabase Auth
- **Random logouts.** The most critical missing config: no session lifetime control. Numerous GitHub reports.
- **JWT headaches.** Legacy 10-year expiry JWTs. Key rotation causes downtime. Migration broke Edge Functions.
- **Coupled to Supabase.** Can't use auth without buying the entire platform. Migration requires complete rewrite.
- **SSR instability.** "Stable in local testing but fails under SSR frameworks, mobile resumes, or multiple tabs."
- **No enterprise features.** No SCIM, deep SSO customization, or fine-grained IAM.

#### Firebase Auth
- **Massive bundle.** `getAuth` alone is ~80KB after minification. Tree-shaking still not effective for auth.
- **Google lock-in.** Data export historically required emailing support. Migration takes 2-4 weeks.
- **No enterprise.** No RBAC, no org management, no flexible MFA options. 58% hit walls within first year.
- **SAML/OIDC pricing surprise.** $0.015/MAU after just 50 users.
- **Abandoned React library.** Firebase's own React library is abandoned.

#### Better Auth
- **TypeScript/JavaScript only.** No Go, Python, or other language SDKs (community Go SDK exists but unofficial).
- **No SCIM.** Multiple commenters said this is a "dealbreaker" for enterprise adoption.
- **Too many dependencies.** TypeScript and Kysely requirements felt heavy.
- **No built-in mail service.** Dealbreaker for drop-in solutions.
- **Poor edge runtime support.** Cloudflare Workers integration challenges.

#### Stack Auth
- **Next.js only.** Requires App Router. No Vite, no other frameworks, no non-JS backends.
- **Confusing architecture.** "After 10 minutes browsing docs I truly can't understand the architecture."
- **No security audit.** No pen testing, no security policy, no responsible disclosure at launch.
- **Young ecosystem.** Small community, limited tutorials, few blog posts.

### What the Market Wants (Cross-Cutting)

1. **No per-user pricing.** The #1 complaint across Clerk and Auth0.
2. **Self-hostable with easy managed option.** Own your data, or let us run it.
3. **Framework-agnostic, language-agnostic.** Every competitor is weak outside JS/TS.
4. **Server-side sessions over JWT.** Lucia proved developers prefer database sessions. JWT creates stale sessions, rotation nightmares, revocation impossibility.
5. **Small SDK footprint.** Firebase at 80KB+ and SuperTokens at 435KB are cautionary tales.
6. **Simple setup, deep when needed.** Clerk wins on "works in 5 minutes." Auth0 loses on "takes weeks."
7. **No vendor lock-in.** Standard formats, easy migration, open source.
8. **Enterprise features without enterprise pricing.** SCIM, SAML, RBAC â€” don't gate them.
9. **Transparent, readable documentation.** Auth0's docs are "really bad." Lucia's were legendary.
10. **Active maintenance.** Lucia died from single-maintainer burnout. Auth0 support degraded post-acquisition.

---

## SharkAuth's Unfair Advantages

### 1. Cookie-Based Sessions = Thin Client SDK

SharkAuth uses encrypted server-side sessions via `gorilla/securecookie`. The browser handles cookie transport natively. This means:

- **No JWT parsing** on the client
- **No token storage** (no localStorage, no sessionStorage)
- **No refresh token management** (no timers, no rotation logic)
- **No token expiry handling** (server manages session lifetime)
- **Core SDK is ~5KB** vs Clerk's massive bundle

The client SDK is literally a typed `fetch` wrapper with `credentials: 'include'`. That's the entire auth transport layer.

### 2. Single Binary, Zero External Dependencies

`./shark serve` â€” one binary, one SQLite file, everything works. No Redis, no Postgres, no external services required. Compare to Auth0 (managed-only) or Better Auth (requires your own database setup).

### 3. Go-Native = Every Language Gets an Admin SDK

The backend is Go, the API is REST+JSON. Admin SDKs for Node, Python, Go, Ruby, Java are trivially thin â€” just typed HTTP wrappers. No complex protocol, no gRPC, no GraphQL.

### 4. Free Forever Self-Hosted

No per-user pricing. No feature gating. Every feature available in self-hosted. Cloud charges for ops, not capabilities.

---

## Config Redesign

### Current Problem

78 lines of YAML to get started. Users must configure passkeys, SSO, MFA, magic links, password reset, audit settings, and API key settings even if they only want email/password auth.

### New Config Philosophy

**Three tiers of config complexity:**

1. **Minimum viable** (3 lines) â€” works for 80% of use cases
2. **Common customization** (10-15 lines) â€” email, OAuth, CORS
3. **Full control** (50+ lines) â€” argon2id params, passkey attestation, audit retention

### Tier 1: Minimum Viable Config

```yaml
url: "https://auth.myapp.com"
secret: "${SHARKAUTH_SECRET}"
app_url: "https://myapp.com"
```

That's it. Everything else has sane defaults:
- Passkey RP ID derived from `url` hostname
- Passkey origin derived from `url`
- CORS auto-allows `app_url`
- Magic link redirect: `{app_url}/auth/callback`
- Password reset redirect: `{app_url}/auth/reset-password`
- OAuth redirect: `{app_url}/auth/callback`
- SSO entity ID: `url`
- Session lifetime: 30 days
- Password min length: 8
- Storage: `./data/sharkauth.db`

### Tier 2: Common Customization

```yaml
url: "https://auth.myapp.com"
secret: "${SHARKAUTH_SECRET}"
app_url: "https://myapp.com"
app_name: "My App"                    # used in emails, passkey RP name, MFA issuer

email:
  provider: resend                     # resend | postmark | sendgrid | smtp
  api_key: "${RESEND_API_KEY}"
  from: "auth@myapp.com"

oauth:
  google:
    client_id: "${GOOGLE_CLIENT_ID}"
    client_secret: "${GOOGLE_CLIENT_SECRET}"
  github:
    client_id: "${GITHUB_CLIENT_ID}"
    client_secret: "${GITHUB_CLIENT_SECRET}"
```

No empty provider blocks. Only declare what you use.

### Tier 3: Full Control

```yaml
url: "https://auth.myapp.com"
secret: "${SHARKAUTH_SECRET}"
app_url: "https://myapp.com"
app_name: "My App"

# Override any derived value
cors_origins:
  - "https://myapp.com"
  - "https://admin.myapp.com"

email:
  provider: smtp
  host: "smtp.example.com"
  port: 587
  username: "user"
  password: "${SMTP_PASSWORD}"
  from: "auth@myapp.com"
  from_name: "My App Auth"

auth:
  session_lifetime: "7d"
  password_min_length: 12

storage:
  path: "/var/lib/sharkauth/data.db"

passkeys:
  rp_name: "Custom RP Name"           # default: app_name
  rp_id: "myapp.com"                  # default: url hostname
  origin: "https://myapp.com"         # default: url
  attestation: "direct"               # default: none
  resident_key: "required"            # default: preferred
  user_verification: "required"       # default: preferred

oauth:
  redirect_url: "https://myapp.com/custom/callback"  # default: {app_url}/auth/callback
  google:
    client_id: "${GOOGLE_CLIENT_ID}"
    client_secret: "${GOOGLE_CLIENT_SECRET}"
    scopes: ["openid", "email", "profile", "https://www.googleapis.com/auth/calendar"]

magic_link:
  token_lifetime: "15m"               # default: 10m
  redirect_url: "https://myapp.com/custom/magic"  # default: {app_url}/auth/callback

password_reset:
  redirect_url: "https://myapp.com/custom/reset"   # default: {app_url}/auth/reset-password

mfa:
  issuer: "Custom Issuer"             # default: app_name
  recovery_codes: 8                   # default: 10

sso:
  saml:
    sp_entity_id: "https://custom.entity"  # default: url

api_keys:
  default_rate_limit: 500
  max_lifetime: "180d"

audit:
  retention: "90d"                    # default: 0 (unlimited)
  cleanup_interval: "6h"             # default: 1h

# Expert: argon2id tuning (don't touch unless you know why)
# argon2id:
#   memory: 65536
#   iterations: 3
#   parallelism: 2
```

### Key Derivation Rules

| Field | Derived From | Override Key |
|-------|-------------|-------------|
| `passkeys.rp_id` | `url` hostname | `passkeys.rp_id` |
| `passkeys.origin` | `url` | `passkeys.origin` |
| `passkeys.rp_name` | `app_name` (default: "SharkAuth") | `passkeys.rp_name` |
| `cors_origins` | `[app_url]` auto-included | `cors_origins` (additive) |
| `magic_link.redirect_url` | `{app_url}/auth/callback` | `magic_link.redirect_url` |
| `password_reset.redirect_url` | `{app_url}/auth/reset-password` | `password_reset.redirect_url` |
| `oauth.redirect_url` | `{app_url}/auth/callback` | `oauth.redirect_url` |
| `sso.saml.sp_entity_id` | `url` | `sso.saml.sp_entity_id` |
| `mfa.issuer` | `app_name` | `mfa.issuer` |
| `email.from_name` | `app_name` | `email.from_name` |

### Email Provider Shortcuts

Instead of raw SMTP config, support named providers:

```yaml
# Resend (just an API key)
email:
  provider: resend
  api_key: "${RESEND_API_KEY}"
  from: "auth@myapp.com"

# Postmark (just an API key)
email:
  provider: postmark
  api_key: "${POSTMARK_API_KEY}"
  from: "auth@myapp.com"

# SendGrid (just an API key)
email:
  provider: sendgrid
  api_key: "${SENDGRID_API_KEY}"
  from: "auth@myapp.com"

# Raw SMTP (full control)
email:
  provider: smtp
  host: "smtp.example.com"
  port: 587
  username: "user"
  password: "${SMTP_PASSWORD}"
  from: "auth@myapp.com"
```

Provider shortcut mapping:
- `resend` -> host: `smtp.resend.com`, port: 465, username: `resend`, password: api_key
- `postmark` -> host: `smtp.postmarkapp.com`, port: 587, username: api_key, password: api_key
- `sendgrid` -> host: `smtp.sendgrid.net`, port: 587, username: `apikey`, password: api_key

### Backward Compatibility

The current config format (`server.port`, `server.secret`, `server.base_url`, etc.) continues to work. The new top-level keys (`url`, `app_url`, `app_name`) are aliases. If both are present, the new keys take precedence. Migration guide:

```yaml
# Old (still works)                    # New (preferred)
server:                                url: "https://auth.myapp.com"
  port: 8080                           # port: 8080 (optional, default 8080)
  secret: "..."                        secret: "..."
  base_url: "https://auth.myapp.com"   app_url: "https://myapp.com"
  cors_origins: [...]                  # auto-derived from app_url
```

### Environment Variable Overrides

The `SHARKAUTH_` prefix convention stays, plus new convenience vars:

```bash
SHARKAUTH_URL="https://auth.myapp.com"
SHARKAUTH_SECRET="..."
SHARKAUTH_APP_URL="https://myapp.com"
SHARKAUTH_APP_NAME="My App"
SHARKAUTH_EMAIL__PROVIDER="resend"
SHARKAUTH_EMAIL__API_KEY="re_..."
SHARKAUTH_EMAIL__FROM="auth@myapp.com"
SHARKAUTH_OAUTH__GOOGLE__CLIENT_ID="..."
SHARKAUTH_OAUTH__GOOGLE__CLIENT_SECRET="..."
```

For Docker deployments, environment variables are the primary config method. The YAML file becomes optional â€” you can run SharkAuth with zero files:

```bash
docker run -e SHARKAUTH_URL=https://auth.myapp.com \
           -e SHARKAUTH_SECRET=... \
           -e SHARKAUTH_APP_URL=https://myapp.com \
           -v data:/data \
           sharkauth/sharkauth
```

---

## SDK Architecture

### Design Principles

1. **Cookie-based = thin client.** The browser handles auth transport. Our SDK is a typed fetch wrapper, not an auth engine.
2. **Framework-agnostic core, framework-native bindings.** One core package, thin adapters per framework.
3. **Admin SDK is separate.** Server-to-server operations don't pollute client bundles.
4. **Zero required dependencies in core.** Optional deps for passkeys (@simplewebauthn/browser).
5. **Every flow is one function call.** Complex ceremonies (passkeys, MFA, OAuth) are handled internally.
6. **TypeScript-first.** Full type safety, discriminated unions for auth states, generics for metadata.
7. **Reactive state.** Session state is observable. Frameworks bind to it natively.

### Frontend vs Backend

| | Frontend SDKs | Backend/Admin SDKs |
|---|---|---|
| **Auth mechanism** | `shark_session` cookie (automatic) | `Authorization: Bearer sk_live_*` header |
| **Runs in** | Browser, SSR | Server (Node.js, Python, Go) |
| **Use case** | User-facing auth flows | User management, RBAC, audit, M2M |
| **Session management** | Cookie-based, server-owned | Stateless API key |
| **Bundle concern** | Yes (keep small) | No (server-side) |
| **Packages** | `@sharkauth/browser`, `/react`, `/next`, `/svelte` | `@sharkauth/node`, `sharkauth-py`, `sharkauth-go` |

---

## Package Breakdown

```
@sharkauth/core           ~3KB  Shared types, error codes, constants (no runtime)
@sharkauth/browser        ~5KB  Browser client (fetch wrapper, session state, cross-tab sync)
@sharkauth/react          ~2KB  React hooks + context provider
@sharkauth/next           ~3KB  Next.js middleware + server helpers
@sharkauth/svelte         ~2KB  Svelte stores + SvelteKit hooks
@sharkauth/passkey        ~8KB  WebAuthn wrapper (optional, uses @simplewebauthn/browser)
@sharkauth/node           ~4KB  Admin SDK for Node.js (API key auth)
sharkauth-py              ~3KB  Admin SDK for Python
sharkauth-go              ~2KB  Admin SDK for Go
```

### Dependency Policy

| Package | Dependencies |
|---------|-------------|
| `@sharkauth/core` | **Zero.** Types only. |
| `@sharkauth/browser` | **Zero.** Uses native `fetch`, `BroadcastChannel`, `navigator.locks`. |
| `@sharkauth/react` | `react` (peer dep) |
| `@sharkauth/next` | `next` (peer dep) |
| `@sharkauth/svelte` | `svelte` (peer dep) |
| `@sharkauth/passkey` | `@simplewebauthn/browser` (~8KB, handles WebAuthn edge cases) |
| `@sharkauth/node` | **Zero.** Uses native `fetch` (Node 18+). |
| `sharkauth-py` | `httpx` (async HTTP) |
| `sharkauth-go` | **Zero.** stdlib only. |

Total client bundle for React app with passkeys: **~18KB gzipped.**
Compare: Clerk ~100KB+, Firebase Auth ~80KB+, SuperTokens ~435KB.

---

## Client SDK (`@sharkauth/browser`)

### Initialization

```typescript
import { SharkAuth } from '@sharkauth/browser'

// One argument. That's it.
const shark = new SharkAuth('https://auth.myapp.com')

// Or with options (rarely needed)
const shark = new SharkAuth('https://auth.myapp.com', {
  credentials: 'include',           // default, handles cookies
  onAuthStateChange: (user) => {},   // optional callback
})
```

### Core Auth

```typescript
// Signup
const user = await shark.signUp({ email, password })
const user = await shark.signUp({ email, password, name: 'Jane Doe' })

// Login â€” returns discriminated union
const result = await shark.signIn({ email, password })

if (result.mfaRequired) {
  // TypeScript knows result.mfa is available
  const user = await result.mfa.verify(totpCode)
  // or
  const user = await result.mfa.recover(recoveryCode)
} else {
  // TypeScript knows result.user is available
  console.log(result.user.email)
}

// Current user (null if not logged in)
const user = await shark.getUser()

// Logout
await shark.signOut()
```

### OAuth â€” One Line

```typescript
// Redirects to provider. That's it.
// After callback, user is logged in via cookie. No token handling needed.
shark.signIn.withGoogle()
shark.signIn.withGithub()
shark.signIn.withApple()
shark.signIn.withDiscord()

// Generic provider
shark.signIn.withProvider('google')

// With custom redirect (overrides default)
shark.signIn.withGoogle({ redirectTo: '/dashboard' })
```

### Magic Links

```typescript
await shark.magicLink.send('user@example.com')
// Email received -> user clicks link -> cookie set -> done

// For SPAs that handle the callback:
const user = await shark.magicLink.verify(token) // token from URL query param
```

### Passkeys (requires `@sharkauth/passkey`)

```typescript
import { enablePasskeys } from '@sharkauth/passkey'

// Extend shark instance with passkey methods
const shark = enablePasskeys(new SharkAuth('https://auth.myapp.com'))

// Register â€” full WebAuthn ceremony in one call
await shark.passkey.register()
await shark.passkey.register({ name: 'My YubiKey' })

// Login â€” discoverable flow, one call
const user = await shark.passkey.signIn()

// Login with email hint
const user = await shark.passkey.signIn({ email: 'user@example.com' })

// Manage credentials
const creds = await shark.passkey.list()
await shark.passkey.rename(credId, 'Work Laptop')
await shark.passkey.delete(credId)
```

### MFA Management

```typescript
// Enroll
const { secret, qrUri } = await shark.mfa.enroll()
// Show QR code from qrUri, user scans with authenticator app

// Confirm enrollment (user enters code from authenticator)
const { recoveryCodes } = await shark.mfa.confirmEnrollment(code)
// Show recovery codes to user ONCE

// Disable
await shark.mfa.disable(code)

// Get recovery codes
const codes = await shark.mfa.getRecoveryCodes()
```

### Password Management

```typescript
// Forgot password (public)
await shark.password.sendResetLink('user@example.com')

// Reset password (from reset page, token from URL)
await shark.password.reset(token, newPassword)

// Change password (authenticated)
await shark.password.change(currentPassword, newPassword)
```

### Email Verification

```typescript
// Send verification email (authenticated)
await shark.email.sendVerification()

// Verify (from verification page, token from URL)
await shark.email.verify(token)
```

### SSO

```typescript
// Discover SSO connection by email domain
const sso = await shark.sso.discover('user@bigcorp.com')
// Returns { connectionId, connectionType, redirectUrl }

// Or auto-redirect:
shark.sso.signIn('user@bigcorp.com')
// Redirects to the right IdP automatically
```

### Auth State & Events

```typescript
// Reactive state
shark.onAuthStateChange((user) => {
  if (user) {
    console.log('Logged in:', user.email)
  } else {
    console.log('Logged out')
  }
})

// Current state (synchronous, from cache)
const user = shark.user // User | null
const isLoggedIn = shark.isAuthenticated // boolean
```

### How It Works Internally

Since SharkAuth uses encrypted cookies, the SDK is remarkably simple:

```typescript
// Every API call is just:
const response = await fetch(`${this.baseUrl}/api/v1/auth/me`, {
  credentials: 'include',  // sends shark_session cookie automatically
  headers: { 'Content-Type': 'application/json' },
})
```

No token management. No refresh logic. No localStorage. The browser and server handle everything.

**Cross-tab sync** uses `BroadcastChannel`:
- Tab A logs in -> broadcasts `{ type: 'auth:change', user }` -> Tab B updates
- Tab A logs out -> broadcasts `{ type: 'auth:change', user: null }` -> Tab B updates

**Session refresh** uses focus events:
- User returns to tab -> `getUser()` called -> session state updated
- No polling. No timers. No Web Workers.

---

## React SDK (`@sharkauth/react`)

### Provider (wrap once)

```tsx
import { SharkAuthProvider } from '@sharkauth/react'

function App() {
  return (
    <SharkAuthProvider url="https://auth.myapp.com">
      <MyApp />
    </SharkAuthProvider>
  )
}
```

### Hooks

```tsx
import { useAuth, useSignIn, useSignUp, usePasskey, useMFA } from '@sharkauth/react'

// ---- useAuth: session state ----
function Dashboard() {
  const { user, isLoading, isAuthenticated, signOut } = useAuth()
  
  if (isLoading) return <Spinner />
  if (!user) return <Navigate to="/login" />
  
  return (
    <div>
      <h1>Welcome, {user.name}</h1>
      <p>{user.email} {user.emailVerified ? '(verified)' : '(unverified)'}</p>
      <button onClick={signOut}>Sign out</button>
    </div>
  )
}

// ---- useSignIn: login flow with MFA handling ----
function LoginPage() {
  const { signIn, mfaRequired, verifyMfa, recoverMfa, error, isLoading } = useSignIn()
  
  // MFA step
  if (mfaRequired) {
    return (
      <form onSubmit={(e) => { e.preventDefault(); verifyMfa(code) }}>
        <input placeholder="6-digit code" value={code} onChange={...} />
        <button type="submit">Verify</button>
        <button type="button" onClick={() => recoverMfa(recoveryCode)}>
          Use recovery code
        </button>
      </form>
    )
  }
  
  // Login step
  return (
    <form onSubmit={(e) => { e.preventDefault(); signIn({ email, password }) }}>
      {error && <div className="error">{error.message}</div>}
      <input type="email" placeholder="Email" ... />
      <input type="password" placeholder="Password" ... />
      <button type="submit" disabled={isLoading}>
        {isLoading ? 'Signing in...' : 'Sign in'}
      </button>
    </form>
  )
}

// ---- useSignUp ----
function SignUpPage() {
  const { signUp, error, isLoading } = useSignUp()
  
  return (
    <form onSubmit={(e) => { e.preventDefault(); signUp({ email, password, name }) }}>
      {error && <div className="error">{error.message}</div>}
      ...
    </form>
  )
}

// ---- OAuth buttons ----
import { OAuthButton } from '@sharkauth/react'

function LoginOptions() {
  return (
    <>
      <OAuthButton provider="google">Continue with Google</OAuthButton>
      <OAuthButton provider="github">Continue with GitHub</OAuthButton>
    </>
  )
}
// OAuthButton just calls shark.signIn.withGoogle() on click. Minimal component.
```

### Route Protection

```tsx
import { ProtectedRoute, PublicOnlyRoute } from '@sharkauth/react'

function AppRoutes() {
  return (
    <Routes>
      {/* Redirects to /login if not authenticated */}
      <Route path="/dashboard" element={
        <ProtectedRoute fallback="/login">
          <Dashboard />
        </ProtectedRoute>
      } />
      
      {/* Redirects to /dashboard if already authenticated */}
      <Route path="/login" element={
        <PublicOnlyRoute fallback="/dashboard">
          <LoginPage />
        </PublicOnlyRoute>
      } />
    </Routes>
  )
}
```

---

## Next.js SDK (`@sharkauth/next`)

### Middleware (protect routes)

```typescript
// middleware.ts
import { sharkMiddleware } from '@sharkauth/next'

export default sharkMiddleware({
  sharkUrl: 'https://auth.myapp.com',
  publicRoutes: ['/', '/login', '/signup', '/pricing'],
  // Everything else requires authentication
})

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
}
```

### Server Components

```tsx
// app/dashboard/page.tsx
import { getUser } from '@sharkauth/next/server'

export default async function DashboardPage() {
  const user = await getUser()
  // user is guaranteed non-null (middleware redirected if unauthenticated)
  
  return <h1>Welcome, {user.name}</h1>
}
```

### Route Handlers

```typescript
// app/api/profile/route.ts
import { withAuth } from '@sharkauth/next/server'

export const GET = withAuth(async (req, { user }) => {
  // user is guaranteed non-null
  return Response.json({ profile: user })
})

export const PATCH = withAuth(async (req, { user }) => {
  const body = await req.json()
  // update profile...
  return Response.json({ ok: true })
})
```

### Server Actions

```typescript
// app/actions.ts
'use server'
import { getUser } from '@sharkauth/next/server'

export async function updateProfile(formData: FormData) {
  const user = await getUser()
  if (!user) throw new Error('Unauthorized')
  
  // ...
}
```

### How SSR Cookie Forwarding Works

The Next.js middleware reads the `shark_session` cookie from the incoming request, calls SharkAuth's `/api/v1/auth/me` endpoint (forwarding the cookie), and stores the auth state in request headers. Server components read from these headers without making additional network calls.

```
Browser -> Next.js Middleware -> SharkAuth API (cookie forwarded)
                             |
                    Auth state stored in request headers
                             |
                    Server Component reads headers (no network call)
```

---

## Svelte SDK (`@sharkauth/svelte`)

### SvelteKit Hook

```typescript
// src/hooks.server.ts
import { sharkHandle } from '@sharkauth/svelte/server'

export const handle = sharkHandle({
  sharkUrl: 'https://auth.myapp.com',
  publicRoutes: ['/', '/login', '/signup'],
})
```

### Stores

```svelte
<!-- src/routes/dashboard/+page.svelte -->
<script>
  import { user, isAuthenticated, signOut } from '@sharkauth/svelte'
</script>

{#if $user}
  <h1>Welcome, {$user.name}</h1>
  <button on:click={signOut}>Sign out</button>
{/if}
```

### Page Load

```typescript
// src/routes/dashboard/+page.server.ts
import { getUser } from '@sharkauth/svelte/server'
import { redirect } from '@sveltejs/kit'

export async function load({ cookies }) {
  const user = await getUser(cookies)
  if (!user) throw redirect(303, '/login')
  return { user }
}
```

---

## Admin SDK â€” Node (`@sharkauth/node`)

For server-to-server operations. Uses API key authentication.

```typescript
import { SharkAdmin } from '@sharkauth/node'

const admin = new SharkAdmin({
  url: 'https://auth.myapp.com',
  apiKey: process.env.SHARKAUTH_API_KEY,  // sk_live_...
})

// ---- Users ----
const { users, total } = await admin.users.list()
const { users } = await admin.users.list({ search: 'jane', limit: 10, offset: 0 })
const user = await admin.users.get('usr_abc123')
await admin.users.update('usr_abc123', { name: 'Jane Smith', metadata: { plan: 'pro' } })
await admin.users.delete('usr_abc123')

// ---- RBAC ----
const allowed = await admin.auth.check({
  userId: 'usr_abc123',
  action: 'write',
  resource: 'documents',
})

// Roles
const role = await admin.roles.create({ name: 'editor', description: 'Can edit' })
const roles = await admin.roles.list()
await admin.roles.update(roleId, { description: 'Updated' })
await admin.roles.delete(roleId)

// Permissions
const perm = await admin.permissions.create({ action: 'read', resource: 'documents' })
const perms = await admin.permissions.list()
await admin.roles.attachPermission(roleId, permId)
await admin.roles.detachPermission(roleId, permId)

// User roles
await admin.users.assignRole('usr_abc123', roleId)
await admin.users.removeRole('usr_abc123', roleId)
const userRoles = await admin.users.listRoles('usr_abc123')
const userPerms = await admin.users.listPermissions('usr_abc123')

// ---- API Keys ----
const { key, id } = await admin.apiKeys.create({
  name: 'Worker Service',
  scopes: ['read:users'],
  rateLimit: 500,
})
// key is the full sk_live_... (shown only once)

const keys = await admin.apiKeys.list()
await admin.apiKeys.update(keyId, { name: 'Renamed' })
await admin.apiKeys.revoke(keyId)
const { key: newKey } = await admin.apiKeys.rotate(keyId)

// ---- Audit Logs ----
const { data, nextCursor, hasMore } = await admin.audit.list({
  action: 'login',
  from: '2026-01-01T00:00:00Z',
  limit: 50,
})
const log = await admin.audit.get(logId)
const exported = await admin.audit.export({ from: '2026-01-01', to: '2026-04-01' })

// ---- SSO Connections ----
const conn = await admin.sso.create({
  type: 'oidc',
  name: 'Okta',
  domain: 'bigcorp.com',
  oidcIssuer: 'https://bigcorp.okta.com',
  oidcClientId: '...',
  oidcClientSecret: '...',
})
const connections = await admin.sso.list()
await admin.sso.update(connId, { enabled: false })
await admin.sso.delete(connId)

// ---- User Audit Logs ----
const { data } = await admin.users.auditLogs('usr_abc123', { limit: 20 })
```

---

## Admin SDK â€” Python (`sharkauth-py`)

```python
from sharkauth import SharkAdmin
import os

admin = SharkAdmin(
    url="https://auth.myapp.com",
    api_key=os.environ["SHARKAUTH_API_KEY"],
)

# Users
users = admin.users.list(search="jane", limit=10)
user = admin.users.get("usr_abc123")
admin.users.update("usr_abc123", metadata={"plan": "pro"})
admin.users.delete("usr_abc123")

# RBAC
allowed = admin.auth.check(user_id="usr_abc123", action="write", resource="documents")

# Roles
role = admin.roles.create(name="editor", description="Can edit")
admin.users.assign_role("usr_abc123", role.id)

# Audit
logs = admin.audit.list(action="login", limit=50)

# API Keys
key = admin.api_keys.create(name="Worker", scopes=["read:users"])
print(key.key)  # sk_live_... (shown only once)
```

### Django Integration

```python
# settings.py
SHARKAUTH_URL = "https://auth.myapp.com"
SHARKAUTH_API_KEY = os.environ["SHARKAUTH_API_KEY"]

# middleware.py
from sharkauth.django import SharkAuthMiddleware
MIDDLEWARE = [..., "myapp.middleware.SharkAuthMiddleware"]

# views.py
from sharkauth.django import require_auth

@require_auth
def dashboard(request):
    user = request.shark_user  # populated by middleware
    return JsonResponse({"email": user.email})
```

### FastAPI Integration

```python
from sharkauth.fastapi import SharkAuth, get_user

shark = SharkAuth(url="https://auth.myapp.com", api_key=os.environ["SHARKAUTH_API_KEY"])

@app.get("/dashboard")
async def dashboard(user = Depends(get_user)):
    return {"email": user.email}
```

---

## Admin SDK â€” Go (`sharkauth-go`)

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/shark-auth/shark-go"
)

func main() {
    client := sharkauth.NewAdmin(
        "https://auth.myapp.com",
        os.Getenv("SHARKAUTH_API_KEY"),
    )

    ctx := context.Background()

    // Users
    users, err := client.Users.List(ctx, &sharkauth.ListUsersOpts{
        Search: "jane",
        Limit:  10,
    })

    user, err := client.Users.Get(ctx, "usr_abc123")
    err = client.Users.Delete(ctx, "usr_abc123")

    // RBAC
    allowed, err := client.Auth.Check(ctx, &sharkauth.CheckOpts{
        UserID:   "usr_abc123",
        Action:   "write",
        Resource: "documents",
    })

    // Roles
    role, err := client.Roles.Create(ctx, &sharkauth.CreateRoleOpts{
        Name:        "editor",
        Description: "Can edit documents",
    })
    err = client.Users.AssignRole(ctx, "usr_abc123", role.ID)

    // Audit
    logs, err := client.Audit.List(ctx, &sharkauth.AuditQuery{
        Action: "login",
        Limit:  50,
    })
}
```

### Go Middleware

```go
package main

import (
    "net/http"
    "github.com/shark-auth/shark-go/middleware"
)

func main() {
    shark := middleware.New("https://auth.myapp.com", os.Getenv("SHARKAUTH_API_KEY"))

    mux := http.NewServeMux()
    mux.Handle("/dashboard", shark.RequireAuth(dashboardHandler))
    mux.Handle("/api/", shark.RequireAPIKey(apiHandler))

    http.ListenAndServe(":3000", mux)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUser(r.Context())
    fmt.Fprintf(w, "Hello, %s", user.Name)
}
```

---

## Mobile Strategy

### The Cookie Problem on Mobile

Mobile platforms (React Native, Flutter, Swift, Kotlin) don't have automatic cookie jars like browsers. Two approaches:

### Option A: Token-Based Auth Endpoint (Recommended)

Add a new endpoint to SharkAuth:

```
POST /api/v1/auth/token
```

Returns a bearer token instead of setting a cookie. The mobile SDK stores it in the platform's secure storage (Keychain, KeyStore, SecureStore).

```typescript
// React Native
import { SharkAuth } from '@sharkauth/react-native'

const shark = new SharkAuth('https://auth.myapp.com')

const result = await shark.signIn({ email, password })
// Token stored in SecureStore automatically

const user = await shark.getUser()
// Sends Authorization: Bearer <token> instead of cookie
```

This requires a small backend change: accept `Authorization: Bearer <session_token>` as an alternative to the `shark_session` cookie. The session token would be the raw session ID (not the encrypted cookie value).

### Option B: Cookie Persistence Library

Use `@react-native-cookies/cookies` to persist cookies. Less clean, more fragile, but requires no backend changes.

### Recommendation

Option A. Add the `/auth/token` endpoint. It's a 20-line handler that creates a session and returns the session ID as a bearer token instead of setting a cookie. The existing session middleware already validates session IDs â€” it just needs to also check the Authorization header.

---

## Passkey/WebAuthn Handling

### Why a Separate Package

WebAuthn requires `@simplewebauthn/browser` (~8KB) for reliable cross-browser support. Not every app needs passkeys. Making it optional keeps the core bundle small.

### The Ceremony Abstraction

Raw WebAuthn flow:
1. Call server to generate challenge options
2. Call `navigator.credentials.create()` or `.get()` with those options
3. Serialize the credential response
4. Send to server for verification
5. Handle challenge key management (X-Challenge-Key header)

SDK flow:
```typescript
await shark.passkey.register()  // one call, all 5 steps handled
```

### Implementation

```typescript
// @sharkauth/passkey/src/index.ts
import { startRegistration, startAuthentication } from '@simplewebauthn/browser'

export function enablePasskeys(shark: SharkAuth) {
  shark.passkey = {
    async register(opts?: { name?: string }) {
      // 1. Get challenge from server
      const { publicKey, challengeKey } = await shark.fetch('/auth/passkey/register/begin', {
        method: 'POST',
      })
      
      // 2. Run WebAuthn ceremony (browser prompt)
      const credential = await startRegistration({ optionsJSON: publicKey })
      
      // 3. Send to server
      return shark.fetch('/auth/passkey/register/finish', {
        method: 'POST',
        headers: { 'X-Challenge-Key': challengeKey },
        body: { ...credential, name: opts?.name },
      })
    },
    
    async signIn(opts?: { email?: string }) {
      const { publicKey, challengeKey } = await shark.fetch('/auth/passkey/login/begin', {
        method: 'POST',
        body: opts?.email ? { email: opts.email } : {},
      })
      
      const assertion = await startAuthentication({ optionsJSON: publicKey })
      
      return shark.fetch('/auth/passkey/login/finish', {
        method: 'POST',
        headers: { 'X-Challenge-Key': challengeKey },
        body: assertion,
      })
    },
    
    // list, delete, rename are simple fetch calls
  }
  
  return shark
}
```

---

## Session Architecture & Cross-Tab Sync

### Why No Polling

Clerk polls every 5 seconds. Supabase polls every 30 seconds. SharkAuth doesn't need to poll because:

1. **Sessions are server-owned.** The cookie is opaque. The server decides if it's valid.
2. **No token expiry on the client.** The cookie has a `Max-Age` but the server is the authority.
3. **No refresh tokens.** One cookie, one session.

### Cross-Tab Sync

When a user logs in/out in one tab, other tabs should update immediately.

```typescript
class AuthState {
  private channel = new BroadcastChannel('sharkauth')
  private listeners: Set<(user: User | null) => void> = new Set()
  
  constructor() {
    this.channel.onmessage = (e) => {
      if (e.data.type === 'auth:change') {
        this.user = e.data.user
        this.notify()
      }
    }
    
    // Refresh on tab focus (user may have logged in/out in another tab)
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') {
        this.refresh()
      }
    })
  }
  
  private broadcast(user: User | null) {
    this.channel.postMessage({ type: 'auth:change', user })
  }
  
  async refresh() {
    const user = await this.fetchUser()
    if (this.hasChanged(user)) {
      this.user = user
      this.broadcast(user)
      this.notify()
    }
  }
}
```

### Concurrent Request Protection

Use `navigator.locks` to prevent multiple tabs from making concurrent session requests:

```typescript
async refresh() {
  if (navigator.locks) {
    await navigator.locks.request('sharkauth:refresh', async () => {
      await this.doRefresh()
    })
  } else {
    await this.doRefresh()
  }
}
```

---

## SSR Cookie Forwarding

### The Problem

Server components (Next.js, SvelteKit) run on the server and don't have access to browser cookies. When they need to call SharkAuth's API, they must forward the cookie from the incoming request.

### The Solution

Each framework SDK provides a server-side helper that reads cookies from the request context and forwards them:

```typescript
// Next.js â€” @sharkauth/next/server
import { cookies } from 'next/headers'

export async function getUser(): Promise<User | null> {
  const cookieStore = await cookies()
  const sessionCookie = cookieStore.get('shark_session')
  if (!sessionCookie) return null
  
  const resp = await fetch(`${SHARK_URL}/api/v1/auth/me`, {
    headers: { cookie: `shark_session=${sessionCookie.value}` },
  })
  
  if (!resp.ok) return null
  return resp.json()
}
```

### Optimization: Middleware Caching

To avoid calling SharkAuth on every server component render, the middleware can cache the auth state in request headers:

```typescript
// middleware.ts
export default async function middleware(req) {
  const session = req.cookies.get('shark_session')
  if (!session) return next(req)
  
  const resp = await fetch(`${SHARK_URL}/api/v1/auth/me`, {
    headers: { cookie: `shark_session=${session.value}` },
  })
  
  if (resp.ok) {
    const user = await resp.json()
    // Store in request header for server components
    req.headers.set('x-shark-user', JSON.stringify(user))
  }
  
  return next(req)
}

// Server component helper reads from header (no network call)
export async function getUser(): Promise<User | null> {
  const headerStore = await headers()
  const cached = headerStore.get('x-shark-user')
  return cached ? JSON.parse(cached) : null
}
```

---

## Error Handling

### Typed Errors

```typescript
// @sharkauth/core
export class SharkAuthError extends Error {
  code: string          // e.g., 'email_taken', 'weak_password', 'account_locked'
  status: number        // HTTP status
  message: string       // Human-readable

  // Convenience checks
  get isValidation() { return this.status === 400 }
  get isUnauthorized() { return this.status === 401 }
  get isForbidden() { return this.status === 403 }
  get isNotFound() { return this.status === 404 }
  get isConflict() { return this.status === 409 }
  get isRateLimited() { return this.status === 429 }
}

// Error codes enum
export const SharkErrorCode = {
  INVALID_REQUEST: 'invalid_request',
  INVALID_EMAIL: 'invalid_email',
  INVALID_TOKEN: 'invalid_token',
  WEAK_PASSWORD: 'weak_password',
  EMAIL_TAKEN: 'email_taken',
  MFA_REQUIRED: 'mfa_required',
  EMAIL_VERIFICATION_REQUIRED: 'email_verification_required',
  UNAUTHORIZED: 'unauthorized',
  NOT_FOUND: 'not_found',
  ACCOUNT_LOCKED: 'account_locked',
  INTERNAL_ERROR: 'internal_error',
} as const
```

### Usage

```typescript
try {
  await shark.signUp({ email, password })
} catch (e) {
  if (e instanceof SharkAuthError) {
    switch (e.code) {
      case 'email_taken':
        showError('An account with this email already exists')
        break
      case 'weak_password':
        showError(e.message) // server provides specific reason
        break
      default:
        showError('Something went wrong')
    }
  }
}

// Or in React with useSignUp:
const { error } = useSignUp()
// error is already typed as SharkAuthError | null
```

---

## Bundle Size Budget

| Package | Budget | Competitor Comparison |
|---------|--------|----------------------|
| `@sharkauth/core` | <1KB | -- |
| `@sharkauth/browser` | <5KB | Supabase auth-js: ~15KB, Clerk: ~100KB+ |
| `@sharkauth/react` | <3KB | -- |
| `@sharkauth/next` | <4KB | -- |
| `@sharkauth/svelte` | <3KB | -- |
| `@sharkauth/passkey` | <10KB | Includes @simplewebauthn/browser |
| **Total (React + passkeys)** | **<18KB** | **Clerk: 100KB+, Firebase: 80KB+, SuperTokens: 435KB** |

### Why We're So Small

1. **Cookie-based = no token management.** Clerk and Supabase spend most of their bundle on JWT parsing, token storage, refresh logic, and polling. We have none of that.
2. **No bundled UI.** Clerk's huge bundle includes pre-built sign-in/sign-up modals. We provide hooks, you build UI.
3. **Native APIs only.** `fetch`, `BroadcastChannel`, `navigator.locks`, `navigator.credentials` -- all built into browsers. Zero polyfills.
4. **Passkeys are optional.** The ~8KB WebAuthn library only loads if you use passkeys.

---

## Implementation Roadmap

### Phase 1: Config Simplification (2-3 days)
- Add `url`, `app_url`, `app_name` top-level keys
- Implement derivation rules (passkey RP from URL, CORS from app_url, etc.)
- Add email provider shortcuts (resend, postmark, sendgrid)
- Backward compatibility with existing config format
- Update docs and example configs

### Phase 2: `@sharkauth/browser` (3-4 days)
- Core fetch wrapper with typed methods
- `SharkAuth` class with all auth flows
- `AuthState` with BroadcastChannel sync and focus refresh
- `SharkAuthError` typed errors
- Full TypeScript types from API analysis
- Zero dependencies

### Phase 3: `@sharkauth/react` (2 days)
- `SharkAuthProvider` context
- `useAuth`, `useSignIn`, `useSignUp` hooks
- `OAuthButton` component
- `ProtectedRoute`, `PublicOnlyRoute` components

### Phase 4: `@sharkauth/node` (1-2 days)
- `SharkAdmin` class with namespaced resources
- All admin API methods typed
- Zero dependencies (Node 18+ native fetch)

### Phase 5: `@sharkauth/next` (2 days)
- `sharkMiddleware` with public/protected route config
- `getUser()` server helper
- `withAuth()` route handler wrapper
- SSR cookie forwarding

### Phase 6: `@sharkauth/passkey` (1 day)
- `enablePasskeys()` extension function
- Registration and login ceremony wrappers
- Credential management methods

### Phase 7: `@sharkauth/svelte` (1-2 days)
- Svelte stores
- SvelteKit hooks and server helpers

### Phase 8: Mobile + Backend SDKs (1 week)
- `/auth/token` endpoint on backend
- `@sharkauth/react-native`
- `sharkauth-py` (with Django/FastAPI helpers)
- `sharkauth-go` (with middleware)

### Total: ~3-4 weeks for complete SDK ecosystem

---

## Marketing Positioning

### One-Liner
> "Auth that respects your stack. 3-line config. 5KB SDK. Free forever."

### Key Claims (all verifiable)
- **3-line config** to production-ready auth (vs Auth0's weeks)
- **5KB client SDK** (vs Clerk's 100KB+, Firebase's 80KB+)
- **Zero dependencies** in core SDK (vs Better Auth's dep tree)
- **Every framework** -- React, Next.js, Svelte, Vue, plus Node, Python, Go admin SDKs (vs Stack Auth's Next.js-only)
- **Server-side sessions** -- no JWT headaches, no random logouts, instant revocation (vs Supabase's JWT problems)
- **Free forever** self-hosted with every feature (vs Clerk's feature gating)
- **No per-user pricing** -- Cloud charges for ops, not headcount (vs Clerk/Auth0 pricing shock)
- **Self-hostable** -- your data, your server, your rules (vs Clerk's managed-only)

### Tagline Options
- "Auth in 3 lines. Not 3 weeks."
- "The auth SDK that gets out of your way."
- "Self-hosted auth that doesn't suck."
- "Your users. Your server. Our SDK."

---

*This document should be treated as the SDK design contract. Implementation should follow these patterns exactly unless a concrete technical reason forces a deviation.*
