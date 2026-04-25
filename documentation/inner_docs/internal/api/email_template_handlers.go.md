# email_template_handlers.go

**Path:** `internal/api/email_template_handlers.go`
**Package:** `api`
**LOC:** 266
**Tests:** likely integration-tested

## Purpose
Admin-key CRUD + preview + send-test + reset for the seeded email templates (Phase A task A7). Renders templates via `internal/email.RenderStructured` with optional branding + sample-data overrides.

## Handlers exposed
- `handleListEmailTemplates` (line 41) — GET `/admin/email-templates`. `{data: []}`.
- `handleGetEmailTemplate` (line 54) — GET `/{id}`. 404 via `isEmailTemplateNotFound`.
- `handlePatchEmailTemplate` (line 72) — PATCH `/{id}`. Storage allowlists fields; returns freshly-read row.
- `handlePreviewEmailTemplate` (line 109) — POST `/{id}/preview`. Optional body `{config?, sample_data?}`; renders HTML+subject for sandboxed iframe.
- `handleSendTestEmail` (line 166) — POST `/{id}/send-test`. Renders w/ default sample data, sends via the same magic-link sender pipe — proves the whole email stack end-to-end.
- `handleResetEmailTemplate` (line 226) — POST `/{id}/reset`. Restores the seeded defaults.

## Key types
- `previewRequest` (line 100) — shared body for preview + send-test: `{config?, sample_data?, to_email?}`.

## Helpers
- `isEmailTemplateNotFound` (line 22) — string-prefix bridge until storage exposes typed error.
- `defaultSampleData` (line 31) — `{AppName, Link, UserEmail}` fallback variable bag.

## Imports of note
- `internal/email` — `RenderStructured`
- `internal/storage` — `EmailTemplate`, `BrandingConfig`, `ResolveBranding`

## Wired by
- `internal/api/router.go:658-663`

## Notes
- `to_email` validation is the only check on send-test (no regex/DNS/allowlist) — admins target whatever they like.
- Empty PATCH bodies short-circuit to a 404 if the template id doesn't exist (so unknown ids return 404, not 200-noop).
