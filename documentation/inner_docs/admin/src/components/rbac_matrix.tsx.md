# rbac_matrix.tsx

**Path:** `admin/src/components/rbac_matrix.tsx`
**Type:** React component
**LOC:** 343

## Purpose
Permission matrix grid — rows = permissions, columns = roles, cells = checkboxes indicating role/permission membership. Optimistic UI with rollback on error and a column-header "grant all / revoke all" bulk operation.

## Exports
- `PermissionMatrix` — `{onSwitchToRoles}`.

## Props / hooks
- `onSwitchToRoles()` — handler invoked from the empty state CTA when no roles exist.
- `useAPI('/roles')`, `useAPI('/permissions')`.
- `useAPI('/admin/permissions/batch-usage?ids=...')` for "used by N users" badge counts.
- State: `matrix` (`{[roleId]: Set<pid>}`), `matrixLoaded`, `filter`, `pending` (Set of in-flight cell keys).
- `useToast()`.

## API calls
- GET `/api/v1/roles` — list roles.
- GET `/api/v1/permissions` — list permissions.
- GET `/api/v1/roles/{id}` — fan-out per role to fetch its permissions (parallel `Promise.all` on mount).
- GET `/api/v1/admin/permissions/batch-usage?ids=...` — usage counts.
- POST `/api/v1/roles/{id}/permissions` `{permission_id}` — grant.
- DELETE `/api/v1/roles/{id}/permissions/{pid}` — revoke.

## Composed by
- RBAC page in `App.tsx` route table; falls back to `TeachEmptyState` when no roles exist (CTA → `onSwitchToRoles`).

## Notes
- `pending` Set keyed by `roleId:permissionId` disables individual checkboxes during the round-trip.
- `bulkToggleColumn` confirms with `window.confirm`, iterates serially through filtered permissions, and reports successes / failures via toast.
- Sorting: permissions sorted by `${action}·${resource}` for stable ordering.
- Loading skeleton renders a 6-row grid sized to `roles.length || 3`.
