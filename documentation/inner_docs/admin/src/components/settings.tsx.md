# settings.tsx

**Path:** `admin/src/components/settings.tsx`
**Type:** React component (page)
**Last rebuild:** 2026-04-25 (strict monochrome/square/editable spec)
**Last audit:** 2026-04-26 (Wave 1.7 Edit 3 — added OAuth & SSO section, received moved sections from Identity Hub)

## Purpose

HOW the system operates. Source of truth for all system-level configuration: server URL/port/CORS,
sessions + JWT (unified), auth policy, OAuth authorization server, SSO connections, MFA enforcement,
email delivery, social provider links, audit retention, maintenance actions.

**Settings is the canonical home for every operational config.** Identity Hub owns only
WHO can authenticate (authentication methods). Everything else is here.

## Exports
- `Settings()` — default + named export
- `system_settings.tsx` is a re-export shim (`Settings as SystemSettings`)

## Sections (left-rail anchored navigation)

- **Server** — base_url, port, cors_origins (add/remove tag list). Derived status strip above sections: version + uptime + DB driver + migration version (NOT a section).
- **Sessions & Tokens** — ONE section. Cookie session_lifetime, JWT lifetime, JWT signing key (active fingerprint + algorithm + Rotate button). Cookie + JWT BOTH always on, no mode toggle.
- **Auth Policy** — password_min_length, magic_link TTL, password_reset TTL. All PATCH /admin/config fields.
- **OAuth & SSO** *(added Wave 1.7 Edit 3 — moved from Identity Hub)* — three sub-sections:
  - OAuth Authorization Server — enabled toggle, DPoP requirement, issuer URL, access token lifetime, refresh token lifetime, device code approval queue (approve/deny per code)
  - MFA Enforcement — enforcement level (off/optional/required), TOTP issuer name, recovery code count
  - SSO Connections — table of connections (name, type, domain, status); Inspect drawer shows all fields + Delete
- **Email Delivery** — provider select (shark/resend/smtp/dev), masked api_key (Show/Hide), from address, from name, Test email drawer
- **OAuth Providers** — social.redirect_url editable field + cross-link to Identity Hub for per-provider credentials
- **Audit & Data** — retention days, cleanup interval, Purge audit logs (drawer w/ PURGE-confirm), Export CSV (date-range drawer → downloads CSV)
- **Maintenance** — Purge expired sessions (drawer w/ confirm)

## Sections moved FROM Identity Hub in Wave 1.7 Edit 3

| Section | Old location | Now in Settings |
|---|---|---|
| OAuth Authorization Server | Identity Hub | Settings → OAuth & SSO |
| MFA enforcement | Identity Hub | Settings → OAuth & SSO |
| SSO Connections | Identity Hub | Settings → OAuth & SSO |
| Device code queue | Identity Hub | Settings → OAuth & SSO |

Active sessions list (revoke/purge) remains in Settings → Maintenance (was already there via drawer).

## Editable fields (~28 after Edit 3 additions)
server.{base_url, port, cors_origins, cors_relaxed}, auth.{session_lifetime, password_min_length}, jwt.lifetime, magic_link.ttl, password_reset.ttl, social.redirect_url, email.{provider, api_key, from, from_name}, audit.{retention, cleanup_interval}, oauth_server.{enabled, issuer, access_token_lifetime, refresh_token_lifetime, require_dpop}, mfa.{enforcement, issuer, recovery_codes}.

## API calls
- `GET /admin/config`, `PATCH /admin/config`
- `GET /admin/health`
- `GET /api/v1/sso/connections` — SSO list
- `DELETE /api/v1/sso/connections/{id}` — SSO delete
- `GET /api/v1/admin/oauth/device-codes` — device code queue
- `POST /api/v1/admin/oauth/device-codes/{user_code}/approve` — approve device code
- `POST /api/v1/admin/oauth/device-codes/{user_code}/deny` — deny device code
- `POST /admin/sessions/purge-expired`
- `POST /admin/audit-logs/purge`
- `POST /admin/audit-logs/export` (streams CSV)
- `POST /admin/test-email`
- `POST /admin/auth/rotate-signing-key`
- `POST /admin/system/swap-mode`
- `POST /admin/system/reset`

## Composed by
- `App.tsx` — route `settings: Settings`

## Visual contract (per .impeccable.md v3)
- Monochrome — status dots only carry color
- Radii: 5px outer, 4px inputs, 3px chips
- 13px base, 11px uppercase labels, hairline borders, 7-10px row padding, 28px input height
- Drawers (right-side fixed, 480px) for Test email / Purge confirms / SSO inspect / mode swap — never modal
- Matches users.tsx rhythm

## Notes
- `server.*` PATCH fields are forward-compatible (backend handler currently ignores; future-proof shape)
- No mode toggle, no JWT-vs-session mode anywhere (both always on)
- No YAML import/export anywhere (deprecated path)
- Branding (GET/PATCH /admin/branding) intentionally excluded — dedicated Branding surface (coming soon)
- `system_settings.tsx` retained as shim for any external import compat
- SSO connections are read-only inspect + delete only; creation is via Admin API (`POST /api/v1/admin/sso/connections`)
