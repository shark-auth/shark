package storage

import (
	"context"
	"database/sql"
	"time"
)

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
	GroupSessionsByAuthMethodSince(ctx context.Context, since time.Time) ([]MethodCount, error)
	GroupUsersCreatedByDay(ctx context.Context, days int) ([]DayCount, error)

	// DevEmails (dev-mode inbox)
	CreateDevEmail(ctx context.Context, e *DevEmail) error
	ListDevEmails(ctx context.Context, limit int) ([]*DevEmail, error)
	GetDevEmail(ctx context.Context, id string) (*DevEmail, error)
	DeleteAllDevEmails(ctx context.Context) error

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
	Limit  int
	Offset int
	Search string // optional email/name search
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
