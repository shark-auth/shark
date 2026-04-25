# audience.go

**Path:** `internal/oauth/audience.go`
**Package:** `oauth`
**LOC:** 52
**Tests:** `resource_test.go`

## Purpose
RFC 8707 Resource Indicators plumbing — threads the `resource` parameter through fosite's request lifecycle and exposes audience-validation for resource servers.

## RFCs implemented
- RFC 8707 Resource Indicators for OAuth 2.0

## Key types / functions
- `resourceContextKey` (type, line 9) — unexported context key. Comment explains the rationale: fosite's `Sanitize()` strips unrecognized form params before `CreateAccessTokenSession`, so `resource` must travel via context instead.
- `contextWithResource` (func, line 12) — context setter.
- `resourceFromContext` (func, line 18) — context getter; empty string when absent.
- `ValidateAudience` (func, line 31) — accepts `interface{}` aud claim (string OR `[]string` OR `[]interface{}` from JSON), checks membership against the expected resource. Empty expected ⇒ accept any.

## Imports of note
- `context` only — zero external deps.

## Wired by
- `handlers.go:87` — `HandleToken` stuffs the form's `resource` into ctx before calling fosite.
- `store.go` — auth-code persistence reads/writes `Resource` field directly from the form.

## Used by
- Resource servers verifying incoming JWTs (validates aud claim shape).

## Notes
- The polymorphic switch is essential: `encoding/json` decodes JWT aud as `[]interface{}`, but our own struct emits `string` or `[]string`.
- This is the cleanest way to round-trip `resource` through fosite without forking it.
