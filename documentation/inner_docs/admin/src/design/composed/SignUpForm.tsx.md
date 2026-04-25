# SignUpForm.tsx

**Path:** `admin/src/design/composed/SignUpForm.tsx`
**Type:** React component (composed surface)
**LOC:** 278

## Purpose
Hosted sign-up form. Configurable fields (`requirePassword`, `requireName`) collect the minimum payload then defer to `onSubmit`. Implements field-level validation, focus-on-first-error, optional terms link, and a sign-in footer link.

## Exports
- `SignUpForm` — see props.
- `SignUpFormProps` (interface).

## Props / hooks
- `appName: string` — heading.
- `requirePassword: boolean` — gates password + confirm fields.
- `requireName?: boolean` (default false).
- `onSubmit({email, password?, name?})` — async submit.
- `signInHref?`, `termsHref?`.
- State: `name`, `email`, `password`, `confirm`, per-field errors, `formError`, `loading`.

## API calls
- None — caller wires `onSubmit` to `/api/v1/auth/signup`.

## Composed by
- `admin/src/hosted/routes/signup.tsx` (`SignupPage`).

## Notes
- `validate()` collects errors then focuses the first invalid input via `document.getElementById(firstId)`.
- Password rules: required, ≥8 chars, must match confirm.
- Submit payload conditionally includes `password` / `name` depending on prop flags.
- Built on `Button`, `FormField`, `Card`, design `tokens`.
