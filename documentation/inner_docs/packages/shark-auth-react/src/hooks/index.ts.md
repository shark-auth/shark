# index.ts (hooks barrel)

**Path:** `packages/shark-auth-react/src/hooks/index.ts`
**Type:** Subpath barrel
**LOC:** 5

## Purpose
Aggregates the hooks + context exports so they're reachable both from the package root and from `@sharkauth/react/hooks`.

## Public API
```ts
export * from './context'          // AuthContext, AuthContextValue, GetTokenOptions, GetTokenResult
export * from './useAuth'          // useAuth()
export * from './useUser'          // useUser()
export * from './useSession'       // useSession()
export * from './useOrganization'  // useOrganization()
```

## Internal dependencies
- All sibling hook modules + `context.ts`.

## Used by (consumer-facing)
- Root `src/index.ts` re-exports this barrel.
- `package.json` `./hooks` subpath points here for lighter-weight imports.

## Notes
- Re-exporting `context.ts` deliberately exposes `AuthContext` and the `AuthContextValue` type, so advanced consumers can `useContext(AuthContext)` directly (e.g. for inline `client.fetch` calls without going through a hook).
