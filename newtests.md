# Phase 3 Testing Insights & Manual Verifications

Internal notes. Phase 3 test coverage map, known gaps, and manual smoke tests to run before tagging a release or deploying Cloud.

Commits in scope: 835fe2a, cb38f10, 3985961, 555f4f6, 68fa1c7, 59fca66, 4ac4aa3, d42e37a.

---

## 1. Automated coverage map

| Layer | Tag | Location | Runtime |
|-------|-----|----------|---------|
| Unit | (none) | `internal/auth/jwt/manager_test.go`, `internal/auth/redirect/redirect_test.go`, `internal/rbac/org_rbac_test.go`, `internal/storage/applications_test.go`, `internal/api/middleware/auth_jwt_test.go` | ~2s |
| Integration | `integration` | `internal/api/org_rbac_integration_test.go`, `internal/api/oauth_redirect_integration_test.go`, `cmd/shark/cmd/app_test.go` | ~10s |
| E2E golden paths | `e2e` | `internal/testutil/e2e/phase3_test.go` (GP1-GP10) | ~30s |

Run: `make verify` — runs all three tiers with `-race`. Zero failures, zero races.

## 2. Per-feature coverage

### JWT subsystem
- Issue (session + access/refresh), validate (valid/expired/alg-confusion/unknown-kid/bad-sig), refresh rotation, revoke with check_per_request, key rotation (2-key JWKS).
- **Not covered**: concurrent EnsureActiveKey races (two processes booting same DB). Relies on partial unique index + INSERT OR IGNORE semantics — **manual verify needed**.
- **Not covered**: JWKS cache behavior under client load (Cache-Control: public, max-age=300). Stateless, cheap — unlikely to matter.

### Middleware dual-accept
- Bearer-valid, Bearer-invalid no-fallthrough, Bearer-expired, refresh-as-bearer rejected (with `error_description`), cookie-valid, no-auth, both-present (Bearer wins).
- **Not covered**: concurrent Bearer + stale cookie session revocation — sessions table + JTI blacklist are independent so not a correctness risk but worth a sanity check.
- **Not covered**: Authorization header with leading spaces / wrong case `bearer` vs `Bearer`. RFC 6750 mandates case-sensitive `Bearer`. Implementation uses `strings.HasPrefix(auth, "Bearer ")` — case-sensitive. Good.

### Org RBAC
- HasOrgPermission wildcards, no-role → ErrNotMember, custom role match, SeedOrgRoles idempotent, builtin delete rejected, grant/revoke round-trip.
- Integration: TestOrgRBAC_FullFlow — create org → 3 seeded roles → custom editor role → PATCH succeeds → revoke → PATCH 403.
- **Not covered**: race between two users inviting same email concurrently to same org. Existing invitation flow not touched by Phase 3.
- **Not covered**: last-owner guard interaction with org_user_roles sync in handleUpdateOrganizationMemberRole — sync is best-effort and swallows failures. **Manual verify needed.**

### Applications + redirect validator
- 10 validator cases: exact, wildcard subdomain, path injection rejected, loopback any-port, loopback not-allowlisted rejected, bad scheme, userinfo, fragment, normalization lowercase, trailing slash.
- CRUD + partial unique index (two is_default=1 → constraint error).
- Integration: OAuth callback with allowlisted URI (302), with non-allowlisted (400).
- **Not covered**: race between two boots seeding default app concurrently. Relies on partial unique index; loser should catch constraint error and re-fetch. **Manual verify.**

### CLI
- `TestE2E_AppCreate` (row in DB after CLI run)
- `TestE2E_AppRotateSecret` (hash changes)
- `TestE2E_AppDeleteDefault_Refused` — **partial**: asserts the guard condition (`IsDefault == true` on lookup) not the actual `os.Exit(1)` because exiting kills the test process.
- **Not covered**: `shark app update --add-callback` + `--remove-callback` in same invocation, `shark app show` with client_id vs id, `shark keys generate-jwt` without running server.
- **Not covered**: `shark app create` URL validation rejecting `javascript:`, `file:`, empty scheme.

### JWKS endpoint
- GP10: rotate → 2 keys. Indirectly tested via middleware validation (issued token, validated via stored public key).
- **Not covered**: JWKS structure against an external JWT validator (e.g. jose.Parse in jwx lib, node-jose, python-jwt). Critical for interop — **manual verify required before Phase 6**.

### Audit logs
- Drive-by wiring on RBAC + apps + org.member.role_update. Not E2E tested end-to-end. Spot-check required.

## 3. Known gaps worth filling (not blockers for Phase 4)

1. External JWKS interop test (spin up a test resource server in another language, have it fetch /.well-known/jwks.json and validate a shark-issued token).
2. Concurrent-boot race tests for EnsureActiveKey + seedDefaultApplication (run two `go test -run X -race -count=10` in parallel).
3. Session JWT TTL hardcoded to 30d in `IssueSessionJWT` — not wired to `cfg.Auth.SessionLifetimeDuration()`. See CHANGELOG.internal.md §15. Needs a unit test once fixed.
4. Refresh token rotation under concurrent use (same refresh token submitted twice in quick succession) — current logic inserts old JTI into revoked_jti before issuing new pair, but no lock. SQLite serializes writes so should be safe; add a TestRefresh_ConcurrentUseOnceEnforced to be sure.
5. HS256 alg-confusion test uses public-key bytes as HMAC secret — covered. But `alg:none` rejection path is covered only implicitly via `WithValidMethods`. Add explicit TestValidate_AlgNone.
6. Revocation TTL-only mode (check_per_request=false) — covered by GP4 indirectly. No negative test asserting that a revoked token STILL validates when check_per_request=false. Add one.
7. Deprecation warning behavior: `social.redirect_url` + `magic_link.redirect_url` still populate default app on first boot. No test asserts the migration. Add one: set cfg values → boot → assert default app's allowed_callback_urls contains both.

## 4. Manual verifications (pre-release smoke)

Run against a fresh `shark init && shark serve` — single-node SQLite, default config.

### Boot & bootstrap
- [ ] First `shark serve` prints: admin key banner, default app banner (client_id + secret). Both shown once. Secret matches `shark_app_` + 21 chars client_id, secret ~43 chars base62.
- [ ] `jwt_signing_keys` has exactly 1 active row (`sqlite3 sharkauth.db "SELECT kid, status FROM jwt_signing_keys"`).
- [ ] `applications` has exactly 1 row with `is_default=1`.
- [ ] `roles` has global default roles (user, admin) — confirms SeedDefaultRoles ran.
- [ ] Restart server. No second banner prints. Keys + app unchanged.

### JWT flow
- [ ] `curl -X POST http://localhost:8080/api/v1/auth/signup -d '{"email":"x@x.com","password":"passwordxxx"}'` — response body includes `token`. `Set-Cookie: shark_session=...` also present.
- [ ] `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/auth/me` → 200 with user.
- [ ] Same call without token, with cookie only → 200.
- [ ] With both bearer + cookie → 200, unchanged identity.
- [ ] Malformed bearer (`Bearer xxx`) → 401 with `WWW-Authenticate: Bearer error="invalid_token"`. Cookie ignored even if present.
- [ ] Refresh-token-as-bearer (issue pair via `auth.jwt.mode: access_refresh`, use refresh as Bearer) → 401 with `error_description="refresh token cannot be used as access credential"`.

### JWKS
- [ ] `curl http://localhost:8080/.well-known/jwks.json` → JSON `{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":"...","n":"...","e":"AQAB"}]}`. `Content-Type: application/json`, `Cache-Control: public, max-age=300`.
- [ ] Paste JWKS + issued token into https://jwt.io or a jose validator. Signature verifies. Claims parseable.
- [ ] `shark keys generate-jwt --rotate` (against running DB — server must be stopped first OR run against the same DB file which is safe). Restart. JWKS returns 2 keys. Tokens issued before rotation still validate (old kid lookup hits retired row within the window).

### Revocation
- [ ] Set `auth.jwt.revocation.check_per_request: true` in YAML, restart. `POST /api/v1/auth/revoke {"token":"..."}` → 200. Next `/me` with same bearer → 401.
- [ ] Toggle back to `false`, restart. Revoked token still works (TTL-only mode). Confirms flag respected.
- [ ] Admin revoke: `curl -H "Authorization: Bearer sk_live_..." -X POST /api/v1/admin/auth/revoke-jti -d '{"jti":"...","expires_at":"2030-01-01T00:00:00Z"}'` → 200.

### Org RBAC
- [ ] User A creates org `/organizations`. Response includes org. `organization_members` has A as owner. `org_user_roles` has A → owner role.
- [ ] User A creates custom role: `POST /organizations/{id}/roles {"name":"editor","description":"..."}` → 200. `org_roles` row exists, `is_builtin=0`.
- [ ] Attach permission: `PATCH /organizations/{id}/roles/{role_id}` with `attach_permissions: [{"action":"org","resource":"update"}]`. Confirm via DB.
- [ ] Invite user B, B accepts. Grant editor to B: `POST /organizations/{id}/members/{b_id}/roles/{role_id}` → 200.
- [ ] B calls `PATCH /organizations/{id}` with new name → 200. (Without editor → 403.)
- [ ] `GET /organizations/{id}/members/{b_id}/permissions` → list includes `{"action":"org","resource":"update"}`.
- [ ] Try to delete builtin role (owner/admin/member) → 409.
- [ ] Audit logs have entries for every mutation: `sqlite3 sharkauth.db "SELECT action FROM audit_logs WHERE action LIKE 'org.%' OR action LIKE 'rbac.%'"`.

### Applications & redirect allowlist
- [ ] `shark app create --name test --callback https://ok.example.com/cb` → exit 0, prints client_id + secret.
- [ ] `shark app list` → 2 rows (default + test).
- [ ] `shark app show <client_id>` → detail without secret hash.
- [ ] `shark app rotate-secret <client_id>` → new secret printed. `client_secret_hash` in DB differs.
- [ ] `shark app delete <default_client_id>` → exit 1, message "cannot delete default".
- [ ] Trigger a magic-link flow with `redirect_url` query param set to an allowed URL → redirects. With a non-allowed URL → 400 with "not allowed" in body. **Confirm there's no redirect to the bad URL** (check `Location` header absent in 400 response).
- [ ] Attempt `?redirect_url=javascript:alert(1)` → 400.
- [ ] Attempt `?redirect_url=https://ok.example.com/cb#frag` → 400 (fragment banned per OAuth 2.1 §3.1.2).
- [ ] Attempt `?redirect_url=https://user:pass@ok.example.com/cb` → 400 (userinfo banned).

### Wildcard & loopback
- [ ] Register app with `--callback https://*.preview.vercel.app` → attempt redirect to `https://abc.preview.vercel.app` → accepted.
- [ ] Same wildcard → `https://abc.def.preview.vercel.app` → rejected (nested label).
- [ ] Register loopback: `--callback http://127.0.0.1` → `?redirect_url=http://127.0.0.1:54321/cb` → accepted (RFC 8252 any port).
- [ ] Without loopback in allowlist → `?redirect_url=http://127.0.0.1:8080` → rejected.

### Admin HTTP apps API
- [ ] `POST /api/v1/admin/apps` with `Authorization: Bearer sk_live_...` → 201 with client_secret in response (shown once).
- [ ] `GET /api/v1/admin/apps` → list.
- [ ] `PATCH /api/v1/admin/apps/{id}` full array replacement of callbacks → 200. `updated_at` changes.
- [ ] `POST /api/v1/admin/apps/{id}/rotate-secret` → 200 with new secret.
- [ ] Every mutation writes an `app.*` audit row.

### Regression smoke
- [ ] Pre-Phase-3 flows still work: signup, login, /me, logout, password reset, MFA enrollment+challenge, passkey register+auth, SSO callback.
- [ ] Existing session cookie behavior unchanged: Set-Cookie, SameSite=Lax, HttpOnly, Secure iff base_url https.
- [ ] Existing org endpoints still work with cookie auth (middleware refactor didn't break).
- [ ] Dev mode: `shark serve --dev` still boots, `--reset` wipes dev.db, dev inbox captures emails.

## 5. Post-merge checks for Phase 4 dashboard devs

When the dashboard starts consuming these endpoints:
- Confirm cookie path still works for browser context (dashboard uses cookie, not Bearer).
- Dashboard fetching `/api/v1/admin/apps` and `/api/v1/organizations/{id}/roles` — ensure CORS preflight passes with `Authorization` header present (already allowed in cors.go).
- Real relying party (dashboard) registered via `shark app create --callback http://localhost:5173/auth/callback` for local dev, then `https://dashboard.shark.email/auth/callback` for production.

## 6. Before Cloud fork consumes this

- JWKS cross-validated by a Node.js or Python resource server (see gap 1 in §3).
- `applications` schema reviewed for Cloud multi-tenant — Cloud uses db-per-client so single-tenant schema holds. Verify no hidden assumptions.
- Deprecation removal ticket filed for `social.redirect_url` + `magic_link.redirect_url` (Phase 6 target).

## 7. Test commands cheat sheet

```
# fast loop during development
go test ./internal/auth/jwt/... ./internal/auth/redirect/... ./internal/rbac/...

# full unit pass
go test -race -count=1 ./...

# unit + integration
go test -race -count=1 -tags=integration ./...

# full verify (CI gate)
make verify

# one GP path in isolation
go test -race -tags=e2e -run TestPhase3_GoldenPaths/GP4 ./internal/testutil/e2e/...
```
