# tsup.config.ts

**Path:** `packages/shark-auth-react/tsup.config.ts`
**Type:** Bundler configuration (tsup)
**LOC:** 10

## Purpose
Defines how `pnpm build` compiles the TypeScript source into the published `dist/` artifact.

## Public API
- Default-exports a `defineConfig({...})` object.

### Config
- `entry: ['src/index.ts', 'src/core/index.ts', 'src/hooks/index.ts']` — three entry points, matching the three subpath exports in `package.json` (`.`, `./core`, `./hooks`).
- `format: ['cjs', 'esm']` — emits both `.cjs` and `.js`.
- `dts: true` — emits `.d.ts` declaration files.
- `clean: true` — wipes `dist/` before each build.
- `splitting: false` — single-file output per entry per format (no shared chunks). Simpler resolution at the cost of some duplication between the three entries.
- `external: ['react', 'react-dom']` — peer deps stay external; consumer's React install is reused.

## Internal dependencies
- `tsup` (devDependency).

## Used by (consumer-facing)
- Indirectly: shapes the `dist/` that npm consumers download.

## Notes
- No `sourcemap` field — set true if debugging the published bundle becomes painful.
- `jose` is bundled (it's a runtime `dependencies` entry, not a peer dep, and not in `external`).
