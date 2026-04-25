# session_debugger.tsx

**Path:** `admin/src/components/session_debugger.tsx`
**Type:** React component (page)
**LOC:** ~400

## Purpose
Session debugger—deep dive into a single session: decode JWT, inspect claims, verify signatures, test token usage.

## Exports
- `SessionDebugger()` (default) — function component

## Features
- **Token input** — paste JWT token
- **Decode view** — header, payload, signature tabs
- **Claim inspector** — all claims decoded and formatted
- **Signature verification** — check if signature valid with current keys
- **Token metadata** — issued at, expires at, remaining lifetime
- **Usage test** — make API request with token, see response
- **Refresh token** — decode and verify refresh token

## Hooks used
- `useAPI()` — verify token endpoint

## State
- `tokenInput` — JWT being debugged
- `tab` — header|payload|signature

## API calls
- `POST /api/v1/admin/debug/verify-token` — verify token and return claims

## Composed by
- App.tsx

## Notes
- Developer tool, not for production use
- Helps debug token/session issues
- Can reveal PII in token claims; handle carefully
