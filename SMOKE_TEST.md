# Shark Smoke Test

Post-build verification. Exercises every feature against a real running binary. Updated through Phase 5.5 (Token Vault — third-party OAuth connection storage).

## Prereqs

- `jq`, `curl`, `sqlite3`, `openssl` in `$PATH` (cross-platform port kill supports Windows `taskkill`, Linux `fuser`, macOS `lsof`)
- Binary built: `go build -ldflags="-s -w" -trimpath -o shark ./cmd/shark` (or `shark.exe` on Windows)
- No server running on `:8080`
- Fresh DB (script handles this)

## Run

```bash
./smoke_test.sh          # Linux/macOS
BIN=./shark.exe bash smoke_test.sh  # Windows (Git Bash / MSYS2)
```

Exits 0 on all-pass, non-zero on first failure. Colored PASS/FAIL per section.

**Current: 222 PASS, 0 FAIL.**

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
| 28 | AS metadata advanced (RFC 8414) | introspection_endpoint, revocation_endpoint, device_authorization_endpoint, token-exchange + authorization_code + refresh_token grants, response_type=code, dpop_signing_alg_values_supported (ES256) |
| 29 | Agent CRUD (admin API) | POST/GET/PATCH /api/v1/agents/* with client_secret shown once, prefix `shark_agent_`, audit log (agent.created + agent.updated), no-auth → 401, lookup by id OR client_id |
| 30 | Client Credentials grant | Fresh agent → `/oauth/token` via Basic auth issues bearer access_token with `expires_in>0`, scope echoed, wrong secret → 401, missing grant_type → 400 |
| 31 | Auth Code + PKCE flow | Consent HTML render, decision POST, redirect with `code` + state echoed. Token exchange noted (fosite Sanitize strips code_challenge end-to-end; covered by unit tests) |
| 32 | PKCE enforcement (OAuth 2.1) | Authorize without `code_challenge` → redirect with `error=invalid_request` |
| 33 | Refresh Token Rotation | Depends on §31; noted when PKCE path incomplete |
| 34 | Device Authorization Grant (RFC 8628) | `/oauth/device` issues device_code + user_code (XXXX-XXXX unambiguous charset) + interval≥5; immediate poll → authorization_pending; after approval via DB → bearer token issued |
| 35 | Token Exchange (RFC 8693) | Best-effort (requires JWT subject_token); CC tokens are opaque so note + defer to `exchange_test.go` |
| 36 | DPoP (RFC 9449) | No-DPoP still works (optional); garbage `DPoP:` header → 400 `invalid_dpop_proof`; metadata advertises ES256 |
| 37 | Token Introspection (RFC 7662) | Valid CC token → `active:true` with client_id/exp/scope; fake token → `active:false` |
| 38 | Token Revocation (RFC 7009) | `/oauth/revoke` → 200; revoked token introspects as `active:false`; invalid revoke → 200 per RFC |
| 39 | Dynamic Client Registration (RFC 7591/7592) | POST /oauth/register → 201 with `shark_dcr_` prefix + client_secret + registration_access_token; GET/PUT/DELETE with RAT; no-auth → 401; RAT after delete → 401 |
| 40 | Resource Indicators (RFC 8707) | `resource=…` param binds token audience; introspect reveals `aud`; no resource → different aud |
| 41 | ES256 JWKS | `/.well-known/jwks.json` includes ES256 key with kty=EC, crv=P-256, use=sig, x/y are 43-char base64url, kid present |
| 42 | Consent Management (self-service) | `GET /api/v1/auth/consents` (session-auth) → data array; DELETE consent → 200; removed on re-list; no-auth → 401 |
| 43 | Vault provider CRUD (admin) | POST `/api/v1/vault/providers` with `template=github` + client_id/secret → 201, id captured; GET list + GET by id return sanitized rows (no `client_secret`); PATCH `display_name` → 200; PATCH `client_secret` rotation → 200; duplicate name → 409; DELETE → 204, subsequent GET → 404; no-auth → 401 |
| 44 | Vault templates discovery | `GET /api/v1/vault/templates` → 200 with 9 built-in entries; each row has snake_case `name`/`display_name`/`auth_url`/`token_url`/`default_scopes`; no PascalCase leaks; `github` present |
| 45 | Vault connect flow (session auth) | Relogin, seed a provider, `GET /api/v1/vault/connect/{provider}` with session → 302 to provider's authorize URL with `client_id=` + `state=`; `shark_vault_state` CSRF cookie set; without session → 401 |
| 46 | Agent token retrieval (OAuth bearer) | `GET /api/v1/vault/{provider}/token` — missing bearer → 401 with `WWW-Authenticate: Bearer`; bogus bearer → 401 with `WWW-Authenticate`. Full happy path noted (requires mock upstream OAuth provider; covered by `internal/vault/vault_test.go`) |
| 47 | Vault connections list (session auth) | `GET /api/v1/vault/connections` with session → 200 `{data:[]}` for a fresh user; no session → 401; DELETE unknown id with session → 404 (IDOR-safe) |
| 48 | Audit events for vault ops | `GET /api/v1/audit-logs?action=vault.provider.created,vault.provider.updated,vault.provider.deleted&limit=200` → 200 with ≥1 of each action; unfiltered list has ≥3 `vault.*` events total |

## Notes

- Random suffix in emails allows repeat runs.
- Each test has `fail "<reason>"` on unexpected output.
- Cross-platform port kill handles Windows (taskkill via netstat), Linux (fuser), macOS (lsof).
- `--dev` mode auto-selects dev email provider; `dev.db` used as storage path.
- `relogin` helper re-authenticates after server restarts (dev mode regenerates secret).

## Coverage to add next

- Full PKCE end-to-end (blocked by fosite Sanitize stripping code_challenge in authorize storage — requires server-side fix or custom flow handler)
- DPoP signed proof JWT happy-path (needs ES256 JWT signing from bash; currently covered by `internal/oauth/dpop_test.go`)
- Token exchange end-to-end with JWT subject (covered by `internal/oauth/exchange_test.go`)
- Refresh token reuse detection end-to-end (depends on PKCE end-to-end)
- Vault `ExchangeAndStore` + `GetFreshToken` happy path (requires a mock upstream OAuth 2.0 provider; covered by `internal/vault/vault_test.go`)

## When to run

- Before every release tag
- After any touch to `internal/auth/jwt`, `internal/api/middleware/auth.go`, `internal/auth/redirect`, `internal/rbac`, `internal/storage/applications.go`, `internal/server/server.go`, `internal/oauth/*`, `internal/vault/*`
- After Phase 4+ merges

## Known gaps

- No external JWKS interop (use jwt.io + foreign-lang resource server)
- No concurrent-boot race test
- No clock-skew / expired-token test (would need time travel)
- OAuth 2.1 authorize flow end-to-end not yet covered (needs Wave C consent templates + Wave D grants fully wired)
