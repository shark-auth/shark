# sqlite_orgs.go

**Path:** `internal/storage/sqlite_orgs.go`
**Package:** `storage`
**LOC:** 267
**Tests:** `sqlite_orgs_test.go`

## Purpose
SQLite implementation of organization, organization-member, and organization-invitation methods on `Store`. The B2B multi-tenancy data layer.

## Interface methods implemented
- Organizations: `CreateOrganization` (11), `GetOrganizationByID` (23), `GetOrganizationBySlug` (28), `UpdateOrganization` (33), `DeleteOrganization` (41), `ListOrganizationsByUserID` (46), `ListAllOrganizations` (69)
- Organization members: `CreateOrganizationMember` (99), `GetOrganizationMember` (108), `UpdateOrganizationMemberRole` (121), `DeleteOrganizationMember` (129), `ListOrganizationMembers` (137) (joins `users` for email/name to avoid N+1), `CountOrganizationMembers` (162), `CountOrganizationsByRole` (172) (used to prevent removing last owner)
- Organization invitations: `CreateOrganizationInvitation` (183), `GetOrganizationInvitationByID` (194), `GetOrganizationInvitationByTokenHash`, `MarkOrganizationInvitationAccepted`, `ListOrganizationInvitationsByOrgID`, `UpdateOrganizationInvitationToken`, `DeleteOrganizationInvitation`

## Tables touched
- organizations
- organization_members
- organization_invitations
- users (LEFT JOIN in `ListOrganizationMembers`)

## Imports of note
- `database/sql`
- `strings` — invitation emails are lowercased on insert

## Used by
- `internal/api/orgs.go` and `internal/api/org_invitations.go` handlers
- `internal/api/middleware/org.go` for membership checks
- Webhook emitters when org events fire

## Notes
- Metadata column defaults to `"{}"` if caller passes empty (line 12) so JSON parsers downstream never see NULL.
- Invitation tokens are hashed (SHA-256) before insert — only the hash lives in the DB.
- Owner-protection is done by handler logic via `CountOrganizationsByRole` rather than a DB constraint, so it's revisitable without a migration.
