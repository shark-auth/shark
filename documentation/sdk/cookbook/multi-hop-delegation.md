# Cookbook: multi-hop delegation chain

User → orchestrator → worker → tool. Each hop narrows scope. The resource server walks the `act` chain to enforce policy without trusting any intermediate party.

## Scenario

Alice (`usr_alice`) authorizes an "Inbox Triage" workflow. The orchestrator agent dispatches a worker that calls a search tool. Three agents in the chain:

```
usr_alice
   │
   └─ orchestrator (scope: inbox:read inbox:write search:query)
            │
            └─ worker        (scope: inbox:read search:query)
                     │
                     └─ search_tool (scope: search:query, audience=https://search.api)
```

## Step 1 — register agents (one-time)

```python
from shark_auth import Client
c = Client("https://auth.example.com", "sk_live_admin")

orch = c.agents.register_agent(
    app_id="app_triage", name="orchestrator",
    scopes=["inbox:read", "inbox:write", "search:query"],
)
worker = c.agents.register_agent(
    app_id="app_triage", name="worker",
    scopes=["inbox:read", "search:query"],
)
search = c.agents.register_agent(
    app_id="app_triage", name="search_tool",
    scopes=["search:query"],
)
```

## Step 2 — Alice authorizes the orchestrator

Authorization-code grant + PKCE. Alice clicks consent on the Shark dashboard.

```python
from shark_auth import OAuthClient, pkce_pair

verifier, challenge, _ = pkce_pair()
authorize_url = OAuthClient.build_authorize_url(
    client_id=orch["client_id"],
    redirect_uri="https://triage.example/cb",
    scope="inbox:read inbox:write search:query",
    state="csrf-xyz",
    code_challenge=challenge,
    base_url="https://auth.example.com",
)
# Redirect Alice to authorize_url. Receive ?code=...&state=... at /cb.
```

## Step 3 — orchestrator obtains user token

```python
from shark_auth import DPoPProver

orch_prover = DPoPProver.generate()
oauth = OAuthClient("https://auth.example.com")

# In real life this is /oauth/token with DPoP header — for token-exchange flows
# the orchestrator typically receives the user's token through a different
# channel (Authorization Code grant, or session bridge). For brevity we assume
# `user_token` exists and is DPoP-bound to orch_prover.
user_token = oauth.get_token_authorization_code(
    code="auth_xyz",
    redirect_uri="https://triage.example/cb",
    code_verifier=verifier,
    client_id=orch["client_id"],
    client_secret=orch["client_secret"],
)
```

## Step 4 — orchestrator delegates to worker

The orchestrator narrows the scope and adds itself as the actor.

```python
worker_prover = DPoPProver.generate()
worker_token = oauth.token_exchange(
    subject_token=user_token.access_token,        # original user
    actor_token=user_token.access_token,          # delegating agent
    dpop_prover=worker_prover,
    scope="inbox:read search:query",              # narrower
)
```

Resulting payload:

```json
{
  "sub": "usr_alice",
  "scope": "inbox:read search:query",
  "cnf": { "jkt": "<worker_prover.jkt>" },
  "act": {
    "sub": "shark_agent_orchestrator",
    "iat": 1714200000,
    "scope": "inbox:read inbox:write search:query",
    "cnf": { "jkt": "<orch_prover.jkt>" }
  }
}
```

## Step 5 — worker delegates to search tool

```python
search_prover = DPoPProver.generate()
search_token = oauth.token_exchange(
    subject_token=worker_token.access_token,
    actor_token=worker_token.access_token,
    dpop_prover=search_prover,
    scope="search:query",
    audience="https://search.api",
)
```

Now the token has `act -> act` — two delegation hops.

## Step 6 — search tool calls resource server

```python
from shark_auth import DPoPHTTPClient

http = DPoPHTTPClient(
    base_url="https://search.api",
    prover=search_prover,
    access_token=search_token.access_token,
)
resp = http.get("/v1/search", params={"q": "from:vendor"})
```

## Step 7 — resource server walks the chain

```python
from shark_auth import AgentTokenClaims

claims = AgentTokenClaims.parse(search_token.access_token)
print(claims.sub)         # usr_alice
print(claims.scope)       # search:query
print(claims.is_delegated())  # True

for i, hop in enumerate(claims.delegation_chain()):
    print(f"hop {i}: {hop.sub} (scope={hop.scope}, jkt={hop.jkt[:8]}…)")
```

```typescript
import { decodeAgentToken } from "@sharkauth/sdk";

const claims = decodeAgentToken(searchToken.accessToken);
console.log(claims.sub);
for (const hop of claims.act_chain ?? []) {
  console.log(hop.sub, hop.scope, hop.cnf?.jkt);
}
```

The resource server policy can now express things like:

> "Allow `search:query` only when the chain includes `shark_agent_worker` AND the original `sub` is a Pro-tier user"

without trusting the worker — every hop's signature was minted by the SharkAuth issuer.

## Constraints the server enforces

- **No scope expansion.** Each `token_exchange` must request a subset of the parent's scope. `invalid_scope` otherwise.
- **DPoP binding preserved.** Each child token's `cnf.jkt` matches the prover used at exchange time. The chain hops carry the per-hop `cnf.jkt` so the resource server can verify each hop's DPoP proof.
- **Revoke any link, kill the chain.** Revoke the orchestrator's token and the worker / search tokens are usable until expiry but no longer refreshable. Use `bulk_revoke_by_pattern` for emergency stops.

## See also

- [Token exchange](../token-exchange.md)
- [Delegation and agents](../delegation-and-agents.md)
- [Quickstart: delegation chains](../../quickstarts/11-delegation-chains.md)
