# Bench Suite — Handoff (deferred 2026-04-27)

Phase A landed. Phases B/C/D deferred — pivoted to SDK gap investigation for launch.

## What's done (Phase A)

`bench/` is a self-contained Go module (separate `go.mod`) — black-box HTTP client, no `internal/` imports.

**Files (1207 LOC):**
- `bench/cmd/bench/main.go` — CLI flags + scenario orchestration
- `bench/internal/client/client.go` — pooled HTTP client w/ retry + DPoP injection
- `bench/internal/client/dpop.go` — ECDSA P-256 prover, batched (30s cache) + resign modes, JKT thumbprint
- `bench/internal/metrics/histogram.go` — sorted-slice histogram, p50/p95/p99 + Counter
- `bench/internal/scenario/scenario.go` — `Scenario` interface + `Opts` + `Result`
- `bench/internal/scenario/common.go` — `runWorkers`, `finalize`, unique-email helper
- `bench/internal/scenario/signup_storm.go` — random-email signup, db_size_mb extra
- `bench/internal/scenario/login_burst.go` — pre-seeds 100 users, hammers `/api/v1/auth/login`
- `bench/internal/scenario/oauth_client_credentials.go` — DCR-registers, hammers `/oauth/token`
- `bench/internal/output/console.go` — progress + summary table
- `bench/internal/fixtures/fixtures.go` — `Bundle` for users/clients

## How to run today

```bash
cd bench
go run ./cmd/bench --profile smoke --admin-key-file ../tests/smoke/data/admin.key.firstboot
```

Output: console summary table (no `REPORT.md` writer yet — Phase C).

## Smoke run today (8GB Win laptop, 10 concurrency / 10s)

```
[1/3] signup_storm ............... ok=60   err=0  rps=5.8   p50=1.69s p95=2.53s p99=3.67s
[2/3] login_burst ................ ok=70   err=0  rps=6.6   p50=1.31s p95=2.48s p99=2.76s
[3/3] oauth_client_credentials ... ok=1830 err=0  rps=179.7 p50=38ms  p95=162ms p99=313ms
```

bcrypt cost=12 dominates signup/login p99. Token issuance hot path is healthy.

## What's NOT done (resume here)

| Phase | Scope | Estimated CC |
|---|---|---|
| B | scenarios 4-7: oauth_dpop, token_exchange_chain, vault_read_concurrent, cascade_revoke_layer3 | ~2h |
| C | scenarios 8-10 + REPORT.md markdown writer + baseline.json regression diff + DB probe (sqlite WAL/BUSY) | ~2h |
| D | marketing run on this laptop, write `bench/README.md` w/ machine specs + hypothesis-validation table, polish | ~1h |

**Spec:** `playbook/16-benchmark-plan.md` — full scenario list, profile defs, hypothesis to validate, marketing-ready stat targets.

## Caveats for resumer

- Dispatch each phase as its own sonnet subagent in worktree, `bench/` is the only writable surface (backend frozen).
- `bench/` is separate Go module — don't try to import `internal/` from shark.
- Profile defaults in `main.go`: smoke=10 conc / 30s ; marketing=200 conc / 60-120s ; stress=500 conc / 5min ; cascade=1 conc / runs-to-completion.
- LSP warnings about "not in workspace module" are spurious — `bench/` has its own go.mod and builds clean. Add `go.work` at repo root only if needed.
- Backend bug surfaced during Phase A run: bcrypt cost=12 explains auth latency floor. Document as architectural fact in REPORT.md, not a regression.

## Why deferred

Pivoted to SDK gap investigation 2026-04-27. Launch concerns: do `shark_auth` Python SDK + future TS SDK cover the full feature surface (human auth + agents + vault + admin), or are there gaps customers will hit during integration? See playbook/17 (incoming) for that report.

Resume bench post-launch alongside compliance polish + acr/amr backend hook.
