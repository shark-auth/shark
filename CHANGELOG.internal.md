# Shark Internal Changelog

> Not for public consumption. Technical notes on what shipped, why it shipped that way, and what trade-offs were made. Cross-reference with commit SHAs in repo.

## Phase 6.6 — Dashboard Deep Audit + P0/P1/P2 Fixes
Shipped: 2026-04-20
Branch: `claude/admin-vendor-assets-fix` (session extension)

### Trigger
Dashboard passed 375 smoke assertions but user reported "malfunctioning." Deep investigation via 3 parallel Sonnet subagents surfaced 24 real bugs spanning shape mismatches, 404s, crashes, hardcoded lies, mock residue. All 24 shipped. Full test suite 17/17 green.

### New API surface

- `GET /api/v1/admin/organizations/{id}/roles` — lists org-scoped custom roles
- `GET /api/v1/admin/organizations/{id}/invitations` — pending, non-expired, non-accepted
- `DELETE /api/v1/admin/organizations/{id}/members/{uid}` — admin-key auth, prevents last-owner removal
- `DELETE /api/v1/admin/sessions` — revoke all active sessions, returns `{revoked: N}`, audit logged
- `POST /api/v1/agents/{id}/rotate-secret` — generates 32-byte secret, returns plaintext once
- `GET /api/v1/admin/permissions/batch-usage?ids=a,b,c,...` — `{[id]: {roles, users}}` in single query
- `DELETE /api/v1/permissions/{id}` — used by RBAC rollback path when create+attach fails
- `GET /api/v1/webhooks/events` — canonical sorted `KnownWebhookEvents` list
- `POST /api/v1/auth/flow/mfa/verify` — consumes authflow MFA challenge, validates TOTP, returns `{verified: true}`
- `?user_id=` filter on `GET /api/v1/admin/oauth/consents`
- `?provider_id=` filter on `GET /api/v1/admin/vault/connections`
- `?org_id=`, `?session_id=`, `?resource_type=`, `?resource_id=` on `GET /api/v1/audit-logs`

### Store interface additions
- `DeleteAllActiveSessions(ctx) (int64, error)`
- `UpdateAgentSecret(ctx, id, secretHash string) error`
- `DeletePermission(ctx, id string) error`
- `BatchCountRolesByPermissionIDs(ctx, ids) (map[string]int, error)`
- `BatchCountUsersByPermissionIDs(ctx, ids) (map[string]int, error)`

### Migrations
- `migrations/00002_audit_logs_extended_filters.sql` — adds `org_id`, `session_id`, `resource_type`, `resource_id` columns + indexes to `audit_logs`. No backfill; filters apply forward only.
- `internal/testutil/migrations/00015_audit_logs_extended_filters.sql` mirror.

### P0 fixes (features broken by default)

1. **sessions.tsx empty page.** Backend returns `{data:[]}` not `{sessions:[]}`. Fixed frontend. Added `LastActivityAt` populated from `se.CreatedAt` (session table has no last-seen column; honest fallback beats fake).
2. **proxy_config.tsx all gauges N/A.** Frontend reads PascalCase; Go emits snake_case. Fixed every access (`state`, `cache_size`, `neg_cache_size`, `failures`, `last_check`, `last_status`, `last_latency_ms`). Renamed `formatLatency(ns)` → `formatLatencyMs(ms)` removing the double-divide-by-1000 bug. Fixed `onTestRule` to use `rule.path || rule.pattern || '/'` so DB override rules populate the simulator. Routed `useProxyStats` raw fetch through `shark-auth-expired` dispatch on 401 instead of silently polling forever.
3. **overview.tsx MFA%/sparkline/donut broken.** Added `Total` to `statsResponse.MFA` (reused `CountUsers` result). Frontend swap `t?.signups` → `t?.signups_by_day`. Rewrote `mapAuthBreakdown` for `[]methodBreakdown` array shape. Removed hardcoded health fallback; replaced with loading skeleton + error state. Wired AttentionPanel Refresh to `useAPI.refresh`.
4. **organizations.tsx ReferenceError crash.** Removed `disabled={deleting}` (undeclared var). Fixed members `m.user_name || m.name`, `m.user_email || m.email` (backend tags). Parse `org.metadata` JSON string before `Object.entries`.
5. **users.tsx camelCase.** `adminUserResponse` JSON tags flipped camelCase → snake_case. Every user row was showing "pending"/"—" because frontend reads snake_case.
6. **applications.tsx /admin/audit 404.** Swapped to `/audit-logs?limit=20`, response key `.events` → `.data`.
7. **GET /admin/orgs/{id}/roles** — new handler.
8. **GET /admin/orgs/{id}/invitations** — new handler.
9. **TestAdminStatsBasicCounts.** Test seeded `MFAEnabled=true` but not `MFAVerified`. Wave 2 tightened `CountMFAEnabled` to require both. Fix: seed both flags.

### P1 fixes

10. **Audit log filters + real CSV export + pagination.** Schema migration adds 4 columns. `handleExportAuditLogs` now emits text/csv (15-column header) instead of JSON envelope. Frontend export modal gets datetime-local pickers + preset chips. Pagination via `next_cursor` + "Load more" (was hard-capped at 100 with silent truncation).
11. **Settings DB chip + delete-all-users.** `health.db.status` comparison was `=== 'healthy'` but backend returns `'ok'`. Fixed. Removed type-to-confirm danger-zone UI for bulk user delete (was a stub); replaced with "Bulk deletion is CLI-only: `shark users delete --all`". `smtpConfigured` reads `config?.smtp_configured` (real field) not `config?.smtp_host` (nonexistent).
12. **authentication.tsx hardcoded lies.** Expanded `adminConfigSummary` with nested `passkey`, `password_policy`, `jwt`, `magic_link`, `session_mode`, `session_lifetime`, `social_providers`. JWT algorithm resolved live from active JWKS. Frontend reads real values. iframe `sandbox=""` → `sandbox="allow-same-origin"` for email preview.
13. **Dev-mode gate.** `adminConfigSummary.dev_mode` already existed. Frontend: `layout.tsx` filters `devOnly: true` entries when false; `App.tsx` fetches `/admin/config` after login and redirects dev-inbox away; `dev_inbox.tsx` shows friendly 404 fallback.
14. **Sessions JTI + last_seen + revoke-all + mock tabs.** Added `JTI` field (omitempty since current sessions are cookie-mode only). `DELETE /admin/sessions` route. SessionEventsTab fetches real `/audit-logs?actor_id=<user_id>`. SessionClaimsTab shows real JTI or "Cookie session — no JWT claims".
15. **Agents rotate-secret + tokens refresh + consents.** New rotate-secret endpoint. `AgentDetail` tracks `tokensVersion` state passed as useAPI dep so AgentTokens refreshes after revoke-all. AgentConsents fetches `/admin/oauth/consents?client_id=<client_id>`.
16. **User tabs: real Consents/Roles/Orgs.** `?user_id=` filter on admin consents. RolesTab uses existing `/users/{id}/permissions` (RBAC.GetEffectivePermissions dedupes). OrgsTab Remove wired to new delete-org-member endpoint. Bulk Export-CSV works client-side; Bulk Delete + Assign/Add-to-org disabled with CLI-only tooltips.

### P2 fixes

17. **Webhook events endpoint** + `whsec_` prefix strip in SigVerifyTab + silent create/update/delete now toast.
18. **Vault `?provider_id=` filter** + OAuth Connect button (opens new tab, polls close) + Test Token modal showing metadata without raw token.
19. **RBAC batch permission usage.** `PermissionsTab` was firing N*2 parallel requests per tab render; now 1 batch request. All `alert()` → `toast.error`. `handleCreateAttach` now rolls back orphan permission via `DELETE /permissions/{id}` on attach failure.
20. **Authflow 3 stubs wired.** `assign_role`, `add_to_org` (idempotent via `INSERT OR IGNORE` / existing-member check), `require_mfa_challenge`. MFA challenges stored in-memory via `internal/authflow/mfa_challenges.go` (TTL 5min, single-instance only — Cloud Phase will migrate per CLOUD.md §1). New `POST /auth/flow/mfa/verify` handler consumes challenge, decrypts user MFA secret, validates TOTP. Deferred stubs (`set_metadata`, `custom_check`, `delay`) tagged `deferred: true` in `flow_builder.tsx` palette with "v0.2" chip.
21. **Swallowed error audit.** 8 `_ = store.X` callsites now `slog.Warn` with context. Legitimate silences (prune ops, idempotent creates) left alone.
22. **Dead `agents.tsx` deleted.** `agents_manage.tsx` is the live page; `agents.tsx` was MOCK stub from Phase 4. No imports.
23-24. Covered above.

### Lessons

1. **`// @ts-nocheck` masks every field mismatch.** Most P0 bugs were backend/frontend schema drift invisible to TypeScript. Follow-up: generate TS types from Go structs + drop nocheck selectively.
2. **Response envelope drift.** Admin endpoints use `{data}`, `{users}`, bare arrays, `{items}` inconsistently. Standardize `{data, has_more?, next_cursor?}` on admin endpoints over time.
3. **LSP stale after multi-file subagent edits.** Compiler diagnostics showed "missing method" errors after Store interface grew, but `go build` was clean. Editor cache lag. Trust `go build` + `go test` output.
4. **Subagents preserve main-thread context.** 9 Sonnet subagents dispatched this session via general-purpose agent type with `model: sonnet`. Atomic file scopes, tests between batches locked in green state. Average batch: 3-7 files + tests + smoke + `npm run build`.

---

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

---

## Phase 6 — Shark Proxy + Visual Auth Flow Builder
Shipped: 2026-04-19
Commits: 50ef326, 6363fe8, 718e1e1, d728114, 7d307ee, a2d8bba, 7fed7fe, 9c9b156, 9663dfd, d74d063

### 1. Proxy architecture (`internal/proxy/`)

Six-file layout: `config.go`, `headers.go`, `proxy.go`, `rules.go`, `circuit.go`, `lru.go`. Package is dependency-free except stdlib (`net/http/httputil`, `container/list`, `log/slog`, `crypto/sha256`). Zero external deps on the hot path means every upstream request goes through <500 LOC of auditable code.

Identity flows through `context.Context` via a struct-typed key (`identityCtxKey{}`), NOT a string key. Prevents cross-package ctx collisions. `WithIdentity(ctx, id)` and `IdentityFromContext(ctx)` are the only public API.

Headers injected AFTER strip, always overwriting any client-supplied variant (belt-and-suspenders anti-spoofing). `X-User-Roles` comma-joined; `X-Shark-Cache-Age` emitted only when > 0 seconds.

Panic recovery wraps `httputil.ReverseProxy.ServeHTTP` in a deferred recover → 503, slog.Error with the panic value. Transport is `http.DefaultTransport.Clone()` (not shared — mutating Timeout would affect every caller). Per-request bound via `context.WithTimeout` in addition to `ResponseHeaderTimeout` on the transport (double-bound).

ErrorHandler writes 502 with `"upstream unreachable"` body. Deliberately opaque — never leak internal error strings to upstream clients.

### 2. Rules engine (`internal/proxy/rules.go`)

Compile-once model: `NewEngine([]RuleSpec)` parses all rules up-front, stores compiled `pathPattern` + `Requirement`. Evaluate is O(rules × segments) per request, no allocation on hot path.

Path matching chi-style. Wildcards: `/foo/*` (prefix), `/foo/*/bar` (single-segment), `/foo/{id}` (alias for single-segment). Case-sensitive. Leading `/` required at compile time.

First-match-wins iteration. Method filter is an exclusion from match, NOT an explicit reject — a GET rule doesn't deny POST to the same path; POST falls through to the next rule. Lets you layer `methods: [POST] require: role:admin` above `require: authenticated` without blocking GET traffic.

Default deny: empty rules → every request denied. Only correct default for an authorization layer.

`Requirement` has 6 kinds: `Anonymous`, `Authenticated`, `Role`, `Permission`, `Agent`, `Scope`. `Permission` stubbed in MVP (always returns false with `"permission-based rules not yet implemented"`) — clean hook for Phase 6.5 to wire RBAC. Every other kind fully evaluated inline from `Identity` fields.

Rule-level `scopes` are AND'd with the primary requirement.

### 3. Circuit breaker (`internal/proxy/circuit.go`)

State machine: `Closed` ↔ `Open` ↔ `HalfOpen`. `Closed` + 3 consecutive failures → `Open`. `Open` + 1 success probe → `HalfOpen`. `HalfOpen` + success → `Closed`, + failure → back to `Open`. Failure counter resets on success in `Closed` (prevents flapping). Defaults: 3 threshold / 10s interval / 3s probe timeout.

LRU (`internal/proxy/lru.go`): custom impl, `container/list` doubly-linked + `map[string]*list.Element`. O(1) get/put. Lazy TTL expiry — entries evicted on get() if expired, not via background sweep. Capacity-based eviction at tail. `sync.Mutex`-guarded.

Two LRU instances per breaker: positive cache (5m TTL, 10K capacity) for known-good `cookie_hash → identity` maps; negative cache (30s TTL) for known-bad tokens. Negative TTL strictly shorter — a token revoked in auth server should be re-checked quickly if re-presented.

`HashCookie(raw)` is SHA-256 hex. Never use raw cookie as cache key — prevents leaky log from revealing session tokens.

Health monitor in dedicated goroutine, `time.Ticker`-driven. `Stop()` signals via `stopCh` AND waits on `doneCh` (idempotent). `StartStop` test asserts no goroutine leak via `runtime.NumGoroutine` diff.

`BreakerResolver` composes `JWTResolver` + `LiveResolver`:
- JWT tokens short-circuit to `JWTResolver` regardless of breaker state (stateless)
- Session cookies route by state: `Closed` → `Live` + cache positive result; `Open` → cache lookup; `HalfOpen` → `Live` (success closes)
- `Open` + cache miss + `miss_behavior: "reject"` → `ErrBreakerOpenNoCache`
- `Open` + cache miss + `miss_behavior: "allow_readonly"` → anonymous identity with `AuthMethod: "anonymous-degraded"` — GET/HEAD only

Negative-cache population: on `Live` error, cookie hash added to neg cache for 30s.

### 4. Integration points (`internal/api/`)

`proxy_resolvers.go`: `JWTResolver` wraps `jwtpkg.Manager.Verify`, `LiveResolver` wraps `auth.SessionManager.Validate` + `rbac.RBACManager.ListUserRoles`. Both implement `proxy.AuthResolver`. `proxy_handlers.go`: admin handlers all 404 themselves when `ProxyBreaker`/`ProxyEngine` is nil — safer than route-level gating because admin API stays at stable URL regardless of config.

SSE endpoint (`/admin/proxy/status/stream`): `http.Flusher` + `text/event-stream` + 2s `time.Ticker`. Client disconnect via `r.Context().Done()`. Native browser `EventSource` can't set `Authorization` header, so dashboard falls back to 2s polling — SSE endpoint works with curl or custom clients.

Catch-all mount: `r.Handle("/*", s.proxyAuthMiddleware(s.ProxyHandler))` at END of router construction. Chi trie precedence gives every other registered route priority. `TestProxyIntegration_AuthRoutesBypassProxy` asserts `/auth/login` continues to function with proxy enabled.

Simulate endpoint reconstructs an `Identity` from request body, calls `proxy.Engine.Evaluate`, then `proxy.InjectIdentity` to produce `injected_headers`. No mocks — dashboard simulator hits same code path as live request.

### 5. Standalone proxy (`cmd/shark/cmd/proxy.go`)

MVP anonymous-only. Full JWT verification requires JWKS fetch + cache with rotation awareness — deferred to P4.1. Documented in command Long help. Standalone acceptable for zero-auth read-through (CDN fronting, static API mirrors) and rule testing in isolation.

### 6. Dashboard (`admin/src/components/proxy_config.tsx`)

Three sections: circuit strip (3 gauges) → URL simulator hero → rules table (read-only). Inline YAML-only note for rule editing (P5.1 adds inline editor + drag reorder).

Polling loop for status: `useProxyStats()` custom hook with `setInterval(2000)` + `document.visibilityState` check (pauses when tab hidden). Native `fetch` not `useAPI` — hook needs HTTP status code to detect 404 → proxy disabled empty state.

Gauges use CSS `@keyframes pulse` on status dot — 3s cycle when Closed, 1s amber when HalfOpen, static red when Open. No box-shadow glow (rejected AI slop), just 1px border with semantic color tokens.

### 7. Auth Flow storage (`internal/storage/auth_flows*`)

Migration 00012 adds `auth_flows` and `auth_flow_runs`. `auth_flows.steps` JSON-encoded `[]FlowStep` (each step: `type`, `config`, plus branch fields `condition`/`then`/`else`). `auth_flow_runs.metadata` opaque JSON populated by engine — holds timeline for dashboard history.

`auth_flow_runs.flow_id` FK cascades on flow delete. `user_id` nullable (pre-signup flows). `blocked_at_step` nullable INTEGER via `sql.NullInt64` with `*int` in Go.

Indexed `(trigger, priority DESC)` for `ListAuthFlowsByTrigger` — used in every handler.

### 8. Flow engine (`internal/authflow/`)

Three-file split: `engine.go` (types, Execute/ExecuteDryRun/persistence), `steps.go` (12 executors), `conditions.go` (map-based predicate DSL).

6 fully-wired step types, 6 stubbed. Stubbed dispatch, log warning, return `Continue` — flows don't break mid-execution. Each has `TODO(F2.1)` comment with specific backend integration needed.

Webhook executor:
- Timeout from `config.timeout` (default 5s, capped 30s)
- `http.NewRequestWithContext(ctx, method, url, body)` with per-request context
- Body is `{trigger, user (sanitized), metadata}` — `sanitizeUser` clears `PasswordHash` and `MFASecret` before marshaling. `TestEngine_Webhook_SanitizesUser_DoesntLeakPasswordHash` asserts.
- Non-2xx → `outcome: error` (non-fatal; handler proceeds)
- Timeout via context, `TestEngine_Webhook_TimeoutErrors` asserts

Conditional evaluator: `condition` is JSON-encoded string on FlowStep. Engine parses at runtime, empty string = "always match" (backward-compat). Bad JSON → error outcome with reason.

`Engine.Execute` idempotent across calls with same Context, modulo timeline timestamps. `ExecuteDryRun` skips `persistRun` entirely.

Timeline populated even on Block/Error — every step that ran appears in `result.Timeline`. Subsequent steps after short-circuit not added; UI shows them as "skipped" via absence.

Injectable clock + `WithHTTPClient` option for tests.

### 9. Condition DSL (`internal/authflow/conditions.go`)

Map-based, NOT expression engine. Keeps semantic surface small and auditable. Predicates: `email_domain`, `has_metadata`, `metadata_eq`, `trigger_eq`, `user_has_role`, `all_of`, `any_of`, `not`. Empty `{}` → always true.

Top-level map is implicit-AND — every key must hold. Collapses simple flows without explicit `all_of` nesting.

Unknown predicate keys return `ErrUnknownPredicate` (exported sentinel). Callers distinguish config errors from data errors via `errors.Is`.

### 10. Flow integration hooks (`internal/api/*_handlers.go`)

`Server.runAuthFlow(w, r, trigger, user, password)` single entry point. Called from `handleSignup`, `handleLogin`, `handleOAuthCallback`, `handleMagicLinkVerify`, `handlePasswordReset`.

Returns `handled bool`. On `block` → 403 with `{"error":"flow_blocked"}`. On `redirect` → 302 with `Location` + JSON body. On `continue` or `error` → returns false so caller proceeds normally.

**Login hook placement deviation**: spec said "after password/MFA verification". Actual fires after password verify, BEFORE MFA challenge. Login handler resolves MFA on separate `/auth/mfa/challenge` endpoint — post-MFA placement would require threading through that endpoint. Tracked as F3.1.

**Mutation happens before flow**: signup/password_reset commit user row / password update before flow runs. Blocking flow leaves DB state mutated but withholds session. Documented in FLOWS.md — flow is authorization gate, not transactional rollback.

### 11. Dashboard Flow Builder (`admin/src/components/flow_builder.tsx`)

Two views: `FlowsList` (table) and `FlowEditor` (three-pane). Single-pane routing with state-backed "Back" rather than split-grid master-detail — editing warrants full focus.

Palette organized by family (Block, Side effect, Branch). Click inserts after selection. Dragging deferred F4.1.

Canvas auto-laid-out: `gap: 24px` vertical flex, trigger pseudo-node top, done pseudo-node bottom. Conditional steps render linearly with indented then/else beneath — forked visualization (two parallel tracks rejoining at merge) deferred F4.1.

Config panel dispatches on `step.type` to render correct field schema. ~50 form components kept inline with repeated patterns rather than extracting JSON-schema renderer (YAGNI — every step shape known at build time).

Preview tab POSTs `/admin/flows/{id}/test`, renders timeline with 80ms stagger per row. `fadeIn` keyframe. Blocked preview shows offending step in red, subsequent rows faded to 40% opacity.

Mock user presets in `localStorage` under `sharkauth.flow.mocks`. Edit and save inline. No server storage.

### 12. Smoke test coverage

Sections 49-54 added to `smoke_test.sh`:
- 49: proxy admin endpoints 404 when disabled + 401 on no-auth
- 50: flow CRUD + validation (bad trigger → 400, empty steps → 400)
- 51: dry-run with verified + unverified users, timeline populated + outcome differs
- 52: signup with blocking flow returns 403 flow_blocked
- 53: disabled flow lets signup through
- 54: runs persisted after real execution

Total smoke: 244 PASS, 0 FAIL (up from 222 in Phase 5.5).

### 13. Trade-offs made

**Custom LRU instead of dep**: `hashicorp/golang-lru` rejected. ~60 LOC hand-rolled eliminates supply-chain surface on hot path.

**SSE despite dashboard polling**: `EventSource` can't set `Authorization`, but curl-based ops tools and future SDK clients will use SSE. Polling fallback is dashboard only.

**Permission requirement stubbed, not omitted**: `require: permission:users:read` parses successfully but always denies. Parse-error alternative would make YAML config hard to author incrementally — users draft rules referencing Phase 6.5 permissions without YAML rejection.

**Standalone proxy anonymous-only**: full JWKS-fetch impl ~200 LOC + cache-invalidation story. Deferred. Embedded mode covers 95% case.

**Linear conditional display**: forked canvas with two visual tracks rejoining at merge requires dagre-style layout (or custom). Deferred. Linear with indented branches readable and ships today.

**Flow error outcome non-fatal**: webhook timeout during login could brick auth. Errors log + proceed. "Strict" flows via `custom_check` (which returns block on non-2xx) — stubbed today, F2.1 wires properly.

**MVP palette-click step insertion**: no drag-drop. Keeps JS bundle free of drag-drop libs (react-dnd, @dnd-kit add 20-50KB gzipped). Palette-click equivalent for building from scratch; drag benefit is reordering, deferred F4.1.

**Steps array serialized as JSON text column, not relational**: `flow_steps` table with `step_order` + self-FK for branches is "more correct" but every access reads all steps anyway. Conditional's nested `then`/`else` arrays don't fit relationally without awkward recursion. JSON column keeps reads single-query and lets engine iterate native slices.

### 14. What's next (Phase 7 dependencies now satisfied)

**Phase 7 SDK**: TypeScript SDK (#54) builds on OAuth 2.1 (Phase 5) + optionally uses proxy for zero-code auth. Standalone proxy JWKS fetch (P4.1) is blocker for edge deployments.

**Phase 8 OIDC Provider**: reuses `/oauth/authorize` flow from Phase 5; flow builder hooks work for `oauth_callback` trigger today.

### 15. Known tech debt (Phase 6)

**Windows IDE path case-insensitivity**: go-build cache caches paths with lowercase `desktop` vs uppercase `Desktop`, leading to spurious "undefined: Server" diagnostics across sessions. Cosmetic — real `go build` clean. Windows + goimports interaction.

**`ProxyConfig.StripIncoming` as `*bool`**: pointer to distinguish unset (default true) from explicit false. YAML parser can't express "not set" for bare bool.

**`proxy_handlers_test.go` fabricated helpers on first pass**: implementer invented `testutil.NewTestJWTManager` and `store.CreateAPIKeyHelper` that didn't exist. Fixed by adding `testutil.NewTestServerWithConfig` and rewriting helper block. Pattern: agents hallucinate helper names when real testutil API isn't loaded in context.

**Flow conditional UI forked-canvas not implemented**: linear-indented works but isn't the "wow" visualization brief called for. F4.1.

**Six stubbed step types**: `require_mfa_challenge`, `set_metadata` (persistence), `assign_role`, `add_to_org`, `custom_check`, `delay` dispatch but don't persist effects. F2.1 wires them.

**Login hook pre-MFA**: fires before MFA challenge, not after. Post-MFA requires wiring through `handleMFAChallenge`. F3.1.

**Proxy rule editor not in dashboard**: rules read-only; edit `sharkauth.yaml` and reload. P5.1 adds inline editor + drag-reorder.
