# API keys

Admin API key CRUD + rotation. Wraps `/api/v1/api-keys`. Used to mint the `sk_live_...` keys you pass to `Client(token=...)` / `SharkClient({ adminKey: ... })`.

> The full key value is shown exactly once at creation or rotation. Store it before continuing.

## Setup

```python
from shark_auth import APIKeysClient
keys = APIKeysClient("https://auth.example.com", "sk_live_bootstrap")
```

```typescript
import { ApiKeysClient } from "@sharkauth/sdk";
const keys = new ApiKeysClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_bootstrap" });
```

Or via the composer:

```python
from shark_auth import Client
c = Client("https://auth.example.com", "sk_live_bootstrap")
keys = c.api_keys
```

```typescript
const c = new SharkClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_bootstrap" });
await c.apiKeys.list();
```

## Create

```python
created = keys.create(
    name="ci-bot",
    scopes=["agents:write", "users:read"],
    expires_at="2027-01-01T00:00:00Z",
)
print(created["key"])  # the only time you see this value
```

```typescript
const created = await keys.create({
  name: "ci-bot",
  scopes: ["agents:write", "users:read"],
  expiresAt: "2027-01-01T00:00:00Z",
});
console.log(created.key);
```

| Param        | Type      | Required | Notes                                   |
| ------------ | --------- | -------- | --------------------------------------- |
| `name`       | string    | yes      | Human-readable label                    |
| `scopes`     | string[]  | no       | Empty array means "no admin scopes"     |
| `expires_at` | ISO 8601  | no       | Defaults to no expiry                   |

Response: `{ id, key, name, scopes, expires_at, created_at }`. The `key` field is the bearer to pass to subsequent clients.

## List

```python
keys.list()
```

```typescript
await keys.list();
```

Returns metadata only — no raw key values.

## Get

```python
keys.get("ak_abc")
```

```typescript
await keys.get("ak_abc");
```

## Rotate

```python
rotated = keys.rotate("ak_abc")
print(rotated["key"])  # new value; old key revoked
```

```typescript
const rotated = await keys.rotate("ak_abc");
console.log(rotated.key);
```

The old key is invalidated immediately. Update your callers before destroying the old value.

## Revoke

```python
keys.revoke("ak_abc")
```

```typescript
await keys.revoke("ak_abc");
```

204 on success.

## Scope matrix

Admin scopes follow a `<resource>:<verb>` pattern:

| Scope                     | What                                        |
| ------------------------- | ------------------------------------------- |
| `users:read` / `:write`   | List/CRUD users, set tier                   |
| `agents:read` / `:write`  | Register, revoke, rotate agents             |
| `oauth:revoke`            | Bulk revoke by GLOB pattern                 |
| `vault:admin`             | List + disconnect vault connections         |
| `vault:read`              | Used on **agent** tokens, not API keys      |
| `webhooks:write`          | CRUD webhook subscriptions                  |
| `audit:read` / `:export`  | Query / export audit logs                   |
| `audit:purge`             | Delete old audit log entries                |
| `rbac:write`              | Manage roles, permissions, role assignments |
| `organizations:write`     | Org CRUD + member + invitation              |
| `apps:write`              | Application CRUD + secret rotation          |

Pass an empty `scopes` list (or omit) to request "all admin scopes" — but production keys should be scoped tightly.

## See also

- [Programmatic admin onboarding](./cookbook/programmatic-admin-onboarding.md) — bootstrap the first key + register integrations
- [Audit logs](./audit-logs.md) — `api_key.created` / `api_key.rotated` events
