# Frontend Wiring Gaps — Phase 6.7 Plan

**Generated:** 2026-04-20 via 4 parallel Explore-agent audit (W01-W15) + 8 parallel DX audit (W16-W22)
**Branch:** `claude/admin-vendor-assets-fix`
**Scope:** every place dashboard claims feature-done but UI is stub/dead/missing + cross-component DX gaps
**Source of truth:** Explore-agent reports compiled against current HEAD (`daabee1`)

**Companion docs:**
- `DEVEX_REVIEW_2026-04-20.md` — full 8-component DX scorecard + 24 top-3 fixes (origin of W16-W22)
- `HANDOFF.md` — Phase 6.7 live-dogfood session summary
- `.github/ISSUE_W15_multi_listener_proxy.md` — GitHub issue draft
- `CHANGELOG.internal.md` — Phase 6.7 entry

Sibling plan to `DASHBOARD_DX_EXECUTION_PLAN.md`. DX plan = polish. This plan = wiring debt + cross-cutting DX gaps. T06/T16/T24 from DX plan may overlap — log as blocked_by when relevant.

---

## Execution Protocol

1. Pick next `pending` task. Claim in `FRONTEND_WIRING_STATUS.json` (create on first run).
2. Atomic commit per task. Convention: `fix(wire): W<N> — <subject>` · `feat(wire): W<N> — <subject>`.
3. Typecheck gate: `cd admin && npx tsc -b --noEmit` before commit (runs despite `@ts-nocheck` since checker still parses).
4. Build gate: `go build ./...` for backend changes.
5. Bundle rebuild: `cd admin && npm run build` after any `admin/src/**.tsx` change. Stage renamed `internal/admin/dist/assets/index-*.js`.
6. Go binary rebuild: `go build -o bin/shark.exe ./cmd/shark` to dogfood.

---

## Evidence Summary (what's actually wrong)

### Confirmed P0 — dead UI

| # | Gap | File:line | User-visible | Root cause |
|---|-----|-----------|--------------|------------|
| E1 | Orgs "New" button does nothing | `organizations.tsx:57` | Clicking does literally nothing | `<button>` has no onClick |
| E2 | Impersonation — zero implementation | grep returns 0 hits anywhere | Feature user asked for, missing | ph:9 gate + no component file |
| E3 | Tokens nav renders empty_shell | `empty_shell.tsx:19` + `layout.tsx` | Nav entry with stub | No real tokens.tsx |
| E4 | API Explorer renders empty_shell | `empty_shell.tsx:20` + `layout.tsx:40` (ph:5) | Nav visible w/ CURRENT_PHASE=6, clicks to empty shell | No real component |
| E5 | Event Schemas renders empty_shell | `empty_shell.tsx:21` + `layout.tsx:42` (ph:5) | Same as E4 | No real component |
| E6 | Apps → Tokens tab empty | `applications.tsx:485` | "Token listing not yet available via API" | No backend list endpoint per-app |
| E7 | Consents page admin-wide gap | `consents_manage.tsx:11,225` | "Admin-wide consent viewing not yet available" | Missing admin endpoint |
| E8 | Flow builder 3 deferred steps | `flow_builder.tsx:38,41,44` | Disabled palette items "v0.2" | wired:false flag |
| E9 | Users bulk Assign-role / Add-to-org disabled | `users.tsx:156,157` | Buttons greyed, "CLI only" tooltip | No picker modal |

### Confirmed P1 — silent masking

| # | Gap | File:line |
|---|-----|-----------|
| E10 | App.tsx 5 silent catches swallow errors | `App.tsx:47,48,114,137,161` |
| E11 | audit.tsx silent catch swallows log fetch | `audit.tsx:92` |
| E12 | flow_builder.tsx silent catch on localStorage | `flow_builder.tsx:1366` |

### Confirmed P2 — stale "use CLI" escape hatches where backend DOES support it

| # | Gap | Current UX | Backend? |
|---|-----|------------|----------|
| E13 | Users Bulk Delete disabled "CLI only" | `users.tsx:174` | Endpoint exists — could wire |
| E14 | Settings Bulk user delete "CLI only" | `settings.tsx:330` | Same endpoint |
| E15 | Proxy YAML rules "edit YAML + reload" | `proxy_config.tsx:561` | DB-rules endpoint exists — YAML rules truly YAML-only |

### Dead backend (exists, nothing calls)

| # | Route | File:line |
|---|-------|-----------|
| E16 | `GET /permissions/{id}/roles` | `router.go:374` |
| E17 | `GET /permissions/{id}/users` | `router.go:375` |
| E18 | `POST /migrate/auth0` (stub returns notImplemented) | `router.go:450` |
| E19 | `GET /migrate/{id}` (stub) | `router.go:451` |

### User's specific complaints — verified status

| Claim | Reality |
|-------|---------|
| "api explorer not wired" | ✓ TRUE — stub renders, nav visible |
| "session debugger not wired" | ✗ FALSE — `session_debugger.tsx` full impl, client-side JWT decode + JWKS verify works (commit 6380fba) |
| "event schemas not wired" | ✓ TRUE — stub renders |
| "settings not writable" | ✓ TRUE — read-only. Phase 8/9 per HANDOFF. See **W11**. |
| "proxy not wired" | ✗ FALSE — `proxy_config.tsx` full CRUD for DB rules + simulator + circuit breaker stats. YAML rules correctly read-only. |
| "add button on users doesn't do nothing" | ✗ FALSE (in source) — `users.tsx:143,331` `CreateUserSlideover` fully wired (commit 2b4d649). **If broken at runtime, likely stale bundle cache OR JS error. Re-test with fresh `bin/shark.exe`.** |
| "impersonation down" | ✓ TRUE — zero impl. See **W02**. |

---

## Task Specs

### W01 — Orgs "New" button wire (P0)

**Files:** `admin/src/components/organizations.tsx`
**Backend:** `POST /admin/organizations` — verify exists, ship if missing
**Acceptance:**
- Line 57 button gets `onClick={() => setCreating(true)}` or equivalent
- `CreateOrgSlideover` component (mirror `CreateUserSlideover` T05 pattern): name (required), slug (auto-derive from name, editable), description (optional), metadata JSON (optional textarea)
- Submit → POST `/admin/organizations` → toast success + refresh list + select new row
- 409 → highlight slug field, error toast
- `?new=1` auto-opens slide-over
- ESC + backdrop close w/ discard confirm if dirty
**Verify:** smoke assertion: POST /admin/organizations returns 201, org appears in list.
**Commit:** `feat(wire): W01 — orgs create slide-over`

### W02 — Impersonation core flow (P0)

**Backend:** NEW
- `POST /admin/users/{id}/impersonate` — admin-key auth — mints short-lived (15 min) impersonation session token scoped to target user, records `admin.user.impersonate` audit with `actor_id`=admin + `target_id`=user + `reason` body field required. Returns `{token, expires_at, target_user}`.
- `POST /admin/impersonation/end` — revokes active impersonation session, records end audit
- `GET /admin/impersonation/current` — returns active impersonation if any for this admin
- Session claim `impersonator_id` set on all impersonation-minted tokens
- Audit entries flagged with `impersonation=true` field for filtering
**Frontend files:** new `admin/src/components/impersonation.tsx`, update `layout.tsx` nav (drop ph:9), update `App.tsx` route, add per-user "Impersonate" action button in `users.tsx` detail panel
**Acceptance:**
- Users detail → "Impersonate user" button (behind confirm with reason textarea)
- On submit → open new tab/window with session cookie set to impersonation token + visible red banner "Impersonating <email> · exit impersonation" at top
- Impersonation page lists active + past impersonations (from audit)
- "End impersonation" button revokes
**Dependencies:** none
**Verify:** smoke — admin impersonates, hits /me as impersonated user, ends, tokens revoked
**Commit:** backend + frontend two commits. `feat(wire): W02a — impersonation backend` + `feat(wire): W02b — impersonation dashboard`

### W03 — Drop ph:5 gates on API Explorer + Event Schemas OR build them (P1)

**Option A (ship-now):** remove API Explorer + Event Schemas from nav until built. Delete from `layout.tsx` NAV array + `App.tsx` routes + `empty_shell.tsx` exports. Zero user confusion.
**Option B (stub-honest):** keep nav entries, replace empty_shell with `<ComingSoon/>` that clearly states "Phase 7 SDK — shipping with SDK release. Track at <link>". Set real release date.
**Option C (build):** real API Explorer = embed Swagger UI or Scalar at `/admin/explorer` loading OpenAPI from `/api/openapi.json` (needs NEW backend endpoint generating spec from chi routes + handler reflection). Real Event Schemas = `GET /webhooks/events` (already exists) + per-event JSON schema docs.
**Recommendation:** A now (remove from nav), C as Phase 7 task.
**Files (Option A):** `layout.tsx`, `App.tsx`, `empty_shell.tsx`
**Commit:** `fix(wire): W03 — drop unbuilt nav entries (api-explorer, event-schemas)`

### W04 — Drop Tokens nav OR build (P1)

**Current:** nav entry → empty_shell stub "Phase 6". User was told tokens management shipped.
**Option A:** drop from nav. Tokens visible per-user (users detail → Sessions tab) + per-agent (agents detail → Tokens tab). Global tokens list rarely useful.
**Option B:** build real `tokens.tsx` — list all active tokens across users + agents + API keys, filter by type, revoke action. Backend: new `GET /admin/tokens?type=access|refresh|api_key&state=active` aggregating across tables.
**Recommendation:** A. Users/agents detail tabs cover it.
**Files (A):** `layout.tsx`, `App.tsx`, `empty_shell.tsx`
**Commit:** `fix(wire): W04 — drop tokens nav (coverage in per-user/agent tabs)`

### W05 — Apps Tokens tab (P1)

**File:** `admin/src/components/applications.tsx:485`
**Backend:** verify `GET /apps/{id}/tokens` or ship new endpoint
**Acceptance:** tab shows active tokens minted via this OAuth client_id: access tokens, refresh tokens, active count. Revoke-all per-app action.
**Commit:** `feat(wire): W05 — apps tokens tab real listing`

### W06 — Consents admin-wide listing (P1)

**Backend:** verify `GET /admin/oauth/consents` (per HANDOFF + CHANGELOG.internal.md — shipped). Frontend needs to use it.
**File:** `admin/src/components/consents_manage.tsx:11,225`
**Acceptance:** drop "not yet available" stub. Full admin consent listing: user, client, scopes, granted_at, revoke action. Filter by user_id / client_id.
**Commit:** `feat(wire): W06 — consents admin-wide listing`

### W07 — Flow builder 3 deferred steps (P2)

**Files:** `internal/authflow/steps.go` (3 new step dispatchers), `admin/src/components/flow_builder.tsx` (drop wired:false)
**Steps to wire:**
- `set_metadata` → `store.UpdateUser` metadata field merge
- `custom_check` → external HTTP call with timeout (like webhook step, but synchronous)
- `delay` → context-deadline sleep (rate-limit sim)
**Acceptance:** all 3 dispatchers ship, palette drops "v0.2" chip + disabled state, smoke assertion per step.
**Commit:** `feat(wire): W07 — flow builder deferred steps`

### W08 — Users bulk actions picker modal (P2)

**File:** `admin/src/components/users.tsx:156,157,174`
**Backend:** endpoints already exist — bulk roles + bulk org via looped calls or new `POST /admin/users/bulk-assign-role` + `POST /admin/users/bulk-add-to-org` + `DELETE /admin/users/bulk` (ship if missing).
**Acceptance:**
- "Assign role" bulk button opens picker modal (role multiselect), submits per selected user → toast "N updated"
- "Add to org" bulk opens org + role-in-org picker
- "Bulk Delete" opens confirm with typed-name pattern (already used in agents/vault) → iterates DELETE /users/{id}
**Commit:** `feat(wire): W08 — users bulk role/org/delete wire`

### W09 — App.tsx silent catches → slog/toast (P2)

**File:** `admin/src/components/App.tsx:47,48,114,137,161`
**Acceptance:** each `catch {}` or `.catch(() => {})` either logs via `console.warn` with context or surfaces via toast if user-facing.
**Commit:** `fix(wire): W09 — surface swallowed App.tsx errors`

### W10 — Audit + flow_builder silent catches (P2)

**Files:** `audit.tsx:92`, `flow_builder.tsx:1366`
**Commit:** `fix(wire): W10 — surface swallowed audit + flow errors`

### W11 — Settings writable (P3 — defer Phase 8)

**Not in this plan's scope.** Documented here for visibility. HANDOFF.md lines 56-62 covered: requires yaml.v3 Node API preserving comments + hot-reload OR 503 "restart required" + `PATCH /admin/config` handler. ~1 week of work. User confirmed acceptable as Phase 8/9.

**Minimum viable now:** add note on settings page top: "Config is read-only from dashboard. Edit `sharkauth.yaml` + restart, or use `shark config set <key> <value>` CLI. Phase 8 will enable dashboard edits."

**Commit (minimum):** `fix(wire): W11 — settings read-only note until Phase 8`

### W12 — Permission reverse lookup wire (P3)

**Files:** `rbac.tsx` / `rbac_matrix.tsx`
**Current:** frontend computes client-side. Dead backend routes E16/E17 exist.
**Option A:** use the endpoints (cleaner, scales past a few hundred permissions)
**Option B:** delete the endpoints (honesty — remove unused code)
**Recommendation:** A. Matrix view already uses `batch-usage` endpoint; single-permission detail could hit `/permissions/{id}/roles` + `/permissions/{id}/users` when drilling in.
**Commit:** `feat(wire): W12 — rbac drill-in uses permission reverse lookup`

### W13 — Remove or stub migrate/auth0 handler (P3)

**File:** `router.go:450-451`
**Current:** returns notImplemented stub. Dead route.
**Option A:** remove routes + handler. Frontend never used.
**Option B:** wire real Auth0 migration (much bigger scope — see `docs/migrations_research.md`).
**Recommendation:** A for now. Migration tooling = Phase 8/9.
**Commit:** `fix(wire): W13 — remove dead migrate/auth0 stub`

### W14 — Drop @ts-nocheck post-launch (P4 — deferred)

**Files:** all 42 `admin/src/components/*.tsx`
**Blocker:** type drift between Go structs + TS consumers. 9/24 Phase 6.6 bugs masked by this.
**Approach:** ship `tygo` config, generate TS types from `internal/api/*` response struct definitions, drop nocheck file-by-file, fix drift.
**Estimate:** 2-3 days.
**Not in this plan.** Track separately.

### W16 — Fix flow builder conditional save bug (P0, user-reported blocker)

**Problem:** Conditional step `then`/`else` branches silently lost on save round-trip. User adds webhook to `then` branch, clicks Save, toast says "Flow saved," branch vanishes on refresh. User-reported bug confirmed by DX audit C7. Root cause is frontend response-deserialization OR backend not re-hydrating nested steps from DB.

**Files:** `internal/api/flow_handlers.go:331` (UpdateFlow response path), `admin/src/components/flow_builder.tsx:293-299` (setFlow after save), `internal/storage/` (verify ConditionalStep persist path)
**Acceptance:**
1. Create conditional step with webhook in `then`, magic_link in `else`, Save
2. Hard refresh → both branches present
3. Add `TestUpdateFlow_ConditionalBranchesPreserved` integration test
4. If frontend state-sync is cause: add `console.warn` if draft round-trip loses branches
**Estimate:** 2h
**Commit:** `fix(wire): W16 — flow builder conditional branches survive save round-trip`

### W17 — AGENT_AUTH.md status rewrite (P0, launch trust gate)

**Problem:** `AGENT_AUTH.md:4` says "**Status: Design spec — not yet implemented**". Code ships 95% of Waves A-E. Every HN/Reddit reader who spot-checks drops trust. Single highest-ROI doc fix in repo per DEVEX_REVIEW finding F5.1.

**Files:** `AGENT_AUTH.md`
**Acceptance:**
1. Header line 4 → "**Status: MVP shipping (Waves A-E complete, 95% feature parity). Phase 6 next.**"
2. Competitive table line ~28: Token Vault row "No (v2)" → "Yes (v1)"; verify every "Shark (planned)" → "Shark"; verify Auth0/Okta claims (some mentioned beta may be GA now, some claimed shipped may be vapor)
3. New "Implementation Status" section after TL;DR: bullet list of shipped RFCs with commit SHAs
4. 48%/93%/97% stats: verify source or drop. If placeholder → drop. Bad stats in launch copy = credibility hit.
**Estimate:** 2h (research + edit)
**Commit:** `docs: W17 — AGENT_AUTH.md status rewrite (shipping, not design spec)`

### W18 — Structured error envelope across auth endpoints (P1)

**Problem:** `/api/v1/auth/**` + `/oauth/*` return `{error, message}` or `{error, error_description}` — no `code`, no `docs_url`, no `details`. Integrators can't distinguish `weak_password` variants (too-short vs common vs dictionary). Clerk/Auth0 ship RFC 7807 problem+detail.

**Files:**
- New: `internal/api/errors.go` — `ErrorResponse` struct with Error/Message/Code/DocsURL/Details
- New: `internal/oauth/errors.go` — RFC 6749 compliant `{error, error_description, error_uri}`
- Update: all handlers under `internal/api/auth*_handlers.go`, `mfa_handlers.go`, `passkey_handlers.go`, `magiclink_handlers.go`
- Update: all handlers under `internal/oauth/` + `internal/api/` that return OAuth errors

**Acceptance:**
1. Every auth error includes `code` (e.g., `auth/password/too_short`)
2. Every auth error includes `docs_url` pointing to `https://sharkauth.com/docs/errors#<code>` (docs site not live yet, but link stable)
3. `details` object with structured hints where useful (current_length, min_length, etc)
4. OAuth errors add `error_uri` per RFC 6749 §4.1.2.1
5. Smoke coverage for 5+ error paths verifying shape
**Estimate:** 2d (new struct + 40+ call-site updates + tests)
**Commit sequence:**
- `feat: W18a — ErrorResponse + OAuthError helpers`
- `fix: W18b — migrate all auth handlers to structured errors`
- `fix: W18c — migrate all oauth handlers to RFC-compliant errors`

### W19 — MFA pending-verification flag (P1)

**Problem:** User starts MFA enroll → closes tab → next enroll attempt blocked by `mfa_already_enabled` despite never verifying. DX audit C3 finding F3.2.

**Files:** `internal/api/mfa_handlers.go:88`, `internal/storage/` (new `mfa_pending_verification` column via migration 00017), `internal/auth/`
**Acceptance:**
1. New column `mfa_pending_verification BOOLEAN DEFAULT FALSE` on users table (migration 00017)
2. `handleMFAEnroll` sets pending=true
3. `handleMFAVerify` on success sets pending=false + enabled=true
4. Re-enroll attempt: if pending=true + enabled=false → overwrite secret, allow re-enroll
5. Smoke: enroll → abandon → re-enroll works
**Estimate:** 3h (backend + migration + smoke)
**Commit:** `fix: W19 — MFA pending-verification flag allows abandoned enrollment retry`

### W20 — Hosted auth pages + components toggle (P1, may need design workflow)

**Problem:** Today shark has no hosted sign-in/sign-up UI for end users. OAuth consent exists server-rendered, but a user of "my-app" that uses shark for auth must build their own login forms OR use CLI-only flows. Gap vs Clerk/Auth0 who ship `auth.<tenant>.<provider>.com` hosted pages + swappable components.

**See:** full design spec at `docs/superpowers/specs/2026-04-20-branding-hosted-components-design.md` (12 decisions + 7 open-question answers logged for bus-factor resilience).

**Scope (pending brainstorm):**
- Backend: new `/hosted/*` routes for public-facing sign-in/sign-up/magic-link/passkey/MFA pages, per-application config (branding, fields, providers to show)
- Backend: new `applications.hosted_auth_enabled BOOLEAN` flag + `hosted_config JSON` with colors/logo/copy overrides
- Frontend: new `admin/src/components/hosted_auth_config.tsx` — toggle per app, branding controls
- Frontend: server-rendered pages (HTML templates like current consent.html) OR embedded React SPA served at `/hosted/<app_slug>/login`
- SDK: `shark-components-react` / `shark-components-vue` — swappable embedded widgets for apps preferring inline

**Toggle semantics:**
- Per-app: "hosted" (shark renders pages) OR "components" (app uses SDK) OR "custom" (app fully rolls its own, uses API only)

**Dependencies:** W18 structured errors, W24 branding config (below), mail-builder (W22)
**Estimate:** Large — break down after brainstorm

### W21 — Mail builder + branding tab (P1, may need design workflow)

**Problem:** Email templates are hardcoded in `internal/email/templates/`. No admin UI to edit copy, subject lines, from-address, or preview. No branding controls (logo, colors, font) for either emails OR hosted auth pages. Gap vs Clerk (Branding tab with logo + color picker) and Auth0 (Templates tab with Handlebars editor).

**See:** full design spec at `docs/superpowers/specs/2026-04-20-branding-hosted-components-design.md` (Phase A section).

**Scope (pending brainstorm):**
- Backend: new `email_templates` table (template_id, subject, html, text, updated_at) seeded from current hardcoded templates
- Backend: new handlers for CRUD + preview endpoint (GET `/admin/email-preview/{template}` exists per DASHBOARD_GAPS — extend)
- Backend: `branding` config (colors as CSS vars, logo URL, font, footer text) — global + per-application override
- Frontend: new `admin/src/components/branding.tsx` — replaces current `empty_shell.tsx` Branding stub. Logo upload, color picker, font picker, live preview of auth pages + sample email
- Frontend: new `admin/src/components/email_templates.tsx` (OR tab inside branding) — list templates, edit subject/html/text, preview with sample data, save
- Frontend: drop `ph:9` gate on Branding once shipped

**Templates to ship editable:**
- welcome / magic-link / password-reset / email-verify / invite / mfa-code / device-approve / agent-consent (verify list vs `internal/email/` code)

**Dependencies:** W20 (shared branding config)
**Estimate:** Large — break down after brainstorm

### W22 — CLI `--json` output (P1)

**Problem:** Zero CLI commands support `--json`. Blocks CI/CD, Terraform, Pulumi, GitHub Actions integration. DX audit C8 finding F8.1.

**Files:** `cmd/shark/cmd/*.go` (every command with output). Add `--json` flag, branch on it in `RunE`.

**Acceptance (per command):**
- `app list --json` → `[{id,name,client_id,client_secret:null,redirect_uris,...}]`
- `app show --json` → `{id,name,...}`
- `app create --json` → `{client_id, client_secret, ...}` (secret-once semantics preserved)
- `app rotate-secret --json` → `{client_id, client_secret}`
- `app update/delete --json` → `{id, status:"updated"|"deleted"}`
- `keys generate-jwt --json` → `{kid, algorithm, status:"active"|"rotated"}`
- `health --json` → existing structure (already JSON)
- `version --json` → `{version, commit, date}`
- No state-mutating command should print secrets twice (stderr + JSON) — pick one

**Estimate:** 3-5d (16 commands/subcommands + tests)
**Commit sequence:** one commit per command family: `feat(cli): W22a — app --json`, `feat(cli): W22b — keys --json`, etc.

### W23 — Shark doctor + global flags (P2)

**Problem:** No `shark doctor` diagnostic command. Global `--verbose`/`--config` inconsistent across commands.

**Files:** `cmd/shark/cmd/root.go` (persistent flags), new `cmd/shark/cmd/doctor.go`
**Acceptance:**
1. `--verbose`, `--config` set as persistent flags on root — inherited by all subcommands
2. New `shark doctor` runs checks: config file exists + parses, DB accessible, secret length >= 32, email provider configured, port available, admin has at least one key, signing keys exist, migration up-to-date
3. Reports pass/fail + remediation hint per check
4. `shark doctor --json` machine-readable output
**Estimate:** 2d
**Commit:** `feat(cli): W23 — shark doctor + global --verbose/--config flags`

### W24 — OAuth `form_post` response mode + secret rotation (P2)

**Problem:** SPAs and native apps with embedded browsers must handle query params. RFC 9207 `form_post` standard. Also: DCR secret rotation endpoint missing — integrator leaking secret must re-register orphaning old.

**Files:**
- `internal/oauth/handlers.go` — add `renderAuthorizeFormPost` HTML template
- `.well-known/oauth-authorization-server` — advertise `"form_post"` in `response_modes_supported`
- `internal/oauth/dcr.go` — `POST /oauth/register/{client_id}/secret` handler
**Estimate:** 4h + 3h = ~1d
**Commit:** two commits: `feat(oauth): W24a — form_post response mode` + `feat(oauth): W24b — DCR secret rotation`

### W25 — OIDC discovery endpoint (P3)

**Problem:** Metadata advertises `openid` scope but no `.well-known/openid-configuration` endpoint. No `id_token` issued. Some OIDC clients break.

**Files:** new `internal/oauth/oidc_discovery.go`, advertise in metadata
**Decision required:** full OIDC provider or metadata stub only? Full is Phase 8.
**Estimate:** stub 2h, full 5d
**Commit:** `feat(oauth): W25 — OIDC discovery stub` OR defer

### W26 — Proxy injection audit + `X-Shark-Injection-ID` (P2)

**Problem:** Shark logs only deny decisions. Self-hosters debugging "why does upstream see wrong user_id" are blind. No trace header.

**Files:** `internal/proxy/proxy.go` — add opt-in verbose logging + response-header injection of trace ID
**Acceptance:**
1. `shark serve --log-proxy-injections` emits `proxy injected` event per request with path, method, user_id, injection_id
2. `X-Shark-Injection-ID` response header on every proxied response (even without verbose) so self-hoster can grep logs from upstream side
3. Smoke: proxy auth'd request → response has X-Shark-Injection-ID
**Estimate:** 2h
**Commit:** `feat(proxy): W26 — injection audit trail + X-Shark-Injection-ID trace header`

### W28-series — Backend-built-frontend-skipped (gap scan 2026-04-20 night)

Separate parallel Explore audit found 40+ backend features shipped but with no/partial dashboard UI. Top 10 launch-relevant:

| Rank | Task | Feature | Effort |
|------|------|---------|--------|
| W28 | oauth-token-introspection-ui | `POST /oauth/introspect` + admin-key works; no dashboard "paste token, inspect" UI | M |
| W29 | device-code-admin-queue | `/admin/oauth/device-codes/{user_code}/{approve\|deny}` live; no admin triage UI | M |
| W30 | audit-export-filter-parity | Backend supports `org_id/session_id/resource_type/resource_id` filters on export (migration 00016); compliance.tsx only exposes `action/from/to` | M |
| W31 | sso-connection-test | OIDC `BeginOIDCAuth` exists; no "test this connection" button that probes issuer reachability | M |
| W32 | lockout-manual-unlock-ui | LockoutManager tracks locked accounts; no dashboard to view + manually unlock (support needs this) | S |
| W33 | email-template-preview-history | `GET /admin/email-preview/{template}` live, authentication.tsx calls it; no edit UI + no per-template send-history | L (merges into W21) |
| W34 | rbac-effective-permissions-tree | `/users/{id}/permissions` computes inheritance; users.tsx shows flat list, no role-inheritance tree view | M |
| W35 | webhook-delivery-filters | `/webhooks/{id}/deliveries?limit=50` exists; no export/filter-by-status/filter-by-timestamp in UI | M |
| W36 | flow-execution-history | `GET /admin/flows/{id}/runs` exists; no UI to view past flow executions + debug failures | M |
| W37 | session-mfa-challenge-state | Backend tracks `mfa_passed` flag per session; no admin UI showing users stuck in MFA challenge state | M |

Lower-priority gaps (post-launch polish): SSO login analytics per connection, vault refresh error alerts, rate-limit quota-per-user UI, domain-based org auto-assignment, Auth0 migration importer (backend is `notImplemented()` stub — see W13), CSV org member bulk import, permission reverse lookup drill-in (merges into W12), proxy SSE status streaming (`/admin/proxy/status/stream` exists, UI polls), signing key JWK export + retire history, bootstrap token generation UI (admin generates ad-hoc token for help-desk password reset).

Full scan output preserved in conversation log; transcribe to `DEVEX_REVIEW_2026-04-20.md` Appendix on next pass.

### W27 — Proxy YAML reload button (P2)

**Problem:** DB rules + YAML rules coexist. YAML changes require restart. Dashboard has placeholder banner "Reload config to apply changes" but no button. Wave D P5.1 TODO.

**Files:** `admin/src/components/proxy_config.tsx:561`, new `internal/api/admin_proxy_handlers.go` reload endpoint
**Acceptance:**
1. Dashboard "Reload YAML" button calls `POST /admin/proxy/reload-yaml`
2. Backend re-reads `sharkauth.yaml` from disk, recompiles engine via `SetRules`
3. Shows bad YAML as red banner + keeps old rules
4. Banner copy: "DB rules take precedence over YAML rules"
**Estimate:** 2h
**Commit:** `feat(proxy): W27 — YAML hot-reload button`

### W15 — Multi-listener proxy (transparent protection on real app port) (P1)

**Problem:** Today shark proxy only works one of two ways:
1. **Embedded** (`serve --proxy-upstream`) — single catch-all on main :8080. User must hit :8080 instead of their app's port. Port change visible, cookie domain shared with admin.
2. **Standalone** (`shark proxy --port 3000 --upstream http://localhost:3001`) — dedicated port works BUT JWT verify is not wired. All requests treated as anonymous. `require:authenticated` denies everything. Per `shark proxy --help`: "MVP scope... follow-up."

**User-visible gap:** To get Caddy-like transparent protection (user hits `:3000`, shark intercepts, forwards to real app on `:3001`), you need an external reverse proxy. Shark has 90% of the code but neither mode delivers the experience.

**Goal:** Shark = the reverse proxy. One binary. N protected apps, each on its own port, auth enforced via existing session + JWT + JWKS path. Admin/auth stay on `:8080`.

**Config shape (new):**
```yaml
proxy:
  listeners:
    - bind: ":3000"
      upstream: "http://localhost:3001"
      session_cookie_domain: localhost
      rules:
        - path: /login
          allow: anonymous
        - path: /_next/*
          allow: anonymous
        - path: /*
          require: authenticated
    - bind: ":3100"
      upstream: "http://localhost:3101"
      rules: [...]
```
Each listener binds own port, owns upstream + rules + trusted-headers config. Legacy top-level `enabled/upstream/rules` fields remain supported (compile to a single implicit listener on the main port) for backwards compat.

**Files:**
- `internal/config/config.go` — new `ProxyListenerConfig` struct; `ProxyConfig.Listeners []ProxyListenerConfig`; Resolve() compiles legacy fields into `listeners[0]` when set
- `internal/proxy/engine.go` — multi-engine support or per-listener engine instances
- `internal/proxy/listener.go` (new) — one `http.Server` per listener, shares the auth middleware + store + RBAC instances
- `internal/server/server.go` — spawn listeners at Build() time; graceful shutdown across all listeners
- `cmd/shark/cmd/serve.go` — drop `--proxy-upstream` single-port flag in favor of yaml-only, OR keep as convenience (compiles to first listener)
- `cmd/shark/cmd/proxy.go` — standalone mode: finish JWT verify by fetching `{base_url}/.well-known/jwks.json`, cache JWKS, verify `Authorization: Bearer` on every request
- `admin/src/components/proxy_config.tsx` — "Protected apps" section, one card per listener, per-listener status (port bound, upstream reachable, req/s, circuit state)
- `admin/src/components/proxy_wizard.tsx` — wizard now asks "which port should shark listen on for this app?" (defaults to upstream-port - 1000 so user doesn't need to think, e.g. upstream :3000 → shark :2000, but editable)
- Smoke: `smoke_test.sh` new section — spawn toy HTTP server on :9001, configure shark listener on :9000 → :9001, hit :9000 unauthed = 401, authed = 200 + headers injected

**Backend:** yes (big: listener refactor + JWT verify in standalone mode)
**Acceptance:**
1. Yaml with 2+ listeners boots without error, binds both ports, admin port still works
2. Hit `localhost:3000` unauthed → 401 (or redirect to login URL config)
3. Log in via `localhost:8080/admin/`, session cookie on `.localhost` → hitting `localhost:3000` works, headers `X-Shark-User-ID` + `X-Shark-Email` injected to upstream
4. Legacy single-port config (no `listeners:` block) still works unchanged — `enabled/upstream/rules` at top level compiles to implicit single listener on main port
5. Standalone mode (`shark proxy --auth http://localhost:8080 --upstream http://localhost:3001 --port 3000`) does JWT verify via JWKS — `Authorization: Bearer <jwt>` required, 401 without, 200 with valid token
6. Admin dashboard shows per-listener health + rules + simulator per listener
7. Smoke assertions for auth + headers + rules + circuit breaker per listener

**Estimate:** 4-6h CC.
**Dependencies:** none (refactor-only).
**Commit sequence:**
- `feat(proxy): W15a — ProxyListenerConfig struct + legacy-compat Resolve`
- `feat(proxy): W15b — multi-listener lifecycle in server.Build`
- `feat(proxy): W15c — standalone mode JWT verify via JWKS cache`
- `feat(proxy+dashboard): W15d — per-listener UI in Proxy page`
- `test: W15e — smoke per-listener + standalone JWT`

**Why this matters (user-outcome framing):**
Without W15, every self-hoster must put Caddy/nginx/Traefik in front of shark to get transparent port protection — three processes instead of one, separate config, separate restart discipline. Shark's pitch is "one binary." W15 delivers on that pitch for the reverse-proxy use case. This is the reason Clerk/WorkOS don't do reverse proxy at all (too much infra). Shark shipping transparent proxy in a single binary = real distribution advantage.

---

## Dependency Graph

```
W01 org-create         ── no deps (P0)
W02 impersonation      ── no deps (P0, backend+frontend)
W03 drop unbuilt nav   ── no deps (P1, 15 min)
W04 drop tokens nav    ── no deps (P1, 15 min)
W05 apps tokens tab    ── no deps (P1)
W06 consents admin     ── no deps (P1)
W07 flow 3 steps       ── no deps (P2)
W08 users bulk         ── no deps (P2)
W09 App.tsx catches    ── no deps (P2)
W10 audit/flow catches ── no deps (P2)
W11 settings note      ── no deps (P3)
W12 rbac drill-in      ── no deps (P3)
W13 drop migrate stub  ── no deps (P3)
W15 multi-listener pxy ── no deps (P1, backend-heavy)
```

Zero cross-task file collisions except:
- `users.tsx` — W08 (may also need W01-pattern CreateSlideover helper extracted to shared)
- `layout.tsx` + `empty_shell.tsx` + `App.tsx` — W03 + W04 serialize
- `proxy_config.tsx` + `proxy_wizard.tsx` — W15 (refactor for per-listener UI)

---

## Wave Dispatch Plan

### Wave 1 (parallel, 4 worktrees — writers on same repo ⇒ **MUST USE `isolation: "worktree"`**)

- **Agent-α worktree-1:** W01 orgs create slide-over (frontend only)
- **Agent-β worktree-2:** W02a impersonation backend (Go)
- **Agent-γ worktree-3:** W03 + W04 nav cleanup (single commit safely; layout/App/empty_shell)
- **Agent-δ worktree-4:** W09 + W10 silent-catch surfacing

### Wave 2 (after Wave 1 merges)

- **Agent-ε:** W02b impersonation frontend (needs W02a endpoints landed)
- **Agent-ζ:** W05 apps tokens tab
- **Agent-η:** W06 consents admin listing
- **Agent-θ:** W07 flow 3 steps (backend + frontend)

### Wave 3

- W08 users bulk (sequential — touches users.tsx where W01 pattern may be extracted)
- W11 settings note (30 min solo)
- W12 rbac drill-in
- W13 drop migrate stub

### Wave 4 — ship review

- Full smoke: `bash smoke_test.sh`
- Go tests: `go test ./... -count=1`
- Typecheck: `cd admin && npx tsc -b --noEmit`
- Rebuild binary, dogfood `bin/shark.exe` on fresh DB
- Update `FRONTEND_WIRING_STATUS.json` to all done
- Tag / merge

---

## Status Tracker

Initialize `FRONTEND_WIRING_STATUS.json` on first execution:

```json
{
  "updated_at": "2026-04-20T23:00:00Z",
  "branch": "claude/admin-vendor-assets-fix",
  "current_task": null,
  "tasks": {
    "W01": {"status": "pending", "priority": "P0", "commit": null},
    "W02a": {"status": "pending", "priority": "P0", "commit": null},
    "W02b": {"status": "pending", "priority": "P0", "commit": null, "blocked_by": ["W02a"]},
    "W03": {"status": "pending", "priority": "P1", "commit": null},
    "W04": {"status": "pending", "priority": "P1", "commit": null},
    "W05": {"status": "pending", "priority": "P1", "commit": null},
    "W06": {"status": "pending", "priority": "P1", "commit": null},
    "W07": {"status": "pending", "priority": "P2", "commit": null},
    "W08": {"status": "pending", "priority": "P2", "commit": null},
    "W09": {"status": "pending", "priority": "P2", "commit": null},
    "W10": {"status": "pending", "priority": "P2", "commit": null},
    "W11": {"status": "pending", "priority": "P3", "commit": null},
    "W12": {"status": "pending", "priority": "P3", "commit": null},
    "W13": {"status": "pending", "priority": "P3", "commit": null},
    "W15": {"status": "pending", "priority": "P1", "commit": null}
  }
}
```

---

## Budget

| Priority | Tasks | Est CC time |
|----------|-------|-------------|
| **P0** | W01 (orgs create), W02 (impersonation), W16 (flow conditional bug), W17 (AGENT_AUTH rewrite) | ~11h |
| **P1** | W03, W04, W05, W06, W15 (multi-listener), W18 (structured errors), W19 (MFA pending), W20 (hosted auth, large), W21 (mail/branding, large), W22 (CLI --json) | ~week+ |
| **P2** | W07, W08, W09, W10, W23 (doctor+global flags), W24 (form_post+secret rotation), W26 (proxy audit), W27 (YAML reload) | ~12h |
| **P3** | W11, W12, W13, W25 (OIDC discovery decision) | ~3h |
| **Total** | **25 tasks** | **~70-90h CC** |

Parallelizable via worktrees drops wall-clock significantly. W17/W16 are single-commit quick wins that unblock launch trust gate.

## Launch-critical path (April 27 target)

| Phase | Tasks | Est | Reason |
|-------|-------|-----|--------|
| **Day 1 (pre-code)** | W17 AGENT_AUTH rewrite | 2h | Trust gate; read first by every HN visitor |
| **Day 1-2 core bugs** | W16 flow conditional save, W01 orgs create | 3h | User-reported blockers |
| **Day 2-3 moat** | W15 multi-listener, agent SDK stub (from F5.2 in DEVEX_REVIEW) | 1-2d | "One binary" pitch + agent-auth moat usability |
| **Day 3-4 scripting** | W22 CLI --json | 3d | CI/CD unblock |
| **Day 4-5 polish** | W18 structured errors (at minimum OAuth side), W02 impersonation | 2d | Integrator trust |

Everything else → Phase 7 or post-launch. Hosted-auth (W20) + mail-builder/branding (W21) — DO NOT attempt pre-launch; they're net-new features needing brainstorm + design work.

---

## What this plan does NOT cover

- `@ts-nocheck` removal (W14 deferred)
- Wave-4 smoke partials from old HANDOFF (F4 Token Exchange, F5 DPoP, F6 Vault, F7 Cache-Control, G1 TOTP, G2 WebAuthn) — track separately
- Launch-side work from `daabee1` design doc (user deprioritized)
- Settings full-writability (Phase 8/9)
- Migration tooling (Phase 8/9)

---

## Next step

Pick one:
1. Dispatch Wave 1 (4 parallel writer agents in worktrees) — needs user OK on worktree count/budget
2. Solo kick W01 + W03 + W04 inline (smallest P0/P1, ~2h, no worktree overhead)
3. User dogfoods current binary first to confirm E1-E9 + suggest prioritization tweaks
