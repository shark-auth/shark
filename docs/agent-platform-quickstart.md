# SharkAuth — Agent Platform Quickstart

## Who this is for

You're building a product where each customer gets their own agents — research
assistants, coding tools, customer-support bots, vertical AI SaaS that ships
an agent per workspace. Think Lovable, Cursor-like products, or any full-stack
app where your users don't just use AI — they _own_ agents that act on their
behalf.

SharkAuth is the auth layer that makes those agents safe to ship:

- Every agent gets a scoped OAuth 2.1 identity, not a shared API key.
- Every token is DPoP-bound (RFC 9449) — useless outside the agent's process.
- Every act-on-behalf-of hop is logged (RFC 8693 `act` claim audit trail).
- When a customer cancels, one call kills every token their agents ever held.

**Not your audience:** If you're wiring up a single MCP server for internal
use, SharkAuth still works — but this doc is written for teams shipping
multi-tenant agent products at scale. MCP server developers: the integration
is identical; start at Step 3.

---

## The 3-minute mental model

```
  Your customer
      │
      ▼
  Your app  ──────────────────────────────────────────────────┐
  (issues user session)                                       │
      │                                                  SharkAuth
      ▼                                                  • issues scoped tokens
  Customer's agent  ──── requests DPoP-bound token ────►• validates DPoP proof
  (per-customer identity                                 • logs act claim
   in SharkAuth)                                         • enforces scopes
      │                                                  └────────────────────
      ▼
  External resource
  (GitHub API, Slack, database, another agent…)
```

Shark sits at every boundary:

1. **App → SharkAuth**: your backend provisions an agent identity when a
   customer signs up. SharkAuth returns a `client_id` + `client_secret`.
2. **Agent → SharkAuth**: before each outbound call, the agent requests a
   short-lived, DPoP-bound access token. The DPoP keypair lives only in the
   agent's memory — token theft is useless without the private key.
3. **Agent → external resource**: the agent presents the DPoP-bound token.
   The resource server validates the token (via SharkAuth's JWKS endpoint)
   and the DPoP proof.
4. **Agent → agent (delegation)**: via RFC 8693 token exchange, agent B can
   act on behalf of agent A. Every hop adds an `act` claim — full chain
   visible in the audit log and on the `/admin/delegation-chains` canvas.

---

## 5-minute integration

### Step 1 — Install and run SharkAuth

```bash
# Download the binary from GitHub Releases
# https://github.com/<your-org>/shark/releases

./shark serve
# SharkAuth running on http://localhost:8080
# Admin key: sk_live_<...>  (also written to data/admin.key.firstboot)
# Dashboard: http://localhost:8080/admin
```

On first boot, SharkAuth writes a single-use admin key to
`data/admin.key.firstboot`. Open the dashboard and paste that key.

To print it again:

```bash
shark admin-key show
```

### Step 2 — Register your application

In the dashboard: open `/admin/applications` → "New Application" → copy the
`app_id`.

Or via CLI:

```bash
shark app create --name "My SaaS"
# app_id: app_abc123
```

You'll need `app_id` when provisioning per-customer agents in Step 3.

### Step 3 — Provision an agent when a customer signs up

When a customer creates an account in your product, call SharkAuth to create
an agent identity scoped to that customer:

```python
from shark_auth import AgentsClient

agents = AgentsClient(
    base_url="http://localhost:8080",
    token="sk_live_<admin-key>",
)

agent = agents.register_agent(
    app_id="app_abc123",
    name=f"agent-{customer_id}",
    scopes=["read:data", "write:data"],
)

# Store these in your DB — client_secret is shown once
client_id     = agent["client_id"]      # "shark_agent_..."
client_secret = agent["client_secret"]  # one-time secret
agent_id      = agent["id"]             # "agent_..."
```

`AgentsClient.register_agent` wraps `POST /api/v1/agents`. The `app_id` is
stored in the agent's metadata so you can list or bulk-revoke by application.

### Step 4 — Agent requests a DPoP-bound token before each external call

Your agent runtime (wherever it runs) uses `exchange_token` to get a fresh,
DPoP-bound token before calling an external resource:

```python
from shark_auth import DPoPProver, exchange_token

# Generate a DPoP keypair once per agent process (or rotate as needed)
prover = DPoPProver()

token_response = exchange_token(
    auth_url="http://localhost:8080",
    client_id=client_id,
    subject_token=user_access_token,   # the user's session token
    client_secret=client_secret,
    scope="read:data",
    dpop_prover=prover,                # binds token to this keypair
)

access_token = token_response.access_token
# Token is now DPoP-bound: useless without prover's private key
```

`exchange_token` implements RFC 8693 token exchange with optional DPoP binding
(RFC 9449). The resulting token carries an `act` claim linking agent to user —
every hop is logged.

### Step 5 — When a customer cancels, revoke all their agents in one call

```python
from shark_auth import UsersClient

users = UsersClient(
    base_url="http://localhost:8080",
    token="sk_live_<admin-key>",
)

result = users.revoke_agents(
    user_id="usr_abc",
    reason="customer cancelled subscription",
)

print(result.revoked_agent_ids)      # every agent this user owned
print(result.revoked_consent_count)  # OAuth consents also cleaned up
print(result.audit_event_id)         # traceable in audit log
```

`UsersClient.revoke_agents` wraps `POST /api/v1/users/{id}/revoke-agents`.
This is Layer 3 (user-cascade) in the five-layer revocation model below.

---

## The wedge: five-layer revocation

Most auth systems give you one revoke primitive: "invalidate this token." That
works for human sessions. It doesn't work when one customer can have dozens of
agents each holding multiple tokens, some delegating to sub-agents, some with
vault connections to third-party APIs.

SharkAuth's revocation model has five layers:

**diagram: 5 revocation layers**

### L1 — Individual token revoke

Revoke a single access token by ID. Use when you know exactly which token was
compromised.

```bash
# Dashboard: /admin/agents → agent row → Tokens tab → revoke individual token
# API: DELETE /oauth/revoke  (RFC 7009)
```

`AgentsClient.list_tokens(agent_id)` returns all active tokens with their
`token_id`, `jkt` (DPoP thumbprint), scope, and expiry so you can target
exactly the right one.

### L2 — Agent-wide revoke

Revoke all tokens for one agent. Use when an agent's credentials are rotated
or the agent is decommissioned.

```python
agents.revoke_agent(agent_id="agent_abc")
# Wraps DELETE /api/v1/agents/{id}
# Deactivates the agent + revokes all its tokens atomically
```

### L3 — User-cascade revoke

Revoke every agent created by a user and all OAuth consents they granted. Use
when a customer cancels or is offboarded. See Step 5 above.

`UsersClient.revoke_agents` is the single call that handles the entire
customer offboarding path. One API call, one audit event, complete.

### L4 — Bulk pattern revoke (v0.2, W18-W19)

**v0.2 (W18-W19)** — Revoke tokens matching a pattern: all agents in an
`app_id`, all tokens with a given scope, all tokens issued before a date. Use
for incident response ("revoke everything issued before we rotated the signing
key") or application-level offboarding.

### L5 — Vault disconnect cascade (v0.2, W18-W19)

**v0.2 (W18-W19)** — When a vault connection to a third-party service
(GitHub, Slack, a database) is disconnected, SharkAuth automatically revokes
all tokens that relied on that vault credential. Disconnecting the vault
entry propagates to every agent that had access.

---

## Code examples

### Provision an agent for a new customer

```python
from shark_auth import AgentsClient

agents = AgentsClient(base_url="http://localhost:8080", token="sk_live_...")

agent = agents.register_agent(
    app_id="app_abc123",
    name=f"research-agent-{customer_id}",
    scopes=["read:documents", "write:notes"],
    description="Per-customer research assistant",
)

# Persist in your DB
store_agent(
    customer_id=customer_id,
    shark_agent_id=agent["id"],
    client_id=agent["client_id"],
    client_secret=agent["client_secret"],  # one-time — store encrypted
)
```

### Get a DPoP-bound token before an external call

```python
from shark_auth import DPoPProver, exchange_token

prover = DPoPProver()  # generates ES256 keypair

resp = exchange_token(
    auth_url="http://localhost:8080",
    client_id=agent_client_id,
    subject_token=user_access_token,
    client_secret=agent_client_secret,
    scope="read:documents",
    dpop_prover=prover,
)

# Use resp.access_token in Authorization header
# Attach DPoP proof header via prover.proof(htm="GET", htu=resource_url)
print(resp.access_token)
print(resp.expires_in)
```

### Revoke all tokens for a departing customer

```python
from shark_auth import UsersClient

users = UsersClient(base_url="http://localhost:8080", token="sk_live_...")

result = users.revoke_agents(
    user_id=customer_shark_user_id,
    reason="subscription cancelled",
)

# result.revoked_agent_ids  — list of decommissioned agents
# result.revoked_consent_count — OAuth consents cleaned up
# result.audit_event_id — traceable in /admin/audit
```

---

## Comparison

| Capability | SharkAuth | Auth0 | Clerk | Stytch | Build your own |
|---|---|---|---|---|---|
| Agent-native primitives (per-agent identity, scopes, audit) | Yes | No | No | No | Manual |
| DPoP support (RFC 9449) | Yes | No | No | No | Hard |
| Token exchange / act-on-behalf-of (RFC 8693) | Yes | No | No | No | Very hard |
| Self-hosted, single binary | Yes | No | No | No | Yes |
| Audit trail per delegation hop | Yes | No | No | No | Manual |
| Five-layer revocation | Yes | L1 only | L1 only | L1 only | Manual |
| Price model | Apache-2.0, free | $23+/mo (machine clients) | Per-MAU | Per-MAU | Engineering cost |

Auth0, Clerk, and Stytch are built for human identity. They treat agents as
machine clients with static secrets. No DPoP support. No delegation chain
primitives. No act-on-behalf-of audit. Retrofitting agent semantics onto those
systems is the work SharkAuth already did.

---

## Where to next

**Dashboard surfaces:**
- `/admin/agents` — view, create, and manage agent identities
- `/admin/delegation-chains` — visualize RFC 8693 delegation chains as a
  directed graph with per-hop audit events
- `/admin/vault` — manage third-party credential connections with cascade
  revocation

**CLI demo:**
```bash
shark demo delegation-with-trace
# Runs a 3-hop delegation chain, prints DPoP proofs + act claims at each step
# Then open /admin/delegation-chains to see the canvas
```

**Install SDK (dogfood mode — PyPI release after internal validation):**
```bash
pip install git+https://github.com/<your-org>/shark#subdirectory=sdk/python
# PyPI release coming after dogfood validation
```

**Full feature reference:** `README.md`

**OpenAPI spec + interactive docs:** `http://localhost:8080/api/docs` (Scalar UI)
