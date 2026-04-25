# sso.tsx

**Path:** `admin/src/components/sso.tsx`
**Type:** React component (page)
**LOC:** ~500

## Purpose
SSO (SAML 2.0 / OpenID Connect) configuration—provider management, org-scoped SSO setup, JIT provisioning, attribute mapping.

## Exports
- `SSO()` (default) — function component

## Features
- **Provider list** — SAML|OIDC SSO connections
- **Create flow** — register new provider (metadata upload, config form)
- **Detail pane** — ACS URL, entity ID, attribute mapping, JIT settings
- **Org-scoped SSO** — enable/disable per organization
- **Attribute mapping** — map SAML attributes to SharkAuth user fields
- **JIT provisioning** — auto-create users on first SSO login

## Hooks used
- `useAPI('/admin/sso-providers')` — list providers
- `useAPI('/admin/sso-providers/{id}')` — provider details

## State
- `selected` — current provider
- `creatingSSO` — create modal
- `filter` — by status (active|inactive|all)

## API calls
- `GET /api/v1/admin/sso-providers` — list
- `POST /api/v1/admin/sso-providers` — create (metadata or manual config)
- `PATCH /api/v1/admin/sso-providers/{id}` — update settings
- `DELETE /api/v1/admin/sso-providers/{id}` — delete

## Composed by
- App.tsx

## Notes
- Supports both SAML 2.0 and OpenID Connect
- JIT provisioning reduces admin overhead
- Attribute mapping allows flexible user field synchronization
