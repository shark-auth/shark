# EmailVerify.tsx

**Path:** `admin/src/design/composed/EmailVerify.tsx`
**Type:** React component (composed surface)
**LOC:** 224

## Purpose
Token-based email verification status surface. Renders one of three states (pending / success / error) with an icon, title, description, and either a Resend or Continue CTA.

## Exports
- `EmailVerify` — `{state, errorMessage?, onResend?, onContinue?}`.
- `EmailVerifyProps` (interface).

## Props / hooks
- `state: 'pending' | 'success' | 'error'` — picks icon + title + default description.
- `errorMessage?` — overrides the default copy in the error state.
- `onResend?: () => Promise<void>` — wires the Resend button (also shown in pending).
- `onContinue?: () => void` — shown only on success.
- Internal state: `resendLoading`, `resendError`, `resendSuccess`.

## API calls
- None directly — caller wires `onResend` to the appropriate `/api/v1/auth/email/verify/send` endpoint.

## Composed by
- `admin/src/hosted/routes/verify.tsx` (`VerifyPage`).

## Notes
- Visual constants: pending uses `--shark-primary`, success `oklch(68% 0.15 160)`, error `oklch(62% 0.2 25)`.
- Built on the `Card` and `Button` primitives + design `tokens`.
- Icons are inline SVGs (`CheckIcon`, `ClockIcon`, `AlertIcon`).
