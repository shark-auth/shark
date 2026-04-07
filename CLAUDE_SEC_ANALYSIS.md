 SharkAuth Security Audit Report

  ---
  STRENGTHS

  ┌─────┬──────────────────┬───────────────────────────────────────────────────────────────┐   
  │  #  │       Area       │                            Details                            │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │     │                  │ Argon2id with configurable params (64MB memory, 3 iterations, │   
  │ 1   │ Password Hashing │  16-byte salt). Supports bcrypt migration with auto-rehash.   │   
  │     │                  │ Constant-time comparison via subtle.ConstantTimeCompare()     │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │     │                  │ Encrypted cookies via gorilla/securecookie (AES-256 +         │   
  │ 2   │ Session Security │ HMAC-SHA256). HttpOnly, SameSite=Lax, conditional Secure      │   
  │     │                  │ flag. No JWT (avoids common JWT pitfalls)                     │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │     │                  │ crypto/rand used everywhere. Zero usage of md5, sha1, or      │   
  │ 3   │ Crypto Hygiene   │ math/rand for security. Rejection sampling eliminates modulo  │   
  │     │                  │ bias in recovery codes                                        │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │     │ Timing Attack    │ subtle.ConstantTimeCompare() on: admin key, OAuth state, API  │   
  │ 4   │ Protection       │ keys, password verification, magic link tokens. Dummy bcrypt  │   
  │     │                  │ comparison when no recovery codes found                       │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │ 5   │ SQL Injection    │ 100% parameterized queries throughout sqlite.go. Dynamic      │   
  │     │ Protection       │ audit log filters use ? placeholders with args array          │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │     │ API Key          │ SHA-256 hashed in DB (never stored plaintext). Full key shown │   
  │ 6   │ Management       │  only once at creation. Revocation, expiration, scope         │   
  │     │                  │ checking, per-key rate limiting                               │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │     │                  │ Comprehensive event trail (actor, action, target, IP, user    │   
  │ 7   │ Audit Logging    │ agent, metadata). Retention policies with background cleanup. │   
  │     │                  │  Export endpoint                                              │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │     │                  │ IP-based token bucket globally + per-API-key rate limits.     │   
  │ 8   │ Rate Limiting    │ Magic link throttled to 1/60s. Returns proper 429 +           │   
  │     │                  │ Retry-After                                                   │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │     │ MFA              │ TOTP via pquerna/otp. 10 recovery codes with bcrypt hashing.  │   
  │ 9   │ Implementation   │ Session-based MFA state with atomic upgrades. Single-use      │   
  │     │                  │ enforcement                                                   │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │ 10  │ Error Disclosure │ Login errors return generic "Invalid email or password" —     │   
  │     │                  │ doesn't leak user existence                                   │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │ 11  │ HTTP Server      │ ReadTimeout, WriteTimeout, IdleTimeout all configured.        │   
  │     │                  │ Graceful shutdown with 30s drain                              │   
  ├─────┼──────────────────┼───────────────────────────────────────────────────────────────┤   
  │ 12  │ Config           │ All production secrets via ${ENV_VAR} interpolation.          │   
  │     │ Architecture     │ SHARKAUTH_ prefix with double-underscore nesting              │   
  └─────┴──────────────────┴───────────────────────────────────────────────────────────────┘   

  ---
  WEAKNESSES

  #: 1
  Severity: HIGH
  Area: Error Leakage (SSO)
  Finding: Raw err.Error() from SAML/OIDC libraries returned to clients
  Location: sso_handlers.go:67,78,94,117,133,152,171,193
  ────────────────────────────────────────
  #: 2
  Severity: MEDIUM
  Area: OAuth State Cookie
  Finding: Missing Secure flag on OAuth state cookie (short-lived but still a gap)
  Location: oauth_handlers.go:47-54
  ────────────────────────────────────────
  #: 3
  Severity: MEDIUM
  Area: Admin Bypass in Dev
  Finding: Empty admin.api_key config disables admin auth entirely — risky if accidentally     
    deployed
  Location: middleware/admin.go:15
  ────────────────────────────────────────
  #: 4
  Severity: LOW
  Area: OIDC State Store
  Finding: In-memory map for OIDC state — won't survive restarts and won't scale to multiple   
    instances
  Location: sso/oidc.go:17-22
  ────────────────────────────────────────
  #: 5
  Severity: LOW
  Area: Integer Overflow (gosec)
  Finding: len(hash) cast to uint32 without bounds check
  Location: password.go:61, passkey.go:439

  ---
  BLOCKERS (Must Fix Before Launch)

  #: 1
  Issue: No Security Headers
  Impact: Clickjacking, MIME sniffing, no HSTS, no CSP. Fails basic security scans
  Fix: Add middleware: X-Frame-Options: DENY, X-Content-Type-Options: nosniff,
    Strict-Transport-Security, Content-Security-Policy, Referrer-Policy
  ────────────────────────────────────────
  #: 2
  Issue: Secrets Stored Plaintext in DB
  Impact: SSO oidc_client_secret, saml_idp_cert, and mfa_secret stored unencrypted in SQLite   
  Fix: Implement field-level AES-256-GCM encryption for these columns using the server secret  
  ────────────────────────────────────────
  #: 3
  Issue: No TLS in Application
  Impact: Server uses http.ListenAndServe only — no HTTPS. Cookie Secure flag depends on config

    URL, not actual TLS
  Fix: Either add ListenAndServeTLS() with cert config, or document that a TLS-terminating     
    reverse proxy (Caddy/Nginx) is required
  ────────────────────────────────────────
  #: 4
  Issue: Docker Runs as Root
  Impact: No USER directive in Dockerfile — container runs as UID 0
  Fix: Add RUN addgroup -g 1000 shark && adduser -D -u 1000 -G shark shark + USER shark        
  ────────────────────────────────────────
  #: 5
  Issue: No Account Lockout
  Impact: No failed login attempt tracking. Brute force possible within rate limit budget (100 
    req/s)
  Fix: Implement per-email failed attempt counter with progressive lockout (e.g., 5 failures = 
    15min lock)

  ---
  UPGRADES (Should Fix, Not Blocking)

  ┌─────┬──────────────────┬──────────────────────┬───────────────────────────────────────┐    
  │  #  │       Area       │    Current State     │          Recommended Upgrade          │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │     │                  │ Min 8 chars, no      │ Add: uppercase + lowercase + digit    │    
  │ 1   │ Password Policy  │ complexity rules     │ required, or use zxcvbn entropy       │    
  │     │                  │                      │ scoring                               │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │     │                  │ Relies on            │ Add explicit CSRF tokens for any      │    
  │ 2   │ CSRF Protection  │ SameSite=Lax only    │ form-based flows (or document         │    
  │     │                  │                      │ API-only usage)                       │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │     │ Auth-specific    │ Generic IP-based     │ Add per-email rate limit on           │    
  │ 3   │ Rate Limiting    │ rate limit covers    │ /auth/login and /auth/signup (e.g., 5 │    
  │     │                  │ login                │  attempts/email/15min)                │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │     │                  │ Magic link and SAML  │                                       │    
  │ 4   │ Redirect URL     │ redirect URLs not    │ Validate all redirect URLs against    │    
  │     │ Validation       │ validated against    │ configured base_url domain            │    
  │     │                  │ allowlist            │                                       │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │     │ Recovery Code    │ 8-char codes from    │ Increase to 10 chars or add mixed     │    
  │ 5   │ Entropy          │ 36-char alphabet     │ case (~47-52 bits)                    │    
  │     │                  │ (~41 bits entropy)   │                                       │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │     │ Session          │                      │ Invalidate all other sessions when    │    
  │ 6   │ Revocation on    │ Not verified         │ password changes or MFA is disabled   │    
  │     │ Password Change  │                      │                                       │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │ 7   │ Docker Health    │ No HEALTHCHECK       │ Add HEALTHCHECK CMD wget -q --spider  │    
  │     │ Check            │ instruction          │ http://localhost:8080/healthz         │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │     │ Suspicious       │ No email             │ Send email on: login from new         │    
  │ 8   │ Activity Alerts  │ notifications        │ IP/device, MFA disabled, password     │    
  │     │                  │                      │ changed                               │    
  ├─────┼──────────────────┼──────────────────────┼───────────────────────────────────────┤    
  │ 9   │ SSO Error        │ Raw library errors   │ Wrap with generic messages, log       │    
  │     │ Wrapping         │ exposed              │ originals server-side                 │    
  └─────┴──────────────────┴──────────────────────┴───────────────────────────────────────┘    

  ---
  MISSING FOR LAUNCH

  ┌─────┬────────────────────┬──────────────────────────────────────────────────┬──────────┐   
  │  #  │      Category      │                  What's Missing                  │ Priority │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 1   │ Password Reset     │ No explicit password reset endpoint found —      │ P0       │   
  │     │ Flow               │ magic links handle login but not password change │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 2   │ Security Headers   │ Zero OWASP-recommended response headers set      │ P0       │   
  │     │ Middleware         │                                                  │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 3   │ Field-Level        │ MFA secrets, SSO credentials stored in cleartext │ P0       │   
  │     │ Encryption         │  DB                                              │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 4   │ Non-Root Docker    │ Production container runs as root                │ P1       │   
  │     │ User               │                                                  │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 5   │ Account Lockout    │ Brute force protection beyond IP rate limiting   │ P1       │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 6   │ Password           │ Only length check, no entropy/complexity         │ P1       │   
  │     │ Complexity         │ validation                                       │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 7   │ CORS Documentation │ No guidance on what origins to configure for     │ P2       │   
  │     │                    │ production                                       │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 8   │ Secret Rotation    │ No documented procedure for rotating             │ P2       │   
  │     │ Docs               │ SHARKAUTH_SECRET or admin key                    │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 9   │ Dependency Audit   │ No go audit or vulnerability scanning in CI      │ P2       │   
  │     │                    │ pipeline                                         │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 10  │ User Deletion/GDPR │ DELETE /users/{id} endpoint not implemented      │ P2       │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 11  │ Login Anomaly      │ No tracking of new devices, IPs, or geolocations │ P3       │   
  │     │ Detection          │                                                  │          │   
  ├─────┼────────────────────┼──────────────────────────────────────────────────┼──────────┤   
  │ 12  │ API Versioning     │ Currently /api/v1 only — no deprecation or       │ P3       │   
  │     │ Strategy           │ migration plan documented                        │          │   
  └─────┴────────────────────┴──────────────────────────────────────────────────┴──────────┘   

  ---
  Bottom line: The core crypto and auth logic is excellent — Argon2id, constant-time
  comparisons, proper CSPRNG usage, encrypted sessions. The gaps are in the operational        
  hardening layer: security headers, field encryption, TLS termination, and brute-force        
  protection. The 5 blockers above are all fixable within a day or two of focused work. 