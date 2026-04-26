# Admin OAuth API

Admin-key authenticated endpoints for OAuth token management. All routes require
`Authorization: Bearer <admin_key>` and are mounted under `/api/v1/admin/`.

---

## POST /api/v1/admin/oauth/revoke-by-pattern

Bulk-revokes all active (non-revoked) OAuth tokens whose `client_id` matches
a SQLite [GLOB](https://www.sqlite.org/lang_expr.html#glob) pattern.

**Use cases**
- Emergency rotation: revoke all tokens issued to a class of agents
- Incident response: revoke tokens matching a known compromised prefix
- Fleet maintenance: cycle tokens before a planned secret rotation

### Request

```http
POST /api/v1/admin/oauth/revoke-by-pattern
Authorization: Bearer <admin_key>
Content-Type: application/json
```

```json
{
  "client_id_pattern": "shark_agent_v3.2_*",
  "reason": "Quarterly credential rotation"
}
```

| Field               | Type   | Required | Description                                                        |
|---------------------|--------|----------|--------------------------------------------------------------------|
| `client_id_pattern` | string | yes      | SQLite GLOB pattern. `*` = any sequence, `?` = single character.   |
| `reason`            | string | no       | Human-readable reason recorded in the audit log.                   |

### Response — 200 OK

```json
{
  "revoked_count":   12,
  "audit_event_id":  "audit_V1StGXR8_Z5jdHi6B-myT",
  "pattern_matched": "shark_agent_v3.2_*"
}
```

| Field             | Type    | Description                                              |
|-------------------|---------|----------------------------------------------------------|
| `revoked_count`   | integer | Number of token rows updated (0 if no tokens matched).   |
| `audit_event_id`  | string  | ID of the audit log entry written for this operation.    |
| `pattern_matched` | string  | Echo of the submitted pattern (for idempotency checks).  |

### Error responses

| Status | `error`           | Cause                                    |
|--------|-------------------|------------------------------------------|
| 400    | `invalid_request` | `client_id_pattern` missing or empty.    |
| 401    | `unauthorized`    | Missing or invalid admin key.            |
| 500    | `internal_error`  | Database error during UPDATE.            |

### Audit event

A row is written to the audit log with:

- `action`: `oauth.bulk_revoke_pattern`
- `actor_type`: `admin`
- `status`: `success`
- `metadata`: `{ "pattern": "...", "revoked_count": N, "reason": "..." }`

### GLOB pattern reference

| Pattern          | Matches                                          |
|------------------|--------------------------------------------------|
| `*`              | All client IDs                                   |
| `agent_*`        | Any client_id starting with `agent_`             |
| `*_v3.2_*`       | Any client_id containing `_v3.2_`                |
| `agent_????`     | `agent_` followed by exactly 4 characters        |
| `*abc123*`       | Any client_id containing `abc123`                |

> **Note:** GLOB is case-sensitive in SQLite. Use `*` wildcards liberally to
> avoid accidentally matching zero tokens.

### Effect on agents

Bulk-revoking tokens **does not deactivate the agent** (OAuth client). The
agent can immediately re-issue new tokens via `/oauth/token`. To disable an
agent permanently, use `DELETE /api/v1/agents/{id}`.

---

## Related endpoints

- `GET /api/v1/admin/oauth/consents` — list all OAuth consents (tenant-wide)
- `DELETE /api/v1/admin/oauth/consents/{id}` — revoke a single consent
- `DELETE /api/v1/agents/{id}` — deactivate agent and revoke all its tokens
