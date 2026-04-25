# App.tsx

**Path:** `admin/src/hosted/App.tsx`
**Type:** React component (page shell)
**LOC:** ~300

## Purpose
Root hosted authentication app—routes to login, signup, MFA, magic link, passkey, password reset, verify email, forgot password, error pages.

## Exports
- `HostedApp({ config })` (default) — function component

## Props
- `config: HostedConfig` — branding, auth methods, OAuth config

## Routes
- `/` or `/login` → Login
- `/signup` → Sign up
- `/mfa` → MFA challenge
- `/magic` → Magic link verification
- `/passkey` → Passkey authentication
- `/reset-password` → Password reset form
- `/verify` → Email verification
- `/forgot-password` → Forgot password flow
- `/error` → Error page

## Features
- **Branding** — applies custom colors, fonts, logo
- **Auth method filtering** — shows only enabled methods
- **OAuth providers** — renders provider buttons
- **Branded layout** — logo, company branding on all pages
- **Error handling** — displays errors on all flows

## Layout
- Centered container with logo/branding
- Form-based authentication flows
- Support for multiple auth methods
- Responsive design

## Composed by
- hosted-entry.tsx

## Notes
- Uses design system for consistency
- All auth flows leverage same branded template
- OAuth redirect handled via OAuth config
