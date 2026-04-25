# index.ts (components barrel)

**Path:** `packages/shark-auth-react/src/components/index.ts`
**Type:** Subpath barrel
**LOC:** 10

## Purpose
Re-exports every shippable React component so they're reachable from the package root.

## Public API
```ts
export * from './SharkProvider'
export * from './SignIn'
export * from './SignUp'
export * from './SignedIn'
export * from './SignedOut'
export * from './UserButton'
export * from './MFAChallenge'
export * from './PasskeyButton'
export * from './OrganizationSwitcher'
export * from './SharkCallback'
```

## Internal dependencies
- All 10 sibling component modules.

## Used by (consumer-facing)
- Root `src/index.ts` re-exports this barrel — consumers import everything from `@shark-auth/react`.

## Notes
- No standalone subpath in `package.json` for components (only `core` and `hooks` have subpaths) — components must be imported from the root.
- Adding a new component requires adding the export here in addition to creating the file.
