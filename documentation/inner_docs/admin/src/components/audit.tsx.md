# audit.tsx

**Path:** `admin/src/components/audit.tsx`
**Type:** React component (page)
**LOC:** 700+

## Purpose
Audit log page—live-tail event stream with time-range filters, actor/severity filters, search, CSV export, 60-bucket activity heatmap.

## Exports
- `Audit()` (default) — function component

## Features
- **Live tail** — 5s polling refresh toggle
- **Time range** — 1h|24h|7d|all buttons
- **Actor filter** — by type (system|user|admin|agent)
- **Severity filter** — info|warn|danger color-coded
- **Search** — full-text across action, actor, target, IP
- **Activity bars** — 60 buckets showing event density
- **Detail view** — selected event detail modal
- **Pagination** — cursor-based, "Load more" button
- **CSV export** — selected events

## Hooks used
- `useAPI(apiPath)` — audit log with filters
- `useURLParam('q', ...)` — search in URL
- `useURLParam('range', '24h')` — time range in URL
- `useURLParam('actor', '')` — actor filter in URL
- `usePageActions({ onRefresh })`

## State
- `liveTail` — polling enabled
- `selected` — selected event detail
- `query`, `filters` — search & filters
- `extraPages` — cursor pagination
- `loadingMore` — pagination loading state

## API calls
- `GET /api/v1/audit-logs?limit=...&actor_type=...&action=...&since=...&cursor=...` — list events

## Event normalization
Maps API fields to internal shape: id, timestamp, action, actor_type, actor, severity, ip, target

## Composed by
- App.tsx

## Notes
- Severity derived from status or action name (failure|error=danger, delete|revoke=warn, etc.)
- Cursor pagination supports pagination through large result sets
- Activity heatmap buckets resize based on time range
