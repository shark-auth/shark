# Organizations

Multi-tenant org CRUD plus members and invitations. Wraps `/api/v1/admin/organizations/*` and the user-side `/api/v1/organizations/invitations/{token}/accept`.

## Construct

```python
from shark_auth import OrganizationsClient
orgs = OrganizationsClient("https://auth.example.com", admin_api_key="sk_live_admin")
```

```typescript
import { OrganizationsClient } from "@sharkauth/sdk";
const orgs = new OrganizationsClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_admin" });
```

## Org CRUD

```python
org = orgs.create(name="Acme Co", slug="acme", metadata={"plan": "enterprise"})
orgs.list()
orgs.get(org["id"])
orgs.update(org["id"], name="Acme Inc")
orgs.delete(org["id"])
```

```typescript
const org = await orgs.create("Acme Co", "acme", { metadata: { plan: "enterprise" } });
await orgs.list();
await orgs.get(org.id);
await orgs.update(org.id, { name: "Acme Inc" });
await orgs.delete(org.id);
```

> The backend stores org `metadata` as a JSON-encoded string (`*string`). Both SDKs auto-`json.dumps()` / `JSON.stringify()` when you pass a dict, and decode on the way back.

## Members

```python
orgs.add_member(org["id"], "usr_alice", "admin")
orgs.list_members(org["id"])
orgs.remove_member(org["id"], "mem_alice")
```

```typescript
await orgs.addMember(org.id, "usr_alice", "admin");
await orgs.listMembers(org.id);
await orgs.removeMember(org.id, "mem_alice");
```

## Invitations

```python
inv = orgs.create_invitation(org["id"], email="bob@example.com", role="member")
orgs.list_invitations(org["id"])
# bob receives email, clicks link, hits your accept endpoint:
orgs.accept_invitation(token=token_from_link)
orgs.delete_invitation(org["id"], inv["id"])
```

```typescript
const inv = await orgs.createInvitation(org.id, "bob@example.com", "member");
await orgs.acceptInvitation(tokenFromLink);
await orgs.deleteInvitation(org.id, inv.id);
```

`accept_invitation` route is `POST /api/v1/organizations/invitations/{token}/accept` — note the user-side prefix (no `/admin/`). The user must be authenticated.

## See also

- [RBAC](./rbac.md) — global roles and permissions (org-scoped roles tracked separately)
- [Audit logs](./audit-logs.md) — `org.member_added`, `org.invitation_*` events
