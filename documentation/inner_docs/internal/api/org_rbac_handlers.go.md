# org_rbac_handlers.go

**Path:** `internal/api/org_rbac_handlers.go`
**Package:** `api`
**LOC:** 284
**Tests:** likely integration-tested

## Purpose
Per-organization RBAC: roles within an org, permission attach/detach, and member role grant/revoke + effective permission inspection. Distinct from the global RBAC in `rbac_handlers.go` — these routes are session-gated and use `RequireOrgPermission` middleware on the wire side.

## Handlers exposed
- `handleListOrgRoles` (line 88) — GET `/organizations/{id}/roles`.
- `handleCreateOrgRole` (line 103) — POST.
- `handleGetOrgRole` (line 128) — GET `/{role_id}` with permissions inlined.
- `handleUpdateOrgRole` (line 152) — PATCH; supports `attach_permissions` + `detach_permissions` arrays in the body.
- `handleDeleteOrgRole` (line 214) — DELETE.
- `handleGrantOrgRole` (line 238) — POST `.../members/{user_id}/roles/{role_id}`.
- `handleRevokeOrgRole` (line 254) — DELETE `.../members/{user_id}/roles/{role_id}`.
- `handleGetEffectiveOrgPerms` (line 270) — GET `.../members/{user_id}/permissions`. Returns the effective permission set computed across the user's org roles.

## Key types
- `orgRoleResponse` (line 19), `orgRoleWithPermsResponse` (line 63)
- `createOrgRoleRequest` (line 41), `updateOrgRoleRequest` (line 46) (with attach/detach permission arrays)
- `orgPermInput` (line 53) / `orgPermResponse` (line 58) — `{action, resource}` pairs.

## Helpers
- `auditOrgRBAC` (line 69) — uniform audit wrapper with `target_type=org_role`.

## Imports of note
- `internal/rbac` — `RBACManager.CreateOrgRole`, `AttachOrgPermission`, `DetachOrgPermission`
- `internal/storage` — `OrgRole`, `GetOrgRolePermissions`
- `internal/api/middleware` (`mw.GetUserID`)

## Wired by
- `internal/api/router.go:382-389` (mounted under org subrouter with `RequireOrgPermission` middleware).

## Notes
- `handleUpdateOrgRole` accepts attach + detach in a single PATCH so dashboards can save the whole permission set in one round-trip.
- All audits target `org_role` even for member grant/revoke (so the dashboard timeline stays role-centric).
