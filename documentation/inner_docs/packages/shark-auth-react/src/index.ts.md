# index.ts

**Path:** `packages/shark-auth-react/src/index.ts`
**Type:** Package barrel
**LOC:** 3

## Purpose
Single root entry that re-exports everything from `core`, `hooks`, and `components` so consumers can `import { SharkProvider, useAuth } from '@shark-auth/react'`.

## Public API
```ts
export * from './core'
export * from './hooks'
export * from './components'
```

Surfaces (transitively):
- All components: `SharkProvider`, `SignIn`, `SignUp`, `SignedIn`, `SignedOut`, `UserButton`, `MFAChallenge`, `PasskeyButton`, `OrganizationSwitcher`, `SharkCallback`
- All hooks: `useAuth`, `useUser`, `useSession`, `useOrganization`, plus the `AuthContext` + `AuthContextValue` type
- All core primitives: `createClient`, `exchangeToken`, `startAuthFlow`, `decodeClaims`, `verifyToken`, `generateDPoPProver`, storage helpers, types

## Internal dependencies
- `./core` (barrel)
- `./hooks` (barrel)
- `./components` (barrel)

## Used by (consumer-facing)
- Every consumer of the package — the default import path.

## Notes
- Subpath exports (`@shark-auth/react/core`, `@shark-auth/react/hooks`) exist in `package.json` for tree-shaking-conscious consumers who only need primitives.
