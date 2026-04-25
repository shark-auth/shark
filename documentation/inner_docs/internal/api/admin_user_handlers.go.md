# admin_user_handlers.go

**Path:** `internal/api/admin_user_handlers.go`
**Package:** `api`
**LOC:** 118
**Tests:** none colocated

## Purpose
Admin-key path for user creation (T04 of DASHBOARD_DX_EXECUTION_PLAN). Parallels `admin_organization_handlers.go`: dashboard pages send the admin Bearer key, so creating a user from the dashboard lands here instead of `/auth/signup`. Password is optional — invite-via-magic-link is a separate task (T05/T06).

## Handlers exposed
- `handleAdminCreateUser` (line 35) — POST `/api/v1/admin/users`. Validates email shape + uniqueness (`emailRegex` from elsewhere in package), optionally hashes password via `auth.HashPassword(..., Argon2id)` and enforces complexity, persists `usr_*` row with `email_verified` honored as-supplied, audits `admin.user.create`, emits `WebhookEventUserCreated`.

## Key types
- `adminCreateUserRequest` (line 24) — `{email, password?, name?, email_verified?}`

## Imports of note
- `internal/auth` — `ValidatePasswordComplexity`, `HashPassword`
- `internal/storage` — User + AuditLog
- `gonanoid` — id suffix

## Wired by
- `internal/api/router.go:629` (`POST /admin/users`)

## Notes
- When `password == ""` the user lands with nil `password_hash` (caller wires invite flow later).
- Default `password_min_length` = 8 if config is zero.
- `metadata` is initialised to `"{}"` so the JSON column never starts NULL.
- Response goes through `adminUserToResponse` (defined elsewhere in package).
