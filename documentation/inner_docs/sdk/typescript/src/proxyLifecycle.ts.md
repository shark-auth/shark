# proxyLifecycle.ts

**Path:** `sdk/typescript/src/proxyLifecycle.ts`
**Type:** Admin namespace — proxy lifecycle control
**LOC:** 125

## Purpose
Start, stop, reload, and inspect the embedded reverse-proxy without restarting the SharkAuth server process. Wraps four admin endpoints.

## Public API
- `class ProxyLifecycleClient`
  - `constructor(opts: ProxyLifecycleClientOptions)`
  - `getProxyStatus(): Promise<ProxyStatus>` — GET `/api/v1/admin/proxy/lifecycle`
  - `startProxy(): Promise<ProxyStatus>` — POST `/api/v1/admin/proxy/start` (409 if already running)
  - `stopProxy(): Promise<ProxyStatus>` — POST `/api/v1/admin/proxy/stop` (idempotent)
  - `reloadProxy(): Promise<ProxyStatus>` — POST `/api/v1/admin/proxy/reload` (atomic stop+start, re-publishes DB rules)

## ProxyStatus shape
- `state: number` — integer enum from server `proxy.State`
- `state_str: "stopped" | "running" | "reloading" | "unknown"` — UI-safe label
- `listeners: number` — bound listener count
- `rules_loaded: number` — total compiled rules across listeners
- `started_at: string` — RFC3339 UTC; empty when stopped
- `last_error: string` — most recent Manager error; empty on success

## Constructor options
- `baseUrl: string`
- `adminKey: string` — Bearer token

## Error mapping
- All non-200 responses → `SharkAPIError(message, code, status)` parsed from server envelope `{error:{code,message}}`.

## Internal dependencies
- `http.ts`, `errors.ts`

## Notes
- All four methods reuse `_parseStatus` for consistent error handling.
- `reloadProxy` is the recommended way to apply rule changes when `engine_refresh_error` returns from `proxyRules` mutations.
- Trailing slashes on `baseUrl` are stripped at construction.
