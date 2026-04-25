# org_rbac.go

**Path:** `internal/storage/org_rbac.go`
**Package:** `storage`
**LOC:** 198
**Tests:** `org_rbac_test.go`

## Purpose
SQLite implementation of org-scoped RBAC. Operates on a parallel set of tables (`org_roles`, `org_role_permissions`, `org_user_roles`) so the global RBAC tables (`roles`, `permissions`, `role_permissions`, `user_roles`) stay untouched. The `OrgRole` type lives in `storage.go`.

## Interface methods implemented
- `CreateOrgRole` (14)
- `GetOrgRoleByID` (24)
- `GetOrgRolesByOrgID` (45)
- `GetOrgRolesByUserID` (65) — joins `org_user_roles`
- `GetOrgRoleByName` (88)
- `UpdateOrgRole` (109), `DeleteOrgRole` (118)
- `AttachOrgPermission` (123) / `DetachOrgPermission` (131) — `INSERT OR IGNORE` for idempotency
- `GetOrgRolePermissions` (139)
- `GrantOrgRole` (159) / `RevokeOrgRole` (168)
- `GetOrgUserRoles` (176) — alias of `GetOrgRolesByUserID` for the interface contract

## Tables touched
- org_roles
- org_role_permissions
- org_user_roles

## Imports of note
- `context`, `time`

## Used by
- `internal/api/org_rbac.go` admin handlers
- `internal/api/middleware/org.go` for permission checks
- Bootstrap code that seeds builtin org roles (owner/admin/member parallel)

## Notes
- Permissions are denormalized as (action, resource) pairs directly on `org_role_permissions` — no separate `permissions` table — because org-scoped permissions are typically per-tenant ad-hoc.
- `is_builtin` flag prevents accidental deletion of seeded org roles (handler enforces).
- Helper `scanOrgRole` (line 181) accepts a `Scan(...)` interface so the same code serves both `*sql.Row` and `*sql.Rows`.
