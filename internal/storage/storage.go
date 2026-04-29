package storage

import (
	"context"
	"database/sql"
	"time"
)

// OrgRole represents an org-scoped RBAC role (parallel to the global roles table).
type OrgRole struct {
	ID             string
	OrganizationID string
	Name           string
	Description    string
	IsBuiltin      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Store defines the interface for all database operations.
// Wave 3 agents implement features against this interface.
type Store interface {
	// DB returns the underlying *sql.DB for direct access when needed.
	DB() *sql.DB

	// Users
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsers(ctx context.Context, opts ListUsersOpts) ([]*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	// MarkWelcomeEmailSent atomically flips welcome_email_sent from 0 to 1 for
	// the given user. Returns sql.ErrNoRows when the flag was already set (or
	// the user doesn't exist) — callers treat that as "don't send" for
	// idempotency. The UPDATE ... WHERE welcome_email_sent = 0 guard makes
	// this race-safe across concurrent verifications.
	MarkWelcomeEmailSent(ctx context.Context, userID string) error

	// Sessions
	CreateSession(ctx context.Context, sess *Session) error
	GetSessionByID(ctx context.Context, id string) (*Session, error)
	GetSessionsByUserID(ctx context.Context, userID string) ([]*Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) (int64, error)
	UpdateSessionMFAPassed(ctx context.Context, id string, mfaPassed bool) error
	ListActiveSessions(ctx context.Context, opts ListSessionsOpts) ([]*SessionWithUser, error)
	DeleteSessionsByUserID(ctx context.Context, userID string) ([]string, error)
	DeleteAllActiveSessions(ctx context.Context) (int64, error)

	// Stats / metrics
	CountUsers(ctx context.Context) (int, error)
	CountUsersCreatedSince(ctx context.Context, since time.Time) (int, error)
	CountActiveSessions(ctx context.Context) (int, error)
	CountMFAEnabled(ctx context.Context) (int, error)
	CountFailedLoginsSince(ctx context.Context, since time.Time) (int, error)
	CountExpiringAPIKeys(ctx context.Context, within time.Duration) (int, error)
	CountSSOConnections(ctx context.Context, enabledOnly bool) (int, error)
	CountSSOIdentitiesByConnection(ctx context.Context) (map[string]int, error)
	GroupSessionsByAuthMethodSince(ctx context.Context, since time.Time) ([]MethodCount, error)
	GroupUsersCreatedByDay(ctx context.Context, days int) ([]DayCount, error)

	// DevEmails (dev-mode inbox)
	CreateDevEmail(ctx context.Context, e *DevEmail) error
	ListDevEmails(ctx context.Context, limit int) ([]*DevEmail, error)
	GetDevEmail(ctx context.Context, id string) (*DevEmail, error)
	DeleteAllDevEmails(ctx context.Context) error

	// Organizations
	CreateOrganization(ctx context.Context, o *Organization) error
	GetOrganizationByID(ctx context.Context, id string) (*Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	UpdateOrganization(ctx context.Context, o *Organization) error
	DeleteOrganization(ctx context.Context, id string) error
	ListOrganizationsByUserID(ctx context.Context, userID string) ([]*Organization, error)
	ListAllOrganizations(ctx context.Context) ([]*Organization, error)

	// Organization members
	CreateOrganizationMember(ctx context.Context, m *OrganizationMember) error
	GetOrganizationMember(ctx context.Context, orgID, userID string) (*OrganizationMember, error)
	UpdateOrganizationMemberRole(ctx context.Context, orgID, userID, role string) error
	DeleteOrganizationMember(ctx context.Context, orgID, userID string) error
	ListOrganizationMembers(ctx context.Context, orgID string) ([]*OrganizationMemberWithUser, error)
	CountOrganizationMembers(ctx context.Context, orgID string) (int, error)
	CountOrganizationsByRole(ctx context.Context, userID, role string) (int, error)

	// Organization invitations
	CreateOrganizationInvitation(ctx context.Context, inv *OrganizationInvitation) error
	GetOrganizationInvitationByID(ctx context.Context, id string) (*OrganizationInvitation, error)
	GetOrganizationInvitationByTokenHash(ctx context.Context, tokenHash string) (*OrganizationInvitation, error)
	MarkOrganizationInvitationAccepted(ctx context.Context, id string, acceptedAt string) error
	ListOrganizationInvitationsByOrgID(ctx context.Context, orgID string) ([]*OrganizationInvitation, error)
	UpdateOrganizationInvitationToken(ctx context.Context, id, tokenHash, expiresAt string) error
	DeleteOrganizationInvitation(ctx context.Context, id string) error

	// Webhooks
	CreateWebhook(ctx context.Context, w *Webhook) error
	GetWebhookByID(ctx context.Context, id string) (*Webhook, error)
	ListWebhooks(ctx context.Context) ([]*Webhook, error)
	ListEnabledWebhooksByEvent(ctx context.Context, event string) ([]*Webhook, error)
	UpdateWebhook(ctx context.Context, w *Webhook) error
	DeleteWebhook(ctx context.Context, id string) error

	// WebhookDeliveries
	CreateWebhookDelivery(ctx context.Context, d *WebhookDelivery) error
	CreateWebhookDeliveriesBatch(ctx context.Context, ds []*WebhookDelivery) error
	UpdateWebhookDelivery(ctx context.Context, d *WebhookDelivery) error
	GetWebhookDeliveryByID(ctx context.Context, id string) (*WebhookDelivery, error)
	ListWebhookDeliveriesByWebhookID(ctx context.Context, webhookID string, limit int, cursor string) ([]*WebhookDelivery, error)
	ListPendingWebhookDeliveries(ctx context.Context, now time.Time, limit int) ([]*WebhookDelivery, error)
	DeleteWebhookDeliveriesBefore(ctx context.Context, before time.Time) (int64, error)

	// OAuthAccounts
	CreateOAuthAccount(ctx context.Context, acct *OAuthAccount) error
	GetOAuthAccountByProviderID(ctx context.Context, provider, providerID string) (*OAuthAccount, error)
	GetOAuthAccountsByUserID(ctx context.Context, userID string) ([]*OAuthAccount, error)
	DeleteOAuthAccount(ctx context.Context, id string) error

	// PasskeyCredentials
	CreatePasskeyCredential(ctx context.Context, cred *PasskeyCredential) error
	GetPasskeyByCredentialID(ctx context.Context, credentialID []byte) (*PasskeyCredential, error)
	GetPasskeysByUserID(ctx context.Context, userID string) ([]*PasskeyCredential, error)
	UpdatePasskeyCredential(ctx context.Context, cred *PasskeyCredential) error
	DeletePasskeyCredential(ctx context.Context, id string) error

	// MagicLinkTokens
	CreateMagicLinkToken(ctx context.Context, token *MagicLinkToken) error
	GetMagicLinkTokenByHash(ctx context.Context, tokenHash string) (*MagicLinkToken, error)
	MarkMagicLinkTokenUsed(ctx context.Context, id string) error
	DeleteExpiredMagicLinkTokens(ctx context.Context) (int64, error)

	// MFARecoveryCodes
	CreateMFARecoveryCodes(ctx context.Context, codes []*MFARecoveryCode) error
	GetMFARecoveryCodesByUserID(ctx context.Context, userID string) ([]*MFARecoveryCode, error)
	MarkMFARecoveryCodeUsed(ctx context.Context, id string) error
	DeleteAllMFARecoveryCodesByUserID(ctx context.Context, userID string) error

	// Roles
	CreateRole(ctx context.Context, role *Role) error
	GetRoleByID(ctx context.Context, id string) (*Role, error)
	GetRoleByName(ctx context.Context, name string) (*Role, error)
	ListRoles(ctx context.Context) ([]*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id string) error

	// Permissions
	CreatePermission(ctx context.Context, perm *Permission) error
	GetPermissionByID(ctx context.Context, id string) (*Permission, error)
	ListPermissions(ctx context.Context) ([]*Permission, error)
	GetPermissionByActionResource(ctx context.Context, action, resource string) (*Permission, error)
	DeletePermission(ctx context.Context, id string) error

	// RolePermissions
	AttachPermissionToRole(ctx context.Context, roleID, permissionID string) error
	DetachPermissionFromRole(ctx context.Context, roleID, permissionID string) error
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*Permission, error)
	GetPermissionsByUserID(ctx context.Context, userID string) ([]*Permission, error)
	GetRolesByPermissionID(ctx context.Context, permissionID string) ([]*Role, error)
	GetUsersByPermissionID(ctx context.Context, permissionID string) ([]*User, error)
	// Batch variants return permission_id → count maps so the dashboard
	// can request all counts in 2 SQL round-trips instead of 2N.
	BatchCountRolesByPermissionIDs(ctx context.Context, permissionIDs []string) (map[string]int, error)
	BatchCountUsersByPermissionIDs(ctx context.Context, permissionIDs []string) (map[string]int, error)

	// UserRoles
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID string) error
	GetRolesByUserID(ctx context.Context, userID string) ([]*Role, error)
	GetUsersByRoleID(ctx context.Context, roleID string) ([]*User, error)

	// SSOConnections
	CreateSSOConnection(ctx context.Context, conn *SSOConnection) error
	GetSSOConnectionByID(ctx context.Context, id string) (*SSOConnection, error)
	GetSSOConnectionByDomain(ctx context.Context, domain string) (*SSOConnection, error)
	ListSSOConnections(ctx context.Context) ([]*SSOConnection, error)
	UpdateSSOConnection(ctx context.Context, conn *SSOConnection) error
	DeleteSSOConnection(ctx context.Context, id string) error

	// SSOIdentities
	CreateSSOIdentity(ctx context.Context, ident *SSOIdentity) error
	GetSSOIdentityByConnectionAndSub(ctx context.Context, connectionID, providerSub string) (*SSOIdentity, error)
	GetSSOIdentitiesByUserID(ctx context.Context, userID string) ([]*SSOIdentity, error)

	// APIKeys
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKeyByKeyHash(ctx context.Context, keyHash string) (*APIKey, error)
	GetAPIKeyByID(ctx context.Context, id string) (*APIKey, error)
	ListAPIKeys(ctx context.Context) ([]*APIKey, error)
	UpdateAPIKey(ctx context.Context, key *APIKey) error
	RevokeAPIKey(ctx context.Context, id string, revokedAt time.Time) error
	DeleteAPIKey(ctx context.Context, id string) error
	CountActiveAPIKeysByScope(ctx context.Context, scope string) (int, error)

	// AuditLogs
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	CreateAuditLogsBatch(ctx context.Context, logs []*AuditLog) error
	GetAuditLogByID(ctx context.Context, id string) (*AuditLog, error)
	QueryAuditLogs(ctx context.Context, opts AuditLogQuery) ([]*AuditLog, error)
	DeleteAuditLogsBefore(ctx context.Context, before time.Time) (int64, error)

	// MayActGrants — operator-issued delegation grants. Verified during
	// token-exchange and surfaced via the dashboard delegation graph.
	CreateMayActGrant(ctx context.Context, g *MayActGrant) error
	GetMayActGrantByID(ctx context.Context, id string) (*MayActGrant, error)
	ListMayActGrants(ctx context.Context, opts ListMayActGrantsQuery) ([]*MayActGrant, error)
	RevokeMayActGrant(ctx context.Context, id string, revokedAt time.Time) error
	FindLiveMayActGrant(ctx context.Context, fromID, toID string, at time.Time) (*MayActGrant, error)

	// Migrations (Auth0 import tracking)
	CreateMigration(ctx context.Context, m *Migration) error
	GetMigrationByID(ctx context.Context, id string) (*Migration, error)
	ListMigrations(ctx context.Context) ([]*Migration, error)
	UpdateMigration(ctx context.Context, m *Migration) error

	// Org RBAC — parallel tables, global RBAC untouched.
	CreateOrgRole(ctx context.Context, orgID, id, name, description string, isBuiltin bool) error
	GetOrgRoleByID(ctx context.Context, roleID string) (*OrgRole, error)
	GetOrgRolesByOrgID(ctx context.Context, orgID string) ([]*OrgRole, error)
	GetOrgRolesByUserID(ctx context.Context, userID, orgID string) ([]*OrgRole, error)
	GetOrgRoleByName(ctx context.Context, orgID, name string) (*OrgRole, error)
	UpdateOrgRole(ctx context.Context, roleID, name, description string) error
	DeleteOrgRole(ctx context.Context, roleID string) error
	AttachOrgPermission(ctx context.Context, orgRoleID, action, resource string) error
	DetachOrgPermission(ctx context.Context, orgRoleID, action, resource string) error
	GetOrgRolePermissions(ctx context.Context, orgRoleID string) ([]Permission, error)
	GrantOrgRole(ctx context.Context, orgID, userID, orgRoleID, grantedBy string) error
	RevokeOrgRole(ctx context.Context, orgID, userID, orgRoleID string) error
	GetOrgUserRoles(ctx context.Context, userID, orgID string) ([]*OrgRole, error)

	// JWT signing keys
	InsertSigningKey(ctx context.Context, key *SigningKey) error
	GetActiveSigningKey(ctx context.Context) (*SigningKey, error)
	GetActiveSigningKeyByAlgorithm(ctx context.Context, algorithm string) (*SigningKey, error)
	GetSigningKeyByKID(ctx context.Context, kid string) (*SigningKey, error)
	RotateSigningKeys(ctx context.Context, newKey *SigningKey) error
	ListJWKSCandidates(ctx context.Context, activeOnly bool, retiredCutoff time.Time) ([]*SigningKey, error)

	// Revoked JTIs
	InsertRevokedJTI(ctx context.Context, jti string, expiresAt time.Time) error
	IsRevokedJTI(ctx context.Context, jti string) (bool, error)
	PruneExpiredRevokedJTI(ctx context.Context) error

	// DPoP JTIs
	InsertDPoPJTI(ctx context.Context, jti string, expiresAt time.Time) error
	IsDPoPJTISeen(ctx context.Context, jti string) (bool, error)
	PruneExpiredDPoPJTIs(ctx context.Context) error

	// Applications
	CreateApplication(ctx context.Context, app *Application) error
	GetApplicationByID(ctx context.Context, id string) (*Application, error)
	GetApplicationByClientID(ctx context.Context, clientID string) (*Application, error)
	GetApplicationBySlug(ctx context.Context, slug string) (*Application, error)
	GetApplicationByProxyDomain(ctx context.Context, domain string) (*Application, error)
	GetDefaultApplication(ctx context.Context) (*Application, error)
	ListApplications(ctx context.Context, limit, offset int) ([]*Application, error)
	UpdateApplication(ctx context.Context, app *Application) error
	RotateApplicationSecret(ctx context.Context, id, newHash, newPrefix string) error
	DeleteApplication(ctx context.Context, id string) error

	// Agents (OAuth 2.1 clients)
	CreateAgent(ctx context.Context, agent *Agent) error
	GetAgentByID(ctx context.Context, id string) (*Agent, error)
	GetAgentByClientID(ctx context.Context, clientID string) (*Agent, error)
	ListAgents(ctx context.Context, opts ListAgentsOpts) ([]*Agent, int, error)
	ListAgentsByUserID(ctx context.Context, userID string) ([]*Agent, error)
	UpdateAgent(ctx context.Context, agent *Agent) error
	UpdateAgentSecret(ctx context.Context, id, secretHash string) error
	DeactivateAgent(ctx context.Context, id string) error
	// RotateDCRClientSecret rotates the client secret for a DCR-registered agent,
	// preserving the previous hash as old_secret_hash valid until oldSecretExpiresAt.
	RotateDCRClientSecret(ctx context.Context, agentID, newSecretHash, oldSecretHash string, oldSecretExpiresAt time.Time) error

	// OAuth Authorization Codes
	CreateAuthorizationCode(ctx context.Context, code *OAuthAuthorizationCode) error
	GetAuthorizationCode(ctx context.Context, codeHash string) (*OAuthAuthorizationCode, error)
	DeleteAuthorizationCode(ctx context.Context, codeHash string) error
	DeleteExpiredAuthorizationCodes(ctx context.Context) (int64, error)

	// OAuth PKCE Sessions
	CreatePKCESession(ctx context.Context, sess *OAuthPKCESession) error
	GetPKCESession(ctx context.Context, signatureHash string) (*OAuthPKCESession, error)
	DeletePKCESession(ctx context.Context, signatureHash string) error
	DeleteExpiredPKCESessions(ctx context.Context) (int64, error)

	// OAuth Tokens
	CreateOAuthToken(ctx context.Context, token *OAuthToken) error
	GetOAuthTokenByJTI(ctx context.Context, jti string) (*OAuthToken, error)
	GetOAuthTokenByHash(ctx context.Context, tokenHash string) (*OAuthToken, error)
	GetActiveOAuthTokenByRequestIDAndType(ctx context.Context, requestID, tokenType string) (*OAuthToken, error)
	RevokeOAuthToken(ctx context.Context, id string) error
	RevokeActiveOAuthTokenByRequestID(ctx context.Context, requestID, tokenType string) (bool, error)
	RevokeOAuthTokensByClientID(ctx context.Context, clientID string) (int64, error)
	RevokeOAuthTokensByClientIDPattern(ctx context.Context, pattern string) (int64, error)
	RevokeOAuthTokenFamily(ctx context.Context, familyID string) (int64, error)
	ListOAuthTokensByAgentID(ctx context.Context, agentID string, limit int) ([]*OAuthToken, error)
	DeleteExpiredOAuthTokens(ctx context.Context) (int64, error)
	UpdateOAuthTokenDPoPJKT(ctx context.Context, id string, jkt string) error

	// OAuth Consents
	CreateOAuthConsent(ctx context.Context, consent *OAuthConsent) error
	GetActiveConsent(ctx context.Context, userID, clientID string) (*OAuthConsent, error)
	ListConsentsByUserID(ctx context.Context, userID string) ([]*OAuthConsent, error)
	ListAllConsents(ctx context.Context) ([]*OAuthConsent, error)
	RevokeOAuthConsent(ctx context.Context, id string) error
	// RevokeConsentsByUserID bulk-revokes all active consents for a user.
	// Returns the number of rows updated.
	RevokeConsentsByUserID(ctx context.Context, userID string) (int64, error)

	// Device Codes (RFC 8628)
	CreateDeviceCode(ctx context.Context, dc *OAuthDeviceCode) error
	GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*OAuthDeviceCode, error)
	GetDeviceCodeByHash(ctx context.Context, hash string) (*OAuthDeviceCode, error)
	ListPendingDeviceCodes(ctx context.Context) ([]*OAuthDeviceCode, error)
	UpdateDeviceCodeStatus(ctx context.Context, hash string, status string, userID string) error
	UpdateDeviceCodePolledAt(ctx context.Context, hash string) error
	DeleteExpiredDeviceCodes(ctx context.Context) (int64, error)

	// Dynamic Client Registration (RFC 7591)
	CreateDCRClient(ctx context.Context, client *OAuthDCRClient) error
	GetDCRClient(ctx context.Context, clientID string) (*OAuthDCRClient, error)
	UpdateDCRClient(ctx context.Context, client *OAuthDCRClient) error
	DeleteDCRClient(ctx context.Context, clientID string) error
	// RotateDCRRegistrationToken replaces the registration_token_hash for a DCR client.
	RotateDCRRegistrationToken(ctx context.Context, clientID, newTokenHash string) error

	// Vault Providers (Token Vault — third-party OAuth providers)
	CreateVaultProvider(ctx context.Context, p *VaultProvider) error
	GetVaultProviderByID(ctx context.Context, id string) (*VaultProvider, error)
	GetVaultProviderByName(ctx context.Context, name string) (*VaultProvider, error)
	ListVaultProviders(ctx context.Context, activeOnly bool) ([]*VaultProvider, error)
	UpdateVaultProvider(ctx context.Context, p *VaultProvider) error
	DeleteVaultProvider(ctx context.Context, id string) error

	// Vault Connections (per-user links to a provider, with encrypted tokens)
	CreateVaultConnection(ctx context.Context, c *VaultConnection) error
	GetVaultConnectionByID(ctx context.Context, id string) (*VaultConnection, error)
	GetVaultConnection(ctx context.Context, providerID, userID string) (*VaultConnection, error)
	ListVaultConnectionsByUserID(ctx context.Context, userID string) ([]*VaultConnection, error)
	ListVaultConnectionsByProviderID(ctx context.Context, providerID string) ([]*VaultConnection, error)
	ListAllVaultConnections(ctx context.Context) ([]*VaultConnection, error)
	UpdateVaultConnection(ctx context.Context, c *VaultConnection) error
	UpdateVaultConnectionTokens(ctx context.Context, id, accessEnc, refreshEnc string, expiresAt *time.Time) error
	MarkVaultConnectionNeedsReauth(ctx context.Context, id string, needs bool) error
	DeleteVaultConnection(ctx context.Context, id string) error
	// ListAgentsByVaultRetrieval returns agents that have ever fetched a token
	// from the given vault connection (via audit_logs vault.token.retrieved events).
	ListAgentsByVaultRetrieval(ctx context.Context, connectionID string) ([]*Agent, error)

	// Auth Flows (Phase 6 Visual Flow Builder)
	CreateAuthFlow(ctx context.Context, flow *AuthFlow) error
	GetAuthFlowByID(ctx context.Context, id string) (*AuthFlow, error)
	ListAuthFlows(ctx context.Context) ([]*AuthFlow, error)
	ListAuthFlowsByTrigger(ctx context.Context, trigger string) ([]*AuthFlow, error)
	UpdateAuthFlow(ctx context.Context, flow *AuthFlow) error
	DeleteAuthFlow(ctx context.Context, id string) error

	// Auth Flow Runs (history for the Flow Builder dashboard)
	CreateAuthFlowRun(ctx context.Context, run *AuthFlowRun) error
	ListAuthFlowRunsByFlowID(ctx context.Context, flowID string, limit int) ([]*AuthFlowRun, error)

	// Branding (global config + per-app overrides via applications.branding_override JSON)
	GetBranding(ctx context.Context, id string) (*BrandingConfig, error)
	UpdateBranding(ctx context.Context, id string, fields map[string]any) error
	SetBrandingLogo(ctx context.Context, id, url, sha string) error
	ClearBrandingLogo(ctx context.Context, id string) error
	ResolveBranding(ctx context.Context, appID string) (*BrandingConfig, error)

	// Email templates (editable copy for hosted emails)
	ListEmailTemplates(ctx context.Context) ([]*EmailTemplate, error)
	GetEmailTemplate(ctx context.Context, id string) (*EmailTemplate, error)
	UpdateEmailTemplate(ctx context.Context, id string, fields map[string]any) error
	SeedEmailTemplates(ctx context.Context) error

	// Proxy Rules (Phase 6.6 / Wave D — runtime override layer for the
	// reverse proxy rule engine; YAML stays the bootstrap source).
	CreateProxyRule(ctx context.Context, rule *ProxyRule) error
	GetProxyRuleByID(ctx context.Context, id string) (*ProxyRule, error)
	ListProxyRules(ctx context.Context) ([]*ProxyRule, error)
	ListProxyRulesByAppID(ctx context.Context, appID string) ([]*ProxyRule, error)
	UpdateProxyRule(ctx context.Context, rule *ProxyRule) error
	DeleteProxyRule(ctx context.Context, id string) error

	// User tier (PROXYV1_5 §4.10) — tier lives inside users.metadata JSON
	// rather than a dedicated column so the schema doesn't fork for a
	// free/pro split. Helpers round-trip through metadata so existing
	// fields (e.g. custom app data) are preserved.
	SetUserTier(ctx context.Context, userID, tier string) error
	GetUserTier(ctx context.Context, userID string) (string, error)

	// System config (W17 yaml-deprecation Phase A) — single-row JSON blob
	// that holds all runtime configuration. DB takes precedence over YAML.
	GetSystemConfig(ctx context.Context) (string, error)
	SetSystemConfig(ctx context.Context, v any) error

	// Secrets (W17 yaml-deprecation Phase A) — named key-value store for
	// session secret, JWT signing keys, admin API key, etc.
	GetSecret(ctx context.Context, name string) (string, error)
	SetSecret(ctx context.Context, name, value string) error
	DeleteSecret(ctx context.Context, name string) error

	// Atomic Operations (Consolidated Transactions for high performance)
	SignupAtomic(ctx context.Context, user *User, sess *Session, log *AuditLog) error
	LoginAtomic(ctx context.Context, user *User, sess *Session, log *AuditLog) error

	// W17 Phase C system operations.
	// DBPath returns the filesystem path of the open SQLite database file.
	// Returns ":memory:" for in-memory stores used in tests.
	DBPath() string
	// WipeAllData truncates all user-data tables (users, sessions, api_keys,
	// OAuth accounts, passkeys, etc.) while preserving goose migration metadata.
	// Used by the admin reset endpoints.
	WipeAllData(ctx context.Context) error
	// RevokeAllAdminAPIKeys soft-deletes every API key that has the "*" (admin)
	// wildcard scope, so existing admin sessions are invalidated after a key rotation.
	RevokeAllAdminAPIKeys(ctx context.Context) error
}

// --- Entity types ---

// User represents a user account.
type User struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"email_verified"`
	PasswordHash  *string `json:"-"`
	HashType      string  `json:"-"`
	Name          *string `json:"name,omitempty"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	MFAEnabled    bool    `json:"mfa_enabled"`
	MFASecret     *string `json:"-"`
	MFAVerified   bool    `json:"mfa_verified"`
	// MFAVerifiedAt is set when the user successfully completes their first TOTP
	// verification after enrollment (F3.2). NULL means enrolled but pending.
	MFAVerifiedAt *string `json:"-"`
	Metadata      string  `json:"metadata"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	LastLoginAt   *string `json:"last_login_at,omitempty"`
}

// Session represents a user session.
type Session struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	IP         string `json:"ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	MFAPassed  bool   `json:"mfa_passed"`
	AuthMethod string `json:"auth_method"`
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at"`
}

// OAuthAccount represents a linked OAuth provider account.
type OAuthAccount struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	Provider     string  `json:"provider"`
	ProviderID   string  `json:"provider_id"`
	Email        *string `json:"email,omitempty"`
	AccessToken  *string `json:"-"`
	RefreshToken *string `json:"-"`
	CreatedAt    string  `json:"created_at"`
}

// PasskeyCredential represents a WebAuthn credential.
type PasskeyCredential struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	CredentialID []byte  `json:"credential_id"`
	PublicKey    []byte  `json:"-"`
	AAGUID       *string `json:"aaguid,omitempty"`
	SignCount    int     `json:"sign_count"`
	Name         *string `json:"name,omitempty"`
	Transports   string  `json:"transports"`
	BackedUp     bool    `json:"backed_up"`
	CreatedAt    string  `json:"created_at"`
	LastUsedAt   *string `json:"last_used_at,omitempty"`
}

// MagicLinkToken represents a magic link authentication token.
type MagicLinkToken struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	TokenHash string `json:"-"`
	Used      bool   `json:"used"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// MFARecoveryCode represents a one-time MFA recovery code.
type MFARecoveryCode struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Code      string `json:"-"`
	Used      bool   `json:"used"`
	CreatedAt string `json:"created_at"`
}

// Role represents an RBAC role.
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Permission represents an RBAC permission.
type Permission struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	CreatedAt string `json:"created_at"`
}

// SSOConnection represents a SAML or OIDC SSO connection.
type SSOConnection struct {
	ID               string  `json:"id"`
	Type             string  `json:"type"`
	Name             string  `json:"name"`
	Domain           *string `json:"domain,omitempty"`
	SAMLIdPURL       *string `json:"saml_idp_url,omitempty"`
	SAMLIdPCert      *string `json:"saml_idp_cert,omitempty"`
	SAMLSPEntityID   *string `json:"saml_sp_entity_id,omitempty"`
	SAMLSPAcsURL     *string `json:"saml_sp_acs_url,omitempty"`
	OIDCIssuer       *string `json:"oidc_issuer,omitempty"`
	OIDCClientID     *string `json:"oidc_client_id,omitempty"`
	OIDCClientSecret *string `json:"-"`
	Enabled          bool    `json:"enabled"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// SSOIdentity represents a user's identity from an SSO provider.
type SSOIdentity struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	ConnectionID string `json:"connection_id"`
	ProviderSub  string `json:"provider_sub"`
	CreatedAt    string `json:"created_at"`
}

// APIKey represents an M2M API key.
type APIKey struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	KeyHash    string  `json:"-"`
	KeyPrefix  string  `json:"key_prefix"`
	KeySuffix  string  `json:"key_suffix"`
	Scopes     string  `json:"scopes"`
	RateLimit  int     `json:"rate_limit"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
	RevokedAt  *string `json:"revoked_at,omitempty"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID           string  `json:"id"`
	ActorID      string  `json:"actor_id,omitempty"`
	ActorType    string  `json:"actor_type"`
	Action       string  `json:"action"`
	TargetType   string  `json:"target_type,omitempty"`
	TargetID     string  `json:"target_id,omitempty"`
	OrgID        *string `json:"org_id,omitempty"`
	SessionID    *string `json:"session_id,omitempty"`
	ResourceType *string `json:"resource_type,omitempty"`
	ResourceID   *string `json:"resource_id,omitempty"`
	IP           string  `json:"ip,omitempty"`
	UserAgent    string  `json:"user_agent,omitempty"`
	Metadata     string  `json:"metadata"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
}

// Migration represents an Auth0 import migration record.
type Migration struct {
	ID            string  `json:"id"`
	Source        string  `json:"source"`
	Status        string  `json:"status"`
	UsersTotal    int     `json:"users_total"`
	UsersImported int     `json:"users_imported"`
	Errors        string  `json:"errors"`
	CreatedAt     string  `json:"created_at"`
	CompletedAt   *string `json:"completed_at,omitempty"`
}

// Organization roles. Canonical list lives here so both storage and API code
// reference the same constants; CHECK constraint in SQL enforces the same set.
const (
	OrgRoleOwner  = "owner"
	OrgRoleAdmin  = "admin"
	OrgRoleMember = "member"
)

// Organization represents a tenant/team. Multi-tenancy unit for B2B use.
type Organization struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Metadata  string `json:"metadata"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// OrganizationMember is the (org, user) membership row with a role.
type OrganizationMember struct {
	OrganizationID string `json:"organization_id"`
	UserID         string `json:"user_id"`
	Role           string `json:"role"`
	JoinedAt       string `json:"joined_at"`
}

// OrganizationMemberWithUser is the member row plus joined user email/name
// so handlers can avoid N+1 lookups when rendering the member list.
type OrganizationMemberWithUser struct {
	OrganizationMember
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name,omitempty"`
}

// OrganizationInvitation is a pending invite. The token is never stored in
// plaintext — only its SHA-256 hash — so a DB leak can't hand attackers
// valid invitation links.
type OrganizationInvitation struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organization_id"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	TokenHash      string  `json:"-"`
	InvitedBy      *string `json:"invited_by,omitempty"`
	AcceptedAt     *string `json:"accepted_at,omitempty"`
	ExpiresAt      string  `json:"expires_at"`
	CreatedAt      string  `json:"created_at"`
}

// Webhook delivery status constants.
const (
	WebhookStatusPending   = "pending"
	WebhookStatusDelivered = "delivered"
	WebhookStatusRetrying  = "retrying"
	WebhookStatusFailed    = "failed"
)

// Webhook event names. Ship minimal set for real dev integration —
// expand alongside SDK in Phase 5.
const (
	WebhookEventUserCreated    = "user.created"
	WebhookEventUserUpdated    = "user.updated"
	WebhookEventUserDeleted    = "user.deleted"
	WebhookEventSessionCreated = "session.created"
	WebhookEventSessionRevoked = "session.revoked"
	WebhookEventMFAEnabled     = "mfa.enabled"
	WebhookEventOrgCreated     = "organization.created"
	WebhookEventOrgDeleted     = "organization.deleted"
	WebhookEventOrgMemberAdded = "organization.member_added"
	WebhookEventSystemAuditLog = "system.audit_log"
	WebhookEventTest           = "webhook.test"
)

// Webhook is an admin-registered outbound event endpoint.
type Webhook struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Secret      string `json:"-"`       // HMAC signing secret, never returned on the wire
	Events      string `json:"events"`  // JSON array of event names
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// WebhookDelivery is one attempt at delivering an event to a webhook URL.
type WebhookDelivery struct {
	ID              string  `json:"id"`
	WebhookID       string  `json:"webhook_id"`
	Event           string  `json:"event"`
	Payload         string  `json:"payload"`
	SignatureHeader string  `json:"signature_header,omitempty"`
	Status          string  `json:"status"`
	StatusCode      *int    `json:"status_code,omitempty"`
	ResponseBody    string  `json:"response_body,omitempty"`
	Error           string  `json:"error,omitempty"`
	Attempt         int     `json:"attempt"`
	NextRetryAt     *string `json:"next_retry_at,omitempty"`
	DeliveredAt     *string `json:"delivered_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// DevEmail represents a single captured email in dev mode.
type DevEmail struct {
	ID        string `json:"id"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	HTML      string `json:"html"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// SessionWithUser joins a session row with the minimal user columns needed
// by the admin sessions list (avoids N+1 lookups in the handler).
type SessionWithUser struct {
	Session
	UserEmail string `json:"user_email"`
}

// MethodCount is a row of GROUP BY auth_method, COUNT(*).
type MethodCount struct {
	AuthMethod string `json:"auth_method"`
	Count      int    `json:"count"`
}

// DayCount is a row of GROUP BY DATE(created_at), COUNT(*).
type DayCount struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int    `json:"count"`
}

// --- Query types ---

// ListUsersOpts configures user list queries.
type ListUsersOpts struct {
	Limit         int
	Offset        int
	Search        string // optional email/name search
	MFAEnabled    *bool  // filter by mfa_enabled
	EmailVerified *bool  // filter by email_verified
	RoleID        string // filter by role assignment
	AuthMethod    string // filter by auth_method (password|oauth|passkey|magic_link|sso)
	OrgID         string // filter by organization membership
}

// ListSessionsOpts configures admin session list queries with cursor pagination.
//
// Pagination contract:
//   - Results are ordered by (created_at DESC, id DESC) for stable iteration.
//   - Cursor is the last item's created_at + "|" + id ("keyset" pagination).
//   - An empty cursor means "first page".
//   - NextCursor is returned alongside results by the handler.
//
// Filter semantics: all filters AND together; empty string = no filter.
type ListSessionsOpts struct {
	UserID     string // exact match
	AuthMethod string // exact match: password | google | github | passkey | magic_link | sso
	MFAPassed  *bool  // nil = no filter
	Limit      int    // default 50, max 200
	Cursor     string // "created_at|id" keyset cursor
}

// AuditLogQuery configures audit log queries with cursor-based pagination.
type AuditLogQuery struct {
	Action       string // filter by event type
	ActorID      string // filter by actor
	ActorType    string // filter by actor type (user|agent|system|admin)
	TargetID     string // filter by target
	OrgID        string // filter by organization
	SessionID    string // filter by session
	ResourceType string // filter by resource type
	ResourceID   string // filter by resource id
	Status       string // "success" or "failure"
	IP           string // filter by IP
	From         string // start of date range (RFC3339)
	To           string // end of date range (RFC3339)
	Limit        int    // page size (default 50, max 200)
	Cursor       string // cursor-based pagination (ID of last item)
	GrantID      string // metadata.grant_id (json_extract) — correlates rows to may_act_grants.id
}

// MayActGrant is an operator-issued delegation grant. Verified at token-exchange
// time and recorded by id into audit metadata so the dashboard can correlate
// hops to their backing grant.
type MayActGrant struct {
	ID         string   `json:"id"`
	FromID     string   `json:"from_id"`
	ToID       string   `json:"to_id"`
	MaxHops    int      `json:"max_hops"`
	Scopes     []string `json:"scopes"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
	RevokedAt  *string  `json:"revoked_at,omitempty"`
	CreatedBy  string   `json:"created_by,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

// ListMayActGrantsQuery filters the grant list. Empty fields = no filter.
type ListMayActGrantsQuery struct {
	FromID         string
	ToID           string
	IncludeRevoked bool
}

// SigningKey represents a JWT signing keypair stored in the database.
// PrivateKeyPEM is AES-GCM encrypted at rest; PublicKeyPEM is plaintext.
type SigningKey struct {
	ID            int64   `json:"id"`
	KID           string  `json:"kid"`
	Algorithm     string  `json:"algorithm"`
	PublicKeyPEM  string  `json:"-"`
	PrivateKeyPEM string  `json:"-"` // encrypted
	CreatedAt     string  `json:"created_at"`
	RotatedAt     *string `json:"rotated_at,omitempty"`
	Status        string  `json:"status"` // "active" | "retired"
}

// EmailTemplate is the editable copy for one hosted email (magic_link, etc).
// BodyParagraphs is stored as a JSON string in SQLite but surfaced as []string.
type EmailTemplate struct {
	ID             string   `json:"id"`
	Subject        string   `json:"subject"`
	Preheader      string   `json:"preheader"`
	HeaderText     string   `json:"header_text"`
	BodyParagraphs []string `json:"body_paragraphs"`
	CTAText        string   `json:"cta_text"`
	CTAURLTemplate string   `json:"cta_url_template"`
	FooterText     string   `json:"footer_text"`
	UpdatedAt      string   `json:"updated_at"`
}

// BrandingConfig is the resolved branding (global + app override merged).
type BrandingConfig struct {
	LogoURL          string `json:"logo_url,omitempty"`
	LogoSHA          string `json:"logo_sha,omitempty"`
	PrimaryColor     string `json:"primary_color"`
	SecondaryColor   string `json:"secondary_color"`
	FontFamily       string `json:"font_family"`
	FooterText       string `json:"footer_text"`
	EmailFromName    string `json:"email_from_name"`
	EmailFromAddress string `json:"email_from_address"`
}

// Application represents a registered OAuth 2.x / OIDC client application.
// JSON columns (AllowedCallbackURLs, AllowedLogoutURLs, AllowedOrigins, Metadata)
// are serialized/deserialized in the storage layer; callers use native Go types.
type Application struct {
	ID                  string         `json:"id"`           // app_<nanoid>
	Name                string         `json:"name"`
	// Slug is the URL-safe identifier used in hosted login page routing.
	// Auto-generated from Name on create if not supplied; must match
	// ^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$ and be unique across all applications.
	Slug                string         `json:"slug"`
	ClientID            string         `json:"client_id"`    // shark_app_<nanoid>
	ClientSecretHash    string         `json:"-"`            // SHA-256 hex, never exposed
	ClientSecretPrefix  string         `json:"client_secret_prefix"` // first 8 chars (UX only)
	AllowedCallbackURLs []string       `json:"allowed_callback_urls"`
	AllowedLogoutURLs   []string       `json:"allowed_logout_urls"`
	AllowedOrigins      []string       `json:"allowed_origins"`
	IsDefault           bool           `json:"is_default"`
	Metadata            map[string]any `json:"metadata"`
	// IntegrationMode is the per-app login-surface picker. One of
	// "hosted", "components", "proxy", "custom". Defaults to "custom".
	IntegrationMode string `json:"integration_mode"`
	// BrandingOverride is a JSON object of per-app branding overrides
	// that merge over the global branding row in ResolveBranding.
	// Empty string means "no override; use global".
	BrandingOverride string `json:"branding_override,omitempty"`
	// ProxyLoginFallback controls where proxy-mode apps send unauthed
	// requests: "hosted" or "custom_url". Defaults to "hosted".
	ProxyLoginFallback    string    `json:"proxy_login_fallback,omitempty"`
	ProxyLoginFallbackURL string    `json:"proxy_login_fallback_url,omitempty"`

	// ProxyPublicDomain is the external domain for transparent proxying (e.g. "api.myapp.com").
	ProxyPublicDomain string `json:"proxy_public_domain,omitempty"`
	// ProxyProtectedURL is the internal destination for transparent proxying (e.g. "http://localhost:3001").
	ProxyProtectedURL string `json:"proxy_protected_url,omitempty"`

	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
