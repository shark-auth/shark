# lockout.go

**Path:** `internal/auth/lockout.go`
**Package:** `auth`
**LOC:** 103
**Tests:** `lockout_test.go`

## Purpose
In-memory failed-login counter and account-lockout enforcement keyed by email.

## Key types / functions
- `LockoutManager` (type, line 9) — mutex-guarded `map[string]*lockoutEntry`, `maxFails`, `lockoutDur`.
- `lockoutEntry` (type, line 16) — failure count, last-failure timestamp, locked-until timestamp.
- `NewLockoutManager` (func, line 25) — constructor; spawns 10-minute cleanup goroutine (line 33).
- `IsLocked` (func, line 45) — true while `time.Now().Before(lockedUntil)`.
- `RecordFailure` (func, line 60) — increments counter; resets if previous lockout expired; locks when `failures >= maxFails`; returns true if it just locked.
- `RecordSuccess` (func, line 87) — clears the entry on successful login.
- `cleanup` (func, line 93) — drops entries with last failure >1h old AND not currently locked.

## Imports of note
- `sync` — `sync.Mutex`.
- `time` — clock-only.

## Used by
- `internal/api/auth_handlers.go` — login path (check before, record after).
- `internal/server/server.go` — wired at startup.

## Notes
- **PROCESS-LOCAL** — see `SCALE.md`. Multi-instance deployments allow lockout bypass by hitting other replicas. Move to Redis/DB for HA.
- Cleanup goroutine is never stopped — fine for app lifetime, leak in tests if `NewLockoutManager` is called repeatedly.
- Email is the lockout key — does not differentiate by IP. Username enumeration mitigation lives elsewhere.
