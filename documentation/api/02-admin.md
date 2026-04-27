# SharkAuth API — Admin Endpoints

All admin endpoints require an admin API key passed as a Bearer token:

```
Authorization: Bearer sk_live_<random>
```

The key is written to `data/admin.key.firstboot` at first boot and is available via `GET /api/v1/admin/firstboot/key` until the admin account is bootstrapped. All responses use the extended error envelope `{error, code, message, docs_url, details}`.

---

## Health & System

### `GET /api/v1/admin/health`

Returns server health including DB status, JWT config, SMTP provider, uptime, active signing keys, SSO connection count, and applied migration version.

**Response `200`**:
```json
{
  "status": "ok",
  "version": "0.9.0",
  "uptime_seconds": 3600,
  "db": { "driver": "sqlite3", "size_mb": 2.4, "status": "ok" },
  "migrations": { "current": 42, "name": "add_proxy_rules" },
  "jwt": { "mode": "asymmetric", "algorithm": "ES256", "active_keys": 1 },
  "smtp": { "tier": "dev", "configured": true, "sent_today": 3, "daily_limit": 100 },
  "oauth_providers": ["google", "github"],
  "sso_connections": 0
}
```

### `GET /healthz` — Liveness probe (no auth)

```json
{ "status": "ok" }
```

Returns `503` with `{ "status": "unhealthy", "error": "database unreachable" }` when the DB ping fails.

### `GET /api/v1/admin/stats`

Aggregate dashboard statistics (user count, active sessions, agent count, etc.).

### `GET /api/v1/admin/stats/trends`

Time-series trends for the dashboard stats chart.

### `GET /api/v1/admin/config` / `PATCH /api/v1/admin/config`

Read or update the runtime configuration stored in the database. Replaces the deprecated YAML config surface (W17).

### `GET /api/v1/admin/system/mode`

Returns the current server mode (`dev` / `production`).

### `POST /api/v1/admin/system/swap-mode`

Swaps between `dev` and `production` mode. Triggers a drain + re-init cycle.

### `POST /api/v1/admin/system/reset`

Hard-resets all runtime state (sessions, tokens, audit logs). Requires explicit confirmation body. Useful in dev/test.

### `POST /api/v1/admin/auth/rotate-signing-key`

Rotates the JWT signing key. Old key is retired (kept in JWKS for 2× token TTL). Subsequent tokens are signed with the new key.

### `POST /api/v1/admin/auth/revoke-jti`

Revokes a specific JWT by JTI (JWT ID). Useful for immediate admin-forced logout of a single token.

**Request**:
```json
{ "jti": "01HZ2XKA..." }
```

---

## Bootstrap & First Boot

### `GET /api/v1/admin/firstboot/key`

Returns the admin API key from first boot. **Public** — returns `404` once the admin account has been bootstrapped. Intended for automated provisioning scripts.

**Response `200`**:
```json
{ "key": "sk_live_abc123..." }
```

### `POST /api/v1/admin/bootstrap/consume`

Consumes the one-time bootstrap token to set up the first admin account. The token in the request body is the credential — no `Authorization` header needed.

**Request**:
```json
{ "token": "<bootstrap-token>", "email": "admin@example.com", "password": "..." }
```

### `GET /api/v1/admin/bootstrap/status`

Returns whether the server has been bootstrapped (admin account created).

---

## Users

Base path: `/api/v1/users`  Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/users` | List all users (paginated) |
| `GET` | `/api/v1/users/{id}` | Get a single user |
| `PATCH` | `/api/v1/users/{id}` | Update user (name, email, metadata) |
| `DELETE` | `/api/v1/users/{id}` | Delete user + cascade |
| `POST` | `/api/v1/admin/users` | Create user (admin-side) |
| `POST` | `/api/v1/users/{id}/roles` | Assign a role |
| `DELETE` | `/api/v1/users/{id}/roles/{rid}` | Remove a role |
| `GET` | `/api/v1/users/{id}/roles` | List user roles |
| `GET` | `/api/v1/users/{id}/permissions` | List effective permissions |
| `GET` | `/api/v1/users/{id}/audit-logs` | User-scoped audit log |
| `GET` | `/api/v1/users/{id}/sessions` | List active sessions |
| `DELETE` | `/api/v1/users/{id}/sessions` | Revoke all sessions |
| `GET` | `/api/v1/users/{id}/oauth-accounts` | List linked social accounts |
| `DELETE` | `/api/v1/users/{id}/oauth-accounts/{oauthId}` | Unlink social account |
| `GET` | `/api/v1/users/{id}/passkeys` | List passkey credentials |
| `DELETE` | `/api/v1/users/{id}/mfa` | Force-disable MFA (support recovery) |
| `POST` | `/api/v1/users/{id}/verify/send` | Send email verification |
| `GET` | `/api/v1/users/{id}/agents` | List agents belonging to / authorized by user |
| `POST` | `/api/v1/users/{id}/revoke-agents` | Cascade revoke all agent tokens for user |

**Example — Get user**:
```json
{
  "id": "usr_01HZ2XKABCDEF",
  "email": "alice@example.com",
  "email_verified": true,
  "name": "Alice Example",
  "mfa_enabled": false,
  "created_at": "2024-04-01T10:00:00Z"
}
```

**Tier management** (proxy paywall gating):

`PATCH /api/v1/admin/users/{id}/tier` — Set user tier for proxy paywall rules.

```json
{ "tier": "pro" }
```

---

## Agents

Base path: `/api/v1/agents`  Auth: Admin API key

Agents are machine-to-machine OAuth clients registered via the admin API (distinct from DCR which is protocol-level).

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/agents` | Register a new agent |
| `GET` | `/api/v1/agents` | List agents |
| `GET` | `/api/v1/agents/{id}` | Get agent |
| `PATCH` | `/api/v1/agents/{id}` | Update agent metadata |
| `DELETE` | `/api/v1/agents/{id}` | Delete agent |
| `GET` | `/api/v1/agents/{id}/tokens` | List active tokens |
| `POST` | `/api/v1/agents/{id}/tokens/revoke-all` | Revoke all active tokens |
| `POST` | `/api/v1/agents/{id}/rotate-secret` | Rotate client secret |
| `POST` | `/api/v1/agents/{id}/rotate-dpop-key` | Rotate DPoP binding key |
| `GET` | `/api/v1/agents/{id}/audit` | Agent audit log |
| `GET` | `/api/v1/agents/{id}/policies` | Get agent policies |
| `POST` | `/api/v1/agents/{id}/policies` | Set agent policies |

**Example — Create agent request**:
```json
{
  "client_name": "my-billing-agent",
  "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
  "scope": "openid billing:read",
  "token_endpoint_auth_method": "client_secret_basic"
}
```

**Response `201`** (client_secret shown once):
```json
{
  "id": "agt_01HZ2XKABCDEF",
  "client_id": "shark_dcr_abc123",
  "client_secret": "cs_live_...",
  "client_name": "my-billing-agent",
  "grant_types": ["client_credentials"],
  "scope": "openid billing:read",
  "active": true,
  "dpop_bound": false,
  "created_at": "2024-04-01T10:00:00Z"
}
```

---

## Applications

Base path: `/api/v1/admin/apps`  Auth: Admin API key

Applications represent OAuth clients used by web/mobile frontends.

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/admin/apps` | Create application |
| `GET` | `/api/v1/admin/apps` | List applications |
| `GET` | `/api/v1/admin/apps/{id}` | Get application |
| `PATCH` | `/api/v1/admin/apps/{id}` | Update application |
| `DELETE` | `/api/v1/admin/apps/{id}` | Delete application |
| `POST` | `/api/v1/admin/apps/{id}/rotate-secret` | Rotate client secret |
| `GET` | `/api/v1/admin/apps/{id}/snippet` | Get integration code snippet |

---

## Audit Logs

Base path: `/api/v1/audit-logs`  Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/audit-logs` | List audit events (filterable by actor, action, date range) |
| `GET` | `/api/v1/audit-logs/{id}` | Get a single audit event |
| `POST` | `/api/v1/audit-logs/export` | Export logs as CSV |
| `POST` | `/api/v1/admin/audit-logs/purge` | Purge logs older than cutoff |

**Query parameters for list**:
- `actor_type` — filter by actor type (`user`, `admin`, `agent`)
- `action` — filter by action name (e.g. `user.created`)
- `from` / `to` — ISO 8601 date range
- `limit` / `cursor` — keyset pagination

**Example event**:
```json
{
  "id": "al_01HZ2XKABCDEF",
  "actor_type": "admin",
  "actor_id": "admin_key",
  "action": "user.deleted",
  "target_type": "user",
  "target_id": "usr_01HZ2XKABCDEF",
  "metadata": {},
  "created_at": "2024-04-01T10:05:00Z"
}
```

---

## Vault (Token Vault)

Base path: `/api/v1/vault`  Auth: Admin API key (providers), Session cookie (user flows), Bearer token (agent retrieval)

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/vault/providers` | Admin key | Register a new vault provider |
| `GET` | `/api/v1/vault/providers` | Admin key | List vault providers |
| `GET` | `/api/v1/vault/providers/{id}` | Admin key | Get provider |
| `PATCH` | `/api/v1/vault/providers/{id}` | Admin key | Update provider |
| `DELETE` | `/api/v1/vault/providers/{id}` | Admin key | Delete provider |
| `GET` | `/api/v1/vault/templates` | Admin key | List built-in provider templates |
| `GET` | `/api/v1/vault/connect/{provider}` | Session cookie | Start user OAuth connect flow |
| `GET` | `/api/v1/vault/callback/{provider}` | Session cookie | OAuth callback for connect flow |
| `GET` | `/api/v1/vault/connections` | Session cookie | List user's vault connections |
| `DELETE` | `/api/v1/vault/connections/{id}` | Session cookie | Disconnect a vault connection |
| `GET` | `/api/v1/vault/{provider}/token` | Bearer (agent) | Retrieve stored token for a provider |
| `GET` | `/api/v1/admin/vault/connections` | Admin key | List all vault connections (cross-user) |
| `DELETE` | `/api/v1/admin/vault/connections/{id}` | Admin key | Admin-delete a connection |

Client secrets are never returned after creation — rotation goes through the `PATCH` body field.

---

## Auth Flows (Flow Builder)

Base path: `/api/v1/admin/flows`  Auth: Admin API key

Auth flows are custom trigger-based pipelines that run at signup, login, password-reset, magic-link, and OAuth callback events.

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/admin/flows` | Create a flow |
| `GET` | `/api/v1/admin/flows` | List flows |
| `GET` | `/api/v1/admin/flows/{id}` | Get flow |
| `PATCH` | `/api/v1/admin/flows/{id}` | Update flow |
| `DELETE` | `/api/v1/admin/flows/{id}` | Delete flow |
| `POST` | `/api/v1/admin/flows/{id}/test` | Dry-run a flow against test data |
| `GET` | `/api/v1/admin/flows/{id}/runs` | List historical run results |

**Supported triggers**: `signup`, `login`, `password_reset`, `magic_link`, `oauth_callback`

**Supported step types**: `require_email_verification`, `require_mfa_enrollment`, `require_mfa_challenge`, `require_password_strength`, `redirect`, `webhook`, `set_metadata`, `assign_role`, `add_to_org`, `custom_check`, `delay`, `conditional`

**Example — Create flow**:
```json
{
  "name": "Require MFA on login",
  "trigger": "login",
  "enabled": true,
  "priority": 10,
  "steps": [
    { "type": "require_mfa_enrollment", "config": {} },
    { "type": "require_mfa_challenge", "config": {} }
  ]
}
```

---

## Webhooks

Base path: `/api/v1/webhooks`  Auth: Admin API key

See [03-webhooks.md](./03-webhooks.md) for the full webhook documentation including signature verification, event catalogue, and retry behavior.

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/webhooks` | Create webhook subscription |
| `GET` | `/api/v1/webhooks` | List webhooks |
| `GET` | `/api/v1/webhooks/events` | List known event names |
| `GET` | `/api/v1/webhooks/{id}` | Get webhook |
| `PATCH` | `/api/v1/webhooks/{id}` | Update webhook |
| `DELETE` | `/api/v1/webhooks/{id}` | Delete webhook |
| `POST` | `/api/v1/webhooks/{id}/test` | Fire a test event |
| `GET` | `/api/v1/webhooks/{id}/deliveries` | Delivery log (paginated) |
| `POST` | `/api/v1/webhooks/{id}/deliveries/{deliveryId}/replay` | Replay a past delivery |

---

## Email Config & Dev Inbox

### `GET /api/v1/admin/email-config` / `PATCH /api/v1/admin/email-config`

Read or update email redirect URL configuration (post-verification redirect, password-reset links, etc.).

### `POST /api/v1/admin/test-email`

Send a test email using the currently configured provider.

### `GET /api/v1/admin/email-preview/{template}`

Preview a rendered email template.

### `GET /api/v1/admin/dev/emails`

List captured dev-mode emails (only functional when email provider is `dev`). Useful for local testing without an SMTP server.

### `GET /api/v1/admin/dev/emails/{id}`

Get a specific dev email.

### `DELETE /api/v1/admin/dev/emails`

Delete all captured dev emails.

---

## Email Templates

Base path: `/api/v1/admin/email-templates`  Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/admin/email-templates` | List all templates |
| `GET` | `/api/v1/admin/email-templates/{id}` | Get a template |
| `PATCH` | `/api/v1/admin/email-templates/{id}` | Update template (subject, HTML body) |
| `POST` | `/api/v1/admin/email-templates/{id}/preview` | Preview rendered HTML |
| `POST` | `/api/v1/admin/email-templates/{id}/send-test` | Send a test email with this template |
| `POST` | `/api/v1/admin/email-templates/{id}/reset` | Reset to default |

---

## Branding

Base path: `/api/v1/admin/branding`  Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/admin/branding` | Get branding config |
| `PATCH` | `/api/v1/admin/branding` | Update branding (name, colors, fonts) |
| `POST` | `/api/v1/admin/branding/logo` | Upload logo (multipart) |
| `DELETE` | `/api/v1/admin/branding/logo` | Delete logo |
| `PATCH` | `/api/v1/admin/branding/design-tokens` | Update design token overrides |

Branding assets (logos) are served publicly at `/assets/branding/*` — content-addressed, immutable-cached.

---

## Sessions (Admin)

Base path: `/api/v1/admin/sessions`  Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/admin/sessions` | List all active sessions |
| `DELETE` | `/api/v1/admin/sessions` | Revoke all sessions |
| `DELETE` | `/api/v1/admin/sessions/{id}` | Revoke a single session |
| `POST` | `/api/v1/admin/sessions/purge-expired` | Purge expired session rows |

---

## Roles & Permissions (Admin RBAC)

Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/roles` | Create role |
| `GET` | `/api/v1/roles` | List roles |
| `GET` | `/api/v1/roles/{id}` | Get role |
| `PUT` | `/api/v1/roles/{id}` | Update role |
| `DELETE` | `/api/v1/roles/{id}` | Delete role |
| `POST` | `/api/v1/roles/{id}/permissions` | Attach permission to role |
| `DELETE` | `/api/v1/roles/{id}/permissions/{pid}` | Detach permission from role |
| `POST` | `/api/v1/permissions` | Create permission |
| `GET` | `/api/v1/permissions` | List permissions |
| `DELETE` | `/api/v1/permissions/{id}` | Delete permission |
| `GET` | `/api/v1/permissions/{id}/roles` | Roles that have this permission |
| `GET` | `/api/v1/permissions/{id}/users` | Users that have this permission |
| `POST` | `/api/v1/auth/check` | Check if a user has a permission |
| `GET` | `/api/v1/admin/permissions/batch-usage` | Batch permission usage stats |

---

## Organizations (Admin)

Base path: `/api/v1/admin/organizations`  Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/admin/organizations` | Create organization |
| `GET` | `/api/v1/admin/organizations` | List organizations |
| `GET` | `/api/v1/admin/organizations/{id}` | Get organization |
| `PATCH` | `/api/v1/admin/organizations/{id}` | Update organization |
| `DELETE` | `/api/v1/admin/organizations/{id}` | Delete organization |
| `GET` | `/api/v1/admin/organizations/{id}/members` | List members |
| `DELETE` | `/api/v1/admin/organizations/{id}/members/{uid}` | Remove member |
| `PATCH` | `/api/v1/admin/organizations/{id}/members/{uid}` | Update member role |
| `GET` | `/api/v1/admin/organizations/{id}/roles` | List org roles |
| `POST` | `/api/v1/admin/organizations/{id}/roles` | Create org role |
| `GET` | `/api/v1/admin/organizations/{id}/invitations` | List invitations |
| `POST` | `/api/v1/admin/organizations/{id}/invitations` | Create invitation |
| `DELETE` | `/api/v1/admin/organizations/{id}/invitations/{invitationId}` | Delete invitation |
| `POST` | `/api/v1/admin/organizations/{id}/invitations/{invitationId}/resend` | Resend invitation |

---

## OAuth Consents (Admin)

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/admin/oauth/consents` | List all OAuth consents (cross-user) |
| `DELETE` | `/api/v1/admin/oauth/consents/{id}` | Revoke a consent |
| `POST` | `/api/v1/admin/oauth/revoke-by-pattern` | Bulk revoke consents by pattern |
| `POST` | `/api/v1/admin/consents` | Admin-grant a consent (authorize agent on behalf of user) |
| `GET` | `/api/v1/admin/oauth/device-codes` | List pending device authorization codes |
| `POST` | `/api/v1/admin/oauth/device-codes/{user_code}/approve` | Approve a device code |
| `POST` | `/api/v1/admin/oauth/device-codes/{user_code}/deny` | Deny a device code |

---

## Proxy (Reverse Proxy Admin)

Base path: `/api/v1/admin/proxy`  Auth: Admin API key

These endpoints return 404 when the proxy is disabled at boot.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/admin/proxy/status` | Breaker stats + upstream health |
| `GET` | `/api/v1/admin/proxy/status/stream` | SSE stream of status updates |
| `GET` | `/api/v1/admin/proxy/rules` | Compiled rule set (YAML-sourced) |
| `POST` | `/api/v1/admin/proxy/simulate` | Simulate rule matching for a request |
| `GET` | `/api/v1/admin/proxy/rules/db` | List DB-backed rule overrides |
| `POST` | `/api/v1/admin/proxy/rules/db` | Create a DB rule override |
| `GET` | `/api/v1/admin/proxy/rules/db/{id}` | Get rule override |
| `PATCH` | `/api/v1/admin/proxy/rules/db/{id}` | Update rule override |
| `DELETE` | `/api/v1/admin/proxy/rules/db/{id}` | Delete rule override |
| `GET` | `/api/v1/admin/proxy/lifecycle` | Proxy lifecycle status |
| `POST` | `/api/v1/admin/proxy/start` | Start the proxy subsystem |
| `POST` | `/api/v1/admin/proxy/stop` | Stop the proxy subsystem |
| `POST` | `/api/v1/admin/proxy/reload` | Reload rules without restart |

---

## SSO Connections (Admin)

Base path: `/api/v1/sso/connections`  Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/sso/connections` | Create SSO connection (SAML or OIDC) |
| `GET` | `/api/v1/sso/connections` | List connections |
| `GET` | `/api/v1/sso/connections/{id}` | Get connection |
| `PUT` | `/api/v1/sso/connections/{id}` | Update connection |
| `DELETE` | `/api/v1/sso/connections/{id}` | Delete connection |

Public SSO endpoints (no auth):
- `GET /api/v1/sso/saml/{connection_id}/metadata` — SAML SP metadata XML
- `POST /api/v1/sso/saml/{connection_id}/acs` — SAML ACS (Assertion Consumer Service)
- `GET /api/v1/sso/oidc/{connection_id}/auth` — OIDC authorization redirect
- `GET /api/v1/sso/oidc/{connection_id}/callback` — OIDC callback

---

## API Keys (Admin)

Base path: `/api/v1/api-keys`  Auth: Admin API key

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/api-keys` | Issue a new admin API key |
| `GET` | `/api/v1/api-keys` | List API keys |
| `GET` | `/api/v1/api-keys/{id}` | Get key details |
| `PATCH` | `/api/v1/api-keys/{id}` | Update key metadata |
| `DELETE` | `/api/v1/api-keys/{id}` | Revoke key |
| `POST` | `/api/v1/api-keys/{id}/rotate` | Rotate to a new key value |

---

## Admin Logs

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/admin/logs/stream` | SSE stream of structured admin log output |
