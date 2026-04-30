package storage

// defaultEmailTemplateSeeds returns the V1 templates. Ported from existing
// internal/email/templates/*.html content to structured fields.
// Welcome is new (fires on email verification per spec OQ1).
func defaultEmailTemplateSeeds() []*EmailTemplate {
	return []*EmailTemplate{
		{
			ID:             "magic_link",
			Subject:        "Sign in to {{.AppName}}",
			Preheader:      "Secure sign-in link for your account.",
			HeaderText:     "Magic sign-in link",
			BodyParagraphs: []string{"We received a request to sign in to your {{.AppName}} account. Click the button below to be logged in automatically.", "This link is valid for a single use and will expire in 15 minutes."},
			CTAText:        "Sign in to {{.AppName}}",
			CTAURLTemplate: "{{.MagicLinkURL}}",
			FooterText:     "If you did not request this link, you can safely ignore this email.",
		},
		{
			ID:             "password_reset",
			Subject:        "Reset your {{.AppName}} password",
			Preheader:      "Secure password reset link for your account.",
			HeaderText:     "Reset your password",
			BodyParagraphs: []string{"A password reset was requested for your {{.AppName}} account. Click the button below to choose a new password.", "For security, this link will expire in 15 minutes."},
			CTAText:        "Choose a new password",
			CTAURLTemplate: "{{.ResetURL}}",
			FooterText:     "If you did not request a password reset, your account is still secure and no action is required.",
		},
		{
			ID:             "verify_email",
			Subject:        "Verify your email for {{.AppName}}",
			Preheader:      "Confirm your email address to complete your registration.",
			HeaderText:     "Verify your email",
			BodyParagraphs: []string{"Thank you for joining {{.AppName}}. To complete your registration and secure your account, please verify your email address by clicking the button below."},
			CTAText:        "Verify email address",
			CTAURLTemplate: "{{.VerifyURL}}",
			FooterText:     "If you did not create an account with us, please ignore this email.",
		},
		{
			ID:             "organization_invitation",
			Subject:        "Invitation to join {{.OrgName}}",
			Preheader:      "{{.InviterEmail}} has invited you to join their team.",
			HeaderText:     "You've been invited",
			BodyParagraphs: []string{"{{.InviterEmail}} has invited you to join the {{.OrgName}} organization on {{.AppName}}.", "Join your team to start collaborating and managing your agent infrastructure."},
			CTAText:        "Accept Invitation",
			CTAURLTemplate: "{{.AcceptURL}}",
			FooterText:     "This invitation was sent to {{.UserEmail}} and will expire in 7 days.",
		},
		{
			ID:             "welcome",
			Subject:        "Welcome to {{.AppName}}!",
			Preheader:      "Your account is ready. Let's get started.",
			HeaderText:     "Welcome to {{.AppName}}",
			BodyParagraphs: []string{"We're excited to have you with us. Your account is now fully verified and ready to use.", "You can now start managing your identities and agents through your dashboard."},
			CTAText:        "Go to Dashboard",
			CTAURLTemplate: "{{.DashboardURL}}",
			FooterText:     "Need help getting started? Check out our documentation or reply to this email.",
		},
	}
}
