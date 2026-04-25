# saml.go

**Path:** `internal/sso/saml.go`
**Package:** `sso`
**LOC:** 234
**Tests:** `saml_test.go`

## Purpose
SAML Service Provider flow: generates SP metadata for IdP admin configuration, processes SAML assertions (ACS callback), extracts attributes, creates/links users, establishes sessions. Shark is SAML SP (NOT SAML IdP).

## Key types / functions
- `SSOManager.GenerateSPMetadata()` (line 20) — returns SAML SP metadata XML for IdP discovery
- `SSOManager.HandleSAMLACS()` (line 44) — processes POSTed SAML response, validates assertion, extracts attributes, finds/creates user, creates session
- `SSOManager.buildSAMLSP()` (line 132) — constructs crewjam/saml ServiceProvider from connection config
- `extractSAMLAttributes()` — parses assertion attributes map (sub, email, name, URI-qualified names)

## Imports of note
- `github.com/crewjam/saml` — SAML SP, assertion parsing, metadata generation
- `github.com/crewjam/saml/samlsp` — Middleware wrapper
- `crypto/x509`, `encoding/pem` — IdP certificate parsing

## Wired by
- `internal/api/sso_handlers.go` (POST /api/v1/sso/saml/{id}/acs endpoint)
- SSOManager.GetConnection (loads connection config from storage)

## Used by
- Web login flow: IdP POSTs SAML response to ACS URL, handler validates and creates session

## Notes
- SP EntityID defaults to BaseURL if not configured in connection (line 137).
- ACS URL auto-constructed from BaseURL + connectionID (line 143).
- Supports multiple IdP certificate/attribute name formats for compatibility (lines 86-96, 104-110).
- Subject extracted from assertion or NameID fallback (line 76).
- Email attribute required; falls back to common URI names (urn:oid:, http://schemas.xmlsoap.org) (line 84).
- IdP certificate parsed if provided; enables signature validation (line 171).
- Session created with auth method "sso" (line 123).
