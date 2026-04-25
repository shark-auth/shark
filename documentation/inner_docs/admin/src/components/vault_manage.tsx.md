# vault_manage.tsx

**Path:** `admin/src/components/vault_manage.tsx`
**Type:** React component (page)
**LOC:** ~300

## Purpose
Vault (secrets) management—store encrypted secrets, API credentials, integration keys, reference in policies/flows.

## Exports
- `Vault()` (default) — function component

## Features
- **Secret list** — name, type (api_key|password|token), last accessed, created
- **Create secret** — name, value (encrypted at rest), metadata
- **Search** — by name or type
- **Audit** — who accessed secret, when, from where
- **Rotation** — update secret value without breaking references
- **Permissions** — grant agents/flows access to specific secrets

## Hooks used
- `useAPI('/admin/vault')` — list secrets
- `useToast()` — feedback

## State
- `selected` — current secret
- `creating` — create modal

## API calls
- `GET /api/v1/admin/vault` — list (values not returned)
- `POST /api/v1/admin/vault` — create secret
- `PATCH /api/v1/admin/vault/{id}` — update value
- `DELETE /api/v1/admin/vault/{id}` — delete
- `GET /api/v1/admin/vault/{id}/audit` — access log

## Composed by
- App.tsx

## Notes
- Secrets encrypted at rest, decrypted on read
- Values never logged or visible in audit trail
- Enterprise feature for secure credential storage
