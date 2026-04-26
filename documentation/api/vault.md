# Vault API

The Token Vault stores and proxies third-party OAuth tokens on behalf of users
and agents. Agents retrieve fresh access tokens via a bearer-authenticated
endpoint; admins manage providers and connections via the admin sub-tree.

---

## Providers

| Method | Path | Auth |
|--------|------|------|
| `POST` | `/api/v1/vault/providers` | admin key |
| `GET` | `/api/v1/vault/providers` | admin key |
| `GET` | `/api/v1/vault/providers/{id}` | admin key |
| `PATCH` | `/api/v1/vault/providers/{id}` | admin key |
| `DELETE` | `/api/v1/vault/providers/{id}` | admin key |
| `GET` | `/api/v1/vault/templates` | admin key |

---

## Connections (user-facing)

| Method | Path | Auth |
|--------|------|------|
| `GET` | `/api/v1/vault/connect/{provider}` | session cookie |
| `GET` | `/api/v1/vault/callback/{provider}` | session cookie |
| `GET` | `/api/v1/vault/connections` | session cookie |
| `DELETE` | `/api/v1/vault/connections/{id}` | session cookie |

---

## Connections (admin)

| Method | Path | Auth |
|--------|------|------|
| `GET` | `/api/v1/admin/vault/connections` | admin key |
| `DELETE` | `/api/v1/admin/vault/connections/{id}` | admin key |

---

## Agent Token Retrieval

| Method | Path | Auth |
|--------|------|------|
| `GET` | `/api/v1/vault/{provider}/token` | OAuth Bearer token (scope: `vault:read`) |

Returns a fresh access token for the authenticated user bound to the bearer
token. The token must carry a `user_id` delegation binding.

---

## Disconnect cascade (Layer 5)

### Behavior

When an admin deletes a vault connection via
`DELETE /api/v1/admin/vault/connections/{id}`, the server performs a **cascade
revocation** of all OAuth tokens belonging to agents that have ever fetched a
token from that vault connection.

The lookup is done by querying `audit_logs` for rows with
`action = 'vault.token.retrieved'` and `target_id = <connectionID>`. Each
distinct `actor_id` is resolved to an agent record; then all active OAuth tokens
for that agent's `client_id` are soft-revoked (their `revoked_at` timestamp is
set).

This ensures that if a vault connection is revoked, no agent can continue to
exchange tokens on behalf of the revoked user account.

### Response shape

```json
{
  "disconnected": true,
  "connection_id": "vc_abc123",
  "revoked_agent_ids": ["agent_xyz", "agent_qrs"],
  "revoked_token_count": 4
}
```

| Field | Type | Description |
|-------|------|-------------|
| `disconnected` | bool | Always `true` on success |
| `connection_id` | string | ID of the vault connection that was deleted |
| `revoked_agent_ids` | string[] | IDs of agents that had at least one token revoked |
| `revoked_token_count` | integer | Total number of OAuth tokens revoked across all agents |

> Note: if no agent ever fetched from this connection, `revoked_agent_ids` is
> `[]` and `revoked_token_count` is `0`. The response is still `200 OK`.

### Audit events emitted

Two audit events are written for every successful admin disconnect:

| Action | Description |
|--------|-------------|
| `vault.disconnected` | The vault connection was deleted. Metadata includes `provider_id` and `user_id`. |
| `vault.disconnect_cascade` | Cascade revocation summary. Metadata includes `vault_connection_id`, `revoked_agent_ids`, and `revoked_token_count`. |

Both events have `actor_type = "admin"` and `target_type = "vault_connection"`.

Query cascade events:

```
GET /api/v1/admin/audit-logs?action=vault.disconnect_cascade
```
