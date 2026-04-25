# sessions.tsx

**Path:** `admin/src/components/sessions.tsx`
**Type:** React component (page)
**LOC:** 600+

## Purpose
Active sessions browser—live header strip, geo map, paginated table with search/filters, detail slide-over. Supports JTI revocation for JWT mode.

## Exports
- `Sessions()` (default) — function component

## Features
- **Live tail** — 5s polling, pulse animation when enabled
- **Live header strip** — total active, suspicious count, client breakdown (web|mobile|api|agent), region distribution, MFA rate, auth method breakdown
- **Search** — email, IP, city
- **Filters** — client type (web|mobile|api|agent), risk level (low|high|critical)
- **Geo view** — map view with session locations (table view by default)
- **Table columns** — user, email, IP, city, client, risk, auth method, MFA, created, last active, expires, actions
- **JTI revocation** — input for JWT JTI, revoke single token
- **Revoke actions** — revoke single session, revoke all (with confirmation)
- **Session normalization** — handles both API and mock field names

## Hooks used
- `useAPI('/admin/sessions')` — fetch all active sessions
- `useURLParam('client', 'all')` — filter client type

## State
- `selected` — detail view session
- `query` — search text
- `clientFilter`, `riskFilter` — active filters
- `view` — table|geo view mode
- `liveTail` — polling enabled
- `pulse` — animation tick counter
- `jtiInput` — JTI revocation input

## API calls
- `GET /api/v1/admin/sessions` — list all active sessions
- `DELETE /api/v1/admin/sessions/{sessionId}` — revoke single session
- `DELETE /api/v1/admin/sessions` — revoke all sessions
- `POST /api/v1/admin/auth/revoke-jti` — revoke by JWT JTI

## Composed by
- App.tsx

## Notes
- Session normalization maps various API/mock field names to UI shape
- Relative time formatting: "5m ago", "in 2h", etc.
- Risk levels: low|high|critical with color-coded badges
