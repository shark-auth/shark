# tsup.config.ts

**Path:** `sdk/typescript/tsup.config.ts`
**Type:** Build config
**LOC:** 13

## Purpose
Configures `tsup` to bundle `src/index.ts` into dual ESM + CJS outputs with `.d.ts` declarations under `dist/`.

## Settings
- `entry`: `["src/index.ts"]` — single barrel
- `format`: `["esm", "cjs"]` — dual publish
- `dts`: `true` — emit `.d.ts` (and `.d.cts` for CJS)
- `clean`: `true` — wipe `dist/` before build
- `sourcemap`: `true`
- `target`: `node18`
- `outDir`: `dist`
- `splitting`: `false` — single-file bundle per format
- `shims`: `true` — adds `__dirname`/`__filename` shims for ESM

## Notes
- Pairs with `package.json` `exports` map for dual-mode resolution.
- No code splitting because the SDK is small enough to ship as one file.
