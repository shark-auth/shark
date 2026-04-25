# rbac.tsx

**Path:** `admin/src/components/rbac.tsx`
**Type:** React component (page)
**LOC:** 500+

## Purpose
Role-based access control (RBAC) manager—two views: role list/detail (left pane) and permission matrix grid.

## Exports
- `RBAC()` (default) — function component
- `PermissionMatrix` (imported from rbac_matrix.tsx)

## Features
- **View toggle** — Roles list view vs Matrix grid view
- **Role list** (left pane) — create inline, search, select role
- **Detail pane** — role info, permissions checklist, member assignments
- **Matrix view** — roles × permissions grid, bulk assignment
- **Create flow** — inline form with name + optional description
- **Delete/edit** — roles can be modified and removed

## Hooks used
- `useAPI('/roles')` — list all roles
- `useURLParam('view', 'roles')` — view toggle in URL
- `useTabParam()` — detail tabs
- `usePageActions({ onNew, onRefresh })`

## State
- `view` — roles|matrix
- `selectedId` — current role
- `creating` — create form visibility
- `newName`, `newDesc` — create form inputs

## API calls
- `GET /api/v1/roles` — list roles
- `POST /api/v1/roles` — create role (name, description)
- `PATCH /api/v1/roles/{id}` — update role
- `DELETE /api/v1/roles/{id}` — delete role
- `POST /api/v1/roles/{id}/permissions` — set permissions for role

## Composed by
- App.tsx

## Notes
- Auto-select first role on load
- Empty state with TeachEmptyState
- Matrix view useful for bulk permission assignment
