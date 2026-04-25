# email_templates.go

**Path:** `internal/storage/email_templates.go`
**Package:** `storage`
**LOC:** 118
**Tests:** `email_templates_test.go`

## Purpose
SQLite implementation of editable email-template CRUD plus `SeedEmailTemplates` for first-boot population. Backs the dashboard "Email templates" section.

## Interface methods implemented
- `ListEmailTemplates` (14) — JSON-decodes `body_paragraphs` into `[]string`
- `GetEmailTemplate` (40)
- `UpdateEmailTemplate` (64) — partial update with allowlist (`subject`, `preheader`, `header_text`, `body_paragraphs`, `cta_text`, `cta_url_template`, `footer_text`); coerces `[]string`/`[]any` for `body_paragraphs`
- `SeedEmailTemplates` (102) — `INSERT OR IGNORE` of V1 seeds (idempotent)

## Tables touched
- email_templates

## Imports of note
- `database/sql`, `encoding/json`, `errors`, `strings`, `time`

## Used by
- `internal/api/email_templates.go` admin CRUD
- `internal/email` for runtime template lookup
- `cmd/shark` first-boot seeding pipeline

## Notes
- Body paragraphs persist as a JSON array string — schema stays a single TEXT column.
- Defaults seeded by `defaultEmailTemplateSeeds()` (in `email_templates_seed.go`): `magic_link`, `password_reset`, `verify_email`, `organization_invitation`, `welcome`.
- `DefaultEmailTemplateSeedsForExport` (in `email_templates_export.go`) re-exposes the seeds so the API can offer a "Reset to defaults" action without re-running migrations.
