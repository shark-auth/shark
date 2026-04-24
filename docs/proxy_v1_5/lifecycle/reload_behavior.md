# Proxy lifecycle — reload behaviour

There are two distinct "reload" surfaces in v1.5 and they do different things. This doc explains both and clarifies when each fires.

## 1. Engine-rule refresh (hot path)

`Server.refreshProxyEngineFromDB` reads every enabled global rule from `proxy_rules` and calls `Engine.SetRules`. No listener lifecycle is touched — ports stay bound, in-flight requests are unaffected.

**When it fires:**

- After every successful `POST /api/v1/admin/proxy/rules/db` (create).
- After every successful `PATCH /api/v1/admin/proxy/rules/db/{id}` (update).
- After every successful `DELETE /api/v1/admin/proxy/rules/db/{id}`.
- After `POST /api/v1/admin/proxy/rules/import` finishes processing the batch.
- After `POST /api/v1/admin/proxy/reload` (in addition to the listener rebuild — see below).

**Atomicity:** `Engine.SetRules` compiles the new spec slice before calling `atomic.Pointer.Store`, so a partially-compiled set is never visible. Readers either see the old snapshot in full or the new one in full. Compile failure leaves the previous set in place and returns the error to the caller — the proxy keeps serving the last-known-good configuration.

**Error surfacing:** refresh failures are non-fatal for the DB mutation that triggered them. The handler attaches the error as `engine_refresh_error` in the JSON response so the operator can see the DB write succeeded but the live engine is stale. Next successful mutation re-publishes the full set.

## 2. Listener rebuild (Manager.Reload)

`proxy.Manager.Reload` is a full Stop + Start in a single critical section. Listeners are re-created via the `ListenerBuilder` closure, ports are re-bound, breakers are re-initialized. In-flight requests are drained via `http.Server.Shutdown` subject to the ctx deadline.

**When it fires:**

- `POST /api/v1/admin/proxy/reload` only. No other handler triggers it.

**Why you'd use it:**

- You changed YAML listener config (bind address, upstream URL, TrustedHeaders, timeout) and want it live without a process restart.
- You want to force a breaker re-init (e.g. after an upstream health URL changes).
- A listener is stuck and you want a guaranteed clean re-bind.

**What it does NOT do:**

- It does not re-load YAML config from disk. Start with a process restart if you edited `sharkauth.yaml`.
- It does not re-run migrations.
- It does not fix a misconfigured `ListenerBuilder`. If the builder is broken, Reload surfaces the error and leaves the Manager Stopped; fix the config and call `POST /api/v1/admin/proxy/start`.

## Race semantics

Because `Manager.mu` serializes Start / Stop / Reload, there is no window where a caller sees a partly-bound pool. Reload holds `mu` across both Stop and Start; a concurrent `Start` that arrives during Reload blocks until Reload returns, then immediately sees "already Running" and returns `409 Conflict`.

Engine refresh uses `atomic.Pointer` rather than a lock, so the two surfaces compose: a rule-CRUD mutation during Reload's brief Reloading window will succeed at the DB layer, publish via `Engine.SetRules` on whichever Engine is currently mounted (the old one, if Reload hasn't torn it down yet; the new one otherwise), and settle correctly on the next Reload or mutation.

## Observability

Every transition updates `Manager.Status()` — listen on the SSE stream at `GET /api/v1/admin/proxy/status/stream` to watch state transitions without polling. The handler ticks at a heartbeat interval and pushes deltas when state or listener count changes.
