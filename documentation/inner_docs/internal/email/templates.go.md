# templates.go

**Path:** `internal/email/templates.go`  
**Package:** `email`  
**LOC:** 222  
**Tests:** `templates_test.go`

## Purpose
Email template rendering with DB fallback. Supports magic link, password reset, email verification, organization invitation, and welcome emails. Loads from embed.FS (embedded HTML) or DB (customizable templates).

## Key types / functions
- `Rendered` (struct, line 30) — Subject, HTML (output of all RenderX funcs)
- `MagicLinkData`, `PasswordResetData`, `VerifyEmailData`, `OrganizationInvitationData`, `WelcomeData` — template input structs
- `RenderMagicLink(ctx, store, branding, data)` (func, line 44) — magic link email
- `RenderPasswordReset(ctx, store, branding, data)` (func, line 65) — password reset email
- `RenderVerifyEmail(ctx, store, branding, data)` (func, line 86) — email verification email
- `RenderOrganizationInvitation(ctx, store, branding, data)` (func, line 110) — org invite email
- `RenderWelcome(ctx, store, branding, data)` (func, line 134) — welcome email (degrades gracefully on template miss)
- `RenderStructured(tmpl, branding, data)` (func, line 148) — public API for preview endpoints

## Imports of note
- `embed` — embedded template files
- `html/template` — Go template engine
- `internal/storage` — EmailTemplate + BrandingConfig types

## Wired by
- `internal/api` handlers call RenderX functions before Sender.Send()
- Admin API preview endpoints call RenderStructured for template validation

## Notes
- Embedded templates: `templates/*.html` + `layout.html`
- DB lookup: tries store.GetEmailTemplate(ctx, template_id) first, falls back to embedded
- Branding config passed for logo/color customization in layout
- Welcome email degrades to minimal hardcoded body on template miss (not critical)
- All others block on template rendering errors
- Subject lines auto-generated from AppName + context (no custom subject templates)

