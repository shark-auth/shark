# Cascade Revoke — Layer 3 of 5 Security Model

## Overview

SharkAuth implements a five-layer security model for agent credential risk management. Cascade revoke is **Layer 3**: a single admin action that terminates all active sessions, tokens, and consents associated with a user's agents simultaneously.

## Security Layers

| Layer | Mechanism | Scope |
|-------|-----------|-------|
| 1 | Per-token revocation (`POST /oauth/revoke`) | Single token |
| 2 | Per-agent token revocation (`POST /api/v1/agents/{id}/tokens/revoke-all`) | All tokens for one agent |
| **3** | **Cascade revoke (`POST /api/v1/users/{id}/revoke-agents`)** | **All agents + consents for a user** |
| 4 | User session termination (`DELETE /api/v1/users/{id}/sessions`) | All user sessions |
| 5 | User deletion (`DELETE /api/v1/users/{id}`) | Full account removal with cascade |

## When to Use Cascade Revoke

- **Rogue insider**: an employee created agents to exfiltrate data; revoke all without deleting the account for forensics.
- **Account compromise**: credentials leaked; kill all delegated access immediately without user interaction.
- **Offboarding**: a contractor's agents must be shut down before their account is deactivated.

## Endpoint

```
POST /api/v1/users/{id}/revoke-agents
Authorization: Bearer <admin-api-key>
Content-Type: application/json
```

### Body (all fields optional)

```json
{
  "agent_ids": ["agent-abc"],
  "reason": "describe the incident"
}
```

Omit `agent_ids` to target **all agents** created by the user.

## Auth Restriction

This endpoint requires an **admin API key**. Session cookies are rejected with `403`, even if the session belongs to the targeted user. This design prevents an account-takeover from self-escalating to cascade revocation.

## Atomicity Note

The revoke loop is best-effort: if an individual `UpdateAgent` fails, the loop continues to the next agent. `RevokeConsentsByUserID` is a single SQL `UPDATE` and is effectively atomic for the consents table.

## Audit Event

A single `user.cascade_revoked_agents` event is written regardless of how many agents were processed, keeping the audit log clean for incident response searches.

## Related Endpoints

- `GET /api/v1/users/{id}/agents?filter=created` — inventory before revoking
- `GET /api/v1/users/{id}/agents?filter=authorized` — see what the user authorized
- `GET /api/v1/audit-logs?action=user.cascade_revoked_agents&target_id={id}` — confirm the event
