# SharkAuth Quickstarts

SharkAuth is a self-hosted auth server with RFC-correct agent-native primitives: DPoP (RFC 9449), token exchange (RFC 8693), MCP-native OAuth 2.1, and a five-layer revocation model. These guides are organized by use-case, not by feature — pick your path below.

## Pick your path

| I want to… | Guide |
|---|---|
| Build a product that ships agents to my customers | [01 — Customer Agents](./01-customer-agents.md) |
| Drop auth in front of my MCP server | [02 — MCP Server](./02-mcp-server.md) |
| Run an internal AI platform with compliance audit trails | [03 — Internal Platform](./03-internal-platform.md) |
| Self-host an Auth0 replacement with agent support | [04 — Auth0 Replacement](./04-auth0-replacement.md) |

## Reference guides

| Topic | Guide |
|---|---|
| Five-layer revocation walkthrough | [10 — Five-Layer Revocation](./10-five-layer-revocation.md) |
| Delegation chains (RFC 8693 act chains) | [11 — Delegation Chains](./11-delegation-chains.md) |

## Prerequisites

```bash
pip install shark-auth
shark serve          # starts on :8080 by default, opens dashboard on first boot
```

First-boot creates an admin API key (`sk_live_...`) and displays it in the terminal. Copy it — it is not shown again.

## SDK surface at a glance

```python
from shark_auth import (
    Client,           # unified admin client
    DPoPProver,       # RFC 9449 DPoP proof generation
    OAuthClient,      # token request, exchange, revocation, introspection
    AgentsClient,     # agent CRUD + token management
    UsersClient,      # user CRUD + cascade revocation
    VaultClient,      # token vault — fetch + disconnect
    DPoPHTTPClient,   # DPoP-authenticated HTTP helpers
    DelegationTokenClaims,  # walk act-claim chains (no sig verify)
    decode_agent_token,     # verify tokens via JWKS
)
```

All sub-clients are also accessible via `Client`:

```python
client = Client(base_url="http://localhost:8080", token="sk_live_...")
client.agents      # AgentsClient
client.users       # UsersClient
client.oauth       # OAuthClient
client.vault       # VaultClient
client.http        # DPoPHTTPClient
```

## Related documentation

- API reference: `/documentation/api/`
- CLI reference: `/documentation/cli/`
- Five-layer revocation design: [`playbook/00-design.md`](../../playbook/00-design.md)
