# organization_handlers.go

**Path:** `internal/api/organization_handlers.go`
**Package:** `api`
**LOC:** 589
**Tests:** `organizations_test.go`

## Purpose
User-facing organization CRUD, membership management, and email invitation flow under `/api/v1/organizations/*`. Session-cookie auth + per-org RBAC permission gates (admin-key org admin lives in `admin_organization_handlers.go`).

## Handlers exposed
- `handleCreateOrganization` (func, line 52) — `POST /api/v1/organizations`; slug regex `^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`; creator becomes `OrgRoleOwner`; seeds 3 builtin org roles via `RBAC.SeedOrgRoles`, grants owner role; on RBAC failure compensates by deleting the org; emits `WebhookEventOrgCreated`
- `handleListMyOrganizations` (func, line 125) — `GET /api/v1/organizations`
- `handleGetOrganization` (func, line 145) — `GET /{id}`; 404 (not 403) when caller isn't a member to avoid existence leak
- `handleUpdateOrganization` (func, line 169) — `PATCH /{id}`; pointer-field PATCH for name/metadata
- `handleDeleteOrganization` (func, line 198) — `DELETE /{id}`
- `handleListOrganizationMembers` (func, line 219) — `GET /{id}/members`
- `handleUpdateOrganizationMemberRole` (func, line 245) — `PATCH /{id}/members/{uid}`; refuses to demote the last owner; syncs RBAC role assignment best-effort
- `handleRemoveOrganizationMember` (func, line 302) — `DELETE /{id}/members/{uid}`; refuses to remove last owner
- `handleCreateOrgInvitation` (func, line 346) — `POST /{id}/invitations`; mints 32-byte hex token, stores SHA-256 hash, expires in 72h, fire-and-forget invitation email
- `handleAcceptOrgInvitation` (func, line 413) — `POST /api/v1/organizations/invitations/{token}/accept`; requires session; matches `token_hash`, enforces email match, idempotent membership creation

## Key types
- `createOrgRequest`, `organizationResponse` (lines 30, 36); `updateOrgRequest` (164); `memberResponse` (211); `updateMemberRoleRequest` (241); `inviteRequest`/`inviteResponse` (330, 335)
- `slugRE` (var, line 26)

## Helpers
- `isOrgMember` (478), `refuseIfLastOwner` (486), `sendOrgInvitationEmail` (503; 15s ctx-detached), `emailSender` (553), `isValidOrgRole` (560), `newInvitationToken` (564), `hashInvitationToken` (574; SHA-256), `normalizeMetadata` (579), `errPayload` (587), `auditOrg` (540)

## Imports of note
- `internal/api/middleware` (mw) — `GetUserID`
- `internal/storage` — `Organization`, `OrganizationMember`, `OrganizationInvitation`, `OrgRoleOwner/Admin/Member`, `WebhookEventOrgCreated/MemberAdded`
- `internal/email` — `RenderOrganizationInvitation`

## Wired by / used by
- Routes registered in `internal/api/router.go:364–391` — RBAC permissions enforced via `rbacpkg.RequireOrgPermission`

## Notes
- Token plaintext only lives in memory during create — DB stores SHA-256 hash so leaks can't reveal valid invite URLs.
- Invitation accept matches caller email to invitation email to block token sharing.
- "Last owner" guard surfaces as `409 last_owner` — no transfer-ownership flow exists yet.
