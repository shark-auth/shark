# SignedOut.tsx

**Path:** `packages/shark-auth-react/src/components/SignedOut.tsx`
**Type:** Conditional render gate
**LOC:** 12

## Purpose
Mirror of `SignedIn`: renders children only when the user is unauthenticated. Returns `null` during initial hydration to avoid flash-of-signed-out content.

## Public API
- `interface SignedOutProps { children: React.ReactNode }`
- `function SignedOut({ children }): JSX.Element | null`

## How it composes
- Reads `{ isLoaded, isAuthenticated }` from `useAuth()`.
- Pre-hydration (`!isLoaded`) → `null`.
- Authenticated → `null`.
- Otherwise → `<>{children}</>`.

## Internal dependencies
- `hooks/useAuth`

## Used by (consumer-facing)
- Wrap login surfaces: `<SignedOut><SignIn /><SignUp /></SignedOut>`.

## Notes
- Renders identically on every context change — no internal state.
- Together with `<SignedIn>` provides the canonical Clerk-style swap pattern.
