"""agent_delegation.py — end-to-end delegation demo (~30 LOC).

Runs against a fresh `shark serve` with default first-boot credentials.
Usage:
    export SHARK_ADMIN_KEY=sk_live_...
    python examples/agent_delegation.py

Demonstrates:
  1. Register an agent
  2. Get a DPoP-bound token (Method 1)
  3. Exchange for a delegated sub-token (Method 2)
  4. Call a protected endpoint with DPoP proof (Method 3)
  5. Fetch audit chain via get_audit_logs (Method 5)
  6. Walk the delegation chain with AgentTokenClaims (Method 4)
"""

import os
import sys

import requests
from shark_auth import Client, DPoPProver
from shark_auth.claims import AgentTokenClaims

BASE_URL = os.environ.get("SHARK_BASE_URL", "http://localhost:8080")
ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")

if not ADMIN_KEY:
    sys.exit("Set SHARK_ADMIN_KEY to your admin API key (printed on first boot).")

client = Client(base_url=BASE_URL, token=ADMIN_KEY)
prover = DPoPProver.generate()

# 1. Register a demo agent
agent = client.agents.register_agent(
    app_id="demo",
    name="delegation-demo-agent",
    scopes=["mcp:read", "mcp:write"],
    grant_types=[
        "client_credentials",
        "urn:ietf:params:oauth:grant-type:token-exchange",
    ],
    token_endpoint_auth_method="client_secret_post",
)
agent_id = agent["id"]
client_id = agent["client_id"]
client_secret = agent["client_secret"]
print(f"Registered agent: {agent_id}  client_id={client_id}")

# 2. Get a DPoP-bound access token
token_url = f"{BASE_URL}/oauth/token"
proof1 = prover.make_proof(htm="POST", htu=token_url)
r1 = requests.post(
    token_url,
    data={"grant_type": "client_credentials", "scope": "mcp:read mcp:write"},
    auth=(client_id, client_secret),
    headers={"DPoP": proof1},
    timeout=10,
)
if r1.status_code != 200:
    sys.exit(f"Failed to get token: {r1.text}")
token_data = r1.json()
print(f"Got DPoP token (scope={token_data.get('scope')})")

# 3. Exchange for a narrower delegated sub-token (RFC 8693)
proof2 = prover.make_proof(htm="POST", htu=token_url)
r2 = requests.post(
    token_url,
    data={
        "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
        "subject_token": token_data["access_token"],
        "subject_token_type": "urn:ietf:params:oauth:token-type:access_token",
        "actor_token": token_data["access_token"],
        "actor_token_type": "urn:ietf:params:oauth:token-type:access_token",
        "requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
        "scope": "mcp:read",
    },
    auth=(client_id, client_secret),
    headers={"DPoP": proof2},
    timeout=10,
)
if r2.status_code != 200:
    sys.exit(f"Token exchange failed: {r2.text}")
sub_token_data = r2.json()
print(f"Got delegated sub-token (scope={sub_token_data.get('scope')})")

# 4. Walk the delegation chain (Method 4 — no backend needed)
claims = AgentTokenClaims.parse(sub_token_data["access_token"])
chain = claims.delegation_chain()
print(f"is_delegated={claims.is_delegated()}  chain_length={len(chain)}")
for i, hop in enumerate(chain):
    print(f"  hop[{i}] sub={hop.sub}  scope={hop.scope}  jkt={hop.jkt}")

# 5. Fetch audit logs for this agent (Method 5)
events = client.agents.get_audit_logs(agent_id, limit=5)
print(f"Audit events for agent: {len(events)}")
for ev in events:
    print(f"  [{ev.created_at}] {ev.event}")

# 6. List active tokens, then revoke them all (Method 5)
tokens = client.agents.list_tokens(agent_id)
print(f"Active tokens: {len(tokens)}")
result = client.agents.revoke_all(agent_id)
print(f"Revoked {result.revoked_count} token(s)")

# Cleanup
client.agents.revoke_agent(agent_id)
print("Demo complete — agent deactivated.")
