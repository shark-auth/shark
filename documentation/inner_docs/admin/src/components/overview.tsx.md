# overview.tsx

**Path:** `admin/src/components/overview.tsx`
**Type:** React component (page)
**LOC:** 500+

## Purpose
Dashboard overview page—metrics (users, sessions, MFA, failures), trends graph, real-time activity stream via SSE, health status, auth method breakdown donut chart.

## Exports
- `Overview({ setPage })` (default) — function component

## Props
- `setPage?: (page, extra?) => void` — navigate to other pages

## Features
- **Magical moment CTA** — "Add auth to any app in 60s" proxy hero (shown on cold start)
- **Stats cards** — users, sessions, MFA%, failures, API keys, agents with 7d deltas
- **Trends sparkline** — 14-day signup trend
- **Live activity stream** — SSE `GET /api/v1/admin/logs/stream?token=...` with retry backoff
- **Auth method breakdown** — donut chart (password|oauth|passkey|magic_link)
- **Health panel** — version, uptime, DB size, migrations, JWT, SMTP, OAuth, SSO
- **Attention panel** — priority alerts

## Hooks used
- `useAPI('/admin/stats')` — system stats
- `useAPI('/admin/stats/trends?days=14')` — signup trends
- `useAPI('/admin/health')` — system health
- `useAPI('/agents?limit=1')` — agent count
- `useRealtimeActivity()` — SSE stream (internal custom hook)
- `useProxyConfigured()` — check if proxy is set up

## State
- `heroHidden` (localStorage: `shark_hide_hero`) — hide proxy CTA after dismissed
- `showHero` — conditional hero visibility

## API calls
- `GET /api/v1/admin/stats` — system stats
- `GET /api/v1/admin/stats/trends?days=14` — signup trends
- `GET /api/v1/admin/health` — full system health
- `GET /api/v1/admin/proxy/status` — 404 if proxy disabled
- `GET /api/v1/agents?limit=1` — agent count

## SSE stream
- Endpoint: `/api/v1/admin/logs/stream?token=<key>`
- Events: JSON-parsed messages including actor, action, resource, timestamp
- Keeps last 50 events in memory
- Backoff retry: 1s, 2s, 4s, ... up to 30s

## Composed by
- App.tsx

## Notes
- Onboarding detection: if users.total === 0 && proxy not configured → show hero
- UX uses soft redirects via `setPage('proxy', { new: '1' })`
