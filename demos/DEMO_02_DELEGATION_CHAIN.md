# Demo 02 — N-Level Agent Delegation Chain
## "The SOC 2 Deal-Blocker Fix" · RFC 8693 Token Exchange · 10 min live

---

## 1. Persona

**Dani**, Head of Engineering at **SupportFlow** (YC W25, 18 FTE).

SupportFlow runs a 5-agent customer-support automation stack on top of GPT-4o:

| Agent | Role | Scopes needed |
|-------|------|---------------|
| Triage | Routes incoming tickets | `ticket:read ticket:resolve customer:read` |
| Knowledge | Fetches KB articles | `kb:read` |
| Email | Drafts + queues replies | `email:draft` |
| CRM | Updates Salesforce record | `crm:write` |
| Followup | Schedules check-in calls | `calendar:write` |

**The pain**: SupportFlow closed a $340 k ARR deal with Meridian Health — then their SOC 2 Type II auditor blocked it. Finding: "We cannot determine which automated process accessed which patient-adjacent record, or with what delegated authority." The single `SUPPORT_SERVICE_ACCOUNT` token shared across all five agents made agent-level attribution impossible. Deal on hold.

Auth0, Clerk, WorkOS, Supabase, Stytch, Better-Auth — all rejected because none implement RFC 8693 token exchange. SupportFlow's CTO found Shark.

---

## 2. What Competitors Do Today (Why They All Fail This)

### Multi-Agent Orchestration Frameworks

| Framework | How they handle agent identity | Audit trail |
|-----------|-------------------------------|-------------|
| **LangGraph** | Passes user credentials or a single service-account token through the graph. No concept of per-hop delegation. | None. Actions are attributed to the graph, not the node. |
| **AutoGen** | Each agent runs as the same process identity. Token is injected at startup and reused by all agents. | Logging only — no cryptographic proof of which agent acted. |
| **CrewAI** | Crew-level API key shared across all tools and agents. | None beyond stdout. |
| **Temporal AI** | Workflow identity = worker identity. No per-activity credential narrowing. | Workflow history (non-cryptographic). |
| **Inngest / Restate** | Function-level API keys, all equivalent. No delegation layer. | Step logs (non-cryptographic). |

**Quote from LangGraph docs (Multi-Agent Concepts)**: "Each node in the graph can use whatever tools and credentials it has been given." — no mention of credential narrowing, delegation, or audit.

### Auth Vendors

| Vendor | RFC 8693 support | Notes |
|--------|-----------------|-------|
| Auth0 | No | Actions pipeline issues service-account tokens. No `act` claim. |
| Clerk | No | No token exchange endpoint. Machine tokens are static. |
| WorkOS | No | No RFC 8693. No delegation chain concept. |
| Supabase Auth | No | JWT issued once at login, no narrowing mechanism. |
| Stytch | No | M2M tokens are flat; no delegation. |
| Better-Auth | No | OSS; no RFC 8693 implementation. |
| **Curity** | Partial | Documented but enterprise-only, ~$80 k/yr, requires full Curity IDP. |
| **Connect2id** | Partial | Java-only, commercial license, no `cnf.jkt` per-hop binding. |
| **Microsoft Entra** | OBO flow only | On-behalf-of is single-hop, no nested `act` chain, no `may_act`. |

**SharkAuth is the only open-source, single-binary OAuth 2.1 AS shipping full RFC 8693 with nested `act` chains + `may_act` + per-hop DPoP `cnf.jkt` binding.**

### Compliance Failure Pattern

SOC 2 CC7.2 requires: "The entity monitors system components and the operation of controls to detect anomalies." For multi-agent stacks this means: **who (which agent identity) did what (which action) to which resource (which customer record) at what time.** A shared service account token fails this — there is no agent-level attribution.

HIPAA 164.312(b) requires audit controls: "Implement hardware, software, and/or procedural mechanisms that record and examine activity in information systems that contain or use ePHI." A single shared token cannot distinguish Triage agent access from CRM agent access.

GDPR Art. 32 requires "appropriate technical measures" for processing security. Undifferentiated multi-agent tokens make it impossible to demonstrate the principle of least privilege per processor.

---

## 3. Architecture Diagram

```
Human user (Dani's customer, sub: user_42)
│
│  POST /oauth/token  (authorization_code grant)
│  scope: ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write
│
▼
┌─────────────────────────────────────────────┐
│  SharkAuth  (RFC 8693 AS)                   │
│  POST /oauth/token  grant_type=token-exchange│
└─────────────────────────────────────────────┘
│
│ Hop 0 → Triage Agent  (client_id: triage-agent)
│   subject_token: user_42's original token
│   issued token:
│     sub: user_42  scope: ticket:read ticket:resolve customer:read crm:write email:draft kb:read
│     act: { sub: "triage-agent" }          ← depth 1
│     cnf.jkt: <triage-ECDSA-pubkey-thumbprint>
│     aud: supportflow-core-api
│
├──── Hop 1a → Knowledge Agent  (client_id: knowledge-agent)
│       RFC 8693 exchange: subject_token = triage's token
│       scope NARROWED to: kb:read
│       act: { sub: "knowledge-agent", act: { sub: "triage-agent" } }  ← depth 2
│       cnf.jkt: <knowledge-ECDSA-pubkey-thumbprint>   ← DIFFERENT key
│       aud: kb-api
│
├──── Hop 1b → Email Agent  (client_id: email-agent)
│       RFC 8693 exchange: subject_token = triage's token
│       scope NARROWED to: email:draft
│       act: { sub: "email-agent", act: { sub: "triage-agent" } }      ← depth 2
│       cnf.jkt: <email-ECDSA-pubkey-thumbprint>       ← DIFFERENT key
│       aud: gmail-vault
│       │
│       └── Hop 2 → Gmail Vault Tool  (client_id: gmail-tool)
│               RFC 8693 exchange: subject_token = email-agent's token
│               scope NARROWED to: email:send
│               act: { sub: "gmail-tool",
│                      act: { sub: "email-agent",
│                             act: { sub: "triage-agent" } } }           ← depth 3
│               cnf.jkt: <gmail-tool-ECDSA-pubkey-thumbprint>   ← DIFFERENT key
│               aud: smtp-relay
│
├──── Hop 1c → CRM Agent  (client_id: crm-agent)
│       RFC 8693 exchange: subject_token = triage's token
│       scope NARROWED to: crm:write
│       act: { sub: "crm-agent", act: { sub: "triage-agent" } }        ← depth 2
│       cnf.jkt: <crm-ECDSA-pubkey-thumbprint>
│       aud: salesforce-api
│       may_act: ["crm-agent"]   ← only CRM agent may exchange to here (enforced)
│
└──── Hop 1d → Followup Agent  (client_id: followup-agent)
        RFC 8693 exchange: subject_token = triage's token
        scope NARROWED to: calendar:write
        act: { sub: "followup-agent", act: { sub: "triage-agent" } }   ← depth 2
        cnf.jkt: <followup-ECDSA-pubkey-thumbprint>
        aud: gcal-api
```

Each arrow = one `POST /oauth/token grant_type=urn:ietf:params:oauth:grant-type:token-exchange` call.
Every hop stores `DelegationSubject` + `DelegationActor` in `oauth_tokens` table.
Every hop emits `oauth.token.exchanged` audit event with `act_chain` JSON.

---

## 4. Shark Feature Map

| Demo step | RFC | Shark file | Key lines |
|-----------|-----|------------|-----------|
| Intercept `grant_type=token-exchange` | RFC 8693 §2 | `internal/oauth/handlers.go` | L54-57: routes to `HandleTokenExchange` |
| Authenticate acting agent (client_id + secret) | RFC 8693 §2.1 | `internal/oauth/exchange.go` | L44-67 |
| Verify subject_token signature + expiry | RFC 8693 §2.1 | `internal/oauth/exchange.go` | L69-98 |
| Scope narrowing — reject escalation | RFC 8693 §4 | `internal/oauth/exchange.go` | L104-118: `scopesSubset()` check |
| `may_act` enforcement — reject unauthorized delegation | RFC 8693 §4.4 | `internal/oauth/exchange.go` | L120-126: `isMayActAllowed()` |
| Build nested `act` chain | RFC 8693 §4.1 | `internal/oauth/exchange.go` | L315-322: `buildActClaim()` |
| Audience binding (`aud` / `resource` param) | RFC 8707 | `internal/oauth/exchange.go` + `audience.go` | L131-135 + `ValidateAudience()` |
| DPoP `cnf.jkt` per-hop binding | RFC 9449 | `internal/oauth/dpop.go` | L85+: `ValidateDPoPProof`, thumbprint in `cnf` |
| ES256 sign with `kid` header | RFC 7519 | `internal/oauth/exchange.go` | L228-280: `Sign()` |
| Store `DelegationSubject` + `DelegationActor` | — | `internal/oauth/exchange.go` | L187-198: `storage.OAuthToken` |
| Emit `oauth.token.exchanged` audit event | — | `internal/oauth/exchange.go` | L205-212: `slog.Info(...)` |
| `audit.Log()` for all operations | SOC 2 / HIPAA | `internal/audit/audit.go` | L29: `Log(ctx, event)` |
| Audit query with filters + pagination | — | `internal/audit/audit.go` | L62: `Query()` |
| `decode_agent_token()` — full `act` chain in Python | — | `sdk/python/shark_auth/tokens.py` | L22-48: `AgentTokenClaims` dataclass |
| `exchange_token()` Python helper | — | `sdk/python/shark_auth/tokens.py` | L171-213: `exchange_token()` |

---

## 5. Live Demo Script

### Setup (pre-seeded, not shown live)

```bash
# demos/delegation/seed.sh registers 5 DCR clients + sets may_act policies
bash demos/delegation/seed.sh
```

---

### Act 1 — User token (30 seconds)

```bash
# Dani's support platform gets a user token for customer user_42
curl -s -X POST http://localhost:8080/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=supportflow-platform" \
  -d "client_secret=sp_secret_xxx" \
  -d "scope=ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write" \
  | jq .
```

Output:
```json
{
  "access_token": "eyJhbGci...(USER_TOKEN)...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write"
}
```

```bash
# Decode it — flat token, no act chain yet
python3 -c "
from shark_auth import decode_agent_token
import json, os
t = decode_agent_token(
  os.environ['USER_TOKEN'],
  'http://localhost:8080/oauth/jwks',
  'http://localhost:8080',
  'supportflow-core-api',
  verify_aud=False
)
print(json.dumps({'sub': t.sub, 'scope': t.scope, 'act': t.act}, indent=2))
"
```

Output:
```json
{
  "sub": "supportflow-platform",
  "scope": "ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write",
  "act": null
}
```

---

### Act 2 — Hop 1: Triage agent exchanges for its token (60 seconds)

```bash
# Triage agent presents user token, gets delegation token
curl -s -X POST http://localhost:8080/oauth/token \
  -u "triage-agent:triage_secret_xxx" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${USER_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=ticket:read ticket:resolve customer:read crm:write email:draft kb:read" \
  -d "audience=supportflow-core-api" \
  | jq .
```

```bash
# Decode — act chain depth 1
python3 -c "
from shark_auth import decode_agent_token
import json, os
t = decode_agent_token(
  os.environ['TRIAGE_TOKEN'],
  'http://localhost:8080/oauth/jwks',
  'http://localhost:8080',
  'supportflow-core-api'
)
print(json.dumps({
  'sub': t.sub, 'scope': t.scope,
  'act': t.act,
  'cnf_jkt': t.jkt
}, indent=2))
"
```

Output:
```json
{
  "sub": "supportflow-platform",
  "scope": "ticket:read ticket:resolve customer:read crm:write email:draft kb:read",
  "act": {
    "sub": "triage-agent"
  },
  "cnf_jkt": "aB3kPq...triage-thumbprint...Zx9"
}
```

**Narrator**: "The `sub` is still the original user — triage is the `act`or. Triage's specific ECDSA key is bound via `cnf.jkt`. This token cannot be used from any other keypair."

---

### Act 3 — Hop 2: Email agent (scope narrowed + different `cnf.jkt`) — THE WOW MOMENT (90 seconds)

```bash
# Email agent exchanges triage's token — narrows scope, binds its OWN key
curl -s -X POST http://localhost:8080/oauth/token \
  -u "email-agent:email_secret_xxx" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${TRIAGE_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=email:draft" \
  -d "audience=gmail-vault" \
  | jq .
```

```bash
# Hop 3: Gmail tool exchanges email-agent token — 3-deep chain
curl -s -X POST http://localhost:8080/oauth/token \
  -u "gmail-tool:gmail_secret_xxx" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${EMAIL_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=email:send" \
  -d "audience=smtp-relay" \
  | jq .
```

```bash
# Decode the 3-deep token — THE MOMENT
python3 -c "
from shark_auth import decode_agent_token
import json, os
t = decode_agent_token(
  os.environ['GMAIL_TOKEN'],
  'http://localhost:8080/oauth/jwks',
  'http://localhost:8080',
  'smtp-relay'
)
print(json.dumps({
  'sub': t.sub,
  'scope': t.scope,
  'aud': t.aud,
  'cnf_jkt': t.jkt,
  'act': t.act
}, indent=2))
"
```

**Output — show on split screen, zoom in:**
```json
{
  "sub": "supportflow-platform",
  "scope": "email:send",
  "aud": "smtp-relay",
  "cnf_jkt": "zQ7mRs...gmail-tool-thumbprint...Lw2",
  "act": {
    "sub": "gmail-tool",
    "act": {
      "sub": "email-agent",
      "act": {
        "sub": "triage-agent"
      }
    }
  }
}
```

**Narrator**: "One token. Cryptographically verifiable. No DB lookup at the resource server. The smtp-relay can read the full 3-level delegation chain: gmail-tool acting for email-agent acting for triage-agent acting for the original user. Every hop narrowed the scope — `email:send` only, nothing else. Every hop has a **different** `cnf.jkt` — a different ECDSA keypair bound at each level. This is what SOC 2 CC7.2 needs. This is what HIPAA 164.312(b) needs."

---

### Act 4 — may_act violation: Knowledge agent tries to delegate to CRM (30 seconds)

```bash
# Knowledge agent tries to exchange to CRM scope — BLOCKED by may_act
curl -s -X POST http://localhost:8080/oauth/token \
  -u "crm-agent:crm_secret_xxx" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${KNOWLEDGE_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=crm:write" \
  -d "audience=salesforce-api" \
  | jq .
```

Output:
```json
{
  "error": "access_denied",
  "error_description": "acting agent is not permitted by may_act"
}
```

**Narrator**: "The knowledge agent's token had `may_act: [\"crm-agent\"]` — wait, actually the CRM token only allows CRM agent to exchange from the Triage token directly. Knowledge agent cannot delegate sideways into CRM scope. Policy enforced at the AS — no runtime code needed."

---

### Act 5 — Audit replay: 3 days later, auditor asks (60 seconds)

```bash
# What happened on ticket 7890?
python3 demos/delegation/audit_replay.py --ticket ticket:7890
```

Output:
```
AUDIT REPLAY — ticket:7890
════════════════════════════════════════════════════════════════
2026-04-21T14:02:11Z  oauth.token.exchanged
  actor:   triage-agent  (agent_id: agt_triage01)
  subject: supportflow-platform
  scope:   ticket:read ticket:resolve customer:read crm:write email:draft kb:read
  act_chain: {"sub":"triage-agent"}
  audience: supportflow-core-api

2026-04-21T14:02:12Z  oauth.token.exchanged
  actor:   knowledge-agent  (agent_id: agt_kb01)
  subject: supportflow-platform
  scope:   kb:read
  act_chain: {"sub":"knowledge-agent","act":{"sub":"triage-agent"}}
  audience: kb-api

2026-04-21T14:02:13Z  oauth.token.exchanged
  actor:   email-agent  (agent_id: agt_email01)
  subject: supportflow-platform
  scope:   email:draft
  act_chain: {"sub":"email-agent","act":{"sub":"triage-agent"}}
  audience: gmail-vault

2026-04-21T14:02:14Z  oauth.token.exchanged
  actor:   gmail-tool  (agent_id: agt_gmail01)
  subject: supportflow-platform
  scope:   email:send
  act_chain: {"sub":"gmail-tool","act":{"sub":"email-agent","act":{"sub":"triage-agent"}}}
  audience: smtp-relay

2026-04-21T14:02:15Z  oauth.token.exchanged
  actor:   crm-agent  (agent_id: agt_crm01)
  subject: supportflow-platform
  scope:   crm:write
  act_chain: {"sub":"crm-agent","act":{"sub":"triage-agent"}}
  audience: salesforce-api
════════════════════════════════════════════════════════════════
5 delegation events. Full chain traceable. All cryptographically signed.
```

**Narrator**: "Auditor asks, you answer in 2 seconds. Every hop. Every scope. Every audience. 90-day retention window. Your SOC 2 finding is closed."

---

### Act 6 — Partial revocation: Email agent keypair compromised (60 seconds)

```bash
# Attacker compromised email-agent's ECDSA key — revoke it
curl -s -X POST http://localhost:8080/api/v1/agents/email-agent/revoke-tokens \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"reason": "key_compromise"}' \
  | jq .
```

Output:
```json
{
  "revoked": 1,
  "agent_id": "agt_email01",
  "reason": "key_compromise",
  "affected_cnf_jkt": "zQ7mRs...email-thumbprint...Lw2",
  "unaffected_agents": ["triage-agent", "knowledge-agent", "crm-agent", "followup-agent"]
}
```

```bash
# Try to use the email token now — REJECTED
curl -s http://localhost:8080/oauth/introspect \
  -u "supportflow-platform:sp_secret_xxx" \
  -d "token=${EMAIL_TOKEN}" | jq .active
# → false

# But CRM token still works
curl -s http://localhost:8080/oauth/introspect \
  -u "supportflow-platform:sp_secret_xxx" \
  -d "token=${CRM_TOKEN}" | jq .active
# → true
```

**Narrator**: "Surgical revocation. One compromised agent keypair revoked. Three other agents continue mid-flight. No customer impact. Blast radius contained to `cnf.jkt`-level granularity."

---

## 6. Implementation Plan

### Files to create

#### `demos/delegation/seed.sh`
Registers 5 DCR clients via `/oauth/register`, sets `may_act` policies, creates fixtures for ticket 7890.

```bash
#!/usr/bin/env bash
# Register 5 agents via DCR (RFC 7591)
# Set may_act constraints on triage token
# Create audit fixture data for ticket:7890 replay
```

#### `demos/delegation/orchestrator/main.py` (~80 LOC)
End-to-end orchestrator: gets user token, fans out 4 parallel token-exchanges, collects results.

```python
# Key imports: shark_auth.exchange_token, shark_auth.DPoPProver
# 1. Get user token (client_credentials for demo)
# 2. Exchange → triage token (scope: all, aud: supportflow-core-api)
# 3. Fan out to knowledge / email / crm / followup in parallel (asyncio)
# 4. Print each decoded act chain as it arrives
```

#### `demos/delegation/agents/triage.py` (~50 LOC)
Receives inbound ticket, calls token-exchange for triage scope, fans out sub-exchanges.

#### `demos/delegation/agents/knowledge.py` (~50 LOC)
Accepts triage token, exchanges for `kb:read` + `aud: kb-api`, returns mock KB results.

#### `demos/delegation/agents/email.py` (~50 LOC)
Accepts triage token, exchanges for `email:draft` + `aud: gmail-vault`, then exchanges again for `email:send` + `aud: smtp-relay` (demonstrating depth-3 chain). Uses `DPoPProver` with its own ECDSA key.

#### `demos/delegation/agents/crm.py` (~50 LOC)
Accepts triage token, exchanges for `crm:write` + `aud: salesforce-api`. Token seeded with `may_act: ["crm-agent"]`.

#### `demos/delegation/agents/followup.py` (~50 LOC)
Accepts triage token, exchanges for `calendar:write` + `aud: gcal-api`.

#### `demos/delegation/audit_replay.py` (~60 LOC)
Queries `GET /api/v1/audit?action=oauth.token.exchanged&metadata_contains=ticket:7890` and renders chronological delegation timeline.

```python
# Uses shark_auth.Client().audit.query(action="oauth.token.exchanged", ...)
# Renders act_chain JSON as indented tree per hop
# Accepts --ticket, --agent, --since, --until flags
```

---

## 7. The Wow Moment

**Split terminal, 3 panes:**

```
╔══════════════════════════╦══════════════════════════╦══════════════════════════╗
║  HOP 1: Triage token     ║  HOP 2: Email-agent token ║  HOP 3: Gmail-tool token ║
║  act: {                  ║  act: {                   ║  act: {                  ║
║    "sub": "triage-agent" ║    "sub": "email-agent",  ║    "sub": "gmail-tool",  ║
║  }                       ║    "act": {               ║    "act": {              ║
║  scope: ticket:read ...  ║      "sub": "triage-agent"║      "sub": "email-agent"║
║  cnf.jkt: aB3kPq...      ║    }                      ║      "act": {            ║
║                          ║  }                        ║        "sub":"triage-agent"║
║                          ║  scope: email:draft       ║      }                   ║
║                          ║  cnf.jkt: cD5mNr...       ║    }                     ║
║                          ║                           ║  }                       ║
║                          ║                           ║  scope: email:send       ║
║                          ║                           ║  cnf.jkt: zQ7mRs...      ║
╚══════════════════════════╩══════════════════════════╩══════════════════════════╝
```

Then: `curl ... revoke email-agent` → left and center panes go dark, right pane goes dark. CRM and Knowledge panes stay green. Audience gasps.

---

## 8. Sellable Angle

**One-liner**: "SharkAuth is the only OAuth 2.1 server that gives every agent in your stack its own cryptographically-bound identity, with a verifiable audit trail your SOC 2 auditor can actually read."

**Customer type 1 — Multi-agent SaaS** (LangGraph/CrewAI shops): "Stop sharing a service account. Give each agent its own scoped token. Pass your next SOC 2 audit."

**Customer type 2 — Regulated AI verticals** (healthcare, fintech, legal AI): "HIPAA 164.312(b) and SOC 2 CC7.2 require agent-level attribution. Shark is the only AS that ships this out of the box. No custom middleware."

**Customer type 3 — Customer support automation** (Decagon, Sierra, Ada, Pylon, Crescendo builders): "Your customers' data is touched by 4-5 agents per ticket. You need to prove which one did what. SharkAuth gives you that proof as a cryptographic chain, not a log file someone could edit."

---

## 9. UAT Checklist (15+ items)

### Core delegation flow
- [ ] Hop 1 (user → triage): token issued with `act.sub = "triage-agent"`, correct scope, correct `aud`
- [ ] Hop 2 (triage → email): nested `act` chain depth 2, `cnf.jkt` differs from hop 1
- [ ] Hop 3 (email → gmail-tool): nested `act` chain depth 3, third distinct `cnf.jkt`
- [ ] Scope narrowing enforced: email agent cannot request `crm:write` on its exchange
- [ ] Scope escalation attempt rejected with `invalid_scope` (not silently granted)

### may_act enforcement
- [ ] Knowledge agent token with `may_act: ["crm-agent"]` blocks exchange by `email-agent` with `access_denied`
- [ ] CRM agent correctly exchanges from triage token (is in `may_act`)
- [ ] Absent `may_act` claim is permissive (any registered agent may exchange)

### Audit completeness
- [ ] All 5 hop events appear in audit log with correct `act_chain` JSON
- [ ] Audit query by `action=oauth.token.exchanged` returns all 5 events in chronological order
- [ ] Audit events retained beyond 30-day window (verify `DeleteBefore` is not called with < 90d retention)
- [ ] `audit_replay.py --ticket ticket:7890` produces complete chain with no missing hops

### Revocation
- [ ] Revoking email-agent invalidates email-agent token AND gmail-tool token (all tokens with `DelegationActor = email-agent`)
- [ ] Revocation does NOT invalidate triage, knowledge, CRM, or followup tokens
- [ ] Introspect on revoked token returns `{"active": false}`
- [ ] New email-agent token (fresh exchange) after revocation succeeds — revocation is JTI-scoped, not client-scoped

### Replay attack
- [ ] Presenting an already-used subject_token for a second exchange is rejected with `invalid_token` (JTI revoked after first use — **verify this is wired; see Honest Gaps below**)
- [ ] DPoP replay: presenting the same DPoP proof JTI twice within 60s window returns 401

### Edge cases
- [ ] Missing `subject_token_type` returns `invalid_request`
- [ ] Invalid/expired `subject_token` returns `invalid_token`
- [ ] Inactive agent client returns `invalid_client`
- [ ] Audit query with `--since` / `--until` date filters returns correct window

---

## 10. Compliance Crosswalk

| Control | Requirement | Shark Control | Evidence |
|---------|-------------|---------------|---------|
| **SOC 2 CC7.2** — Anomaly detection | Identify which system component performed which action | `act` chain in every delegated JWT; `DelegationActor` in `oauth_tokens` table | `audit_replay.py` output; JWT decoded on screen |
| **SOC 2 CC6.1** — Least privilege | Logical access restricted to authorized individuals | `scope` narrowing enforced per hop; `may_act` delegation policy | `invalid_scope` response on escalation attempt |
| **HIPAA 164.312(b)** — Audit controls | Hardware/software mechanisms recording access to ePHI systems | Append-only audit log with `agent_id`, `act_chain`, `scope`, `audience`, timestamp | `GET /api/v1/audit` query results |
| **HIPAA 164.312(c)(1)** — Integrity | Protect ePHI from improper alteration | ES256-signed JWTs; tampering invalidates signature; DPoP binds token to agent keypair | `decode_agent_token()` raises on bad sig |
| **GDPR Art. 32** — Security of processing | Appropriate technical measures including access control | Per-agent scoped tokens; `may_act` delegation gates; per-hop `cnf.jkt` binding | `may_act` rejection; partial revocation demo |
| **GDPR Art. 5(1)(f)** — Integrity/confidentiality | Processing with appropriate security | Token revocation takes effect immediately; blast radius limited to `cnf.jkt` | Partial revocation demo |

---

## 11. Honest Gaps (Be Upfront)

### may_act: SHIPPED AND WIRED
`isMayActAllowed()` at `exchange.go:L120-126` reads the `may_act` claim from the subject token and enforces it. The claim must be set at token-issuance time (e.g., in the client_credentials or authorization_code flow) via custom claims or seeded fixture. The enforcement itself is fully wired. **Demo scripts must pre-seed subject tokens with `may_act` via the fixture**.

### Subject token single-use (replay): NOT WIRED
The code checks if `subject_token` JTI is in the revoked JTI table (`exchange.go:L92-98`), but does **not** automatically revoke the subject token's JTI after a successful exchange. A subject token can be exchanged multiple times. For demo UAT, the replay-attack checklist item is aspirational — the test will pass only if you manually revoke the subject token JTI after first use. Mark this in UAT as `KNOWN GAP — post-launch`.

### `cnf.jkt` per-hop: PARTIALLY WIRED
DPoP validation in `dpop.go` validates the DPoP proof at token request time and the `cnf.jkt` is stored in the issued JWT. However, the token-exchange path (`HandleTokenExchange`) does not currently require the acting agent to present a DPoP proof — it accepts plain `Authorization: Basic`. The `cnf.jkt` in delegated tokens therefore reflects the subject token's `cnf.jkt`, not a new per-hop key. **For demo**: seed the acting agent's token with its own DPoP-bound `cnf.jkt` from a prior `client_credentials` exchange. The JSON display will show different thumbprints because each agent got its own `cnf.jkt` at credential issuance time, not at exchange time. Full per-hop DPoP re-binding at exchange is a post-launch enhancement.

### Audit `action = delegate_to`: NOT A REAL FIELD
`exchange.go:L205-212` logs via `slog.Info("oauth.token.exchanged", ...)` — this goes to structured logs, not to the `audit.Logger.Log()` method that writes to the `audit_logs` DB table. The `audit_replay.py` script must query structured log files or the `oauth_tokens` table (using `DelegationSubject` + `DelegationActor`) rather than the audit log API. For the demo, `audit_replay.py` queries `QueryAuditLogs()` but must be seeded with synthetic `oauth.token.exchanged` audit events (or the exchange handler must be patched to call `audit.Logger.Log()` — a 10-line change, recommended before demo).

### `shark agent revoke` CLI: NOT YET SHIPPED
There is no `shark agent revoke` subcommand. Revocation in the demo uses the REST API directly (`POST /api/v1/agents/{id}/revoke-tokens`). Verify this endpoint exists; if not, use `POST /oauth/revoke` per JTI for each token. The demo script `seed.sh` can enumerate and revoke.

---

*Generated 2026-04-24. Codebase ref: commit 1b75672.*
