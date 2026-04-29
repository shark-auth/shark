package email

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"

	"github.com/shark-auth/shark/internal/storage"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates *template.Template

//go:embed layout.html
var emailLayoutHTML string

var emailLayout = template.Must(template.New("layout").Parse(emailLayoutHTML))

func init() {
	templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))
}

// Rendered is the output of every Render* function: the resolved subject
// line plus the fully-rendered HTML body. Subject is separated so callers
// can pass it to the SMTP sender without re-parsing the body.
type Rendered struct {
	Subject string
	HTML    string
}

// MagicLinkData holds the template data for a magic link email.
type MagicLinkData struct {
	AppName       string
	MagicLinkURL  string
	ExpiryMinutes int
}

// RenderMagicLink renders the magic link email. It first tries a DB-backed
// template by id ("magic_link") and falls back to the embedded HTML on miss.
func RenderMagicLink(ctx context.Context, store storage.Store, branding *storage.BrandingConfig, data MagicLinkData) (Rendered, error) {
	if store != nil {
		if tmpl, err := store.GetEmailTemplate(ctx, "magic_link"); err == nil && tmpl != nil {
			return renderStructured(tmpl, branding, data)
		}
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "magic_link.html", data); err != nil {
		return Rendered{}, fmt.Errorf("fallback render magic link: %w", err)
	}
	return Rendered{Subject: fmt.Sprintf("Sign in to %s", data.AppName), HTML: buf.String()}, nil
}

// PasswordResetData holds the template data for a password reset email.
type PasswordResetData struct {
	AppName       string
	ResetURL      string
	ExpiryMinutes int
}

// RenderPasswordReset renders the password reset email.
func RenderPasswordReset(ctx context.Context, store storage.Store, branding *storage.BrandingConfig, data PasswordResetData) (Rendered, error) {
	if store != nil {
		if tmpl, err := store.GetEmailTemplate(ctx, "password_reset"); err == nil && tmpl != nil {
			return renderStructured(tmpl, branding, data)
		}
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "password_reset.html", data); err != nil {
		return Rendered{}, fmt.Errorf("fallback render password reset: %w", err)
	}
	return Rendered{Subject: fmt.Sprintf("Reset your %s password", data.AppName), HTML: buf.String()}, nil
}

// VerifyEmailData holds the template data for an email verification email.
type VerifyEmailData struct {
	AppName       string
	VerifyURL     string
	ExpiryMinutes int
}

// RenderVerifyEmail renders the email verification email.
func RenderVerifyEmail(ctx context.Context, store storage.Store, branding *storage.BrandingConfig, data VerifyEmailData) (Rendered, error) {
	if store != nil {
		if tmpl, err := store.GetEmailTemplate(ctx, "verify_email"); err == nil && tmpl != nil {
			return renderStructured(tmpl, branding, data)
		}
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "verify_email.html", data); err != nil {
		return Rendered{}, fmt.Errorf("fallback render verify email: %w", err)
	}
	return Rendered{Subject: fmt.Sprintf("Verify your %s email", data.AppName), HTML: buf.String()}, nil
}

// OrganizationInvitationData holds template data for an invitation email.
type OrganizationInvitationData struct {
	AppName      string
	OrgName      string
	Role         string
	AcceptURL    string
	InviterEmail string
	ExpiryHours  int
}

// RenderOrganizationInvitation renders the organization invitation email.
func RenderOrganizationInvitation(ctx context.Context, store storage.Store, branding *storage.BrandingConfig, data OrganizationInvitationData) (Rendered, error) {
	if store != nil {
		if tmpl, err := store.GetEmailTemplate(ctx, "organization_invitation"); err == nil && tmpl != nil {
			return renderStructured(tmpl, branding, data)
		}
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "organization_invitation.html", data); err != nil {
		return Rendered{}, fmt.Errorf("fallback render organization invitation: %w", err)
	}
	return Rendered{Subject: fmt.Sprintf("You're invited to %s", data.OrgName), HTML: buf.String()}, nil
}

// WelcomeData holds the template data for the welcome email sent after
// successful email verification.
type WelcomeData struct {
	AppName      string
	UserEmail    string
	DashboardURL string
}

// RenderWelcome renders the welcome email. When no DB template exists (or
// lookup fails) it degrades to a minimal hardcoded body rather than failing
// the caller â€” a missed welcome is not worth blocking signup on.
func RenderWelcome(ctx context.Context, store storage.Store, branding *storage.BrandingConfig, data WelcomeData) (Rendered, error) {
	if store != nil {
		if tmpl, err := store.GetEmailTemplate(ctx, "welcome"); err == nil && tmpl != nil {
			return renderStructured(tmpl, branding, data)
		}
	}
	return Rendered{
		Subject: fmt.Sprintf("Welcome to %s", data.AppName),
		HTML:    "<p>Thanks for verifying your email.</p>",
	}, nil
}

// RenderStructured exposes the internal render path for preview endpoints.
// Matches renderStructured signature exactly.
func RenderStructured(tmpl *storage.EmailTemplate, branding *storage.BrandingConfig, data any) (Rendered, error) {
	return renderStructured(tmpl, branding, data)
}

// renderStructured takes a DB-backed EmailTemplate and composes it into the
// branded shared layout. Per-field Go-template syntax inside the EmailTemplate
// rows is evaluated against data, then the final strings are slotted into
// layout.html alongside the resolved branding (colors, logo, footer).
func renderStructured(tmpl *storage.EmailTemplate, branding *storage.BrandingConfig, data any) (Rendered, error) {
	subject, err := execString(tmpl.Subject, data)
	if err != nil {
		return Rendered{}, fmt.Errorf("email subject: %w", err)
	}
	header, _ := execString(tmpl.HeaderText, data)
	paragraphs := make([]string, len(tmpl.BodyParagraphs))
	for i, p := range tmpl.BodyParagraphs {
		paragraphs[i], _ = execString(p, data)
	}
	ctaText, _ := execString(tmpl.CTAText, data)
	ctaURL, _ := execString(tmpl.CTAURLTemplate, data)
	footer, _ := execString(tmpl.FooterText, data)
	preheader, _ := execString(tmpl.Preheader, data)

	layoutData := struct {
		Branding   *storage.BrandingConfig
		Preheader  string
		Header     string
		Paragraphs []string
		CTAText    string
		CTAURL     string
		Footer     string
	}{
		Branding:   brandingOrDefault(branding),
		Preheader:  preheader,
		Header:     header,
		Paragraphs: paragraphs,
		CTAText:    ctaText,
		CTAURL:     ctaURL,
		Footer:     footer,
	}

	var buf bytes.Buffer
	if err := emailLayout.Execute(&buf, layoutData); err != nil {
		return Rendered{}, fmt.Errorf("email layout render: %w", err)
	}
	return Rendered{Subject: subject, HTML: buf.String()}, nil
}

// brandingOrDefault guarantees the layout template never dereferences a nil
// pointer. Callers that don't resolve branding (e.g. pure-fallback tests)
// should still get a valid render with sensible defaults.
func brandingOrDefault(b *storage.BrandingConfig) *storage.BrandingConfig {
	if b != nil {
		return b
	}
	return &storage.BrandingConfig{PrimaryColor: "#2563eb"}
}

// execString parses a small Go-template snippet from a DB column against the
// caller's data struct. Empty snippets short-circuit so we don't waste a
// template.New per missing field.
func execString(tmplStr string, data any) (string, error) {
	if tmplStr == "" {
		return "", nil
	}
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
