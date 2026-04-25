# api_key.go + api_key_*.go

**Path:** `cmd/shark/cmd/api_key*.go`
**Package:** `cmd`

## Purpose
Implements `shark api-key` parent + create/list/rotate/revoke subcommands wrapping `/api/v1/api-keys`.

## Subcommands
- `shark api-key list` — GET /api/v1/api-keys
- `shark api-key create --name` — POST /api/v1/api-keys
- `shark api-key rotate <id>` — POST /api/v1/api-keys/{id}/rotate (secret shown once)
- `shark api-key revoke <id>` — DELETE /api/v1/api-keys/{id}
