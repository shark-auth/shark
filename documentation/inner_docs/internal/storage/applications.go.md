# applications.go

**Path:** `internal/storage/applications.go`
**Package:** `storage`
**LOC:** 406
**Tests:** `applications_test.go`

## Purpose
SQLite implementation of `Application` CRUD (registered OAuth 2.x / OIDC client applications). The `Application` type itself lives in `storage.go`; this file is the persistence-only sibling.

## Interface methods implemented
- `CreateApplication` (13) — JSON-encodes string-slice columns, NULL-safes optional strings, defaults `integration_mode="custom"` and `proxy_login_fallback="hosted"`
- `GetApplicationByID` (81), `GetApplicationByClientID` (91)
- `GetApplicationBySlug`, `GetApplicationByProxyDomain`, `GetDefaultApplication`
- `ListApplications` with limit/offset
- `UpdateApplication`, `RotateApplicationSecret` (writes new hash + prefix), `DeleteApplication`
- Internal scanner helpers

## Tables touched
- applications

## Columns
`id, name, slug, client_id, client_secret_hash, client_secret_prefix, allowed_callback_urls (JSON), allowed_logout_urls (JSON), allowed_origins (JSON), is_default, metadata (JSON), created_at, updated_at, integration_mode, branding_override (JSON), proxy_login_fallback, proxy_login_fallback_url, proxy_public_domain, proxy_protected_url`

## Imports of note
- `database/sql`, `encoding/json`, `time`

## Used by
- `internal/api/applications.go` admin handlers
- `internal/auth` — redirect URL allowlist checks
- `internal/oauth` — client lookup for hosted/custom apps
- `internal/proxy` — `GetApplicationByProxyDomain` for the transparent proxy router
- `internal/branding` — `branding_override` merge in `ResolveBranding`

## Notes
- Slug uniqueness + format are enforced at the API layer (regex `^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`).
- Default values for `integration_mode` and `proxy_login_fallback` ensure rows from older migrations stay valid.
- `branding_override` is a JSON blob (string subset of branding fields) merged in `branding.go::ResolveBranding`.
- Proxy fields (`proxy_public_domain`, `proxy_protected_url`) added in migration 00021 enable transparent reverse-proxy mode.
