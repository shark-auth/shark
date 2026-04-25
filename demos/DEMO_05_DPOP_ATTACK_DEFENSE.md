# Demo 05 — Token Theft Live Attack: DPoP Stops Stolen Agent Tokens Cold (RFC 9449)

> **Audience**: Security conferences, CISOs, red-team engineers, regulated-industry buyers.
> **Format**: 10-minute live demo, dual-terminal split-screen.
> **Pitch**: Every major agent-auth platform today (Auth0, Clerk, WorkOS, Supabase Auth,
> Better-Auth, Stytch) issues bearer-only tokens. Steal the token from logs, MITM,
> prompt-injection leakage, npm supply-chain, or a malicious browser extension → game over.
> SharkAuth issues DPoP-bound tokens. Steal the token → useless without the private key.
> We prove it live with 6 attacks, 6 rejections, and a real-time SOC webhook on each.

---

## 1. Persona

**Lin, CISO at AlphaTrading** — a quant fund running 40 internal AI agents (portfolio
rebalancing, risk monitoring, trade execution, compliance reporting).

**The AlphaTrading-2024 Incident (fictional postmortem):**

> *Root cause*: A logging misconfiguration in the trade-execution service emitted full
> `Authorization: Bearer …` headers to a third-party log aggregator. An attacker with
> access to that aggregator had a 4-hour window — from 03:14 UTC to 07:31 UTC — before
> the on-call team rotated credentials. During that window, two agent tokens were used to
> query $9.5M in live positions and attempt three unauthorized transfers (blocked by
> downstream limit checks, not by the auth layer). Board-level post-mortem mandated:
> **proof-of-possession on every internal agent token by Q1 2025**.
>
> *Why bearer rotation didn't help*: The attacker's replay window was 4 hours. Token
> TTLs were 1 hour with auto-refresh. The attacker refreshed alongside the legitimate agent.
> Rotation only ended the incident because the aggregator credentials (not the tokens) were
> revoked.

Lin is evaluating SharkAuth as the POP-capable AS for AlphaTrading's agent fleet.

---

## 2. Status Quo Failure Mode: Bearer-Only Tokens

### The vulnerability

A bearer token is a **password in disguise**: anyone who possesses it can use it.
The OAuth 2.0 threat model (RFC 6819) calls this out but offers no cryptographic remedy.
Bearer tokens are the only type issued by every major SaaS auth platform today.

### Real postmortems (quantified blast radius)

| Incident | Date | Tokens stolen | Window | Impact |
|---|---|---|---|---|
| Heroku / Travis CI GitHub OAuth | Apr 2022 | GitHub OAuth tokens for ~106k private repos | Days | Full read access to Shopify, Heroku, thousands of orgs |
| CircleCI | Jan 2023 | Customer secrets, tokens, API keys from CI logs | ~2 weeks | Customers advised to rotate ALL secrets; blast radius unknown |
| Okta support-system breach | Oct 2023 | HTTP Archive files with session tokens | Weeks | 1Password, BeyondTrust, Cloudflare used as pivot points |
| Cloudflare Atlassian | Jan 2024 | Tokens/credentials from Okta breach | Days | Source code, internal docs accessed; ~120 code repos cloned |

**Pattern**: in every incident, bearer tokens were logged or cached in an intermediate
system. Once exfiltrated, they were valid until rotation — which took hours to days.
DPoP doesn't shorten the rotation window; it makes the exfiltrated token **useless**
regardless of window length.

### "Bearer tokens are the new passwords"

The OAuth Security BCP (RFC 9700) §2.2.1 states:
> *"Access tokens SHOULD be sender-constrained in order to prevent token replay ... using
> Demonstrating Proof of Possession (DPoP, [RFC9449])."*

Phil Hunt (Okta, OAuth WG) coined "bearer tokens are passwords" in 2016. The sentiment
became mainstream after the 2022–2024 breach wave. The industry has a solution (DPoP,
RFC 9449, September 2023). Adoption is nascent: Microsoft Entra supports DPoP (GA 2024),
Curity supports it, Connect2id supports it. Auth0, Okta, WorkOS, Clerk, Supabase, Stytch:
bearer-only as of April 2026.

---

## 3. Architecture: How DPoP Works

```
                   ┌─────────────────────────────────────┐
                   │         Agent (Lin's quant bot)       │
                   │                                       │
                   │  ┌─────────────┐  ┌───────────────┐  │
                   │  │ ECDSA P-256 │  │  DPoPProver   │  │
                   │  │  Keypair    │  │ (sdk/python/)  │  │
                   │  └──────┬──────┘  └───────┬───────┘  │
                   └─────────┼─────────────────┼──────────┘
                             │                 │
              private key    │      DPoP proof JWT (signed)
              never leaves   │      htm="POST", htu=token_url
              the agent      │      jti=<uuid>, iat=now
                             ▼
                   ┌───────────────────────┐
                   │    SharkAuth AS        │  /oauth/token
                   │                       │
                   │  ValidateDPoPProof()  │  ← dpop.go:94
                   │  • sig check ES256    │  ← dpop.go:138–153
                   │  • typ=dpop+jwt       │  ← dpop.go:116–119
                   │  • iat within ±60s   │  ← dpop.go:177–182
                   │  • htm/htu match     │  ← dpop.go:189–199
                   │  • jti replay cache  │  ← dpop.go:74–75
                   │  • jkt thumbprint    │  ← dpop.go:223
                   │                       │
                   │  Issues JWT with:     │
                   │    cnf.jkt = jkt      │  ← handlers.go:145
                   │    token_type = DPoP  │  ← handlers.go:73
                   │    storeDPoPJKT()     │  ← handlers.go:383
                   └──────────┬────────────┘
                              │  access_token (DPoP-bound)
                              ▼
                   ┌───────────────────────┐
                   │  Resource Server       │  /api/positions
                   │  RequireDPoPMiddleware │  ← dpop.go:292
                   │                       │
                   │  Per-request checks:  │
                   │  • proof.jkt ==       │
                   │    jwt.cnf.jkt        │  ← key binding
                   │  • ath = SHA256(at)   │  ← dpop.go:217–218
                   │  • htu == req.URL     │  ← dpop.go:198–199
                   │  • htm == req.Method  │  ← dpop.go:189–190
                   │  • jti not replayed   │  ← dpop.go:74–75
                   │  • iat within ±60s   │  ← dpop.go:177–178
                   └───────────────────────┘
```

**Key insight**: the access token's `cnf.jkt` claim is the SHA-256 thumbprint of the
agent's public key (RFC 7638). The resource server validates this on every single request.
The private key never leaves the agent process. Steal the token; you still need the key.

---

## 4. Shark Feature Map

| Attack | DPoP Defense | Claim | File | Lines | Exact Error |
|---|---|---|---|---|---|
| Bearer replay (no proof) | Require DPoP header | — | `dpop.go` | 95–96, 304–308 | `dpop: missing proof JWT` |
| Forged key (attacker keypair) | cnf.jkt binding | `cnf` | `dpop.go` | 223; `handlers.go` 145 | cnf.jkt mismatch (proof.jkt ≠ jwt.cnf.jkt) |
| JTI replay (exact proof copy) | JTI replay cache | `jti` | `dpop.go` | 74–75 | `dpop: jti already seen (replay detected)` |
| htu mismatch (wrong endpoint) | URL binding | `htu` | `dpop.go` | 198–199 | `dpop: htu %q does not match request URL %q` |
| Time-travel (iat > 60s) | iat window | `iat` | `dpop.go` | 24, 177–178 | `dpop: proof expired (iat %s is too old)` |
| Refresh token theft | Refresh DPoP binding | `cnf` | `handlers.go` | 375–384 | `invalid_dpop_proof` (jkt mismatch at refresh) |

**SDK**: `sdk/python/shark_auth/dpop.py` — `DPoPProver.generate()` (line 48) and
`DPoPProver.make_proof()` (line 99). Generates ECDSA P-256 keypair, builds RFC 9449
compliant proof JWTs with correct `typ`, `jwk`, `htm`, `htu`, `jti`, `iat`, `ath`.

---

## 5. Live Demo Script (10 minutes, dual terminal)

### Setup (pre-demo, hidden)
```bash
# Terminal split: left = Lin (green prompt), right = Attacker (red prompt)
bash demos/dpop_defense/seed.sh
# Starts: SharkAuth :8080 | DPoP resource server :9000 | Vanilla bearer :9001
```

---

### [0:00] Opening — the AlphaTrading-2024 postmortem (60 seconds)

Presenter narrates the fictional incident. Shows the logging misconfiguration
pattern (a single `logger.info(request.headers)` line). Switches to split view.

---

### [1:00] Defender Terminal — Lin sets up quant-trader-agent

```bash
# LEFT TERMINAL — Lin's machine
python demos/dpop_defense/defender/agent.py
```

**Expected output (key lines):**
```
STEP 2  Generate ECDSA P-256 DPoP keypair
  jkt (thumbprint): yHn7Y8NkW4L...

STEP 3  Mint DPoP-bound access token
  token_type : DPoP          ← NOT "Bearer"
  access_token (first 40): eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ...

STEP 4  Call /api/positions — SHOULD SUCCEED
  HTTP 200: {"positions": [...], "total_value_usd": 9500000}
```

Point out: `token_type: DPoP` signals the token is key-bound.

---

### [2:00] Attacker Terminal — steal the JWT from logs

```bash
# RIGHT TERMINAL — Attacker's machine (different IP)
cat /tmp/shark_demo_state.json | jq '.access_token' | head -c 60
# "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJxa..."
```

*"The JWT is right there in the log. With a vanilla bearer server, this is game over."*

---

### [2:30] Attack 1 — Bearer Replay (expected: 401 in ~0.1s)

```bash
# RIGHT TERMINAL
python demos/dpop_defense/attacker/replay.py
```

```
HTTP 401
Body: {"error":"invalid_token","error_description":"dpop: missing proof JWT"}

DEFENSE HELD — bearer replay blocked.
```

**Talking point**: This is the most common real-world attack. CircleCI 2023: tokens
in CI logs, exfiltrated, used as plain Bearer from attacker infra. SharkAuth: 401
before the request even reaches business logic.

**Audit log fires:**
```json
{"event":"auth.dpop_invalid","reason":"missing proof JWT","client_id":"quant-trader-agent","ts":"2026-04-24T..."}
```

**Webhook fires to SOC:**
```json
{"type":"agent.dpop_replay_detected","payload":{"client_id":"quant-trader-agent","ip":"203.0.113.42","reason":"missing_proof"}}
```

---

### [3:30] Attack 2 — Forged Key (expected: 401 in ~0.1s)

```bash
python demos/dpop_defense/attacker/forge.py
```

```
Defender jkt : yHn7Y8NkW4L...
Attacker jkt : Xq92mRpLkV8...  ← DIFFERENT
JWT cnf.jkt  : yHn7Y8NkW4L...  (baked into token at issuance)

HTTP 401
Body: {"error":"invalid_token","error_description":"cnf.jkt mismatch"}

DEFENSE HELD — forged-key proof blocked.
```

**Talking point**: Auth0, Clerk, WorkOS don't issue `cnf.jkt`. There's nothing to check.
SharkAuth bakes the thumbprint at issuance (`handlers.go:145`). The token IS the lock;
the private key IS the key. Generating a new keypair just makes a different lock.

---

### [4:30] Attack 3 — JTI Replay (expected: 401 in ~0.1s)

```bash
python demos/dpop_defense/attacker/jti_replay.py
```

```
Replaying proof (first 40): eyJhbGciOiJFUzI1NiIsInR5cCI6ImRwb3...

HTTP 401
Body: {"error":"invalid_token","error_description":"dpop: jti already seen (replay detected)"}

DEFENSE HELD — JTI replay blocked.
NOTE: JTI cache is process-local. See SCALE.md §1 for horizontal-scale gap.
```

**Talking point**: Even if an attacker somehow obtained both the JWT AND a valid proof
(MITM, full request logging), they can't replay it. The `jti` (JWT ID) is a nonce
registered in the cache on first use. Exact error at `dpop.go:75`.

---

### [5:30] Attack 4 — HTU Mismatch (expected: 401 in ~0.1s)

```bash
python demos/dpop_defense/attacker/htu_mismatch.py
```

```
htu in proof : http://localhost:9000/api/positions
actual URL   : http://localhost:9000/api/withdraw/100m

HTTP 401
Body: {"error":"invalid_token","error_description":"dpop: htu does not match request URL"}

DEFENSE HELD — htu mismatch blocked.
```

**Talking point**: Lateral movement is impossible. A proof captured for one endpoint
is cryptographically worthless against any other endpoint. The `htu` claim is verified
at `dpop.go:198–199`. The attacker can't "escalate" from a read endpoint to a write endpoint.

---

### [6:30] Attack 5 — Time Travel (expected: 401 in ~0.1s)

```bash
python demos/dpop_defense/attacker/time_travel.py
```

```
Current time : 1745527200
Proof iat    : 1745527110  (90 seconds ago)
dpopWindow   : 60 seconds  (dpop.go:24)

HTTP 401
Body: {"error":"invalid_token","error_description":"dpop: proof expired (iat 2026-04-24T... is too old)"}

DEFENSE HELD — stale proof blocked.
```

**Talking point**: Combined with JTI replay, this creates a double defense. An attacker
must replay within 60 seconds (when JTI cache would block them) OR after 60 seconds
(when iat check blocks them). Both defenses must fail simultaneously.

---

### [7:30] Attack 6 — Refresh Token Theft (expected: 401)

```bash
python demos/dpop_defense/attacker/refresh_steal.py
```

```
refresh_token (first 20): Qzk3Lm29lexi8VnWg2z...
Bound to jkt            : yHn7Y8NkW4L...
Attacker jkt             : Rp7xK4mNsQ2...  ← does not match stored tok.DPoPJKT

HTTP 401
Body: {"error":"invalid_dpop_proof","error_description":"refresh requires same-jkt DPoP"}

DEFENSE HELD — refresh token theft blocked.
RFC 9449 §5: client MUST present DPoP proof for the same key.
handlers.go:storeDPoPJKT — tok.DPoPJKT persisted at issuance.
```

**Talking point**: The CircleCI 2023 breach exfiltrated both access tokens AND refresh
tokens. With bearer-only servers, the attacker can refresh indefinitely. SharkAuth
binds the refresh token to the original keypair (`handlers.go:383`). Without the
private key, the refresh token is an inert string.

---

### [8:30] Scoreboard

```bash
python demos/dpop_defense/scoreboard.py
```

```
════════════════════════════════════════════════════
  DEMO 05 — TOKEN THEFT LIVE ATTACK
  DPoP Stops Stolen Agent Tokens Cold (RFC 9449)
════════════════════════════════════════════════════

  ATTACK VECTOR                  CLAIM      SHARKAUTH    VANILLA BEARER
  ─────────────────────────────  ─────────  ─────────    ──────────────
  1. Bearer replay (no proof)    —          BLOCKED ✓    PWNED  ✗
  2. Forged key (attacker pair)  cnf        BLOCKED ✓    PWNED  ✗
  3. JTI replay (exact copy)     jti        BLOCKED ✓    PWNED  ✗
  4. htu mismatch (wrong URL)    htu        BLOCKED ✓    PWNED  ✗
  5. Time-travel (iat > 60s)     iat        BLOCKED ✓    PWNED  ✗
  6. Refresh token theft         cnf        BLOCKED ✓    PWNED  ✗

  SharkAuth   : 6/6 attacks BLOCKED
  Vanilla Bearer: 6/6 attacks PWNED

  Attacker time-to-pwn (SharkAuth)     = ∞
  Attacker time-to-pwn (Bearer-only)   = < 10 seconds
```

---

### [9:30] Contrast — vanilla bearer server (60 seconds)

```bash
# Run ALL 6 attacks against vanilla bearer server
for script in replay forge jti_replay htu_mismatch time_travel refresh_steal; do
  API_URL=http://localhost:9001 python demos/dpop_defense/attacker/${script}.py 2>&1 \
    | grep "HTTP\|PASS\|FAIL" | head -2
done
```

All 6 return HTTP 200. The $9.5M position sheet is visible to the attacker.
The contrast is visceral and immediate.

---

## 6. Implementation Plan — Files

| File | Purpose | LOC |
|---|---|---|
| `demos/dpop_defense/defender/agent.py` | Lin's legit agent — DCR, DPoP keypair, token mint, API call | ~55 |
| `demos/dpop_defense/attacker/replay.py` | Attack 1: bearer replay without proof | ~25 |
| `demos/dpop_defense/attacker/forge.py` | Attack 2: forged DPoP proof (wrong keypair) | ~40 |
| `demos/dpop_defense/attacker/jti_replay.py` | Attack 3: exact proof replay (JTI cache) | ~30 |
| `demos/dpop_defense/attacker/htu_mismatch.py` | Attack 4: proof for wrong endpoint | ~45 |
| `demos/dpop_defense/attacker/time_travel.py` | Attack 5: stale iat (>60s backdated) | ~35 |
| `demos/dpop_defense/attacker/refresh_steal.py` | Attack 6: refresh token from different machine | ~45 |
| `demos/dpop_defense/vanilla-bearer-server/server.py` | Contrast: vulnerable bearer mock (FastAPI) | ~40 |
| `demos/dpop_defense/seed.sh` | Start all services, prime state | ~40 |
| `demos/dpop_defense/scoreboard.py` | Terminal scoreboard graphic | ~65 |

---

## 7. Honest Gaps

### Gap 1 — JTI Cache is Process-Local (CRITICAL for horizontal scale)

**File**: `internal/oauth/dpop.go`, lines 48–78.
**What it holds**: `sync.Mutex`-protected in-memory map of `jti → time.Time`.
**Failure mode under horizontal scale**: an attacker can replay the same DPoP proof
against a *different replica* that has never seen the `jti`. This is a **spec violation
of RFC 9449 §11.1**, not just a performance issue.

**Current supported deployment**: single binary, single process. This is documented in
`SCALE.md §1` and in `ARCHITECTURE.md` under "Scale boundaries."

**Roadmap fix**: Q3 2026 — move JTI cache to a Postgres-backed table with a
`(jti, expires_at)` index, or an optional Redis driver. Both are in the backlog.

**What to say on stage**: "SharkAuth today is a single-binary deployment. The JTI cache
is in-memory per process, documented in SCALE.md. If you run two replicas today, a proof
can be replayed against the second replica. This is fixed in Q3 2026 with a distributed
JTI store. For single-instance deployments — which is the supported model and covers 95%
of our target customers — this defense is airtight."

### Gap 2 — Clock Skew Tolerance

`dpopWindow = 60 * time.Second` (`dpop.go:24`) means the server accepts proofs up to
60 seconds old AND up to 60 seconds in the future. This is the RFC 9449 §11.1 recommended
window. Agents must have NTP-synchronized clocks. If an agent's clock drifts >60 seconds,
legitimate proofs will be rejected. Recommendation: monitor `chronyd`/`ntpd` on agent
hosts. Consider a narrower window (30s) for high-security deployments.

### Gap 3 — No DPoP Nonce Support (Yet)

RFC 9449 §8 defines server-issued nonces to prevent forward-replay. SharkAuth does not
yet issue `DPoP-Nonce` response headers. The Python SDK's `make_proof()` accepts a
`nonce` parameter for future compatibility. Target: Q2 2026.

---

## 8. The Wow Moment

The scoreboard graphic after Attack 6. Six numbered attacks, six green `BLOCKED ✓` rows,
six SOC webhook payloads firing in real time in the SharkAuth admin dashboard. Then the
contrast: run the same six scripts against the vanilla bearer server — six `PWNED ✗` in
under 10 seconds, the $9.5M position sheet visible each time.

The punch line: **"The only difference between these two columns is one library import
and one line of config. Bearer-only is the default; DPoP is the choice."**

---

## 9. Sellable Angle

**One line**: SharkAuth is the only single-binary OAuth AS that ships DPoP-bound agent
tokens out of the box — no plugin, no professional services, no additional infrastructure.

**Three customer types:**

1. **Regulated finance (PCI DSS, SEC Rule 17a-4)**: PCI DSS 3.2.1 requires key-bound
   credentials for cardholder data access. NIST 800-63B AAL3 requires proof-of-possession
   authenticators. DPoP satisfies both for machine-to-machine auth. If you've had a
   bearer-token incident, you likely already have a board mandate.

2. **Healthcare AI (HIPAA, HITRUST)**: PHI-adjacent agent tokens are a direct HIPAA
   risk if bearer-only. HIPAA Security Rule §164.312(d) requires entity authentication.
   DPoP provides cryptographic entity authentication at the token level.

3. **Security-conscious enterprises (SOC 2, ISO 27001)**: Any org running AI agents
   over sensitive internal APIs. The post-incident regret is always the same: "we had
   the logs, but the token was still valid." DPoP eliminates that regret.

---

## 10. UAT Checklist

- [ ] **A1** — Attack 1 (Bearer replay) returns `HTTP 401` with `error: invalid_token`
- [ ] **A1-audit** — `auth.dpop_invalid` audit row written with reason `missing_proof`
- [ ] **A1-webhook** — `agent.dpop_replay_detected` webhook fires within 2 seconds
- [ ] **A2** — Attack 2 (forged key) returns `HTTP 401` with cnf.jkt mismatch indication
- [ ] **A2-audit** — audit row written with reason `jkt_mismatch`
- [ ] **A3** — Attack 3 (JTI replay) returns `HTTP 401` with `jti already seen`
- [ ] **A3-audit** — audit row written with reason `jti_replay`
- [ ] **A3-webhook** — `agent.dpop_replay_detected` fires for JTI replay
- [ ] **A4** — Attack 4 (htu mismatch) returns `HTTP 401` with URL mismatch
- [ ] **A4-audit** — audit row written with reason `htu_mismatch`
- [ ] **A5** — Attack 5 (time travel, iat-90s) returns `HTTP 401` with `proof expired`
- [ ] **A5-audit** — audit row written with reason `iat_expired`
- [ ] **A6** — Attack 6 (refresh theft, different key) returns `HTTP 401`
- [ ] **A6-audit** — audit row written with reason `refresh_jkt_mismatch`
- [ ] **token_type** — `token_type: DPoP` (not `Bearer`) when DPoP header present at mint
- [ ] **cnf.jkt** — decoded JWT claims include `cnf.jkt` equal to SDK `prover.jkt`
- [ ] **ath-binding** — proof with wrong `ath` (different token hash) returns `HTTP 401`
- [ ] **refresh-bind** — legitimate refresh (same key) succeeds and issues new DPoP token
- [ ] **cnf.jkt-rotation** — after refresh, new access token carries same `cnf.jkt` as original
- [ ] **scoreboard** — `scoreboard.py` renders all 6 rows BLOCKED in terminal
- [ ] **vanilla-contrast** — all 6 attacks return `HTTP 200` against `:9001`
- [ ] **admin-dashboard** — audit log tab shows all rejection events with timestamps
- [ ] **webhook-dashboard** — webhook delivery tab shows each rejection event delivery

---

## 11. Compliance Angle

| Standard | Requirement | How DPoP Satisfies It |
|---|---|---|
| **NIST SP 800-63B AAL3** | §5.1.9 — "proof of possession of a private key corresponding to a public key registered with the verifier" | `cnf.jkt` is the registered public key thumbprint; every request proves possession |
| **PCI DSS 4.0 §8.2.8** | "Service accounts and system accounts managed in a non-human manner ... must authenticate using authentication factors appropriate to account privilege" | DPoP = cryptographic proof per request; satisfies "something you have" for machine identities |
| **HIPAA Security Rule §164.312(d)** | "Entity authentication — implement procedures to verify that a person or entity seeking access to ePHI is the one claimed" | Per-request key proof = continuous entity authentication |
| **ISO 27001:2022 A.9.4.2** | Secure log-on procedures | Bearer token theft from logs → no access; key-binding prevents log-exfiltration attacks |
| **SOC 2 CC6.1** | Logical and physical access controls | DPoP provides cryptographic access control at the protocol layer |

---

## 12. Source File Reference

| File | Purpose |
|---|---|
| `internal/oauth/dpop.go` | Core RFC 9449 validation — all 6 defense mechanisms |
| `internal/oauth/handlers.go` | Token endpoint DPoP interception + cnf.jkt issuance + storeDPoPJKT |
| `sdk/python/shark_auth/dpop.py` | Python SDK DPoPProver — keypair generation + proof emission |
| `sdk/typescript/src/dpop.ts` | TypeScript SDK equivalent |
| `SCALE.md` | Documents JTI cache process-local gap (§1) + Q3 2026 roadmap |
| `ARCHITECTURE.md` | Scale boundaries section, data flow diagrams |
| `demos/dpop_defense/` | All demo files (this plan) |
