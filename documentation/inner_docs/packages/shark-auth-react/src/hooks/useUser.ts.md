# useUser.ts

**Path:** `packages/shark-auth-react/src/hooks/useUser.ts`
**Type:** React hook — current user
**LOC:** 11

## Purpose
Tiny hook that returns the resolved `User` object hydrated by `SharkProvider`.

## Public API
- `useUser(): { isLoaded: boolean; user: User | null }`

### Return shape
- `isLoaded` — provider hydration completion flag.
- `user` — `User` (`{ id, email, firstName?, lastName?, imageUrl? }`) or `null` when signed out.

## Params
None.

## When it re-renders
Whenever `AuthContext` value changes — which is whenever the provider re-memoizes (sign-in, sign-out, hydrate, refresh, signOut).

## Internal dependencies
- `react.useContext`
- `./context.AuthContext`

## Used by (consumer-facing)
- `<UserButton>` — derives initials, displays email, decides whether to render at all.
- Consumer pages that need the current user.

## Notes
- Throws `'useUser must be used within SharkProvider'` if used outside the provider.
- The `User` object comes from `/api/v1/users/me` on hydrate, with a JWT-claim fallback (`sub`, `email`, `firstName`, `lastName`, `imageUrl`) when that endpoint is unreachable.
