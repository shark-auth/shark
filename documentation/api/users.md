# Users API

## DELETE /api/v1/users/{id}

Permanently deletes a user account.

### Token and session revocation on deletion (Wave 1.5)

Before the user record is removed, the server now performs the following revocation steps to prevent orphaned tokens from passing introspection:

1. **List all agents** created by the user (`created_by = user_id`).
2. **Revoke all OAuth tokens** for each agent via `RevokeOAuthTokensByClientID`.
3. **Delete all active sessions** for the user via `DeleteSessionsByUserID`.
4. **Delete the user** (ON DELETE CASCADE handles FK-linked rows in the schema).

**Audit event**: `user.deleted_with_token_revocation` is written with metadata:

```json
{
  "revoked_token_count": <number>,
  "revoked_session_count": <number>
}
```

Previously issued tokens will return `active: false` on introspection immediately after the DELETE completes.

### Response

Returns `200 OK` with `{ "message": "User deleted" }` on success, or `404` if the user does not exist.

---

## GET /api/v1/users/{id}/agents

Returns agents associated with a user. Auth: **admin API key**.

### Query parameters

| Parameter | Values | Default | Description |
|-----------|--------|---------|-------------|
| `filter`  | `created` \| `authorized` | `created` | Which relationship to query |

**`filter=created`** — agents where `created_by = user_id`.

**`filter=authorized`** — agents where an active `oauth_consents` row exists for the user (`revoked_at IS NULL`).

### Response

```json
{
  "data": [ ...agents ],
  "total": 2,
  "filter": "created"
}
```

---

## GET /api/v1/me/agents

Same as `/users/{id}/agents` but scoped to the **calling user**. Auth: **session cookie**.

### Query parameters

Same `filter` parameter as above (`created` | `authorized`).

### Response

Same shape as `/users/{id}/agents`.

---

## POST /api/v1/users/{id}/revoke-agents

**Cascade revoke** — deactivates all of a user's agents and revokes all associated OAuth tokens and consents in a single operation. Part of the **Layer 3 of 5** security model (see `cascade-revoke.md`).

Auth: **ADMIN API KEY ONLY**. Session tokens are explicitly rejected (mitigates account-takeover blast radius).

### Request body (optional)

```json
{
  "agent_ids": ["agent-abc", "agent-def"],
  "reason": "rogue insider 2026-04-26"
}
```

When `agent_ids` is omitted, **all** agents created by the user are targeted.

### Server-side actions

1. If `agent_ids` is omitted, list all agents where `created_by = user_id`.
2. For each agent: set `active = false`, revoke all OAuth tokens via `RevokeOAuthTokensByClientID`.
3. Bulk-revoke all `oauth_consents` rows where `user_id = ?` and `revoked_at IS NULL`.
4. Write a single audit event `user.cascade_revoked_agents` with metadata.
5. Return summary.

### Response

```json
{
  "revoked_agent_ids": ["agent-abc", "agent-def"],
  "revoked_consent_count": 3,
  "audit_event_id": "audit_XYZ"
}
```

### Audit metadata

```json
{
  "revoked_agent_count": 2,
  "revoked_consent_count": 3,
  "reason": "rogue insider 2026-04-26",
  "by_actor": "admin"
}
```
