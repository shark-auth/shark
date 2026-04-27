package storage

import "time"

// VaultProvider represents a third-party OAuth provider that users can connect
// their account to (Google Calendar, Slack, GitHub, etc.). Admins configure
// these; users establish per-user connections against them.
//
// ClientSecretEnc is stored encrypted via FieldEncryptor (enc::<b64> prefix).
// Handlers decrypt it before using in OAuth flows; storage never touches crypto.
type VaultProvider struct {
	ID              string            `json:"id"`              // vp_<nanoid>
	Name            string            `json:"name"`            // unique short key, e.g. "google_calendar"
	DisplayName     string            `json:"display_name"`    // user-facing, e.g. "Google Calendar"
	AuthURL         string            `json:"auth_url"`
	TokenURL        string            `json:"token_url"`
	ClientID        string            `json:"client_id"`
	ClientSecretEnc string            `json:"-"`               // encrypted; never exposed on the wire
	Scopes          []string          `json:"scopes"`          // default scopes requested at auth time
	IconURL         string            `json:"icon_url,omitempty"`
	Active          bool              `json:"active"`
	// ExtraAuthParams holds provider-specific query parameters appended to the
	// authorize URL at BuildAuthURL time (e.g. prompt=consent, audience=...).
	// Persisted per-provider so manual providers get the same behaviour as
	// template-created ones. Empty map serialises as "{}".
	ExtraAuthParams map[string]string `json:"extra_auth_params,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// VaultConnection represents a single user's OAuth connection to a given
// provider. Holds encrypted access/refresh tokens, scopes granted, and a
// flag that goes hot when the refresh token is rejected (user must re-auth).
type VaultConnection struct {
	ID              string         `json:"id"`               // vc_<nanoid>
	ProviderID      string         `json:"provider_id"`
	UserID          string         `json:"user_id"`
	AccessTokenEnc  string         `json:"-"`                // encrypted
	RefreshTokenEnc string         `json:"-"`                // encrypted; empty when provider issued no refresh token
	TokenType       string         `json:"token_type"`       // typically "Bearer"
	Scopes          []string       `json:"scopes"`           // scopes the user actually granted
	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`
	Metadata        map[string]any `json:"metadata"`
	NeedsReauth     bool           `json:"needs_reauth"`
	LastRefreshedAt *time.Time     `json:"last_refreshed_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
