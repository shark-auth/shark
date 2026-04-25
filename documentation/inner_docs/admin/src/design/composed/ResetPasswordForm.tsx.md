# ResetPasswordForm.tsx

**Path:** `admin/src/design/composed/ResetPasswordForm.tsx`
**Type:** React component (composed surface)
**LOC:** 220

## Purpose
"Set new password" form — collects new password + confirmation, validates length and equality, submits via the supplied async handler, and renders a success banner with a primary "Sign in" CTA.

## Exports
- `ResetPasswordForm` — `{appName, onSubmit, signInHref?}`.
- `ResetPasswordFormProps` (interface).

## Props / hooks
- `appName` — page heading.
- `onSubmit(password)` — async submit (caller supplies the token from URL).
- `signInHref?` — destination after success.
- State: `password`, `confirm`, `passwordError`, `confirmError`, `formError`, `loading`, `success`.

## API calls
- None — caller wires `onSubmit` to `/api/v1/auth/password/reset` with `{token, password}`.

## Composed by
- `admin/src/hosted/routes/reset_password.tsx` (`ResetPasswordPage`).

## Notes
- Validation: required, ≥8 chars, must equal confirm field.
- Success state replaces the form with a green banner + Continue button (`window.location.href`).
- Built on `Button`, `FormField`, `Card`, design `tokens`.
- File continues past line 150 with the form JSX (omitted from inspection — same FormField pattern as ForgotPasswordForm).
