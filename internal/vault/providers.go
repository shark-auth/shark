package vault

import (
	"sort"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// ProviderTemplate is a pre-configured OAuth 2.0 endpoint descriptor.
// Admins pick a template + supply their own (client_id, client_secret) —
// the template itself never carries credentials. Turning a template into a
// *storage.VaultProvider goes through ApplyTemplate; actual persistence
// (including secret encryption) still flows through Manager.CreateProvider.
type ProviderTemplate struct {
	// Name is the unique short key — e.g. "google_calendar", "slack",
	// "github". Used as the VaultProvider.Name when the template is
	// applied, and as the lookup key in the registry.
	Name string
	// DisplayName is the user-visible label, e.g. "Google Calendar".
	DisplayName string
	// AuthURL is the provider's OAuth 2.0 authorization endpoint.
	AuthURL string
	// TokenURL is the provider's OAuth 2.0 token endpoint.
	TokenURL string
	// DefaultScopes is the recommended scope list for this template. May
	// be empty (e.g. Notion doesn't use scopes). Never nil.
	DefaultScopes []string
	// IconURL is an optional public URL for a branded icon. Empty when
	// we don't want to commit to a specific CDN path.
	IconURL string
}

// builtinTemplates is the internal registry. Access via Templates() /
// Template() / ListTemplates() rather than touching the map directly so
// we can swap storage strategies later without breaking callers.
//
// TODO(T4): some providers require extra auth-URL query params that the
// oauth2.Config doesn't model cleanly:
//   - Linear: prompt=consent to force the consent screen every time.
//   - Jira (Atlassian): audience=api.atlassian.com & prompt=consent.
//
// The handler layer (T4) adds these via oauth2.AuthCodeOption when building
// the authorize URL; we intentionally do NOT bake them into the template so
// the template stays a pure endpoint descriptor.
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
		Name:          "slack",
		DisplayName:   "Slack",
		AuthURL:       "https://slack.com/oauth/v2/authorize",
		TokenURL:      "https://slack.com/api/oauth.v2.access",
		DefaultScopes: []string{"chat:write", "channels:read", "users:read"},
		IconURL:       "https://a.slack-edge.com/80588/marketing/img/meta/favicon-32.png",
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
	},
	"notion": {
		Name:        "notion",
		DisplayName: "Notion",
		AuthURL:     "https://api.notion.com/v1/oauth/authorize",
		TokenURL:    "https://api.notion.com/v1/oauth/token",
		// Notion's OAuth 2.0 flow does not use scopes — the integration's
		// permissions are configured in the Notion admin UI at install time.
		DefaultScopes: []string{},
		IconURL:       "https://www.notion.so/front-static/favicon.ico",
	},
	"linear": {
		Name:        "linear",
		DisplayName: "Linear",
		AuthURL:     "https://linear.app/oauth/authorize",
		TokenURL:    "https://api.linear.app/oauth/token",
		// Linear requires prompt=consent at authorize-URL build time; see
		// the TODO on builtinTemplates above.
		DefaultScopes: []string{"read", "write"},
		IconURL:       "https://linear.app/favicon.ico",
	},
	"jira": {
		Name:        "jira",
		DisplayName: "Jira Cloud",
		AuthURL:     "https://auth.atlassian.com/authorize",
		TokenURL:    "https://auth.atlassian.com/oauth/token",
		// Atlassian requires audience=api.atlassian.com & prompt=consent at
		// authorize-URL build time; see the TODO on builtinTemplates above.
		DefaultScopes: []string{"read:jira-work", "write:jira-work", "offline_access"},
		IconURL:       "https://www.atlassian.com/favicon.ico",
	},
}

// Templates returns a copy of the built-in catalog keyed by Name. The map
// is a shallow copy so callers can iterate/mutate without disturbing the
// package-level registry, but the *ProviderTemplate pointers are shared —
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
// stable) — the order an admin UI should render in the template picker.
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
// argument — it is intentionally NOT set here, keeping the crypto boundary
// with the Manager).
//
// Behaviour:
//   - Name is always copied from the template (admins shouldn't rename it).
//   - displayName, when non-empty, overrides the template DisplayName;
//     otherwise the template default is used.
//   - scopes, when non-nil and non-empty, overrides DefaultScopes; otherwise
//     a fresh copy of DefaultScopes is used (so callers can't mutate the
//     shared slice).
//   - Active defaults to true — admins register templates to use them.
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

	return &storage.VaultProvider{
		Name:        tpl.Name,
		DisplayName: effectiveDisplayName,
		AuthURL:     tpl.AuthURL,
		TokenURL:    tpl.TokenURL,
		ClientID:    clientID,
		// ClientSecretEnc is intentionally left empty — Manager.CreateProvider
		// encrypts the plaintext secret the caller supplies separately.
		Scopes:  effectiveScopes,
		IconURL: tpl.IconURL,
		Active:  true,
	}
}
