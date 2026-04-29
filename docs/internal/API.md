# SharkAuth API Reference

Base URL: `/api/v1`

## Authentication Methods

| Level | Mechanism | Used For |
|-------|-----------|----------|
| Public | None | Signup, login, OAuth, magic links, password reset, SSO public endpoints |
| Session | `shark_session` cookie | `/me`, passkey registration, MFA enrollment |
| Session + MFA | Session with `mfa_passed=true` | `/me`, password change, MFA management |
| Admin | `Authorization: Bearer sk_live_*` (admin scope) | RBAC, user management, API keys, audit logs |
| Bearer | `Authorization: Bearer sk_live_*` | Machine-to-machine authentication |

All responses return JSON. Errors use `{error, message}` format.

---

## Interactive API Docs

| Path | Description |
|------|-------------|
| `GET /api/docs` | Scalar UI — interactive explorer for all ~200 endpoints |

The bundled OpenAPI spec at `documentation/api_reference/openapi.yaml` (v0.9.0) is served at `/api/docs` via the Scalar UI. Run `npx @scalar/cli serve documentation/api_reference/openapi.yaml` to explore locally.

---

## Health

| Method | Path | Auth | Response |
|--------|------|------|----------|
| GET | `/healthz` | Public | `{status: "ok"}` |

---

## Auth - Basic

### POST `/auth/signup`

Create a new user account.

**Request:**
```json
{ "email": "user@example.com", "password": "securepass", "name": "Jane Doe" }
```

**Response (201):**
```json
{ "id": "uuid", "email": "user@example.com", "name": "Jane Doe", "emailVerified": false, "mfaEnabled": false, "createdAt": "...", "updatedAt": "..." }
```

### POST `/auth/login`

Authenticate with email and password. Sets a `shark_session` cookie on success.

**Request:**
```json
{ "email": "user@example.com", "password": "securepass" }
```

**Response (200):** User object, or `{mfaRequired: true}` if MFA is enabled (partial session created).

### POST `/auth/logout`

**Auth:** Session cookie

Invalidates the current session.

**Response (200):** `{}`

### GET `/auth/me`

**Auth:** Session + MFA

Returns the authenticated user.

**Response (200):** User object.

---

## Auth - Password Management

### Forgot Password Flow

The password reset flow involves three steps:

1. **Your app** calls `POST /auth/password/send-reset-link` with the user's email
2. **SharkAuth** sends an email with a link to your frontend's reset page (configured via `password_reset.redirect_url`), e.g. `https://yourapp.com/auth/reset-password?token=abc123`
3. **Your frontend** reads the `token` query parameter, shows a "new password" form, and submits it to `POST /auth/password/reset`

#### Configuration

In `sharkauth.yaml`:
```yaml
password_reset:
  redirect_url: "https://yourapp.com/auth/reset-password"
```

Or via environment variable: `SHARKAUTH_PASSWORD_RESET__REDIRECT_URL`

The email link will be: `{redirect_url}?token={token}`

### POST `/auth/password/send-reset-link`

Send a password reset email. Always returns 200 regardless of whether the email exists (to prevent user enumeration).

**Request:**
```json
{ "email": "user@example.com" }
```

**Response (200):**
```json
{ "message": "If an account with that email exists, a password reset link has been sent" }
```

**Notes:**
- The reset token expires after **15 minutes**
- The email link points to your `password_reset.redirect_url` with a `?token=` query parameter

### POST `/auth/password/reset`

Reset password using a token from the reset email. This is the endpoint your frontend's reset page should POST to.

**Request:**
```json
{ "token": "reset-token-from-query-param", "password": "newpassword" }
```

**Response (200):**
```json
{ "message": "Password has been reset successfully" }
```

**Errors:**
| Status | Error | Cause |
|--------|-------|-------|
| 400 | `invalid_request` | Missing or malformed JSON body |
| 400 | `invalid_token` | Token is invalid, expired, or already used |
| 400 | `weak_password` | Password does not meet minimum length requirement |

### POST `/auth/password/change`

**Auth:** Session + MFA

Change password for the authenticated user (requires knowing the current password).

**Request:**
```json
{ "current_password": "oldpass", "new_password": "newpass" }
```

**Response (200):** `{message: "Password changed."}`

---

## Auth - OAuth

Supported providers: Google, GitHub, Apple, Discord (if configured).

### GET `/auth/oauth/{provider}`

Redirects to the OAuth provider's authorization page.

### GET `/auth/oauth/{provider}/callback`

**Query params:** `code`, `state`

Handles the OAuth callback. Returns the user object and sets a session cookie.

---

## Auth - Magic Links

### POST `/auth/magic-link/send`

Send a magic link email. Rate limited to 1 per email per 60 seconds.

**Request:**
```json
{ "email": "user@example.com" }
```

**Response (200):** `{message: "..."}`

### GET `/auth/magic-link/verify`

**Query params:** `token`

Verifies the magic link token. Returns the user object or redirects.

---

## Auth - Passkeys

### POST `/auth/passkey/register/begin`

**Auth:** Session

Begin passkey registration.

**Response (200):**
```json
{ "publicKey": { ... }, "challengeKey": "..." }
```

### POST `/auth/passkey/register/finish`

**Auth:** Session  
**Headers:** `X-Challenge-Key`

Complete passkey registration.

**Response (200):**
```json
{ "credential_id": "...", "name": "..." }
```

### POST `/auth/passkey/login/begin`

Begin passkey authentication.

**Request:**
```json
{ "email": "user@example.com" }
```
`email` is optional.

**Response (200):**
```json
{ "publicKey": { ... }, "challengeKey": "..." }
```

### POST `/auth/passkey/login/finish`

**Headers:** `X-Challenge-Key`

Complete passkey authentication. Returns the user object and sets a session cookie.

### GET `/auth/passkey/credentials`

**Auth:** Session

List passkey credentials for the authenticated user.

**Response (200):**
```json
{ "credentials": [{ "id": "...", "name": "...", "transports": [...], "backed_up": true, "created_at": "...", "last_used_at": "..." }] }
```

### DELETE `/auth/passkey/credentials/{id}`

**Auth:** Session

Delete a passkey credential.

### PATCH `/auth/passkey/credentials/{id}`

**Auth:** Session

Rename a passkey credential.

**Request:**
```json
{ "name": "My Yubikey" }
```

**Response (200):** Updated credential object.

---

## Auth - MFA (TOTP)

### Enrollment

#### POST `/auth/mfa/enroll`

**Auth:** Session + MFA

Begin MFA enrollment. Returns a TOTP secret and QR URI.

**Response (200):**
```json
{ "secret": "BASE32SECRET", "qr_uri": "otpauth://totp/..." }
```

#### POST `/auth/mfa/verify`

**Auth:** Session + MFA

Verify TOTP code to complete enrollment.

**Request:**
```json
{ "code": "123456" }
```

**Response (200):**
```json
{ "mfa_enabled": true, "recovery_codes": ["code1", "code2", "..."] }
```

### Authentication

After login with MFA enabled, the session is in a partial state (`mfa_passed=false`). Use one of these to upgrade:

#### POST `/auth/mfa/challenge`

**Auth:** Session (any MFA state)

**Request:**
```json
{ "code": "123456" }
```

**Response (200):** User object (session upgraded to `mfa_passed=true`).

#### POST `/auth/mfa/recovery`

**Auth:** Session (any MFA state)

**Request:**
```json
{ "code": "recovery-code" }
```

**Response (200):** User object (session upgraded to `mfa_passed=true`).

### Management

#### DELETE `/auth/mfa`

**Auth:** Session + MFA

Disable MFA.

**Request:**
```json
{ "code": "123456" }
```

**Response (200):** `{mfa_enabled: false}`

#### GET `/auth/mfa/recovery-codes`

**Auth:** Session + MFA

List recovery codes.

**Response (200):**
```json
{ "recovery_codes": ["code1", "code2", "..."] }
```

---

## Auth - SSO

### GET `/auth/sso`

**Query params:** `email`

Auto-route to the appropriate SSO connection based on email domain.

**Response (200):**
```json
{ "connection_id": "...", "connection_type": "oidc", "redirect_url": "..." }
```

### SAML (Public)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/sso/saml/{connection_id}/metadata` | SP metadata (XML) |
| POST | `/sso/saml/{connection_id}/acs` | Assertion Consumer Service. Returns `{user, session}` |

### OIDC (Public)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/sso/oidc/{connection_id}/auth` | Redirects to IdP authorization |
| GET | `/sso/oidc/{connection_id}/callback` | Handles callback. Returns `{user, session}` |

---

## SSO Connections (Admin)

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required for all endpoints.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/sso/connections` | Create connection (`type`: `oidc` or `saml`) |
| GET | `/sso/connections` | List all connections |
| GET | `/sso/connections/{id}` | Get connection |
| PUT | `/sso/connections/{id}` | Update connection |
| DELETE | `/sso/connections/{id}` | Delete connection |

---

## Roles (Admin)

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/roles` | Create role (`name`, `description`) |
| GET | `/roles` | List all roles |
| GET | `/roles/{id}` | Get role with permissions |
| PUT | `/roles/{id}` | Update role |
| DELETE | `/roles/{id}` | Delete role |

**Role object:** `{id, name, description, permissions, created_at, updated_at}`

---

## Permissions (Admin)

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/permissions` | Create permission (`action`, `resource`) |
| GET | `/permissions` | List all permissions |

**Permission object:** `{id, action, resource, created_at}`

---

## Role-Permission Mapping (Admin)

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/roles/{id}/permissions` | Attach permission (`permission_id`) |
| DELETE | `/roles/{id}/permissions/{pid}` | Detach permission |

---

## Auth Check (Admin)

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

### POST `/auth/check`

Check if a user has a specific permission.

**Request:**
```json
{ "user_id": "uuid", "action": "read", "resource": "documents" }
```

**Response (200):**
```json
{ "allowed": true }
```

---

## User Management (Admin)

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/users` | List users (query: `email`, `limit`, `offset`) |
| GET | `/users/{id}` | Get user |
| PATCH | `/users/{id}` | Update user (`email`, `name`, `email_verified`, `metadata`) |
| DELETE | `/users/{id}` | Delete user (cascades sessions, roles) |
| POST | `/admin/users` | Create user with optional pre-verified email |
| PATCH | `/admin/users/{id}/tier` | Set billing tier (`free` \| `pro`) |
| POST | `/users/{id}/roles` | Assign role (`role_id`) |
| DELETE | `/users/{id}/roles/{rid}` | Remove role |
| GET | `/users/{id}/roles` | List user's roles |
| GET | `/users/{id}/permissions` | List user's effective permissions |
| GET | `/users/{id}/audit-logs` | User's audit logs (`limit`, `cursor` query params) |

---

## API Keys (Admin)

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

### POST `/api-keys`

Create a new API key. The full key is shown **only once** in the response.

**Request:**
```json
{ "name": "My Service", "scopes": ["read:users", "write:users"], "rate_limit": 1000, "expires_at": "2025-12-31T23:59:59Z" }
```

`rate_limit` and `expires_at` are optional.

**Response (201):**
```json
{ "id": "uuid", "name": "My Service", "key": "sk_live_abc123...", "key_prefix": "sk_live_abc", "scopes": ["read:users", "write:users"], "rate_limit": 1000, "expires_at": "...", "created_at": "..." }
```

### Other API Key Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api-keys` | List all keys (prefix only, no full key) |
| GET | `/api-keys/{id}` | Get key details (no full key) |
| PATCH | `/api-keys/{id}` | Update key (`name`, `scopes`, `rate_limit`, `expires_at`) |
| DELETE | `/api-keys/{id}` | Revoke key |
| POST | `/api-keys/{id}/rotate` | Rotate key (revokes old, returns new full key) |

---

## Audit Logs (Admin)

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

### GET `/audit-logs`

List audit logs with cursor-based pagination (max 200 per page).

**Query params:** `limit`, `cursor`, `action`, `actor_id`, `target_id`, `status`, `ip`, `from` (RFC3339), `to` (RFC3339)

**Response (200):**
```json
{ "data": [{ "id": "...", "action": "...", "actor_id": "...", "target_id": "...", "status": "...", "ip": "...", "created_at": "...", "metadata": {} }], "next_cursor": "...", "has_more": true }
```

### GET `/audit-logs/{id}`

Get a single audit log entry.

### POST `/audit-logs/export`

Export audit logs as a JSON attachment.

**Request:**
```json
{ "from": "2025-01-01T00:00:00Z", "to": "2025-03-01T00:00:00Z", "action": "login" }
```

All fields are optional.

---

## Sessions - Self-Service

**Auth:** Session cookie.

### GET `/auth/sessions`

List the caller's own active and expired sessions. The session backing the current request is flagged with `current: true`.

**Response (200):**
```json
{
  "data": [
    {
      "id": "sess_abc",
      "user_id": "usr_123",
      "ip": "203.0.113.10",
      "user_agent": "Mozilla/5.0",
      "mfa_passed": true,
      "auth_method": "password",
      "expires_at": "2026-05-16T00:00:00Z",
      "created_at": "2026-04-16T00:00:00Z",
      "current": true
    }
  ]
}
```

### DELETE `/auth/sessions/{id}`

Revoke one of the caller's sessions. Foreign session IDs return `404` (no existence oracle). Revoking the current session also clears the cookie on the next request.

**Response (200):** `{ "message": "Session revoked" }`

---

## Admin - Stats

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

### GET `/admin/stats`

Overview counts. Always bounded, indexed — safe to poll.

**Response (200):**
```json
{
  "users": { "total": 12, "created_last_7d": 3 },
  "sessions": { "active": 5 },
  "mfa": { "enabled": 4, "adoption_pct": 33.33 },
  "failed_logins_24h": 1,
  "api_keys": { "active": 2, "expiring_7d": 0 },
  "sso_connections": { "total": 1, "enabled": 1 }
}
```

### GET `/admin/stats/trends`

Heavier charts data. Split from `/admin/stats` so the overview stays instant.

**Query params:** `days` (default 30, max 90).

**Response (200):**
```json
{
  "days": 30,
  "signups_by_day": [
    { "date": "2026-03-18", "count": 0 },
    { "date": "2026-03-19", "count": 1 }
  ],
  "auth_methods": [
    { "auth_method": "password", "count": 40 },
    { "auth_method": "google", "count": 15 }
  ]
}
```

`signups_by_day` is zero-filled across the requested window so the frontend chart can plot without gap logic.

---

## Admin - Sessions

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

### GET `/admin/sessions`

List all active sessions with the user email joined. Uses keyset cursor pagination on `(created_at DESC, id DESC)` for stable iteration under concurrent writes.

**Query params:**
- `limit` — page size (default 50, max 200)
- `cursor` — opaque string; pass the previous response's `next_cursor` verbatim
- `user_id` — filter by user
- `auth_method` — filter by method (`password`, `google`, `github`, `passkey`, `magic_link`, `sso`, ...)
- `mfa_passed` — `true` / `false`

**Response (200):**
```json
{
  "data": [
    {
      "id": "sess_abc",
      "user_id": "usr_123",
      "user_email": "alice@example.com",
      "ip": "203.0.113.10",
      "user_agent": "Mozilla/5.0",
      "mfa_passed": true,
      "auth_method": "password",
      "expires_at": "...",
      "created_at": "..."
    }
  ],
  "next_cursor": "2026-04-16T00:00:00Z|sess_abc"
}
```

`next_cursor` is omitted when the page is not full (no more rows). To iterate the full set, follow cursors until it's absent.

### DELETE `/admin/sessions/{id}`

Revoke any session. Emits a `session.revoke` audit entry with `actor_type=admin`.

### GET `/users/{id}/sessions`

List a single user's sessions (admin scope).

### DELETE `/users/{id}/sessions`

Revoke every session for the given user. Returns the number revoked. Emits one granular `session.revoke` audit entry per deleted session so compliance review can see exactly which device tokens were invalidated.

**Response (200):** `{ "message": "Sessions revoked", "count": 3 }`

---

## Admin - Dev Inbox

**Available only when the server is started with `shark serve --dev`.** These routes are unmounted entirely in production.

**Auth:** `Authorization: Bearer sk_live_*` (admin scope) required.

### GET `/admin/dev/emails`

List captured outbound emails.

**Query params:** `limit` (default 100, max 500).

**Response (200):**
```json
{
  "data": [
    {
      "id": "de_...",
      "to": "user@example.com",
      "subject": "Your magic link",
      "html": "<html>...</html>",
      "text": "...",
      "created_at": "..."
    }
  ]
}
```

### GET `/admin/dev/emails/{id}`

Full payload of a single captured email.

### DELETE `/admin/dev/emails`

Clear the dev inbox. Returns `204 No Content`.

---

## Organizations

B2B multi-tenancy. **Auth:** session cookie. Per-handler role gates (`owner` / `admin` / `member`).

### POST `/organizations`

Create an organization. The caller is enrolled as `owner`.

**Request:**
```json
{ "name": "Acme Corp", "slug": "acme", "metadata": "{\"plan\":\"free\"}" }
```

Slug: 3–64 chars, lowercase `a-z0-9-`, no leading/trailing hyphen.

**Response (201):** organization object.

### GET `/organizations`

List orgs the caller belongs to.

### GET `/organizations/{id}`

Get one org. Non-members receive `404` (no existence oracle).

### PATCH `/organizations/{id}`

Admin+. Update `name` and/or `metadata`. Slug is immutable.

### DELETE `/organizations/{id}`

Owner only. Cascade-deletes members + invitations.

### GET `/organizations/{id}/members`

Member list with joined user email/name.

### PATCH `/organizations/{id}/members/{uid}`

Admin+. Change member role. Demoting the last owner returns `409 last_owner`.

```json
{ "role": "admin" }
```

### DELETE `/organizations/{id}/members/{uid}`

Admin+. Same last-owner guard applies.

### POST `/organizations/{id}/invitations`

Admin+. Sends an email containing a one-shot accept link. The token is stored as SHA-256 hash only — DB leaks cannot produce valid links.

```json
{ "email": "new@example.com", "role": "member" }
```

**Response (201):** `{ id, email, role, expires_at, created_at }`. Expires in 72h.

### POST `/organizations/invitations/{token}/accept`

Requires a session whose email **exactly matches** the invitation. Idempotent. Consumes the token — second accept returns `409 invitation_used`.

---

## Webhooks

Outbound event delivery. HMAC-SHA256 signed, durable, auto-retry.

**Auth:** admin Bearer key.

### Event catalog (phase 2)

| Event | Emitted when | Payload |
|-------|--------------|---------|
| `user.created` | POST /auth/signup | safe user object (no password hash, no MFA secret) |
| `user.deleted` | DELETE /users/{id} or /auth/me | `{ id }` |
| `session.revoked` | self or admin session revoke (granular per session on bulk) | `{ session_id, user_id, revoked_by }` |
| `organization.created` | POST /organizations | `{ id, name, slug, created_by }` |
| `organization.member_added` | Invitation accepted | `{ organization_id, user_id, role, via }` |

### POST `/webhooks`

```json
{
  "url": "https://your-app.example.com/shark/webhook",
  "events": ["user.created", "session.revoked"],
  "description": "production"
}
```

**Response (201):** webhook object **plus** the signing secret (prefix `whsec_`). Secret is returned **once**. Store it; `GET` never includes it again.

### GET `/webhooks` / `GET /webhooks/{id}` / `PATCH /webhooks/{id}` / `DELETE /webhooks/{id}`

Standard CRUD. `PATCH` accepts any subset of `url`, `events`, `enabled`, `description`.

### POST `/webhooks/{id}/test`

Queues a synthetic `webhook.test` delivery. Returns `{ delivery_id }`. Use this to validate network reach + signing before enabling a real integration.

### GET `/webhooks/{id}/deliveries`

Delivery log, keyset cursor pagination.

**Query params:** `limit` (default 50, max 200), `cursor`.

### Delivery contract

Every delivery POSTs a JSON envelope:

```json
{ "event": "user.created", "created_at": "2026-04-16T12:00:00Z", "data": { /* event payload */ } }
```

Headers:
- `X-Shark-Event: user.created`
- `X-Shark-Delivery: whd_...`
- `X-Shark-Signature: t=<unix_ts>,v1=<hex>`

Verify signature:
```
hmac_sha256(secret, fmt.Sprintf("%d.%s", t, rawBody)) == v1
```

### Retry policy

5 attempts, backoff `1m, 5m, 30m, 2h, 12h` (~14h40m). 2xx = delivered. Past budget → `status=failed`. Retention: 90d (configurable).

---

## Rate Limiting

- **Global:** 100 requests/second with burst tolerance
- **Magic links:** 1 per email per 60 seconds
- **API keys:** Configurable per-key rate limit

## Session Management

- Cookie-based (`shark_session`)
- Validates user, IP, and User-Agent
- MFA sessions use a two-step upgrade flow (partial → full)

## Password Hashing

- Argon2id (with automatic migration from bcrypt)
