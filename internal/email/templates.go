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

// PasswordResetData holds the template data for a password reset email.
type PasswordResetData struct {
	AppName       string
	ResetURL      string
	ExpiryMinutes int
}

// RenderPasswordReset renders the password reset email template.
func RenderPasswordReset(data PasswordResetData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "password_reset.html", data); err != nil {
		return "", fmt.Errorf("rendering password reset template: %w", err)
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

// OrganizationInvitationData holds template data for an invitation email.
type OrganizationInvitationData struct {
	AppName       string
	OrgName       string
	Role          string
	AcceptURL     string
	InviterEmail  string
	ExpiryHours   int
}

// RenderOrganizationInvitation renders the invitation email template.
func RenderOrganizationInvitation(data OrganizationInvitationData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "organization_invitation.html", data); err != nil {
		return "", fmt.Errorf("rendering organization invitation template: %w", err)
	}
	return buf.String(), nil
}
