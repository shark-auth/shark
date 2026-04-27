# SharkAuth API — Overview

SharkAuth exposes a unified authentication platform API combining OAuth 2.1 authorization server endpoints (DCR RFC 7591/7592, authorization code + PKCE, client credentials, token exchange RFC 8693, device authorization RFC 8628), DPoP-bound token flows (RFC 9449), and a full admin REST surface covering users, agents, applications, audit logs, webhooks, vault, auth flows, SSO, RBAC, branding, and system health. All OAuth protocol endpoints follow RFC 6749/9449 semantics and are compatible with standard OAuth 2.1 client libraries; the admin REST surface uses an admin API key and is purpose-built for dashboard and automation tooling.

## Authentication Mechanisms

| Mechanism | Header | Used For |
|---|---|---|
| Admin API Key | `Authorization: Bearer sk_live_<random>` | All `/api/v1/*` admin endpoints |
| DPoP-bound access token | `Authorization: DPoP <token>` + `DPoP: <proof-jwt>` | Agent endpoints requiring RFC 9449 binding |
| OAuth 2.1 Bearer token | `Authorization: Bearer <access-token>` | OAuth resource endpoints (UserInfo, Vault token retrieval) |
| Session cookie | `shark_session=<id>` (HttpOnly) | User-facing auth flows, magic links, passkeys |

The admin API key is issued at first boot and stored at `data/admin.key.firstboot`. It can also be retrieved via `GET /api/v1/admin/firstboot/key` before the first admin account is bootstrapped (returns 404 afterward).

## Base URL Conventions

| Prefix | Surface |
|---|---|
| `/api/v1/*` | Admin REST API (users, agents, apps, audit, webhooks, vault, flows, system) |
| `/oauth/*` | OAuth 2.1 flows (token, authorize, DCR, introspect, revoke, device) |
| `/.well-known/*` | Discovery — authorization server metadata + JWKS |
| `/api/v1/auth/*` | User-facing auth flows (signup, login, MFA, passkeys, magic links, SSO) |
| `/api/v1/organizations/*` | Organization management (session-cookie auth) |
| `/admin/*` | Admin dashboard SPA (embedded React bundle) |
| `/api/docs` | Interactive Scalar UI (OpenAPI 3.1) |

## Error Format

Two error envelopes coexist by surface:

**OAuth surface** (`/oauth/*`) — RFC 6749 §5.2:
```json
{
  "error": "invalid_client",
  "error_description": "Client authentication failed.",
  "error_uri": "https://tools.ietf.org/html/rfc6749#section-5.2"
}
```

**Admin + Platform surface** (`/api/v1/*`) — extended envelope:
```json
{
  "error": "invalid_request",
  "code": "password_too_short",
  "message": "Password must be at least 12 characters.",
  "docs_url": "https://docs.shark-auth.com/errors/password_too_short",
  "details": { "min_length": 12 }
}
```

Common error codes: `invalid_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `internal_error`, `invalid_credentials`, `account_locked`, `mfa_required`, `session_expired`, `invalid_token`.

## Rate Limiting & Body Limits

- **Rate limit**: 100 requests/second burst, 100-token bucket (global, applied via token bucket middleware). Exceeded requests receive `429 Too Many Requests` with `{"error":"rate_limited"}`.
- **Request body limit**: 1 MB. Requests exceeding this size receive `413 Request Entity Too Large`.
- **Magic link rate limit**: 1 send per 60-second window per email address.

## Interactive API Reference

A live Scalar UI is available at **`/api/docs`** (no auth required). The raw OpenAPI 3.1 spec is served at `/api/openapi.yaml` and `/api/docs/openapi.yaml`.
