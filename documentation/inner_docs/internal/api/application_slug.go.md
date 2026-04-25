# application_slug.go

**Path:** `internal/api/application_slug.go`
**Package:** `api`
**LOC:** 65
**Tests:** likely covered by `application_handlers_test.go` indirectly

## Purpose
Helper utilities for deriving + validating URL-safe slugs for applications. Sibling to `application_handlers.go`; isolated so the slug rules are easy to test + reuse.

## Handlers exposed
None — utility file.

## Functions
- `generateSlug(name string) string` (line 19) — derives a slug:
  1. lowercase + trim
  2. spaces and underscores → hyphens
  3. strip non-`[a-z0-9-]`
  4. collapse `--` to `-`, trim leading/trailing `-`
  5. prefix `app-` if shorter than 3
  6. truncate to max 64 chars
- `validateSlug(s string) error` (line 60) — checks against the package-shared `slugRE` (declared in `organization_handlers.go`): 3–64 chars, lowercase alnum + internal hyphens, first/last char must be alnum.

## Key types
None.

## Imports of note
- `fmt`, `strings` only — no external deps.

## Wired by
- Used by `application_handlers.go` (handleCreateApp, handleUpdateApp) when slug isn't supplied or needs validation.

## Notes
- `slugRE` lives in `organization_handlers.go`; this file deliberately depends on that shared regex rather than redefining its own.
- `generateSlug` always returns at least 3 chars (the `app-` fallback), so it never produces an empty slug.
