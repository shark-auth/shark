# Phase 3 — JWT, Org RBAC, Applications & Redirect Allowlist

> **Phase goal**: Build the foundation that Agent Auth (Phase 6), OIDC Provider (Phase 8), and Cloud (separate fork) all depend on — industry-standard JWT issuance with JWKS, org-scoped RBAC, registered applications with redirect URI allowlist. Plus drive-by fixes for two latent bugs (`SeedDefaultRoles` never wired, RBAC grants never audited).
>
> **Estimated duration**: 5–7 days (4 waves, 11 sub-agents).
>
> **Done definition**: every preexisting test passes + new unit tests per component pass + integration & e2e tests for new flows pass under `make verify` with `-race`.

---

## 0. Scope & Decisions (locked)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| JWT shape | **Both modes selectable** (`session` default, `access_refresh` optional) | Configless ergonomics for hobby installs; OAuth-style for SDK/agent paths |
| JWT algorithm | **RS256** with JWKS endpoint | Industry-standard, ubiquitous client support; ES256 path documented as future drop-in |
| Revocation | **TTL-only by default** + `/api/v1/auth/revoke` endpoint + `revoked_jti` table | Pure stateless validation; `check_per_request` opt-in for compliance scenarios |
| Cookie ↔ JWT toggle | **Both always-on**, middleware tries Bearer first, falls back to cookie | Zero migration; clients pick |
| JWT transport | **`Authorization: Bearer <token>`** (RFC 6750) | Industry standard; no JWT-in-cookie variant |
| Org RBAC schema | **Parallel tables** (`org_roles`, `org_role_permissions`, `org_user_roles`) | Global RBAC untouched; clean separation |
| `organization_members.role` enum | **Keep CHECK enum** as membership tier; custom roles layer on top | Less invasive; existing handlers untouched at storage layer |
| Custom org roles | **Full RBAC self-host** (no Cloud-only gating) | Shark is open-core; no enterprise paywalls per STRATEGY.md |
| Org permission enforcement | **Refactor** `requireOrgRole` calls into `RequireOrgPermission` middleware | Handlers become thin; permission strings are explicit at route registration |
| Applications scope | **Schema + CLI + admin HTTP + redirect validator NOW**; `/oauth/authorize` deferred to Phase 6 | Cloud needs the schema landed; allowlist enforcement applied to existing surfaces (oauth callback, magic link verify) |
| Multi-tenancy | **Single-tenant** (no `tenant_id` column) | Cloud uses SQLite WAL with db-per-client; each shark binary stays single-tenant |
| `SeedDefaultRoles` bug | **Fix in this phase** — wire into `server.go` `Build()` | Trivial 5-line fix; production currently has no roles seeded |
| Audit logging gap | **Fix in this phase** — add audit calls on every RBAC + app mutation | Compliance requirement; matches existing org audit pattern |

---

## 1. JWT Subsystem

### 1.1 Schema changes

Migration files (identical content): `cmd/shark/migrations/00006_jwt.sql` AND `internal/testutil/migrations/00006_jwt.sql`.

```sql
-- +goose Up

CREATE TABLE jwt_signing_keys (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    kid             TEXT    NOT NULL UNIQUE,
    algorithm       TEXT    NOT NULL DEFAULT 'RS256',
    public_key_pem  TEXT    NOT NULL,
    private_key_pem TEXT    NOT NULL,   -- AES-GCM encrypted, base64-encoded ciphertext
    created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    rotated_at      DATETIME,           -- NULL while active
    status          TEXT    NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'retired'))
);

CREATE INDEX idx_jwt_signing_keys_status ON jwt_signing_keys(status);

CREATE TABLE revoked_jti (
    jti         TEXT     PRIMARY KEY,
    revoked_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at  DATETIME NOT NULL
);

CREATE INDEX idx_revoked_jti_expires_at ON revoked_jti(expires_at);

-- +goose Down

DROP INDEX IF EXISTS idx_revoked_jti_expires_at;
DROP TABLE IF EXISTS revoked_jti;
DROP INDEX IF EXISTS idx_jwt_signing_keys_status;
DROP TABLE IF EXISTS jwt_signing_keys;
```

Expired `revoked_jti` rows are pruned lazily on each `Validate` call (when `check_per_request` is enabled) and on each `Revoke` call: `DELETE FROM revoked_jti WHERE expires_at < datetime('now')` runs before the real query.

### 1.2 Key management

**Generation.** `crypto/rsa.GenerateKey(rand.Reader, 2048)`. Compute `kid = base64url(SHA-256(DER-encoded public key))[:16]`. Encode private key to PKCS#8 PEM via `x509.MarshalPKCS8PrivateKey`.

**Encryption at rest.** Derive a 32-byte AES-GCM key with `sha256.Sum256([]byte(server.secret + "jwt-key-encryption"))`. Encrypt the PKCS#8 PEM bytes with `aes.NewCipher` → `cipher.NewGCM` → `Seal`. Store as `base64.StdEncoding.EncodeToString(nonce + ciphertext)` in `private_key_pem`.

**Justification**: reuses the single mandatory secret the operator already owns. Adding a second secret would complicate the 1-question install. Domain separator `+ "jwt-key-encryption"` prevents key reuse with the session cookie encryption that uses the same `server.secret`.

**CLI command.** New file `cmd/shark/cmd/keys.go`:

```
shark keys generate-jwt [--rotate]
```

- Without `--rotate`: generate keypair, insert row with `status='active'`. Fails if an active key exists.
- With `--rotate`: mark all `status='active'` rows as `status='retired'`, set `rotated_at=now()`, insert new active row. Both keys remain in JWKS until `retired.rotated_at + auth.jwt.access_token_ttl * 2` has elapsed (lazy filter — no background job needed).

**Auto-bootstrap.** `server.go` calls `jwtMgr.EnsureActiveKey(ctx)` at startup. If `jwt_signing_keys` has no active row, generate one automatically. Manual `shark keys generate-jwt` becomes optional, not required for first-boot UX.

**Key zeroization.** After decrypting a private key into an `*rsa.PrivateKey`, wipe the intermediate `[]byte` manually with `for i := range b { b[i] = 0 }`. Go's GC does not guarantee zeroization.

### 1.3 Config additions

Add under `auth:` block in YAML and as nested struct in `internal/config/config.go`:

```yaml
auth:
  jwt:
    enabled: true              # default true (always-on, both modes accepted)
    mode: "session"            # "session" | "access_refresh"
    issuer: ""                 # empty → falls back to server.base_url
    audience: "shark"
    access_token_ttl: "15m"    # only used when mode=access_refresh
    refresh_token_ttl: "30d"   # only used when mode=access_refresh
    clock_skew: "30s"
    revocation:
      check_per_request: false # default false (TTL-only)
```

Go structs:

```go
type JWTRevocationConfig struct {
    CheckPerRequest bool `koanf:"check_per_request"`
}

type JWTConfig struct {
    Enabled         bool                `koanf:"enabled"`
    Mode            string              `koanf:"mode"`
    Issuer          string              `koanf:"issuer"`
    Audience        string              `koanf:"audience"`
    AccessTokenTTL  time.Duration       `koanf:"access_token_ttl"`
    RefreshTokenTTL time.Duration       `koanf:"refresh_token_ttl"`
    ClockSkew       time.Duration       `koanf:"clock_skew"`
    Revocation      JWTRevocationConfig `koanf:"revocation"`
}

// Add to AuthConfig:
JWT JWTConfig `koanf:"jwt"`
```

Defaults set in `config.Load()` defaults map: `Enabled=true`, `Mode="session"`, `Audience="shark"`, `AccessTokenTTL=15m`, `RefreshTokenTTL=30d`, `ClockSkew=30s`, `CheckPerRequest=false`.

### 1.4 JWT issuance

Package `internal/auth/jwt/`. Single `Manager` type with config + storage references.

```go
package jwt

type Manager struct {
    cfg   *config.JWTConfig
    store storage.Store
    base  string // server.base_url, fallback issuer
}

func NewManager(cfg *config.JWTConfig, store storage.Store, baseURL string) *Manager
func (m *Manager) EnsureActiveKey(ctx context.Context) error
func (m *Manager) IssueSessionJWT(ctx context.Context, user *storage.User, sessionID string, mfaPassed bool) (string, error)
func (m *Manager) IssueAccessRefreshPair(ctx context.Context, user *storage.User, sessionID string, mfaPassed bool) (access, refresh string, err error)
func (m *Manager) Refresh(ctx context.Context, refreshToken string) (newAccess, newRefresh string, err error)
func (m *Manager) Validate(ctx context.Context, token string) (*Claims, error)
func (m *Manager) RevokeJTI(ctx context.Context, jti string, expiresAt time.Time) error
```

**Claims struct** (shared issue/validate):

```go
type Claims struct {
    jwt.RegisteredClaims                    // iss, sub, aud, exp, nbf, iat, jti
    MFAPassed bool   `json:"mfa_passed"`
    SessionID string `json:"session_id"`
    TokenType string `json:"token_type"` // "session" | "access" | "refresh"
}
```

**Plug-in points.** After existing `SessionManager.SetSessionCookie(...)` calls, check `cfg.JWT.Enabled` and call the appropriate issue method. Return token(s) in JSON response body alongside the cookie. Cookie + JWT are additive at login.

- `internal/api/auth_handlers.go` — password login
- `internal/api/oauth_handlers.go` — social OAuth callback
- `internal/api/sso_handlers.go` — SSO assertion success
- `internal/api/magiclink_handlers.go` — magic link verify

Response body shape when JWT enabled:

```json
{ "token": "<session_jwt>" }
// or in access_refresh mode:
{ "access_token": "...", "refresh_token": "..." }
```

### 1.5 JWT validation

```go
var (
    ErrExpired          = errors.New("jwt: token expired")
    ErrInvalidSignature = errors.New("jwt: invalid signature")
    ErrRevoked          = errors.New("jwt: token revoked")
    ErrUnknownKid       = errors.New("jwt: unknown kid")
    ErrAlgMismatch      = errors.New("jwt: algorithm mismatch")
)
```

Validation order:

1. Parse header without verifying. Reject if `alg` is `none` or any HMAC variant (`HS256/384/512`) → `ErrAlgMismatch`.
2. Look up `kid` in `jwt_signing_keys` where `status IN ('active','retired')`. Return `ErrUnknownKid` if absent.
3. Parse `public_key_pem` (plaintext column) into `*rsa.PublicKey`.
4. Verify signature via `golang-jwt/jwt/v5` with explicit `jwt.WithValidMethods([]string{"RS256"})`.
5. Check `exp`, `iat`, `nbf` with `jwt.WithLeeway(m.cfg.ClockSkew)`.
6. Check `iss == m.issuer()` and `aud` contains `m.cfg.Audience`.
7. If `cfg.Revocation.CheckPerRequest`, prune expired rows then `SELECT 1 FROM revoked_jti WHERE jti = ?`. Return `ErrRevoked` on hit.
8. Return `*Claims` on success.

### 1.6 JWKS endpoint

Route in `internal/api/router.go`:

```go
r.Get("/.well-known/jwks.json", h.HandleJWKS)
```

No auth. Handler in new file `internal/api/well_known_handlers.go`. Query rows where `status='active'` OR (`status='retired'` AND `rotated_at + access_token_ttl * 2 > now()`). For each, build RFC 7517 JWK with `kty:"RSA"`, `use:"sig"`, `alg:"RS256"`, `kid`, `n` (base64url modulus), `e` (base64url exponent). Return `{"keys": [...]}` with `Content-Type: application/json` and `Cache-Control: public, max-age=300`.

### 1.7 Manual revocation endpoints

**User self-revoke** (mounted under `RequireSessionFunc`):

```
POST /api/v1/auth/revoke
Body: {"token": "..."}
Response: 204 No Content
```

Decode token without signature verification (use `jwt.ParseUnverified`) to extract `jti`, `exp`, `sub`. Verify `sub == ctx.UserID` (prevents cross-user revoke). Insert `(jti, expires_at=exp)` into `revoked_jti`. Idempotent.

**Admin revoke** (under `AdminAPIKeyFromStore`):

```
POST /api/v1/admin/auth/revoke-jti
Body: {"jti": "...", "expires_at": "2026-05-01T00:00:00Z"}
Response: 204 No Content
```

Direct insert without requiring the full token. Useful for emergency revocation when the token isn't accessible.

### 1.8 Library choice

**`github.com/golang-jwt/jwt/v5`**. Direct successor of `dgrijalva/jwt-go`, actively maintained, explicit `WithValidMethods` enforcement (mitigates alg-confusion at lib level), sufficient API for serving our own JWKS. `lestrrat-go/jwx` is heavier; appropriate later when shark validates third-party JWTs (resource server mode). `ES256` future drop-in: replace `rsa.GenerateKey` with `ecdsa.GenerateKey(elliptic.P256, ...)`, change `algorithm` column value, update `WithValidMethods`. No schema change required.

### 1.9 Security checklist (must-implement)

- Reject `alg: none` and HMAC variants at step 1 of `Validate` (alg-confusion).
- Token type enforcement: refresh tokens MUST be rejected by middleware (`token_type=="access"` or `"session"` only).
- `nbf`/`exp` always with `clock_skew`.
- `kid` required; missing kid → `ErrUnknownKid`.
- Wipe `[]byte` holding decrypted private key material after use.
- Refresh is one-time-use: `Refresh()` inserts old jti into `revoked_jti` before issuing the new pair (this check runs regardless of global `check_per_request`).
- Private key encrypted at rest (AES-GCM, domain-separated key); plain PEM never written to disk or logged.
- No key bytes, no full token strings, no decrypted PEM in logs.

### 1.10 Test plan

Unit (`internal/auth/jwt/manager_test.go`):

| Test | Asserts |
|------|---------|
| `TestIssueSession` | `token_type="session"`, `exp = iat + session_lifetime` |
| `TestIssueAccessRefresh` | Both parse; access exp=15m, refresh exp=30d; distinct jti |
| `TestValidate_Valid` | Happy path; claims match input |
| `TestValidate_Expired` | `ErrExpired` for past-exp (beyond skew) |
| `TestValidate_BadSignature` | Different key → `ErrInvalidSignature` |
| `TestValidate_AlgConfusion` | HS256 token (signed with public key) → `ErrAlgMismatch` |
| `TestValidate_UnknownKid` | Unknown kid → `ErrUnknownKid` |
| `TestRefresh_Rotates` | New pair issued; old refresh jti in `revoked_jti` |
| `TestRevoke_PreventsValidationWhenChecked` | After revoke, validate with `check_per_request=true` → `ErrRevoked` |
| `TestKeyRotation_BothInJWKS` | After `--rotate`, JWKS lists old + new |

Integration (`internal/api/jwt_e2e_test.go`):

`TestE2E_LoginToJWTToMeToRevoke`:
1. `POST /api/v1/auth/login` → 200, extract token from body
2. `GET /api/v1/auth/me` with `Authorization: Bearer <token>` → 200
3. `POST /api/v1/auth/revoke` with `{"token":"..."}` → 204
4. With `check_per_request=false`, repeat step 2 → 200 (TTL-only)
5. With `check_per_request=true`, repeat step 2 → 401

---

## 2. Session Middleware Refactor (cookie + JWT dual-accept)

### 2.1 New middleware signature

```go
func RequireSessionFunc(sm *auth.SessionManager, jwtMgr *jwt.Manager) func(http.Handler) http.Handler
```

Decision tree (in order):

1. `Authorization` header starts with `"Bearer "` → extract token → `jwtMgr.Validate(ctx, token)`.
   - Success: set ctx with `UserIDKey=claims.UserID`, `SessionIDKey=claims.SessionID` (may be `""`), `MFAPassedKey=claims.MFAPassed`, `AuthMethodKey="jwt"`. Continue.
   - Failure: 401 with `WWW-Authenticate: Bearer error="invalid_token"`. **Do NOT fall through to cookie.**
2. No Bearer header → existing cookie path (`sm.GetSessionFromRequest` + `sm.ValidateSession`). Set `AuthMethodKey="cookie"` on success.
3. Neither: 401 with `WWW-Authenticate: Bearer`.

**No-fallthrough rationale**: when caller declares Bearer, that credential MUST be valid. Falling through silently masks token expiry/revocation/key-rollover failures. RFC 6750 §3.1 mandates 401 `invalid_token` for malformed/expired bearer. Fallthrough also enables a confused-deputy attack where a rejected JWT alongside a stolen cookie gets through.

### 2.2 Context contract

Add new key in `internal/api/middleware/auth.go`:

```go
const AuthMethodKey contextKey = "shark.auth_method"

func GetAuthMethod(ctx context.Context) string {
    if v, ok := ctx.Value(AuthMethodKey).(string); ok {
        return v
    }
    return ""
}
```

| Key | Type | Set by | Meaning |
|-----|------|--------|---------|
| `UserIDKey` | string | mw | Authenticated user ID; non-empty on success |
| `SessionIDKey` | string | mw | DB session UUID; **empty when `AuthMethod=="jwt"` AND `token_type=="access"`** |
| `MFAPassedKey` | bool | mw | From `sess.MFAPassed` (cookie) or `claims.MFAPassed` (JWT) |
| `AuthMethodKey` | string | mw | `"jwt"` or `"cookie"` |

`SessionIDKey` is non-empty in JWT mode only when `claims.TokenType=="session"` (JWT was issued alongside a cookie at login and carries the DB session row's ID). Pure access tokens are stateless → empty `SessionIDKey`.

### 2.3 Token type enforcement

After successful `jwtMgr.Validate`, accept only `claims.TokenType ∈ {"session", "access"}`. Reject `"refresh"`:

```
WWW-Authenticate: Bearer error="invalid_token", error_description="refresh token cannot be used as access credential"
```

Blocks the refresh-as-bearer attack.

### 2.4 Cookie name configurability

**Recommendation: keep `"shark_session"` hardcoded** at `internal/auth/session.go:19`. Reason: changing the cookie name on upgrade silently invalidates every active session with no migration path. Operational cost is not justified by any security or integration benefit. If a deployment genuinely needs a custom name (multi-instance same-domain), file a follow-up; solution is a `ServerOption`, not YAML.

### 2.5 Wiring changes

**`internal/server/server.go`** in `Build()`:

```go
jwtMgr := jwt.NewManager(&cfg.Auth.JWT, store, cfg.Server.BaseURL)
if err := jwtMgr.EnsureActiveKey(ctx); err != nil {
    return nil, fmt.Errorf("ensure active jwt key: %w", err)
}
apiSrv := api.NewServer(store, cfg,
    api.WithEmailSender(sender),
    api.WithWebhookDispatcher(dispatcher),
    api.WithJWTManager(jwtMgr),
)
```

**`internal/api/router.go`** — add `JWTManager *jwt.Manager` field to `Server` struct, add `WithJWTManager` option. Update **10 call sites** of `mw.RequireSessionFunc(sm)` to `mw.RequireSessionFunc(sm, s.JWTManager)`:

| Line | Route context |
|------|---------------|
| 147 | `/auth` authenticated group |
| 153 | `/auth/sessions` group |
| 161 | `/auth/email` verified sub-group |
| 177 | `/auth/passkey` authenticated group |
| 187 | `/auth/passkey` registration group |
| 208 | `/auth/password` change group |
| 219 | `/auth/mfa` challenge group |
| 225 | `/auth/mfa` management group |
| 240 | `/auth` MFA-required group |
| 248 | `/organizations` group |

### 2.6 Backward compat audit

- **`session_handlers.go:50` `handleListMySessions`** — uses `currentID` to mark current session. In JWT access mode, `currentID==""`, so no session marked current. Acceptable. Document that `Current` is always `false` in stateless JWT mode.
- **`auth_handlers.go:278-280` `handleLogout`** — currently reads cookie directly. In JWT-only mode, cookie absent → `DeleteSession` skipped. Prescribe: read `AuthMethodKey`; if `"jwt"`, call `jwtMgr.RevokeJTI(ctx, claims.Jti, claims.ExpiresAt)`. Add new `GetClaims(ctx)` helper that returns the JWT claims if set by middleware.
- **`mfa_handlers.go:203-204` `handleMFAChallenge`** and **`mfa_handlers.go:283-284` `handleMFARecovery`** — require `sessionID != ""` (correct: MFA challenge is a session-mutation op, inherently session-backed). Document that stateless JWT access tokens are correctly rejected; use cookie or `token_type=="session"` JWT.
- **`session_handlers.go:80` `handleRevokeMySession`** — uses URL param, unaffected.

All other `GetUserID` call sites are unaffected.

### 2.7 CORS implications

`internal/api/middleware/cors.go:30` already includes `Authorization` in `Access-Control-Allow-Headers`. Verify with cross-origin request during implementation. No code change needed; add a comment confirming verification.

### 2.8 Rate limiting

`mw.RateLimit(100, 100)` at `router.go:128` is IP-based, runs before route handlers. JWT auth doesn't change identity used for rate limiting. No changes. Future improvement (out of scope): per-`UserID` rate limiter for authenticated routes.

### 2.9 Test plan

Unit (`internal/api/middleware/auth_test.go`):

- `TestRequireSession_BearerValid` → 200, `AuthMethod="jwt"`, `SessionID=""`
- `TestRequireSession_BearerInvalid` → 401, `WWW-Authenticate`, no fallthrough (cookie ignored)
- `TestRequireSession_BearerExpired` → 401 with `error="invalid_token"`
- `TestRequireSession_BearerRefreshToken` → 401 with `error_description` mentioning refresh
- `TestRequireSession_CookieValid` → 200, `AuthMethod="cookie"`, `SessionID` non-empty
- `TestRequireSession_NoAuth` → 401 with `WWW-Authenticate: Bearer`
- `TestRequireSession_BothPresent` → 200, `AuthMethod="jwt"` (Bearer wins)

Integration: `TestE2E_MeEndpointBothAuthModes` — login captures cookie + JWT. Three calls to `/me`: cookie-only, Bearer-only, both. All return identical user_id.

### 2.10 Files touched (exhaustive)

- `internal/api/middleware/auth.go` (refactor + AuthMethodKey + GetAuthMethod + GetClaims helper)
- `internal/api/middleware/cors.go` (no code change; add verified comment)
- `internal/api/router.go` (Server struct field + WithJWTManager + 10 call sites updated)
- `internal/server/server.go` (instantiate jwtMgr + EnsureActiveKey + pass via option)
- `internal/api/middleware/auth_test.go` (7 new unit tests)
- `internal/api/auth_handlers.go` (`handleLogout` JWT branch)
- `internal/api/mfa_handlers.go` (comments only)
- `internal/api/session_handlers.go` (comments only)

---

## 3. Org-Scoped RBAC

### 3.1 Schema

Migration files (identical): `cmd/shark/migrations/00007_org_rbac.sql` AND `internal/testutil/migrations/00007_org_rbac.sql`.

```sql
-- +goose Up

CREATE TABLE IF NOT EXISTS org_roles (
  id TEXT PRIMARY KEY,                        -- orgrole_<nanoid>
  organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  is_builtin BOOLEAN NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(organization_id, name)
);

CREATE TABLE IF NOT EXISTS org_role_permissions (
  org_role_id TEXT NOT NULL REFERENCES org_roles(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  resource TEXT NOT NULL,
  PRIMARY KEY (org_role_id, action, resource)
);

CREATE TABLE IF NOT EXISTS org_user_roles (
  organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  org_role_id TEXT NOT NULL REFERENCES org_roles(id) ON DELETE CASCADE,
  granted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  granted_by TEXT REFERENCES users(id) ON DELETE SET NULL,
  PRIMARY KEY (organization_id, user_id, org_role_id)
);

CREATE INDEX idx_org_roles_org ON org_roles(organization_id);
CREATE INDEX idx_org_user_roles_user ON org_user_roles(user_id);

-- +goose Down

DROP INDEX IF EXISTS idx_org_user_roles_user;
DROP INDEX IF EXISTS idx_org_roles_org;
DROP TABLE IF EXISTS org_user_roles;
DROP TABLE IF EXISTS org_role_permissions;
DROP TABLE IF EXISTS org_roles;
```

### 3.2 Permission model

Same `(action, resource)` tuple as global RBAC. Wildcards (`*`) work the same. Standard permission strings to seed:

```
members:read          members:invite        members:remove        members:update_role
org:read              org:update            org:delete
roles:create          roles:assign          roles:revoke
billing:read          billing:update
webhooks:manage
```

`HasOrgPermission(ctx, userID, orgID, "members", "invite")` fetches all `org_role_permissions` rows for the user's org roles in that org and returns true if any matches exactly or via wildcard.

### 3.3 Builtin org roles (auto-seeded on org creation)

`rbac.SeedOrgRoles(ctx, orgID)` is called inside `handleCreateOrganization` after the `organizations` row commits, before HTTP response.

| Role | Permissions |
|------|-------------|
| `owner` | `(*,*)` — full wildcard |
| `admin` | `members:*`, `org:update`, `roles:create`, `roles:assign`, `roles:revoke`, `webhooks:manage` |
| `member` | `members:read`, `org:read` |

All inserted with `is_builtin=1`. Seeding idempotent via `INSERT OR IGNORE`.

**Membership tier → org role sync (app-layer)**: every write to `organization_members` also calls `rbac.GrantOrgRole` with the builtin role matching the tier string. In `handleUpdateOrganizationMemberRole`, after enum update, revoke old builtin + grant new.

### 3.4 RBAC manager extension

Methods on `rbac.Manager` in `internal/rbac/rbac.go`:

```go
HasOrgPermission(ctx, userID, orgID, action, resource string) (bool, error)
GetEffectiveOrgPermissions(ctx, userID, orgID string) ([]Permission, error)
CreateOrgRole(ctx, orgID, name, description string) (*OrgRole, error)
DeleteOrgRole(ctx, orgID, roleID string) error            // ErrBuiltinRole if is_builtin
GrantOrgRole(ctx, orgID, userID, roleID, grantedBy string) error
RevokeOrgRole(ctx, orgID, userID, roleID string) error
AttachOrgPermission(ctx, orgRoleID, action, resource string) error
DetachOrgPermission(ctx, orgRoleID, action, resource string) error
SeedOrgRoles(ctx, orgID string) error                     // idempotent
```

Storage interface additions (`internal/storage/storage.go`):

```go
CreateOrgRole(ctx, orgID, id, name, description string, isBuiltin bool) error
GetOrgRoleByID(ctx, roleID string) (*OrgRole, error)
GetOrgRolesByOrgID(ctx, orgID string) ([]*OrgRole, error)
GetOrgRolesByUserID(ctx, userID, orgID string) ([]*OrgRole, error)
GetOrgRoleByName(ctx, orgID, name string) (*OrgRole, error)
UpdateOrgRole(ctx, roleID, name, description string) error
DeleteOrgRole(ctx, roleID string) error
AttachOrgPermission(ctx, orgRoleID, action, resource string) error
DetachOrgPermission(ctx, orgRoleID, action, resource string) error
GetOrgRolePermissions(ctx, orgRoleID string) ([]Permission, error)
GrantOrgRole(ctx, orgID, userID, orgRoleID, grantedBy string) error
RevokeOrgRole(ctx, orgID, userID, orgRoleID string) error
GetOrgUserRoles(ctx, userID, orgID string) ([]*OrgRole, error)
```

`OrgRole` struct: `ID`, `OrganizationID`, `Name`, `Description`, `IsBuiltin`, `CreatedAt`, `UpdatedAt`.

### 3.5 Middleware refactor

New file `internal/rbac/org_middleware.go`:

```go
func RequireOrgPermission(rbacMgr *Manager, action, resource string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            orgID := chi.URLParam(r, "org_id")
            userID := middleware.GetUserID(r.Context())
            ok, err := rbacMgr.HasOrgPermission(r.Context(), userID, orgID, action, resource)
            if err == ErrNotMember {
                writeError(w, http.StatusNotFound, "organization not found")
                return
            }
            if err != nil || !ok {
                writeError(w, http.StatusForbidden, "insufficient org permission")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Mounted on chi sub-router at `/api/v1/organizations/{org_id}` so `{org_id}` is already resolved. Non-member returns 404 (don't leak existence); insufficient permission returns 403.

Route → permission map:

| Route | Middleware |
|-------|-----------|
| `PATCH /organizations/{org_id}` | `RequireOrgPermission("org","update")` |
| `DELETE /organizations/{org_id}` | `RequireOrgPermission("org","delete")` |
| `PATCH /organizations/{org_id}/members/{user_id}/role` | `RequireOrgPermission("members","update_role")` |
| `DELETE /organizations/{org_id}/members/{user_id}` | `RequireOrgPermission("members","remove")` |
| `POST /organizations/{org_id}/invitations` | `RequireOrgPermission("members","invite")` |

### 3.6 Handler simplification

Before (`handleUpdateOrganization`):

```go
if !h.requireOrgRole(w, r.Context(), orgID, userID, OrgRoleOwner, OrgRoleAdmin) {
    return
}
// ...handler body
```

After (router):

```go
r.With(rbac.RequireOrgPermission(h.rbacMgr, "org", "update")).
    Patch("/organizations/{org_id}", h.handleUpdateOrganization)
```

After (handler — guard removed):

```go
func (h *Handler) handleUpdateOrganization(w http.ResponseWriter, r *http.Request) {
    orgID := chi.URLParam(r, "org_id")
    // decode body, call storage, respond
}
```

Apply removal to all 5 handlers. Once no call sites remain, delete `requireOrgRole` helper at `organization_handlers.go:461`.

### 3.7 API endpoints — org role CRUD

New file `internal/api/org_rbac_handlers.go`. All sub-mounted under `/api/v1/organizations/{org_id}`.

```
GET    /organizations/{org_id}/roles                              handleListOrgRoles      (roles:read)
POST   /organizations/{org_id}/roles                              handleCreateOrgRole     (roles:create)
GET    /organizations/{org_id}/roles/{role_id}                    handleGetOrgRole        (roles:read)
PATCH  /organizations/{org_id}/roles/{role_id}                    handleUpdateOrgRole     (roles:create)
DELETE /organizations/{org_id}/roles/{role_id}                    handleDeleteOrgRole     (roles:create)  -- 409 if is_builtin
POST   /organizations/{org_id}/members/{user_id}/roles/{role_id}  handleGrantOrgRole      (roles:assign)
DELETE /organizations/{org_id}/members/{user_id}/roles/{role_id}  handleRevokeOrgRole     (roles:revoke)
GET    /organizations/{org_id}/members/{user_id}/permissions      handleGetEffectivePerms (members:read)
```

### 3.8 SeedDefaultRoles wiring (drive-by fix)

In `internal/server/server.go`, inside `Build()` after migrations succeed and before `srv.ListenAndServe`:

```go
seedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := rbacMgr.SeedDefaultRoles(seedCtx); err != nil {
    return nil, fmt.Errorf("seeding default roles: %w", err)
}
```

`SeedDefaultRoles` is already idempotent (`INSERT OR IGNORE`). Safe on every restart.

### 3.9 Audit log wiring (drive-by fix)

Use existing `AuditLogger.Log(ctx, &storage.AuditLog{...})` pattern from `organization_handlers.go:545`.

Global RBAC handlers (`internal/api/rbac_handlers.go`):

| Handler | Action |
|---------|--------|
| `handleCreateRole` | `rbac.role.create` |
| `handleAssignRole` | `rbac.role.assign` |
| `handleRevokeRole` | `rbac.role.revoke` |
| `handleAttachPermission` | `rbac.permission.attach` |
| `handleDetachPermission` | `rbac.permission.detach` |

Org RBAC handlers (`internal/api/org_rbac_handlers.go`):

| Handler | Action |
|---------|--------|
| `handleCreateOrgRole` | `org.role.create` |
| `handleDeleteOrgRole` | `org.role.delete` |
| `handleGrantOrgRole` | `org.role.grant` |
| `handleRevokeOrgRole` | `org.role.revoke` |
| `handleUpdateOrgRole` (perm changes) | `org.permission.attach` / `org.permission.detach` |

`organization_handlers.go` `handleUpdateOrganizationMemberRole` — add `org.member.role_update` audit (currently missing). Metadata: `old_role`, `new_role`, `target_user_id`, `org_id`.

### 3.10 Backward compat

Global RBAC tables untouched. `organization_members.role` enum unchanged. New org RBAC system is purely additive. Existing org test suite passes unchanged.

### 3.11 Test plan

Unit additions to `internal/rbac/rbac_test.go`:

- `TestHasOrgPermission_Wildcard`, `TestHasOrgPermission_NoRole`
- `TestSeedOrgRoles_Idempotent` (call twice, exactly 3 rows)
- `TestCreateOrgRole_DuplicateName` (unique-constraint wrapped error)
- `TestDeleteOrgRole_Builtin_Refused` (`ErrBuiltinRole`)

Integration `TestOrgRBAC_FullFlow`:

1. Create org → 3 builtin roles seeded
2. Create custom role `editor`
3. `AttachOrgPermission(editor, "org", "update")`
4. `GrantOrgRole(B, editor)`
5. `HasOrgPermission(B, orgID, "org", "update")` → true
6. PATCH org as user B → 200
7. `RevokeOrgRole(B, editor)`
8. PATCH org as user B → 403

Regression: existing `TestOrganization*` suite passes unchanged — middleware behavior equivalent to removed `requireOrgRole` guard.

---

## 4. Applications + Redirect URI Allowlist

### 4.1 Schema

Migration files (identical): `cmd/shark/migrations/00008_applications.sql` AND `internal/testutil/migrations/00008_applications.sql`.

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS applications (
  id                    TEXT NOT NULL PRIMARY KEY,         -- app_<nanoid>
  name                  TEXT NOT NULL,
  client_id             TEXT NOT NULL UNIQUE,              -- shark_app_<nanoid>
  client_secret_hash    TEXT NOT NULL,                     -- SHA-256 hex
  client_secret_prefix  TEXT NOT NULL,                     -- first 8 chars (UX display)
  allowed_callback_urls TEXT NOT NULL DEFAULT '[]',        -- JSON array
  allowed_logout_urls   TEXT NOT NULL DEFAULT '[]',        -- JSON array
  allowed_origins       TEXT NOT NULL DEFAULT '[]',        -- JSON array
  is_default            BOOLEAN NOT NULL DEFAULT 0,
  metadata              TEXT NOT NULL DEFAULT '{}',
  created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_applications_one_default
  ON applications(is_default) WHERE is_default = 1;

-- +goose Down
DROP INDEX IF EXISTS idx_applications_one_default;
DROP TABLE IF EXISTS applications;
```

SQLite partial indexes require ≥ 3.8.0. Shark embeds `modernc.org/sqlite` (3.45+) — no concern.

### 4.2 Default app seeding

New file `internal/server/seed_app.go`. Wired in `server.go` after `goose.Up` returns, before `net.Listen`:

```go
if err := seedDefaultApplication(ctx, s.store, s.cfg); err != nil {
    return fmt.Errorf("seed default application: %w", err)
}
```

Logic:

1. `store.GetDefaultApplication(ctx)` — if found, return nil.
2. If error is not `ErrNotFound`, return err.
3. `clientID = "shark_app_" + gonanoid.New(21)`.
4. `clientSecret = base62(crypto/rand[32 bytes])` (~43 chars).
5. `secretHash = hex(sha256(clientSecret))`, `secretPrefix = clientSecret[:8]`.
6. Build `allowed_callback_urls`: append `cfg.Social.RedirectURL` if non-empty; append `cfg.MagicLink.RedirectURL` if non-empty + distinct.
7. `store.CreateApplication(ctx, app)`.
8. Print banner to stdout (matches admin API key bootstrap at `server.go:218`):

```
============================================================
  Default application created
  client_id:     shark_app_xxxxxxxxxxxxxxxxxxxxxxxxxxx
  client_secret: <full-secret>   (shown once — save it)
============================================================
```

### 4.3 redirect_validator package

New package `internal/auth/redirect/`. File `redirect.go`:

```go
package redirect

import (
    "errors"
    "net/url"
    "strings"
)

var (
    ErrNotAllowed = errors.New("redirect URL not in allowlist")
    ErrInvalidURL = errors.New("redirect URL not parseable")
)

type Kind int
const (
    KindCallback Kind = iota
    KindLogout
    KindOrigin
)

type Application struct {
    AllowedCallbackURLs []string
    AllowedLogoutURLs   []string
    AllowedOrigins      []string
}

func Validate(app *Application, kind Kind, requestedURL string) error
```

`Validate` rules in order:

1. `url.Parse(requestedURL)` → `ErrInvalidURL` on err or empty scheme.
2. Allowed schemes: `http`, `https`, plus custom schemes for native apps (e.g., `com.example.app`). Reject empty or whitespace-containing schemes.
3. `u.User != nil` → `ErrNotAllowed` (userinfo banned).
4. `u.Fragment != ""` → `ErrNotAllowed` (OAuth 2.1 §3.1.2).
5. Select allowlist by `kind`.
6. Normalize via `normalize(u *url.URL) string`: lowercase scheme + host, strip default port (`:80`/`:443`), strip trailing `/` if path is exactly `/`.
7. Iterate allowlist:
   - Normalize pattern same way.
   - **Exact match** → allow.
   - **Wildcard**: pattern starts with `https://*.` → extract base domain. Requested host must be one subdomain label + `.` + base. Subdomain itself must not contain `.`. No path wildcard.
   - **Loopback**: pattern is exactly `http://127.0.0.1` or `http://localhost` (post-normalize) AND requested scheme `http` AND requested host (port-stripped) is `127.0.0.1` or `localhost` → allow (RFC 8252 §8.3).
8. No match → `ErrNotAllowed`.

### 4.4 Apply to existing surfaces

`oauth_handlers.go` around line 133 — before/after:

```go
// BEFORE
http.Redirect(w, r, cfg.Social.RedirectURL+"?code="+token, http.StatusFound)

// AFTER
defaultApp, err := h.store.GetDefaultApplication(r.Context())
if err != nil {
    http.Error(w, "internal error", http.StatusInternalServerError)
    return
}
redirectTarget := r.URL.Query().Get("redirect_uri")
if redirectTarget == "" {
    redirectTarget = h.cfg.Social.RedirectURL
}
appModel := redirect.Application{AllowedCallbackURLs: defaultApp.AllowedCallbackURLs}
if err := redirect.Validate(&appModel, redirect.KindCallback, redirectTarget); err != nil {
    http.Error(w, "redirect_uri not allowed: "+err.Error(), http.StatusBadRequest)
    return
}
http.Redirect(w, r, redirectTarget+"?code="+token, http.StatusFound)
```

Apply identical pattern at `magiclink_handlers.go:170` with `cfg.MagicLink.RedirectURL` fallback. Per-request `GetDefaultApplication` is acceptable — single indexed row read.

### 4.5 CLI commands

Files: `cmd/shark/cmd/app.go` (parent + register) plus subcommand files. All open storage via `serve.go` pattern: `config.Load` → `storage.Open(cfg.Storage.Path)` → `defer store.Close()`.

**`shark app create`** (`app_create.go`)

Flags: `--name` (required), `--callback` (string array), `--logout` (string array), `--origin` (string array). Validate URLs via redirect package normalization. Generate clientID + secret. Insert row. Print banner.

**`shark app list`** (`app_list.go`)

`store.ListApplications(ctx, 100, 0)`. Render via `text/tabwriter`: `ID | NAME | CLIENT ID | CALLBACKS | CREATED`.

**`shark app show <id_or_client_id>`** (`app_show.go`)

Try `GetApplicationByID`; fallback `GetApplicationByClientID`. Print aligned key:value. JSON arrays via `json.MarshalIndent`. Never print secret hash.

**`shark app update <id_or_client_id>`** (`app_update.go`)

Flags: `--name`, `--add-callback`, `--remove-callback`, `--add-logout`, `--remove-logout`, `--add-origin`, `--remove-origin`. Fetch, mutate, validate URLs, `store.UpdateApplication`.

**`shark app rotate-secret <id_or_client_id>`** (`app_rotate.go`)

Generate new secret + hash + prefix. `store.RotateApplicationSecret`. Print new secret once with banner. Warn: "Old secret immediately invalid."

**`shark app delete <id_or_client_id>`** (`app_delete.go`)

Flags: `--yes`. Refuse if `is_default=true`. Confirmation prompt unless `--yes` (use `isatty.IsTerminal` pattern from init.go). `store.DeleteApplication`.

### 4.6 Storage interface methods

In `internal/storage/storage.go` Store interface, implement on `SQLiteStore`:

```go
type Application struct {
    ID                  string
    Name                string
    ClientID            string
    ClientSecretHash    string
    ClientSecretPrefix  string
    AllowedCallbackURLs []string   // deserialized from JSON
    AllowedLogoutURLs   []string
    AllowedOrigins      []string
    IsDefault           bool
    Metadata            map[string]any
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

CreateApplication(ctx, app *Application) error
GetApplicationByID(ctx, id string) (*Application, error)
GetApplicationByClientID(ctx, clientID string) (*Application, error)
GetDefaultApplication(ctx) (*Application, error)
ListApplications(ctx, limit, offset int) ([]*Application, error)
UpdateApplication(ctx, app *Application) error
RotateApplicationSecret(ctx, id, newHash, newPrefix string) error
DeleteApplication(ctx, id string) error
```

Implement in new file `internal/storage/applications.go`. JSON columns serialized via `encoding/json`. `UpdateApplication` sets `updated_at = CURRENT_TIMESTAMP` in SQL.

### 4.7 Config interaction

Mark `cfg.Social.RedirectURL` and `cfg.MagicLink.RedirectURL` as deprecated aliases:

```go
// Deprecated: migrated to default application's allowed_callback_urls on first boot.
// Removal target: Phase 6 (/oauth/authorize landing).
RedirectURL string `koanf:"redirect_url"`
```

Read at startup, passed to `seedDefaultApplication`, ignored at runtime thereafter. Add note to README.md config reference.

### 4.8 HTTP admin API

New file `internal/api/application_handlers.go`. `AppHandler` struct holding `store` + `auditLogger`. All gated by existing `AdminAPIKeyFromStore`.

| Method | Path | Action |
|--------|------|--------|
| `POST` | `/api/v1/admin/apps` | create |
| `GET` | `/api/v1/admin/apps` | list (`limit`, `offset`) |
| `GET` | `/api/v1/admin/apps/{id}` | show (id or client_id) |
| `PATCH` | `/api/v1/admin/apps/{id}` | update |
| `DELETE` | `/api/v1/admin/apps/{id}` | delete |
| `POST` | `/api/v1/admin/apps/{id}/rotate-secret` | rotate |

Create + rotate responses include `client_secret` once. Others omit.

### 4.9 ID prefixes

| Prefix | Entity |
|--------|--------|
| `app_` | `applications.id` (internal) |
| `shark_app_` | `applications.client_id` (public OAuth client identifier) |
| `orgrole_` | `org_roles.id` |

Update RM.md prefix table.

### 4.10 Audit logging

Every mutation:

- `app.create` (actor: admin, target: app ID)
- `app.update` (actor: admin, target: app ID, metadata: changed fields)
- `app.delete` (actor: admin, target: app ID)
- `app.secret.rotate` (actor: admin, target: app ID)

Actor type `"admin"` (admin-key-gated, no user context).

### 4.11 Test plan

Unit `internal/auth/redirect/redirect_test.go` (table-driven):

- `TestValidate_ExactMatch`, `TestValidate_WildcardSubdomain`, `TestValidate_WildcardNoPathInjection`
- `TestValidate_LoopbackAnyPort`, `TestValidate_LoopbackProductionRejected`
- `TestValidate_BadScheme` (`javascript:` rejected), `TestValidate_Userinfo`, `TestValidate_Fragment`
- `TestValidate_NormalizationLowercase`, `TestValidate_TrailingSlash`

Unit `internal/storage/applications_test.go`:

- `TestApplicationCRUD`
- `TestApplicationDefaultUniqueIndex` (two `is_default=1` → constraint error)

Integration `cmd/shark/cmd/app_test.go`:

- `TestE2E_AppCreate` — runs CLI, asserts row via `GetApplicationByClientID`
- `TestE2E_AppRotateSecret` — new hash ≠ old hash
- `TestE2E_AppDeleteDefault_Refused` — exit 1, stderr message

E2E additions to `internal/api/oauth_handlers_test.go`:

- `TestOAuthCallback_RedirectInAllowlist_Succeeds` (302)
- `TestOAuthCallback_RedirectNotAllowed_400` (body contains "not allowed")

---

## 5. Test Strategy, Verification Gate & Executor Waves

### 5.1 Existing test inventory

26 test files across `internal/` and `internal/testutil/cli/`. Harness already production-quality: `internal/testutil/server.go` has `NewTestServer(t)` (real SQLite + chi router), `NewTestDB(t)` (embedded migrations), HTTP client with cookie jar. CLI harness at `internal/testutil/cli/harness.go` boots a live listener for binary-level e2e. **No new harness needed**; extend in place.

Files most affected and break vectors:

| File | Coverage | Phase 3 break vector |
|------|----------|----------------------|
| `internal/api/middleware/auth.go` | `RequireSessionFunc` | Signature gains `jwtMgr` — every call site updates |
| `internal/api/auth_handlers_test.go` | login, signup, /me | Login response shape adds `access_token`; /me accepts Bearer |
| `internal/api/oauth_handlers_test.go` | OAuth callback | Redirect validator applied — tests with hardcoded URIs need default app fixture |
| `internal/api/magiclink_handlers_test.go` | magic-link | Same redirect risk; JWT response fields |
| `internal/api/organizations_test.go` | org CRUD, invite | Middleware refactor changes auth path |
| `internal/api/rbac_handlers_test.go` | global RBAC | Manager extension; existing tests must compile |
| `internal/api/sso_handlers_test.go` | SAML/OIDC | JWT response additions |
| `internal/storage/phase2_test.go` | org + webhook storage | New migrations 00006-00008 must be in fixture |
| `internal/testutil/cli/e2e_test.go` | full serve | `NewServer` now takes `jwtMgr`; first boot auto-generates key |
| `internal/rbac/rbac_test.go` | global RBAC | Adds `HasOrgPermission`; existing must still compile |

Session-only files (`password_test.go`, `apikey_test.go`, `audit_test.go`, `bodylimit_test.go`, `ratelimit_test.go`) unaffected.

### 5.2 Per-component test matrix

| Component | Unit | Integration | E2E |
|-----------|------|-------------|-----|
| **JWT** | `TestIssueSession`, `TestIssueAccessRefresh`, `TestValidate_*` (Valid/Expired/AlgConfusion/UnknownKid/BadSignature), `TestRefresh_Rotates`, `TestRevoke_*`, `TestKeyRotation_BothInJWKS` | `TestJWTManager_FullCycle` (real DB key gen + sign + validate + revoke) | `TestE2E_LoginToJWTToMe`, `TestE2E_RefreshFlow`, `TestE2E_RevokedTokenRejected` |
| **Middleware** | `TestRequireSession_BearerOnly/CookieOnly/BothBearerWins/BearerMalformed/CookieExpired/Refresh_Rejected/NoCredentials_401` | `TestE2E_MeBothAuthModes` | `TestE2E_CookieAndJWTCoexist` |
| **Org RBAC** | `TestHasOrgPermission_Owner/Member_NoGrant/CustomRole`, `TestSeedOrgRoles_Idempotent`, `TestCreateOrgRole_OK/BuiltinConflict`, `TestDeleteOrgRole_Builtin_Rejected/Custom_OK` | `TestOrgRBAC_FullFlow` | `TestE2E_CustomRoleGrantsAccess` |
| **Apps + Redirect** | 10 `TestValidate_*` cases, `TestApplicationCRUD`, `TestApplicationDefaultUniqueIndex` | `TestE2E_AppCreate_RowExists`, `TestE2E_AppRotateSecret` | `TestE2E_OAuthRedirectAllowed`, `TestE2E_OAuthRedirectBlocked` |
| **Cross-cutting** | — | `TestE2E_AuditLogsWritten` | `TestE2E_FullPhase3_GoldenPath` (GP1–GP10) |

### 5.3 Integration test harness extensions

Three in-place extensions to `internal/testutil/server.go`:

- **A — JWT-aware construction**: After Wave 1 Agent A ships `internal/auth/jwt`, add `WithJWTManager(jm)` option to `api.NewServer`. If no option passed, auto-generate ephemeral signing key at startup.
- **B — Bearer helpers**: Add `GetWithBearer(path, token)` and `PostJSONWithBearer(path, body, token)` alongside existing `GetWithAdminKey`.
- **C — Test migration parity**: After each Wave 1 migration lands, copy into `internal/testutil/migrations/` before Wave 2. Agent K enforces.

CLI harness needs no changes; `TestE2EServeFlow` exercises JWT auto-key-generation naturally.

### 5.4 E2E flows

**Golden paths (GP1–GP10) — must pass**:

- **GP1** Password login → response has `access_token` → `GET /me` Bearer → 200
- **GP2** Same login → `shark_session` cookie set → `GET /me` cookie-only → 200
- **GP3** Both cookie + Bearer → 200, `AuthMethod=jwt`, cookie session not touched
- **GP4** `POST /auth/revoke` own JWT → 204 → `GET /me` same JWT (with `check_per_request=true`) → 401 `code:"token_revoked"`
- **GP5** Org owner creates `inviter` role with `members:invite` → grants member → member POSTs invitation → 200
- **GP6** Member without `members:invite` → same endpoint → 403
- **GP7** `shark app create --name myapp --callback https://app.example.com/cb` → exit 0 → restart → `shark app list` shows row
- **GP8** OAuth `authorize` with `redirect_uri=https://app.example.com/cb` (in allowlist) → 302 to that URI
- **GP9** OAuth `authorize` with `redirect_uri=https://evil.example.com` → 400 HTML error (no redirect)
- **GP10** `shark keys generate-jwt` → JWKS returns 1 key → `--rotate` → JWKS returns 2 keys (old still valid)

**Regressions — must not break**:

- **R1** `TestRequireSessionFunc` passes (signature change atomically updated in Wave 2 D)
- **R2** `organizations_test.go` passes (middleware refactor preserves cookie-auth behavior)
- **R3** `oauth_handlers_test.go` passes (redirect validation passes when default app has cfg URL migrated in)

### 5.5 Verification gate

`Makefile` (extends existing `test` target):

```makefile
.PHONY: verify

verify:
	go vet ./...
	go test -race -count=1 ./...
	go test -race -count=1 -tags=integration ./...
	go test -race -count=1 -tags=e2e ./internal/testutil/e2e/...
```

Build tags: `//go:build integration` (real SQLite, slower), `//go:build e2e` (under `internal/testutil/e2e/`, GP1-GP10). Plain `go test ./...` runs unit only. CI runs `make verify`.

### 5.6 Wave plan for executor agents

All execution agents are Sonnet (per user directive — no Opus for boilerplate). Each gets a self-contained brief, files-to-read, files-to-modify, atomic commit boundary, acceptance criteria.

#### Wave 1 — schema + foundations (3 parallel agents)

**Agent A — JWT package**
- Read: `internal/auth/`, `internal/storage/sqlite_test.go`, `cmd/shark/migrations/`
- Create/modify: `internal/auth/jwt/{manager,keys,revoke}.go`, `cmd/shark/migrations/00006_jwt.sql`, `internal/testutil/migrations/00006_jwt.sql`, `cmd/shark/cmd/keys.go`, JWT config struct in `internal/config/config.go`
- Commit: `feat(auth): JWT package + signing-key migration + keys CLI`
- Acceptance: `go build ./internal/auth/jwt/...` succeeds; `TestIssueSession` + `TestValidate_Valid` pass; `shark keys generate-jwt` exits 0

**Agent B — Org RBAC storage**
- Read: `internal/rbac/rbac_test.go`, `internal/storage/`, `cmd/shark/migrations/00004_organizations.sql`
- Create/modify: `cmd/shark/migrations/00007_org_rbac.sql`, `internal/testutil/migrations/00007_org_rbac.sql`, `internal/storage/org_rbac.go`, `internal/rbac/rbac.go` (add `HasOrgPermission`, `SeedOrgRoles`, full extension list from §3.4)
- Commit: `feat(rbac): org-scoped roles migration + storage + Manager extensions`
- Acceptance: `go test ./internal/rbac/... ./internal/storage/...` passes; `TestHasOrgPermission_Owner` passes

**Agent C — Applications + redirect validator**
- Read: `internal/api/oauth_handlers.go`, `internal/api/magiclink_handlers.go`, `cmd/shark/migrations/`
- Create/modify: `cmd/shark/migrations/00008_applications.sql`, `internal/testutil/migrations/00008_applications.sql`, `internal/storage/applications.go`, `internal/auth/redirect/redirect.go`
- Commit: `feat(apps): applications migration + redirect validator package`
- Acceptance: `go build ./internal/auth/redirect/...` succeeds; `TestValidate_ExactMatch` + `TestValidate_OpenRedirectBlocked` (i.e. `TestValidate_WildcardNoPathInjection`) pass

#### Wave 2 — wiring (3 agents, after Wave 1 merges)

**Agent D — Middleware refactor + JWT issuance**
- Depends on: Agent A
- Read: `internal/api/middleware/auth.go`, `internal/api/router.go`, all login-issuing handlers
- Modify: `internal/api/middleware/auth.go` (dual-accept), `internal/api/router.go` (pass jwtMgr to all 10 sites), all login handlers (issue JWT alongside cookie), `internal/api/session_handlers.go` (add `/auth/revoke`), new `internal/api/well_known_handlers.go` (JWKS endpoint), `internal/server/server.go` (instantiate + EnsureActiveKey)
- Commit: `feat(middleware): dual-accept JWT+cookie + access/refresh token issuance + JWKS endpoint`
- Acceptance: `go test ./internal/api/... -run TestRequireSession` passes; `TestE2E_MeBothAuthModes` passes

**Agent E — Org RBAC middleware + handler simplification + role CRUD + SeedDefaultRoles wiring**
- Depends on: Agent B
- Read: `internal/api/organization_handlers.go`, `internal/api/rbac_handlers.go`, `internal/rbac/rbac.go`
- Modify: `internal/api/organization_handlers.go` (remove inline `requireOrgRole` calls, delete helper), new `internal/rbac/org_middleware.go`, new `internal/api/org_rbac_handlers.go`, `internal/api/router.go` (mount new routes + RequireOrgPermission middleware), `internal/server/server.go` (call `SeedDefaultRoles` at boot)
- Commit: `feat(rbac): org-scoped permission middleware + role CRUD + default role seeding`
- Acceptance: `go test ./internal/api/... -run TestOrgRBAC` passes; existing `organizations_test.go` passes

**Agent F — Apps CLI + admin HTTP + redirect validator applied**
- Depends on: Agent C
- Read: `internal/api/oauth_handlers.go:133`, `internal/api/magiclink_handlers.go:170`, `cmd/shark/cmd/`
- Modify: `cmd/shark/cmd/app.go` + subcommand files (`app_create.go`, `app_list.go`, `app_show.go`, `app_update.go`, `app_rotate.go`, `app_delete.go`), `internal/api/application_handlers.go` (admin CRUD), `internal/api/oauth_handlers.go` + `magiclink_handlers.go` (call `redirect.Validate`), new `internal/server/seed_app.go`, `internal/server/server.go` (call `seedDefaultApplication`)
- Commit: `feat(apps): app CLI + admin endpoints + redirect validation at OAuth/magic-link + default app seeding`
- Acceptance: `shark app create` exits 0; `TestE2E_OAuthRedirect*` passes

#### Wave 3 — tests + audit (4 parallel agents)

**Agent G — JWT + middleware unit tests**
- Create: `internal/auth/jwt/manager_test.go`, `internal/api/middleware/auth_jwt_test.go`
- Acceptance: all `TestIssueSession*`, `TestValidate_*`, `TestRequireSession_Bearer*/Cookie*` pass with `-race`

**Agent H — RBAC unit + org integration tests**
- Create: `internal/rbac/org_rbac_test.go`, `internal/api/org_rbac_integration_test.go` (build tag `integration`)
- Acceptance: `TestHasOrgPermission_*`, `TestSeedOrgRoles_Idempotent`, `TestOrgRBAC_FullFlow` pass

**Agent I — Redirect + apps unit + CLI integration tests**
- Create: `internal/auth/redirect/redirect_test.go`, `internal/storage/applications_test.go`, `internal/api/application_handlers_test.go`, CLI integration test in `internal/testutil/cli/`
- Acceptance: all 10 `TestValidate_*` cases pass; `TestApplicationCRUD` + `TestApplicationDefaultUniqueIndex` pass

**Agent J — Audit log wiring**
- Read: `internal/audit/`, all RBAC + app + org handler files from Wave 2
- Modify: insert `audit.Log(...)` calls at every mutation per §3.9 + §4.10
- Acceptance: `go test ./internal/audit/... -run TestE2E_AuditLogsWritten` passes; no regression

#### Wave 4 — final verification (1 agent)

**Agent K — E2E golden path suite + Makefile + full verify run**
- Read: all handler files, `internal/testutil/`, `Makefile`
- Create: `internal/testutil/e2e/phase3_test.go` (build tag `e2e`, covers GP1-GP10)
- Modify: `Makefile` (add `verify` target)
- Run: `make verify` — fix any compilation errors or test failures
- Commit: `test(e2e): Phase 3 golden-path suite + verify Makefile target`
- Acceptance: `make verify` exits 0, zero failures, zero races

### 5.7 Risks + rollback

- **Middleware signature break**: every call site in `router.go` must update atomically in W2 D. Mitigation: D runs full unit suite before commit.
- **Empty keys table on first boot**: token issuance fails. Mitigation: `EnsureActiveKey` auto-generates if none active. Manual `shark keys generate-jwt` becomes optional.
- **Default app seeding race**: two concurrent boots both insert. Mitigation: `INSERT OR IGNORE` + partial unique index on `is_default=1`. Second insert is silent no-op.
- **Redirect validator rejects existing URLs**: legitimate URIs not in default app allowlist after migration. Mitigation: Agent F's `seedDefaultApplication` copies `cfg.Social.RedirectURL` and `cfg.MagicLink.RedirectURL` into allowlist; mismatches log warning, don't fail startup.
- **Rollback**: every wave is a separate atomic commit. Revert any wave individually. Phase 3 schema is purely additive — no `DROP` or `ALTER COLUMN`. Reverting all of Phase 3 leaves DB consistent.

### 5.8 Estimated days

| Wave | Duration | Notes |
|------|----------|-------|
| Wave 1 (A+B+C parallel) | 1 day | Longest agent ~1 day; all parallel |
| Wave 2 (D+E+F, partial overlap) | 2 days | D and E can interleave once D's middleware compiles |
| Wave 3 (G+H+I+J parallel) | 1 day | Pure test authoring, no new interfaces |
| Wave 4 (K) | 1 day | E2E suite + fix-up pass |
| **Total** | **5 days optimistic / 7 with slack** | Slack absorbs middleware refactor surprises and audit wiring edge cases |

---

## 6. File Inventory (exhaustive)

### Created
- `cmd/shark/migrations/00006_jwt.sql` + mirror in testutil
- `cmd/shark/migrations/00007_org_rbac.sql` + mirror
- `cmd/shark/migrations/00008_applications.sql` + mirror
- `internal/auth/jwt/manager.go`, `keys.go`, `revoke.go`, `manager_test.go`
- `internal/auth/redirect/redirect.go`, `redirect_test.go`
- `internal/storage/org_rbac.go`
- `internal/storage/applications.go`, `applications_test.go`
- `internal/rbac/org_middleware.go`, `org_rbac_test.go`
- `internal/api/well_known_handlers.go`
- `internal/api/org_rbac_handlers.go`
- `internal/api/application_handlers.go`, `application_handlers_test.go`
- `internal/api/jwt_e2e_test.go`
- `internal/api/middleware/auth_jwt_test.go`
- `internal/api/org_rbac_integration_test.go`
- `internal/server/seed_app.go`
- `internal/testutil/e2e/phase3_test.go`
- `cmd/shark/cmd/keys.go`
- `cmd/shark/cmd/app.go`, `app_create.go`, `app_list.go`, `app_show.go`, `app_update.go`, `app_rotate.go`, `app_delete.go`, `app_test.go`

### Modified
- `internal/config/config.go` (add `JWTConfig`, deprecate Social/MagicLink RedirectURL fields)
- `internal/storage/storage.go` (add Application + OrgRole structs + interface methods)
- `internal/rbac/rbac.go` (add 9 org-scoped methods)
- `internal/api/middleware/auth.go` (refactor RequireSessionFunc + AuthMethodKey + GetClaims)
- `internal/api/router.go` (10 call sites + WithJWTManager + new routes mount + Server struct field)
- `internal/server/server.go` (instantiate jwtMgr + EnsureActiveKey + SeedDefaultRoles + seedDefaultApplication)
- `internal/api/auth_handlers.go` (issue JWT at login + JWT-aware logout)
- `internal/api/oauth_handlers.go` (issue JWT + redirect.Validate)
- `internal/api/sso_handlers.go` (issue JWT)
- `internal/api/magiclink_handlers.go` (issue JWT + redirect.Validate)
- `internal/api/session_handlers.go` (add `/auth/revoke` + comments)
- `internal/api/organization_handlers.go` (remove `requireOrgRole` calls, delete helper, add `org.member.role_update` audit)
- `internal/api/rbac_handlers.go` (add audit calls on every mutation)
- `internal/api/middleware/cors.go` (verified comment, no code change)
- `internal/api/mfa_handlers.go` (comments only)
- `internal/testutil/server.go` (WithJWTManager + Bearer helpers)
- `Makefile` (add `verify` target)
- `RM.md` (add `app_`, `shark_app_`, `orgrole_` prefixes; update CLI section)
- `README.md` (deprecate Social/MagicLink RedirectURL note; document `/.well-known/jwks.json`, `/api/v1/auth/revoke`, `shark app *`, `shark keys generate-jwt`)
- `cmd/shark/cmd/init.go` (Long help: mention JWT keys auto-generated on first boot)

### Deleted (after refactor)
- `requireOrgRole` helper at `internal/api/organization_handlers.go:461` (only after all 5 call sites removed in Wave 2 E)

---

## 7. Done Criteria

Phase 3 ships when:

1. ✅ All preexisting tests pass: `go test -race -count=1 ./...`
2. ✅ All new unit tests pass per §5.2 matrix
3. ✅ All integration tests pass: `go test -race -count=1 -tags=integration ./...`
4. ✅ All E2E tests pass: `go test -race -count=1 -tags=e2e ./internal/testutil/e2e/...`
5. ✅ `make verify` exits 0
6. ✅ Manual smoke: fresh `shark init && shark serve`, login via password, confirm both cookie + JWT issued in response, JWKS endpoint returns active key, `shark app list` shows default app
7. ✅ Audit log spot-check: trigger one RBAC grant + one app create, confirm both rows in `audit_logs`
8. ✅ ATTACK.md Phase 3 marked Done

---

## 8. Out of Scope (deferred)

- `/oauth/authorize` endpoint and full OAuth 2.1 server flow → Phase 6 (Agent Auth)
- OIDC Provider mode → Phase 8
- Token Vault → Phase 6
- Per-tenant config in Cloud → handled in Cloud fork (separate repo, Next.js + db-per-client SQLite)
- Edge sessions / standalone JWKS for edge auth → not in shark, defer indefinitely
- Cookie name configurability → not adding (see §2.4)
