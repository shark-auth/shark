# MFAChallenge.tsx

**Path:** `packages/shark-auth-react/src/components/MFAChallenge.tsx`
**Type:** React component — TOTP / one-time-code submission form
**LOC:** 93

## Purpose
A small numeric input + submit button that posts an MFA code to the SharkAuth server, used as a step-up after a session is already half-authenticated.

## Public API
- `interface MFAChallengeProps { onSuccess?: () => void; onError?: (err: Error) => void }`
- `function MFAChallenge(props): JSX.Element`

### Props
- `onSuccess` — fired after the server returns 2xx; the local input clears.
- `onError` — fired with an `Error` whenever the request fails or returns non-2xx (also displayed inline in red).

## How it composes
1. Reads `AuthContext` directly (not via hook) to access the bound `client`. Errors with `'Not inside SharkProvider'` if missing.
2. Numeric-only input (sanitizes via `replace(/\D/g, '')`), max 8 chars, autocomplete `one-time-code`, inputmode `numeric`.
3. Submit button is disabled while loading or until the code is at least 6 chars.
4. On submit, **POST** `/auth/mfa/challenge` with JSON body `{ code }`.
5. Non-2xx → throws `MFA challenge failed <status>: <text>`.

## Internal dependencies
- `hooks/context.AuthContext`

## Used by (consumer-facing)
- Step-up flow: render in a modal or dedicated route after the server signals MFA is required.

## Notes
- Inline `React.CSSProperties` styling — no class hooks; consumers must wrap to restyle heavily.
- Endpoint `/auth/mfa/challenge` returns a session-cookie-bound MFA pass on the server (no token returned in this UI; provider re-hydrates on the next mount or context refresh).
- Resets only the local `code` state on success — does not auto-trigger a re-hydrate of `AuthContext`. Consumers may want to call a custom refresh in `onSuccess`.
