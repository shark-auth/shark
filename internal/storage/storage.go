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

	// Sessions
	CreateSession(ctx context.Context, sess *Session) error
	GetSessionByID(ctx context.Context, id string) (*Session, error)
	GetSessionsByUserID(ctx context.Context, userID string) ([]*Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) (int64, error)
	UpdateSessionMFAPassed(ctx context.Context, id string, mfaPassed bool) error
	ListActiveSessions(ctx context.Context, opts ListSessionsOpts) ([]*SessionWithUser, error)
	DeleteSessionsByUserID(ctx context.Context, userID string) ([]string, error)

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

	// RolePermissions
	AttachPermissionToRole(ctx context.Context, roleID, permissionID string) error
	DetachPermissionFromRole(ctx context.Context, roleID, permissionID string) error
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*Permission, error)

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
	CountActiveAPIKeysByScope(ctx context.Context, scope string) (int, error)

	// AuditLogs
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	GetAuditLogByID(ctx context.Context, id string) (*AuditLog, error)
	QueryAuditLogs(ctx context.Context, opts AuditLogQuery) ([]*AuditLog, error)
	DeleteAuditLogsBefore(ctx context.Context, before time.Time) (int64, error)

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

	// Applications
	CreateApplication(ctx context.Context, app *Application) error
	GetApplicationByID(ctx context.Context, id string) (*Application, error)
	GetApplicationByClientID(ctx context.Context, clientID string) (*Application, error)
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
	UpdateAgent(ctx context.Context, agent *Agent) error
	DeactivateAgent(ctx context.Context, id string) error

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
	RevokeOAuthTokensByClientID(ctx context.Context, clientID string) (int64, error)
	RevokeOAuthTokenFamily(ctx context.Context, familyID string) (int64, error)
	ListOAuthTokensByAgentID(ctx context.Context, agentID string, limit int) ([]*OAuthToken, error)
	DeleteExpiredOAuthTokens(ctx context.Context) (int64, error)
	UpdateOAuthTokenDPoPJKT(ctx context.Context, id string, jkt string) error

	// OAuth Consents
	CreateOAuthConsent(ctx context.Context, consent *OAuthConsent) error
	GetActiveConsent(ctx context.Context, userID, clientID string) (*OAuthConsent, error)
	ListConsentsByUserID(ctx context.Context, userID string) ([]*OAuthConsent, error)
	RevokeOAuthConsent(ctx context.Context, id string) error

	// Device Codes (RFC 8628)
	CreateDeviceCode(ctx context.Context, dc *OAuthDeviceCode) error
	GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*OAuthDeviceCode, error)
	GetDeviceCodeByHash(ctx context.Context, hash string) (*OAuthDeviceCode, error)
	UpdateDeviceCodeStatus(ctx context.Context, hash string, status string, userID string) error
	UpdateDeviceCodePolledAt(ctx context.Context, hash string) error
	DeleteExpiredDeviceCodes(ctx context.Context) (int64, error)

	// Dynamic Client Registration (RFC 7591)
	CreateDCRClient(ctx context.Context, client *OAuthDCRClient) error
	GetDCRClient(ctx context.Context, clientID string) (*OAuthDCRClient, error)
	UpdateDCRClient(ctx context.Context, client *OAuthDCRClient) error
	DeleteDCRClient(ctx context.Context, clientID string) error

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
	UpdateVaultConnection(ctx context.Context, c *VaultConnection) error
	UpdateVaultConnectionTokens(ctx context.Context, id, accessEnc, refreshEnc string, expiresAt *time.Time) error
	MarkVaultConnectionNeedsReauth(ctx context.Context, id string, needs bool) error
	DeleteVaultConnection(ctx context.Context, id string) error

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
	ID         string `json:"id"`
	ActorID    string `json:"actor_id,omitempty"`
	ActorType  string `json:"actor_type"`
	Action     string `json:"action"`
	TargetType string `json:"target_type,omitempty"`
	TargetID   string `json:"target_id,omitempty"`
	IP         string `json:"ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	Metadata   string `json:"metadata"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
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
	Action   string // filter by event type
	ActorID  string // filter by actor
	TargetID string // filter by target
	Status   string // "success" or "failure"
	IP       string // filter by IP
	From     string // start of date range (RFC3339)
	To       string // end of date range (RFC3339)
	Limit    int    // page size (default 50, max 200)
	Cursor   string // cursor-based pagination (ID of last item)
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

// Application represents a registered OAuth 2.x / OIDC client application.
// JSON columns (AllowedCallbackURLs, AllowedLogoutURLs, AllowedOrigins, Metadata)
// are serialized/deserialized in the storage layer; callers use native Go types.
type Application struct {
	ID                  string         `json:"id"`           // app_<nanoid>
	Name                string         `json:"name"`
	ClientID            string         `json:"client_id"`    // shark_app_<nanoid>
	ClientSecretHash    string         `json:"-"`            // SHA-256 hex, never exposed
	ClientSecretPrefix  string         `json:"client_secret_prefix"` // first 8 chars (UX only)
	AllowedCallbackURLs []string       `json:"allowed_callback_urls"`
	AllowedLogoutURLs   []string       `json:"allowed_logout_urls"`
	AllowedOrigins      []string       `json:"allowed_origins"`
	IsDefault           bool           `json:"is_default"`
	Metadata            map[string]any `json:"metadata"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}
