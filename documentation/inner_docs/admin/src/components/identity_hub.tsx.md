# identity_hub.tsx

**Path:** `admin/src/components/identity_hub.tsx`
**Type:** React component (page)
**Route:** `/admin/auth`
**Nav:** IDENTITY group → "Identity" (Lock icon)
**Last rebuild:** 2026-04-26 (Wave 1.7 Edit 3 — stripped to auth-methods-only)

## Purpose

WHO can authenticate. Authentication method configuration only.

**Identity Hub is NOT the source of truth for system operation config.** Those surfaces
are in Settings (source of truth):
- Sessions & Tokens lifetimes → Settings → Sessions & Tokens
- JWT signing key rotation → Settings → Sessions & Tokens
- OAuth Authorization Server (issuer, DPoP, lifetimes) → Settings → OAuth & SSO
- SSO Connections → Settings → OAuth & SSO
- MFA enforcement policy → Settings → OAuth & SSO
- Device code queue → Settings → OAuth & SSO
- Active session list + purge → Settings → Maintenance

A cross-link banner inside the component directs users to Settings for those surfaces.

## Exports
- `IdentityHub()` (named + default export) — function component

## Sections (after Wave 1.7 Edit 3 cleanup)

### 1. Authentication Methods (sole section — source of truth here)
- **Password** — `password_min_length`, `password_reset_ttl`
- **Magic Link** — enable toggle, `magic_link.ttl`
- **Passkeys (WebAuthn)** — `rp_name`, `rp_id`, `user_verification` (required/preferred/discouraged)
- **Social providers** — Google, GitHub, Apple, Discord rows; click → drawer with `client_id`, `client_secret`, callback URL + copy button

### Cross-link banner (bottom)
Text: "Sessions, Tokens, OAuth Server, SSO & MFA" with "Open Settings →" button.
Directs users to Settings for all moved sections.

## Sections removed in Wave 1.7 Edit 3 (moved to Settings → OAuth & SSO)

| Section | New location |
|---|---|
| Sessions & Tokens config | Settings → Sessions & Tokens |
| Active Sessions list | Settings → Maintenance |
| OAuth Server config | Settings → OAuth & SSO |
| MFA enforcement | Settings → OAuth & SSO |
| SSO Connections | Settings → OAuth & SSO |
| Device code approval queue | Settings → OAuth & SSO |

## Config I/O
- **Read:** `GET /api/v1/admin/config` via `useAPI('/admin/config')`
- **Write:** `PUT /api/v1/admin/config` — payload contains only `auth`, `passkeys`, `magic_link`, `social`
  (oauth_server, mfa, jwt, session fields are NOT included — owned by Settings)

## Visual Contract (.impeccable.md v3)
- Strict monochrome: `var(--fg)`, `var(--fg-dim)`, `var(--surface-0/1/2)`, `var(--hairline)`, `var(--danger)` only
- Square corners: 4-5px cards/inputs, 3px chips/badges
- Compact rows: ~36px height, 7px padding
- Drawers: `position:fixed; top:0; right:0; bottom:0; width:420px; border-left:1px solid var(--hairline)`
- Sticky save bar appears only when dirty; includes Discard + Save buttons
- NO session-vs-JWT mode toggle — both always on (not exposed here)

## Composed by
- `App.tsx` (route key `auth`)
- `layout.tsx` NAV — IDENTITY group
