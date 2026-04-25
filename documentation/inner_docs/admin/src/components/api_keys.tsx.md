# api_keys.tsx

**Path:** `admin/src/components/api_keys.tsx`
**Type:** React component (page)
**LOC:** 600+

## Purpose
M2M API key management—table, detail pane, create/rotate modals with one-time key reveal, status filters, expiration warnings.

## Exports
- `ApiKeys()` (default) — function component

## Features
- **Reveal banner** — non-dismissable until user acknowledges (dark bg, success border)
- **Stats** — active, expiring (≤7 days), revoked counts
- **Table columns** — name, key prefix (masked), scopes, created, last used, expires, status, actions
- **Filters** — all|active|revoked|expiring status
- **Search** — by name or key prefix
- **Rotation** — reveals new key on rotate
- **Expiration tracking** — flags keys expiring within 7 days
- **Detail pane** — key info, scope matrix, rotation timeline

## Hooks used
- `useAPI('/api-keys')` — list keys
- `useURLParam('status', 'all')` — filter in URL
- `usePageActions({ onNew, onRefresh })`

## State
- `selected` — current key
- `filter`, `env`, `query` — filters & search
- `createOpen`, `rotateModal` — modals
- `revealKey` — revealed key for display

## API calls
- `GET /api/v1/api-keys` — list keys
- `POST /api/v1/api-keys` — create (scopes, expiration)
- `POST /api/v1/api-keys/{id}/rotate` — rotate key
- `DELETE /api/v1/api-keys/{id}` — revoke key
- `PATCH /api/v1/api-keys/{id}` — update metadata

## Composed by
- App.tsx

## Notes
- Key revealed only once; user must copy immediately
- Expiration optional (null = no expiration)
- Scopes are matrix-selectable
