# RBAC

Roles, permissions, role assignments. Wraps `/api/v1/roles`, `/api/v1/permissions`, and `/api/v1/users/{id}/roles`.

## Construct

```python
from shark_auth import RBACClient
rbac = RBACClient("https://auth.example.com", admin_api_key="sk_live_admin")
```

```typescript
import { RbacClient } from "@sharkauth/sdk";
const rbac = new RbacClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_admin" });
```

## Roles

```python
role = rbac.create_role(name="editor", description="Can edit but not delete")
rbac.list_roles()
rbac.get_role(role["id"])
rbac.update_role(role["id"], description="Updated")
rbac.delete_role(role["id"])
```

```typescript
const role = await rbac.createRole("editor", "Can edit but not delete");
await rbac.listRoles();
await rbac.updateRole(role.id, { description: "Updated" });
await rbac.deleteRole(role.id);
```

Backend `update_role` uses `PUT /api/v1/roles/{id}` (NOT PATCH).

## Permissions

```python
perm = rbac.create_permission("documents:write", "folder_123")
rbac.list_permissions()
rbac.delete_permission(perm["id"])
```

```typescript
const perm = await rbac.createPermission("documents:write", "folder_123");
await rbac.listPermissions();
// Note: deletePermission is not yet implemented in the TS SDK.
```

## Attach permissions to roles

```python
rbac.attach_permission_to_role(role["id"], perm["id"])
rbac.detach_permission_from_role(role["id"], perm["id"])
```

```typescript
await rbac.attachPermission(role.id, perm.id);
await rbac.detachPermission(role.id, perm.id);
```

## Assign roles to users

```python
rbac.assign_role_to_user("usr_alice", role["id"])
rbac.revoke_role_from_user("usr_alice", role["id"])
rbac.list_user_roles("usr_alice")
```

```typescript
await rbac.assignRole("usr_alice", role.id);
await rbac.revokeRole("usr_alice", role.id);
await rbac.listUserRoles("usr_alice");
```

Backend assign route: `POST /api/v1/users/{user_id}/roles` (NOT under `/admin/`).

## Permission check on the user side

```python
from shark_auth import AuthClient

auth = AuthClient("https://auth.example.com")
auth.login("alice@example.com", "...")
result = auth.check(action="write", resource="documents:123")
# {"allowed": True}
```

```typescript
const result = await auth.check("write", "documents:123");
```

The check evaluates against the user's current roles + attached permissions.

## See also

- [Audit logs](./audit-logs.md) — `rbac.role_assigned` etc.
- [Organizations](./organizations.md)
