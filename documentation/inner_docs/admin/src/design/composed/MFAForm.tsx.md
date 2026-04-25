# MFAForm.tsx

**Path:** `admin/src/design/composed/MFAForm.tsx`
**Type:** React component (composed surface)
**LOC:** 271

## Purpose
6-digit MFA challenge surface supporting four method modes (`totp` / `sms` / `email` / `webauthn`). Implements per-digit input boxes with auto-advance, paste-to-fill, backspace navigation, optional resend (sms/email), and an optional "use backup code" link.

## Exports
- `MFAForm` — `{method, onSubmit, onResend?, onUseBackup?}`.
- `MFAFormProps` (interface).

## Props / hooks
- `method` — drives description copy; `webauthn` skips the digit grid (UI hidden lower in file).
- `onSubmit(code: string)` — full 6-digit code.
- `onResend?` — typically wired only for sms/email.
- `onUseBackup?` — secondary navigation (e.g. → `/mfa?backup=1`).
- State: `digits` (string[6]), `error`, `loading`, `resendLoading`, `inputRefs` array.

## API calls
- None — caller wires to `/api/v1/auth/mfa/challenge`.

## Composed by
- `admin/src/hosted/routes/mfa.tsx` (`MFAPage`).

## Notes
- Paste handler accepts up to 6 digits; non-digits stripped before fill.
- Submit-on-incomplete: focuses the first empty box rather than refusing silently.
- Wrong-code branch resets all digits and re-focuses the first input.
- Built on `Card` + `Button` primitives, custom 44x52 monospace digit boxes.
