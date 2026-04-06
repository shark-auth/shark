# SharkAuth Postman Testing Guide

## Setup

### 1. Start the server

```bash
./shark.exe --config sharkauth.local.yaml
```

### 2. Postman environment variables

Create a Postman environment called **SharkAuth Local** with:

| Variable | Initial Value |
|----------|---------------|
| `base_url` | `http://localhost:8080` |
| `admin_key` | `admin-dev-key-123` |
| `user_email` | `test@example.com` |
| `user_password` | `password123` |
| `user_id` | _(empty — auto-set by tests)_ |
| `mfa_secret` | _(empty — auto-set)_ |
| `api_key` | _(empty — auto-set)_ |
| `role_id` | _(empty — auto-set)_ |
| `perm_id` | _(empty — auto-set)_ |

### 3. Cookie handling

Go to **Settings > General** and enable:
- **Automatically follow redirects** — OFF (so you can inspect 302s)
- **Send cookies** — ON

Postman handles cookies automatically per domain. After signup/login, the `shark_session` cookie persists across requests.

---

## Request Collection

### Auth — Core

#### POST Signup
```
POST {{base_url}}/api/v1/auth/signup
Content-Type: application/json

{
  "email": "{{user_email}}",
  "password": "{{user_password}}",
  "name": "Test User"
}
```
**Tests tab** (auto-save user_id):
```javascript
if (pm.response.code === 201) {
    var body = pm.response.json();
    pm.environment.set("user_id", body.id);
}
```
**Expected:** 201, session cookie set

---

#### POST Login
```
POST {{base_url}}/api/v1/auth/login
Content-Type: application/json

{
  "email": "{{user_email}}",
  "password": "{{user_password}}"
}
```
**Expected:** 200 `{ "id", "email", "name" }` or `{ "mfa_required": true }` if MFA enabled

---

#### GET Me
```
GET {{base_url}}/api/v1/auth/me
```
**Expected:** 200 with user object (uses session cookie automatically)

---

#### POST Logout
```
POST {{base_url}}/api/v1/auth/logout
```
**Expected:** 200, session cookie cleared

---

### Auth — MFA

#### POST MFA Enroll
```
POST {{base_url}}/api/v1/auth/mfa/enroll
```
**Tests tab:**
```javascript
if (pm.response.code === 200) {
    var body = pm.response.json();
    pm.environment.set("mfa_secret", body.secret);
    // Copy the qr_uri to a QR code generator or Google Authenticator
}
```
**Expected:** 200 `{ "secret": "BASE32...", "qr_uri": "otpauth://totp/..." }`

---

#### POST MFA Verify (confirm setup)
```
POST {{base_url}}/api/v1/auth/mfa/verify
Content-Type: application/json

{
  "code": "123456"
}
```
Use a TOTP app (Google Authenticator, Authy) or https://totp.danhersam.com with the secret.

**Expected:** 200 with recovery codes

---

#### POST MFA Challenge (during login)
```
POST {{base_url}}/api/v1/auth/mfa/challenge
Content-Type: application/json

{
  "code": "123456"
}
```
**Expected:** 200, session upgraded to full access

---

#### POST MFA Recovery
```
POST {{base_url}}/api/v1/auth/mfa/recovery
Content-Type: application/json

{
  "code": "abcd1234"
}
```
Use one of the recovery codes from the enroll step.

---

#### DELETE Disable MFA
```
DELETE {{base_url}}/api/v1/auth/mfa
Content-Type: application/json

{
  "code": "123456"
}
```
Requires a valid current TOTP code.

---

### Auth — Magic Links

#### POST Send Magic Link
```
POST {{base_url}}/api/v1/auth/magic-link/send
Content-Type: application/json

{
  "email": "magicuser@example.com"
}
```
**Expected:** 200 always (doesn't leak if email exists). Check server logs or MemoryEmailSender for the token. Without SMTP configured, the email won't actually send — check logs.

---

#### GET Verify Magic Link
```
GET {{base_url}}/api/v1/auth/magic-link/verify?token=<token_from_email>
```
**Expected:** 302 redirect with session cookie set

---

### Auth — Passkeys

#### POST Passkey Register Begin
```
POST {{base_url}}/api/v1/auth/passkey/register/begin
```
Requires session cookie (login first).

**Expected:** 200 with `PublicKeyCredentialCreationOptions` (challenge, rp info, user info)

---

#### POST Passkey Login Begin
```
POST {{base_url}}/api/v1/auth/passkey/login/begin
Content-Type: application/json

{
  "email": "{{user_email}}"
}
```
Or send empty body `{}` for discoverable credential flow.

**Expected:** 200 with `PublicKeyCredentialRequestOptions`

> Note: The finish endpoints require actual WebAuthn browser interaction — can't test fully from Postman.

---

### Auth — OAuth

#### GET OAuth Start
```
GET {{base_url}}/api/v1/auth/oauth/github
```
**Expected:** 302 redirect to GitHub's OAuth consent page. Requires `github.client_id` configured.

---

### RBAC

All RBAC endpoints require the admin key header:
```
X-Admin-Key: {{admin_key}}
```

#### POST Create Role
```
POST {{base_url}}/api/v1/roles
X-Admin-Key: {{admin_key}}
Content-Type: application/json

{
  "name": "editor",
  "description": "Can edit content"
}
```
**Tests tab:**
```javascript
if (pm.response.code === 201) {
    pm.environment.set("role_id", pm.response.json().id);
}
```

---

#### GET List Roles
```
GET {{base_url}}/api/v1/roles
X-Admin-Key: {{admin_key}}
```

---

#### POST Create Permission
```
POST {{base_url}}/api/v1/permissions
X-Admin-Key: {{admin_key}}
Content-Type: application/json

{
  "action": "write",
  "resource": "posts"
}
```
**Tests tab:**
```javascript
if (pm.response.code === 201) {
    pm.environment.set("perm_id", pm.response.json().id);
}
```

---

#### POST Attach Permission to Role
```
POST {{base_url}}/api/v1/roles/{{role_id}}/permissions
X-Admin-Key: {{admin_key}}
Content-Type: application/json

{
  "permission_id": "{{perm_id}}"
}
```

---

#### POST Assign Role to User
```
POST {{base_url}}/api/v1/users/{{user_id}}/roles
X-Admin-Key: {{admin_key}}
Content-Type: application/json

{
  "role_id": "{{role_id}}"
}
```

---

#### GET User Roles
```
GET {{base_url}}/api/v1/users/{{user_id}}/roles
X-Admin-Key: {{admin_key}}
```

---

#### GET User Effective Permissions
```
GET {{base_url}}/api/v1/users/{{user_id}}/permissions
X-Admin-Key: {{admin_key}}
```

---

#### POST Check Permission
```
POST {{base_url}}/api/v1/auth/check
Content-Type: application/json

{
  "user_id": "{{user_id}}",
  "action": "write",
  "resource": "posts"
}
```
**Expected:** `{ "allowed": true }` or `{ "allowed": false }`

---

### M2M API Keys

#### POST Create API Key
```
POST {{base_url}}/api/v1/api-keys
X-Admin-Key: {{admin_key}}
Content-Type: application/json

{
  "name": "backend-service",
  "scopes": ["users:read", "roles:read"],
  "rate_limit": 100
}
```
**Tests tab:**
```javascript
if (pm.response.code === 201) {
    // SAVE THIS — the full key is only shown once!
    pm.environment.set("api_key", pm.response.json().key);
}
```

---

#### GET List API Keys
```
GET {{base_url}}/api/v1/api-keys
X-Admin-Key: {{admin_key}}
```
Shows prefix + metadata only, never the full key.

---

#### POST Rotate API Key
```
POST {{base_url}}/api/v1/api-keys/{{key_id}}/rotate
X-Admin-Key: {{admin_key}}
```
Atomically creates new key and revokes old.

---

#### Use API Key (Bearer auth)
```
GET {{base_url}}/api/v1/users
Authorization: Bearer {{api_key}}
```
Works if the key has the required scope.

---

### SSO

#### POST Create SSO Connection
```
POST {{base_url}}/api/v1/sso/connections
X-Admin-Key: {{admin_key}}
Content-Type: application/json

{
  "type": "oidc",
  "name": "Okta Production",
  "domain": "company.com",
  "oidc_issuer": "https://dev-123456.okta.com",
  "oidc_client_id": "your-client-id",
  "oidc_client_secret": "your-client-secret"
}
```

---

#### GET SSO Auto-Route
```
GET {{base_url}}/api/v1/auth/sso?email=user@company.com
```
**Expected:** Redirect to the matching IdP based on email domain.

---

### Audit Logs

#### GET List Audit Logs
```
GET {{base_url}}/api/v1/audit-logs
X-Admin-Key: {{admin_key}}
```
**Query params** (all optional):
- `?action=user.login,user.signup`
- `?actor_id=usr_xxx`
- `?status=failure`
- `?ip=127.0.0.1`
- `?from=2026-04-01T00:00:00Z&to=2026-04-30T00:00:00Z`
- `?limit=20&cursor=aud_xxx`

---

#### GET User Audit Logs
```
GET {{base_url}}/api/v1/users/{{user_id}}/audit-logs
X-Admin-Key: {{admin_key}}
```

---

#### POST Export Audit Logs
```
POST {{base_url}}/api/v1/audit-logs/export
X-Admin-Key: {{admin_key}}
Content-Type: application/json

{
  "from": "2026-04-01T00:00:00Z",
  "to": "2026-04-30T00:00:00Z"
}
```

---

### Admin

#### GET Health Check
```
GET {{base_url}}/healthz
```
**Expected:** `{ "status": "ok" }`

---

## Recommended Test Flow

Run these in order to exercise the full system:

1. **Healthz** — verify server is up
2. **Signup** — create account, saves user_id
3. **Me** — verify session works
4. **Create role** — RBAC setup
5. **Create permission** — RBAC setup
6. **Attach permission to role** — wire RBAC
7. **Assign role to user** — give user access
8. **Check permission** — verify RBAC works
9. **Create API key** — M2M setup
10. **List audit logs** — see all events so far
11. **MFA enroll** — start MFA setup
12. **MFA verify** — complete setup with TOTP code
13. **Logout** — clear session
14. **Login** — should get `mfa_required: true`
15. **MFA challenge** — complete login with TOTP
16. **Me** — full access again
17. **Logout + Login** — verify flow is repeatable
