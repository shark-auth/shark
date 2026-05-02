# Delegation Chains — How agents act on behalf of users

RFC 8693 Token Exchange allows an agent to receive a token that says: "Agent A is acting on behalf of User B." The token carries a structured `act` claim chain proving the full lineage. SharkAuth implements this end-to-end: server, SDK, and audit log.

This is one of two primitives that make SharkAuth categorically different from Auth0. (The other is DPoP binding.) You cannot bolt this onto a hosted provider.

## Mental model

A delegation chain is a nested `act` claim inside a JWT:

```json
{
  "sub": "shark_agent_b",       // agent B is the current bearer
  "scope": "docs:read",
  "act": {
    "sub": "shark_agent_a",     // agent A delegated to agent B
    "iat": 1745640000,
    "act": {
      "sub": "usr_alice",       // Alice delegated to agent A
      "iat": 1745639900
    }
  }
}
```

Reading bottom-up: Alice authorized Agent A → Agent A delegated to Agent B → Agent B is presenting this token. The resource server can inspect the full lineage without a round-trip to SharkAuth.

## `may_act` policy

Before an agent can receive a delegated token, SharkAuth checks the `may_act` policy on the parent token's subject. The policy specifies which agents are permitted to act on behalf of which subjects.

Configure in the dashboard under **Agents → [agent name] → Delegation Policies**:

**screenshot: delegation canvas with 3-hop chain — Alice → Agent A → Agent B — with may_act policy editor visible**

Or via the admin API (see [`Delegation and agents`](../sdk/delegation-and-agents.md)).

## 3-hop chain walkthrough

### Setup

```python
from shark_auth import Client, DPoPProver, OAuthClient

SHARK_URL = "http://localhost:8080"
ADMIN_KEY = "sk_live_..."

admin = Client(base_url=SHARK_URL, token=ADMIN_KEY)
oauth = OAuthClient(base_url=SHARK_URL)

# Register two platform agents
agent_a = admin.agents.register_agent(
    app_id="app_internal",
    name="orchestrator-agent",
    scopes=["docs:read", "docs:write"],
)
agent_b = admin.agents.register_agent(
    app_id="app_internal",
    name="executor-agent",
    scopes=["docs:read"],
)
```

### Hop 1 — Human authenticates, obtains user token

The human authenticates via your app (authorization code flow, magic link, or password). They receive a user access token from SharkAuth:

```python
# Human-issued token — obtained via your app's login flow
# (POST /api/v1/auth/login or POST /oauth/token authorization_code)
human_token = "eyJhbGci...user_token..."
```

### Hop 2 — Agent A acts on behalf of the human (RFC 8693)

Agent A exchanges the human token for a delegated sub-token. SharkAuth checks `may_act` policy, then issues a new token with an `act` claim identifying Agent A as acting on behalf of the human.

```python
prover_a = DPoPProver.generate()

# Agent A's own token (for actor_token field)
token_a = oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover_a,
    client_id=agent_a["client_id"],
    client_secret=agent_a["client_secret"],
    scope="docs:read docs:write",
)

# Exchange: agent A acts on behalf of human
token_a_delegated = oauth.token_exchange(
    subject_token=human_token,           # the human's token
    dpop_prover=prover_a,
    actor_token=token_a.access_token,    # agent A identifies itself
    scope="docs:read docs:write",
    audience="https://docs.internal",
)
# token_a_delegated.act.sub == agent_a["client_id"]
# token_a_delegated.act.act.sub == human_user_id
```

### Hop 3 — Agent B acts on behalf of Agent A (2nd exchange)

Agent A narrows the scope further and delegates to Agent B:

```python
prover_b = DPoPProver.generate()

token_b = oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover_b,
    client_id=agent_b["client_id"],
    client_secret=agent_b["client_secret"],
    scope="docs:read",
)

# Exchange: agent B acts on behalf of agent A (which already acts on behalf of human)
token_b_delegated = oauth.token_exchange(
    subject_token=token_a_delegated.access_token,
    dpop_prover=prover_b,
    actor_token=token_b.access_token,
    scope="docs:read",                   # narrower — can only narrow, never widen
    audience="https://docs.internal",
)
```

`token_b_delegated` now carries a 3-level `act` chain:

```json
{
  "sub": "shark_agent_executor",
  "scope": "docs:read",
  "act": {
    "sub": "shark_agent_orchestrator",
    "act": {
      "sub": "usr_alice"
    }
  }
}
```

### Inspect the chain

```python
from shark_auth import DelegationTokenClaims

claims = DelegationTokenClaims.parse(token_b_delegated.access_token)

print(f"Bearer:       {claims.sub}")
print(f"Is delegated: {claims.is_delegated()}")
print(f"Scope:        {claims.scope}")
print(f"DPoP JKT:     {claims.jkt}")

chain = claims.delegation_chain()
print(f"\nDelegation chain ({len(chain)} hops):")
for i, hop in enumerate(chain):
    print(f"  hop {i+1}: sub={hop.sub}  iat={hop.iat}  scope={hop.scope}")
```

Output:

```
Bearer:       shark_agent_executor
Is delegated: True
Scope:        docs:read
DPoP JKT:     abc123...

Delegation chain (2 hops):
  hop 1: sub=shark_agent_orchestrator  iat=1745640050  scope=None
  hop 2: sub=usr_alice                 iat=1745639900  scope=None
```

**screenshot: delegation canvas with 3-hop chain — usr_alice → shark_agent_orchestrator → shark_agent_executor — with DPoP JKT visible per hop**

### Verify on the resource server

```python
from shark_auth import decode_agent_token

claims = decode_agent_token(
    token_b_delegated.access_token,
    jwks_url=f"{SHARK_URL}/.well-known/jwks.json",
    expected_issuer=SHARK_URL,
    expected_audience="https://docs.internal",
)

# claims.act walks the chain
# claims.jkt is the DPoP thumbprint — must match the DPoP proof header
```

The resource server does not need to call SharkAuth to validate — JWKS is cached, verification is local.

## Audit log for a 3-hop chain

```json
[
  {
    "event": "oauth.token_issued",
    "actor_id": "usr_alice",
    "metadata": { "scope": "docs:read docs:write", "jkt": null }
  },
  {
    "event": "oauth.token_exchanged",
    "actor_id": "shark_agent_orchestrator",
    "metadata": {
      "subject_id": "usr_alice",
      "scope": "docs:read docs:write",
      "audience": "https://docs.internal",
      "jkt": "prover_a_jkt..."
    }
  },
  {
    "event": "oauth.token_exchanged",
    "actor_id": "shark_agent_executor",
    "metadata": {
      "subject_id": "shark_agent_orchestrator",
      "scope": "docs:read",
      "audience": "https://docs.internal",
      "jkt": "prover_b_jkt..."
    }
  }
]
```

Query the full chain for an audit review:

```python
events = admin.agents.get_audit_logs("shark_agent_executor", limit=20)
exchanges = [e for e in events if e.event == "oauth.token_exchanged"]
for ev in exchanges:
    print(f"{ev.created_at}  subject={ev.metadata.get('subject_id')}  scope={ev.metadata.get('scope')}")
```

## Revoking a delegation chain

Revoking at any level in the chain cascades downward:

```python
# Revoke human's consent → all delegated tokens derived from it are invalid
admin.users.revoke_agents("usr_alice", reason="consent withdrawn")

# Revoke agent A only → agent B's delegated token is also invalid (derived from A)
admin.agents.revoke_all("shark_agent_orchestrator")
```

See [10 — Five-Layer Revocation](./10-five-layer-revocation.md) for the full model.

## Scope narrowing enforcement

The server enforces that each exchange can only narrow scope, never widen it. If you request a scope not present in the subject token, the server returns:

```json
{ "error": "invalid_scope", "error_description": "requested scope exceeds subject token grant" }
```

The SDK raises `OAuthError` with `error="invalid_scope"`.

## Reference

- `DelegationTokenClaims.parse()` → `claims.py` — no signature verification, pure `act` chain walk
- `decode_agent_token()` → `tokens.py` — full signature verification via JWKS
- `OAuthClient.token_exchange()` → `oauth.py` — RFC 8693 exchange with DPoP binding
- API reference: [`../sdk/token-exchange.md`](../sdk/token-exchange.md)
- Five-layer revocation: [10 — Five-Layer Revocation](./10-five-layer-revocation.md)
