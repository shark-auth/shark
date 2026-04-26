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

---

## POST /api/v1/agents/{id}/rotate-dpop-key

Rotates the DPoP keypair binding for an agent. The caller supplies the new public key JWK; the server records it as the new `cnf.jkt` for future token issuance and revokes all tokens bound to the old key.

**Auth:** Admin API key required (`Authorization: Bearer sk_live_...`).

### Request body

```json
{
  "new_public_jwk": {
    "kty": "EC",
    "crv": "P-256",
    "x": "<base64url>",
    "y": "<base64url>"
  },
  "reason": "scheduled rotation 2026-04-26"
}
```

| Field           | Type   | Required | Description                                          |
|-----------------|--------|----------|------------------------------------------------------|
| `new_public_jwk`| object | yes      | New public key in JWK format (EC P-256, RSA, or OKP) |
| `reason`        | string | no       | Audit-log reason string                              |

### Response (200 OK)

```json
{
  "old_jkt": "aB3kQ...",
  "new_jkt":  "xZ9mR...",
  "revoked_token_count": 3,
  "audit_event_id": "audit_abc123"
}
```

| Field                | Type    | Description                                           |
|----------------------|---------|-------------------------------------------------------|
| `old_jkt`            | string  | RFC 7638 SHA-256 thumbprint of the replaced key (empty if none was stored) |
| `new_jkt`            | string  | RFC 7638 SHA-256 thumbprint of the new key            |
| `revoked_token_count`| integer | Number of tokens revoked (bound to the old key)       |
| `audit_event_id`     | string  | ID of the emitted `agent.dpop_key_rotated` audit event |

### Error responses

| Status | Error code     | Meaning                              |
|--------|----------------|--------------------------------------|
| 400    | `invalid_request` | Missing or malformed JSON body     |
| 400    | `invalid_jwk`  | `new_public_jwk` is not a valid JWK  |
| 401    | —              | Missing or invalid admin API key     |
| 404    | `not_found`    | Agent not found                      |

### Audit event

Emits `agent.dpop_key_rotated` with metadata:
```json
{
  "old_jkt": "...",
  "new_jkt":  "...",
  "revoked_token_count": 3,
  "reason": "..."
}
```
