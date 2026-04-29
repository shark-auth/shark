# Vault

Shark Token Vault — broker third-party OAuth credentials (Google, Microsoft, Slack, Linear, Jira, custom) without your agents ever seeing the refresh tokens.

The vault stores encrypted refresh tokens server-side. Agents authenticate with their own DPoP-bound Shark token and ask the vault for a fresh access token for a connection they've been authorized to use. The third-party access token is never persisted on the agent.

## Mental model

```
[user signed in with Shark]  ── connects "google_gmail" via OAuth ──▶  [vault stores refresh_token]
                                                                              │
[agent with DPoP-bound Shark token, scope vault:read]  ───────────▶  [vault decrypts + refreshes if needed]
                                                                              │
                                                                              ▼
                                                        returns fresh access_token to agent
```

Out of scope today: provider/template CRUD (P1 backlog), enroll-from-code helper. Connections are created via the dashboard's OAuth flow or the lower-level vault HTTP routes.

## Construct

```python
from shark_auth import VaultClient

vault = VaultClient(
    base_url="https://auth.example.com",
    admin_api_key="sk_live_admin",   # for admin ops (list, disconnect)
)
```

```typescript
import { VaultClient } from "@sharkauth/sdk";

const vault = new VaultClient({
  authUrl: "https://auth.example.com",
  accessToken: agentDpopToken,    // for fetch_token
  adminKey: "sk_live_admin",      // for list/disconnect
});
```

## List connections (admin)

```python
result = vault.list_connections(provider_id="google_gmail", limit=50)
print(result.total)
for conn in result.items:
    print(conn.id, conn.user_id, conn.provider_id, conn.needs_reauth)
```

```typescript
const result = await vault.listConnections({ limit: 50 });
```

`VaultConnectionRecord` carries `id`, `user_id`, `provider_id`, `provider_name`, `created_at`, `updated_at`, `needs_reauth`.

## Fetch a fresh token (agent path)

The agent must have a DPoP-bound Shark access token with `vault:read` scope. The vault validates the DPoP proof, the scope, and the `tok.UserID` binding before decrypting.

```python
from shark_auth import OAuthClient, DPoPProver, VaultClient

prover = DPoPProver.generate()
oauth = OAuthClient("https://auth.example.com")
agent_token = oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="shark_agent_xxx",
    client_secret="...",
    scope="vault:read",
)

vault = VaultClient(base_url="https://auth.example.com")
fresh = vault.fetch_token(
    provider="google_gmail",
    bearer_token=agent_token.access_token,
    prover=prover,
)
print(fresh.access_token)   # short-lived Google access token
print(fresh.expires_at)
```

```typescript
// TS path — exchange for a vault-scoped child token, then call fetchToken with dpopProver.
import { exchangeToken, VaultClient, DPoPProver } from "@sharkauth/sdk";

const prover = await DPoPProver.generate();
// ... obtain agentToken via /oauth/token with DPoP and scope=vault:read ...

const vault = new VaultClient({
  authUrl: "https://auth.example.com",
  accessToken: agentToken,
});
// Note: TS fetchToken signature differs slightly — check vault.ts; it currently
// exposes exchange(referenceToken) + fetchToken(referenceToken). For full parity
// with Python's fetch_token(provider, bearer, prover) see HANDOFF.md gap.
```

The Python signature is the canonical one. The TypeScript SDK's `fetchToken(referenceToken)` currently uses an older shape — close-but-not-identical. Track the parity gap in `sdk/HANDOFF.md`.

| Param          | Notes                                              |
| -------------- | -------------------------------------------------- |
| `provider`     | Provider slug (e.g. `google_gmail`, `slack`)       |
| `bearer_token` | Agent's DPoP-bound Shark access token              |
| `prover`       | The keypair whose JWK matches token's `cnf.jkt`    |

Returns `VaultTokenResult` with `access_token`, `token_type`, `expires_at`, `provider`.

## Disconnect a connection

Optionally cascade-revokes tokens for every agent that has touched the connection (Layer 5 of the five-layer revocation model).

```python
result = vault.disconnect("conn_abc123", cascade_to_agents=True)
print(result.revoked_agent_ids)
print(result.revoked_token_count)
print(result.cascade_audit_event_id)
```

```typescript
const result = await vault.disconnect("conn_abc123", true);
```

Pass `cascade_to_agents=False` to silently disconnect without revoking agent tokens.

## Common providers

| Provider slug    | Notes                                  |
| ---------------- | -------------------------------------- |
| `google_gmail`   | Read/send Gmail                        |
| `google_calendar`| Calendar API                           |
| `google_drive`   | Drive API                              |
| `microsoft`      | M365 unified                           |
| `slack`          | Slack workspace                        |
| `linear`         | Linear API                             |
| `jira`           | Atlassian Jira                         |
| `notion`         | Notion API                             |

For a custom OAuth provider, the dashboard can register a template — but the SDK does not yet expose programmatic CRUD for provider templates (P1 backlog).

## Errors

`VaultError(message, status_code)`:

- `401` — invalid or expired DPoP token
- `403` — missing `vault:read` scope on the agent token
- `404` — no connection for that provider/user

## See also

- [Delegation and agents](./delegation-and-agents.md) — vault is Layer 5 of revocation
- [Cookbook: agent acts on Gmail](./cookbook/agent-acts-on-gmail.md) — end-to-end recipe
