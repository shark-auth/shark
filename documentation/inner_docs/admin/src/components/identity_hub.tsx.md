# identity_hub.tsx

**Path:** `admin/src/components/identity_hub.tsx`
**Type:** React component (page)
**LOC:** ~500

## Purpose
Identity hub—central configuration for authentication methods, MFA policies, passwordless auth, DPoP, risk assessment rules.

## Exports
- `IdentityHub()` (default) — function component

## Features
- **Auth methods** — enable/disable password, OAuth, passkey, magic link
- **MFA settings** — TOTP, SMS, backup codes, enforcement policies (optional|required|risk-based)
- **Passwordless** — passkey FIDO2 configuration, magic link settings
- **DPoP** — Demonstrating Proof-of-Possession settings for API security
- **Risk assessment** — rules for flagging suspicious logins (geo, time-of-day, device)
- **Session binding** — device fingerprinting, IP lock

## Hooks used
- `useAPI('/admin/auth-config')` — fetch config
- `useToast()` — save feedback

## State
- `config` — current settings
- `saving` — submit state
- `changes` — unsaved changes

## API calls
- `GET /api/v1/admin/auth-config` — fetch settings
- `PATCH /api/v1/admin/auth-config` — update

## Composed by
- App.tsx

## Notes
- Controls the entire authentication posture
- Risk-based MFA triggers MFA only on suspicious logins
- DPoP prevents token theft for API clients
- Passkey FIDO2 most secure, requires device enrollment
