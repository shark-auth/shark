# organizations.tsx

**Path:** `admin/src/components/organizations.tsx`
**Type:** React component (page)
**LOC:** 600+

## Purpose
Organizations manager—split-pane UI (org list left, detail/tabs right), members, invitations, roles, SSO config, audit, settings.

## Exports
- `Organizations()` (default) — function component
- `OrgListItem(props)` — list item sub-component
- `OrgDetail(props)` — detail pane sub-component

## Features
- **Split pane** — persistent left list, right detail scrolls independently
- **Search** — filter orgs by name or slug
- **Tabs** — overview, members, invitations, roles, SSO, audit, settings
- **Detail sections** — org info, member list, role list, SSO config
- **Auto-select** — first org on load, sticky selection in localStorage
- **Create modal** — `?new=1` opens create slide-over

## Hooks used
- `useAPI('/admin/organizations')` — list all orgs
- `useAPI('/admin/organizations/{id}/members')` — members of org
- `useAPI('/admin/organizations/{id}/roles')` — roles in org
- `useTabParam('overview')` — tab state in URL
- `usePageActions({ onRefresh, onNew })` — keyboard shortcuts

## State
- `selectedId` (localStorage: `org-selected`) — current org
- `query` — search text
- `creating` — show create modal

## API calls
- `GET /api/v1/admin/organizations` — list orgs
- `GET /api/v1/admin/organizations/{id}/members` — members
- `GET /api/v1/admin/organizations/{id}/roles` — roles

## Composed by
- App.tsx

## Notes
- Empty state with TeachEmptyState component
- Org avatars use `hashColor(org.name)` for consistent coloring
- Breadcrumb shows member count, role count on tabs
