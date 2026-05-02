# Internal Platform — AI platform with compliance audit trails

**Use-case:** Your engineering or ops team runs an internal AI platform. Multiple agents run workloads on behalf of human users. Compliance requires knowing who authorized what, when, and through which delegation chain.

SharkAuth gives you:
- Every token issuance, exchange, and revocation logged to a queryable audit log
- RFC 8693 token exchange chains — structured proof of who-acted-on-behalf-of-whom
- Webhook events for streaming to your security tooling (SIEM, Slack, PagerDuty)
- Bulk revocation patterns for incident response (v0.2 roadmap — see below)

## Architecture

```
human user (engineer)
  └─ authenticates → SharkAuth (issues user token)
       └─ agent-A requests token exchange → scoped sub-token with act chain
            └─ agent-B requests further exchange → deeper act chain
                 └─ resource server validates full chain via JWKS
```

The audit log records every hop. The delegation canvas in the dashboard visualizes the chain.

**screenshot: delegation canvas showing a 3-hop chain with audit events per hop**

## Step 1 — Start the server

```bash
shark serve
```

For production, configure persistence and SMTP:

```bash
shark serve --db ./data/shark.db --smtp-host mail.internal --smtp-port 587
```

See [`/documentation/cli/serve.md`](/documentation/cli/serve.md) for full options.

## Step 2 — Provision platform agents

Register one agent per workload type. Tag them with your internal app ID for filtering.

```python
from shark_auth import Client

admin = Client(base_url="http://shark.internal:8080", token="sk_live_...")

# Register agents for each workload
summarizer = admin.agents.register_agent(
    app_id="app_internal_platform",
    name="summarizer-agent",
    scopes=["docs:read"],
)
executor = admin.agents.register_agent(
    app_id="app_internal_platform",
    name="executor-agent",
    scopes=["docs:read", "code:execute"],
)
```

## Step 3 — Token exchange chain

Agent A holds a user-delegated token. It exchanges it for a narrower sub-token before handing work to Agent B. Each exchange records an `act` claim hop.

```python
from shark_auth import DPoPProver, OAuthClient

SHARK_URL = "http://shark.internal:8080"

# Agent A — has a user-delegated token (see guide 11 for how user obtains it)
prover_a = DPoPProver.generate()
oauth     = OAuthClient(base_url=SHARK_URL)

# Exchange: agent A narrows scope for agent B
token_b = oauth.token_exchange(
    subject_token=user_delegated_token,
    dpop_prover=prover_a,
    scope="docs:read",                       # narrower than original grant
    audience="https://docs.internal",         # RFC 8707 audience lock
)
# token_b.act["sub"] == agent_a_client_id
# token_b.act["act"]["sub"] == human_user_id  (the original principal)
```

See [11 — Delegation Chains](./11-delegation-chains.md) for the full 3-hop walkthrough.

## Step 4 — Query the audit log

Every event is queryable. Pull a specific agent's history:

```python
events = admin.agents.get_audit_logs("shark_agent_summarizer", limit=50)
for ev in events:
    print(f"{ev.created_at}  {ev.event:30s}  actor={ev.actor_id}")
```

Or hit the API directly for cross-agent queries:

```bash
curl -H "Authorization: Bearer sk_live_..." \
  "http://shark.internal:8080/api/v1/audit-logs?limit=100&since=2026-04-26T00:00:00Z"
```

Event types relevant to compliance:

| Event | Meaning |
|---|---|
| `oauth.token_issued` | New token minted — includes `scope`, `jkt`, `audience` |
| `oauth.token_exchanged` | RFC 8693 exchange — includes `actor_id`, `subject_id`, new scope |
| `oauth.token_revoked` | Individual token revoked |
| `agent.tokens_revoked_all` | All tokens for an agent nuked |
| `user.agents_revoked` | Cascade: all user's agents and tokens revoked |
| `agent.dpop_key_rotated` | DPoP key rotation — old key tokens gone |

See [`../sdk/audit-logs.md`](../sdk/audit-logs.md) for the full event schema.

## Step 5 — Webhook to security tooling

Subscribe to `oauth.token.exchanged` (and others) to stream events to your SIEM or Slack channel.

In the dashboard: **Settings → Webhooks → Add endpoint**. Or via API:

```bash
curl -X POST -H "Authorization: Bearer sk_live_..." \
  -H "Content-Type: application/json" \
  "http://shark.internal:8080/api/v1/admin/webhooks" \
  -d '{
    "url": "https://hooks.slack.com/...",
    "events": ["oauth.token.exchanged", "user.agents_revoked", "agent.dpop_key_rotated"]
  }'
```

**screenshot: webhook configuration panel with event filter checkboxes**

## Step 6 — Incident response: revoke a specific agent's tokens

If an agent is suspected compromised:

```python
# L2: revoke all tokens for a single agent
result = admin.agents.revoke_all("shark_agent_executor")
print(f"Revoked {result.revoked_count} tokens")

# L2-b: rotate the DPoP key — old-key tokens gone, agent gets a fresh binding
new_prover = DPoPProver.generate()
rotation   = admin.agents.rotate_dpop_key(
    "shark_agent_executor",
    new_public_key_jwk=new_prover.public_key_jwk(),
    reason="incident-2026-04-26",
)
print(f"Old JKT: {rotation.old_jkt} → New JKT: {rotation.new_jkt}")
print(f"Audit event: {rotation.audit_event_id}")
```

## Roadmap: bulk pattern revocation (v0.2)

Layer 4 bulk revocation will let you kill all tokens matching a GLOB pattern — useful for rolling back a compromised agent version across all instances:

```python
# v0.2 — bulk_revoke_by_pattern ships in Wave 1.6
result = admin.oauth.bulk_revoke_by_pattern(
    client_id_pattern="shark_agent_executor_v2.*",
    reason="CVE-2026-xxxx rollback",
)
# result.revoked_count tokens gone across all matching client_ids
```

The method exists in the SDK today; the backend endpoint ships in v0.2. See [`10-five-layer-revocation.md`](./10-five-layer-revocation.md) for Layer 4 detail.

## Inspecting active tokens per agent

```python
tokens = admin.agents.list_tokens("shark_agent_summarizer")
for tok in tokens:
    print(f"  id={tok.token_id}  scope={tok.scope}  jkt={tok.jkt}  expires={tok.expires_at}")
```

## Next steps

- Full revocation model (all 5 layers): [10 — Five-Layer Revocation](./10-five-layer-revocation.md)
- Delegation chains in depth: [11 — Delegation Chains](./11-delegation-chains.md)
- Audit log API reference: [`../sdk/audit-logs.md`](../sdk/audit-logs.md)
- Webhook API reference: [`../sdk/webhooks.md`](../sdk/webhooks.md)
