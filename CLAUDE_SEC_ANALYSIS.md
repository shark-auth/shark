# SharkAuth Security Audit Report

A comprehensive security analysis of the SharkAuth authentication server, covering cryptographic practices, session management, API security, and operational hardening.

---

## Strengths

| # | Area | Details |
|---|------|---------|
| 1 | **Password Hashing** | Argon2id with configurable params (`64MB` memory, `3` iterations, `16`-byte salt). Supports bcrypt migration with auto-rehash. Constant-time comparison via `subtle.ConstantTimeCompare()`. |
| 2 | **Session Security** | Encrypted cookies via `gorilla/securecookie` (AES-256 + HMAC-SHA256). `HttpOnly`, `SameSite=Lax`, conditional `Secure` flag. No JWT (avoids common JWT pitfalls). |
| 3 | **Crypto Hygiene** | `crypto/rand` used everywhere. Zero usage of `md5`, `sha1`, or `math/rand` for security. Rejection sampling eliminates modulo bias in recovery codes. |
| 4 | **Timing Attack Protection** | `subtle.ConstantTimeCompare()` on: admin key, OAuth state, API keys, password verification, magic link tokens. Dummy bcrypt comparison when no recovery codes found. |
| 5 | **SQL Injection Protection** | 100% parameterized queries throughout `sqlite.go`. Dynamic audit log filters use `?` placeholders with args array. |
| 6 | **API Key Management** | SHA-256 hashed in DB (never stored plaintext). Full key shown only once at creation. Revocation, expiration, scope checking, per-key rate limiting. |
| 7 | **Audit Logging** | Comprehensive event trail (actor, action, target, IP, user agent, metadata). Retention policies with background cleanup. Export endpoint. |
| 8 | **Rate Limiting** | IP-based token bucket globally + per-API-key rate limits. Magic link throttled to `1/60s`. Returns proper `429` + `Retry-After`. |
| 9 | **MFA Implementation** | TOTP via `pquerna/otp`. 10 recovery codes with bcrypt hashing. Session-based MFA state with atomic upgrades. Single-use enforcement. |
| 10 | **Error Disclosure** | Login errors return generic "Invalid email or password" -- doesn't leak user existence. |
| 11 | **HTTP Server** | `ReadTimeout`, `WriteTimeout`, `IdleTimeout` all configured. Graceful shutdown with 30s drain. |
| 12 | **Config Architecture** | All production secrets via `${ENV_VAR}` interpolation. `SHARKAUTH_` prefix with double-underscore nesting. |

---

## Weaknesses

### 1. Error Leakage (SSO) -- **HIGH**

- **Finding:** Raw `err.Error()` from SAML/OIDC libraries returned to clients
- **Location:** `sso_handlers.go:67,78,94,117,133,152,171,193`

### 2. OAuth State Cookie -- **MEDIUM**

- **Finding:** Missing `Secure` flag on OAuth state cookie (short-lived but still a gap)
- **Location:** `oauth_handlers.go:47-54`

### 3. Admin Bypass in Dev -- **MEDIUM**

- **Finding:** Empty `admin.api_key` config disables admin auth entirely -- risky if accidentally deployed
- **Location:** `middleware/admin.go:15`

### 4. OIDC State Store -- **LOW**

- **Finding:** In-memory map for OIDC state -- won't survive restarts and won't scale to multiple instances
- **Location:** `sso/oidc.go:17-22`

### 5. Integer Overflow (gosec) -- **LOW**

- **Finding:** `len(hash)` cast to `uint32` without bounds check
- **Location:** `password.go:61`, `passkey.go:439`

---

## Blockers (Must Fix Before Launch)

### 1. No Security Headers

- **Impact:** Clickjacking, MIME sniffing, no HSTS, no CSP. Fails basic security scans.
- **Fix:** Add middleware: `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Strict-Transport-Security`, `Content-Security-Policy`, `Referrer-Policy`.

### 2. Secrets Stored Plaintext in DB

- **Impact:** SSO `oidc_client_secret`, `saml_idp_cert`, and `mfa_secret` stored unencrypted in SQLite.
- **Fix:** Implement field-level AES-256-GCM encryption for these columns using the server secret.

### 3. No TLS in Application

- **Impact:** Server uses `http.ListenAndServe` only -- no HTTPS. Cookie `Secure` flag depends on config URL, not actual TLS.
- **Fix:** Either add `ListenAndServeTLS()` with cert config, or document that a TLS-terminating reverse proxy (Caddy/Nginx) is required.

### 4. Docker Runs as Root

- **Impact:** No `USER` directive in Dockerfile -- container runs as UID 0.
- **Fix:** Add `RUN addgroup -g 1000 shark && adduser -D -u 1000 -G shark shark` + `USER shark`.

### 5. No Account Lockout

- **Impact:** No failed login attempt tracking. Brute force possible within rate limit budget (`100 req/s`).
- **Fix:** Implement per-email failed attempt counter with progressive lockout (e.g., 5 failures = 15min lock).

---

## Upgrades (Should Fix, Not Blocking)

| # | Area | Current State | Recommended Upgrade |
|---|------|---------------|---------------------|
| 1 | **Password Policy** | Min 8 chars, no complexity rules | Add: uppercase + lowercase + digit required, or use zxcvbn entropy scoring |
| 2 | **CSRF Protection** | Relies on `SameSite=Lax` only | Add explicit CSRF tokens for any form-based flows (or document API-only usage) |
| 3 | **Auth-specific Rate Limiting** | Generic IP-based rate limit covers login | Add per-email rate limit on `/auth/login` and `/auth/signup` (e.g., 5 attempts/email/15min) |
| 4 | **Redirect URL Validation** | Magic link and SAML redirect URLs not validated against allowlist | Validate all redirect URLs against configured `base_url` domain |
| 5 | **Recovery Code Entropy** | 8-char codes from 36-char alphabet (~41 bits entropy) | Increase to 10 chars or add mixed case (~47-52 bits) |
| 6 | **Session Revocation on Password Change** | Not verified | Invalidate all other sessions when password changes or MFA is disabled |
| 7 | **Docker Health Check** | No `HEALTHCHECK` instruction | Add `HEALTHCHECK CMD wget -q --spider http://localhost:8080/healthz` |
| 8 | **Suspicious Activity Alerts** | No email notifications | Send email on: login from new IP/device, MFA disabled, password changed |
| 9 | **SSO Error Wrapping** | Raw library errors exposed | Wrap with generic messages, log originals server-side |

---

## Missing for Launch

| # | Category | What's Missing | Priority |
|---|----------|----------------|----------|
| 1 | **Password Reset Flow** | No explicit password reset endpoint found -- magic links handle login but not password change | **P0** |
| 2 | **Security Headers Middleware** | Zero OWASP-recommended response headers set | **P0** |
| 3 | **Field-Level Encryption** | MFA secrets, SSO credentials stored in cleartext DB | **P0** |
| 4 | **Non-Root Docker User** | Production container runs as root | **P1** |
| 5 | **Account Lockout** | Brute force protection beyond IP rate limiting | **P1** |
| 6 | **Password Complexity** | Only length check, no entropy/complexity validation | **P1** |
| 7 | **CORS Documentation** | No guidance on what origins to configure for production | **P2** |
| 8 | **Secret Rotation Docs** | No documented procedure for rotating `SHARKAUTH_SECRET` or admin key | **P2** |
| 9 | **Dependency Audit** | No `go audit` or vulnerability scanning in CI pipeline | **P2** |
| 10 | **User Deletion / GDPR** | `DELETE /users/{id}` endpoint not implemented | **P2** |
| 11 | **Login Anomaly Detection** | No tracking of new devices, IPs, or geolocations | **P3** |
| 12 | **API Versioning Strategy** | Currently `/api/v1` only -- no deprecation or migration plan documented | **P3** |

---

## Bottom Line

> **The core crypto and auth logic is excellent** -- Argon2id, constant-time comparisons, proper CSPRNG usage, encrypted sessions. The gaps are in the **operational hardening layer**: security headers, field encryption, TLS termination, and brute-force protection. The 5 blockers above are all fixable within a day or two of focused work.
