# PasskeyButton.tsx

**Path:** `admin/src/design/composed/PasskeyButton.tsx`
**Type:** React component
**LOC:** 67

## Purpose
"Continue with passkey" CTA — wraps the design-system `Button` (ghost / large) with a key icon and an internal loading state that disables the button while `onClick` is awaited.

## Exports
- `PasskeyButton` — `{onClick, label?, loading?, disabled?}`.
- `PasskeyButtonProps` (interface).
- Internal `KeyIcon` SVG.

## Props / hooks
- `onClick: () => Promise<void>` — typically launches the WebAuthn flow.
- `label?` — defaults to "Continue with passkey".
- `loading?`, `disabled?` — externally controlled flags merged with internal state.
- State: `internalLoading`.

## API calls
- None — caller wires `onClick` to `/api/v1/auth/passkey/login/begin` + finish.

## Composed by
- `SignInForm` (alternative auth section), `admin/src/hosted/routes/passkey.tsx` (dedicated passkey page).

## Notes
- Internal loading is `try/finally`-bracketed around the awaited handler so it always clears (unlike the OAuth button which uses a 3s timeout because of full-page redirects).
- KeyIcon hidden when loading so the spinner from `Button` isn't crowded.
