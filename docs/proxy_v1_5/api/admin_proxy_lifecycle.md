# Admin API — Proxy lifecycle

## Purpose

Lets admins flip the embedded reverse-proxy subsystem on and off at runtime without restarting the main `shark serve` process. The handlers delegate to `proxy.Manager` (see `lifecycle/state_machine.md`) which owns the listener pool and the Stopped/Running/Reloading state machine; every transition runs in a single critical section so a caller cannot observe a partly-bound pool.

## Routes

| Method | Path | Handler symbol |
|---|---|---|
| GET  | `/api/v1/admin/proxy/lifecycle` | `Server.handleProxyLifecycleStatus` |
| POST | `/api/v1/admin/proxy/start`     | `Server.handleProxyLifecycleStart`  |
| POST | `/api/v1/admin/proxy/stop`      | `Server.handleProxyLifecycleStop`   |
| POST | `/api/v1/admin/proxy/reload`    | `Server.handleProxyLifecycleReload` |

## Auth required

Admin API key. The same `AdminAPIKeyFromStore` middleware gates every v1.5 admin route.

## Request shape

All four handlers accept an empty body. No query parameters are consumed today; if a future hot-reload wants to accept (say) a "rebuild listeners" flag the convention is a JSON POST body with an explicit field, never a query string — query strings would be lossy for the audit log.

Example:

```bash
curl -XPOST -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
     https://auth.example.com/api/v1/admin/proxy/start
```

## Response shape

### Success

All four routes return the same `Status` envelope so clients can share a single parse path:

```json
{
  "data": {
    "state":        1,
    "state_str":    "running",
    "listeners":    2,
    "rules_loaded": 17,
    "started_at":   "2026-04-24T10:09:45Z",
    "last_error":   ""
  }
}
```

Field notes:

- `state` is the integer enum from `proxy.State`; prefer `state_str` for UI.
- `state_str` is one of `stopped`, `running`, `reloading`, `unknown`.
- `listeners` counts actively bound listener ports.
- `rules_loaded` is summed across every listener's compiled rule set.
- `started_at` is RFC3339 UTC; empty string when stopped.
- `last_error` is the most recent error the Manager recorded; empty string when the current transition succeeded.

### Error

```json
{ "error": { "code": "proxy_start_failed", "message": "bind: address already in use" } }
```

Error codes: `proxy_start_failed`, `proxy_stop_failed`, `proxy_reload_failed`.

## Status codes

- `200 OK` — successful start/stop/reload (Stop is idempotent — stopping a stopped Manager returns 200 with current state).
- `401 Unauthorized` — missing/invalid admin key.
- `404 Not Found` — `ProxyManager` was never wired (proxy disabled at boot). The route always 404s in that case so the dashboard can branch cleanly without inspecting config first.
- `409 Conflict` — transition rejected (e.g. Start on a Manager already Running, or builder/bind failure).

## Side effects

- Start: builds a fresh listener set via the `ListenerBuilder` closure, binds each one, transitions `state` to Running, stamps `started_at`.
- Stop: shuts down every listener (ctx governs the per-listener http.Server.Shutdown deadline), clears `listeners`, transitions to Stopped, zeroes `started_at`.
- Reload: Stop + Start in one critical section — no window exists where a concurrent Start can sneak in. On failure the Manager drops back to Stopped and surfaces the error via `last_error`. Reload also calls `refreshProxyEngineFromDB` so DB rule mutations layered since the last reload go live.
- No DB writes by lifecycle handlers themselves. No audit-log entry is written by these handlers today — add one in a follow-up if operators want the transitions in their audit history.

## Frontend hint

Render a lifecycle toggle + status chip in the Proxy tab header: GET `/lifecycle` every 5s (or open the SSE stream if you're already consuming `/proxy/status/stream`), then show a "Start / Stop / Reload" split button whose enabled state derives from `state_str`. A red badge on `last_error` lets the operator page-down into the message. Persist the last-known state in dashboard local storage so the page doesn't flicker during the initial fetch.
