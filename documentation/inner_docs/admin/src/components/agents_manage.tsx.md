# agents_manage.tsx

**Path:** `admin/src/components/agents_manage.tsx`
**Type:** React component (page)
**LOC:** 500+

## Purpose
Agent (OAuth 2.1 client) management—table of agents with config, grants, scopes; detail pane with config/logs/activity tabs; create modal with one-time secret reveal; deactivate modal.

## Exports
- `Agents()` (default) — function component

## Features
- **Table** — name, client_id, grants (auth_code|client_creds|refresh|device|token-exchange), scopes, created, status (active|inactive)
- **Search & filter** — by name or client_id; active|inactive|all status
- **Detail pane** (right-side) — tabs for config, logs, activity
- **Create flow** — form + one-time secret reveal banner
- **Deactivate flow** — confirm modal, toggle active status
- **Auto-refresh** — reload on create/deactivate

## Hooks used
- `useAPI('/agents?limit=200')` — agent list
- `useTabParam('config')` — tab state
- `usePageActions({ onNew, onRefresh })`

## State
- `selected` — current agent
- `tab` — detail tab (config|logs|activity)
- `filter` — all|active|inactive
- `query` — search text
- `createOpen` — create modal visibility
- `deactivateModal` — deactivate confirmation modal

## API calls
- `GET /api/v1/agents?limit=200` — list agents
- `POST /api/v1/agents` — create new agent
- `PATCH /api/v1/agents/{id}` — toggle active status
- `GET /api/v1/agents/{id}/logs` — activity logs (if detail logs tab)

## Composed by
- App.tsx

## Notes
- Grant types: authorization_code, client_credentials, refresh_token, device_code, token-exchange
- Active count shown in header
- Empty state with TeachEmptyState component
