# vitest.config.ts

**Path:** `sdk/typescript/vitest.config.ts`
**Type:** Test runner config
**LOC:** 20

## Purpose
Configures vitest for the SDK test suite with v8 coverage and CI thresholds.

## Settings
- `environment`: `node`
- `include`: `test/**/*.test.ts`
- Coverage `provider`: `v8`
- Coverage `include`: `src/**/*.ts`
- Coverage `exclude`: `src/index.ts` (barrel only re-exports)
- Reporters: `text`, `lcov`

## Coverage thresholds
- lines: 80
- functions: 80
- branches: 70
- statements: 80

## Notes
- Tests live in `test/`, not `src/__tests__/`.
- `lcov` output enables CI integration (Codecov, Coveralls).
