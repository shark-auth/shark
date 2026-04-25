# reset_password.tsx

**Path:** `admin/src/hosted/routes/reset_password.tsx`
**Type:** React route shell
**LOC:** 63

## Purpose
Token-bound "set new password" route. Reads the token from the URL, wires the `ResetPasswordForm` design surface to the password-reset endpoint, and renders an error fallback when the token is missing.

## Exports
- `ResetPasswordPage` — `{config: HostedConfig}`.

## Props / hooks
- `useToast()`.
- `useState` initialized once with `URLSearchParams.get('token')`.

## API calls
- POST `/api/v1/auth/password/reset` `{token, password}` (cookies included).

## Composed by
- `admin/src/hosted/App.tsx` route table at `/reset-password`.

## Notes
- Missing-token state renders a minimal full-screen explainer with a "Back to login" link, bypassing the form entirely.
- Errors toasted AND re-thrown so the form's inline banner picks them up.
- Success state lives in the design surface — this shell just calls `toast.success` after the API resolves.
