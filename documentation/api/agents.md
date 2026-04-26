# Agents API

## PATCH /api/v1/agents/{id}

Updates an agent's properties.

### Token revocation on deactivation (Wave 1.5)

When `active` is set to `false`, the server now **immediately revokes all existing OAuth tokens** issued to that agent's `client_id` in addition to blocking new token issuance.

This ensures the UI promise is kept: "Deactivating will prevent new tokens and revoke all active tokens."

**Audit event**: `agent.deactivated_with_revocation` is written with metadata:

```json
{ "revoked_token_count": <number> }
```

Previously issued tokens will return `active: false` on introspection and `401` on protected resource requests as soon as the PATCH completes.

### Request body

| Field            | Type     | Description                          |
|------------------|----------|--------------------------------------|
| `name`           | string   | Agent display name                   |
| `description`    | string   | Agent description                    |
| `active`         | boolean  | `false` deactivates + revokes tokens |
| `scopes`         | string[] | Allowed OAuth scopes                 |
| `token_lifetime` | integer  | Token TTL in seconds                 |
| `metadata`       | object   | Arbitrary key/value metadata         |

### Response

Returns the updated agent object (200 OK) or 404 if not found.
