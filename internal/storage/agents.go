package storage

import "time"

// Agent represents an OAuth 2.1 client with agent identity.
type Agent struct {
	ID               string         `json:"id"`                // agent_<nanoid>
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	ClientID         string         `json:"client_id"`
	ClientSecretHash string         `json:"-"`                 // SHA-256, never exposed
	ClientType       string         `json:"client_type"`       // confidential | public
	AuthMethod       string         `json:"auth_method"`       // client_secret_basic | client_secret_post | private_key_jwt | none
	JWKS             string         `json:"jwks,omitempty"`    // JSON string
	JWKSURI          string         `json:"jwks_uri,omitempty"`
	RedirectURIs     []string       `json:"redirect_uris"`
	GrantTypes       []string       `json:"grant_types"`
	ResponseTypes    []string       `json:"response_types"`
	Scopes           []string       `json:"scopes"`
	TokenLifetime    int            `json:"token_lifetime"`    // seconds
	Metadata         map[string]any `json:"metadata"`
	LogoURI          string         `json:"logo_uri,omitempty"`
	HomepageURI      string         `json:"homepage_uri,omitempty"`
	Active           bool           `json:"active"`
	CreatedBy        string         `json:"created_by,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	// F4.3: previous secret hash kept valid during the 1-hour grace window.
	OldSecretHash      string     `json:"-"`
	OldSecretExpiresAt *time.Time `json:"-"`
}

// ListAgentsOpts configures agent list queries.
type ListAgentsOpts struct {
	Limit  int
	Offset int
	Search string // search name/description
	Active *bool  // filter by active status
}
