# email_templates_export.go

**Path:** `internal/storage/email_templates_export.go`
**Package:** `storage`
**LOC:** 7
**Tests:** none (one-line wrapper).

## Purpose
Public re-export of the unexported `defaultEmailTemplateSeeds()` so the API package can offer a "Reset to defaults" action without duplicating the seed table.

## Functions
- `DefaultEmailTemplateSeedsForExport()` (line 5) — returns `[]*EmailTemplate` from `defaultEmailTemplateSeeds()`.

## Imports of note
None.

## Used by
- `internal/api/email_templates.go` — POST /email-templates/{id}/reset handler.

## Notes
- Pure naming wrapper. Could be inlined if Go ever supported package-private exports differently; today this is the cleanest separation.
