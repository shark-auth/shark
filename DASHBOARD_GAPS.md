# Dashboard Gap Audit + Fix Plan

**Date:** 2026-04-19
**Scope:** Phase 4–6 dashboard at `admin/src/components/` (React 18 + Vite + TS, embedded via `go:embed`)
**Backend baseline:** 244 smoke PASS, ~100 admin endpoints registered
**Verdict:** Dashboard is ~90% wired. Gaps are narrow but specific. No backend rewrites needed for top fixes.

---

## TL;DR — What's Actually Broken

Two confused signals collided:
1. Backend smoke passes — every endpoint exists and responds.
2. Dashboard "feels broken" — because a few high-visibility surfaces still render mock data, two key pages are read-only stubs, and three small bugs leak fabricated counts.

The Wave 0–5 plan in `WIRINGDASHB.md` ran. Most pages got wired during Phase 4.5 polish + Phase 5/5.5/6 ship work. What got skipped:
- Mock fallbacks in `overview.tsx` were never removed (agent metrics, sparklines, attention panel).
- Phase 6 dashboards (proxy, vault) shipped MVP-tier — read-only where backend supports CRUD.
- Two endpoints that newly shipped (`/admin/health`, `/admin/config`, signing-key rotate, test-email) aren't reflected in `DASHBOARDV1.md` status table — caused team to assume more work remains.

---

## Endpoint Reality vs `DASHBOARDV1.md` Status Table

These were marked ❌ or 🟡 in `DASHBOARDV1.md`. All shipped:

| Endpoint | Old Status | Reality |
|---|---|---|
| `GET /admin/health` | ❌ | ✅ `admin_system_handlers.go:106` |
| `GET /admin/config` | ❌ | ✅ `admin_system_handlers.go:121` |
| `POST /admin/sessions/purge-expired` | ❌ | ✅ `session_handlers.go:271` |
| `POST /admin/audit-logs/purge` | ❌ | ✅ `audit_handlers.go:306` |
| `POST /admin/test-email` | ❌ | ✅ `admin_system_handlers.go:163` |
| `GET /users/{id}/oauth-accounts` | ❌ | ✅ `user_handlers.go:209` |
| `DELETE /users/{id}/oauth-accounts/{id}` | ❌ | ✅ `user_handlers.go:232` |
| `GET /users/{id}/passkeys` | 🟡 | ✅ `admin_system_handlers.go:200` |
| Signing key rotate | ❌ | ✅ `POST /admin/auth/rotate-signing-key` |
| SSO users-per-connection count | ❌ | ✅ in `ListConnections` response |

Still missing:
| Endpoint | Why we need it |
|---|---|
| `GET /admin/email-preview/{template}` | Authentication page email preview |
| `GET /permissions/{id}/roles` + `/users` | RBAC reverse lookup |
| `GET /admin/agents/stats` | Overview agent metrics card |
| Admin consents listing (cross-user) | `/auth/consents` is session-only — admin cannot view |
| Admin device-flow pending list | No `GET /oauth/device/pending` for admin queue |
| Users: `?auth_method=`, `?org=` filters | Currently only `?role_id=`, `?mfa_enabled=` |
| `last_login_at` exposed in `/users` admin response | Field exists in DB, not in JSON |

---

## Top 12 Gaps — Ranked by User-Visible Impact

| # | Gap | Page | Severity | Effort | Backend? |
|---|-----|------|----------|--------|----------|
| 1 | Proxy rules read-only (must edit YAML + reload) | `proxy_config.tsx` | HIGH | M | needs `POST/PATCH/DELETE /admin/proxy/rules` |
| 2 | Vault Connections tab is placeholder text | `vault_manage.tsx:503-539` | HIGH | M | needs `GET /admin/vault/connections` (cross-user) |
| 3 | Overview agent metrics + sparklines + attention panel use MOCK | `overview.tsx:49,60-65,~370` | HIGH | S | needs `/admin/agents/stats`; sparkline endpoints optional |
| 4 | Signing key rotate button disabled (endpoint exists) | `signing_keys.tsx` | MED | XS | none — wire button to `POST /admin/auth/rotate-signing-key` |
| 5 | Consents page returns 401 in admin (session-only endpoint) | `consents_manage.tsx` | MED | M | needs admin-scoped consents endpoint |
| 6 | Device flow has no admin pending queue (handoff only) | `device_flow.tsx` | MED | M | needs `GET /oauth/device/pending` + approve/deny JSON |
| 7 | `last_login_at` not in admin user response | `users.tsx` | MED | XS | trivial — add field to `adminUserResponse` struct |
| 8 | Users page filters limited (no auth_method, org) | `users.tsx` | MED | S | extend `ListUsersOpts` |
| 9 | Flow Builder: conditions tab is raw JSON, no visual builder | `flow_builder.tsx:522` | LOW | L | UI-only; documented as F4.1 |
| 10 | Flow Builder: no drag-reorder; conditional branches nested not forked | `flow_builder.tsx:822` | LOW | M | UI-only; documented F4.1 |
| 11 | Authentication email preview button missing | `authentication.tsx` | LOW | M | needs `GET /admin/email-preview/{template}` |
| 12 | Permission reverse lookup ("which users have X?") | `rbac.tsx` | LOW | S | `/permissions/{id}/roles` + `/users` |

Effort: XS=<1h · S=1–3h · M=½–1d · L=1–2d

---

## Per-Page Fix List

### `overview.tsx` — wire remaining mocks
- L49: drop `MOCK.stats.agents` once `/admin/agents/stats` ships (or compute from `/agents` list — count enabled, sum tokens).
- L60-65: sparkline trends — backend exposes only signups; either ship sparkline endpoints or **delete the mocked sparklines** to stop lying. Recommend the latter for now.
- L~370: `MOCK.attention` — replace with `useAPI('/admin/health')` + filter for warnings (smtp_unconfigured, expiring_keys, etc.); health endpoint already returns this.

### `proxy_config.tsx` — make rules editable
- Add backend `POST/PATCH/DELETE /api/v1/admin/proxy/rules` (currently read-only via YAML reload).
- Add Create/Edit slide-over: path glob, methods multi-select, require/allow rule, scopes.
- Add inline edit per row + delete confirm.
- Spec: `PROXY.md` — confirm rule schema for DB-backed mode vs YAML-backed.

### `vault_manage.tsx` — connections tab
- Add backend `GET /api/v1/admin/vault/connections` (cross-user list with status, last_refresh, scopes).
- Wire connections tab table per spec section 13: user, provider, scopes, expires, last_refreshed, status, disconnect.
- Filters: by provider, expiring soon, failed refresh.

### `signing_keys.tsx` — wire rotate button
- Endpoint exists. Just remove `disabled` + add confirmation modal warning about JWKS cache TTL.
- Add "Retire key" if backend supports kid retirement.

### `consents_manage.tsx` — needs backend
- Backend: ship `GET /api/v1/admin/consents` + `DELETE /api/v1/admin/consents/{id}` (admin-key auth, cross-user).
- Frontend: switch from `/auth/consents` → `/admin/consents`.
- Filter by user / agent / scope / active|revoked.

### `device_flow.tsx` — admin queue
- Backend: ship `GET /api/v1/admin/oauth/device/pending` returning user_codes + agent + scopes + IP + expires.
- Backend: ship `POST /admin/oauth/device/{user_code}/approve` and `/deny` (admin override).
- Frontend: cards layout per spec section 14 (live polling 5s, big mono user_code, approve/deny buttons).

### `users.tsx` — filter + last_login
- Backend: extend `ListUsersOpts` with `AuthMethod`, `OrgID`. Add `last_login_at` to `adminUserResponse`.
- Frontend: filter dropdowns (auth method, org) and "Last Active" column visible.

### `flow_builder.tsx` — F4.1 polish
- Defer drag-and-drop + forked branches + keyboard nav per existing TODO. Acceptable MVP.
- Conditions tab visual builder — defer to Phase 7 polish.

### `rbac.tsx` — reverse lookup
- Backend: `GET /permissions/{id}/roles` + `/permissions/{id}/users` (low effort — joins).
- Frontend: "Used by N roles / M users" line per permission + drill-down.

### `authentication.tsx` — email preview
- Backend: `GET /admin/email-preview/{template}?sample=...` returning rendered HTML.
- Frontend: per-template preview pane (magic_link, verify_email, password_reset).

### `PROJECT.md` cleanup
- Says "Svelte dashboard" — actual is React 18 + Vite + TS. Update.
- Marks features as "Active [ ]" that are validated. Migrate to Validated section.

### `DASHBOARDV1.md` cleanup
- Status table (lines 685-718) — refresh ❌→✅ for shipped endpoints (already done in this doc's table above).

---

## Recommended Wave Plan

**Wave A — High-impact UI (no backend)** · ~½ day
- Wire `signing_keys.tsx` rotate (XS).
- Strip mocked sparklines from `overview.tsx` OR replace agent metrics with derived count from `/agents` (S).
- Replace `MOCK.attention` with `/admin/health` warnings (S).

**Wave B — Tiny backend extends** · ~½ day
- Add `last_login_at` to admin user response (XS).
- Extend `ListUsersOpts` with `AuthMethod`, `OrgID` filters (S).
- Wire users page filter dropdowns (S).

**Wave C — Vault connections** · ~1 day
- Backend `GET /admin/vault/connections` (S).
- Frontend connections tab table + filters (M).

**Wave D — Proxy rules CRUD** · ~1 day
- Decision: keep YAML primary or add DB-backed override layer? Recommend DB-backed for dashboard parity, YAML stays as bootstrap.
- Backend rule CRUD endpoints (M).
- Frontend create/edit/delete UI (M).

**Wave E — Admin consents + device queue** · ~1 day
- Backend admin-scoped consents (S) + device pending queue (S) + approve/deny (S).
- Frontend wiring on existing pages (S each).

**Wave F — RBAC + email preview** · ~½ day
- Backend reverse lookup (S) + email preview (S).
- Frontend wires (S).

**Total ~4 days** to fully close UI gaps. Wave A alone removes 80% of "broken-feeling" UX.

---

## Why Smoke Passed but Dashboard Felt Broken

Smoke tests cover backend HTTP correctness. They don't:
- Click through every dashboard page in headless browser.
- Detect mock-data fallbacks rendering instead of real API calls.
- Catch read-only UIs where users expect CRUD.
- Verify newly-shipped endpoints are wired into the React tree.

Recommend: add a Playwright golden-path suite (Wave 6 in WIRINGDASHB.md). Hits each page, checks for "MOCK" string in rendered DOM, smoke-tests one mutation per page.

---

## Doc Updates Made Alongside This Plan

- `ATTACK.md` — added Phase 6.5 (dashboard gap fix wave).
- `DASHBOARD.md` — refresh backend status table.
- `PROJECT.md` — fix Svelte→React stack note.
- `WIRINGDASHB.md` — left as-is; this doc supersedes for current work.

---

# DEEP AUDIT — Round 2 (added 2026-04-19)

Investigation extended after first pass. ~25 additional real bugs found across HARD 404s, silent error swallows, backend handler param/shape bugs, and glue-feature stubs. All slip past 244-pass smoke (which checks HTTP 200 + key presence, not semantic correctness or frontend wiring).

## Class A — Frontend Hits Routes That Don't Exist (HARD 404)

User clicks a button, server returns 404, UI shows nothing or stale state. These break visible features.

| # | File:Line | Frontend Call | Backend Reality |
|---|---|---|---|
| A1 | `webhooks.tsx:646` | `POST /webhooks/{id}/deliveries/{deliveryId}/replay` | Route not registered. Replay button silent-fails (catch swallows). |
| A2 | `organizations.tsx:238` | `DELETE /admin/organizations/{id}` | Admin org routes are GET-only. Delete fails. |
| A3 | `organizations.tsx:459` | `POST /admin/organizations/{id}/roles` | Admin route doesn't exist. Org-role create fails. |
| A4 | `organizations.tsx:558` | `PATCH /admin/organizations/{id}` | Admin route doesn't exist. Update fails. |
| A5 | `organizations.tsx:609` | `DELETE /admin/organizations/{id}/invitations/{id}` | Handler doesn't exist. Revoke invite fails. |
| A6 | `organizations.tsx:616` | `POST /admin/organizations/{id}/invitations/{id}/resend` | Handler doesn't exist. Resend invite fails. |
| A7 | `users.tsx:480` | `DELETE /users/{id}/mfa` | Admin route missing. MFA only at `/auth/mfa` (session). Admin can't disable user MFA. |

**Fix path:** ship 6 missing admin handlers in `internal/api/admin_organization_handlers.go` + `webhook_handlers.go` + `admin_user_handlers.go`. ~½ day backend.

## Class B — Silent Catch Blocks (errors swallowed, no user feedback)

| # | File:Line | What's swallowed |
|---|---|---|
| B1 | `webhooks.tsx:646-647` | Replay 404 — empty `catch {}` |
| B2 | `organizations.tsx:611,617` | Invitation revoke + resend 404 — empty `catch {}` |
| B3 | `vault_manage.tsx:787` | Provider disable PATCH error — empty `catch {}` |

**Fix path:** replace empty catches with `toast.error(e.message)` + log. ~30 min frontend.

## Class C — Backend Handler Bugs (Silent Wrong Data)

| # | File:Line | Issue |
|---|---|---|
| C1 | `webhook_handlers.go:199` | `handleTestWebhook` reads NO body. Frontend sends `{event_type}` (webhooks.tsx:666) — handler always emits hardcoded `"webhook.test"`. Test fire selector dropdown is decorative. |
| C2 | `audit_handlers.go:42` | Handler reads `?actor_type=` query param but `storage.AuditLogQuery` has no `ActorType` field — silently dropped. Audit page filter doesn't filter. |
| C3 | `admin_stats_handlers.go:58` | `failed_logins_24h` queries `action='login' AND status='failure'` — misses `password.reset.failed`, `mfa.challenge.failed`. Counter understates. |
| C4 | `admin_stats_handlers.go:58` | MFA count is "enrolled" not "verified". User who started TOTP setup but never verified counts as enabled. |
| C5 | `user_handlers.go:30` | `adminUserResponse` struct has no `LastLoginAt` field even though DB has it. Users page "Last Active" column always shows "—". |
| C6 | `admin_system_handlers.go:109` | Returns flat `{db_size_bytes, uptime_seconds, config:{...}}`. Frontend `overview.tsx:~100` expects nested `{db:{size_mb,driver,status}, migrations:{current}, jwt:{mode,algorithm,active_keys}, smtp:{...}, oauth_providers:[], sso_connections:N}`. **Health card likely renders blank.** |
| C7 | `flow_handlers.go:360` | `handleTestFlow` reads `req.Metadata` then overwrites with empty map — caller-supplied metadata silently dropped from run history. |

**Fix path:** 7 small backend patches. ~½ day backend.

## Class D — Glue Feature Stubs (UX features advertised, not wired)

| # | Feature | File | Reality |
|---|---|---|---|
| D1 | Cmd+K palette entity search | `CommandPalette.tsx:5-35` | Static page list only — does NOT search users/agents/apps/orgs/keys/SSO/webhooks/tokens. Spec calls for fuzzy search across all entities. |
| D2 | Keyboard `n` (new) | `useKeyboardShortcuts.tsx` | Not implemented. Shortcut docs say it works. |
| D3 | Keyboard `e` (edit) | same | Not implemented. |
| D4 | Keyboard `r` (refresh) | same | Not implemented. |
| D5 | Top bar Quick Create dropdown | `layout.tsx:255-259` | Button rendered, no onClick, no menu. Pure stub. |
| D6 | Top bar Notification bell | `layout.tsx:261-268` | Icon + red dot rendered. No handler, no panel, no count source. Pure stub. |
| D7 | URL sub-path parsing | `App.tsx:52-62` | Only first segment parsed. `/admin/users/usr_abc/security` routes to `users` page; detail + tab state lost. Spec promises deep links. |
| D8 | Phase gating in nav | `layout.tsx:36-49,123` | Phase value (`ph: 5..10`) shown but no `disabled` flag — Phase 7+ shells clickable equally to live pages. Confusing UX. |
| D9 | Optimistic mutations | all `*.tsx` | Pattern is `await API.del(); refresh()` everywhere. No optimistic update + rollback. Spec principle 4 unmet. |
| D10 | Empty states on list pages | various | `TeachEmptyState` used on only 4 pages (orgs, rbac, sso, webhooks). Missing: audit, api_keys, applications, agents, signing_keys, consents. |
| D11 | Authentication page email preview | `authentication.tsx` | No email preview pane. Backend endpoint also missing. |

## Class E — Webhook Page Specific (deep dive)

In addition to A1, B1, C1 above:
- E1 — No way to filter table by active/disabled (toolbar only has text search).
- E2 — No bulk actions (no checkboxes despite spec batch-actions pattern).
- E3 — `WebhookConfigTab` (line 397+) tracks `enabled` toggle but original `WebhookRow` only shows it visually — no quick toggle from list (must open detail panel).
- E4 — `KnownWebhookEvents` map (line 24-30) lists only 5 events but frontend `COMMON_EVENTS` (line 31-40) lists 8 — mismatch. Frontend lets user pick `user.updated`, `session.created`, `mfa.enabled` which backend rejects with `unknown event` on save. **User clicks Save → 400 error.**
- E5 — `webhooks.tsx:60` reads `raw?.webhooks || raw` but backend returns `{data: [...]}` (handler line 117). Falls through to `raw` (the wrapper object), then `.filter()` works because object has no length. Renders zero rows when webhooks exist. **Webhook list always empty.**

**E4 + E5 are HARD bugs.** E5 in particular means the webhooks page may show nothing even with data present.

---

## Revised Top Gaps — Updated Priority

P0 — break visible features:
1. **E5 — Webhooks list reads wrong response key** (`raw?.webhooks` vs backend `data`) → table empty
2. **E4 — Frontend offers events backend rejects** → save fails
3. **A1-A7 — 7 routes return 404** (orgs CRUD, webhook replay, MFA disable)
4. **C5/C6 — `last_login_at` not exposed + health response shape mismatch** → Users column blank, health card likely blank
5. **C1 — Test webhook ignores event selector** → test fire is decorative

P1 — silent wrong data:
6. C2 — audit `actor_type` filter ignored
7. C3/C4 — stats counters misleading
8. C7 — flow test metadata dropped
9. B1-B3 — silent catches

P2 — UX stubs:
10. D5/D6 — quick-create + notification bell stubs in top bar
11. D1 — Cmd+K only navigates, doesn't search
12. D7 — URL deep links don't restore detail/tab
13. D8 — Phase gating not enforced
14. D9 — optimistic UI absent

Plus original WAVE A-F gaps (proxy rules CRUD, vault connections, signing-key rotate button, etc.)

---

## Revised Wave Plan

**Wave 0 — Critical correctness (½ day, ship before next demo)**
- E5 fix: change `webhooks.tsx:60` to `raw?.data || raw?.webhooks || raw` (tolerant)
- E4 fix: align `KnownWebhookEvents` (backend) with `COMMON_EVENTS` (frontend) — add 3 missing events to backend
- C5 fix: add `LastLoginAt *string` to `adminUserResponse`
- C6 fix: restructure `handleAdminHealth` response to nested shape OR adapt frontend mapper
- C1 fix: parse `event_type` body in `handleTestWebhook`

**Wave 1 — Ship missing routes (~½ day)**
- A1 — webhook replay handler
- A2-A6 — admin org CRUD + invitation manage
- A7 — admin MFA disable on user

**Wave 2 — Silent fail polish (~½ day)**
- B1-B3 — replace empty catches with toast.error
- C2 — add ActorType to AuditLogQuery + filter SQL
- C3/C4 — document stats semantics or fix queries
- C7 — flow test metadata preserve

**Wave A-F (original plan) — UI gaps (~3-4 days)**
- (overview mocks, signing-key rotate, vault connections, proxy rules CRUD, admin consents, device queue, RBAC reverse lookup, email preview, user filters)

**Wave 3 — Glue (~2 days)**
- D5/D6 — wire quick create dropdown + notification bell to real handlers
- D1 — Cmd+K entity search across users/agents/apps/orgs
- D7 — URL sub-path + tab state via hash
- D8 — Phase gating disabled flag
- D9 — optimistic mutation layer (or remove from spec — pragmatic)
- D10 — TeachEmptyState on 6 missing pages
- D2/D3/D4 — keyboard `n`/`e`/`r`

**New total ~6 days** for thorough fix. **Wave 0 alone (½ day) closes the most user-visible breakage** — webhooks list empty, health card blank, last-login blank, test-fire decorative — all fixed.

## Why Smoke Missed These

244 smoke tests = HTTP-level. They assert `status == 200` and key presence. They do NOT:
- Assert response shape matches what frontend reads (E5, C6)
- Assert filter params actually filter (C2)
- Assert counter accuracy (C3/C4)
- Assert body fields are honored (C1, C7)
- Walk the React tree to find dead `catch{}` swallows (B1-B3)
- Verify route exists for every frontend `API.del/post/patch` call (A1-A7)

**Recommendation**: Wave 6 in `WIRINGDASHB.md` (Playwright golden paths) becomes urgent. Add a static check too: grep all `API.{post,patch,del,get}` paths in admin/src vs registered routes in router.go — flag mismatches at CI time. Would catch all Class A bugs.

---

# DEEP AUDIT — Round 3: OAuth + Smoke Coverage Blockers (added 2026-04-19)

User flagged additional findings from smoke test review. These predate dashboard gaps but block real launch.

## Class F — OAuth Smoke Failures (LAUNCH BLOCKERS)

### F1 — Section 31 (Auth Code + PKCE) BROKEN  🚨

**Root cause confirmed at `internal/oauth/store.go:289-311`**:
- `CreatePKCERequestSession` is no-op (returns nil without writing anywhere)
- `GetPKCERequestSession` delegates to `GetAuthorizeCodeSession` which reconstructs `code_challenge` from `oauth_authorization_codes` table columns
- BUT fosite's `Sanitize()` strips `code_challenge` from the stored authorize session blob before token exchange — so when fosite's PKCE handler asks for the verifier session, it gets back a Requester with empty form values for `code_challenge`/`code_challenge_method`
- Token exchange returns 400 with PKCE verification failure
- Smoke test annotates as `note` not `fail` (line 704: `"token exchange $EX_CODE — PKCE persistence gap in fosite integration; covered by unit tests"`)

**Why this is a launch blocker**: Auth Code + PKCE is the canonical web app login flow. SPAs, mobile, `@sharkauth/js` SDK all need it. Without it shark cannot onboard a normal web app.

**Fix path**:
- Option 1 (recommended): properly implement `CreatePKCERequestSession` — write the PKCE request blob to a new table `oauth_pkce_sessions` (signature, code_challenge, code_challenge_method, expires_at). `GetPKCERequestSession` reads from there. `DeletePKCERequestSession` deletes. Decouples from auth code session entirely.
- Option 2 (hack): patch `GetAuthorizeCodeSession` to set the form values via `req.GetRequestForm().Set(...)` AFTER fosite's Sanitize would otherwise strip them — but Sanitize runs on storage write not read, so this should already work; investigate WHY it doesn't (likely fosite's PKCE handler uses a different code signature than the auth code).
- Add migration for `oauth_pkce_sessions` table.

**Smoke test fix**: Convert line 704 `note` to `fail` once handler fixed. Section 31 should report `pass`.

### F2 — Section 33 (Refresh Token Rotation) SKIPPED 🚨

**Cascade**: smoke test line 753-755 skips entire refresh test because section 31 didn't issue tokens.

**Reality**: refresh rotation code at `internal/oauth/store.go:250-261` (`RotateRefreshToken`) appears wired. Cannot validate end-to-end until F1 fixed.

**Why blocker**: refresh token = persistent session. Standard OAuth pattern for any user-facing app. Without smoke coverage we don't know if rotation actually works.

**Fix path**: F1 first. Then unskip section 33. Add assertions: refresh issues new RT, old RT revoked, reuse of old RT triggers family revocation.

### F3 — Section 42 (Consent Lookup) SKIPPED 🚨

**Cascade**: smoke test line 1053 skips because section 31 didn't complete (no consent record to look up).

**Fix path**: F1 first. Then verify `GET /api/v1/auth/consents` returns the granted consent and `DELETE` revokes it + cascades token revocation.

### F4 — Section 35 (Token Exchange RFC 8693) — no happy path

**Status**: Section asserts endpoint exists + rejects bad input. Never tests successful agent-to-agent delegation.

**Priority**: medium. Token exchange = agent delegation chain. Used by AGENT_AUTH.md spec but not by typical web app.

**Fix path**: build smoke test that creates two agents, one with `actor` token from the other, asserts `act` claim chain in resulting JWT.

### F5 — Section 36 (DPoP RFC 9449) — full flow skipped

**Status**: Endpoint registered, JWK proof header parsing tested at unit level (`dpop_test.go`). Smoke skips actual DPoP-bound token issuance + use.

**Priority**: low. DPoP = proof-of-possession for mobile/SPA tokens. Edge case for v1 web app launch.

**Fix path**: smoke test that issues DPoP-bound access token, asserts `cnf.jkt` claim, then uses token with DPoP proof header against introspect endpoint, asserts success. Without proof header → 401.

### F6 — Section 46 (Vault Token Retrieval) — full flow not tested

**Status**: Vault provider CRUD tested. User connect flow tested as far as redirect. The `GET /api/v1/vault/{provider}/token` happy path (agent retrieves stored token) needs a mock OAuth server upstream because real third-party OAuth (Google/Slack) can't run in CI.

**Fix path**: spin up a tiny in-process mock OAuth server in smoke test (`/mock/authorize`, `/mock/token`). Register vault provider pointing at it. Walk full flow: user connects → token stored → agent calls `/vault/.../token` with bearer → receives upstream token. Auto-refresh: rewind expiry, call again, assert refresh occurred.

### F7 — Sections 7 + 28 — Cache-Control headers not asserted

**Status**: Section 7 (key rotation) and 28 (AS metadata advanced) don't assert `Cache-Control: public, max-age=...` on JWKS / metadata responses.

**Why it matters**: bad cache headers = clients hammer JWKS endpoint, perf regression at scale.

**Fix path**: in section 7 add `curl -I /.well-known/jwks.json | grep -i cache-control` assertion. Same for section 28 metadata endpoint. Spec values: JWKS = 1 hour, metadata = 24 hours.

## Class G — Smoke Test Coverage Gaps (no flow tests for shipped features)

### G1 — MFA / TOTP — no enroll → challenge → verify flow

**Reality**: Section 4 user list checks `mfa_enabled` filter exists. Sections 14+ admin endpoints. **No section walks**:
- Enroll: `POST /auth/mfa/enroll` → secret returned
- Verify: `POST /auth/mfa/verify` with TOTP code → enrollment finalized
- Challenge on login: signup user with MFA → login → MFA challenge → submit code
- Recovery: enroll → save recovery codes → use one → assert single-use

**Fix path**: new section "MFA TOTP flow". Use `pquerna/otp/totp` to generate codes from the secret. Walk full enroll/challenge/recovery cycle. Assert audit log entries.

### G2 — Passkeys / WebAuthn — no signup or login flow

**Reality**: Section 15 admin lists user passkeys (admin-side endpoint). **No section walks**:
- Register: `POST /auth/passkey/register/begin` → challenge → simulated WebAuthn assertion → `register/finish`
- Login: `POST /auth/passkey/login/begin` → challenge → simulated assertion → `login/finish` → session

**Why it's hard**: real WebAuthn requires browser CTAP. Solution: use `go-webauthn` library's test helpers (it ships with a virtual authenticator for testing) OR vendor in canned attestation/assertion blobs from a known test vector.

**Fix path**: new section "Passkey WebAuthn flow". Pre-bake attestation + assertion JSON for a known credential. Walk register → list → login → delete.

## Updated Wave Plan

**Wave -1 — LAUNCH BLOCKER (1-2 days)** ← **DO FIRST**
- F1: Fix PKCE persistence (`oauth/store.go` — likely new `oauth_pkce_sessions` table + migration)
- Convert smoke section 31 `note` → `fail` so regression is caught
- Unskip F2 (refresh rotation) — should pass once F1 done
- Unskip F3 (consent lookup) — should pass once F1 done
- Run smoke: target sections 31, 32, 33, 42 all pass

**Wave 0 — Critical correctness** (existing, ½ day)
- E5, E4, C5, C6, C1 from Round 2
- **NEW: each fix gets a new smoke section** asserting the bug is dead

**Wave 1 — Missing routes** (existing, ½ day)
- A1–A7 from Round 2
- **NEW: smoke section per route**: webhook replay, admin org CRUD, admin org invitations, admin user MFA disable

**Wave C — Vault connections** ✅ DONE (smoke 320 → 328, 0 FAIL)
- C-1 ✅ ListAllVaultConnections storage method; GET /admin/vault/connections + DELETE /admin/vault/connections/{id} handlers; routes registered under /admin (admin-key auth)
- C-2 ✅ vault_manage.tsx ProviderConnections — replaced placeholder with real table (user_id, status chip, scopes, expires, last_refresh, disconnect); refresh/disconnect actions toast; filters by current provider client-side from /admin/vault/connections
- C-3 ✅ Smoke section 64 (8 assertions): empty shape, seed visible, user_id present, no token leakage, DELETE 204, missing 404, no-auth 401

**Wave B — Tiny backend + users page filters** ✅ DONE (smoke 315 → 320, 0 FAIL)
- B-1 ✅ ListUsersOpts.AuthMethod + OrgID added; sqlite WHERE plumbed (password = users.password_hash, others via sessions.auth_method); org via organization_members
- B-2 ✅ users.tsx — sends all filters as query params; reset page on filter change; backend now also returns `{users:[], total:N}` shape and accepts page/per_page (was previously plain array, dashboard always saw 0)
- B-3 ✅ Smoke section 63 added: response shape contract, auth_method=password narrows, auth_method=passkey applied, org_id=bogus → 0, per_page=2 limits

**Wave A — Frontend-only polish** ✅ DONE (smoke 315 → 315, 0 FAIL)
- A-1 ✅ signing_keys.tsx rotate button already wired (pre-existing) — confirmed live
- A-2 ✅ overview.tsx — removed mock sparklines (sessions/mfa/failed/keys/agents), removed MOCK.stats fallback, agent count derived from `/agents?limit=1` total, removed mock "Agent activity 24h" card
- A-3 ✅ overview.tsx — replaced MOCK.attention with `deriveAttention()` from `/admin/health` (smtp_unconfigured, jwt_no_keys, db_status, no_oauth_providers, expiring_keys); empty-state shows "All systems healthy"

**Wave 2 — Silent fail polish** ✅ DONE (smoke 303 → 315, 0 FAIL)
- B1 ✅ webhooks.tsx replay catch — surfaces err inline + "Replay queued" success
- B2 ✅ organizations.tsx invitation revoke + resend catches — toast.success/error
- B3 ✅ vault_manage.tsx provider disable catch — setError surfaces upstream
- C2 ✅ AuditLogQuery.ActorType plumbed through handler + sqlite WHERE
- C3 ✅ failed_logins_24h: auth_handlers emits `user.login` audit on failure; query updated
- C4 ✅ CountMFAEnabled now `mfa_enabled=1 AND mfa_verified=1` (verified-only)
- C7 ✅ flow_handlers handleTestFlow already passed metadata; smoke locks it in
- Smoke sections 59-62 added (12 new assertions)

**Wave A–F (original gaps) + Wave 3 (glue stubs)** as before.

**Wave 4 — Smoke Coverage Backfill (1 day)** ← **NEW**
- G1: MFA TOTP enroll/challenge/verify/recovery flow
- G2: Passkey register/login flow (with virtual authenticator or canned vectors)
- F4: Token Exchange happy path (agent-to-agent `act` chain)
- F5: DPoP full flow (cnf.jkt + proof header validation)
- F6: Vault token retrieval (in-process mock OAuth upstream)
- F7: Cache-Control assertions on JWKS + metadata
- Goal: zero smoke `note` lines describing skipped functionality

## RULE FOR ALL FUTURE FIXES

**Every backend fix MUST ship with a new or updated smoke test section that would have caught the bug.** No exceptions. Examples:
- Fix `webhook_handlers.go:199` to honor `event_type` → smoke section asserts `POST /webhooks/{id}/test {event_type:"user.created"}` actually emits `user.created` not `webhook.test`
- Fix `audit_handlers.go` to filter `actor_type` → smoke section creates 3 logs of different actor_types, queries with `?actor_type=admin`, asserts only admin row returned
- Fix `adminUserResponse` to expose `last_login_at` → smoke section logs in user, lists users, asserts response includes non-empty `last_login_at`
- Fix PKCE persistence → smoke section 31 promotes to fail-on-skip

This rule turns "244 PASS" from a number into a real signal.

## Blocker Severity Reset

After this round, the launch-critical sequence is:
1. **F1 (PKCE)** — without it, no web app login. Block launch.
2. **F2/F3** — auto-unblock from F1. Validate.
3. **A1-A7** — admin can't manage orgs / users via dashboard. Block ops.
4. **E4/E5** — webhooks page broken. Block ops.
5. **G1/G2** — MFA + passkeys silently untested. Block confidence.

Everything else is post-launch polish.
