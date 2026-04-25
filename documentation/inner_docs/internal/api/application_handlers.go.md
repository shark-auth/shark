# application_handlers.go

**Path:** `internal/api/application_handlers.go`
**Package:** `api`
**LOC:** 581
**Tests:** likely integration-tested

## Purpose
CRUD for registered "applications" — first-party tenant apps (not OAuth clients/agents). Each app owns its slug, allowed callback/logout/origin URLs, integration mode (hosted/components/proxy/custom), branding override JSON, and proxy fallback config. Secrets returned once on create + rotate.

## Handlers exposed
- `handleCreateApp` (line 213) — POST `/admin/apps`. Generates `client_id` + base62 `client_secret`. Validates URLs, integration_mode, slug. Slug auto-derived if missing.
- `handleListApps` (line 341) — GET.
- `handleGetApp` (line 369) — GET `/{id}` (via `getAppByIDOrClientID`).
- `handleUpdateApp` (line 384) — PATCH. Field-by-field; rejects bad slug/URLs/mode.
- `handleDeleteApp` (line 519) — DELETE.
- `handleRotateAppSecret` (line 545) — POST `/{id}/rotate-secret`. Returns fresh `client_secret` once.

## Key types
- `applicationResponse` (line 24) — full wire shape (no hash).
- `applicationResponseWithSecret` (line 47) — adds `client_secret`.

## Helpers
- `appToResponse` (line 52)
- `validIntegrationMode` (line 80) — enum: `hosted|components|proxy|custom`.
- `validProxyLoginFallback` (line 89) — enum: `hosted|custom_url`.
- `validateAppURL`/`validateAppURLs` (lines 98, 121) — http/https only; rejects `javascript:`, `file:`, `data:`, `vbscript:`.
- `generateAppSecret` (line 131) — base62-encoded 32-byte random + SHA-256 hash.
- `apiBase62Encode`/`apiIsZero`/`apiDivmod` (lines 148, 165, 174) — local base62.
- `getAppByIDOrClientID` (line 185), `auditApp` (line 197).

## Imports of note
- `internal/storage` — Application + AuditLog
- `crypto/rand`, `crypto/sha256`

## Wired by
- `internal/api/router.go:524-529`

## Notes
- ClientSecretPrefix exposes the first 8 chars of the secret (for display).
- Integration mode controls whether `/hosted/{slug}` or `/paywall/{slug}` shells render.
