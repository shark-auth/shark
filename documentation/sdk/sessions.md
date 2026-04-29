# Sessions

User-side session listing and revocation. Wraps `/api/v1/auth/sessions`.

For admin-side session listing across users, see `sdk/HANDOFF.md` post-launch backlog (P1 — not yet shipped).

## Setup

The sessions client requires the authenticated user's `requests.Session` (Python) or session shim (TS).

```python
from shark_auth import AuthClient, SessionsClient

auth = AuthClient("https://auth.example.com")
auth.login("alice@example.com", "...")
sessions = SessionsClient("https://auth.example.com", session=auth._session)
```

```typescript
import { AuthClient, SessionsClient } from "@sharkauth/sdk";

const auth = new AuthClient("https://auth.example.com");
await auth.login("alice@example.com", "...");
const sessions = new SessionsClient("https://auth.example.com", { session: auth.session });
```

## List

```python
rows = sessions.list()
for s in rows:
    print(s["id"], s.get("user_agent"), s.get("current"))
```

```typescript
const rows = await sessions.list();
for (const s of rows) console.log(s.id, s.user_agent, s.current);
```

Each row carries `id`, `created_at`, `last_used_at`, `user_agent`, `ip`, `current` (boolean).

## Revoke a single session

```python
sessions.revoke("sess_abc")
```

```typescript
await sessions.revoke("sess_abc");
```

## Revoke all (except current)

The user-facing surface has no bulk-revoke endpoint. The SDK iterates `list()` and skips the row flagged `current`.

```python
sessions.revoke_all()
```

```typescript
await sessions.revokeAll();
```

## Consents

Sibling client for user-granted OAuth consents.

```python
from shark_auth import ConsentsClient
consents = ConsentsClient("https://auth.example.com", session=auth._session)
consents.list()
consents.revoke("cnst_abc")
```

```typescript
import { ConsentsClient } from "@sharkauth/sdk";
const consents = new ConsentsClient("https://auth.example.com", { session: auth.session });
await consents.list();
await consents.revoke("cnst_abc");
```

## See also

- [Delegation and agents](./delegation-and-agents.md) — `users.revoke_agents()` cascades agent token revocation when a user loses their device
- [Audit logs](./audit-logs.md) — `auth.session_revoked` events
