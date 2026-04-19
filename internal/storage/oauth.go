package storage

import "time"

// OAuthAuthorizationCode represents a short-lived authorization code.
type OAuthAuthorizationCode struct {
	CodeHash             string    `json:"-"`
	ClientID             string    `json:"client_id"`
	UserID               string    `json:"user_id"`
	RedirectURI          string    `json:"redirect_uri"`
	Scope                string    `json:"scope"`
	CodeChallenge        string    `json:"-"`
	CodeChallengeMethod  string    `json:"-"`
	Resource             string    `json:"resource,omitempty"`
	AuthorizationDetails string    `json:"authorization_details,omitempty"`
	Nonce                string    `json:"nonce,omitempty"`
	ExpiresAt            time.Time `json:"expires_at"`
	CreatedAt            time.Time `json:"created_at"`
}

// OAuthToken represents an access or refresh token record.
type OAuthToken struct {
	ID                   string     `json:"id"`
	JTI                  string     `json:"jti"`
	RequestID            string     `json:"-"` // fosite request ID (may repeat across rotations)
	ClientID             string     `json:"client_id"`
	AgentID              string     `json:"agent_id,omitempty"`
	UserID               string     `json:"user_id,omitempty"`
	TokenType            string     `json:"token_type"` // access | refresh
	TokenHash            string     `json:"-"`
	Scope                string     `json:"scope"`
	Audience             string     `json:"audience,omitempty"`
	AuthorizationDetails string     `json:"authorization_details,omitempty"`
	DPoPJKT              string     `json:"dpop_jkt,omitempty"`
	DelegationSubject    string     `json:"delegation_subject,omitempty"`
	DelegationActor      string     `json:"delegation_actor,omitempty"`
	FamilyID             string     `json:"family_id,omitempty"`
	ExpiresAt            time.Time  `json:"expires_at"`
	CreatedAt            time.Time  `json:"created_at"`
	RevokedAt            *time.Time `json:"revoked_at,omitempty"`
}

// OAuthConsent represents a user's consent grant for an agent.
type OAuthConsent struct {
	ID                   string     `json:"id"`
	UserID               string     `json:"user_id"`
	ClientID             string     `json:"client_id"`
	Scope                string     `json:"scope"`
	AuthorizationDetails string     `json:"authorization_details,omitempty"`
	GrantedAt            time.Time  `json:"granted_at"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	RevokedAt            *time.Time `json:"revoked_at,omitempty"`
}

// OAuthDeviceCode represents a pending device authorization (RFC 8628).
type OAuthDeviceCode struct {
	DeviceCodeHash string     `json:"-"`
	UserCode       string     `json:"user_code"`
	ClientID       string     `json:"client_id"`
	Scope          string     `json:"scope"`
	Resource       string     `json:"resource,omitempty"`
	UserID         string     `json:"user_id,omitempty"`
	Status         string     `json:"status"`          // pending | approved | denied | expired
	LastPolledAt   *time.Time `json:"last_polled_at,omitempty"`
	PollInterval   int        `json:"poll_interval"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// OAuthDCRClient represents a dynamically registered client (RFC 7591).
type OAuthDCRClient struct {
	ClientID              string     `json:"client_id"`
	RegistrationTokenHash string     `json:"-"`
	ClientMetadata        string     `json:"client_metadata"` // full JSON
	CreatedAt             time.Time  `json:"created_at"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
}

// OAuthPKCESession persists the PKCE challenge associated with an authorization
// code so the token endpoint can validate the code_verifier on exchange.
// Required because fosite calls CreateAuthorizeCodeSession with a sanitized
// Requester (form values stripped) but calls CreatePKCERequestSession with the
// unsanitized challenge separately.
type OAuthPKCESession struct {
	SignatureHash       string    `json:"-"`
	CodeChallenge       string    `json:"code_challenge"`
	CodeChallengeMethod string    `json:"code_challenge_method"`
	ClientID            string    `json:"client_id"`
	ExpiresAt           time.Time `json:"expires_at"`
	CreatedAt           time.Time `json:"created_at"`
}
