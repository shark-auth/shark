# admin_organization_handlers.go

**Path:** `internal/api/admin_organization_handlers.go`
**Package:** `api`
**LOC:** 465
**Tests:** none colocated

## Purpose
Admin-key-authenticated parallel of `organization_handlers.go`. Dashboard pages send the admin Bearer key (not a session), so org CRUD goes through this group instead of the session+RBAC user-facing routes. No owner-membership row is created on bootstrap.

## Handlers exposed
- `handleAdminCreateOrganization` (line 44) — POST `/api/v1/admin/organizations`. Validates name + slug (`slugRE`), creates row, seeds builtin RBAC roles via `s.RBAC.SeedOrgRoles`, audits + emits `org.created` webhook.
- `handleAdminUpdateOrganization` (line 111) — PATCH `/admin/organizations/{id}`. Optional fields; rejects duplicate slug.
- `handleAdminDeleteOrganization` (line 166) — DELETE; relies on FK ON DELETE CASCADE; emits `org.deleted`.
- `handleAdminCreateOrgRole` (line 192) — POST `.../{id}/roles`; admin override of session-gated handler.
- `handleAdminListOrgRoles` (line 227) — GET `.../{id}/roles`.
- `handleAdminListOrgInvitations` (line 254) — GET `.../{id}/invitations`.
- `handleAdminDeleteOrgInvitation` (line 289) — DELETE.
- `handleAdminResendOrgInvitation` (line 321) — POST; rotates token, persists hash, sends invitation email.
- `handleAdminRemoveOrgMember` (line 421) — DELETE `.../{id}/members/{uid}`.
- `auditAdminOrg` (line 451, helper) — uniform audit wrapper with `actor_type=admin`.

## Key types
- `adminCreateOrgRequest` (line 31), `adminUpdateOrgRequest` (line 102)

## Imports of note
- `internal/email` — invitation email send via `sendAdminInvitationResend`
- `internal/storage` — Organization + OrganizationInvitation
- `gonanoid` — id suffixes

## Wired by
- `internal/api/router.go:631-642`

## Notes
- Slug regex (`slugRE`) is shared from `organization_handlers.go`.
- Resending an invitation rotates `TokenHash` + `ExpiresAt`; the raw token is delivered only via the email send.
