# SignedIn.tsx

**Path:** `packages/shark-auth-react/src/components/SignedIn.tsx`
**Type:** Conditional render gate
**LOC:** 12

## Purpose
Renders children only when an authenticated session is present. Renders `null` while loading or when signed out.

## Public API
- `interface SignedInProps { children: React.ReactNode }`
- `function SignedIn({ children }): JSX.Element | null`

## How it composes
- Calls `useAuth()` to read `{ isLoaded, isAuthenticated }`.
- Returns `null` until `isLoaded` is true (avoids flash of authenticated UI before hydration completes).
- Returns `<>{children}</>` when authenticated, otherwise `null`.

## Internal dependencies
- `hooks/useAuth`

## Used by (consumer-facing)
- Wrapping authenticated UI: `<SignedIn><Dashboard /></SignedIn>`.
- Pairs with `<SignedOut>` to swap login/app shells without manual `if (isAuthenticated)` branching.

## Notes
- Re-renders whenever `AuthContext` value changes — i.e. on hydrate, sign-in, sign-out, refresh failure.
- No props beyond `children`. To customize the loading state, render a sibling that watches `useAuth().isLoaded`.
