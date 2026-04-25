# Demo #1 — AI Agent Marketplace Backbone

**Theme:** A multi-tenant platform where end-users connect autonomous AI agents
that act on their behalf across Gmail, Slack, GitHub, and Notion — with
cryptographic delegation chains, full audit, and one-click revocation.

**Duration:** 10 minutes live  
**Vertical:** Personal AI Ops / B2B Productivity Agent Stack  
**Binary:** `sharkauth` (single-process, SQLite, no external dependencies)

---

## Story

Marcus is a founder at a 12-person startup. He pays for five AI agents: an
inbox assistant (reads/replies Gmail), a standup bot (posts to Slack), a PR
reviewer (comments on GitHub), a meeting scheduler (writes to Notion), and an
orchestrator agent (Claude) that coordinates all four when he asks "handle my
morning routine." Each agent was connected via OAuth — meaning Marcus clicked
"Authorize" five times and handed out five long-lived tokens that never expire,
have no scope limits, and can't be traced back to a specific task.

Last Tuesday the PR reviewer agent's API key leaked in a public repo. Marcus
had to rotate credentials in four different dashboards, unsure which tokens were
poisoned and which weren't. He still doesn't know if the agent accessed
anything it shouldn't have.

SharkAuth replaces all of that. Every agent gets a DPoP-bound, audience-locked
JWT. The orchestrator delegates to sub-agents via RFC 8693 token exchange,
building a cryptographically signed `act` chain. Every hop is audited. Marcus
revokes the PR reviewer in one click — nothing else breaks.

---

## Architecture Diagram

```
Marcus (browser)
    │
    │  1. User authorizes orchestrator agent
    │     GET /oauth/authorize (PKCE S256, RFC 7636)
    │     → consent screen → code → /oauth/token
    │     ← JWT₀: sub=marcus, aud=orchestrator, scope=read:gmail write:slack
    │        cnf.jkt=<orchestrator-pubkey-thumbprint>  (RFC 9449 DPoP)
    ▼
┌─────────────────────────────────────────────────────┐
│               SharkAuth (single binary)              │
│  /.well-known/oauth-authorization-server  (RFC 8414) │
│  /.well-known/jwks.json                  (ES256 pub) │
│  /oauth/register  (RFC 7591 DCR)                     │
│  /oauth/authorize / /oauth/token                     │
│  /oauth/token  (RFC 8693 exchange, RFC 8707 aud)     │
│  /oauth/device  (RFC 8628 headless)                  │
│  /oauth/revoke  (RFC 7009)                           │
│  /api/v1/admin/agents                                │
│  /api/v1/admin/audit-logs                            │
│  /api/v1/admin/sessions                              │
│  /api/v1/admin/vault  (Token Vault)                  │
│  /api/v1/vault/{provider}/token  (agent token fetch) │
└──────────────────────┬──────────────────────────────┘
                       │
       ┌───────────────┼──────────────────────┐
       │               │                      │
       ▼               ▼                      ▼
 ┌───────────┐   ┌───────────┐         ┌───────────┐
 │Orchestrat-│   │ Headless  │         │  Device   │
 │or Agent   │   │ Device    │         │  Flow     │
 │(Claude)   │   │ Agent     │         │  Agent    │
 │           │   │ (RFC 8628)│         │  (IoT/CLI)│
 └─────┬─────┘   └─────┬─────┘         └───────────┘
       │               │
       │  2. Orchestrator exchanges its JWT₀
       │     for a narrower, audience-locked
       │     token for the Gmail agent
       │
       │  POST /oauth/token
       │    grant_type=urn:ietf:params:oauth:
       │                grant-type:token-exchange
       │    subject_token=JWT₀
       │    requested_token_type=access_token
       │    audience=gmail-agent
       │    scope=read:gmail          ← scope-narrowed (RFC 8693)
       │    resource=https://gmail.example.com  (RFC 8707)
       │
       │  ← JWT₁: sub=marcus, aud=gmail-agent
       │           act={"sub":"orchestrator"}
       │           scope=read:gmail  (ONLY)
       │           cnf.jkt=<gmail-agent-pubkey>
       │
       ▼
 ┌───────────┐         ┌───────────┐         ┌─────────────┐
 │ Gmail     │         │ Slack     │         │  GitHub     │
 │ Agent     │         │  Bot      │         │  PR Review  │
 │           │         │           │         │  Agent      │
 │ JWT₁      │         │ JWT₂      │         │ JWT₃        │
 │ scope:    │         │ scope:    │         │ scope:      │
 │ read:gmail│         │write:slack│         │read:github  │
 │ aud:gmail │         │aud:slack  │         │aud:github   │
 │ act:orch  │         │ act:orch  │         │ act:orch    │
 └─────┬─────┘         └─────┬─────┘         └──────┬──────┘
       │                     │                       │
       │  Vault fetch        │  Vault fetch          │  Vault fetch
       │  GET /vault/        │  GET /vault/          │  GET /vault/
       │  google/token       │  slack/token          │  github/token
       ▼                     ▼                       ▼
  Gmail API              Slack API              GitHub API
  (Bearer = vault        (Bearer = vault        (Bearer = vault
   access_token)          access_token)          access_token)
```

**Token types at each hop:**

| Hop | Token | Key Claims |
|-----|-------|-----------|
| User → Orchestrator | JWT₀ (ES256) | sub=user, aud=orchestrator, scope=full, cnf.jkt |
| Orchestrator → Gmail Agent | JWT₁ (ES256) | sub=user, aud=gmail-agent, scope=read:gmail ONLY, act.sub=orchestrator, cnf.jkt |
| Orchestrator → Slack Bot | JWT₂ (ES256) | sub=user, aud=slack-bot, scope=write:slack ONLY, act.sub=orchestrator, cnf.jkt |
| Agent → 3rd-party API | Vault token | Provider-issued OAuth token, fetched encrypted-at-rest |

---

## Shark Feature Map

| Demo Step | Shark Feature | RFC | File / Endpoint |
|-----------|--------------|-----|-----------------|
| 1. MCP client discovers auth server | Authorization Server Metadata | RFC 8414 | `GET /.well-known/oauth-authorization-server` → `oauth/metadata.go` |
| 2. Gmail agent self-registers | Dynamic Client Registration | RFC 7591 | `POST /oauth/register` → `oauth/dcr.go HandleDCRRegister` |
| 3. User authorizes orchestrator | Auth Code + PKCE + DPoP | RFC 7636, 9449 | `GET /oauth/authorize` → `oauth/handlers.go HandleAuthorize` |
| 4. Orchestrator gets DPoP JWT | Token issuance w/ cnf.jkt | RFC 9449, 7519 | `POST /oauth/token` → `oauth/handlers.go HandleToken`, `oauth/dpop.go ValidateDPoPProof` |
| 5. Orchestrator delegates to Gmail agent | Token Exchange + scope narrowing | RFC 8693, 8707 | `POST /oauth/token` (grant_type=token-exchange) → `oauth/handlers.go HandleToken` |
| 6. Headless CLI agent gets token | Device Authorization Grant | RFC 8628 | `POST /oauth/device` → admin approves at `/oauth/device/verify` |
| 7. Agent fetches Google token from Vault | Token Vault encrypted retrieval | — | `GET /api/v1/vault/google/token` → `internal/vault/providers.go` |
| 8. View delegation chain in Audit | Audit log with act chain | — | `GET /api/v1/admin/audit-logs` → Admin UI → Audit tab |
| 9. View live sessions | Session list per agent | — | `GET /api/v1/admin/sessions` → Admin UI → Sessions tab |
| 10. One-click revoke PR agent | Token revocation by jti | RFC 7009 | `POST /oauth/revoke` → `oauth/revoke.go`; Admin UI → Agents tab → Deactivate |
| 11. Verify other agents unaffected | Per-token isolation | RFC 8707 | Audience binding; other JWTs have different aud — unaffected |
| 12. Decode JWT locally (no introspection) | Local JWT verification | RFC 7519 | `shark_auth.decode_agent_token(token, jwks_url, issuer, audience)` |

---

## Demo Script (Live Walkthrough — 10 Minutes)

### Pre-flight (done before audience arrives)
```bash
# Start SharkAuth
sharkauth serve --config sharkauth.yaml

# Confirm health
curl http://localhost:8080/api/v1/health

# Seed demo data (run demos/agent_marketplace/seed.sh)
# Creates: user:marcus, 4 agents, 3 vault connections
```

---

### Act 1 — "The World Today" (~2 min)

**[Slide or whiteboard]** Show the current state: 5 agents, 5 long-lived tokens,
zero auditability, one leaked key = manual credential rotation across 4 dashboards.

"This is Zapier AI, Lindy, Pylon, every other agent platform today. You hand
out tokens, you pray."

---

### Act 2 — Agent Discovery + Self-Registration (~2 min)

**[Terminal 1]**

```bash
# Step 1: MCP client hits discovery (RFC 8414)
curl -s http://localhost:8080/.well-known/oauth-authorization-server | jq .

# Audience sees:
# {
#   "issuer": "http://localhost:8080",
#   "authorization_endpoint": "http://localhost:8080/oauth/authorize",
#   "token_endpoint": "http://localhost:8080/oauth/token",
#   "registration_endpoint": "http://localhost:8080/oauth/register",
#   "jwks_uri": "http://localhost:8080/.well-known/jwks.json",
#   "grant_types_supported": ["authorization_code","client_credentials",
#     "urn:ietf:params:oauth:grant-type:token-exchange","device_code","refresh_token"],
#   "dpop_signing_alg_values_supported": ["ES256"]
# }

# Step 2: Gmail agent self-registers (RFC 7591 DCR) — no manual config
curl -s -X POST http://localhost:8080/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "gmail-agent",
    "redirect_uris": ["http://localhost:9001/callback"],
    "grant_types": ["authorization_code","urn:ietf:params:oauth:grant-type:token-exchange","refresh_token"],
    "scope": "read:gmail write:gmail",
    "token_endpoint_auth_method": "client_secret_basic"
  }' | jq '{client_id, client_name, grant_types}'
```

**Talking point:** "The agent registered itself. No Terraform, no admin portal,
no ticket to IT. That's RFC 7591 Dynamic Client Registration. SharkAuth is
MCP-spec compliant out of the box."

---

### Act 3 — User Authorizes Orchestrator with DPoP (~2 min)

**[Terminal 2 — Python SDK or curl]**

```bash
# Step 3: Orchestrator generates ES256 keypair (DPoP, RFC 9449)
# demos/agent_marketplace/gen_dpop_key.py — outputs JWK + thumbprint

# Step 4: User grants orchestrator access (simulated consent)
# In real demo: browser opens /oauth/authorize, user clicks Approve
# For CLI demo, use client_credentials shortcut:
ORCH_TOKEN=$(curl -s -X POST http://localhost:8080/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: <dpop-proof-jwt>" \
  -u "orchestrator-client-id:secret" \
  -d "grant_type=client_credentials&scope=read:gmail+write:slack+read:github&resource=http://localhost:8080" \
  | jq -r '.access_token')

# Step 5: Decode it locally — NO introspection round-trip
python3 -c "
import shark_auth
claims = shark_auth.decode_agent_token(
    '$ORCH_TOKEN',
    jwks_url='http://localhost:8080/.well-known/jwks.json',
    expected_issuer='http://localhost:8080',
    expected_audience='http://localhost:8080'
)
import json; print(json.dumps(claims, indent=2))
"
```

**Audience sees claims including `cnf.jkt` — the DPoP key thumbprint.**

"This token is cryptographically bound to the orchestrator's private key.
Even if someone steals the token, they can't use it — they don't have the key.
That's RFC 9449 DPoP. Zero extra infrastructure."

---

### Act 4 — Delegation Chain: Orchestrator → Gmail Agent (~2 min)

**[Terminal 2 — THE WOW MOMENT]**

```bash
# Step 6: Orchestrator exchanges its token for a NARROWER gmail-agent token
# RFC 8693 Token Exchange + RFC 8707 Resource Indicators
GMAIL_TOKEN=$(curl -s -X POST http://localhost:8080/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: <gmail-agent-dpop-proof>" \
  -u "gmail-agent-client-id:secret" \
  -d "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Atoken-exchange\
&subject_token=$ORCH_TOKEN\
&subject_token_type=urn%3Aietf%3Aparams%3Aoauth%3Atoken-type%3Aaccess_token\
&requested_token_type=urn%3Aietf%3Aparams%3Aoauth%3Atoken-type%3Aaccess_token\
&audience=gmail-agent\
&scope=read%3Agmail\
&resource=https%3A%2F%2Fgmail.example.com" \
  | jq -r '.access_token')

# Decode the delegated token
python3 -c "
import shark_auth, json
claims = shark_auth.decode_agent_token('$GMAIL_TOKEN',
    'http://localhost:8080/.well-known/jwks.json',
    'http://localhost:8080', 'gmail-agent')
print(json.dumps(claims, indent=2))
"
```

**THE WOW OUTPUT — audience sees:**
```json
{
  "sub": "user_marcus_001",
  "aud": "gmail-agent",
  "scope": "read:gmail",
  "cnf": { "jkt": "a7b3c9d..." },
  "act": {
    "sub": "orchestrator",
    "act": {
      "sub": "user_marcus_001"
    }
  },
  "exp": 1745798400,
  "jti": "tok_7x9mK..."
}
```

**"Look at the `act` chain. `user_marcus_001` delegated to `orchestrator` which
delegated to `gmail-agent`. Scope narrowed from full access down to `read:gmail`
ONLY. Audience locked to `gmail-agent` — this token is useless against Slack.
All cryptographically verifiable. No database lookup. No API call.
This is what enterprise-grade agentic auth looks like."**

---

### Act 5 — Audit + One-Click Revocation (~2 min)

**[Admin Dashboard — Browser]**

```
Tab: Admin Dashboard → Audit Logs
  → Filter: actor=orchestrator
  → See: token_exchange, token_issued, vault_token_retrieved events
  → Each entry: timestamp, actor (with act chain), action, resource, IP

Tab: Admin Dashboard → Agents
  → Row: "github-pr-agent" — status: active
  → Click: Deactivate
  → Confirm modal

Tab: Admin Dashboard → Sessions
  → github-pr-agent sessions: 0 active (all revoked)
  → gmail-agent sessions: 1 active (UNAFFECTED)
  → slack-bot sessions: 1 active (UNAFFECTED)
```

**[Terminal — verify revocation]**
```bash
# Attempt to use the now-revoked github-pr-agent token
curl -s -X POST http://localhost:8080/oauth/introspect \
  -u "github-pr-agent-client-id:secret" \
  -d "token=$GITHUB_TOKEN" | jq .active
# → false

# Gmail agent token still valid
curl -s -X POST http://localhost:8080/oauth/introspect \
  -u "gmail-agent-client-id:secret" \
  -d "token=$GMAIL_TOKEN" | jq .active
# → true
```

"One button. GitHub agent: dead. Everything else: untouched. No cascading
failures. No 2am credential rotation. That's audience-bound tokens plus
per-jti revocation."

---

## Implementation Plan

### Directory structure to create
```
demos/
  agent_marketplace/
    seed.sh              — creates user marcus, 4 agents, vault connections via admin API
    gen_dpop_key.py      — generates ES256 JWK, prints thumbprint + DPoP proof JWT
    orchestrate.py       — full delegation chain: DCR → authorize → exchange → vault fetch
    revoke_agent.sh      — calls /oauth/revoke for github-pr-agent, shows others unaffected
    verify_chain.py      — decodes JWT₀, JWT₁, JWT₂ and prints act chain comparison
```

### Endpoints used (all real, no mocking)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/.well-known/oauth-authorization-server` | GET | Discovery (RFC 8414) |
| `/.well-known/jwks.json` | GET | ES256 public key for local verify |
| `/oauth/register` | POST | Agent self-registration (RFC 7591) |
| `/oauth/authorize` | GET | User consent + PKCE (RFC 7636) |
| `/oauth/token` | POST | Token issuance + exchange (RFC 8693) + DPoP (RFC 9449) |
| `/oauth/device` | POST | Headless agent bootstrap (RFC 8628) |
| `/oauth/device/verify` | GET | Admin approves device flow |
| `/oauth/revoke` | POST | Per-token revocation (RFC 7009) |
| `/oauth/introspect` | POST | Verify active/revoked state (RFC 7662) |
| `/api/v1/vault/google/token` | GET | Fetch encrypted Google OAuth token |
| `/api/v1/vault/slack/token` | GET | Fetch encrypted Slack OAuth token |
| `/api/v1/vault/github/token` | GET | Fetch encrypted GitHub OAuth token |
| `/api/v1/admin/agents` | GET/PATCH | Agent listing + deactivation |
| `/api/v1/admin/audit-logs` | GET | Event stream with act chain |
| `/api/v1/admin/sessions` | GET/DELETE | Session view + revoke-all |

### Seeded data (seed.sh)

```bash
BASE=http://localhost:8080
ADMIN_KEY=<from sharkauth.yaml>

# 1. Create user marcus
curl -s -X POST $BASE/api/v1/admin/users \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -d '{"email":"marcus@acme.com","password":"demo1234","name":"Marcus Chen"}'

# 2. Create orchestrator agent
curl -s -X POST $BASE/oauth/register \
  -d '{"client_name":"orchestrator","grant_types":["client_credentials",
       "urn:ietf:params:oauth:grant-type:token-exchange"],"scope":"read:gmail write:slack read:github write:notion"}'

# 3. Create gmail-agent, slack-bot, github-pr-agent, notion-agent (same pattern)

# 4. Add vault connections (Google, Slack, GitHub) via admin vault API
curl -s -X POST $BASE/api/v1/admin/vault \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -d '{"name":"marcus-google","provider":"google","user_id":"<marcus_id>",
       "access_token":"ya29.demo_token","refresh_token":"1//demo_refresh",
       "expires_at":"2026-12-31T00:00:00Z"}'
```

### Shark CLI calls
```bash
shark serve --config demos/agent_marketplace/sharkauth.demo.yaml
shark health
shark app list --json
```

---

## The "Wow" Moment

**The single screen that lands:**

Terminal showing the decoded `GMAIL_TOKEN` JWT payload side-by-side with the
decoded `ORCH_TOKEN` — the audience sees in one glance:

```
ORCHESTRATOR TOKEN              →   GMAIL AGENT TOKEN
──────────────────────              ─────────────────────
sub: user_marcus_001                sub: user_marcus_001
aud: orchestrator                   aud: gmail-agent         ← LOCKED
scope: read:gmail write:slack       scope: read:gmail        ← NARROWED
       read:github write:notion
cnf.jkt: a7b3c9d...                 cnf.jkt: f2e8b1a...     ← DIFFERENT KEY
act: (none — top of chain)          act: {sub: orchestrator} ← CHAIN
                                         → {sub: user_marcus_001}
```

Then: one click in the admin UI to revoke `github-pr-agent`. Cut to terminal
showing `github-pr-agent` introspection returns `"active": false` while
`gmail-agent` returns `"active": true`.

**Why this lands:** The audience has never seen a token that is simultaneously
scoped, audience-locked, key-bound, and carries its own delegation provenance.
Every other platform shows them a string. SharkAuth shows them a contract.

---

## Sellable Angle

**One-liner:**
> "SharkAuth is the auth layer that makes AI agents enterprise-safe — every
> agent gets a cryptographically-bound, scope-narrowed, auditable token that
> you can revoke in one click without breaking anything else."

**Three customer types this lands with:**

| Customer | Pain Today | Shark Solves |
|----------|-----------|-------------|
| **Agent platform builders** (Lindy, Pylon, Bardeen-style) | Building homegrown token delegation is 6-month project; no DPoP, no act chain, no audit | Drop-in OAuth AS with all RFCs shipped; build marketplace in weeks not months |
| **Enterprise IT/Security buyers** | Shadow agents with untracked token sprawl; no way to audit "which agent accessed what on behalf of whom" | Full delegation audit trail with act chain; per-agent revocation; RBAC |
| **Developer-tool platforms** (AI-native SaaS, Claude/GPT integrations) | MCP spec requires RFC 9728 discovery + DCR; Auth0/Okta don't support it | SharkAuth is MCP-spec compliant out of the box; single binary, 5-minute setup |

---

## UAT Checklist

Run these before every demo. All must pass.

- [ ] **U1** — `GET /.well-known/oauth-authorization-server` returns `200` with
  `registration_endpoint` and `dpop_signing_alg_values_supported: ["ES256"]`
- [ ] **U2** — `POST /oauth/register` with DCR payload returns `201` with
  `client_id`, `client_secret`, and `registration_access_token`
- [ ] **U3** — `POST /oauth/token` with valid DPoP header returns JWT access token
  containing `cnf.jkt` claim matching DPoP key thumbprint (verify with
  `shark_auth.decode_agent_token`)
- [ ] **U4** — Token Exchange (`grant_type=token-exchange`) returns JWT with:
  - `aud` = requested audience
  - `scope` = narrowed scope (not broader than subject token)
  - `act.sub` = delegating agent client_id
- [ ] **U5** — `GET /api/v1/vault/google/token` returns non-empty `access_token`
  for a seeded vault connection
- [ ] **U6** — Admin UI → Audit Logs shows `token_exchange` events with actor
  and act chain visible
- [ ] **U7** — Admin UI → Agents tab shows all 4 agents with correct grant types
  and status = active
- [ ] **U8** — Admin UI → Deactivate `github-pr-agent` → `POST /oauth/introspect`
  with its token returns `"active": false`
- [ ] **U9** — After revoking `github-pr-agent`, gmail-agent token introspection
  still returns `"active": true` (isolation verified)
- [ ] **U10** — Device flow: `POST /oauth/device` returns `device_code` +
  `user_code`; admin approves at `/oauth/device/verify`; polling returns token
- [ ] **U11** — Decode JWT locally (no network call to introspect endpoint) via
  `shark_auth.decode_agent_token` with correct issuer/audience returns valid claims
- [ ] **U12** — `POST /oauth/revoke` by jti returns `200`; subsequent
  introspection returns `"active": false` for that specific token only

---

## Reference RFCs

| RFC | Feature | Shark endpoint |
|-----|---------|---------------|
| RFC 7591/7592 | Dynamic Client Registration | `/oauth/register`, `/oauth/register/{client_id}` |
| RFC 9449 | DPoP Proof-of-Possession | `/oauth/token` (DPoP header) + `cnf.jkt` in JWT |
| RFC 8693 | Token Exchange + delegation `act` chain | `/oauth/token` (grant_type=token-exchange) |
| RFC 8707 | Resource Indicators (audience binding) | `/oauth/token` (resource param) |
| RFC 8628 | Device Authorization Grant | `/oauth/device`, `/oauth/device/verify` |
| RFC 8414 | Authorization Server Metadata | `/.well-known/oauth-authorization-server` |
| RFC 7009 | Token Revocation | `/oauth/revoke` |
| RFC 7662 | Token Introspection | `/oauth/introspect` |
| RFC 9728 | MCP Protected Resource Metadata | `/.well-known/oauth-protected-resource` (MCP discovery) |
