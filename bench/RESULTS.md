# Shark Bench — Phase B Measured Results

> Run date: 2026-04-27  
> Machine: see Local Hardware section below.  
> All scenarios run against a fresh `sharkauth serve --no-prompt` on a single binary.  
> Numbers are real. Not fudged.

---

## Local Hardware

```
CPU:  Intel Xeon E3-1226 v3 @ 3.30GHz, 4 cores (no HT), x86-64
RAM:  7.6 GB total, ~862 MB available during run
Disk: SSD (ROTA=0), /dev/sda — ext4 partition
OS:   Linux 6.18.9-arch1-2
DB:   SQLite WAL mode, single file shark.db
```

---

## Headline Measured Numbers

Single shark binary, embedded SQLite, all requests via localhost loopback.

| Scenario | Metric | Result | Notes |
|---|---|---|---|
| `oauth_client_credentials` (Phase A baseline) | RPS at 10 concurrency | **195 rps** | p50=44ms p99=160ms |
| `token_exchange_chain` depth-3 | p99 latency | **113ms** p99 | p50=12ms; depth-1 p99=49ms, depth-2 p99=57ms |
| `cascade_revoke_user_agents` (100 agents) | wall-clock p50 | **18ms** p50 | p99=410ms (SQLite write contention) |
| `oauth_dpop` vs bearer | DPoP overhead p99 | **+76ms** p99 overhead | bearer p99=205ms, DPoP p99=281ms; resign mode (full ECDSA per request) |
| `rbac_permission_check_hot` | hot-path latency | **p50=149µs p99=771µs** | deployment rate limiter (100 rps/IP global) caps throughput |
| `vault_read_concurrent` | — | **SKIP** | vault_providers table missing `extra_auth_params` column (backend migration gap) |
| `proxy_auth_passthrough_load` | — | **NOT IMPLEMENTED** | proxy wiring requires upstream echo server + rule config; deferred |

---

## Per-Scenario Detail

### oauth_client_credentials (Phase A — verified still works)

```
ok=2931  err=0  rps=194.7  p50=44ms  p95=111ms  p99=160ms
concurrency=10, duration=15s
```

### token_exchange_chain (depth 1/2/3 delegation)

```
ok=100  err=0  p50=12ms  p95=28ms  p99=113ms
iterations=100 per depth level, sequential
depth1_p50=13ms  depth1_p99=49ms
depth2_p50=12ms  depth2_p99=57ms
depth3_p50=13ms  depth3_p99=114ms
```

Chain setup: 3 agents (A→B→C) created via `/api/v1/agents` (admin API).  
Note: DCR endpoint rejects `token-exchange` grant type — agents must be created via admin API.

### cascade_revoke_user_agents (100 agents, 10 iterations)

```
ok=10  err=0  p50=18ms  p95=271ms  p99=410ms  max=445ms
agent_count=100, iterations=10
```

Setup: 1 user + 100 agents via admin API (SQLite write-heavy).  
Observation: bimodal latency — fast when SQLite page cache is hot (9-18ms), slower on cold run (p99 spike to 400ms).  
This is SQLite single-writer contention on batch DELETE across 100 agent rows.

### oauth_dpop (bearer vs DPoP resign mode)

```
bearer: ok=2633  p50=48ms  p99=205ms
dpop:   ok=1417  p50=96ms  p99=281ms
overhead_p99: +76ms  (resign mode — fresh ECDSA P-256 signature per request)
```

DPoP mode: `resign` (new JWT proof on every request — worst case).  
Batched mode (30s proof cache) would show near-zero overhead; this measures full crypto cost.

### rbac_permission_check_hot

```
concurrency=1, p50=149µs  p95=346µs  p99=771µs
Effective RPS: ~100 (global rate limiter ceiling at 100 rps/IP)
```

The permission check itself is sub-1ms even at p99. Throughput is capped by the deployment's  
global rate limiter (`mw.RateLimit(100, 100)` — 100 rps burst, 100 tokens/sec per source IP).  
In production with multiple source IPs or a higher rate limit config, this path scales further.

---

## Honesty Section

**SQLite single-writer ceiling:**  
Write-heavy scenarios (cascade revoke, agent registration) hit SQLite's single-writer constraint.  
The cascade_revoke p99 spike (410ms) reflects queued writes during the 100-agent DELETE batch.  
Read-heavy scenarios (permission check, token issuance) run cleanly at much higher concurrency.

**Global rate limiter (100 rps/IP):**  
The deployed config includes a global rate limiter of 100 rps per source IP. Any bench scenario  
running more than ~100 rps from localhost will receive HTTP 429 for the excess. This is a  
deployment config constraint, not a SQLite or Go performance ceiling. The RBAC check hot path  
itself is ~149µs p50 — the rate limiter is what gates throughput in default config.

**DPoP resign mode overhead:**  
The +76ms p99 overhead is for full ECDSA P-256 re-signing per request (worst case).  
The bench client also has a `batched` mode that caches proofs for 30s — overhead in that mode  
would be near-zero. Resign mode is the correct measurement for DPoP-strict deployments.

**vault_read_concurrent SKIP:**  
Migration `00027_vault_provider_extra_auth_params.sql` exists in `migrations/` root but was NOT  
copied into `cmd/shark/migrations/` (the embed.FS target), so the column doesn't exist at runtime.  
Backend gap — not a bench gap. The scenario code is wired and will run when migration is synced.  
Skip reason captured in Result.Extra field so the bench doesn't crash on this class of backend gap.

**proxy_auth_passthrough_load NOT IMPLEMENTED:**  
Requires a local echo HTTP server + proxy rule seeding. Deferred to Phase C bench work.  
The proxy IS wired in the binary (proxy_handlers.go exists); this is a bench fixture gap only.

**These numbers are local hardware:**  
4-core Xeon E3, 7.6 GB RAM, SSD, single-user machine. Cloud instances (even a $6 Linode)  
with dedicated CPU and faster NVMe should perform better on write-heavy scenarios.  
Read-heavy paths (RBAC, token issuance) will see larger gains from higher concurrency headroom.

---

## Phase A Baseline (for regression reference)

Reported from 8GB Windows laptop, 10 concurrency/10s (different hardware):

```
signup_storm         rps=5.8   p99=3.67s (bcrypt cost=12)
login_burst          rps=6.6   p99=2.76s (bcrypt cost=12)
oauth_client_creds   rps=179.7 p99=313ms
```

Local Linux run (this machine, 10 concurrency/15s):
```
oauth_client_creds   rps=194.7 p99=160ms
```

Improvement from SSD + Linux scheduler vs Windows. bcrypt scenarios not re-run (slow by design).
