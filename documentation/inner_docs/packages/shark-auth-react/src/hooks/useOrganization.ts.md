# useOrganization.ts

**Path:** `packages/shark-auth-react/src/hooks/useOrganization.ts`
**Type:** React hook — current active organization
**LOC:** 11

## Purpose
Returns the user's currently active `Organization` (single tenancy slot) as hydrated by `SharkProvider`.

## Public API
- `useOrganization(): { isLoaded: boolean; organization: Organization | null }`

### Return shape
- `isLoaded` — provider hydration completion flag.
- `organization` — `Organization` (`{ id, name, slug }`) or `null` (not signed in / not in any org).

## Params
None.

## When it re-renders
Whenever `AuthContext` updates — hydrate, sign-in, sign-out, refresh, etc. Note: switching the active org via `<OrganizationSwitcher />` does NOT auto-refresh this hook — the server is informed but `SharkProvider` doesn't re-hydrate. Consumers can listen via the switcher's `onOrganizationChange` callback.

## Internal dependencies
- `react.useContext`
- `./context.AuthContext`

## Used by (consumer-facing)
- `<OrganizationSwitcher>` — drives the `<select value=...>` reflecting the current org.
- Consumer pages scoped to "current org" data.

## Notes
- Throws `'useOrganization must be used within SharkProvider'` if used outside the provider.
- The `organization` is populated only by `/api/v1/users/me` on hydrate. The JWT-claim fallback path in `SharkProvider` deliberately leaves it `null`.
