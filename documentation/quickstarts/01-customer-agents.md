# Customer Agents — Build a product that ships agents to your customers

**Time to working integration: ~10 minutes.**

This is the primary SharkAuth use-case: you build a SaaS product, each of your customers gets their own agent identity, and you need a way to provision, bound, and revoke those agents without writing the OAuth server yourself.

## Mental model

```
human (your customer)
  └─▶ your app (your backend)
        └─▶ customer's agent (provisioned by you per customer)
              └─▶ external resource (MCP server, API, etc.)
                    └─▶ validates via JWKS at /.well-known/jwks.json
```

SharkAuth sits between your app and the external resource. Your app holds the admin key. The agent holds a DPoP-bound token. The external resource trusts the JWKS endpoint.

When a customer cancels, one call (`revoke_agents`) kills every token across all their agents instantly — no token TTL to wait out.

## Step 1 — Start the server

```bash
shark serve
```

First boot prints your admin API key (`sk_live_...`) and opens the dashboard at `http://localhost:8080`. Copy the key.

**screenshot: first-boot terminal with admin key highlighted**

## Step 2 — Create an Application (your app's OAuth client)

In the dashboard: **Applications → New Application**. Give it a name (e.g. `my-saas-backend`). The Application ID (`app_...`) scopes all agents created by your backend.

**screenshot: new application dialog with app_id visible**

Alternatively via CLI:

```bash
shark app create --name "my-saas-backend"
```

See [`/documentation/cli/app.md`](/documentation/cli/app.md) for full CLI reference.

## Step 3 — Implement signup: provision a per-customer agent

When a customer signs up in your product, call `register_agent()` to create their agent identity in SharkAuth. Store the returned `client_id` and `client_secret` in your database — the secret is shown once.

```python
from shark_auth import Client

admin = Client(base_url="http://localhost:8080", token="sk_live_...")

def on_customer_signup(customer_id: str, customer_name: str) -> dict:
    """Call this from your signup handler."""
    agent = admin.agents.register_agent(
        app_id="app_your_app_id",
        name=f"agent-{customer_name}",
        scopes=["mcp:read", "mcp:write"],
        description=f"Agent for customer {customer_id}",
    )
    # agent["client_id"]     — store in your DB alongside customer_id
    # agent["client_secret"] — store encrypted, shown once
    return agent
```

The agent `client_id` (`shark_agent_...`) is the stable identifier you'll use for every subsequent operation.

## Step 4 — Agent runtime requests a DPoP-bound token

At agent runtime (not signup time), the agent generates a keypair, requests a token bound to that key, and uses it for every outbound call.

```python
from shark_auth import DPoPProver, OAuthClient

def get_agent_token(client_id: str, client_secret: str) -> tuple:
    """Agent calls this on startup to get its DPoP-bound token."""
    prover = DPoPProver.generate()          # fresh keypair per agent instance

    oauth = OAuthClient(base_url="http://localhost:8080")
    token = oauth.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id=client_id,
        client_secret=client_secret,
        scope="mcp:read mcp:write",
    )
    # token.cnf_jkt == prover.jkt  (key binding confirmed)
    # token theft alone is useless — holder must also own the private key
    return token, prover
```

`OAuthClient.get_token_with_dpop()` → [`/documentation/api/oauth.md#get_token_with_dpop`](/documentation/api/oauth.md)

## Step 5 — Agent calls an external resource

The agent makes DPoP-authenticated HTTP calls. The proof is freshly signed per request, binding `htm`, `htu`, and `ath`.

```python
from shark_auth import DPoPHTTPClient

def call_mcp_server(token_str: str, prover: DPoPProver) -> dict:
    http = DPoPHTTPClient(base_url="https://mcp.example.com")
    resp = http.get_with_dpop("/resource", token=token_str, prover=prover)
    resp.raise_for_status()
    return resp.json()
```

The MCP server validates the token via the JWKS endpoint at:

```
GET http://localhost:8080/.well-known/jwks.json
```

No SharkAuth SDK required on the resource server — any RFC 7517-compliant JWKS validator works.

See [`/documentation/api/jwks.md`](/documentation/api/jwks.md) for the JWKS endpoint spec.

## Step 6 — Customer cancels: cascade revoke all their agents

One call. All tokens across all agents created by this customer are immediately invalid — no TTL wait, no polling.

```python
def on_customer_cancel(user_id: str) -> None:
    result = admin.users.revoke_agents(
        user_id,
        reason="customer cancelled subscription",
    )
    print(f"Revoked agents: {result.revoked_agent_ids}")
    print(f"Audit event: {result.audit_event_id}")
```

This is Layer 3 of the five-layer revocation model. See [`10-five-layer-revocation.md`](./10-five-layer-revocation.md) for the full model.

## Full example (≤25 lines)

```python
from shark_auth import Client, DPoPProver, OAuthClient

SHARK_URL = "http://localhost:8080"
ADMIN_KEY = "sk_live_..."   # from first-boot output

admin = Client(base_url=SHARK_URL, token=ADMIN_KEY)

# 1. Signup: provision agent
agent = admin.agents.register_agent(
    app_id="app_abc123",
    name="agent-alice",
    scopes=["mcp:read", "mcp:write"],
)
client_id     = agent["client_id"]
client_secret = agent["client_secret"]

# 2. Runtime: get DPoP-bound token
prover = DPoPProver.generate()
token = OAuthClient(SHARK_URL).get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id=client_id,
    client_secret=client_secret,
    scope="mcp:read mcp:write",
)

# 3. Call external resource (validates via /.well-known/jwks.json)
from shark_auth import DPoPHTTPClient
resp = DPoPHTTPClient(base_url="https://mcp.example.com").get_with_dpop(
    "/resource", token=token.access_token, prover=prover,
)

# 4. Customer cancels: one call, all tokens gone
admin.users.revoke_agents("usr_alice_id", reason="cancelled")
```

## What happens in the audit log

Every step above emits an audit event queryable at `GET /api/v1/audit-logs`. Example events:

```json
{ "event": "agent.created",        "actor_id": "app_abc123",   "target_id": "shark_agent_..." }
{ "event": "oauth.token_issued",   "actor_id": "shark_agent_...", "metadata": { "scope": "mcp:read mcp:write", "jkt": "..." } }
{ "event": "user.agents_revoked",  "actor_id": "usr_alice_id",  "metadata": { "revoked_count": 1 } }
```

See [`/documentation/api/audit-logs.md`](/documentation/api/audit-logs.md).

## Next steps

- Add delegation chains so agents act on behalf of your human users: [11 — Delegation Chains](./11-delegation-chains.md)
- Understand all five revocation layers: [10 — Five-Layer Revocation](./10-five-layer-revocation.md)
- Drop SharkAuth in front of an MCP server directly: [02 — MCP Server](./02-mcp-server.md)
