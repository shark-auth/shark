# Proxy lifecycle — state machine

The `proxy.Manager` owns the lifecycle of a pool of `*Listener`. It exposes four public methods — `Start`, `Stop`, `Reload`, `Status` — and serializes every transition through a single `sync.Mutex` so concurrent calls line up deterministically.

## States

```
         Start (ok)
Stopped ─────────────▶ Running
   ▲                      │
   │ Stop               Reload │ (Stop+Start in one critical section)
   │                      ▼
   └──────────────── Reloading
                         │
            Start (ok)   │   Start (err)
         ┌───────────────┴───────────────┐
         ▼                               ▼
      Running                        Stopped
```

- **Stopped** — initial state. No listeners are bound, no goroutines are serving traffic. Idempotent; `Stop` on a stopped Manager returns nil without touching state.
- **Running** — every listener built by the `ListenerBuilder` closure has successfully bound its port and is serving traffic. `startedAt` is stamped with the UTC wall clock.
- **Reloading** — a transient state held during a `Reload` call so a concurrent `Status` snapshot mid-reload reflects the truth. Readers should back off rather than assume the engine is quiescent.

## Transitions

| From → To | Triggered by | Notes |
|---|---|---|
| Stopped → Running | `Start(ctx)` | Builder is invoked, every listener binds, state + `startedAt` updated atomically under the mutex. On any bind failure every already-started listener is Shutdown'd so partial-Running is never observable. Returns error on builder failure, bind failure, or already-Running. |
| Running → Stopped | `Stop(ctx)` | Every listener is Shutdown'd (ctx governs per-listener http.Server.Shutdown deadline). First error wins; subsequent errors are logged but the transition completes. |
| Running → Reloading → Running | `Reload(ctx)` | Stop + Start executed inside a single `mu.Lock`/`defer mu.Unlock`. If Stop fails, state drops to Stopped and error is surfaced. If Start fails, state remains Stopped and `lastError` carries the cause. |
| Reloading → Stopped | `Reload(ctx)` with Start failure | Builder failure after Stop completes leaves the Manager Stopped; admin must call `Start` or fix the builder config and retry. |

## Status projection

`Manager.Status()` returns a JSON-friendly snapshot. Every field is safe to marshal on a stopped Manager — empty string for `started_at` and `last_error` means "no value, not an error".

```json
{
  "state":        1,
  "state_str":    "running",
  "listeners":    2,
  "rules_loaded": 17,
  "started_at":   "2026-04-24T10:09:45Z",
  "last_error":   ""
}
```

`rules_loaded` sums the compiled rule count across every listener's engine, so multi-listener setups get a total. Admins who want per-listener counts should iterate `Manager.Listeners()`.

## Builder contract

`ListenerBuilder` is invoked on every `Start` and every `Reload`, not just the first `Start`. This lets an admin mutate YAML listener config or DB rules and call `Reload` to get a fresh listener set without process restart. Old listeners are Shutdown'd before the new builder is called, so the builder never sees stale ports held by the previous generation.

A nil builder is rejected eagerly at `NewManager` time so misuse surfaces at wiring time, not at first `Start`.

## Race handling

Every public method acquires `mu` before inspecting state. The only lock-free reader is `Manager.Listeners()` which returns a defensive copy of the slice header — callers must not mutate the returned listeners' lifecycle directly, because doing so would bypass Manager and break state tracking.

`CurrentEngine()` exposes the first listener's `*Engine` so the admin rule-CRUD handlers can call `SetRules` for hot-reload. It returns nil when the Manager is not Running or has no listeners; call sites branch on nil rather than waiting — there is no semantically correct wait in the middle of a CRUD handler.
