# OrganizationSwitcher.tsx

**Path:** `packages/shark-auth-react/src/components/OrganizationSwitcher.tsx`
**Type:** React component — `<select>` org picker
**LOC:** 87

## Purpose
Lists the signed-in user's organizations and lets them switch the active one with a `PUT` to the server.

## Public API
- `interface OrganizationSwitcherProps { onOrganizationChange?: (org: Organization) => void }`
- `function OrganizationSwitcher(props): JSX.Element | null`

## How it composes
1. Reads `AuthContext` for the bound client and `useOrganization()` for `{ isLoaded, organization }` (current active org).
2. On mount (when context loaded), **GET** `/api/v1/organizations`. Accepts either `{ organizations: Organization[] }` or a bare array.
3. Renders a styled `<select>` with each org by id+name. Returns `null` if list is empty.
4. On change: looks up the picked org, **PUT** `/api/v1/organizations/active` with JSON `{ organizationId }`. Best-effort — errors are swallowed silently after the local switch.
5. Calls `onOrganizationChange?.(selected)` if the PUT didn't throw.

## Loading / error states
- `Loading organizations…` (gray text) while the GET is in flight.
- Red error message if the GET fails.
- `null` until `useOrganization().isLoaded` is true.

## Internal dependencies
- `hooks/useOrganization`
- `hooks/context.AuthContext`
- `core/types.Organization`

## Used by (consumer-facing)
- Drop-in for header/navbar org switching, typically alongside `<UserButton>`.

## Notes
- `<select value={organization?.id ?? ''}>` — reflects current active org once context updates after hydration.
- Inline-styled; no theming hook.
- After a successful switch the SDK does NOT re-fetch `/api/v1/users/me` automatically — `onOrganizationChange` is the consumer's hook to refresh app state.
