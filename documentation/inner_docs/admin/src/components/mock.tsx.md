# mock.tsx

**Path:** `admin/src/components/mock.tsx`
**Type:** Mock data module
**LOC:** ~500

## Purpose
Test/demo data for development—mock users, sessions, audit logs, API responses without live backend.

## Exports
- `MOCK` (object) — collection of mock generators and data

## Mock data types
- `users` — array of mock user objects
- `sessions` — array of mock session objects
- `auditLog` — array of mock audit events
- `organizations` — array of mock orgs
- `roles` — array of mock roles
- `apiKeys` — array of mock API keys
- `webhooks` — array of mock webhook configs

## Helper functions
- `relativeTime(timestamp)` — format relative time (5m ago, in 2h)
- `randomUser()` — generate random user
- `randomSession()` — generate random session
- `randomEvent()` — generate random audit event

## Composed by
- Multiple pages (Users, Sessions, Audit, Organizations)

## Notes
- Used in development/storybook without backend
- Realistic data to catch rendering edge cases
- Can be swapped out with real API calls
