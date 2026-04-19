# Shark Smoke Test

Post-build verification. Exercises every feature against a real running binary. Updated through Phase 5 Wave C (OAuth 2.1 + Agent Auth).

## Prereqs

- `jq`, `curl`, `sqlite3` in `$PATH` (cross-platform port kill supports Windows `taskkill`, Linux `fuser`, macOS `lsof`)
- Binary built: `go build -ldflags="-s -w" -trimpath -o shark ./cmd/shark`
- No server running on `:8080`
- Fresh DB (script handles this)

## Run

```bash
./smoke_test.sh
```

Exits 0 on all-pass, non-zero on first failure. Colored PASS/FAIL per section.

**Current: 91 PASS, 0 FAIL.**

## What it covers

| # | Section | Verifies |
|---|---------|----------|
| 1 | Bootstrap | First-boot banners: admin API key + default app |
| 2 | JWT at signup | Response includes `token`; RS256; cookie set |
| 3 | Dual-accept middleware | Bearer /me 200, cookie /me 200, both 200, garbage Bearer + valid cookie → 401 (no-fallthrough) |
| 4 | JWKS endpoint | `/.well-known/jwks.json` returns RS256 key (session JWT) + ES256 key (OAuth 2.1); kty/alg/use correct; kid matches token header |
| 5 | User revoke | `POST /api/v1/auth/revoke` → 200; TTL-only behavior when `check_per_request=false` |
| 6 | Admin revoke | `POST /api/v1/admin/auth/revoke-jti` admin-gated; non-admin → 401 |
| 7 | Key rotation | `shark keys generate-jwt --rotate`; JWKS returns ≥3 keys (old RS256 + new RS256 + ES256); old token still validates |
| 8 | Apps CLI | `shark app create/list/show/rotate-secret/delete`; default-delete refused |
| 9 | Admin apps HTTP | POST/GET/PATCH/DELETE /api/v1/admin/apps/*; secret shown once |
| 10 | Redirect allowlist | Magic-link with allowed URL → 302; not allowlisted → 400; `javascript:` → 400 |
| 11 | Org RBAC | Create org seeds 3 builtins; custom role grants access; revoke removes access; builtin delete → 409 |
| 12 | Audit logs | Mutations land rows in `audit_logs` with expected action strings |
| 13 | Regression | `/auth/logout`, `/healthz`, legacy cookie path |
| 14 | Admin system endpoints | test-email, purge-expired-sessions, purge audit logs, oauth-accounts, passkeys, rotate-signing-key |
| 15 | User list filters | `?mfa_enabled=false`, `?email_verified=true` |
| 16 | Sessions (self-service) | `GET /api/v1/auth/sessions` returns user's own sessions with `current:true` flag |
| 17 | Admin sessions | `GET /api/v1/admin/sessions` with joined user_email |
| 18 | Stats + Trends | `GET /api/v1/admin/stats`, `GET /api/v1/admin/stats/trends?days=N` |
| 19 | Webhooks CRUD | create (201 + secret once), list, test fire, delete |
| 20 | API Key CRUD | create (201 + full key once), list, revoke |
| 21 | User CRUD (admin) | list, get, update |
| 22 | Dev Inbox | `GET /api/v1/admin/dev/emails` — available only when `--dev` |
| 23 | Password change | authenticated change with current_password verification |
| 24 | SSO Connections | create OIDC connection, list, delete (admin) |
| 25 | Admin Config + Health | `GET /api/v1/admin/health`, `GET /api/v1/admin/config` |
| 26 | OAuth 2.1 AS Metadata (RFC 8414) | `/.well-known/oauth-authorization-server` — issuer, authorization_endpoint, token_endpoint, registration_endpoint, PKCE S256, client_credentials + device_code grants |
| 27 | OAuth 2.1 tables | agents, oauth_authorization_codes, oauth_tokens, oauth_consents, oauth_device_codes, oauth_dcr_clients exist |

## Notes

- Random suffix in emails allows repeat runs.
- Each test has `fail "<reason>"` on unexpected output.
- Cross-platform port kill handles Windows (taskkill via netstat), Linux (fuser), macOS (lsof).
- `--dev` mode auto-selects dev email provider; `dev.db` used as storage path.
- `relogin` helper re-authenticates after server restarts (dev mode regenerates secret).

## Coverage to add next

**Wave D (OAuth 2.1 advanced grants):**
- Dynamic Client Registration (POST /oauth/register) — RFC 7591
- Device Authorization Grant — POST /oauth/device + polling
- Token Exchange — `urn:ietf:params:oauth:grant-type:token-exchange`
- Agent CRUD API (POST/GET/PATCH/DELETE /api/v1/agents + tokens + audit)
- Consent management (GET/DELETE /api/v1/auth/consents)

**Wave E (security hardening):**
- DPoP proof validation (RFC 9449)
- Token Introspection (RFC 7662)
- Token Revocation (RFC 7009)
- Resource Indicators (RFC 8707) — audience binding

**Wave F (dashboard + final tests):**
- OAuth 2.1 end-to-end flows: auth code + PKCE full flow, client_credentials, refresh token rotation + reuse detection
- ES256 JWT verification via JWKS
- Consent HTML page renders + decision flow

## When to run

- Before every release tag
- After any touch to `internal/auth/jwt`, `internal/api/middleware/auth.go`, `internal/auth/redirect`, `internal/rbac`, `internal/storage/applications.go`, `internal/server/server.go`, `internal/oauth/*`
- After Phase 4+ merges

## Known gaps

- No external JWKS interop (use jwt.io + foreign-lang resource server)
- No concurrent-boot race test
- No clock-skew / expired-token test (would need time travel)
- OAuth 2.1 authorize flow end-to-end not yet covered (needs Wave C consent templates + Wave D grants fully wired)
