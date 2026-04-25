# vitest.config.ts

**Path:** `packages/shark-auth-react/vitest.config.ts`
**Type:** Test runner configuration (vitest)
**LOC:** 11

## Purpose
Wires up vitest for the package: jsdom environment for component tests, automatic JSX runtime, global test APIs.

## Public API
- Default-exports a vitest `defineConfig({...})` object.

### Config
- `esbuild.jsx: 'automatic'` — modern React 17+ JSX transform; no need to import React in `.tsx` test files.
- `test.environment: 'jsdom'` — enables `window`, `document`, `sessionStorage`, `crypto.subtle` polyfills required by storage, DPoP, and Testing Library.
- `test.globals: true` — allows `describe`/`it`/`expect` without explicit imports.

## Internal dependencies
- `vitest` (devDependency)
- Implicit: `@testing-library/react`, `jsdom` (devDependencies).

## Used by (consumer-facing)
- Test-time only — does not influence the published `dist/`.

## Notes
- No setup file — global mocks would have to be added via a `setupFiles` array if needed.
- `crypto.subtle` is provided by jsdom in modern Node, so `dpop.ts` runs in tests without polyfills.
