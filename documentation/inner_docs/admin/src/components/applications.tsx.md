# applications.tsx

**Path:** `admin/src/components/applications.tsx`
**Type:** React component (page)
**LOC:** ~600

## Purpose
OAuth 2.1 application (client) management—table of registered apps, detail pane with credentials, redirect URIs, scopes, create/edit modals.

## Exports
- `Applications()` (default) — function component

## Features
- **Table** — app name, client_id, allowed redirect URIs, scopes, created, status
- **Search & filter** — by name or client_id; active|inactive|all
- **Detail pane** — client credentials, secret reveal, redirect URIs list, scope matrix
- **Create flow** — name, redirect URIs, scope selection
- **Secret rotation** — revoke and generate new secret
- **Redirect URI management** — add/remove URIs

## Hooks used
- `useAPI('/applications')` — list apps
- `usePageActions({ onNew, onRefresh })`

## State
- `selected` — current app
- `createOpen` — create modal
- `revealSecret` — secret reveal banner
- `filter`, `query` — filters and search

## API calls
- `GET /api/v1/applications` — list apps
- `POST /api/v1/applications` — create (name, redirect_uris, scopes)
- `PATCH /api/v1/applications/{id}` — update
- `POST /api/v1/applications/{id}/rotate-secret` — rotate secret
- `DELETE /api/v1/applications/{id}` — delete

## Composed by
- App.tsx

## Notes
- Client secret only shown once on creation/rotation
- Scopes selectable from available permissions matrix
- Support for multiple redirect URIs (OAuth 2.1 requirement)
