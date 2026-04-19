package oauth

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed consent_templates/*.html
var consentTemplatesFS embed.FS

var consentTemplates *template.Template

func init() {
	consentTemplates = template.Must(template.ParseFS(consentTemplatesFS, "consent_templates/*.html"))
}

// ConsentData holds data for the consent template.
type ConsentData struct {
	AgentName    string
	AgentLogoURI string
	ClientID     string
	Scopes       []string
	RedirectURI  string
	State        string
	// Hidden form fields to POST back (option B: re-send all OAuth params)
	Challenge string // opaque identifier for this consent request
	Issuer    string
}

// RenderConsentPage renders the consent HTML page.
func RenderConsentPage(w http.ResponseWriter, data ConsentData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'")
	consentTemplates.ExecuteTemplate(w, "consent.html", data) //nolint:errcheck
}

// ErrorData holds data for the error template.
type ErrorData struct {
	Error       string
	Description string
	Issuer      string
}

// RenderErrorPage renders the OAuth error HTML page.
func RenderErrorPage(w http.ResponseWriter, status int, data ErrorData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	consentTemplates.ExecuteTemplate(w, "error.html", data) //nolint:errcheck
}
