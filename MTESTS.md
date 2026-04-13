# Manual Tests

Manual test procedures to validate SharkAuth before release. Run these against a local instance with `sharkauth.local.yaml`.

**Prerequisites:**
- SharkAuth running locally: `go run ./cmd/shark --config sharkauth.local.yaml`
- Admin API key from first-run stdout (referred to as `$ADMIN_KEY` below)
- A tool for HTTP requests (curl, Postman, HTTPie)
- For email tests: a configured SMTP/Resend sender, or check server logs for verification URLs

---

## 1. Startup Validation

### 1.1 Secret validation rejects short secrets
```bash
# Set a short secret and try to start
# Expected: fatal error "server.secret must be at least 32 characters"
SHARKAUTH_SECRET="short" go run ./cmd/shark
```
- [ ] Server refuses to start with secret < 32 chars
- [ ] Error message shows the actual length and suggests `openssl rand -hex 32`

### 1.2 Admin key generated on first run
```bash
rm -rf ./data/sharkauth.db
go run ./cmd/shark --config sharkauth.local.yaml
```
- [ ] Admin API key printed to stdout on first run
- [ ] Key starts with `sk_live_`
- [ ] Second run does NOT generate a new key

### 1.3 Health check pings database
```bash
curl -s http://localhost:8080/healthz | jq
```
- [ ] Returns `{"status": "ok"}` with 200

---

## 2. Security Headers

```bash
curl -sI http://localhost:8080/healthz
```
- [ ] `X-Content-Type-Options: nosniff`
- [ ] `X-Frame-Options: DENY`
- [ ] `Content-Security-Policy: default-src 'self'; frame-ancestors 'none'`
- [ ] `Referrer-Policy: strict-origin-when-cross-origin`
- [ ] `X-XSS-Protection: 0`
- [ ] `Permissions-Policy: camera=(), microphone=(), geolocation=()`
- [ ] No `Strict-Transport-Security` on HTTP (only set when HTTPS detected)

---

## 3. Password Authentication

### 3.1 Signup with weak password rejected
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"weak"}'
```
- [ ] Returns 400 with `weak_password` error

### 3.2 Signup with common password rejected
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```
- [ ] Returns 400 — "too common" or missing uppercase

### 3.3 Signup with no uppercase rejected
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"securepassword123"}'
```
- [ ] Returns 400 — must contain uppercase letter

### 3.4 Successful signup
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePass123","name":"Test User"}'
```
- [ ] Returns 201 with user object
- [ ] `emailVerified: false`
- [ ] `Set-Cookie` header contains `shark_session`

### 3.5 Login
```bash
curl -s -c cookies.txt -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePass123"}'
```
- [ ] Returns 200 with user object
- [ ] Cookie jar has `shark_session`

### 3.6 Get current user
```bash
curl -s -b cookies.txt http://localhost:8080/api/v1/auth/me
```
- [ ] Returns 200 with user details

### 3.7 Logout
```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/logout
```
- [ ] Returns 200
- [ ] Subsequent `/auth/me` returns 401

---

## 4. Account Lockout

### 4.1 Trigger lockout with 5 failed attempts
```bash
for i in $(seq 1 5); do
  curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"WrongPassword1"}'
done
```
- [ ] First 4 attempts return 401 `invalid_credentials`
- [ ] 5th attempt returns 429 `account_locked`

### 4.2 Locked account rejects correct password
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePass123"}'
```
- [ ] Returns 429 `account_locked` even with correct password
- [ ] Wait 15 minutes, then retry — should succeed

---

## 5. Email Verification

### 5.1 Send verification email (requires session)
```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/email/verify/send
```
- [ ] Returns 200 `"Verification email sent"`
- [ ] Check server logs or email inbox for the verification URL

### 5.2 Verify email with token
```bash
# Extract token from the verification URL and call:
curl -s "http://localhost:8080/api/v1/auth/email/verify?token=TOKEN_HERE"
```
- [ ] Returns 200 `"Email verified successfully"`
- [ ] Subsequent `GET /auth/me` shows `emailVerified: true`

### 5.3 Already verified user
```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/email/verify/send
```
- [ ] Returns 200 `"Email is already verified"`

---

## 6. Password Reset

### 6.1 Send reset link
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/password/send-reset-link \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'
```
- [ ] Returns 200 regardless of whether email exists (anti-enumeration)

### 6.2 Reset with token
```bash
# Extract token from reset email/logs
curl -s -X POST http://localhost:8080/api/v1/auth/password/reset \
  -H "Content-Type: application/json" \
  -d '{"token":"TOKEN_HERE","password":"NewSecurePass123"}'
```
- [ ] Returns 200 on success
- [ ] Old password no longer works
- [ ] New password works

### 6.3 Reset with weak password rejected
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/password/reset \
  -H "Content-Type: application/json" \
  -d '{"token":"TOKEN_HERE","password":"weak"}'
```
- [ ] Returns 400 `weak_password`

---

## 7. User CRUD (Admin)

### 7.1 List users
```bash
curl -s -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/api/v1/users | jq
```
- [ ] Returns JSON array of users
- [ ] Each user has id, email, emailVerified, name, createdAt, updatedAt

### 7.2 List users with search
```bash
curl -s -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/users?search=test&limit=10" | jq
```
- [ ] Returns filtered results matching "test" in email or name

### 7.3 Get user by ID
```bash
curl -s -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/api/v1/users/USER_ID | jq
```
- [ ] Returns single user object

### 7.4 Update user
```bash
curl -s -X PATCH -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/users/USER_ID \
  -d '{"name":"Updated Name","email_verified":true}'
```
- [ ] Returns updated user
- [ ] Changes reflected in subsequent GET

### 7.5 Delete user
```bash
curl -s -X DELETE -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/api/v1/users/USER_ID
```
- [ ] Returns 200
- [ ] User no longer appears in list
- [ ] User's sessions, OAuth accounts, passkeys, roles all cascade-deleted

---

## 8. User Self-Deletion

### 8.1 Delete own account
```bash
# Login first, then:
curl -s -b cookies.txt -X DELETE http://localhost:8080/api/v1/auth/me
```
- [ ] Returns 200 `"Account deleted"`
- [ ] Cookie is cleared
- [ ] Cannot login with that email anymore (account gone)
- [ ] Re-signup with same email works (fresh account)

---

## 9. MFA (TOTP)

### 9.1 Enroll MFA
```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/mfa/enroll | jq
```
- [ ] Returns `secret` (base32) and `qr_uri` (otpauth:// URL)
- [ ] QR URI scannable by Google Authenticator

### 9.2 Verify first code
```bash
# Generate TOTP code from the secret, then:
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/mfa/verify \
  -H "Content-Type: application/json" \
  -d '{"code":"123456"}'
```
- [ ] Returns 200 with `mfa_enabled: true` and `recovery_codes` (array of 10)
- [ ] Recovery codes are 8-char alphanumeric strings

### 9.3 Login with MFA
```bash
# Login returns mfaRequired: true
curl -s -c cookies.txt -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"mfa-user@example.com","password":"SecurePass123"}'
```
- [ ] Returns `{"mfaRequired": true}`
- [ ] `GET /auth/me` returns 401 (session has mfa_passed=false)

### 9.4 MFA challenge
```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/mfa/challenge \
  -H "Content-Type: application/json" \
  -d '{"code":"CURRENT_TOTP_CODE"}'
```
- [ ] Returns 200 with user object
- [ ] `GET /auth/me` now works

### 9.5 Recovery code
```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/mfa/recovery \
  -H "Content-Type: application/json" \
  -d '{"code":"RECOVERY_CODE_HERE"}'
```
- [ ] Returns 200 — session upgraded
- [ ] Same code fails on second use (one-time)

### 9.6 Disable MFA
```bash
curl -s -b cookies.txt -X DELETE http://localhost:8080/api/v1/auth/mfa \
  -H "Content-Type: application/json" \
  -d '{"code":"CURRENT_TOTP_CODE"}'
```
- [ ] Returns 200 `mfa_enabled: false`
- [ ] Next login does not require MFA

---

## 10. OAuth

### 10.1 Start OAuth flow
```bash
# Open in browser:
http://localhost:8080/api/v1/auth/oauth/google
```
- [ ] Redirects to Google consent screen
- [ ] `shark_oauth_state` cookie is set

### 10.2 OAuth callback
- [ ] After consent, redirected back to callback URL
- [ ] User created (if new) or found (if existing)
- [ ] Session cookie set
- [ ] If `social.redirect_url` is configured, redirects to frontend
- [ ] If not configured, returns JSON user response
- [ ] Avatar URL saved from provider (check user via admin API)

### 10.3 Invalid provider
```bash
curl -s http://localhost:8080/api/v1/auth/oauth/invalid_provider
```
- [ ] Returns 400 `invalid_provider`

---

## 11. API Keys

### 11.1 Create key
```bash
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/api-keys \
  -d '{"name":"test-key","scopes":["users:read"],"rate_limit":100}'
```
- [ ] Returns 201 with full key (starts with `sk_live_`)
- [ ] Key shown once — not retrievable later

### 11.2 Rotate key
```bash
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/api/v1/api-keys/KEY_ID/rotate
```
- [ ] Returns new key
- [ ] Old key immediately stops working

### 11.3 Cannot revoke last admin key
```bash
# If there's only one admin key, try to revoke it:
curl -s -X DELETE -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/api/v1/api-keys/KEY_ID
```
- [ ] Returns 400 — cannot revoke last admin-scoped key

---

## 12. RBAC

### 12.1 Create role and permission
```bash
# Create role
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/roles \
  -d '{"name":"editor","description":"Can edit content"}'

# Create permission
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/permissions \
  -d '{"action":"write","resource":"articles"}'

# Attach permission to role
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/roles/ROLE_ID/permissions \
  -d '{"permission_id":"PERM_ID"}'

# Assign role to user
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/api/v1/users/USER_ID/roles \
  -H "Content-Type: application/json" \
  -d '{"role_id":"ROLE_ID"}'
```
- [ ] Role created with 201
- [ ] Permission created with 201
- [ ] Permission attached to role
- [ ] Role assigned to user

### 12.2 Check permission
```bash
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/auth/check \
  -d '{"user_id":"USER_ID","action":"write","resource":"articles"}'
```
- [ ] Returns `{"allowed": true}`

---

## 13. SSO

### 13.1 Create OIDC connection
```bash
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/sso/connections \
  -d '{"type":"oidc","name":"Corporate SSO","domain":"corp.com","oidc_issuer":"https://idp.corp.com","oidc_client_id":"client123","oidc_client_secret":"secret456"}'
```
- [ ] Returns 201 with connection object
- [ ] Error messages do NOT leak internal details

### 13.2 Auto-route by email domain
```bash
curl -s "http://localhost:8080/api/v1/auth/sso?email=user@corp.com"
```
- [ ] Returns connection info + redirect URL
- [ ] Unknown domain returns generic error (not internal details)

---

## 14. Audit Logs

### 14.1 Query logs
```bash
curl -s -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/audit-logs?limit=10" | jq
```
- [ ] Returns array of audit events
- [ ] Events include: signup, login, logout, etc.

### 14.2 Filter by action
```bash
curl -s -H "Authorization: Bearer $ADMIN_KEY" \
  "http://localhost:8080/api/v1/audit-logs?action=user.login&limit=5" | jq
```
- [ ] Returns only login events

### 14.3 CSV export
```bash
curl -s -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/api/v1/audit-logs/export
```
- [ ] Returns CSV content
- [ ] CSV has proper headers and row data

---

## 15. Rate Limiting

### 15.1 Global rate limit
```bash
# Send 150 requests rapidly
for i in $(seq 1 150); do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/healthz
done | sort | uniq -c
```
- [ ] Most return 200
- [ ] Some return 429 after bucket exhaustion
- [ ] Response includes `Retry-After: 1` header

---

## 16. Docker

### 16.1 Build image
```bash
docker build -t sharkauth .
```
- [ ] Build succeeds
- [ ] Image size < 30MB (check with `docker images sharkauth`)

### 16.2 Run container
```bash
docker run -p 8080:8080 \
  -e SHARKAUTH_SECRET=$(openssl rand -hex 32) \
  -v sharkauth-data:/app/data \
  sharkauth --config /dev/null
```
- [ ] Container starts
- [ ] Admin key printed to logs
- [ ] Health check responds at `http://localhost:8080/healthz`
- [ ] Process runs as non-root user (check with `docker exec CONTAINER_ID whoami`)

---

## 17. Field-Level Encryption

### 17.1 MFA secret encrypted at rest
```bash
# After enrolling MFA for a user, check the database:
sqlite3 ./data/sharkauth.db "SELECT mfa_secret FROM users WHERE mfa_enabled=1 LIMIT 1;"
```
- [ ] Value starts with `enc::` (encrypted)
- [ ] Value is NOT a readable base32 TOTP secret
- [ ] MFA challenge still works (decryption is transparent)

### 17.2 Legacy unencrypted data migrates transparently
- [ ] If an existing user has a plaintext MFA secret (no `enc::` prefix), TOTP validation still works
- [ ] On next MFA enroll, the new secret is stored encrypted

---

## Pass Criteria

All checkboxes above should be checked before the dashboard milestone begins. Tests in sections 1-8 and 15-17 are highest priority. OAuth (10) and SSO (13) tests depend on provider configuration and can be validated in the production deployment at BuildersMTY.
