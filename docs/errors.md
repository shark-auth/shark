# Error Code Catalog (W18)

Two envelopes. Pick the right one by URL prefix.

## Auth + Admin envelope (`/auth/**`, `/admin/**`, everything non-OAuth)

```json
{
  "error": "password_too_short",
  "message": "Password must be at least 12 characters",
  "code": "password_too_short",
  "docs_url": "https://docs.shark-auth.com/errors/password_too_short",
  "details": { "min_length": 12 }
}
```

- `error` and `code` are both machine-readable slugs. They are usually identical; they diverge only when a legacy top-level class (`weak_password`) fans out into refined codes (`password_too_short`, `password_too_common`, `password_in_breach`). Integrators should switch on `code`.
- `message` is safe to show end-users verbatim.
- `docs_url` points to the per-code doc page (stub today — site build is separate).
- `details` is optional structured context: `min_length`, `retry_after`, `required_scope`, etc.

Helpers: see [`internal/api/errors.go`](../internal/api/errors.go).

## OAuth envelope (`/oauth/**`, RFC 6749 §5.2)

```json
{
  "error": "invalid_grant",
  "error_description": "refresh token has been revoked",
  "error_uri": "https://docs.shark-auth.com/errors/invalid_grant"
}
```

Strictly RFC-compliant. No Shark extension fields — breaks AppAuth, x/oauth2, Authlib, etc. Response carries `Cache-Control: no-store` and `Pragma: no-cache` per §5.1. Extension hints go on `error_uri` or separate headers (e.g. `Location:`), never the body.

Helpers: see [`internal/oauth/errors.go`](../internal/oauth/errors.go).

## Code catalog

Every code links to `https://docs.shark-auth.com/errors/<code>` — all stubs until the public site ships.

### Generic (any non-OAuth endpoint)
| Code | HTTP | When |
|---|---|---|
| `invalid_request` | 400 | Malformed body, missing field, bad content-type. |
| `unauthorized` | 401 | Missing/invalid credentials, expired session, bad JWT. |
| `forbidden` | 403 | Authenticated but not permitted. |
| `not_found` | 404 | Resource does not exist (or caller lacks visibility). |
| `conflict` | 409 | Duplicate resource, version mismatch. |
| `rate_limited` | 429 | Per-IP/per-user quota exceeded; `details.retry_after`. |
| `internal_error` | 500 | Bug or unrecoverable upstream failure. |

### Auth / credentials
| Code | HTTP | When |
|---|---|---|
| `invalid_credentials` | 401 | Email/password mismatch, wrong TOTP. |
| `account_locked` | 423 | Too many failed attempts; `details.retry_after`. |
| `mfa_required` | 403 | Session needs an MFA step before proceeding. |
| `session_expired` | 401 | Token still parses but exp has passed. |
| `token_used` | 400 | Single-use magic-link / reset token already consumed. |
| `invalid_token` | 400 | Magic-link or reset token is malformed / unknown. |

### Password policy (split from legacy `weak_password`)
| Code | HTTP | When | details |
|---|---|---|---|
| `weak_password` | 400 | Generic fallback; prefer a refined code below. | — |
| `password_too_short` | 400 | Under minimum length. | `{min_length}` |
| `password_too_common` | 400 | In top-N most-common-passwords list. | — |
| `password_in_breach` | 400 | HIBP k-anon hit. | `{breach_count}` |

### MFA / passkey flows
| Code | HTTP | When |
|---|---|---|
| `enrollment_already_complete` | 409 | Tried to start a second enrollment for the same factor. |
| `challenge_expired` | 400 | Client took too long to answer the challenge. |
| `challenge_invalid` | 400 | Signature/verification failed. |
| `no_matching_credential` | 400 | Presented credential ID isn't registered for the user. |

### Email verification / magic link
| Code | HTTP | When |
|---|---|---|
| `email_already_verified` | 409 | Link used after the fact. |
| `magic_link_expired` | 400 | Past TTL. |

### Admin
| Code | HTTP | When |
|---|---|---|
| `bootstrap_locked` | 409 | Admin bootstrap already completed. |
| `invalid_scope` | 403 | API key missing a required scope. |

### OAuth 2.1 (`/oauth/**` ONLY — RFC-mandated)
RFC 6749 §5.2 plus OIDC, RFC 7662, RFC 8628, RFC 9449.

| Code | RFC |
|---|---|
| `invalid_request` | 6749 §5.2 |
| `invalid_client` | 6749 §5.2 |
| `invalid_grant` | 6749 §5.2 |
| `unauthorized_client` | 6749 §5.2 |
| `unsupported_grant_type` | 6749 §5.2 |
| `invalid_scope` | 6749 §5.2 |
| `access_denied` | 6749 §4.1.2.1 |
| `server_error` | 6749 §4.1.2.1 |
| `temporarily_unavailable` | 6749 §4.1.2.1 |
| `invalid_token` | 6750 §3.1 |
| `unsupported_token_type` | 7009 §2.2 |
| `invalid_dpop_proof` | 9449 §7 |
| `interaction_required` | OIDC Core §3.1.2.6 |
| `login_required` | OIDC Core §3.1.2.6 |
| `consent_required` | OIDC Core §3.1.2.6 |
| `authorization_pending` | 8628 §3.5 |
| `slow_down` | 8628 §3.5 |
| `expired_token` | 8628 §3.5 |
