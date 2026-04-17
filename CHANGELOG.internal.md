# Shark Internal Changelog

> Not for public consumption. Technical notes on what shipped, why it shipped that way, and what trade-offs were made. Cross-reference with commit SHAs in repo.

## Phase 3 — JWT, Org RBAC, Applications, Redirect Allowlist
Shipped: 2026-04-17
Commits: 835fe2a, cb38f10, 3985961, 555f4f6, 68fa1c7, 59fca66, 4ac4aa3

### 1. JWT subsystem (`internal/auth/jwt/`)

Algorithm: RS256 only. `peekAlg()` manually decodes the JWT header BEFORE the library parser to reject `alg=none` and all HMAC variants (HS256/384/512) — guards against alg-confusion attacks per RFC 8725 §2.1. The `golang-jwt/jwt/v5` library is used for actual parsing, with `gojwt.WithValidMethods([]string{"RS256"})` as belt-and-suspenders.

Key storage: 2048-bit RSA keypairs. Private key is encoded as PKCS#8 PEM, then encrypted with AES-GCM before persisting to `jwt_signing_keys.private_key_pem`. The AES key is derived via `SHA-256(server_secret + "jwt-key-encryption")` — the domain separator `"jwt-key-encryption"` prevents key-reuse with session-cookie AES derivation. Nonce is prepended to ciphertext; result is base64-encoded. Plaintext PEM is zeroed out in memory after encryption.

KID strategy: `base64url(SHA-256(DER-encoded public key))[:16]`. Deterministic from the public key, no random suffix. Fits in HTTP headers, unique enough for the key-set sizes Shark operates at.

TTLs: stored as strings in `JWTConfig` (`AccessTokenTTL`, `RefreshTokenTTL`, `ClockSkew`) with accessor methods `AccessTokenTTLDuration()`, `RefreshTokenTTLDuration()`, `ClockSkewDuration()`. Defaults: 15m / 30d / 30s. This matches the existing `SessionLifetime` string-duration pattern already in the config.

`NewManager` takes 4 parameters: `cfg *config.JWTConfig`, `store storage.Store`, `baseURL string`, `serverSecret string`. The 4th param (`serverSecret`) is used exclusively for AES-GCM private-key encryption/decryption. It is not embedded in tokens.

`GenerateAndStore` is exported so the `shark keys generate-jwt` CLI can call it directly. `EnsureActiveKey` calls `GenerateAndStore(ctx, false)` if no active key exists — this is called by `server.Build()` so manual key generation is optional.

Token types: `Claims.TokenType` is `"session"` | `"access"` | `"refresh"`. Session mode issues a single JWT with `token_type=session` and session lifetime (30d hardcoded, matching `SessionManager`). Access-refresh mode issues an `access` (15m) + `refresh` (30d) pair.

Refresh one-time-use: `Manager.Refresh()` runs an unconditional revocation check (regardless of `cfg.Revocation.CheckPerRequest`), inserts the old refresh JTI into `revoked_jti` before issuing the new pair.

Sentinel errors: `ErrExpired`, `ErrInvalidSignature`, `ErrRevoked`, `ErrUnknownKid`, `ErrAlgMismatch`, `ErrRefreshToken`. `ErrRefreshToken` was added post-hoc to give the middleware a specific error to detect and surface as `error_description="refresh token cannot be used as access credential"` (§2.3). Without it the middleware would return a generic 401.

Functions `jwkFromPublicKey` and `pubKeyFromJWK` exist in both `internal/auth/jwt/keys.go` (unexported, used in tests) and `internal/api/well_known_handlers.go` (unexported, used by the JWKS endpoint). This duplication is intentional: the packages have different import paths and combining them would create a cycle. The `pubKeyFromJWK` function in `jwt/keys.go` is currently used only in tests; production code uses the stored PEM directly.

### 2. Middleware dual-accept (`internal/api/middleware/auth.go`)

`RequireSessionFunc(sm, jwtMgr)` replaces all call sites that previously used the old `RequireSession` stub (which only checked the cookie). 10 route-groups updated atomically in `router.go`.

Decision tree per RFC 6750: Bearer header present → validate JWT; **no fallthrough on failure** — a bad Bearer token returns 401 immediately, it does not fall through to cookie auth. This prevents a class of downgrade attacks where a stolen but invalid JWT still grants access via a valid cookie.

`AuthMethodKey` context key stores `"jwt"` or `"cookie"`. `GetAuthMethod(ctx)` and `GetClaims(ctx)` helpers added for handler code that needs to distinguish (e.g. logout, which revokes the JWT JTI if auth method is jwt).

`ErrRefreshToken` from the jwt package is checked explicitly; the middleware returns a `WWW-Authenticate` header with `error_description` to give API clients an actionable error.

Secondary token_type guard after `claims.TokenType == "refresh"` check remains as belt-and-suspenders; in practice `ErrRefreshToken` catches the case first.

Cookie path: `sm.GetSessionFromRequest(r)` → `sm.ValidateSession(ctx, sessionID)`. Unchanged from Phase 2.

MFA handlers (`/auth/mfa/challenge`, `/auth/mfa/recovery`) use `RequireSessionFunc` without `RequireMFA` — this is correct, they are the pre-MFA path.

### 3. JWKS endpoint (`internal/api/well_known_handlers.go`)

Mounted at `/.well-known/jwks.json` outside the `/api/v1` prefix, per RFC 8414. No auth.

Retired-key window: `2 × accessTokenTTL` — at 15m default, retired keys stay in JWKS for 30 minutes. In-flight access tokens signed with the old key remain verifiable during this window.

`Cache-Control: public, max-age=300` (5 minutes). Public caching is intentional: JWKS is static for the TTL window and resource servers / CDN edge validators can cache it aggressively.

`ListJWKSCandidates(ctx, false, retiredCutoff)` — first bool is `activeOnly`; false means include retired keys newer than the cutoff. Malformed keys (PEM parse failure) are skipped rather than causing a 500.

### 4. Revocation

`revoked_jti` table: `jti TEXT PRIMARY KEY`, `revoked_at`, `expires_at`. Indexed on `expires_at` for pruning.

TTL: callers pass `expires_at` = the token's own `ExpiresAt` claim. When the JWT itself would expire, the JTI row becomes eligible for pruning. `PruneExpiredRevokedJTI` is called lazily (on every `RevokeJTI` call and on every `Refresh` call) to prevent unbounded table growth — no separate cleanup goroutine needed.

Per-request check: `cfg.Auth.JWT.Revocation.CheckPerRequest` (default false). When true, every Bearer-authenticated request queries `revoked_jti`. Adds a DB round-trip per request; recommended only for high-security deployments. `SetCheckPerRequest(bool)` on the Manager enables runtime toggle (used in e2e tests, can support dynamic config reload later).

`POST /api/v1/auth/revoke`: returns 200 (not 204). The PHASE3.md plan said 204 but the implementation returns 200 with `{"message": "Token revoked"}` to give clients a parseable confirmation. GP4 e2e test documents this deviation.

`POST /api/v1/admin/auth/revoke-jti`: also returns 200. Admin path for revoking arbitrary JTIs by (jti, expires_at) without needing the full token string. Useful for server-side session termination in cloud scenarios.

### 5. Org RBAC (`internal/rbac/`, `internal/storage/org_rbac.go`)

Schema design: three parallel tables (`org_roles`, `org_role_permissions`, `org_user_roles`) that are completely separate from the global RBAC tables (`roles`, `permissions`, `role_permissions`, `user_roles`). No nullable `org_id` column on global tables — clean separation avoids NULL-check complexity and prevents accidental cross-scope queries.

`org_roles`: `id TEXT PRIMARY KEY` (prefix `orgrole_<nanoid>`), `organization_id` FK → `organizations(id) ON DELETE CASCADE`, `UNIQUE(organization_id, name)` constraint.

`org_role_permissions`: `(org_role_id, action, resource)` composite PK. Wildcard matching (`users:*`) is evaluated in Go, not SQL — same `rbac.matchPermission` function as the global RBAC.

`org_user_roles`: `(organization_id, user_id, org_role_id)` composite PK. `granted_by` FK with `ON DELETE SET NULL`.

`SeedOrgRoles` idempotency: uses application-layer `GetOrgRoleByName` + `CreateOrgRole` rather than `INSERT OR IGNORE` because role IDs are Go-generated nanoids. `INSERT OR IGNORE` would silently discard the new nanoid and leave the existing row, which is correct functionally but loses the ability to detect first-boot vs re-seed. The app-layer check is explicit.

`RequireOrgPermission` middleware mount point: `{id}` URL param (not `{org_id}`) for chi compatibility. The middleware reads `{id}` as a fallback when `{org_id}` is absent. This keeps backward compatibility with the existing organization routes that used `{id}`.

Error mapping: `ErrNotMember` → 404 (org existence hidden from non-members); `ErrForbidden` → 403.

### 6. Handler refactor

`requireOrgRole` helper was deleted. The 5 existing org handlers (`handleUpdateOrganization`, `handleDeleteOrganization`, `handleUpdateOrganizationMemberRole`, `handleRemoveOrganizationMember`, `handleCreateOrgInvitation`) now use `r.With(rbacpkg.RequireOrgPermission(...))` at route registration. Handlers became thin.

`handleCreateOrganization`: seeds org roles + grants owner role atomically. Compensating delete of org on failure (if role seeding fails, the org is deleted to avoid orphaned orgs with no owner). Pattern: create org → seed roles → grant owner → on any error: delete org + return 500.

`handleUpdateOrganizationMemberRole`: syncs `org_user_roles` to match the enum tier change (`owner`/`admin`/`member`). This is best-effort: if the org has custom roles the sync only touches the built-in tier assignment.

### 7. Drive-by fixes

`SeedDefaultRoles` was defined and tested in Phase 1 (`internal/rbac/rbac.go`) but was never called from `server.Build()`. Wired into `internal/server/server.go` with a 10-second context timeout. Non-fatal: logs a warning and continues if seeding fails.

Audit logs for RBAC mutations were missing. Added `AuditLogger.Log(...)` calls on: org role create, org role delete, org permission attach/detach, org user role grant/revoke, global role assign/remove. Matches the existing `organization.member_added` audit pattern.

### 8. Applications (`internal/storage/applications.go`, `cmd/shark/cmd/app*.go`, `internal/api/application_handlers.go`)

Schema: `id TEXT NOT NULL PRIMARY KEY` (prefix `app_<nanoid>`), `client_id TEXT NOT NULL UNIQUE` (prefix `shark_app_<nanoid>`), `client_secret_hash TEXT NOT NULL` (SHA-256 hex of the plaintext secret), `client_secret_prefix TEXT NOT NULL` (first 8 chars for UX display). URL arrays stored as JSON text columns (`allowed_callback_urls`, `allowed_logout_urls`, `allowed_origins`) — SQLite has no native array type; JSON text was chosen over separate junction tables for simplicity given the expected scale.

Partial unique index: `CREATE UNIQUE INDEX idx_applications_one_default ON applications(is_default) WHERE is_default = 1` — enforces at most one default application at the DB level without a trigger.

Secret encoding: base62 (alphabet `0-9A-Za-z`) big-int divmod implementation (`cliBase62Encode` / `apiBase62Encode`). 32 random bytes → ~43 char output. SHA-256 hashed at rest — NOT AES-GCM encrypted. Unlike JWT private keys, secrets are one-way: the server only needs to verify them, never recover the plaintext. The AES-GCM pattern from the JWT subsystem was explicitly not reused here.

Default app auto-seed: `seedDefaultApplication` in `internal/server/seed_app.go` runs in `server.Build()` with a 10s context. Idempotent: no-ops if a default app already exists. On first boot, reads `cfg.Social.RedirectURL` and `cfg.MagicLink.RedirectURL` and adds them to `allowed_callback_urls` if non-empty. This means existing deployments get their redirect URLs migrated into the app model automatically.

`handleCreateApp` and `handleRotateAppSecret` return the plaintext secret in the response exactly once. Subsequent reads via `GET /admin/apps/{id}` return only `client_secret_prefix` (first 8 chars) for identification.

### 9. Redirect validator (`internal/auth/redirect/`)

Pure package — no dependency on storage, config, or net/http. Accepts a `redirect.Application` struct (not `storage.Application`) to avoid import cycles; callers populate it from their source.

Normalization: lowercase scheme+host, strip default ports (`:80` for http, `:443` for https), strip trailing `/` when path is exactly `/`.

Wildcard support: `https://*.<basedomain>` — one subdomain label only, no nested dots in the label (`a.b.preview.vercel.app` does NOT match `https://*.preview.vercel.app`). No path wildcard.

Loopback (RFC 8252 §8.3): pattern `http://127.0.0.1` or `http://localhost` in the allowlist allows any port (`http://127.0.0.1:3000`, `http://localhost:8080`, etc.).

Rejections: userinfo in URL → `ErrNotAllowed`; fragment in URL → `ErrNotAllowed`; `javascript:` / `file:` / `data:` / `vbscript:` schemes → `ErrInvalidURL`; scheme with whitespace → `ErrInvalidURL`.

Applied at: `oauth_handlers.go` (OAuth callback redirect) and `magiclink_handlers.go` (magic link verify redirect). `/oauth/authorize` for the "Shark as IdP" flow is deferred to Phase 6.

`Kind` enum: `KindCallback`, `KindLogout`, `KindOrigin` selects which allowlist field to validate against.

### 10. Deprecations

`social.redirect_url` and `magic_link.redirect_url` config fields: still read at startup by `seedDefaultApplication` to backfill the default app's `allowed_callback_urls`. Both fields have Go `//Deprecated:` doc comments. They remain functional until Phase 6. Removal target: Phase 6, when `/oauth/authorize` lands and all redirect management moves to the applications table.

`password_reset.redirect_url` is NOT yet migrated — it still functions as a standalone config field. Tracked for Phase 6 cleanup.

### 11. Test infrastructure

E2e tests use the `//go:build e2e` build tag (`internal/testutil/e2e/phase3_test.go`). Excluded from plain `go test ./...`; included in `make verify`.

Integration tests reuse the existing `//go:build integration` tag.

`make verify` target: `go vet ./... && go test -race -count=1 ./internal/... && go test -race -count=1 -tags=integration ./... && go test -race -count=1 -tags=e2e ./...`.

GP4 (`TestPhase3_GP4_RevokeJWT_ThenBlocked`): revoke endpoint returns 200, not 204 per original PHASE3.md plan. Test comment documents the deviation. `SetCheckPerRequest(true)` is called on the JWT Manager directly to enable per-request revocation checking for the test, then cleaned up via `t.Cleanup`.

GP7 (`TestPhase3_GP7_AppCreate_RowExists`): uses admin HTTP API not `shark app create` CLI, because the CLI test harness (`internal/testutil/cli`) starts its own server and the CLI's `RunE` opens its own DB connection — there is no mechanism to run CLI sub-commands against the in-memory test DB. The CLI itself is tested separately in `cmd/shark/cmd/app_test.go` with the `//go:build integration` tag.

GP8 (`TestPhase3_GP8_RedirectAllowlisted_302`): uses magic-link verify instead of OAuth callback per PHASE3.md plan, because there is no OAuth provider stub. The magic-link verify path calls `redirect.Validate` with `KindCallback` — same code path as the OAuth callback handler.

`TestServer` gets `PostJSONWithBearer` and `PostJSONWithAdminKey` helpers for Phase 3 test patterns.

### 12. Migrations added

Three new migration files, each propagated to 4 locations:

- `00006_jwt.sql`: tables `jwt_signing_keys` (id, kid, algorithm, public_key_pem, private_key_pem, created_at, rotated_at, status), `revoked_jti` (jti, revoked_at, expires_at). Indexes on `status` and `expires_at`.
- `00007_org_rbac.sql`: tables `org_roles`, `org_role_permissions`, `org_user_roles`. Cascading FK deletes. Indexes on `organization_id` and `user_id`.
- `00008_applications.sql`: table `applications` with JSON text columns for URL arrays. Partial unique index on `is_default`.

The 4 migration copy locations (a consequence of Wave 2F migration propagation):
1. `cmd/shark/migrations/` — production binary embeds these
2. `internal/testutil/migrations/` — in-memory test DB
3. `cmd/shark/cmd/testdata/migrations/` — CLI integration tests
4. `internal/testutil/cli/testmigrations/` — CLI test harness

### 13. ID prefix registry (updated)

| prefix | entity |
|--------|--------|
| `app_` | `applications.id` |
| `shark_app_` | `applications.client_id` (client-facing) |
| `orgrole_` | `org_roles.id` |

Full registry including pre-Phase 3 prefixes: `usr_`, `sess_`, `pk_`, `mlt_`, `mrc_`, `role_`, `perm_`, `key_`, `aud_`, `de_`, `org_`, `inv_`, `wh_`, `whd_`, `sk_live_`, `whsec_`.

### 14. What's next (Phase 4–6 dependencies now satisfied)

**Phase 4 dashboard**: can hit `GET /api/v1/admin/apps` to list/create applications, and the 8 org RBAC endpoints for the roles UI. JWT tokens make the dashboard authentication model more flexible (no cookie required for React SPA if deployed cross-origin).

**Phase 6 Agent Auth** (OAuth 2.1 server — `client_credentials`, `auth_code+PKCE`, `device_flow`): reuses the JWT Manager (`IssueAccessRefreshPair`, key rotation, revocation). Will add issuance for these grant types to the Manager. The `applications` table already has `client_id` / `client_secret_hash` / `allowed_callback_urls` columns — Phase 6 extends it with grant types and scopes. The redirect validator package is already in place for `/oauth/authorize`.

**Phase 8 OIDC Provider**: reuses `/.well-known/jwks.json` endpoint and the JWT signing infrastructure directly. No changes to the key management layer needed.

**Cloud fork**: `applications` table is the write target for the dashboard "Add Application" flow. `seedDefaultApplication` ensures every fresh tenant has a working default app from day one.

### 15. Known tech debt

**`interface{}` → `any`**: ~40 lint suggestions across the codebase for pre-Go 1.18 style `interface{}` that could be `any`. Untouched — cosmetic, not a correctness issue.

**`jwkFromPublicKey` and `pubKeyFromJWK` in `jwt/keys.go`**: `jwkFromPublicKey` is duplicated in `well_known_handlers.go` (separate package, different import context). `pubKeyFromJWK` in `jwt/keys.go` is currently test-only. Both will be used when Phase 6/8 adds external JWT validation (e.g. validating tokens from upstream OIDC providers). Kept to avoid churn when that lands.

**`ErrNotFound = sql.ErrNoRows` alias**: introduced in Wave 1C, not uniformly adopted — pre-existing callers in `storage/` still compare directly against `sql.ErrNoRows`. New Phase 3 code uses `errors.Is(err, sql.ErrNoRows)` consistently but doesn't retroactively fix older callers.

**Session JWT TTL hardcoded**: `IssueSessionJWT` uses `30 * 24 * time.Hour` hardcoded, not `cfg.Auth.SessionLifetimeDuration()`. The session JWT and cookie session lifetimes are therefore independent. If `auth.session_lifetime` is changed in config, the cookie session will reflect it but the JWT will not. Fix: wire `cfg.Auth.SessionLifetimeDuration()` through to `IssueSessionJWT`. Low priority since both default to 30d.

**`handleAdminRevokeJTI` returns 200 not 204**: the PHASE3.md spec said 204 for the admin endpoint. The implementation returns 200 with a body for consistency with the user revoke endpoint. The API contract is not yet stable so this can be corrected before Phase 6 ships OAuth token introspection.
