# agent_register.go

**Path:** `cmd/shark/cmd/agent_register.go`
**Package:** `cmd`
**LOC:** ~165
**Tests:** none direct

## Purpose
Implements `shark agent register` — registers a new agent identity.

**Default (DCR mode):** POSTs RFC 7591 client metadata to `POST /oauth/register`.
No authentication required (per RFC 7591 spec). Returns `client_id`,
`client_secret`, and `registration_access_token` (shown once).

**Admin mode (`--admin`):** POSTs to `POST /api/v1/agents` using the operator
key (`SHARK_ADMIN_TOKEN`). For use by platform operators only.

## Key types / functions
- `agentCmd` (var) — parent cobra command, wired to root.
- `agentRegisterCmd` (var) — child command, dispatches to `agentRegisterDCR`
  or `agentRegisterAdmin` based on the `--admin` flag.
- `agentRegisterDCR` — builds RFC 7591 metadata body, calls `POST /oauth/register`
  with no auth, prints `client_id`, `client_secret`, `registration_access_token`.
- `agentRegisterAdmin` — uses `adminDo` to call `POST /api/v1/agents` with
  operator key, prints `id`, `client_id`, `client_secret`.

## Flags
| Flag | Default | Description |
|---|---|---|
| `--name` | (required) | Human-readable client_name |
| `--description` | "" | Embedded into `client_name` for DCR; `description` field for admin |
| `--app` | "" | DCR: `software_id`; admin: `metadata.app_slug` |
| `--admin` | false | Use admin endpoint instead of DCR |
| `--json` | false | Emit raw JSON response |

## RFC 7591 body fields sent
- `client_name` (required)
- `grant_types`: `["client_credentials"]`
- `token_endpoint_auth_method`: `"client_secret_basic"`
- `software_id`: set when `--app` is provided

## Imports of note
- Uses `adminDo`, `adminClient`, `resolveAdminURL`, `apiError`, `extractData`,
  `maybeJSONErr`, `writeJSON`, `jsonFlag` from sibling files.

## Wired by / used by
- DCR endpoint: `internal/oauth/dcr.go:HandleDCRRegister` → `POST /oauth/register`
- Admin endpoint: `internal/api/agent_handlers.go:handleCreateAgent` → `POST /api/v1/agents`

## Notes
- The DCR endpoint is unauthenticated per RFC 7591 §3.1.
- The `registration_access_token` from DCR enables GET/PUT/DELETE on the
  client configuration via `/oauth/register/{client_id}`.
- Secret is shown only once in both modes.
