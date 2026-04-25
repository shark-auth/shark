# settings.tsx

**Path:** `admin/src/components/settings.tsx`
**Type:** React component (page)
**Last rebuild:** 2026-04-25 (strict monochrome/square/editable spec)

## Purpose
Global server settings: server URL/port/CORS, sessions+JWT (unified), email delivery, OAuth providers (links), audit retention, maintenance actions. Every field editable via PATCH `/admin/config`.

## Exports
- `Settings()` — default + named export
- `system_settings.tsx` is a re-export shim (`Settings as SystemSettings`)

## Sections
- **Server** — base_url, port, cors_origins (add/remove tag list). Tiny derived status strip above sections: version + uptime + DB driver + migration version (NOT a section).
- **Sessions & Tokens** — ONE section. Cookie session_lifetime, JWT lifetime, JWT signing key (active fingerprint + algorithm + Rotate button). Cookie + JWT BOTH always on.
- **Email Delivery** — provider select (shark/resend/smtp/dev), masked api_key (Show/Hide), from address, from name, Test email drawer
- **OAuth Providers** — quick links to Identity Hub
- **Audit & Data** — retention days, cleanup interval, Purge audit logs (drawer w/ PURGE-confirm)
- **Maintenance** — Purge expired sessions (drawer w/ confirm)

## Editable fields (~12)
server.{base_url, port, cors_origins}, auth.session_lifetime, jwt.lifetime, email.{provider, api_key, from, from_name}, audit.{retention, cleanup_interval}.

## API calls
- `GET /admin/config`, `PATCH /admin/config`
- `GET /admin/health`
- `POST /admin/sessions/purge-expired`
- `POST /admin/audit-logs/purge`
- `POST /admin/test-email`
- `POST /admin/auth/rotate-signing-key`

## Composed by
- `App.tsx` — route `settings: Settings`

## Visual contract (per .impeccable.md v3)
- Monochrome — status dots only carry color
- Radii: 5px outer, 4px inputs, 3px chips
- 13px base, 11px uppercase labels, hairline borders, 7-10px row padding, 28px input height
- Drawers (right-side fixed) for Test email / Purge confirms — never modal
- Matches users.tsx rhythm

## Notes
- `server.*` PATCH fields are forward-compatible (backend handler currently ignores; future-proof shape)
- No mode toggle, no issuer/audience/clock-skew/signing-alg knobs in UI
- No YAML import/export anywhere (deprecated path)
- `system_settings.tsx` retained as shim for any external import compat
