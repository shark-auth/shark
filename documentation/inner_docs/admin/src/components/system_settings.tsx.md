# system_settings.tsx

**Path:** `admin/src/components/system_settings.tsx`
**Type:** React component (page)
**LOC:** ~400

## Purpose
System-wide configuration—email/SMTP settings, password policies, session timeouts, rate limiting, feature toggles.

## Exports
- `SystemSettings()` (default) — function component

## Features
- **SMTP config** — host, port, auth, from address, test email
- **Password policy** — min length, complexity requirements, expiration
- **Session settings** — timeout, absolute max lifetime, refresh token rotation
- **Rate limiting** — login attempts, API rate limits, webhook retries
- **Feature toggles** — enable/disable auth methods, MFA requirement, passwordless
- **OAuth defaults** — default scopes, access token lifetime

## Hooks used
- `useAPI('/admin/system-config')` — fetch settings
- `useToast()` — save feedback

## State
- `tab` — current settings section
- `settings` — form state
- `saving` — submit state
- `testingSMTP` — SMTP test in progress

## API calls
- `GET /api/v1/admin/system-config` — fetch all settings
- `PATCH /api/v1/admin/system-config` — update settings
- `POST /api/v1/admin/system-config/test-email` — test SMTP

## Composed by
- App.tsx

## Notes
- Changes affect all users system-wide
- SMTP required for email-based auth (password reset, email verification)
- Rate limiting prevents brute force attacks
- Feature toggles allow gradual rollout of new auth methods
