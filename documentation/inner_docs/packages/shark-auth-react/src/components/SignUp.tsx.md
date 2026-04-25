# SignUp.tsx

**Path:** `packages/shark-auth-react/src/components/SignUp.tsx`
**Type:** React component — sign-up launcher
**LOC:** 46

## Purpose
Identical to `SignIn` except it appends `&screen_hint=signup` to the authorize URL so the auth server can render the registration screen instead of login.

## Public API
- `interface SignUpProps { redirectUrl?: string; children?: React.ReactNode; className?: string }`
- `function SignUp(props): JSX.Element`

### Props
- `redirectUrl` — where to land after registration completes (defaults to `window.location.href`). Persisted via `shark_redirect_after`.
- `children` — when present, rendered inside a `<span style={cursor:pointer}>`; otherwise renders `<button>Sign up</button>`.
- `className` — applied to whichever element is rendered.

## How it composes
1. Pulls `authUrl` + `publishableKey` from `AuthContext`; logs an error if missing.
2. `startAuthFlow(redirectUrl, authUrl, publishableKey)` to build the authorize URL with PKCE.
3. Concatenates `&screen_hint=signup` onto the result.
4. `window.location.href = finalUrl`.

## Internal dependencies
- `core/auth.startAuthFlow`
- `hooks/context.AuthContext`

## Used by (consumer-facing)
- Drop-in next to `<SignIn>`, typically inside `<SignedOut>` blocks.

## Notes
- The `screen_hint=signup` parameter must be honored by the auth server; if it's not, the user lands on the standard login screen with a toggle.
- Click handler is `useCallback`-memoized on `[ctx, redirectUrl]`.
