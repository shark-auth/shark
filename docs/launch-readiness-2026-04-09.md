# SharkAuth Launch Readiness Report

**Date:** 2026-04-09
**Target window:** April 13-16, 2026
**Branch:** `feat/admin-key-m2m`

---

## Verdict

Backend is shippable with 2 days of fixes. Full launch (dashboard + SDKs) is not realistic for April 13-16.

---

## P0 — Must fix before any deploy (~1 day)

| Issue | Effort | Details |
|-------|--------|---------|
| Migration 00002 breaks fresh installs | 30 min | `key_suffix` is in both `00001_init.sql` and `00002_api_key_suffix.sql` — any new deploy crashes on startup |
| Empty `server.secret` accepted silently | 30 min | No validation — sessions become insecure with empty HMAC key |
| Rate limiter uses `r.RemoteAddr` behind proxy | 1 hr | All clients share one bucket behind Caddy/nginx — rate limiting is broken. Fix: read `X-Real-Ip` after `RealIP` middleware |
| No Dockerfile | 30 min | Can't deploy to any container platform (Fly.io, Railway, Render, etc.) |

## P1 — Should fix for v1 credibility (~1 day)

| Issue | Effort | Details |
|-------|--------|---------|
| `GET /users`, `GET /users/{id}`, `DELETE /users/{id}` handlers | 2 hr | Store layer is ready, just missing handlers — returns 501 today |
| `PATCH /users/{id}` (admin update user) | 1 hr | Can't update user name/email/metadata from admin side |
| `DELETE /auth/me` (user self-deletion) | 1 hr | GDPR compliance — users can't delete their own account |
| Email verification for password signups | 3 hr | `verify_email.html` template exists, no endpoints — password users stay `emailVerified: false` forever |
| Missing DB indexes | 1 hr | `magic_link_tokens(token_hash)`, `sessions(user_id)`, `sessions(expires_at)`, `user_roles(user_id)`, `passkey_credentials(user_id)`, `oauth_accounts(user_id)` |
| Post-OAuth redirect to frontend | 1 hr | `handleOAuthCallback` returns JSON, never redirects — every deploy needs a Caddy/nginx workaround. Add `social.redirect_url` config |

## P2 — Nice for launch, not blocking

| Issue | Effort | Details |
|-------|--------|---------|
| `http.MaxBytesReader` on JSON endpoints | 1 hr | No request body size limits — any endpoint accepts unbounded bodies |
| Structured logging | 2 hr | Swap stdlib `log` for `slog` with `JSONHandler` — makes log aggregation usable |
| Deep health check | 30 min | `/healthz` doesn't ping the DB — liveness != readiness |
| Middleware test coverage | 3 hr | 5 middleware files (~500 LOC) with zero tests — guards all protected routes |
| Makefile / CI workflow | 1 hr | No `.github/workflows/`, no `Makefile`, no automated tests on push |
| Dead code cleanup | 15 min | Legacy `RequireSession` in `middleware/auth.go` is a no-op stub |

---

## What's solid

- **Tests pass.** 6 packages, all green. Coverage on auth, handlers, RBAC, SSO, audit, API keys.
- **Security patterns are strong.** Constant-time compares everywhere, anti-enumeration on login/magic-link/reset, Argon2id hashing, bcrypt migration path, SHA-256 API key storage.
- **All auth flows work end-to-end.** Password, OAuth (Google/GitHub/Apple/Discord), passkeys (WebAuthn/FIDO2), magic links, TOTP MFA with recovery codes, SAML/OIDC SSO.
- **M2M API key system is well-designed.** `sk_live_` prefix, auto-generation on first run, scopes, rotation, revocation, can't revoke last admin key.
- **Admin features are complete.** RBAC (roles, permissions, wildcard matching), audit logs (cursor pagination, filtering, CSV export), SSO connection CRUD, API key CRUD.
- **Infrastructure basics are correct.** Graceful shutdown with 30s timeout, HTTP timeouts (read 15s, write 15s, idle 60s), config interpolation with `${ENV_VAR}` + `SHARKAUTH_` env overrides.
- **Auth methods are composable.** OAuth-only users can add passwords/passkeys later. Password reset works on OAuth accounts. All methods are additive, never exclusive. Matches Auth0/Clerk behavior.

---

## API surface vs Auth0/Clerk

### Implemented

| Area | Endpoints |
|------|-----------|
| Password auth | signup, login, logout, me, change password |
| Password reset | send reset link, reset with token |
| OAuth | start + callback for Google, GitHub, Apple, Discord |
| Passkeys | register begin/finish, login begin/finish, list/delete/rename credentials |
| Magic links | send, verify (with redirect) |
| MFA (TOTP) | enroll, verify, challenge, recovery, disable, view recovery codes |
| SSO | SAML ACS + metadata, OIDC auth + callback, connection CRUD, domain auto-routing |
| RBAC | role/permission CRUD, user-role assignment, permission check |
| M2M API keys | create, list, get, update, revoke, rotate |
| Audit logs | query (cursor pagination), get by ID, CSV export |
| User-role management | assign/remove roles, list roles, effective permissions, user audit trail |

### Missing — needed for v1

| Endpoint | Notes |
|----------|-------|
| `GET /users` | Store has `ListUsers` with search/pagination — no handler |
| `GET /users/{id}` | Store has `GetUserByID` — no handler |
| `DELETE /users/{id}` | Store has `DeleteUser` — no handler |
| `PATCH /users/{id}` | No handler, no route registered |
| `DELETE /auth/me` | User self-deletion (GDPR) |
| `POST /auth/email/verify/send` | Email verification for password signups |
| `GET /auth/email/verify` | Token verification endpoint |

### Missing — v2

| Feature | Notes |
|---------|-------|
| Organizations / multi-tenancy | Explicitly post-launch in PROJECT.md |
| Session listing/revocation by user | Store supports it, no API surface |
| Webhooks / event system | No infrastructure — blocks integrations (sync user to CRM, etc.) |
| Per-endpoint rate limit config | Hardcoded `100, 100` in `router.go` |
| `POST /users` (admin create user) | Server-side user creation without signup flow |
| Auth0 migration endpoint | Registered as 501 placeholder |
| Distributed rate limiting | In-memory map won't work multi-instance |

---

## Timeline assessment

| Deliverable | Estimated effort | Status |
|-------------|-----------------|--------|
| P0 + P1 backend fixes | 2 days | Not started |
| OAuth redirect config | 2 hrs | Not started |
| Admin dashboard | 1-2 weeks | Not started |
| TypeScript SDK | 3-5 days | Not started |
| Python SDK | 3-5 days | Not started |

### What can ship April 13-16

The backend API with P0/P1 fixes, Dockerfile, deployed and usable via raw HTTP or Postman. A legitimate "API launch" or "early access" release.

### What cannot ship April 13-16

Dashboard, TypeScript SDK, Python SDK. These are 2-3 weeks of work combined. The SDKs are what make shark usable for developers — without them you're shipping an API, not a product.

### Recommended release plan

| Version | Target | Scope |
|---------|--------|-------|
| **v0.1.0** | April 13-16 | Backend API — P0/P1 fixes, Dockerfile, deploy. "Early access / API-first" |
| **v0.2.0** | End of April | TypeScript SDK + dashboard MVP |
| **v0.3.0** | Mid May | Python SDK + organizations + webhooks |
