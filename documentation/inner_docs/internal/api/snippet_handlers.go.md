# snippet_handlers.go

**Path:** `internal/api/snippet_handlers.go`
**Package:** `api`
**LOC:** 79
**Tests:** likely integration-tested

## Purpose
Phase A task A8 — renders a framework-specific "paste this into your app" code-snippet bundle for a registered application. Substitutes the real ClientID + auth base URL into the install command, provider setup, and a page-usage example so copy-paste just works.

## Handlers exposed
- `handleAppSnippet` (line 22) — GET `/api/v1/admin/apps/{id}/snippet?framework=react`. Default framework is `react`; any other value returns 501 `framework_not_supported`. Unknown app id returns 404. Returns `{framework, snippets: [{label, lang, code}]}`.

## Key types
None — inline `map[string]any` response.

## Imports of note
- `github.com/go-chi/chi/v5` — URL params
- Resolves app via `s.getAppByIDOrClientID` (declared in `application_handlers.go`).

## Wired by
- `internal/api/router.go:530`

## Notes
- Only React is supported today; the 501 on other framework values is the published growth surface for vue/svelte/etc.
- The base URL comes from `s.Config.Server.BaseURL`; if Config is nil (test mode), it falls back to empty string.
- Snippet labels: `Install`, `Provider setup`, `Page usage` — three blocks.
