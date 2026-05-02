# Cookbook: agent acts on user's Gmail

End-to-end recipe. The user has connected Gmail to their Shark account via the dashboard's OAuth flow. You're building an agent that needs to read Alice's recent emails on her behalf.

## Architecture

```
1. Admin registers an agent for the integration   (one-time)
2. Agent generates DPoP keypair                   (per process)
3. Agent requests DPoP-bound Shark token          (scope: vault:read gmail:read)
4. Agent calls vault.fetch_token(provider=gmail)  (vault returns Google access token)
5. Agent calls Gmail API with Google token        (Gmail handles auth)
```

The Google access token returned by the vault is short-lived (Google manages expiry). Re-fetch when it expires — the vault will refresh it server-side using the stored refresh token.

## Step 1 — register the agent (admin)

Run once during onboarding.

```python
from shark_auth import Client

c = Client("https://auth.example.com", "sk_live_admin")
agent = c.agents.register_agent(
    app_id="app_inbox",
    name="alice-inbox-summarizer",
    scopes=["vault:read", "gmail:read"],
)
# Persist these securely — the client_secret is shown exactly once.
client_id = agent["client_id"]
client_secret = agent["client_secret"]
```

```typescript
import { SharkClient } from "@sharkauth/sdk";

const c = new SharkClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_admin" });
const agent = await c.agents.registerAgent({
  name: "alice-inbox-summarizer",
  scopes: ["vault:read", "gmail:read"],
  metadata: { app_id: "app_inbox" },
});
```

## Step 2 — agent boot

```python
from shark_auth import DPoPProver, OAuthClient, VaultClient

prover = DPoPProver.generate()
oauth  = OAuthClient("https://auth.example.com")
vault  = VaultClient(base_url="https://auth.example.com")
```

```typescript
import { DPoPProver, VaultClient } from "@sharkauth/sdk";

const prover = await DPoPProver.generate();
const vault = new VaultClient({
  authUrl: "https://auth.example.com",
  accessToken: "",  // set after step 3
});
```

## Step 3 — agent token

```python
shark_token = oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id=client_id,
    client_secret=client_secret,
    scope="vault:read gmail:read",
)
assert shark_token.cnf_jkt == prover.jkt
```

```typescript
// Until TS ships getTokenWithDpop helper, POST /oauth/token form-encoded with DPoP header.
// See sdk/HANDOFF.md parity gap.
```

## Step 4 — fetch Google token

```python
google = vault.fetch_token(
    provider="google_gmail",
    bearer_token=shark_token.access_token,
    prover=prover,
)
print(google.access_token)   # short-lived Google OAuth token
print(google.expires_at)
```

```typescript
// TS fetchToken takes a connection ID, not a provider slug.
// Use listConnections() to find the correct connection ID first.
const connections = await vault.listConnections({ limit: 10 });
const conn = connections.items.find(c => c.provider_id === "google_gmail");
const google = await vault.fetchToken(conn!.id);
```

## Step 5 — call Gmail

```python
import requests

resp = requests.get(
    "https://gmail.googleapis.com/gmail/v1/users/me/messages",
    headers={"Authorization": f"Bearer {google.access_token}"},
    params={"q": "newer_than:1d", "maxResults": 10},
)
resp.raise_for_status()
for msg in resp.json().get("messages", []):
    print(msg["id"])
```

```typescript
const resp = await fetch(
  "https://gmail.googleapis.com/gmail/v1/users/me/messages?q=newer_than:1d&maxResults=10",
  { headers: { Authorization: `Bearer ${google.accessToken}` } }
);
const { messages } = await resp.json();
```

## Refresh strategy

The Google access token will expire (typically in 60 minutes). When it does, just call `vault.fetch_token` again — the vault refreshes against Google's token endpoint server-side. You never see Alice's refresh token.

If the Shark agent token itself expires (default lifetime is server-configured), re-run step 3.

## Failure modes

| What happens                  | Cause                             | Action                                  |
| ----------------------------- | --------------------------------- | --------------------------------------- |
| `VaultError 401`              | Shark token expired or revoked    | Re-run step 3                           |
| `VaultError 403`              | Agent token missing `vault:read`  | Bug — fix scope at registration         |
| `VaultError 404`              | User has not connected Gmail      | Prompt: "connect your Gmail account"    |
| Gmail returns 401             | Google token expired              | Re-run step 4                           |
| Gmail returns 403             | User revoked Gmail access at Google | Re-run user-facing connect flow       |

## Revoking access

If Alice loses her phone, an admin can cascade-revoke every agent acting for her in one call:

```python
c.users.revoke_agents("usr_alice", reason="device-lost")
```

This kills all her agents' tokens in a single transaction (Layer 3 of [five-layer revocation](../delegation-and-agents.md#revocation--five-layers)).

## See also

- [Vault](../vault.md)
- [Delegation and agents](../delegation-and-agents.md)
- [DPoP](../dpop.md)
