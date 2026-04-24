# Secret Rotation

This document covers how to rotate sensitive credentials in SharkAuth.

---

## server.secret

The `server.secret` is the most critical secret in SharkAuth. It's used for:

- **Session encryption** (AES-256 + HMAC-SHA256 via gorilla/securecookie)
- **Field-level encryption** (AES-256-GCM for MFA secrets, SSO credentials)

### Generating a new secret

```bash
# Generate a cryptographically random 64-character hex string
openssl rand -hex 32
```

Minimum length: 32 characters. Recommended: 64 characters.

### Rotation procedure

**Rotating `server.secret` is a breaking change.** All active sessions and encrypted fields become unreadable with the old key.

1. **Schedule a maintenance window.** All users will be logged out.
2. **Generate the new secret:**
   ```bash
   export NEW_SECRET=$(openssl rand -hex 32)
   ```
3. **Update the config** (either `sharkauth.yaml` or the `SHARKAUTH_SECRET` env var):
   ```yaml
   server:
     secret: "your-new-64-char-secret-here"
   ```
4. **Restart SharkAuth.** On startup:
   - All existing sessions become invalid (users must re-login)
   - Existing MFA secrets encrypted with the old key will fail to decrypt. The `FieldEncryptor.Decrypt` method transparently handles unencrypted legacy data, but data encrypted with a *different* key will return errors.
5. **If users have MFA enabled**, they will need to re-enroll MFA:
   - Disable MFA on affected accounts via the admin API: `DELETE /api/v1/auth/mfa` (requires their TOTP code) or update the user directly via `PATCH /api/v1/users/{id}`
   - Users re-enroll via `POST /api/v1/auth/mfa/enroll`

### Avoiding downtime (future)

A dual-key rotation scheme (try new key, fall back to old key) is planned for a future release. For now, treat secret rotation as a maintenance event.

---

## Admin API Keys

Admin API keys (`sk_live_*` with scope `["*"]`) authenticate all admin endpoints.

### Rotating an admin key

1. **Create a new admin key** using the current admin key:
   ```bash
   curl -X POST https://auth.example.com/api/v1/api-keys \
     -H "Authorization: Bearer sk_live_CURRENT_KEY" \
     -H "Content-Type: application/json" \
     -d '{"name": "admin-rotated-2026-04", "scopes": ["*"]}'
   ```
   Save the returned key — it's shown only once.

2. **Update all services** that use the old key to use the new one.

3. **Revoke the old key:**
   ```bash
   curl -X DELETE https://auth.example.com/api/v1/api-keys/KEY_ID \
     -H "Authorization: Bearer sk_live_NEW_KEY"
   ```

**Safety:** SharkAuth prevents revoking the last admin-scoped key. You must always have at least one active `["*"]` key.

### Rotating a scoped API key

Use the built-in rotation endpoint, which atomically creates a new key and revokes the old one:

```bash
curl -X POST https://auth.example.com/api/v1/api-keys/KEY_ID/rotate \
  -H "Authorization: Bearer sk_live_ADMIN_KEY"
```

The response includes the new full key (shown once). The old key is immediately revoked.

---

## OAuth Provider Secrets

OAuth `client_secret` values are stored in the YAML config (or env vars), not in the database.

### Rotation

1. Generate a new client secret in the provider's developer console (Google Cloud Console, GitHub Settings, etc.)
2. Update `sharkauth.yaml` or the corresponding `SHARKAUTH_SOCIAL__GOOGLE__CLIENT_SECRET` env var
3. Restart SharkAuth
4. Users with active OAuth sessions are unaffected — secrets are only used during the OAuth code exchange

---

## SMTP / Resend API Key

1. Generate a new API key in Resend (or your SMTP provider)
2. Update `sharkauth.yaml` or the `RESEND_API_KEY` env var
3. Restart SharkAuth
4. Test by triggering a magic link or password reset email

---

## SSO Connection Credentials

OIDC client secrets and SAML IdP certificates are stored in the database and encrypted at rest with the field encryption key (derived from `server.secret`).

### Rotating OIDC client secret

```bash
curl -X PUT https://auth.example.com/api/v1/sso/connections/CONNECTION_ID \
  -H "Authorization: Bearer sk_live_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"oidc_client_secret": "new-secret-from-idp"}'
```

### Rotating SAML IdP certificate

```bash
curl -X PUT https://auth.example.com/api/v1/sso/connections/CONNECTION_ID \
  -H "Authorization: Bearer sk_live_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"saml_idp_cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"}'
```

---

## Environment Variable Reference

| Variable | Maps to | Sensitive |
|----------|---------|-----------|
| `SHARKAUTH_SECRET` | `server.secret` | Yes |
| `RESEND_API_KEY` | `smtp.password` | Yes |
| `GOOGLE_CLIENT_SECRET` | `social.google.client_secret` | Yes |
| `GITHUB_CLIENT_SECRET` | `social.github.client_secret` | Yes |
| `APPLE_KEY_ID` | `social.apple.key_id` | Moderate |
| `DISCORD_CLIENT_SECRET` | `social.discord.client_secret` | Yes |

Never commit these values to version control. Use `.env` files, container secrets, or a secrets manager.
