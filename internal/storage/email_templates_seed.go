package storage

// defaultEmailTemplateSeeds returns the V1 templates. Ported from existing
// internal/email/templates/*.html content to structured fields.
// Welcome is new (fires on email verification per spec OQ1).
func defaultEmailTemplateSeeds() []*EmailTemplate {
	return []*EmailTemplate{
		{
			ID:             "magic_link",
			Subject:        "Sign in to {{.AppName}}",
			Preheader:      "Your magic link expires in 15 minutes.",
			HeaderText:     "Click to sign in",
			BodyParagraphs: []string{"Use the link below to sign in to your account."},
			CTAText:        "Sign in now",
			CTAURLTemplate: "{{.Link}}",
			FooterText:     "This link expires in 15 minutes. If you didn't request it, ignore this email.",
		},
		{
			ID:             "password_reset",
			Subject:        "Reset your {{.AppName}} password",
			Preheader:      "Click to choose a new password.",
			HeaderText:     "Reset your password",
			BodyParagraphs: []string{"We received a request to reset your password. Click the button below to choose a new one."},
			CTAText:        "Reset password",
			CTAURLTemplate: "{{.Link}}",
			FooterText:     "This link expires in 15 minutes. If you didn't request a reset, ignore this email.",
		},
		{
			ID:             "verify_email",
			Subject:        "Verify your email",
			Preheader:      "Confirm your email to finish signing up.",
			HeaderText:     "One last step",
			BodyParagraphs: []string{"Verify your email so we know it's really you."},
			CTAText:        "Verify email",
			CTAURLTemplate: "{{.Link}}",
			FooterText:     "If you didn't create an account, ignore this email.",
		},
		{
			ID:             "organization_invitation",
			Subject:        "{{.InviterName}} invited you to {{.OrgName}}",
			Preheader:      "Accept your invitation.",
			HeaderText:     "You've been invited",
			BodyParagraphs: []string{"{{.InviterName}} invited you to join {{.OrgName}} on {{.AppName}}."},
			CTAText:        "Accept invitation",
			CTAURLTemplate: "{{.Link}}",
			FooterText:     "This invitation expires in 7 days.",
		},
		{
			ID:             "welcome",
			Subject:        "Welcome to {{.AppName}}",
			Preheader:      "You're all set.",
			HeaderText:     "Welcome aboard",
			BodyParagraphs: []string{"Thanks for verifying your email. You're ready to go."},
			CTAText:        "Get started",
			CTAURLTemplate: "{{.DashboardURL}}",
			FooterText:     "Questions? Just reply to this email.",
		},
	}
}
