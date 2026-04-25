# package.json

**Path:** `sdk/typescript/package.json`
**Type:** NPM package manifest
**LOC:** 47

## Purpose
Defines the published `@sharkauth/node` package — Node 18+ TypeScript SDK for SharkAuth agent-auth primitives and v1.5 admin APIs.

## Identity
- `name`: `@sharkauth/node`
- `version`: `0.1.0`
- `description`: TypeScript SDK for SharkAuth agent-auth primitives
- `type`: `module` (native ESM)
- `engines.node`: `>=18`

## Build outputs (dist/)
- `main`: `./dist/index.cjs` (CommonJS)
- `module`: `./dist/index.js` (ESM)
- `types`: `./dist/index.d.ts`

## Exports
Single root export `"."` with dual-mode resolution:
- ESM: `dist/index.js` + `dist/index.d.ts`
- CJS: `dist/index.cjs` + `dist/index.d.cts`

## Files (published to npm)
`dist/`, `README.md`, `LICENSE`.

## Scripts
- `build` — `tsup` (ESM + CJS + dts)
- `test` — `vitest run --coverage`
- `test:watch` — `vitest`
- `typecheck` / `lint` — `tsc --noEmit`

## Dependencies
- Runtime: `jose ^5.9.6` (JOSE/JWT primitives — DPoP signing, JWKS verification)

## Dev dependencies
- `tsup ^8.3.0` — bundler
- `typescript ^5.7.0`
- `vitest ^2.0.0` + `@vitest/coverage-v8`
- `@types/node ^22`

## Publish workflow
1. `npm run build` — emits `dist/` via `tsup.config.ts`
2. `npm run test` — vitest with 80/80/70/80 coverage gates
3. `npm publish` — only `dist/` + README + LICENSE go up

## Notes
- Single dependency footprint by design — `jose` is the only runtime dep.
- Native fetch (Node 18+) — no `node-fetch` polyfill.
- Hand-written types throughout (no codegen).
