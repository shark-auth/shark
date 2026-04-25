# harness.go

**Path:** `internal/testutil/cli/harness.go`
**Package:** `cli` (under `internal/testutil/cli`)
**LOC:** 152
**Tests:** `e2e_test.go`, `proxy_rules_e2e_test.go`

## Purpose
Test harness that spins up a real HTTP listener on an ephemeral port via `server.Build` so CLI subcommands and end-to-end flows can be exercised against a real running server. Intentionally separate from `internal/testutil` (which uses `httptest.NewServer`) — this one uses `net.Listen` so process-level shutdown signals stay realistic. Test helper — **not for production runtime.**

## Type
- `Harness` (line 24): `BaseURL`, `AdminKey`, internal `shutdown` cancel + `done` channel.

## Functions
- `Start(t)` (line 34) — picks an ephemeral port via `net.Listen`+close, builds the server with `DevMode=true` and a temp `dev.db`, launches `http.ListenAndServe` in a goroutine, waits for `/healthz` 200 within 3s, registers `t.Cleanup`.
- `Stop()` (line 106) — graceful shutdown, idempotent.
- `AdminRequest(method, path)` (line 118) — pre-stamped Bearer header.
- `Do(req)` (line 129) — runs against the default client, fails the test on transport error.
- `waitForHealth(baseURL, deadline)` (line 138) — 50ms poll loop on `/healthz`.

## Imports of note
- `net`, `net/http`, `testing`
- `cmd/shark/migrations` (embedded SQL set)
- `internal/server`

## Used by
- `internal/testutil/cli/e2e_test.go`
- `internal/testutil/cli/proxy_rules_e2e_test.go`

## Notes
- Test helper — **not for production runtime.**
- Picks a port by listening then closing — race-window for another process is tiny in practice and the loopback interface is single-tenant in CI.
- Reuses `server.Build` so the same wiring path that `cmd/shark serve` uses gets exercised, including `MigrationsFS` + admin-key bootstrap.
- 5s read/write timeouts on the http.Server keep flaky tests from hanging.
