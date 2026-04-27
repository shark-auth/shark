# MCP Server — Drop SharkAuth in front of your MCP server

**Time to working integration: ~15 minutes.**

This guide covers the agent-native OAuth 2.1 flow for MCP servers: agents self-register (DCR, RFC 7591), obtain DPoP-bound tokens with audience binding (RFC 8707), and call your MCP server. The server validates via JWKS — no SharkAuth dependency required on the server side.

## Architecture

```
MCP client (agent)
  ├─ self-registers via DCR → SharkAuth
  ├─ requests DPoP token (audience=mcp-server) → SharkAuth
  └─ calls MCP server with DPoP proof
        └─ MCP server validates token via /.well-known/jwks.json
```

Two security properties that Auth0 cannot provide:

1. **DPoP binding (RFC 9449)** — token is bound to the agent's keypair. Stolen token alone is useless; the attacker must also steal the private key.
2. **Audience binding (RFC 8707)** — token is restricted to one specific MCP server URL. A token issued for `mcp-server-A` is rejected by `mcp-server-B`.

## Step 1 — Start SharkAuth

```bash
shark serve
```

Note your admin key from first-boot output, and your server URL (default `http://localhost:8080`).

## Step 2 — Agent self-registers (DCR cold-start)

No human in the loop. The agent calls the DCR endpoint on first boot, stores its `client_id` and `client_secret`, and reuses them on subsequent boots.

```python
import requests

SHARK_URL = "http://localhost:8080"

def dcr_register(agent_name: str) -> dict:
    """RFC 7591 Dynamic Client Registration — no admin key needed."""
    resp = requests.post(
        f"{SHARK_URL}/oauth/register",
        json={
            "client_name": agent_name,
            "grant_types": ["client_credentials"],
            "token_endpoint_auth_method": "client_secret_basic",
            "scope": "mcp:read mcp:write",
        },
    )
    resp.raise_for_status()
    creds = resp.json()
    # Persist creds["client_id"] and creds["client_secret"]
    return creds
```

See [`/documentation/api/oauth.md#dcr`](/documentation/api/oauth.md) for the full DCR spec.

## Step 3 — Agent requests a DPoP-bound, audience-restricted token

```python
from shark_auth import DPoPProver, OAuthClient

MCP_SERVER_URL = "https://mcp.example.com"

def get_mcp_token(client_id: str, client_secret: str) -> tuple:
    prover = DPoPProver.generate()
    oauth  = OAuthClient(base_url=SHARK_URL)

    token = oauth.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id=client_id,
        client_secret=client_secret,
        scope="mcp:read mcp:write",
        audience=MCP_SERVER_URL,   # RFC 8707 audience binding
    )
    # token.cnf_jkt bound to prover's keypair
    # token is only valid for MCP_SERVER_URL
    return token, prover
```

## Step 4 — Agent calls the MCP server

```python
from shark_auth import DPoPHTTPClient

def call_mcp(token_str: str, prover: DPoPProver, path: str) -> dict:
    http = DPoPHTTPClient(base_url=MCP_SERVER_URL)
    resp = http.get_with_dpop(path, token=token_str, prover=prover)
    resp.raise_for_status()
    return resp.json()
```

`DPoPHTTPClient` generates a fresh proof per request, binding `htm`, `htu`, and `ath` (token hash). Replay attacks are blocked.

## Step 5 — MCP server validates via JWKS

The MCP server does not need the SharkAuth SDK. Any RFC 7517-compliant library works:

```python
# Example: FastAPI MCP server using PyJWT
import jwt
from jwt import PyJWKClient

JWKS_URL  = "http://localhost:8080/.well-known/jwks.json"
jwks_client = PyJWKClient(JWKS_URL)

def verify_mcp_token(token: str, expected_audience: str) -> dict:
    signing_key = jwks_client.get_signing_key_from_jwt(token)
    claims = jwt.decode(
        token,
        signing_key.key,
        algorithms=["ES256", "RS256"],
        audience=expected_audience,
    )
    # Confirm DPoP binding: claims["cnf"]["jkt"] must match the DPoP proof header
    return claims
```

Or use the SharkAuth SDK's `decode_agent_token` helper (handles key rotation + JWKS caching):

```python
from shark_auth import decode_agent_token

claims = decode_agent_token(
    token,
    jwks_url=f"{SHARK_URL}/.well-known/jwks.json",
    expected_issuer=SHARK_URL,
    expected_audience=MCP_SERVER_URL,
)
```

See [`/documentation/api/jwks.md`](/documentation/api/jwks.md).

## Full cold-start flow

```python
from shark_auth import DPoPProver, OAuthClient, DPoPHTTPClient
import requests, json, pathlib

SHARK_URL      = "http://localhost:8080"
MCP_SERVER_URL = "https://mcp.example.com"
CREDS_FILE     = pathlib.Path(".agent_creds.json")

# --- cold start: register once, cache credentials ---
if not CREDS_FILE.exists():
    creds = requests.post(f"{SHARK_URL}/oauth/register", json={
        "client_name": "my-mcp-agent",
        "grant_types": ["client_credentials"],
        "scope": "mcp:read mcp:write",
    }).json()
    CREDS_FILE.write_text(json.dumps(creds))
else:
    creds = json.loads(CREDS_FILE.read_text())

# --- runtime: DPoP-bound, audience-restricted token ---
prover = DPoPProver.generate()
token  = OAuthClient(SHARK_URL).get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id=creds["client_id"],
    client_secret=creds["client_secret"],
    scope="mcp:read",
    audience=MCP_SERVER_URL,
)

# --- call ---
result = DPoPHTTPClient(base_url=MCP_SERVER_URL).get_with_dpop(
    "/tools/list", token=token.access_token, prover=prover,
)
print(result.json())
```

## Token revocation

To immediately invalidate this agent (e.g. compromised key):

```python
from shark_auth import OAuthClient
OAuthClient(base_url=SHARK_URL, token="sk_live_...").revoke_token(token.access_token)
```

Or rotate the DPoP key and invalidate all tokens bound to the old keypair:

```python
from shark_auth import Client, DPoPProver
admin     = Client(base_url=SHARK_URL, token="sk_live_...")
new_prover = DPoPProver.generate()
result = admin.agents.rotate_dpop_key(
    agent_id,
    new_public_key_jwk=new_prover.public_key_jwk(),
    reason="scheduled rotation",
)
# result.revoked_token_count tokens bound to old key are gone
```

See [10 — Five-Layer Revocation](./10-five-layer-revocation.md) for the full revocation model.

## Next steps

- Delegation chains (agent acts on behalf of user): [11 — Delegation Chains](./11-delegation-chains.md)
- Internal platform with audit trails: [03 — Internal Platform](./03-internal-platform.md)
- Full revocation model: [10 — Five-Layer Revocation](./10-five-layer-revocation.md)
- API reference: [`/documentation/api/oauth.md`](/documentation/api/oauth.md)
