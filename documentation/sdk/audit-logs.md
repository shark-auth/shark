# Audit logs

Query, export, and purge audit log events. Wraps `/api/v1/audit-logs` and `/api/v1/admin/audit-logs/purge`.

Every state-changing admin operation, every token issuance, every revocation, every consent grant emits an audit event. Use it for compliance reporting, anomaly detection, and incident forensics.

## Construct

```python
from shark_auth import AuditClient
audit = AuditClient("https://auth.example.com", admin_api_key="sk_live_admin")
```

```typescript
import { AuditClient } from "@sharkauth/sdk";
const audit = new AuditClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_admin" });
```

Or via the composer:

```python
c.audit.list(...)
```

```typescript
await c.audit.list(...);
```

## Query

```python
result = audit.list(
    actor_id="agent_abc",
    action="agent.token_issued",
    since="2026-04-20T00:00:00Z",
    until="2026-04-27T00:00:00Z",
    limit=100,
)
events = result["events"]
cursor = result.get("next_cursor")
```

```typescript
const result = await audit.list({
  actorId: "agent_abc",
  action: "agent.token_issued",
  since: "2026-04-20T00:00:00Z",
  until: "2026-04-27T00:00:00Z",
  limit: 100,
});
```

| Filter        | Notes                                              |
| ------------- | -------------------------------------------------- |
| `actor_id`    | Subject that triggered the event                   |
| `actor_type`  | `user`, `agent`, `api_key`, `system`               |
| `action`      | Event string (e.g. `agent.created`)                |
| `target_id`   | Affected resource id                               |
| `target_type` | Resource type                                      |
| `since` / `until` | ISO 8601 timestamps                            |
| `limit`       | Default 100                                        |
| `cursor`      | Opaque pagination cursor from previous response    |

Returns `{events: [...], next_cursor: ...}`. Each event carries `id`, `action` (or `event`), `actor_id`, `actor_type`, `target_id`, `target_type`, `metadata`, `created_at`, `ip`, `user_agent`.

## Get one event

```python
event = audit.get("audit_xxx")
```

```typescript
const event = await audit.get("audit_xxx");
```

## Export

```python
csv = audit.export(
    format="ndjson",
    since="2026-04-01T00:00:00Z",
    until="2026-05-01T00:00:00Z",
)
open("april.ndjson", "w").write(csv)
```

```typescript
const text = await audit.export("ndjson", {
  since: "2026-04-01T00:00:00Z",
  until: "2026-05-01T00:00:00Z",
});
```

| Format    | Notes                                         |
| --------- | --------------------------------------------- |
| `csv`     | Default server emission today                 |
| `ndjson`  | Newline-delimited JSON                        |
| `json`    | Single JSON array                             |

`since` and `until` are required. Returns the raw response body as a string.

Backend route: `POST /api/v1/audit-logs/export` (admin-key gated, NOT under `/admin/`).

## Purge

```python
result = audit.purge(before="2025-01-01T00:00:00Z", dry_run=True)
print(result["deleted"])
```

```typescript
const result = await audit.purge("2025-01-01T00:00:00Z", true);
```

Deletes all entries strictly older than `before`. Use `dry_run=True` to preview the count without mutating.

## Common event types

| Event                        | Emitted when                              |
| ---------------------------- | ----------------------------------------- |
| `user.created`               | Signup or admin create                    |
| `user.deleted`               | DELETE /me or admin                       |
| `user.session_created`       | Login                                     |
| `user.session_revoked`       | Logout / revoke / cascade                 |
| `agent.created`              | Register agent                            |
| `agent.token_issued`         | `client_credentials` or exchange          |
| `agent.token_revoked`        | revoke / revoke_all / bulk                |
| `agent.dpop_key_rotated`     | `rotate_dpop_key`                         |
| `agent.cascade_revoked`      | `users.revoke_agents`                     |
| `vault.connection_disconnected` | `vault.disconnect`                     |
| `oauth.bulk_revoked_pattern` | `bulk_revoke_by_pattern`                  |
| `webhook.delivery_failed`    | Webhook target returned non-2xx           |

## See also

- [Webhooks](./webhooks.md) — real-time event subscriptions on the same event stream
- [Delegation and agents](./delegation-and-agents.md) — what each agent.* event means
