# email_templates_seed.go

**Path:** `internal/storage/email_templates_seed.go`
**Package:** `storage`
**LOC:** 59
**Tests:** none direct (covered by `email_templates_test.go::TestSeed`).

## Purpose
Holds the V1 default email template content as Go literals so the storage layer is self-contained for seeding. Ported from existing `internal/email/templates/*.html` content into structured fields.

## Functions
- `defaultEmailTemplateSeeds()` (line 6) — returns `[]*EmailTemplate` for: `magic_link`, `password_reset`, `verify_email`, `organization_invitation`, `welcome`.

## Imports of note
None.

## Used by
- `email_templates.go::SeedEmailTemplates` (idempotent first-boot seed).
- `email_templates_export.go::DefaultEmailTemplateSeedsForExport` (admin reset action).

## Notes
- All copy uses Go-template placeholders (`{{.AppName}}`, `{{.MagicLinkURL}}`, etc.) — actual rendering happens in `internal/email`.
- `welcome` template was added per spec OQ1 (fires on email verification, not on signup).
- Copy is intentionally short and English-only; localization would extend the row schema.
