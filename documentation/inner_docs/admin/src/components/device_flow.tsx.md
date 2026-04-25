# device_flow.tsx

**Path:** `admin/src/components/device_flow.tsx`
**Type:** React component (page)
**LOC:** ~400

## Purpose
Device authorization flow (RFC 8628)—live device code verification, polling status, success/denial tracking.

## Exports
- `DeviceFlow()` (default) — function component

## Features
- **Active flows** — real-time list of pending device authorizations
- **Device code** — uniquely issued code for command-line/TV apps
- **User code** — short, user-friendly code for user to enter
- **Status** — pending|verified|denied|expired
- **Approve/deny** — admin can approve or reject device flows
- **Time-to-completion** — how long until user enters code

## Hooks used
- `useAPI('/admin/device-flow')` — list flows
- SSE stream for live updates

## State
- `flows` — active device flows
- `live` — live polling toggle

## API calls
- `GET /api/v1/admin/device-flow` — list flows
- `POST /api/v1/admin/device-flow/{code}/approve` — approve flow
- `POST /api/v1/admin/device-flow/{code}/deny` — deny flow

## Composed by
- App.tsx

## Notes
- Device flow useful for CLI, smart TV, IoT devices without browsers
- User-friendly code (e.g., "BCFG9Z") easier to type than full device code
- Live updates via SSE for admin dashboard
