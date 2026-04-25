# package.json

**Path:** `packages/shark-auth-react/package.json`
**Type:** npm package manifest
**LOC:** 51

## Identity
- **name:** `@shark-auth/react`
- **version:** `0.1.0`
- **description:** React components + hooks for SharkAuth
- **type:** `module` (ESM-first)

## Entry points
- **main (CJS):** `./dist/index.cjs`
- **module (ESM):** `./dist/index.js`
- **types:** `./dist/index.d.ts`

## Exports map
- `.` → root (`SharkProvider`, all components/hooks/core re-exports)
- `./core` → low-level primitives (`createClient`, `exchangeToken`, `decodeClaims`, storage, DPoP)
- `./hooks` → just the hooks bundle (`useAuth`, `useUser`, `useSession`, `useOrganization`)

Each subpath ships `import` (ESM), `require` (CJS), and `types`.

## Files shipped
- `dist/` only — no source, no tests, no configs.

## Dependencies
- `jose ^5.9.0` — JWT verification (`createRemoteJWKSet`, `jwtVerify`) for JWKS-based token validation in `core/jwt.ts`.

## Peer dependencies
- `react >=18.0.0`
- `react-dom >=18.0.0`

Treated as `external` in tsup so consumers de-dupe their React install.

## Dev dependencies
- `tsup ^8.0.0` — bundler
- `vitest ^1.0.0` + `@testing-library/react` + `jsdom` — test runner
- `typescript ^5.0.0`, `@types/react ^18.0.0`

## Scripts
- `build` → `tsup` (reads `tsup.config.ts`)
- `dev` → `tsup --watch`
- `test` → `vitest`

## Publish workflow
No `prepublishOnly` or `publishConfig` set — assume manual `pnpm build && npm publish` from the package dir. The `files` allowlist guarantees only `dist/` ships.
