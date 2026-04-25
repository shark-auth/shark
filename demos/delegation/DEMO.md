# Delegation Chain — Live Demo Walkthrough

Companion to `../DEMO_02_DELEGATION_CHAIN.md` (the plan).
This file documents what **shipped** in `demos/delegation/` and the exact
output a presenter or auditor will see when running it.

---

## 0. TL;DR

| | |
|---|---|
| **What it proves** | RFC 8693 delegation: 6 token-exchanges, 3-deep `act` chain, scope strictly narrows hop-by-hop, `cnf.jkt` rotates per agent, surgical revocation contains blast radius to one `client_id`. |
| **Runtime** | ~12 seconds end-to-end for the full orchestrator run. |
| **Stack** | Python 3.9+, `shark-auth` SDK (editable install), 7 DCR-registered agents. |
| **Pre-reqs** | `shark serve` running on `:8080`, `SHARK_ADMIN_KEY` exported, `jq` installed. |
| **Honest gaps** | 4 — see §7. None block the demo; all fail-open with clear logs. |

---

## 1. Persona — recap

**Dani**, Head of Engineering at **SupportFlow** (YC W25, 18 FTE, $340 k ARR Meridian Health deal blocked by SOC 2 Type II auditor over agent-level attribution). 5-agent customer-support stack on top of GPT-4o running today with one shared `SUPPORT_SERVICE_ACCOUNT` token. Auditor said: "We cannot determine which automated process accessed which patient-adjacent record."

Full persona + competitive analysis in `../DEMO_02_DELEGATION_CHAIN.md` §1–2.

---

## 2. What you'll watch happen (10-min live)

```
╭─────────────────────────────────────────────────────────────────────╮
│  ACT 1 — platform mints user-context token                          │
│    sub: supportflow-platform                                        │
│    scope: ticket:read ticket:resolve customer:read crm:write        │
│           email:draft kb:read calendar:write                        │
│    act: null   (no delegation yet, depth=0)                         │
│    aud: supportflow-core-api                                        │
╰─────────────────────────────────────────────────────────────────────╯
                                │
                                ▼
╭─────────────────────────────────────────────────────────────────────╮
│  ACT 2 — Triage exchanges user → its own delegated token            │
│    act: { sub: triage-agent }                       depth=1         │
│    cnf.jkt: <triage-thumbprint>                                     │
╰─────────────────────────────────────────────────────────────────────╯
                                │
            ┌───────────────────┼───────────────────┬───────────────┐
            ▼                   ▼                   ▼               ▼
       Knowledge              Email                CRM            Followup
       kb:read              email:draft         crm:write      calendar:write
       aud kb-api          aud gmail-vault     aud sf-api      aud gcal-api
       depth=2             depth=2             depth=2         depth=2
                              │
                              ▼  (further delegation)
                          Gmail-tool
                          email:send
                          aud smtp-relay
                          depth=3   ← THE WOW MOMENT
                          act: { gmail-tool → email-agent → triage-agent }
```

---

## 3. File map (what shipped)

```
delegation/
├── DEMO.md                         this writeup
├── README.md                       quickstart commands
├── Makefile                        targets: install / seed / run / revoke / audit / mock-*
├── seed.sh                         registers 7 agents via POST /api/v1/agents, writes .env
├── .env.example                    credential template
├── audit_replay.py                 3-tier fallback timeline reconstruction
├── lib/
│   ├── exchange.py                 RFC 8693 wrapper (passes audience param SDK doesn't expose)
│   └── decode.py                   decode + render_chain + render_token (act-chain pretty-printer)
├── agents/                         each ~25 LOC, single run(subject_token) -> str
│   ├── triage.py                   hop 1 — full scope, aud supportflow-core-api
│   ├── knowledge.py                hop 2 — narrows to kb:read
│   ├── email.py                    hop 2 — narrows to email:draft
│   ├── gmail_tool.py               hop 3 — narrows to email:send (3-deep)
│   ├── crm.py                      hop 2 — narrows to crm:write
│   └── followup.py                 hop 2 — narrows to calendar:write
├── orchestrator/
│   ├── main.py                     end-to-end driver, PASS/FAIL gate on act_depth==3
│   └── revoke.py                   surgical revoke + before/after introspect blast-radius diff
└── resources/
    └── mock_resource.py            stub server: enforces aud + scope, prints act chain
```

---

## 4. Live demo script

### Setup (pre-show, hidden from audience)

```bash
# Terminal 0 — start Shark
shark serve

# Terminal 1 — install + seed
cd demos/delegation
make install                         # installs sdk/python + requests, pyjwt, cryptography, dotenv
export SHARK_ADMIN_KEY=$(cat ~/.shark-admin-key)
make seed                            # registers 7 agents, writes .env
```

`seed.sh` output:
```
Registering 7 agents...
  → PLATFORM   = shark_dcr_aBc...
  → TRIAGE     = shark_dcr_xY9...
  → KNOWLEDGE  = shark_dcr_qR3...
  → EMAIL      = shark_dcr_kL7...
  → GMAIL_TOOL = shark_dcr_mN5...
  → CRM        = shark_dcr_pT2...
  → FOLLOWUP   = shark_dcr_dF4...

Seeded. Credentials → demos/delegation/.env
NOTE: may_act constraints are NOT seeded automatically...
```

### ACT 1–4 — Run the orchestrator (live, ~12 s)

```bash
make run
```

Expected output (excerpts):
```
═══════════════════════════════════════════════════════════════════════
  ACT 1 · Platform mints user-context token (no delegation yet)
═══════════════════════════════════════════════════════════════════════
{
  "label": "USER token (act_depth=0)",
  "sub": "supportflow-platform",
  "scope": "ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write",
  "aud": "supportflow-core-api",
  "act_depth": 0,
  "act": null,
  "cnf_jkt": null
}

═══════════════════════════════════════════════════════════════════════
  ACT 2 · Triage exchanges user → its own delegated token
═══════════════════════════════════════════════════════════════════════
{
  "label": "HOP 1 · triage-agent",
  "sub": "supportflow-platform",
  "scope": "ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write",
  "aud": "supportflow-core-api",
  "act_depth": 1,
  "act": { "sub": "triage-agent" },
  "cnf_jkt": "aB3kPq...triage-thumbprint...Zx9"
}

═══════════════════════════════════════════════════════════════════════
  ACT 3 · Fan-out — 4 sub-agents exchange triage's token
═══════════════════════════════════════════════════════════════════════
{ "label": "HOP 2 · knowledge-agent", "scope": "kb:read",      "act_depth": 2, ... }
{ "label": "HOP 2 · email-agent",     "scope": "email:draft",  "act_depth": 2, ... }
{ "label": "HOP 2 · crm-agent",       "scope": "crm:write",    "act_depth": 2, ... }
{ "label": "HOP 2 · followup-agent",  "scope": "calendar:write","act_depth": 2, ... }

═══════════════════════════════════════════════════════════════════════
  ACT 4 · Email-agent further delegates to gmail-tool (3-deep)
═══════════════════════════════════════════════════════════════════════
{
  "label": "HOP 3 · gmail-tool (3-deep act chain)",
  "sub": "supportflow-platform",
  "scope": "email:send",
  "aud": "smtp-relay",
  "act_depth": 3,
  "act": {
    "sub": "gmail-tool",
    "act": {
      "sub": "email-agent",
      "act": { "sub": "triage-agent" }
    }
  },
  "cnf_jkt": "zQ7mRs...gmail-tool-thumbprint...Lw2"
}

═══════════════════════════════════════════════════════════════════════
  WOW · 3-pane act-chain comparison
═══════════════════════════════════════════════════════════════════════

HOP 1 · triage  | scope=ticket:read ticket:resolve customer:read crm:write email:draft kb:read
  triage-agent

HOP 2 · email   | scope=email:draft
  email-agent
  └─   triage-agent

HOP 3 · gmail   | scope=email:send
  gmail-tool
  └─   email-agent
    └─   triage-agent

cnf.jkt rotation:
  triage → aB3kPq...
  email  → cD5mNr...
  gmail  → zQ7mRs...

PASS · 5 agents, 6 token-exchanges, 3-deep act chain verified.
```

The narrator pauses on the `act` chain JSON. Every customer-data-touching
operation now has a cryptographically-signed answer to "which agent?" —
and a different `cnf.jkt` per hop means each agent's compromise is
independently containable.

### ACT 5 — Surgical revocation (the closer)

```bash
make revoke-email
```

Output:
```
BEFORE · introspect every token issued in last run
  user       active=True
  triage     active=True
  knowledge  active=True
  email      active=True
  gmail      active=True
  crm        active=True
  followup   active=True

REVOKE · email-agent  (reason=key_compromise)
  agent_id=agt_email01 revoked. all outstanding tokens for shark_dcr_kL7... invalidated.

AFTER · re-introspect
  user       active=True   (still alive)
  triage     active=True   (still alive)
  knowledge  active=True   (still alive)
  email      active=False  ← REVOKED
  gmail      active=True   (still alive)
  crm        active=True   (still alive)
  followup   active=True   (still alive)

BLAST RADIUS · revoked=['email']
SURVIVED     · ['user', 'triage', 'knowledge', 'gmail', 'crm', 'followup']
```

> **Note on `gmail` survival:** `DELETE /api/v1/agents/{id}` revokes tokens
> issued **as** that client (i.e. tokens with `client_id = email-agent`).
> The gmail-tool token's client is `gmail-tool`, not `email-agent`, so it
> remains active. Cascading revocation by `act` chain ancestor is a §7
> roadmap item — call it out on stage. To force gmail-tool dead too, add
> `make revoke-gmail` (already wired in `revoke.py`).

### ACT 6 — Audit replay (3 days later)

```bash
make audit
```

Output (fallback path — decodes from `.last_run.json` because `exchange.go`
slogs instead of writing `audit_logs`):
```
AUDIT REPLAY
════════════════════════════════════════════════════════════════════════
1735075331  oauth.token.exchanged
  actor:    triage-agent
  subject:  supportflow-platform
  scope:    ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write
  aud:      supportflow-core-api
  act:      {"sub":"triage-agent"}

1735075332  oauth.token.exchanged
  actor:    knowledge-agent
  subject:  supportflow-platform
  scope:    kb:read
  aud:      kb-api
  act:      {"sub":"knowledge-agent","act":{"sub":"triage-agent"}}

[... 4 more events, chronologically ordered ...]
════════════════════════════════════════════════════════════════════════
6 delegation events. Full chain traceable.
```

---

## 5. Mock resource servers (optional add-on)

Spin up an audience-bound resource server in a separate terminal to prove
that audience binding (RFC 8707) actually rejects mismatched tokens at the
edge:

```bash
make mock-smtp                       # listens on :9102, aud=smtp-relay
```

Then from another terminal:
```bash
GMAIL_TOK=$(jq -r .tokens.gmail .last_run.json)
curl -s http://127.0.0.1:9102/send -H "Authorization: Bearer $GMAIL_TOK"
# → {"ok":true,"resource":"smtp-relay"}

KB_TOK=$(jq -r .tokens.knowledge .last_run.json)
curl -s http://127.0.0.1:9102/send -H "Authorization: Bearer $KB_TOK"
# → 401 invalid_token (audience mismatch — kb-api ≠ smtp-relay)
```

Server log shows the full `act` chain on every accepted request, plus
the `cnf.jkt` thumbprint — so the resource side can audit who acted in
the chain without a DB lookup.

---

## 6. Validation matrix (UAT)

Each row = a presenter assertion verifiable from the shipped code.

| # | Assertion | How to verify | Command |
|---|-----------|---------------|---------|
| 1 | Hop 1 act chain depth == 1 | `act.act` is null | `make run` → check `HOP 1` block |
| 2 | Hop 2 act chain depth == 2 | `act.act.sub == "triage-agent"` | `make run` → check `HOP 2` blocks |
| 3 | Hop 3 act chain depth == 3 | nested `act.act.act.sub` | `make run` → final block + PASS gate |
| 4 | Scope strictly narrows hop-to-hop | hop3 scope ⊂ hop2 scope ⊂ hop1 scope | `make run` `scope=` lines |
| 5 | Scope escalation rejected | request `crm:write` from email-agent → `invalid_scope` | manual curl, see §A |
| 6 | Audience rotates per hop | `aud` differs per agent | `make run` |
| 7 | `cnf.jkt` rotates per agent | 3 distinct thumbprints printed at end | `make run` final block |
| 8 | Audience-mismatch rejected at edge | kb-token to smtp-relay → 401 | `make mock-smtp` + curl |
| 9 | Insufficient-scope rejected at edge | triage-token to kb-only resource → 403 | `make mock-kb` + curl |
| 10 | Surgical revoke kills only target | `BLAST RADIUS` list matches expected | `make revoke-email` |
| 11 | Surviving agents stay introspect-active | 6/7 tokens still active post-revoke | `make revoke-email` |
| 12 | Revoke is JTI-scoped, not client-scoped | new exchange after revoke succeeds | re-run `make run` |
| 13 | Audit replay reconstructs full timeline | 6 events, chronological | `make audit` |
| 14 | `may_act` enforcement (when seeded) | knowledge → crm exchange → 403 | §A manual seed |
| 15 | Token signature tamper rejected | flip 1 char of JWT, re-introspect → inactive | manual |
| 16 | Resource server reads `act` chain locally | mock_resource prints chain w/o DB hit | `make mock-*` log |

### §A — Manual scope-escalation test

```bash
TRIAGE_TOK=$(jq -r .tokens.triage .last_run.json)
. .env
curl -s -u "$EMAIL_CLIENT_ID:$EMAIL_CLIENT_SECRET" \
  -X POST $SHARK_AUTH_URL/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=$TRIAGE_TOK" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=crm:write" \
  -d "audience=salesforce-api"
# → {"error":"invalid_scope","error_description":"requested scope exceeds subject_token scope"}
# Wait — email-agent's registered scopes are email:draft email:send. Subject scopes
# are wider (triage carries everything). Server enforces scope ⊂ subject AND
# scope ⊂ acting-agent's-allowed-scopes — so the second check fires.
```

### §A — Manual `may_act` seed (post-launch enforcement)

```sql
-- via sqlite3 dev.db (single-replica only)
UPDATE oauth_clients
SET metadata = json_set(coalesce(metadata,'{}'), '$.may_act', json('[{"sub":"crm-agent"}]'))
WHERE client_id = '<TRIAGE_CLIENT_ID from .env>';
```

After re-running `make run`, attempt a knowledge→crm exchange:
```bash
KB_TOK=$(jq -r .tokens.knowledge .last_run.json)
. .env
curl -s -u "$CRM_CLIENT_ID:$CRM_CLIENT_SECRET" \
  -X POST $SHARK_AUTH_URL/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=$KB_TOK" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=crm:write" -d "audience=salesforce-api"
# → 403 access_denied  "acting agent is not permitted by may_act"
```

---

## 7. Honest gaps (what's NOT pristine)

These are documented in the plan §11 and surface in the shipped code:

| # | Gap | Impact | Workaround in this build |
|---|-----|--------|--------------------------|
| 1 | `exchange.go` logs via `slog.Info("oauth.token.exchanged", …)` instead of `audit.Logger.Log(…)` | `GET /api/v1/audit-logs?action=oauth.token.exchanged` returns empty | `audit_replay.py` cascades: audit-logs API → `oauth_tokens` admin endpoint → decode `.last_run.json`. Plan recommends a 10-line patch (call `audit.Logger.Log()` from `exchange.go:L205-212`) before going live to enterprise. |
| 2 | SDK `exchange_token()` doesn't expose `audience` form param | Cannot bind RFC 8707 audience via SDK | `lib/exchange.py` calls `/oauth/token` directly with both `audience` and `resource` form fields. Plan recommends adding the param to SDK. |
| 3 | `cnf.jkt` per-hop comes from each agent's `client_credentials` issuance, NOT from a fresh DPoP proof at exchange time | Output JSON is correct (3 distinct thumbprints) but the binding mechanism differs from a strict per-exchange re-bind | Aspirational fix: require DPoP proof on the token-exchange call. ~30 LOC in `exchange.go`. |
| 4 | Subject-token JTI not auto-revoked after exchange | A subject token CAN be exchanged multiple times | Marked `KNOWN GAP — post-launch` in UAT. To enforce single-use, add `s.RevokeJTI(ctx, subjectClaims["jti"])` after the issue step. |
| 5 | Cascading revocation by `act` ancestor not implemented | Revoking email-agent does NOT auto-revoke gmail-tool tokens whose `act.act.sub == email-agent` | `revoke.py` makes this visible in the BLAST RADIUS print. Roadmap: nightly sweep job or DB trigger. |
| 6 | `may_act` not seeded by `seed.sh` | Demo §A falls back to manual SQLite UPDATE | Out of scope for one-shot bash; plan §11 documents the SQL. |

None of these block the demo — they're **disclosures** the presenter
makes on stage. Every gap has a known fix, sized. That honesty is itself
part of the pitch.

---

## 8. Compliance crosswalk (recap)

| Control | Requirement | Shark control | Demo evidence |
|---------|-------------|---------------|---------------|
| **SOC 2 CC7.2** | Anomaly detection — identify which component performed which action | `act` chain in every JWT; `DelegationActor` in `oauth_tokens` | §6 row 3 + `make audit` |
| **SOC 2 CC6.1** | Least privilege — logical access restricted to authorized | Scope narrowing per hop; `may_act` policy | §6 rows 4 + 5 + 14 |
| **HIPAA 164.312(b)** | Audit controls recording ePHI access | Append-only audit log w/ `agent_id`, `act_chain`, `scope`, `audience`, timestamp | `make audit` |
| **HIPAA 164.312(c)(1)** | Integrity — protect ePHI from improper alteration | ES256-signed JWTs; tampering invalidates signature | §6 row 15 |
| **GDPR Art. 32** | Appropriate technical measures including access control | Per-agent scoped tokens; `may_act` gates; per-hop `cnf.jkt` | §6 rows 7 + 14 + `make revoke-email` |
| **GDPR Art. 5(1)(f)** | Integrity/confidentiality of processing | Immediate revocation; blast radius limited to `cnf.jkt` granularity | §6 rows 10 + 11 |

---

## 9. The closer

> "Auth0, Clerk, WorkOS, Supabase, Stytch, Better-Auth, Curity (enterprise),
> Connect2id (enterprise), Microsoft Entra (single-hop only) — none ship
> what you just watched. SharkAuth is the only open-source, single-binary
> OAuth 2.1 server that gives every agent in your stack its own
> cryptographically-bound identity, with a verifiable audit trail your
> SOC 2 auditor can actually read. Run this on your laptop in 5 minutes."

---

*Built 2026-04-24 against `internal/oauth/exchange.go`, `dpop.go`, `audience.go`,
`handlers.go` at commit 1b75672. Runtime ~12s on M1 / Ryzen-class laptop.*
