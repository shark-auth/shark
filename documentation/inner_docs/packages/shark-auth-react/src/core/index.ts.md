# index.ts (core barrel)

**Path:** `packages/shark-auth-react/src/core/index.ts`
**Type:** Subpath barrel
**LOC:** 5

## Purpose
Aggregates everything in `core/` so consumers can `import { createClient, exchangeToken, decodeClaims, generateDPoPProver } from '@sharkauth/react/core'` for non-React (e.g. Node, server, custom UI) usage.

## Public API
```ts
export * from './client'    // createClient, SharkClient
export * from './auth'      // startAuthFlow, exchangeToken, exchangeCodeForToken, code verifier helpers, types
export * from './jwt'       // decodeClaims, verifyToken, SharkClaims
export * from './storage'   // get/set access/refresh tokens, code verifier, redirect, clearAll
export * from './types'     // User, Session, Organization, AuthConfig, TokenPair
```

## Internal dependencies
- All five sibling modules.

## Used by (consumer-facing)
- The package's root `src/index.ts` re-exports this barrel.
- The `./core` subpath export in `package.json` points here directly so consumers can avoid pulling in React-only code if they only need wire primitives.

## Notes
- Importing `@sharkauth/react/core` does **not** transitively import any React component — useful for SSR/middleware token verification or for non-React frameworks reusing these primitives.
- DPoP is intentionally **not** in this barrel (`core/dpop.ts` exists but isn't re-exported here); it's surfaced from the root `src/index.ts` via component re-exports — see notes in the root barrel doc.
