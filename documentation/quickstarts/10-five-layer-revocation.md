# Five-Layer Revocation

SharkAuth's revocation model has five layers. Each layer is a broader blast radius than the last. The five together are the moat: Auth0 exposes one layer (individual token revoke). None of the others are available in any hosted auth provider today.

```
L1 — Individual token           → revoke one token
L2 — Agent-wide                 → revoke all tokens for one agent
L3 — User cascade               → revoke all agents + tokens for one customer
L4 — Bulk pattern               → revoke all tokens matching a GLOB across agents (v0.2)
L5 — Vault disconnect cascade   → revoke all agents that ever touched a vault connection (v0.2)
```

Each layer is one SDK call. Each call emits an audit event. Each audit event is queryable.

---

## L1 — Individual token revoke

**When:** a specific token is compromised, expired early, or a session needs to be killed.

```python
from shark_auth import OAuthClient

oauth = OAuthClient(base_url="http://localhost:8080", token="sk_live_...")
oauth.revoke_token("eyJhbGci...", token_type_hint="access_token")
# Server always returns 200 (RFC 7009 compliant).
# Token is immediately invalid — introspect will return { "active": false }.
```

Confirm via introspection:

```python
info = oauth.introspect_token("eyJhbGci...")
assert info["active"] is False
```

Expected audit log entry:

```json
{
  "event": "oauth.token_revoked",
  "actor_id": "sk_live_...",
  "target_id": "tok_abc123",
  "metadata": { "token_type_hint": "access_token" },
  "created_at": "2026-04-26T12:00:00Z"
}
```

**API reference:** [`/documentation/api/oauth.md#revoke`](/documentation/api/oauth.md)

---

## L2 — Agent-wide revoke

**When:** an agent is compromised or decommissioned. All active tokens for that agent — across all sessions, all scopes — are revoked in one call.

```python
from shark_auth import Client

admin = Client(base_url="http://localhost:8080", token="sk_live_...")

# Optional: inspect what will be revoked first
tokens = admin.agents.list_tokens("shark_agent_abc123")
print(f"Active tokens to revoke: {len(tokens)}")
for tok in tokens:
    print(f"  {tok.token_id}  scope={tok.scope}  jkt={tok.jkt}")

# Revoke all
result = admin.agents.revoke_all("shark_agent_abc123")
print(f"Revoked {result.revoked_count} tokens for agent {result.agent_id}")
```

Variant: rotate the DPoP key instead of revoking directly. All tokens bound to the old keypair are invalidated; the agent can re-authenticate with the new key.

```python
from shark_auth import DPoPProver

new_prover = DPoPProver.generate()
rotation   = admin.agents.rotate_dpop_key(
    "shark_agent_abc123",
    new_public_key_jwk=new_prover.public_key_jwk(),
    reason="key compromise detected 2026-04-26",
)
print(f"Old JKT: {rotation.old_jkt}")
print(f"New JKT: {rotation.new_jkt}")
print(f"Revoked: {rotation.revoked_token_count} tokens")
print(f"Audit:   {rotation.audit_event_id}")
```

Expected audit log entry:

```json
{
  "event": "agent.tokens_revoked_all",
  "actor_id": "sk_live_...",
  "target_id": "shark_agent_abc123",
  "metadata": { "revoked_count": 3 },
  "created_at": "2026-04-26T12:01:00Z"
}
```

**API reference:** [`/documentation/api/agents.md#revoke-all`](/documentation/api/agents.md)

---

## L3 — User cascade revoke

**When:** a customer cancels, is suspended, or their account is compromised. One call revokes all agents created by or authorized by that user, and all tokens across those agents.

This is the layer that Auth0 most conspicuously lacks. Without it, you must enumerate every agent, revoke each one, and hope you didn't miss one created by a background process.

```python
from shark_auth import Client

admin = Client(base_url="http://localhost:8080", token="sk_live_...")

# Optional: see which agents will be affected
agent_list = admin.users.list_agents("usr_alice", filter="all")
print(f"Agents to revoke: {agent_list.total}")
for agent in agent_list.data:
    print(f"  {agent['id']}  {agent['name']}")

# Cascade revoke — one call
result = admin.users.revoke_agents(
    "usr_alice",
    reason="customer cancelled subscription 2026-04-26",
)
print(f"Revoked agents:  {result.revoked_agent_ids}")
print(f"Revoked consent records: {result.revoked_consent_count}")
print(f"Audit event: {result.audit_event_id}")
```

Selective: revoke only specific agents under a user (still user-scoped, so no accidental cross-customer impact):

```python
result = admin.users.revoke_agents(
    "usr_alice",
    agent_ids=["shark_agent_abc", "shark_agent_def"],
    reason="specific agents compromised",
)
```

Expected audit log entry:

```json
{
  "event": "user.agents_revoked",
  "actor_id": "sk_live_...",
  "target_id": "usr_alice",
  "metadata": {
    "revoked_agent_ids": ["shark_agent_abc", "shark_agent_def"],
    "revoked_consent_count": 2,
    "reason": "customer cancelled subscription 2026-04-26"
  },
  "created_at": "2026-04-26T12:02:00Z"
}
```

**API reference:** [`/documentation/api/users.md#revoke-agents`](/documentation/api/users.md)

---

## L4 — Bulk pattern revoke (v0.2)

**When:** a specific agent version or agent type is compromised across all customers. Pattern matches on `client_id` using SQLite GLOB syntax.

The backend endpoint ships in v0.2 (Wave 1.6). The SDK method exists today.

```python
from shark_auth import Client

admin = Client(base_url="http://localhost:8080", token="sk_live_...")

# Kill all v3.2 agents across every customer — regardless of how many there are
result = admin.oauth.bulk_revoke_by_pattern(
    client_id_pattern="shark_agent_summarizer_v3.2_*",
    reason="CVE-2026-xxxx — v3.2 summarizer compromised, rolling back",
)
print(f"Revoked: {result.revoked_count} tokens")
print(f"Pattern: {result.pattern_matched}")
print(f"Audit:   {result.audit_event_id}")
```

Pattern syntax (SQLite GLOB):
- `*` — any sequence of characters
- `?` — any single character
- `[abc]` — character class

Examples:
- `shark_agent_v3.*` — all v3.x agents
- `shark_agent_*_prod_*` — all production agents
- `shark_agent_summarizer_?` — all single-suffix summarizer agents

Expected audit log entry (v0.2):

```json
{
  "event": "oauth.tokens_bulk_revoked",
  "actor_id": "sk_live_...",
  "metadata": {
    "pattern": "shark_agent_summarizer_v3.2_*",
    "revoked_count": 847,
    "reason": "CVE-2026-xxxx — v3.2 summarizer compromised"
  },
  "created_at": "2026-04-26T12:03:00Z"
}
```

**Status:** SDK method `OAuthClient.bulk_revoke_by_pattern()` ships in v0.1. Backend endpoint `POST /api/v1/admin/oauth/revoke-by-pattern` ships in v0.2.

---

## L5 — Vault disconnect cascade (v0.2)

**When:** a third-party OAuth connection stored in the vault is revoked at the provider (e.g. user revokes Google OAuth in their Google account). You need to immediately invalidate every agent that has ever fetched credentials from that connection.

```python
from shark_auth import VaultClient

vault = VaultClient(base_url="http://localhost:8080", admin_key="sk_live_...")

result = vault.disconnect(
    "conn_google_alice",
    cascade_to_agents=True,   # default — triggers L5 cascade
)
print(f"Disconnected: {result.connection_id}")
print(f"Cascade-revoked agents: {result.revoked_agent_ids}")
print(f"Revoked token count:    {result.revoked_token_count}")
print(f"Cascade audit event:    {result.cascade_audit_event_id}")
```

To disconnect without cascading (e.g. planned migration):

```python
vault.disconnect("conn_google_alice", cascade_to_agents=False)
```

Expected audit log entry (v0.2):

```json
{
  "event": "vault.connection_disconnected",
  "actor_id": "sk_live_...",
  "target_id": "conn_google_alice",
  "metadata": {
    "cascade": true,
    "revoked_agent_ids": ["shark_agent_abc", "shark_agent_def"],
    "revoked_token_count": 6
  },
  "created_at": "2026-04-26T12:04:00Z"
}
```

**Status:** `VaultClient.disconnect()` ships in v0.1. The cascade backend logic ships in v0.2.

---

## Summary: when to use each layer

| Layer | Trigger | Blast radius | SDK call |
|---|---|---|---|
| L1 | One session killed / token stolen | 1 token | `oauth.revoke_token()` |
| L2 | Agent compromised or decommissioned | All tokens for 1 agent | `agents.revoke_all()` or `agents.rotate_dpop_key()` |
| L3 | Customer cancels / account suspended | All agents + tokens for 1 user | `users.revoke_agents()` |
| L4 | Agent version compromised (v0.2) | All matching agents across all users | `oauth.bulk_revoke_by_pattern()` |
| L5 | 3rd-party OAuth revoked at provider (v0.2) | All agents that touched a vault connection | `vault.disconnect()` |

## Related

- Customer agent provisioning: [01 — Customer Agents](./01-customer-agents.md)
- Delegation chains (what's in the audit trail): [11 — Delegation Chains](./11-delegation-chains.md)
- Audit log API: [`/documentation/api/audit-logs.md`](/documentation/api/audit-logs.md)
- Agents API: [`/documentation/api/agents.md`](/documentation/api/agents.md)
