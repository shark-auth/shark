# compliance.tsx

**Path:** `admin/src/components/compliance.tsx`
**Type:** React component (page)
**LOC:** ~300

## Purpose
Compliance controls—data retention policies, GDPR/CCPA support, audit log archival, consent management.

## Exports
- `CompliancePage()` (default) — function component

## Features
- **Data retention** — auto-delete sessions, logs after N days
- **Export controls** — DPA, SOC2, HIPAA attestations
- **Consent** — user consent tracking, cookie banner integration
- **Right to deletion** — bulk user deletion, GDPR compliance
- **Audit archival** — export audit logs to cold storage

## Hooks used
- `useAPI('/admin/compliance')` — fetch settings
- `useToast()` — feedback

## Composed by
- App.tsx

## Notes
- Enterprise feature addressing regulatory requirements
- Audit trails immutable for compliance
- Data deletion requests logged and tracked
