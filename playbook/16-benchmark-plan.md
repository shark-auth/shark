# Shark Benchmark Suite — Implementation Plan

> **For agentic workers:** Black-box HTTP client. Single Go binary. Public-facing numbers. Manual run only, no CI.

**Goal:** Ship `bench/` — comprehensive benchmark suite measuring shark perf across 10 standard-user-flow scenarios, producing a public-shareable report (RPS, p50/p95/p99, cascade speed, DB contention, memory).

**Architecture:** Single Go binary, pure HTTP client (no `internal/` imports), HTTP/2 multiplex via `golang.org/x/net/http2`, real DPoP proof signing (ECDSA P-256) with realistic batched-reuse mode, sqlite-backed shark target.

**Tech stack:** Go 1.23, `net/http`, `golang.org/x/sync/errgroup`, `github.com/HdrHistogram/hdrhistogram-go` (or stdlib if avoidable), ASCII bar chart renderer (no chart libs).

---

## Decisions locked

| Q | A |
|---|---|
| Target hardware | 8GB Windows laptop. Numbers documented w/ machine specs. "If it runs here, flies on Railway." |
| Database | sqlite only. Postgres support absent — not benchmarked. |
| Comparison baseline | None. Standalone shark numbers. |
| Audience | Public-facing — HN post + README. Polish accordingly. |
| CI regression threshold | 15% p99 regression OR 20% throughput drop fails `--baseline=baseline.json` diff. (Not run on CI; threshold for manual diff only.) |
| Cascade scale cap | 100-500 agents per user. Standard flows, not 10k stress. |
| DPoP mode | Realistic batched proof reuse (sign once per `iat` window). Re-sign per-request mode optional via flag for comparison. |
| Repo location | `bench/` at root. |
| Run policy | Manual only. `go run ./bench` or `make bench`. Never on CI. |
| Coupling | Black-box. HTTP only. No imports from `internal/`. |

---

## File structure

```
bench/
  README.md                          # how to run, machine specs, sample output
  cmd/bench/main.go                  # entry point + flag parsing
  internal/
    client/                          # HTTP client w/ DPoP + connection pool
      client.go
      dpop.go                        # ECDSA P-256 prover, batched-reuse mode
    metrics/
      histogram.go                   # p50/p95/p99 + counters
      probe.go                       # GOMAXPROCS, mem, goroutines, db file size
    fixtures/
      seed.go                        # bulk-load N users / agents / vaults
    scenario/
      scenario.go                    # interface
      signup_storm.go
      login_burst.go
      oauth_client_credentials.go
      oauth_dpop.go
      token_exchange_chain.go
      vault_read_concurrent.go
      cascade_revoke_layer3.go
      cascade_revoke_layer5.go
      audit_emission.go
      mixed_realistic.go
    output/
      markdown.go                    # human report w/ ASCII bar charts
      json.go                        # baseline / regression diff
      console.go                     # live progress
    workload/
      mix.go                         # blend definitions (smoke / marketing / stress)
  REPORT.md                          # generated, gitignored before first run
  baseline.json                      # generated, committed for regression diff
```

---

## Profiles

| Profile | Concurrency | Duration | Scenarios | Use |
|---|---|---|---|---|
| `--profile smoke` | 10 | 30s each | mixed_realistic only | local sanity check |
| `--profile marketing` | 200 | 60-120s each | all 10 | numbers for HN post |
| `--profile stress` | 500 | 5 min each | mixed_realistic + cascade | find breaking point |
| `--profile cascade` | 1 | runs to completion | cascade L3 + L5 only | demo cascade speed |

---

## Scenarios

| # | Name | What it measures | Output |
|---|---|---|---|
| 1 | signup_storm | bcrypt cost + DB write throughput | RPS, p99, SQLITE_BUSY count |
| 2 | login_burst | bcrypt verify + session create | RPS, p99 |
| 3 | oauth_client_credentials | `/oauth/token` hot path, no DPoP | RPS, p99, JWT sign cost |
| 4 | oauth_dpop | same w/ DPoP proof, batched reuse | RPS, p99, DPoP overhead vs bearer |
| 5 | token_exchange_chain | depth 1, 2, 3 | latency vs depth |
| 6 | vault_read_concurrent | jkt-bound retrieval, 100 parallel | RPS, p99, decryption cost |
| 7 | cascade_revoke_layer3 | revoke user w/ 50 + 100 agents | ms to complete cascade |
| 8 | cascade_revoke_layer5 | disconnect vault w/ 50 retrievals | ms to complete cascade |
| 9 | audit_emission | 10k events written | RPS, lag from action → audit visible |
| 10 | mixed_realistic | weighted blend (35% token issuance, 25% vault read, 20% login, 10% signup, 5% exchange, 5% cascade) | sustained RPS over 60s, p99 |

---

## Marketing-ready hypothesis to validate

These go into the public report if they hold. If not — they're the "catch problems" half, file P1s.

- 1000 client_credentials tokens/sec sustained on 8GB laptop
- p99 < 100ms for `/oauth/token` at 100 sustained RPS
- Cascade-revoke 100 agents in < 1 second
- Vault retrieval p99 < 50ms at 100 concurrent
- DPoP overhead < 3ms p99 vs bearer
- Audit emission overhead < 5% throughput cost
- Token-exchange depth-3 < 30ms p99 vs depth-1 ~10ms

---

## Output deliverables

1. **`bench/REPORT.md`** — auto-generated, human-readable, ASCII bar charts. Top section: machine specs + git SHA + total runtime. Per-scenario: throughput table, latency table, ASCII histogram of p50→p99 distribution.
2. **`bench/baseline.json`** — committed JSON. `bench --profile marketing --update-baseline` rewrites it. `bench --baseline=baseline.json` diffs current run vs baseline, fails non-zero if any p99 regression > 15% OR throughput drop > 20%.
3. **Console summary** — live progress dots, final summary table.

---

## Phases

### Phase A: Skeleton + harness + 3 scenarios (sonnet subagent #1, worktree)

- [ ] `bench/cmd/bench/main.go` — flag parsing (--profile, --baseline, --output, --target-url, --duration, --concurrency)
- [ ] `bench/internal/client/client.go` — HTTP client with connection pool, retry on transient
- [ ] `bench/internal/client/dpop.go` — ECDSA P-256 prover, `batched` mode (default: re-sign once per 30s window), `resign` mode for comparison
- [ ] `bench/internal/metrics/histogram.go` — p50/p95/p99 + counters (use `hdrhistogram-go` or stdlib min-heap if dep cost too high)
- [ ] `bench/internal/scenario/scenario.go` — interface `Run(ctx, client, opts) Result`
- [ ] Scenarios 1, 2, 3 (signup_storm, login_burst, oauth_client_credentials)
- [ ] `bench/internal/output/console.go` — live progress + summary
- [ ] Boot shark, `--profile smoke` runs end-to-end, prints summary

### Phase B: 4 more scenarios (sonnet subagent #2, worktree)

- [ ] Scenario 4 (oauth_dpop) — measure overhead vs bearer
- [ ] Scenario 5 (token_exchange_chain) — depth 1, 2, 3
- [ ] Scenario 6 (vault_read_concurrent) — fixture: bulk-seed vault; measure parallel retrieval
- [ ] Scenario 7 (cascade_revoke_layer3) — fixture: 50 then 100 agents under one user; measure ms to complete
- [ ] `bench/internal/fixtures/seed.go` — bulk loader with progress bar

### Phase C: Last 3 scenarios + outputs (sonnet subagent #3, worktree)

- [ ] Scenario 8 (cascade_revoke_layer5) — fixture: vault w/ 50 retrievals, then disconnect
- [ ] Scenario 9 (audit_emission) — measure throughput + lag from commit to visible
- [ ] Scenario 10 (mixed_realistic) — weighted blend per workload mix table
- [ ] `bench/internal/output/markdown.go` — REPORT.md with ASCII bar charts
- [ ] `bench/internal/output/json.go` — baseline writer + diff reader
- [ ] `bench/internal/metrics/probe.go` — runtime + DB probes (sqlite file size, WAL size, SQLITE_BUSY count via `pragma wal_checkpoint`)

### Phase D: Polish + marketing run + README (orchestrator)

- [ ] Run `--profile marketing` on this laptop, capture full report
- [ ] Author `bench/README.md` with: machine specs, how to run, sample numbers, hypothesis-validation table (which targets hit, which missed)
- [ ] Write commit + push

---

## Dispatch sequencing

```
Phase A ──> Phase B ──> Phase C ──> Phase D (orchestrator + marketing run)
```

All subagent phases run in `worktree` isolation. Backend frozen — bench only HTTP-pokes the running shark. Sonnet subagents may NOT modify `internal/`, `cmd/`, `admin/`, `tests/`, `sdk/`, or `tools/`.

---

## Risks + mitigations

- **SQLITE_BUSY storms tank throughput** — known: single-writer-only even under WAL. Document in report as architectural fact, not regression. Bench captures and reports BUSY count per scenario.
- **DPoP proof generation CPU-bound** — batched-reuse mode is the realistic prod pattern; report both batched and resign numbers so readers see the worst case too.
- **Marketing numbers are laptop-specific** — top-of-report machine specs block. Numbers document as "on M-class hardware, under load X."
- **bcrypt cost dominates auth scenarios** — true. Report bcrypt cost as a fact ("auth latency floor is ~150ms p99 because of bcrypt cost=12; lowering cost trades CPU for security; this is intentional").
- **Cascade benchmark seed time** — capped at 100 agents max per fixtures.seed.go to keep total bench under 30 min.
- **Backend frozen** — concurrent agent owns `internal/`. Bench is HTTP-only black-box. If bench discovers a backend bug, it goes in REPORT.md "issues found" section, not patched.
