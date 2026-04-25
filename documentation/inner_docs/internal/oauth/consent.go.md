# consent.go

**Path:** `internal/oauth/consent.go`
**Package:** `oauth`
**LOC:** 54
**Tests:** none direct (templates exercised via `handlers_test.go`)

## Purpose
HTML rendering for the OAuth consent page and OAuth-error page. Templates are embedded at compile time.

## RFCs implemented
- N/A — UX layer over RFC 6749 §4.1 authorize flow.

## Key types / functions
- `consentTemplatesFS` (var, line 10) — `//go:embed consent_templates/*.html` embeds templates into the binary.
- `consentTemplates` (var, line 12) — parsed once in `init()` (line 14).
- `ConsentData` (struct, line 19) — fields: AgentName, AgentLogoURI, ClientID, Scopes, Resource (RFC 8707 audience disclosure), RedirectURI, State, Challenge, Issuer.
- `RenderConsentPage` (func, line 35) — sets `X-Frame-Options: DENY` + restrictive CSP, executes `consent.html`.
- `ErrorData` (struct, line 43) + `RenderErrorPage` (func, line 50) — error page renderer.

## Imports of note
- `embed`, `html/template`, `net/http` only.

## Wired by
- `handlers.go` — `HandleAuthorize` calls `RenderConsentPage` when no prior consent record exists.

## Used by
- End users in the browser during authorize-code flow.

## Notes
- CSP `default-src 'self'; style-src 'unsafe-inline'` — inline styles only, no remote scripts.
- `X-Frame-Options: DENY` blocks clickjacking on the consent page.
- Templates live in `internal/oauth/consent_templates/` (intentionally skipped per task scope).
- Errors silently dropped (`//nolint:errcheck`) because response is already committed.
