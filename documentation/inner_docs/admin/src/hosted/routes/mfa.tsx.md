# mfa.tsx

**Path:** `admin/src/hosted/routes/mfa.tsx`
**Type:** React route shell
**LOC:** 102

## Purpose
MFA challenge page. Reads `?method=` and `?session=` from the URL, wires the design `MFAForm` to the challenge endpoint, supports resend for sms/email methods, and exposes a "use backup code" navigation for TOTP.

## Exports
- `MFAPage` — `{config: HostedConfig}`.
- Internal helpers: `parseMFAMethod`, `VALID_METHODS`.

## Props / hooks
- `useLocation` for in-app navigation (backup-code link).
- `useToast()`.
- `useState` initialized once for `method` (defaulting to `totp`) and `session` token.

## API calls
- POST `/api/v1/auth/mfa/challenge` `{code [, session]}` for submit.
- POST `/api/v1/auth/mfa/challenge` `{resend: true [, session]}` for resend.

## Composed by
- `admin/src/hosted/App.tsx` route table at `/mfa`.

## Notes
- On success, builds the OAuth callback URL with `state` and `window.location.href` redirects.
- `supportsResend = method === 'sms' || method === 'email'` — TOTP / WebAuthn omit the resend button.
- `onUseBackup` only wired for TOTP; other methods leave it undefined so the design surface hides the link.
- Errors are toasted AND re-thrown to surface in the form.
