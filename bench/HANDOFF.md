# Bench Suite — Handoff (Phase B done 2026-04-27)

Phase A + Phase B (marquee scenarios) shipped. vault_read_concurrent and proxy_auth_passthrough skipped (backend gaps documented below).

## Phase B status

| Scenario | Status | Key number |
|---|---|---|
| `oauth_client_credentials` (Phase A) | ✓ verified | 195 rps, p99=160ms |
| `token_exchange_chain` | ✓ shipped | depth-3 p50=12ms p99=113ms |
| `cascade_revoke_user_agents` | ✓ shipped | 100 agents p50=18ms p99=410ms |
| `oauth_dpop` | ✓ shipped | +76ms overhead p99 (resign mode) |
| `rbac_permission_check_hot` | ✓ shipped | p50=149µs p99=771µs; rate-limited at 100 rps/IP |
| `vault_read_concurrent` | SKIP | backend bug: `extra_auth_params` column missing in vault_providers |
| `proxy_auth_passthrough_load` | NOT IMPL | requires echo upstream + proxy rule fixture; deferred |

See `bench/RESULTS.md` for full numbers with hardware context.

## Phase B implementation notes

- `token_exchange_chain` uses `/api/v1/agents` (admin API) not DCR — DCR rejects `token-exchange` grant type.
- `rbac_permission_check_hot` uses `runID`-suffixed resource to avoid conflict across runs.
- `vault_read_concurrent` soft-skips with `skipReason` in `Extra` when setup fails (graceful degradation pattern).
- Global rate limiter (`mw.RateLimit(100, 100)`) limits all scenarios to ~100 rps/IP from localhost. This is an honest deployment constraint documented in RESULTS.md.

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
| B-remaining | vault_read_concurrent (blocked on backend migration gap) | unblock when `extra_auth_params` column lands |
| B-remaining | proxy_auth_passthrough_load (bench fixture needed: echo upstream server) | ~1h |
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
