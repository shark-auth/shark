package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates *template.Template

func init() {
	templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))
}

// MagicLinkData holds the template data for a magic link email.
type MagicLinkData struct {
	AppName       string
	MagicLinkURL  string
	ExpiryMinutes int
}

// RenderMagicLink renders the magic link email template.
func RenderMagicLink(data MagicLinkData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "magic_link.html", data); err != nil {
		return "", fmt.Errorf("rendering magic link template: %w", err)
	}
	return buf.String(), nil
}

// VerifyEmailData holds the template data for an email verification email.
type VerifyEmailData struct {
	AppName       string
	VerifyURL     string
	ExpiryMinutes int
}

// RenderVerifyEmail renders the email verification template.
func RenderVerifyEmail(data VerifyEmailData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "verify_email.html", data); err != nil {
		return "", fmt.Errorf("rendering verify email template: %w", err)
	}
	return buf.String(), nil
}
