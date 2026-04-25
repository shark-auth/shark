# SharkCallback.tsx

**Path:** `packages/shark-auth-react/src/components/SharkCallback.tsx`
**Type:** React component — OAuth redirect handler
**LOC:** 68

## Purpose
The page that lives at `/shark/callback` (the URI hard-coded in `core/auth.ts`). Reads `?code=` from the URL, swaps it for tokens, then redirects to the post-login destination.

## Public API
- `interface SharkCallbackProps { fallbackRedirectUrl?: string; loading?: React.ReactNode; onError?: (err: Error) => void }`
- `function SharkCallback(props): JSX.Element`

### Props
- `fallbackRedirectUrl` — used only when `shark_redirect_after` is missing from sessionStorage. Defaults to `'/'`.
- `loading` — custom React node shown during exchange. Defaults to gray `Signing in…` text.
- `onError` — fired with an `Error` if no code is present or token exchange fails.

## How it composes
1. Reads `authUrl` + `publishableKey` from `AuthContext`. Bails silently if missing (dev-time misconfig).
2. Parses `window.location.search` for `code`. Missing → fires `onError(new Error('No authorization code in URL'))`.
3. Calls `exchangeCodeForToken(code, authUrl, publishableKey)` (which internally PKCE-verifies and persists tokens via `setAccessToken`).
4. On success: `window.location.replace(getRedirectAfter() ?? fallbackRedirectUrl)`.
5. On failure: shows red `Authentication failed: <message>` and fires `onError`.

## Internal dependencies
- `core/auth.exchangeCodeForToken`
- `core/storage.getRedirectAfter`
- `hooks/context.AuthContext`

## Used by (consumer-facing)
- Mounted at the route `/shark/callback`. With React Router:
  ```tsx
  <Route path="/shark/callback" element={<SharkCallback />} />
  ```

## Notes
- Uses `window.location.replace` (not `assign`) so the callback URL is removed from browser history.
- Effect deps `[ctx, fallbackRedirectUrl, onError]` — re-runs if `onError` identity changes; consumers should `useCallback` it to avoid duplicate exchanges.
