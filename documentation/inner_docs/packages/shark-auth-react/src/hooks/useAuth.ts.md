# useAuth.ts

**Path:** `packages/shark-auth-react/src/hooks/useAuth.ts`
**Type:** React hook — primary auth API
**LOC:** 13

## Purpose
The main hook a consumer reaches for. Returns the auth status flags plus `getToken` and `signOut` callbacks.

## Public API
- `useAuth(): { isLoaded, isAuthenticated, getToken, signOut }`

### Return shape
- `isLoaded: boolean` — true once the provider has finished its first hydrate (or determined no token).
- `isAuthenticated: boolean` — true when the provider has a valid `user` + `session`.
- `getToken(opts?: GetTokenOptions): Promise<string | GetTokenResult | null>` — see `context.ts`. Auto-refreshes via stored refresh token; with `{ dpop: true, method, url }` returns `{ token, dpop }`.
- `signOut(): Promise<void>` — best-effort revoke + local clearAll.

## Params
None.

## When it re-renders
Whenever the `AuthContext` value changes — i.e. on hydrate completion, sign-in, sign-out, refresh failure, or when `SharkProvider` rebuilds its memoized value.

## Internal dependencies
- `react.useContext`
- `./context.AuthContext`

## Used by (consumer-facing)
- `<SignedIn>` / `<SignedOut>` — gating render.
- `<UserButton>` — `signOut`.
- Every consumer page that needs to call an authenticated API (`getToken`).

## Notes
- Throws `'useAuth must be used within SharkProvider'` if context is absent.
- Intentionally narrow surface — `user`/`session`/`organization` live on dedicated hooks (`useUser`, `useSession`, `useOrganization`) for selective re-renders, even though all four hooks currently subscribe to the same context value.
