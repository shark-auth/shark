# MagicLinkSent.tsx

**Path:** `admin/src/design/composed/MagicLinkSent.tsx`
**Type:** React component (composed surface)
**LOC:** 187

## Purpose
"Check your email" confirmation screen shown after a magic-link request. Displays the recipient address, an inline mail icon, and an optional resend button gated by a countdown cooldown.

## Exports
- `MagicLinkSent` — `{email, onResend?, resendCooldownSeconds?}`.
- `MagicLinkSentProps` (interface).

## Props / hooks
- `email` — emphasized in body copy.
- `onResend?` — async handler; absent disables the resend button entirely.
- `resendCooldownSeconds?` (default 60) — initial + post-resend cooldown window.
- State: `cooldown`, `loading`, `error`, `success`. `useRef` holds the `setInterval` token.

## API calls
- None — caller wires `onResend` to e.g. `/api/v1/auth/magic-link/send`.

## Composed by
- `admin/src/hosted/routes/magic.tsx` (`MagicPage`).

## Notes
- `startCooldown` clears any existing timer before installing a new one and bottoms out at 0.
- Initial cooldown auto-starts on mount when `onResend` is provided and the prop is positive.
- Button label flips between `Resend in Ns` and `Resend email` based on `cooldown`.
