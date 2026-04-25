# errors.ts

**Path:** `sdk/typescript/src/errors.ts`
**Type:** Exception hierarchy
**LOC:** 53

## Purpose
Single source of truth for SDK error classes. All errors extend `SharkAuthError`, so consumers can catch broadly or narrowly.

## Public API
- `class SharkAuthError extends Error` — base, fixes prototype chain for transpiled ES5+
- `class DPoPError extends SharkAuthError` — DPoP signing / key load failures
- `class DeviceFlowError extends SharkAuthError` — RFC 8628 grant failures
- `class VaultError extends SharkAuthError`
  - `readonly statusCode: number | undefined` — HTTP status from vault endpoint
- `class TokenError extends SharkAuthError` — JWT verify/decode failures
- `class SharkAPIError extends SharkAuthError` — admin API non-2xx
  - `readonly code: string` — server-supplied error code (e.g. `invalid_proxy_rule`)
  - `readonly status: number` — HTTP status

## Usage patterns
```ts
try { await client.proxyRules.createRule(...) }
catch (e) {
  if (e instanceof SharkAPIError && e.code === "invalid_proxy_rule") { ... }
  if (e instanceof SharkAuthError) { ... } // catch-all
}
```

## Internal dependencies
- None.

## Notes
- `Object.setPrototypeOf(this, new.target.prototype)` keeps `instanceof` working under tsup CJS down-compilation.
- `name` is auto-set from `this.constructor.name` so stack traces show the subclass.
- Every admin namespace throws `SharkAPIError` exclusively (no per-namespace error subclasses).
