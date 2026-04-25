# providers.go

**Path:** `internal/vault/providers.go`  
**Package:** `vault`  
**LOC:** 221  
**Tests:** `providers_test.go`

## Purpose
Pre-configured OAuth 2.0 provider templates. Static registry of well-known integrations (Google, GitHub, Slack, Microsoft, Notion, Linear, Jira) with auth/token endpoints and default scopes.

## Key types / functions
- `ProviderTemplate` (struct, line 14) — Name, DisplayName, AuthURL, TokenURL, DefaultScopes, IconURL
- `Templates()` (func, line 143) — returns shallow copy of builtin template registry
- `Template(name)` (func) — lookup single template by Name
- `ListTemplates()` (func) — returns sorted list of available templates
- `ApplyTemplate(tmpl, clientID, clientSecret)` (func) — convert template → storage.VaultProvider

## Imports of note
- `sort` — template list ordering

## Builtin templates
- google_calendar — OpenID, calendar scope
- google_drive — OpenID, drive scope
- google_gmail — OpenID, gmail read+send
- slack — chat:write, channels:read, users:read
- github — repo, read:user, user:email
- microsoft — offline_access, User.Read, Mail.Read
- notion — (no scopes; Notion uses UI-based permissions)
- linear — read, write (requires prompt=consent; see TODO)
- jira — read:jira-work, write:jira-work, offline_access (requires audience=api.atlassian.com; see TODO)

## Wired by
- Admin API lists/queries templates for user selection during provider setup
- Manager.CreateProvider applies template + supplies client credentials

## Notes
- builtinTemplates map internal; access via public funcs to enable future storage migration
- Templates are read-only; never carry credentials
- Some providers (Linear, Jira) require extra auth-URL query params (see TODO comments)
- Handler layer adds those params via oauth2.AuthCodeOption at authorize-URL build time
- IconURL: Favicon or official branding icon (e.g. Slack edge CDN, GitHub favicons)
- Scopes sourced from provider; merged with user-requested scopes at authorize time

