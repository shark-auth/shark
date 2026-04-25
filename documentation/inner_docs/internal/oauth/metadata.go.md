# metadata.go

**Path:** `internal/oauth/metadata.go`
**Package:** `oauth`
**LOC:** 81
**Tests:** `metadata_test.go`

## Purpose
RFC 8414 Authorization Server Metadata document — the MCP discovery entrypoint at `/.well-known/oauth-authorization-server`.

## RFCs implemented
- RFC 8414 OAuth 2.0 Authorization Server Metadata
- Advertises support for: RFC 6749, RFC 7591 (DCR), RFC 7009 (revoke), RFC 7662 (introspect), RFC 8628 (device), RFC 8693 (token-exchange), RFC 9449 (DPoP), PKCE S256.

## Key types / functions
- `authServerMetadata` (struct, line 9) — full RFC 8414 payload including `dpop_signing_alg_values_supported` and `device_authorization_endpoint`.
- `MetadataHandler` (func, line 36) — closure-pattern: marshals payload once at startup, serves the byte slice on every request (zero per-request alloc). Panics at startup if marshal fails.

## Imports of note
- `encoding/json`, `net/http` only — no fosite dependency.

## Wired by
- `internal/server/server.go` — mounts at `GET /.well-known/oauth-authorization-server`.

## Used by
- MCP clients (Claude Desktop), AppAuth, dynamic OAuth discovery libs.

## Notes
- `Cache-Control: public, max-age=3600` — clients SHOULD cache for an hour.
- Algorithm advertisement (`ES256`, `RS256`) must match what server.go actually signs with.
- Endpoint URLs derived from the `issuer` arg — single source of truth.
- `service_documentation` hard-coded to `https://sharkauth.com/docs`.
