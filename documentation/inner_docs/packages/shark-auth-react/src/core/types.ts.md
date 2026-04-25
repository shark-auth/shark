# types.ts

**Path:** `packages/shark-auth-react/src/core/types.ts`
**Type:** Public TypeScript type definitions
**LOC:** 30

## Purpose
The package's exported domain model — the shapes consumers see in hook return values and component props.

## Public API (exported types)
- `User` — `{ id: string; email: string; firstName?: string; lastName?: string; imageUrl?: string }`
- `Session` — `{ id: string; userId: string; expiresAt: number }` (`expiresAt` is ms since epoch)
- `Organization` — `{ id: string; name: string; slug: string }`
- `AuthConfig` — `{ authUrl: string; publishableKey: string }` (informational; `SharkProvider` accepts these as separate props)
- `TokenPair` — `{ accessToken: string; refreshToken?: string; expiresAt: number }`

## Internal dependencies
None. Pure types.

## Used by (consumer-facing)
- `useUser` returns `{ user: User | null }`
- `useSession` returns `{ session: Session | null }`
- `useOrganization` returns `{ organization: Organization | null }`
- `OrganizationSwitcher` uses `Organization` for its `onOrganizationChange` callback
- `SharkProvider` populates the context with these types after hydration

## Notes
- `TokenPair` is currently exported for API completeness but the actual provider state stores tokens in sessionStorage rather than passing this shape around in React state.
- `AuthConfig` mirrors the props of `SharkProvider` — no runtime use.
