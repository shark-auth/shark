# login.tsx

**Path:** `admin/src/components/login.tsx`
**Type:** React component
**LOC:** 200+

## Purpose
Admin login page—accepts API key or bootstrap link.

## Exports
- `Login({ onLogin })` (default) — function component

## Props
- `onLogin: (apiKey: string) => void` — callback on successful auth

## Features
- **API key input** — paste `sk_live_…` key
- **Bootstrap link** — `?bootstrap=<token>` auto-consumes token and mints key
- **Key validation** — hits `GET /api/v1/admin/stats` to verify
- **Error messages** — "Invalid API key", "Bootstrap expired", etc.
- **Help tooltip** — "Where is my key?" expands CLI hint

## State
- `key` (text input) — API key being entered
- `loading` (bool) — during validation
- `error` (string) — validation error message
- `hintOpen` (bool) — help text visibility
- `bootstrapTried` (ref) — prevents retry loop on re-mount

## API calls
- `POST /api/v1/admin/bootstrap/consume` — exchange bootstrap token for API key
- `GET /api/v1/admin/stats` — validate API key

## Notes
- Bootstrap token stripped from URL after attempt (prevent replay/leakage)
- Stores validated key in localStorage (`shark_admin_key`)
- CLI hint shows `shark admin-key show` command
