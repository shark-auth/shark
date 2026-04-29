package vault

import (
	"sort"

	"github.com/shark-auth/shark/internal/storage"
)

// ProviderTemplate is a pre-configured OAuth 2.0 endpoint descriptor.
// Admins pick a template + supply their own (client_id, client_secret) â€”
// the template itself never carries credentials. Turning a template into a
// *storage.VaultProvider goes through ApplyTemplate; actual persistence
// (including secret encryption) still flows through Manager.CreateProvider.
type ProviderTemplate struct {
	// Name is the unique short key â€” e.g. "google_calendar", "slack",
	// "github". Used as the VaultProvider.Name when the template is
	// applied, and as the lookup key in the registry.
	Name string `json:"name"`
	// DisplayName is the user-visible label, e.g. "Google Calendar".
	DisplayName string `json:"display_name"`
	// AuthURL is the provider's OAuth 2.0 authorization endpoint.
	AuthURL string `json:"auth_url"`
	// TokenURL is the provider's OAuth 2.0 token endpoint.
	TokenURL string `json:"token_url"`
	// DefaultScopes is the recommended scope list for this template. May
	// be empty (e.g. Notion doesn't use scopes). Never nil.
	DefaultScopes []string `json:"default_scopes"`
	// IconURL is an optional public URL for a branded icon. Empty when
	// we don't want to commit to a specific CDN path.
	IconURL string `json:"icon_url,omitempty"`
	// ExtraAuthParams holds provider-specific query parameters appended to the
	// authorize URL at BuildAuthURL time (e.g. prompt=consent, audience=...).
	// These are baked into the template so the handler doesn't need provider-
	// specific branches; they are NOT stored on VaultProvider because they are
	// a function of the provider type, not a per-install admin choice.
	ExtraAuthParams map[string]string `json:"extra_auth_params,omitempty"`
	// TokenResponseShape names the token-response unmarshaler to use.
	// "" or "rfc6749" â†’ standard golang.org/x/oauth2 path.
	// "slack_v2"      â†’ Slack's ok-flag + nested authed_user token.
	TokenResponseShape string `json:"token_response_shape,omitempty"`
}

// builtinTemplates is the internal registry. Access via Templates() /
// Template() / ListTemplates() rather than touching the map directly so
// we can swap storage strategies later without breaking callers.
var builtinTemplates = map[string]*ProviderTemplate{
	"google_calendar": {
		Name:        "google_calendar",
		DisplayName: "Google Calendar",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		DefaultScopes: []string{
			"openid",
			"email",
			"https://www.googleapis.com/auth/calendar",
		},
		IconURL: "https://www.google.com/s2/favicons?domain=google.com&sz=64",
	},
	"google_drive": {
		Name:        "google_drive",
		DisplayName: "Google Drive",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		DefaultScopes: []string{
			"openid",
			"email",
			"https://www.googleapis.com/auth/drive",
		},
		IconURL: "https://www.google.com/s2/favicons?domain=google.com&sz=64",
	},
	"google_gmail": {
		Name:        "google_gmail",
		DisplayName: "Gmail",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		DefaultScopes: []string{
			"openid",
			"email",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.send",
		},
		IconURL: "https://www.google.com/s2/favicons?domain=google.com&sz=64",
	},
	"slack": {
		Name:               "slack",
		DisplayName:        "Slack",
		AuthURL:            "https://slack.com/oauth/v2/authorize",
		TokenURL:           "https://slack.com/api/oauth.v2.access",
		DefaultScopes:      []string{"chat:write", "channels:read", "users:read"},
		IconURL:            "https://a.slack-edge.com/80588/marketing/img/meta/favicon-32.png",
		TokenResponseShape: "slack_v2",
	},
	"github": {
		Name:          "github",
		DisplayName:   "GitHub",
		AuthURL:       "https://github.com/login/oauth/authorize",
		TokenURL:      "https://github.com/login/oauth/access_token",
		DefaultScopes: []string{"repo", "read:user", "user:email"},
		IconURL:       "https://github.githubassets.com/favicons/favicon.png",
	},
	"microsoft": {
		Name:          "microsoft",
		DisplayName:   "Microsoft Graph",
		AuthURL:       "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:      "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		DefaultScopes: []string{"offline_access", "User.Read", "Mail.Read"},
		IconURL:       "https://c.s-microsoft.com/favicon.ico?v2",
		ExtraAuthParams: map[string]string{
			"prompt": "select_account",
		},
	},
	"notion": {
		Name:        "notion",
		DisplayName: "Notion",
		AuthURL:     "https://api.notion.com/v1/oauth/authorize",
		TokenURL:    "https://api.notion.com/v1/oauth/token",
		// Notion's OAuth 2.0 flow does not use scopes â€” the integration's
		// permissions are configured in the Notion admin UI at install time.
		DefaultScopes: []string{},
		IconURL:       "https://www.notion.so/front-static/favicon.ico",
	},
	"linear": {
		Name:          "linear",
		DisplayName:   "Linear",
		AuthURL:       "https://linear.app/oauth/authorize",
		TokenURL:      "https://api.linear.app/oauth/token",
		DefaultScopes: []string{"read", "write"},
		IconURL:       "https://linear.app/favicon.ico",
		ExtraAuthParams: map[string]string{
			"prompt": "consent",
		},
	},
	"jira": {
		Name:          "jira",
		DisplayName:   "Jira Cloud",
		AuthURL:       "https://auth.atlassian.com/authorize",
		TokenURL:      "https://auth.atlassian.com/oauth/token",
		DefaultScopes: []string{"read:jira-work", "write:jira-work", "offline_access"},
		IconURL:       "https://www.atlassian.com/favicon.ico",
		ExtraAuthParams: map[string]string{
			"audience": "api.atlassian.com",
			"prompt":   "consent",
		},
	},
}

// Templates returns a copy of the built-in catalog keyed by Name. The map
// is a shallow copy so callers can iterate/mutate without disturbing the
// package-level registry, but the *ProviderTemplate pointers are shared â€”
// treat them as read-only.
func Templates() map[string]*ProviderTemplate {
	out := make(map[string]*ProviderTemplate, len(builtinTemplates))
	for k, v := range builtinTemplates {
		out[k] = v
	}
	return out
}

// Template returns a single template by name. The second return is false
// when no template matches, mirroring the two-value map lookup idiom.
func Template(name string) (*ProviderTemplate, bool) {
	tpl, ok := builtinTemplates[name]
	return tpl, ok
}

// ListTemplates returns the catalog sorted by DisplayName (case-sensitive,
// stable) â€” the order an admin UI should render in the template picker.
func ListTemplates() []*ProviderTemplate {
	out := make([]*ProviderTemplate, 0, len(builtinTemplates))
	for _, tpl := range builtinTemplates {
		out = append(out, tpl)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].DisplayName < out[j].DisplayName
	})
	return out
}

// ApplyTemplate builds a *storage.VaultProvider from a template plus caller
// overrides. The returned struct is ready to hand to Manager.CreateProvider
// (which expects the caller to pass client_secret as a separate plaintext
// argument â€” it is intentionally NOT set here, keeping the crypto boundary
// with the Manager).
//
// Behaviour:
//   - Name is always copied from the template (admins shouldn't rename it).
//   - displayName, when non-empty, overrides the template DisplayName;
//     otherwise the template default is used.
//   - scopes, when non-nil and non-empty, overrides DefaultScopes; otherwise
//     a fresh copy of DefaultScopes is used (so callers can't mutate the
//     shared slice).
//   - Active defaults to true â€” admins register templates to use them.
//
// Returns nil when tpl is nil; callers should validate before calling.
func ApplyTemplate(tpl *ProviderTemplate, clientID, displayName string, scopes []string) *storage.VaultProvider {
	if tpl == nil {
		return nil
	}

	effectiveDisplayName := tpl.DisplayName
	if displayName != "" {
		effectiveDisplayName = displayName
	}

	var effectiveScopes []string
	if len(scopes) > 0 {
		effectiveScopes = append([]string(nil), scopes...)
	} else {
		// Defensive copy so later mutations of the caller-supplied struct
		// don't bleed back into the shared template DefaultScopes slice.
		effectiveScopes = append([]string(nil), tpl.DefaultScopes...)
	}
	if effectiveScopes == nil {
		effectiveScopes = []string{}
	}

	// Copy ExtraAuthParams from the template so template-created providers
	// persist them and BuildAuthURL can read them directly from storage.
	var effectiveExtra map[string]string
	if len(tpl.ExtraAuthParams) > 0 {
		effectiveExtra = make(map[string]string, len(tpl.ExtraAuthParams))
		for k, v := range tpl.ExtraAuthParams {
			effectiveExtra[k] = v
		}
	} else {
		effectiveExtra = map[string]string{}
	}

	return &storage.VaultProvider{
		Name:        tpl.Name,
		DisplayName: effectiveDisplayName,
		AuthURL:     tpl.AuthURL,
		TokenURL:    tpl.TokenURL,
		ClientID:    clientID,
		// ClientSecretEnc is intentionally left empty â€” Manager.CreateProvider
		// encrypts the plaintext secret the caller supplies separately.
		Scopes:          effectiveScopes,
		IconURL:         tpl.IconURL,
		Active:          true,
		ExtraAuthParams: effectiveExtra,
	}
}
