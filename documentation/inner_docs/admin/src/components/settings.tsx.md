# settings.tsx

**Path:** `admin/src/components/settings.tsx`
**Type:** React component (page)
**LOC:** 424

## Purpose
Settings page — server runtime info (read-only), editable auth/passkey/email/audit configuration, and a danger-zone with session purge + audit-log purge. Single PATCH submits all dirty fields.

## Exports
- `Settings` (named) — page component.
- Internal helpers: `ConfigRow`, `SectionHeader`, `Skeleton`, `formatUptime`, `formatBytes`.

## Props / hooks
- props: `{}`.
- `useAPI('/admin/health')` and `useAPI('/admin/config')` for live state.
- State: `formState` (nested auth/passkeys/email/audit), `isDirty`, `saving`, local `toast`, `purgingSession`, `purgingAudit`, `auditBefore` (date input).

## API calls
- GET `/api/v1/admin/health`
- GET `/api/v1/admin/config`
- PATCH `/api/v1/admin/config` (full nested formState).
- POST `/api/v1/admin/sessions/purge-expired`.
- POST `/api/v1/admin/audit-logs/purge` `{before: ISO}`.
- POST `/api/v1/admin/test-email` `{to}`.

## Composed by
- `App.tsx` route (settings page).

## Notes
- `updateForm('a.b.c', val)` walks the dotted path immutably so React re-renders only the touched branch.
- `formState` reset whenever `config` reloads; `isDirty` cleared after Save.
- Self-rolled toast (`setTimeout` + local state) instead of the shared `useToast()` provider — predates the migration.
- Sections rendered: Server (system), Authentication, Passkeys, Email, Audit, Danger Zone.
- Trailing `<CLIFooter>` mirrors CLI parity.
