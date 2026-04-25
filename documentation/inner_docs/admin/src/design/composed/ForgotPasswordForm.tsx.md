# ForgotPasswordForm.tsx

**Path:** `admin/src/design/composed/ForgotPasswordForm.tsx`
**Type:** React component (composed surface)
**LOC:** 186

## Purpose
"Reset your password" entry form — collects email, calls `onSubmit`, and renders a generic success banner that does not leak account existence.

## Exports
- `ForgotPasswordForm` — `{appName, onSubmit, signInHref?}`.
- `ForgotPasswordFormProps` (interface).

## Props / hooks
- `appName: string` — heading.
- `onSubmit(email)` — async submit handler.
- `signInHref?: string` — back-to-sign-in footer link.
- State: `email`, `emailError`, `formError`, `loading`, `success`.

## API calls
- None — caller wires `onSubmit` to `/api/v1/auth/password/send-reset-link`.

## Composed by
- `admin/src/hosted/routes/forgot_password.tsx` (`ForgotPasswordPage`).

## Notes
- Email validation via simple regex; field-level error renders inside `FormField`, form-level error in a danger banner.
- Success banner uses non-leaky copy: "If an account exists for {email}, you will receive a password reset link shortly."
- Built on `Button`, `FormField`, `Card` primitives + design `tokens`.
