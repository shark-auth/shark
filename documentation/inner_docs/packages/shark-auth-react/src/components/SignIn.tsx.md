# SignIn.tsx

**Path:** `packages/shark-auth-react/src/components/SignIn.tsx`
**Type:** React component — sign-in launcher
**LOC:** 44

## Purpose
A clickable element (button by default, `<span>` if children) that kicks off the OAuth 2.1 PKCE flow by redirecting to `<authUrl>/oauth/authorize`.

## Public API
- `interface SignInProps { redirectUrl?: string; children?: React.ReactNode; className?: string }`
- `function SignIn(props): JSX.Element`

### Props
- `redirectUrl` — where to send the user after a successful login (defaults to `window.location.href`). Stored as `shark_redirect_after` and consumed by `SharkCallback`.
- `children` — if present, renders a `<span>` wrapper around them so consumers can supply custom triggers (icons, links). If absent, renders `<button>Sign in</button>`.
- `className` — passed to whichever element is rendered.

## How it composes
1. Reads `authUrl` + `publishableKey` from `AuthContext`. If missing, logs `[SharkAuth] SignIn must be rendered inside <SharkProvider>` and bails.
2. Calls `startAuthFlow(redirectUrl, authUrl, publishableKey)` to build the authorize URL (PKCE verifier + state + storage side-effects happen inside).
3. `window.location.href = url`.

## Internal dependencies
- `core/auth.startAuthFlow`
- `hooks/context.AuthContext`

## Used by (consumer-facing)
- Direct: dropped anywhere consumers want a sign-in trigger, often inside `<SignedOut>...</SignedOut>`.

## Notes
- Re-renders only when `ctx` or `redirectUrl` changes (callback memoized via `useCallback`).
- Wrapping a custom element in `<span>` rather than spreading onClick onto children means children with their own click handlers will see both.
