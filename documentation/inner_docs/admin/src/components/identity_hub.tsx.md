# identity_hub.tsx

**Path:** `admin/src/components/identity_hub.tsx`
**Type:** React component (page)
**Route:** `/admin/auth`
**Nav:** IDENTITY group → "Identity" (Lock icon)

## Purpose
Unified identity configuration surface — all authentication methods, session/token settings, active session management, MFA, SSO connections, and OAuth server settings in one place.

## Exports
- `IdentityHub()` (named + default export) — function component

## Sections

### 1. Authentication Methods
- **Password** — `password_min_length`, `password_reset_ttl`
- **Magic Link** — enable toggle, `magic_link.ttl`
- **Passkeys (WebAuthn)** — `rp_name`, `rp_id`, `user_verification` (required/preferred/discouraged)
- **Social providers** — Google, GitHub, Apple, Discord rows; click row → drawer with `client_id`, `client_secret`, callback URL + copy button

### 2. Sessions & Tokens
- Cookie session lifetime (`auth.session_lifetime`) — always on, no mode toggle
- JWT access token lifetime, issuer, audience — always on, no mode toggle
- Signing keys table — pulled from `/.well-known/jwks.json`; KID, algorithm, use, active marker; Inspect button → drawer showing full key fields

### 3. Active Sessions
- Fetches `GET /api/v1/admin/sessions?limit=50`
- Columns: user_email, auth_method, MFA chip, created (relative time)
- Per-row **Revoke** (`DELETE /api/v1/admin/sessions/{id}`)
- **Revoke all** (`DELETE /api/v1/admin/sessions`) — with confirm step
- **Purge expired** (`POST /api/v1/admin/sessions/purge-expired`)

### 4. MFA
- Enforcement select: off / optional / required
- TOTP issuer name
- Recovery code count

### 5. SSO Connections
- Fetches `GET /api/v1/sso/connections`
- Per-row Inspect drawer (shows all fields + copy) + Delete (`DELETE /api/v1/sso/connections/{id}`)
- Refresh button; empty state with API hint for creating connections

### 6. OAuth Server
- Enabled toggle, issuer URL, access/refresh token lifetimes, DPoP requirement
- Device code approval queue — `GET /api/v1/admin/oauth/device-codes`; per-row Approve (`POST /api/v1/admin/oauth/device-codes/{user_code}/approve`) / Deny (`…/deny`)

## Config I/O
- **Read:** `GET /api/v1/admin/config` via `useAPI('/admin/config')`
- **Write:** `PUT /api/v1/admin/config` — payload mirrors server config shape for auth, passkeys, magic_link, jwt, social, mfa, oauth_server

## Visual Contract (.impeccable.md v3)
- Strict monochrome: `var(--fg)`, `var(--fg-dim)`, `var(--surface-0/1/2)`, `var(--hairline)`, `var(--danger)` only
- Square corners: 4-5px cards/inputs, 3px chips/badges
- Compact rows: ~36px height, 7px padding
- Drawers: `position:fixed; top:0; right:0; bottom:0; width:420px; border-left:1px solid var(--hairline)`
- Sticky save bar appears only when dirty; includes Discard + Save buttons
- NO session-vs-JWT mode toggle anywhere — both always on

## Composed by
- `App.tsx` (route key `auth`)
- `layout.tsx` NAV — IDENTITY group
