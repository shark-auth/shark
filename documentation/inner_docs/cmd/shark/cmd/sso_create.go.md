# `cmd/shark/cmd/sso_create.go`

## Purpose

Implements `shark sso create` — creates a new SSO connection. `--name` and `--type` (oidc or saml) are required. An optional `--domain` enables email-domain-based auto-routing to this connection.

## Command shape

`shark sso create --name <name> --type <oidc|saml> [--domain <domain>] [--json]`

## Flags

- `--name` — connection display name (required)
- `--type` — protocol type: `oidc` or `saml` (required)
- `--domain` — email domain for automatic connection routing
- `--json` — emit raw JSON output

## API endpoint(s) called

- `POST /api/v1/sso/connections` — create the SSO connection

## Example

`shark sso create --name "Acme OIDC" --type oidc --domain acme.com`
