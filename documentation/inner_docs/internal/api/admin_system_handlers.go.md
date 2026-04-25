# admin_system_handlers.go

**Path:** `internal/api/admin_system_handlers.go`
**Package:** `api`
**LOC:** 782
**Tests:** none colocated (largest admin file)

## Purpose
Catch-all admin "system" surface: health probe, runtime config view + patch, log streaming (SSE), test email, email previews, signing-key rotation, MFA disable, passkey list, plus admin reads for orgs/sessions. Read-mostly views the dashboard's Settings + Health pages depend on.

## Handlers exposed
- `handleAdminHealth` (line 301) — GET `/admin/health`. Aggregates DB driver/size, migration version, JWT mode/active keys, SMTP config, OAuth providers, SSO connection count.
- `handleAdminConfig` (line 370) — GET `/admin/config`. Returns `adminConfigSummary`.
- `handleAdminUpdateConfig` (line 581) — PATCH `/admin/config`. Field-allowlisted live config edits (server, auth, passkey, email, audit, jwt, magic_link, password_reset, social).
- `handleAdminLogStream` (line 747) — GET `/admin/logs/stream` (SSE).
- `handleAdminListOrganizations` (line 375), `handleAdminGetOrganization` (line 389), `handleAdminListOrgMembers` (line 400) — admin reads (writes live in `admin_organization_handlers.go`).
- `handleAdminTestEmail` (line 411) — POST `/admin/test-email`.
- `handleAdminListUserPasskeys` (line 445), `handleAdminDisableUserMFA` (line 459).
- `handleAdminEmailPreview` (line 498) — GET `/admin/email-preview/{template}`.
- `handleAdminRotateSigningKey` (line 556) — POST. Mints new JWKS keypair.

## Key types
- `adminHealthResponse` (line 21) + nested `healthDBSection`, `healthMigrations`, `healthJWTSection`, `healthSMTPSection`.
- `adminConfigSummary` (line 63) + per-section structs: `adminServerConfig`, `adminAuthConfig`, `adminPasskeyConfig`, `adminEmailConfig`, `adminAuditConfig`, `adminJWTConfig`, `adminMagicLinkConfig`, `adminPasswordResetConfig`, `adminSocialConfig`, `adminOAuthCreds`.

## Helpers
- `resolveAppVersion` (line 146), `dbSizeBytes` (line 154), `buildConfigSummary` (line 166), `hideSecret` (line 282), `currentMigrationVersion` (line 290).

## Wired by
- `internal/api/router.go:581-594` (health/config/logs/stats/sessions/audit-logs/test-email/email-preview/rotate-signing-key)

## Notes
- `dbSizeBytes` uses SQLite-only `PRAGMA page_count * page_size`; non-SQLite drivers return 0.
- Secrets in config responses are masked via `hideSecret`.
