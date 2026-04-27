# Concierge Demo Tester Implementation Plan

> **For agentic workers:** Built via sequential sonnet subagents per phase. ENTER-pause UX, leave-on-Ctrl-C cleanup, real-life-inspired scenario.

**Goal:** Ship `tools/agent_demo_concierge.py` — interactive 23-step demo exercising every major shark feature in a single coherent travel-booking narrative, ending with depth-3 delegation chain visible in dashboard.

**Architecture:** Single Python script reusing `shark_auth` SDK + admin REST API. ENTER-pause between steps. Each step prints dashboard URL where effect should be visible. Cleanup on completion ONLY — Ctrl-C leaves state for inspection. Synthetic vault data via `/admin/vault/connections/seed-demo` endpoint already in repo.

**Tech Stack:** Python 3.11+, `requests`, `shark_auth.dpop.DPoPProver`, `shark_auth.OAuthClient`. ANSI colors via plain escape sequences (no `colorama`).

---

## Scenario: Acme Travel — AI Concierge

End customer "Maria Chen" runs an AI-powered booking concierge. Concierge orchestrates 4 specialist sub-agents and 1 sub-sub-agent. Final delegation chain depth: 3.

```
Maria (human)
└─ Travel Concierge (master, client_credentials + DPoP)
   ├─ Flight Booker  (act_chain depth 2, vault: Amadeus)
   │  └─ Payment Processor  (act_chain depth 3, vault: Stripe)
   ├─ Hotel Booker   (act_chain depth 2, vault: Booking.com)
   ├─ Calendar Sync  (act_chain depth 2, vault: Google Calendar)
   └─ Expense Filer  (act_chain depth 2, vault: Concur)
```

---

## File Structure

- Create: `tools/agent_demo_concierge.py` (single file, ~600-800 LOC)
- Modify: none (uses existing endpoints + SDK)
- Reference: `tools/agent_demo_tester.py` (existing simpler demo — same helper style)
- New backend if MFA missing: TBD pending /investigate report

---

## Decisions locked

| Q | A |
|---|---|
| MFA step-up backend | **Option A locked** — script stubs the 403 + re-exchange. Real `/auth/mfa/enroll` + `/auth/mfa/challenge` endpoints used (session upgrade is real); 403 step_up_required emitted by demo policy layer because backend has no `acr`/`amr` claim emission and no per-vault MFA gating. |
| Org-scoping on agents | Drop. Org step is cosmetic only. |
| Vault provisioning | Use `/admin/vault/connections/seed-demo` (already in `internal/api/vault_handlers.go`) |
| Pacing | ENTER to continue (no auto-advance default) |
| Cleanup | Run on natural finish ONLY. Ctrl-C → leave state for inspection. |

---

## Step list (23 steps)

| # | Step | Shark feature |
|---|---|---|
| 1 | Signup Maria via `POST /auth/signup` | Human auth |
| 2 | Capture magic-link from `/admin/dev-email`, verify | Magic link + dev inbox |
| 3 | Login + capture session cookie | Session manager |
| 4 | Create org "Acme Travel", assign Maria as owner | Org RBAC (cosmetic) |
| 5 | DCR-register Travel Concierge as OAuth client (RFC 7591) | DCR |
| 6 | Create scoped admin API key for the demo | API keys |
| 7 | Create 4 specialist agents (flight/hotel/calendar/expense), all with `created_by=Maria.user_id` | Agent CRUD + cascade-revoke binding |
| 8 | Configure `may_act` policies (Concierge → all 4 specialists; Flight Booker → Payment Processor) | Delegation policies |
| 9 | Provision 5 vault entries via `/admin/vault/connections/seed-demo` (Amadeus, Booking, GCal, Concur, Stripe) | Vault FieldEncryptor |
| 10 | Concierge issues DPoP-bound client_credentials token | OAuth + RFC 9449 |
| 11 | Concierge → Flight Booker via token-exchange (act_chain depth 2) | RFC 8693 |
| 12 | Flight Booker fetches Amadeus token via vault retrieval w/ DPoP jkt match | Vault gating by jkt |
| 13 | Parallel: Hotel/Calendar/Expense each retrieve their vault token | Concurrent depth-2 |
| 14 | Flight Booker → Payment Processor token-exchange (act_chain depth 3) | Multi-hop chain |
| 15 | Payment Processor charges $850 via Stripe vault → success | Sub-sub-agent vault retrieval |
| 16 | Payment Processor charges $1500 → 403 step_up_required | MFA enforcement |
| 17 | Maria submits TOTP code → session elevated, token re-issued with acr=mfa | Step-up auth |
| 18 | Re-attempt $1500 → success | Elevated chain |
| 19 | Audit log fetch — render full ASCII tree of chain + metadata | Audit specificity (just shipped) |
| 20 | Rotate Flight Booker DPoP key mid-flight (Layer 4 / Method 10) | DPoP rotation |
| 21 | Bulk-revoke `tc_payment_*` pattern | Layer 4 bulk |
| 22 | Disconnect Stripe vault → Payment Processor token revoked, others survive | Layer 5 cascade |
| 23 | Cascade-revoke Maria → all agents + vault tokens wiped | Layer 3 cascade |

---

## Phase A: Investigation (parallel, sonnet)

- [ ] Sonnet `/investigate` agent reports MFA step-up state. Output: BLOCKED / PARTIAL / PRESENT + recommended option (A vs B).

## Phase B: Core script — steps 1-13 (sonnet subagent #1, worktree)

- [ ] Header + helpers (step/info/ok/fail/dashboard/pause)
- [ ] `--fast` and `--no-cleanup` arg parsing (only ENTER-pause; auto-advance only when --fast set)
- [ ] State dataclass for Maria + agents + vault keys
- [ ] Step 1-13 sequentially, each as its own function `step_01_signup`, etc.
- [ ] After each step: dashboard URL + 1-line hint
- [ ] Run script against running shark.exe to confirm steps 1-13 pass

## Phase C: Chain depth + cascades — steps 14-23 (sonnet subagent #2, worktree)

- [ ] Steps 14-15: depth-3 token-exchange + Stripe charge
- [ ] Steps 16-18: MFA step-up branch (option A stub OR option B real)
- [ ] Step 19: ASCII tree renderer of audit chain
- [ ] Step 20-22: rotation + bulk revoke + vault disconnect
- [ ] Step 23: full cascade
- [ ] Cleanup function (only on natural finish; Ctrl-C handler that prints "leaving state for inspection" + exits)
- [ ] E2E run — all 23 steps green

## Phase D: Optional MFA backend hook (sonnet subagent #3, only if Option B picked)

- [ ] Add `mfa_at` column to sessions table (or reuse existing if present)
- [ ] Token-exchange handler emits `acr=mfa` if session is MFA-elevated within last 5 min
- [ ] Vault retrieval middleware checks per-vault `requires_mfa` flag against token acr
- [ ] Smoke test: `tests/smoke/test_mfa_step_up.py`

## Phase E: Demo copy doc (`docs/demos/concierge.md`)

Customer-facing narrative + moat positioning. Not technical reference — copywriting.

Sections:
- **The story** — Maria runs an AI travel app. Her concierge agent books trips. End-customer plain English.
- **What you just saw** — per-step plain English (not the 23-row table; 6-8 paragraphs grouped by phase: signup → spawn agents → delegate → step-up → cascade)
- **The moat** —

  > **Your agents are already doing this. They're just not doing it safely.**
  >
  > Sub-agents, vault, delegation chains, MFA step-up, cascade revoke — these are not optional features for production agents. Every team shipping AI agents has them. The question is whether you're rolling your own (and getting it wrong), or using shark.
  >
  > Without shark: 3 months of OAuth + DPoP + RFC 8693 + audit infra + cascade-revoke schema design. Without shark: token theft is a P0 incident. Without shark: you're discovering MFA-on-agent-tokens isn't really a thing the day a customer asks for it.
  >
  > With shark: same code, same security primitives, cryptographic prevention not just detection. The right way. The secure way. The secure AND fast way.
- **Try it** — `python tools/agent_demo_concierge.py`. CTA to dashboard.

## Phase F: Polish + commit

- [ ] ASCII delegation tree at step 19 mirrors Railway canvas in dashboard
- [ ] ANSI colors for OH-SHIT moments (chain depth, MFA elevation, cascade)
- [ ] Final summary table on completion
- [ ] Commit in 1-3 commits depending on backend touch

---

## Dispatch sequencing

```
Phase A (investigate)  ──┐
                          ├──> Phase B (steps 1-13) ──> Phase C (steps 14-23) ──> Phase E (polish + commit)
Phase D (only if Opt-B) ──┘
```

Phase A runs in background while plan is finalised. Phase B starts after Phase A returns (need MFA verdict to know if Phase D fires).

---

## Risks + mitigations

- **Backend MFA gap blocks demo realism** → Phase A decides A-stub vs B-real; budget caps option B at ~1h CC
- **Agent stalling mid-script (seen in prior dispatches)** → split into two sequential subagent passes (B + C) with explicit step-list deliverable per pass; if either stalls, orchestrator re-dispatches with narrower scope
- **Vault seed endpoint incompatibility with 5 distinct providers** → endpoint already accepts arbitrary `provider` field per `internal/api/vault_handlers.go`; verified during Wave 1.6
- **Real Stripe/Amadeus calls unsafe** → all "external API" calls are stubbed in-script (script prints "[mock] Stripe charged $850" — vault retrieval is real, downstream HTTP call is fake). Demo value is in delegation+vault chain, not actual API integration.
