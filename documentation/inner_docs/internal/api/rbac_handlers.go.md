# rbac_handlers.go

**Path:** `internal/api/rbac_handlers.go`
**Package:** `api`
**LOC:** 762
**Tests:** likely integration-tested

## Purpose
Global (non-org) RBAC: tenant-wide roles, permissions, role↔permission attach/detach, user↔role grants, permission inspection, and an `auth/check` predicate endpoint. Distinct from `org_rbac_handlers.go` (per-org RBAC).

## Handlers exposed
**Roles**
- `handleCreateRole` (line 107), `handleListRoles` (line 164), `handleGetRole` (line 182), `handleUpdateRole` (line 220), `handleDeleteRole` (line 264)

**Permissions**
- `handleCreatePermission` (line 295), `handleListPermissions` (line 350), `handleDeletePermission` (line 368)
- `handleAttachPermission` (line 397) — POST `/roles/{id}/permissions`
- `handleDetachPermission` (line 460) — DELETE `/roles/{id}/permissions/{pid}`

**User membership**
- `handleAssignRole` (line 478) — POST `/users/{id}/roles`
- `handleRemoveRole` (line 541) — DELETE `/users/{id}/roles/{rid}`
- `handleListUserRoles` (line 557), `handleListUserPermissions` (line 576)

**Inspection**
- `handleAuthCheck` (line 605) — POST `/auth/check`. `{user_id, action, resource}` → `{allowed: bool}`.
- `handleListRolesByPermission` (line 646), `handleListUsersByPermission` (line 672)
- `handlePermissionsBatchUsage` (line 707) — GET `/admin/permissions/batch-usage`. Per-permission usage counts for the dashboard.

## Key types
- `roleResponse` (line 45) (with optional inlined `Permissions`)
- `permissionResponse` (line 59)
- Request types: `createRoleRequest`, `updateRoleRequest`, `createPermissionRequest`, `attachPermissionRequest`, `assignRoleRequest`, `checkPermissionRequest`/`checkPermissionResponse`

## Helpers
- `auditRBAC` (line 18) — `actor_type=service`.
- `permToResponse` (line 86), `roleToResponse` (line 95)

## Imports of note
- `internal/storage` — `Role`, `Permission`, all RBAC tables
- `gonanoid` — id suffixes (`role_*`, `perm_*`)

## Wired by
- `internal/api/router.go:396-419` (roles, permissions, user roles, auth check)
- `internal/api/router.go:605` (batch usage)

## Notes
- Conflict checks on role.Name and permission.(Action,Resource) before insert.
- `handleAuthCheck` is the predicate endpoint other services can hit to ask "can user X do action Y on Z?".
