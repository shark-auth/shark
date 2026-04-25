# storage.go

**Path:** `internal/storage/storage.go`
**Package:** `storage`
**LOC:** 794
**Tests:** indirect — storage interface is exercised by every domain `*_test.go` in this dir.

## Purpose
Master contract: declares the `Store` interface every persistence backend must satisfy, plus all entity types, query option structs, and domain constants. Every other file in the dir is an implementation of a slice of this interface (currently the only backend is SQLite via `sqlite*.go` files).

## Store interface — methods grouped by domain

### Users (lines 27-38)
- `CreateUser`, `GetUserByID`, `GetUserByEmail`, `ListUsers`, `UpdateUser`, `DeleteUser`
- `MarkWelcomeEmailSent` — atomic flag-flip; returns `sql.ErrNoRows` to signal "already sent"

### Sessions (lines 41-49)
- `CreateSession`, `GetSessionByID`, `GetSessionsByUserID`, `DeleteSession`
- `DeleteExpiredSessions`, `UpdateSessionMFAPassed`, `ListActiveSessions`
- `DeleteSessionsByUserID`, `DeleteAllActiveSessions`

### Stats / metrics (lines 52-61)
- `CountUsers`, `CountUsersCreatedSince`, `CountActiveSessions`, `CountMFAEnabled`
- `CountFailedLoginsSince`, `CountExpiringAPIKeys`
- `CountSSOConnections`, `CountSSOIdentitiesByConnection`
- `GroupSessionsByAuthMethodSince`, `GroupUsersCreatedByDay`

### DevEmails (lines 64-67)
- `CreateDevEmail`, `ListDevEmails`, `GetDevEmail`, `DeleteAllDevEmails`

### Organizations (lines 70-76)
- `CreateOrganization`, `GetOrganizationByID`, `GetOrganizationBySlug`
- `UpdateOrganization`, `DeleteOrganization`
- `ListOrganizationsByUserID`, `ListAllOrganizations`

### Organization members (lines 79-85)
- `CreateOrganizationMember`, `GetOrganizationMember`
- `UpdateOrganizationMemberRole`, `DeleteOrganizationMember`
- `ListOrganizationMembers`, `CountOrganizationMembers`, `CountOrganizationsByRole`

### Organization invitations (lines 88-94)
- `CreateOrganizationInvitation`, `GetOrganizationInvitationByID`, `GetOrganizationInvitationByTokenHash`
- `MarkOrganizationInvitationAccepted`, `ListOrganizationInvitationsByOrgID`
- `UpdateOrganizationInvitationToken`, `DeleteOrganizationInvitation`

### Webhooks (lines 97-102) + WebhookDeliveries (lines 105-110)
- CRUD + `ListEnabledWebhooksByEvent`
- Delivery: `Create`, `Update`, `Get`, `List...ByWebhookID`, `ListPendingWebhookDeliveries`, `DeleteWebhookDeliveriesBefore`

### OAuthAccounts (lines 113-116)
- `CreateOAuthAccount`, `GetOAuthAccountByProviderID`, `GetOAuthAccountsByUserID`, `DeleteOAuthAccount`

### Passkeys (lines 119-123) — WebAuthn credentials
- `CreatePasskeyCredential`, `GetPasskeyByCredentialID`, `GetPasskeysByUserID`, `UpdatePasskeyCredential`, `DeletePasskeyCredential`

### MagicLinks (lines 126-129)
- `CreateMagicLinkToken`, `GetMagicLinkTokenByHash`, `MarkMagicLinkTokenUsed`, `DeleteExpiredMagicLinkTokens`

### MFA recovery codes (lines 132-135)
- `CreateMFARecoveryCodes`, `GetMFARecoveryCodesByUserID`, `MarkMFARecoveryCodeUsed`, `DeleteAllMFARecoveryCodesByUserID`

### Roles + Permissions + RolePermissions + UserRoles (lines 138-167)
- Roles CRUD; Permissions CRUD
- `AttachPermissionToRole` / `DetachPermissionFromRole`
- `GetPermissionsByRoleID`, `GetRolesByPermissionID`, `GetUsersByPermissionID`
- `BatchCountRolesByPermissionIDs`, `BatchCountUsersByPermissionIDs` — N+1 killers
- `AssignRoleToUser`, `RemoveRoleFromUser`, `GetRolesByUserID`, `GetUsersByRoleID`

### SSO (lines 170-180)
- Connections CRUD; `GetSSOConnectionByDomain`
- `CreateSSOIdentity`, `GetSSOIdentityByConnectionAndSub`, `GetSSOIdentitiesByUserID`

### APIKeys (lines 183-189)
- `CreateAPIKey`, `GetAPIKeyByKeyHash`, `GetAPIKeyByID`, `ListAPIKeys`
- `UpdateAPIKey`, `RevokeAPIKey`, `CountActiveAPIKeysByScope`

### Audit logs (lines 192-195)
- `CreateAuditLog`, `GetAuditLogByID`, `QueryAuditLogs`, `DeleteAuditLogsBefore`

### Migrations (Auth0 import, lines 198-201)
- `CreateMigration`, `GetMigrationByID`, `ListMigrations`, `UpdateMigration`

### Org RBAC (lines 204-216) — parallel to global RBAC
- `CreateOrgRole`, `Get/Update/Delete OrgRole`, `GetOrgRoleByName`
- `AttachOrgPermission`, `DetachOrgPermission`, `GetOrgRolePermissions`
- `GrantOrgRole`, `RevokeOrgRole`, `GetOrgUserRoles`

### JWT signing keys + revoked JTIs (lines 219-229)
- `InsertSigningKey`, `GetActiveSigningKey(ByAlgorithm)`, `GetSigningKeyByKID`
- `RotateSigningKeys`, `ListJWKSCandidates`
- `InsertRevokedJTI`, `IsRevokedJTI`, `PruneExpiredRevokedJTI`

### Applications (lines 232-241)
- `CreateApplication`, `GetApplicationByID/ClientID/Slug/ProxyDomain`, `GetDefaultApplication`
- `ListApplications`, `UpdateApplication`, `RotateApplicationSecret`, `DeleteApplication`

### Agents — OAuth 2.1 clients (lines 244-253)
- `CreateAgent`, `GetAgentByID`, `GetAgentByClientID`, `ListAgents`
- `UpdateAgent`, `UpdateAgentSecret`, `DeactivateAgent`
- `RotateDCRClientSecret` — 1-hour grace window for DCR rotations (F4.3)

### OAuth — Authorization codes / PKCE / Tokens / Consents (lines 256-285)
- Auth codes: `CreateAuthorizationCode`, `Get`, `Delete`, `DeleteExpiredAuthorizationCodes`
- PKCE: `CreatePKCESession`, `Get`, `Delete`, `DeleteExpiredPKCESessions`
- Tokens: `CreateOAuthToken`, `GetOAuthTokenByJTI`, `GetOAuthTokenByHash`, `GetActiveOAuthTokenByRequestIDAndType`, `RevokeOAuthToken`, `RevokeActiveOAuthTokenByRequestID`, `RevokeOAuthTokensByClientID`, `RevokeOAuthTokenFamily`, `ListOAuthTokensByAgentID`, `DeleteExpiredOAuthTokens`, `UpdateOAuthTokenDPoPJKT`
- Consents: `CreateOAuthConsent`, `GetActiveConsent`, `ListConsentsByUserID`, `ListAllConsents`, `RevokeOAuthConsent`

### Device codes — RFC 8628 (lines 288-294)
- `CreateDeviceCode`, `GetDeviceCodeByUserCode`, `GetDeviceCodeByHash`
- `ListPendingDeviceCodes`, `UpdateDeviceCodeStatus`, `UpdateDeviceCodePolledAt`, `DeleteExpiredDeviceCodes`

### DCR clients — RFC 7591 (lines 297-302)
- `CreateDCRClient`, `GetDCRClient`, `UpdateDCRClient`, `DeleteDCRClient`
- `RotateDCRRegistrationToken`

### Vault providers + connections (lines 305-322)
- Providers CRUD; `GetVaultProviderByName`
- Connections: CRUD plus `GetVaultConnection(providerID,userID)`, `ListVaultConnectionsByUserID/ProviderID`, `ListAllVaultConnections`, `UpdateVaultConnectionTokens`, `MarkVaultConnectionNeedsReauth`

### Auth Flows + Runs (lines 325-334)
- Flows CRUD; `ListAuthFlowsByTrigger`
- `CreateAuthFlowRun`, `ListAuthFlowRunsByFlowID`

### Branding (lines 337-341)
- `GetBranding`, `UpdateBranding` (allowlisted fields incl. design_tokens JSON)
- `SetBrandingLogo`, `ClearBrandingLogo`, `ResolveBranding(appID)` — global + per-app override

### Email templates (lines 344-347)
- `ListEmailTemplates`, `GetEmailTemplate`, `UpdateEmailTemplate`, `SeedEmailTemplates`

### Proxy rules — Phase 6.6 / Wave D (lines 351-356)
- `CreateProxyRule`, `GetProxyRuleByID`, `ListProxyRules`, `ListProxyRulesByAppID`
- `UpdateProxyRule`, `DeleteProxyRule`

### User tier — PROXYV1_5 §4.10 (lines 362-363)
- `SetUserTier`, `GetUserTier` — tier stored inside `users.metadata` JSON, not its own column

## Entity types defined here
`User`, `Session`, `OAuthAccount`, `PasskeyCredential`, `MagicLinkToken`, `MFARecoveryCode`, `Role`, `Permission`, `SSOConnection`, `SSOIdentity`, `APIKey`, `AuditLog`, `Migration`, `Organization`, `OrganizationMember(WithUser)`, `OrganizationInvitation`, `Webhook`, `WebhookDelivery`, `DevEmail`, `SessionWithUser`, `MethodCount`, `DayCount`, `SigningKey`, `EmailTemplate`, `BrandingConfig`, `Application`, `OrgRole` (top of file).

## Query option types
`ListUsersOpts`, `ListSessionsOpts` (cursor-based), `AuditLogQuery` (cursor-based).

## Constants
- Org roles: `owner`, `admin`, `member` (lines 540-543)
- Webhook delivery statuses: `pending`, `delivered`, `retrying`, `failed` (lines 588-592)
- Webhook event names: `user.created`, `user.updated`, etc. (lines 597-608)

## Notes
- This is the **contract**. Implementations live in: `sqlite.go` (users/sessions/stats/audit), `sqlite_orgs.go`, `sqlite_webhooks.go`, `oauth_sqlite.go`, `agents_sqlite.go`, `applications.go`, `auth_flows_sqlite.go`, `vault_sqlite.go`, `branding.go`, `email_templates.go`, `jwt_keys.go`, `org_rbac.go`, `proxy_rules_sqlite.go`, `user_tier.go`.
- `Wave 3 agents implement features against this interface` — comment at line 21 indicates the pattern: contract first, then per-domain SQLite file.
- Adding a new method requires touching `storage.go` + the relevant impl file + tests.
