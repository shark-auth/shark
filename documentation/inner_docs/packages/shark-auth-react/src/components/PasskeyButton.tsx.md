# PasskeyButton.tsx

**Path:** `packages/shark-auth-react/src/components/PasskeyButton.tsx`
**Type:** React component — WebAuthn passkey sign-in
**LOC:** 132

## Purpose
End-to-end WebAuthn assertion flow: fetches a challenge from the server, calls `navigator.credentials.get`, and posts the assertion back. No password / OAuth redirect required.

## Public API
- `interface PasskeyButtonProps { onSuccess?, onError?, children?, className? }`
- `function PasskeyButton(props): JSX.Element`

### Props
- `onSuccess` — fired after `/auth/passkey/login/finish` returns 2xx.
- `onError` — fired with an `Error` for any failure (network, missing credential, server reject).
- `children` — button label; defaults to `'Sign in with passkey'`.
- `className` — passed to the button.

## How it composes
1. Reads `AuthContext` for the bound `client`. Errors `'Not inside SharkProvider'` if absent.
2. **POST** `/auth/passkey/login/begin` (no body) → JSON `{ challenge: base64url, allowCredentials?: [{id, type}], userVerification?, timeout? }`.
3. Decodes `challenge` and each `allowCredentials[i].id` from base64url to `ArrayBuffer`.
4. Calls `navigator.credentials.get({ publicKey: { challenge, allowCredentials, userVerification: 'preferred' } })`.
5. Re-encodes `rawId`, `authenticatorData`, `clientDataJSON`, `signature`, `userHandle` to base64url.
6. **POST** `/auth/passkey/login/finish` with JSON `{ id, rawId, type, response: { authenticatorData, clientDataJSON, signature, userHandle } }`.
7. Non-2xx at either step throws `Passkey {begin|finish} failed <status>[: <text>]`.

## Internal dependencies
- `hooks/context.AuthContext`

## Used by (consumer-facing)
- Drop-in inside `<SignedOut>` next to (or instead of) `<SignIn>`.

## Notes
- Inline-styled dark button (`#111827`); error appears beneath in red.
- `userVerification: 'preferred'` lets the browser decide whether to require a biometric/PIN.
- Like `MFAChallenge`, it doesn't proactively re-hydrate `SharkProvider` on success — callers should reload or trigger a context refresh in `onSuccess`.
