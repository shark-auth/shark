# useSession.ts

**Path:** `packages/shark-auth-react/src/hooks/useSession.ts`
**Type:** React hook — current session
**LOC:** 11

## Purpose
Returns the session record hydrated by `SharkProvider`.

## Public API
- `useSession(): { isLoaded: boolean; session: Session | null }`

### Return shape
- `isLoaded` — provider hydration completion flag.
- `session` — `Session` (`{ id, userId, expiresAt }`, `expiresAt` is ms since epoch) or `null` when signed out.

## Params
None.

## When it re-renders
On every `AuthContext` value change — hydrate, sign-in, sign-out, refresh failure.

## Internal dependencies
- `react.useContext`
- `./context.AuthContext`

## Used by (consumer-facing)
- Consumers that need `session.expiresAt` for UI countdowns or to decide when to proactively call `getToken`.

## Notes
- Throws `'useSession must be used within SharkProvider'` if context is missing.
- The `Session.id` falls back to the JWT's `jti` claim when `/api/v1/users/me` is unreachable; `expiresAt` falls back to `exp * 1000` (or `now + 1h` if absent).
