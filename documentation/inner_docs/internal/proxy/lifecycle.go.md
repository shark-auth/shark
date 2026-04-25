# lifecycle.go

**Path:** `internal/proxy/lifecycle.go`
**Package:** `proxy`
**LOC:** 257
**Tests:** `lifecycle_test.go`

## Purpose
Explicit state machine (Stopped → Running → Reloading → Running → Stopped) for proxy listener pool lifecycle management.

## Key types / functions
- `State` (enum, line 23) — StateStopped, StateRunning, StateReloading
- `Status` (struct, line 56) — JSON wire shape (State, StateStr, Listeners count, RulesLoaded, StartedAt, LastError)
- `ListenerBuilder` (func type, line 70) — factory closure that produces fresh []*Listener on each call
- `Manager` (struct, line 75) — owns listener slice, state, startedAt, lastError, builder closure
- `Manager.Start()` (line 99) — builds listeners via builder, starts each, rolls back on any failure
- `Manager.Stop()` (line 148) — shutdown all listeners, transition to StateStopped
- `Manager.Reload()` (line 180) — Stop + Start in single lock-held critical section, transitions through StateReloading
- `StatusStringFor()` (line 40) — renders State enum to lowercase string ("stopped", "running", "reloading")

## Imports of note
None beyond stdlib sync, errors, fmt.

## Wired by
- `internal/api/proxy_lifecycle_handlers.go` (GET status, POST start/stop/reload endpoints)
- Server wiring code that constructs Manager with builder closure

## Used by
- Admin API for operator control of proxy subsystem
- Dashboard polling Status for real-time state display

## Notes
- State transitions are serialized via mu.Lock — concurrent Start/Stop/Reload calls wait (line 76).
- Builder invoked fresh on each Start/Reload — old listeners Shutdown'd and discarded (line 70).
- Reload on StateStopped acts as "start it" for idempotent admin semantics (line 193).
- On partial listener bind failure, rollback shuts down successfully-started ones (line 128).
- lastError preserved across state transitions so operators can inspect startup failures (line 140).
