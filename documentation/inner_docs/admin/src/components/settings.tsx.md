# settings.tsx

**Path:** `admin/src/components/settings.tsx`
**Type:** React component (page)
**Last rebuild:** 2026-04-25 (strict monochrome/square/editable spec)
**Last audit:** 2026-04-25 (gap-fill: auth policy, webhooks, social redirect, audit CSV export)

## Purpose
Global server settings: server URL/port/CORS, sessions+JWT (unified), auth policy, email delivery, OAuth providers (links + redirect URL), webhook subscriptions, audit retention + CSV export, maintenance actions. Every field editable via PATCH `/admin/config` or webhook CRUD endpoints.

## Exports
- `Settings()` — default + named export
- `system_settings.tsx` is a re-export shim (`Settings as SystemSettings`)

## Sections
- **Server** — base_url, port, cors_origins (add/remove tag list). Tiny derived status strip above sections: version + uptime + DB driver + migration version (NOT a section).
- **Sessions & Tokens** — ONE section. Cookie session_lifetime, JWT lifetime, JWT signing key (active fingerprint + algorithm + Rotate button). Cookie + JWT BOTH always on.
- **Auth Policy** — password_min_length, magic_link TTL, password_reset TTL. All PATCH /admin/config fields.
- **Email Delivery** — provider select (shark/resend/smtp/dev), masked api_key (Show/Hide), from address, from name, Test email drawer
- **OAuth Providers** — social.redirect_url editable field + quick links to Identity Hub
- **Webhooks** — inline list (url, enabled dot, event count); New webhook drawer (URL, events multi-select from /admin/webhooks/events); Edit webhook drawer (enable/disable, URL, events, Send test, Delete)
- **Audit & Data** — retention days, cleanup interval, Purge audit logs (drawer w/ PURGE-confirm), Export CSV (date-range drawer → downloads CSV)
- **Maintenance** — Purge expired sessions (drawer w/ confirm)

## Editable fields (~20)
server.{base_url, port, cors_origins}, auth.{session_lifetime, password_min_length}, jwt.lifetime, magic_link.ttl, password_reset.ttl, social.redirect_url, email.{provider, api_key, from, from_name}, audit.{retention, cleanup_interval}. Webhooks via separate CRUD endpoints.

## API calls
- `GET /admin/config`, `PATCH /admin/config`
- `GET /admin/health`
- `GET /admin/webhooks`, `POST /admin/webhooks`, `PATCH /admin/webhooks/{id}`, `DELETE /admin/webhooks/{id}`
- `GET /admin/webhooks/events`
- `POST /admin/webhooks/{id}/test`
- `POST /admin/sessions/purge-expired`
- `POST /admin/audit-logs/purge`
- `POST /admin/audit-logs/export` (streams CSV)
- `POST /admin/test-email`
- `POST /admin/auth/rotate-signing-key`

## Composed by
- `App.tsx` — route `settings: Settings`

## Visual contract (per .impeccable.md v3)
- Monochrome — status dots only carry color
- Radii: 5px outer, 4px inputs, 3px chips
- 13px base, 11px uppercase labels, hairline borders, 7-10px row padding, 28px input height
- Drawers (right-side fixed) for Test email / Purge confirms / Webhook create+edit — never modal
- Matches users.tsx rhythm

## Notes
- `server.*` PATCH fields are forward-compatible (backend handler currently ignores; future-proof shape)
- No mode toggle, no issuer/audience/clock-skew/signing-alg knobs in UI (low-level per .impeccable.md)
- No YAML import/export anywhere (deprecated path)
- Webhook HMAC secret shown exactly once after create — subsequent reads omit it (backend behavior)
- Branding (GET/PATCH /admin/branding) intentionally excluded — belongs on a dedicated Branding surface, not Settings
- `system_settings.tsx` retained as shim for any external import compat
