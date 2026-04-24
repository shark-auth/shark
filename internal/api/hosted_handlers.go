// Package api — hosted SPA shell handler (Phase B, task B7).
//
// handleHostedPage renders the HTML shell that bootstraps the hosted auth
// SPA at /hosted/{app_slug}/{page}. The shell inlines branding CSS vars,
// injects window.__SHARK_HOSTED with the per-request config JSON, and loads
// the immutable-cached JS bundle from /admin/hosted/assets/*.
package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sharkauth/sharkauth/internal/admin"
)

// validHostedPages is the allowlist of page names the hosted SPA handles.
var validHostedPages = map[string]bool{
	"login":   true,
	"signup":  true,
	"magic":   true,
	"passkey": true,
	"mfa":     true,
	"verify":  true,
	"error":   true,
}

// cssColorRE accepts values that look like CSS color values:
// hex (#abc, #aabbcc), named colors (all alpha), rgb/rgba, hsl/hsla.
// Rejects anything with semi-colons, braces, or other injection vectors.
var cssColorRE = regexp.MustCompile(`^[a-zA-Z0-9#(),. %]+$`)

// hostedBundleName caches the resolved bundle filename at init time.
// Empty string means no bundle was found; the handler serves a degraded shell
// that still loads (the SPA just won't bootstrap).
var hostedBundleName string

func init() {
	name, err := findHostedBundle()
	if err != nil {
		slog.Warn("hosted: bundle not found in embedded FS; hosted pages will load without JS",
			"err", err)
		return
	}
	hostedBundleName = name
}

// findHostedBundle reads the embedded admin FS and returns the filename of
// the hosted-*.js entry point. Returns an error when none is found. Safe to
// call in tests — it operates on the embedded FS, not the filesystem.
func findHostedBundle() (string, error) {
	// The embedded FS lives in internal/admin. Reach into its dist/hosted/assets/ tree.
	sub, err := fs.Sub(admin.DistFS(), "dist/hosted/assets")
	if err != nil {
		return "", fmt.Errorf("hosted assets sub-FS: %w", err)
	}
	entries, err := fs.ReadDir(sub, ".")
	if err != nil {
		return "", fmt.Errorf("hosted assets readdir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasPrefix(n, "hosted-") && strings.HasSuffix(n, ".js") {
			return n, nil
		}
	}
	return "", fmt.Errorf("no hosted-*.js found in dist/hosted/assets/")
}

// hostedAppInfo is the app sub-object injected into the SPA config.
type hostedAppInfo struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	LogoURL string `json:"logo_url,omitempty"`
}

// hostedBrandingInfo is the branding sub-object injected into the SPA config.
type hostedBrandingInfo struct {
	PrimaryColor   string `json:"primary_color,omitempty"`
	SecondaryColor string `json:"secondary_color,omitempty"`
	FontFamily     string `json:"font_family,omitempty"`
	LogoURL        string `json:"logo_url,omitempty"`
}

// hostedOAuthProvider is one entry in the oauthProviders array.
type hostedOAuthProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"iconUrl,omitempty"`
}

// hostedOAuthParams captures the OAuth authorize query params forwarded here
// by the authorization server's redirect-to-hosted-login flow.
type hostedOAuthParams struct {
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
	Scope       string `json:"scope,omitempty"`
}

// hostedConfig is the full window.__SHARK_HOSTED payload serialised to JSON
// and embedded in the shell's inline <script>.
type hostedConfig struct {
	App            hostedAppInfo         `json:"app"`
	Branding       hostedBrandingInfo    `json:"branding"`
	AuthMethods    []string              `json:"authMethods"`
	OAuthProviders []hostedOAuthProvider `json:"oauthProviders"`
	OAuth          hostedOAuthParams     `json:"oauth"`
}

// handleHostedPage serves the HTML shell for hosted auth pages.
// Route: GET /hosted/{app_slug}/{page}
func (s *Server) handleHostedPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appSlug := chi.URLParam(r, "app_slug")
	page := chi.URLParam(r, "page")

	// 1. Validate page name.
	if !validHostedPages[page] {
		http.NotFound(w, r)
		return
	}

	// 2. Resolve the application by slug.
	app, err := s.Store.GetApplicationBySlug(ctx, appSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		slog.Error("hosted: GetApplicationBySlug", "slug", appSlug, "err", err)
		http.NotFound(w, r)
		return
	}

	// 3. Integration mode gate — only "hosted" and "proxy" serve the shell.
	switch app.IntegrationMode {
	case "hosted", "proxy":
		// allowed
	default:
		http.Error(w, "hosted auth disabled", http.StatusNotFound)
		return
	}

	// 4. Resolve branding (global + per-app override merged).
	branding, err := s.Store.ResolveBranding(ctx, app.ID)
	if err != nil {
		slog.Warn("hosted: ResolveBranding failed; using defaults", "app_id", app.ID, "err", err)
	}

	// 5. Collect OAuth params from query string.
	q := r.URL.Query()
	oauthParams := hostedOAuthParams{
		ClientID:    q.Get("client_id"),
		RedirectURI: q.Get("redirect_uri"),
		State:       q.Get("state"),
		Scope:       q.Get("scope"),
	}

	// 6. Build auth methods list from server config.
	authMethods := s.resolveAuthMethods()

	// 7. Build OAuth providers list from config.
	oauthProviders := s.resolveOAuthProviders()

	// Build the app info.
	appInfo := hostedAppInfo{
		Slug: app.Slug,
		Name: app.Name,
	}
	var brandInfo hostedBrandingInfo
	if branding != nil {
		appInfo.LogoURL = branding.LogoURL
		brandInfo = hostedBrandingInfo{
			PrimaryColor:   branding.PrimaryColor,
			SecondaryColor: branding.SecondaryColor,
			FontFamily:     branding.FontFamily,
			LogoURL:        branding.LogoURL,
		}
	}

	// LIVE PREVIEW OVERRIDES (Phase Wave E)
	if q.Get("preview") == "true" {
		if p := q.Get("primary"); p != "" {
			brandInfo.PrimaryColor = p
		}
		if s := q.Get("secondary"); s != "" {
			brandInfo.SecondaryColor = s
		}
		if f := q.Get("font"); f != "" {
			brandInfo.FontFamily = f
		}
	}

	cfg := hostedConfig{
		App:            appInfo,
		Branding:       brandInfo,
		AuthMethods:    authMethods,
		OAuthProviders: oauthProviders,
		OAuth:          oauthParams,
	}

	// 8. Serialise config to JSON.
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		slog.Error("hosted: marshal config", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 9. Render the HTML shell.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	bundleSrc := ""
	if hostedBundleName != "" {
		bundleSrc = "/admin/hosted/assets/" + hostedBundleName
	}

	// Sanitise branding colors before embedding in CSS to prevent injection.
	primaryColor := sanitizeCSSValue(brandInfo.PrimaryColor, "#6366f1")
	secondaryColor := sanitizeCSSValue(brandInfo.SecondaryColor, "#4f46e5")
	fontFamily := sanitizeCSSValue(brandInfo.FontFamily, "Inter, system-ui, sans-serif")

	// Use template.JS so Go's html/template doesn't HTML-escape the JSON.
	// json.Marshal never emits </script> literally (it escapes < as \u003c),
	// so the XSS break-out vector is closed at the json.Marshal level.
	configScript := template.JS(cfgJSON) //nolint:gosec // json.Marshal escapes <, >, &

	data := struct {
		AppName      string
		PrimaryColor string
		SecColor     string
		FontFamily   string
		ConfigScript template.JS
		BundleSrc    string
	}{
		AppName:      app.Name,
		PrimaryColor: primaryColor,
		SecColor:     secondaryColor,
		FontFamily:   fontFamily,
		ConfigScript: configScript,
		BundleSrc:    bundleSrc,
	}

	const shellTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="color-scheme" content="dark">
  <title>{{.AppName}}</title>
  <style>
    :root {
      --shark-primary: {{.PrimaryColor}};
      --shark-secondary: {{.SecColor}};
      --font-display: {{.FontFamily}};
    }
    body { margin: 0; font-family: var(--font-display, Inter, system-ui, sans-serif); }
  </style>
  <script>window.__SHARK_HOSTED = {{.ConfigScript}};</script>
</head>
<body>
  <div id="hosted-root"></div>
  {{if .BundleSrc}}<script type="module" src="{{.BundleSrc}}"></script>{{end}}
</body>
</html>`

	tmpl, err := template.New("hosted").Parse(shellTmpl)
	if err != nil {
		slog.Error("hosted: parse shell template", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("hosted: execute shell template", "err", err)
		// Response may already be partially written; nothing more we can do.
	}
}

// handlePaywallPage serves the 402 Payment Required upgrade page shown
// when the proxy denies a request due to a ReqTier mismatch. Rendered as
// an inline HTML template keyed on the calling app's branding (same
// ResolveBranding seam as handleHostedPage).
//
// Route: GET /paywall/{app_slug}?tier=<required>&return=<url>
// Public — not admin-gated. The required tier comes from the proxy
// Decision.RequiredTier; the return URL is the original request URL so
// the upgrade CTA can loop the caller back after payment.
//
// XSS guards: tier + return are sanitized via sanitizeCSSValue-style
// allowlist before embedding in attribute/CSS contexts; html/template
// also escapes on expansion.
func (s *Server) handlePaywallPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appSlug := chi.URLParam(r, "app_slug")
	q := r.URL.Query()
	rawTier := q.Get("tier")
	rawReturn := q.Get("return")

	if strings.TrimSpace(rawTier) == "" {
		http.Error(w, "tier query param required", http.StatusBadRequest)
		return
	}

	// Resolve the application by slug. Fall through to 404 rather than
	// leaking "which apps exist" via a different error shape.
	app, err := s.Store.GetApplicationBySlug(ctx, appSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		slog.Error("paywall: GetApplicationBySlug", "slug", appSlug, "err", err)
		http.NotFound(w, r)
		return
	}

	// Resolve branding so the upgrade page shows the app's own colors.
	branding, err := s.Store.ResolveBranding(ctx, app.ID)
	if err != nil {
		slog.Warn("paywall: ResolveBranding failed; using defaults", "app_id", app.ID, "err", err)
	}
	var brandInfo hostedBrandingInfo
	if branding != nil {
		brandInfo = hostedBrandingInfo{
			PrimaryColor:   branding.PrimaryColor,
			SecondaryColor: branding.SecondaryColor,
			FontFamily:     branding.FontFamily,
			LogoURL:        branding.LogoURL,
		}
	}

	primaryColor := sanitizeCSSValue(brandInfo.PrimaryColor, "#6366f1")
	secondaryColor := sanitizeCSSValue(brandInfo.SecondaryColor, "#4f46e5")
	fontFamily := sanitizeCSSValue(brandInfo.FontFamily, "Inter, system-ui, sans-serif")

	// Tier label is surfaced inside the headline. Allowlist keeps it to
	// [a-zA-Z0-9 _-] so a URL-param injection can't break out of the
	// attribute/text context. html/template escaping is belt + suspenders.
	tier := sanitizeTierLabel(rawTier)

	// return param must be a same-origin-safe HTTP(S) URL. Reject anything
	// with a scheme we don't recognize (javascript:, data:, file:, ...).
	// The CTA href appends ?upgrade=<tier>; if return is empty we fall
	// back to "/".
	returnURL := sanitizeReturnURL(rawReturn)
	upgradeHref := buildUpgradeHref(returnURL, tier)

	data := struct {
		AppName        string
		Tier           string
		UpgradeHref    string
		PrimaryColor   string
		SecondaryColor string
		FontFamily     string
	}{
		AppName:        app.Name,
		Tier:           tier,
		UpgradeHref:    upgradeHref,
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,
		FontFamily:     fontFamily,
	}

	const paywallTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="color-scheme" content="dark light">
  <title>Upgrade required — {{.AppName}}</title>
  <style>
    :root {
      --shark-primary: {{.PrimaryColor}};
      --shark-secondary: {{.SecondaryColor}};
      --font-display: {{.FontFamily}};
    }
    html, body { margin: 0; padding: 0; min-height: 100%; font-family: var(--font-display); }
    body { display: flex; align-items: center; justify-content: center; background: #0b0b0f; color: #f5f5f7; }
    main { max-width: 520px; padding: 48px 32px; text-align: center; }
    h1 { margin: 0 0 16px; font-size: 2rem; color: var(--shark-primary); }
    p { margin: 0 0 24px; color: #c7c7cc; line-height: 1.5; }
    .cta {
      display: inline-block; padding: 12px 24px; border-radius: 8px;
      background: var(--shark-primary); color: #fff; text-decoration: none;
      font-weight: 600; transition: background 120ms ease;
    }
    .cta:hover { background: var(--shark-secondary); }
    .app { margin-top: 24px; font-size: 0.875rem; color: #8e8e93; }
  </style>
</head>
<body>
  <main>
    <h1>Upgrade to {{.Tier}}</h1>
    <p>This feature requires the <strong>{{.Tier}}</strong> plan. Upgrade to continue using {{.AppName}}.</p>
    <a class="cta" href="{{.UpgradeHref}}">Upgrade to {{.Tier}}</a>
    <div class="app">{{.AppName}}</div>
  </main>
</body>
</html>`

	tmpl, err := template.New("paywall").Parse(paywallTmpl)
	if err != nil {
		slog.Error("paywall: parse template", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusPaymentRequired)
	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("paywall: execute template", "err", err)
	}
}

// tierLabelRE restricts tier labels to characters safe for embedding in
// HTML text + query params. Anything else falls back to a generic label.
var tierLabelRE = regexp.MustCompile(`^[a-zA-Z0-9 _-]{1,32}$`)

// sanitizeTierLabel returns value if it passes the tier-label regex,
// otherwise returns "upgrade" as a safe fallback.
func sanitizeTierLabel(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "upgrade"
	}
	if tierLabelRE.MatchString(v) {
		return v
	}
	return "upgrade"
}

// sanitizeReturnURL allowlists only absolute http(s) URLs and absolute
// paths. Anything else (javascript:, data:, relative with weird chars)
// falls back to "/" so the CTA still works without opening an
// open-redirect / XSS vector.
func sanitizeReturnURL(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "/"
	}
	// Reject control chars / quotes / angle-brackets up front — these
	// should never appear in a legitimate URL.
	for _, c := range v {
		if c < 0x20 || c == '"' || c == '\'' || c == '<' || c == '>' || c == '`' {
			return "/"
		}
	}
	if strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "//") {
		return v
	}
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	return "/"
}

// buildUpgradeHref appends ?upgrade=<tier> to returnURL while preserving
// any existing query string.
func buildUpgradeHref(returnURL, tier string) string {
	esc := escapeQueryValue(tier)
	if strings.Contains(returnURL, "?") {
		return returnURL + "&upgrade=" + esc
	}
	return returnURL + "?upgrade=" + esc
}

// escapeQueryValue URL-encodes a restricted tier label. We already
// constrained tier via sanitizeTierLabel so the output is safe; this
// handles the embedded space character defensively.
func escapeQueryValue(s string) string {
	// sanitizeTierLabel already stripped anything outside [a-zA-Z0-9 _-]
	// so a simple space → %20 swap keeps the URL valid. All other
	// allowed characters are URL-safe.
	return strings.ReplaceAll(s, " ", "%20")
}

// handleHostedAssets serves embedded static assets for the hosted SPA from
// /admin/hosted/assets/*. The files are content-hash-named so they get
// immutable cache headers.
func (s *Server) handleHostedAssets(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(admin.DistFS(), "dist/hosted/assets")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Resolve the path segment after /admin/hosted/assets/
	file := strings.TrimPrefix(r.URL.Path, "/admin/hosted/assets/")
	file = path.Clean("/" + file)
	if file == "/" || strings.Contains(file[1:], "/") {
		http.NotFound(w, r)
		return
	}

	// Hashed filenames are safe to cache forever.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	// Strip prefix and serve from the sub-FS.
	http.StripPrefix("/admin/hosted/assets",
		http.FileServer(http.FS(sub)),
	).ServeHTTP(w, r)
}

// resolveAuthMethods returns the list of enabled auth methods from server config.
// Assumption: all four methods are always present unless a method-specific config
// explicitly disables it. Currently SharkAuth has no per-method enable flag at
// the application level — the config drives which sub-systems are wired in at
// server startup. We surface all four so the SPA renders the full UI; the
// individual sub-handlers will return 404/error if a method isn't configured.
func (s *Server) resolveAuthMethods() []string {
	return []string{"password", "magic_link", "passkey", "oauth"}
}

// resolveOAuthProviders returns the configured social OAuth providers derived
// from the server config, matching the shape the SPA's oauthProviders array.
func (s *Server) resolveOAuthProviders() []hostedOAuthProvider {
	if s.Config == nil {
		return []hostedOAuthProvider{}
	}
	cfg := s.Config
	var out []hostedOAuthProvider
	if cfg.Social.Google.ClientID != "" && cfg.Social.Google.ClientSecret != "" {
		out = append(out, hostedOAuthProvider{ID: "google", Name: "Google"})
	}
	if cfg.Social.GitHub.ClientID != "" && cfg.Social.GitHub.ClientSecret != "" {
		out = append(out, hostedOAuthProvider{ID: "github", Name: "GitHub"})
	}
	if cfg.Social.Apple.ClientID != "" && cfg.Social.Apple.TeamID != "" {
		out = append(out, hostedOAuthProvider{ID: "apple", Name: "Apple"})
	}
	if cfg.Social.Discord.ClientID != "" && cfg.Social.Discord.ClientSecret != "" {
		out = append(out, hostedOAuthProvider{ID: "discord", Name: "Discord"})
	}
	if out == nil {
		return []hostedOAuthProvider{}
	}
	return out
}

// sanitizeCSSValue returns value if it passes the CSS-safe regex, otherwise
// returns fallback. Prevents CSS injection through branding values.
func sanitizeCSSValue(value, fallback string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return fallback
	}
	if cssColorRE.MatchString(v) {
		return v
	}
	return fallback
}

