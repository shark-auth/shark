# proxy_lifecycle_handlers.go

**Path:** `internal/api/proxy_lifecycle_handlers.go`
**Package:** `api`
**LOC:** 72
**Tests:** likely integration-tested

## Purpose
PROXYV1_5 §4.9 Lane B — runtime start/stop/reload of the reverse proxy without restarting the process. Delegates to the proxy `Manager` (`internal/proxy/lifecycle.go`) which owns the listener pool + state machine. Every handler is 404-safe so the dashboard can branch cleanly when the proxy was never wired.

## Handlers exposed
- `handleProxyLifecycleStart` (line 18) — POST `/admin/proxy/start`. Stopped → Running. 409 if already running.
- `handleProxyLifecycleStop` (line 32) — POST `/admin/proxy/stop`. Idempotent; stopping a stopped manager returns 200 with current state.
- `handleProxyLifecycleReload` (line 47) — POST `/admin/proxy/reload`. Stop+Start in one critical section. Also calls `refreshProxyEngineFromDB` so DB-backed override rules go live.
- `handleProxyLifecycleStatus` (line 66) — GET `/admin/proxy/lifecycle`. Returns `{data: ManagerStatus}` (state, listener count, rules loaded, started_at, last_error).

## Key types
None — uses `proxy.Manager.Status()` shape directly.

## Imports of note
- Stdlib `net/http` only — manager methods called via `s.ProxyManager`.

## Wired by
- `internal/api/router.go:694-697`

## Notes
- 404 when `s.ProxyManager == nil` (proxy disabled at boot).
- Reload is the only lifecycle endpoint that also refreshes the engine rule set from DB; start/stop touch only the listener pool.
- Separate route from `/proxy/status` (the breaker-stats endpoint in `proxy_handlers.go`) so existing dashboards don't break.
