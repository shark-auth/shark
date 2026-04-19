# Dashboard Gap Spec — What's Missing vs DASHBOARD.md / DASHBOARDV1.md

Cross-reference of DASHBOARD.md (v2) + DASHBOARDV1.md against actual implementation in `admin/src/`. Framework changed from Svelte to React — intentional, not a gap. This doc covers feature gaps only.

---

## 1. Global UX Patterns (zero implemented)

These are spec'd across both v1 and v2 and none exist in the codebase.

### 1.1 Command Palette (Cmd+K)
**Spec:** Fuzzy search across users, agents, apps, orgs, roles, keys, SSO conns, webhooks, tokens, pages, actions. Results grouped by type, keyboard navigable.
**Status:** Not implemented. No component, no keybinding, no search index.
**Effort:** Medium. Needs client-side fuzzy search over cached entity lists + page navigation map.

### 1.2 Keyboard Shortcuts
**Spec:** `g u` → Users, `g a` → Audit, `g k` → Keys, `g o` → Orgs, `g w` → Webhooks, `g d` → Dev Inbox, `g g` → Agents, `g v` → Vault, `g t` → Tokens, `/` → focus search, `n` → new (contextual), `e` → edit, `r` → refresh, `?` → cheatsheet, `Esc` → close panel.
**Status:** Zero keyboard handling. No `keydown` listeners anywhere.
**Effort:** Low-medium. Global keydown handler + route map.

### 1.3 Undo Toasts
**Spec:** Destructive actions → toast with 5s undo countdown. No confirm dialogs (except user delete / agent delete which require typing identifier).
**Status:** All destructive actions use `window.confirm()` — the exact anti-pattern the spec prohibits. Toast system exists (`toast.tsx`) but no undo mechanism.
**Effort:** Medium. Need deferred-delete pattern: UI removes immediately, timer fires actual DELETE, undo cancels timer.

### 1.4 Deep Linking
**Spec:** URL encodes filter state + open panels. `/admin/users?search=alice&verified=true`, `/admin/users/usr_abc/security`, `/admin/audit?action=login&status=failure&from=2026-04-01`.
**Status:** Minimal. `App.tsx` does basic `pushState` for page navigation. No filter/panel state in URL. Audit log builds URLSearchParams for API calls but doesn't sync to browser URL.
**Effort:** Medium. Each page needs to read/write filter state to URL search params.

### 1.5 Responsive Design
**Spec:** Desktop-first ≥1200px. ≥768px sidebar → icons. <768px sidebar → hamburger, tables → cards, slide-overs → full-screen.
**Status:** Not implemented. No `@media` queries. No breakpoint handling. Sidebar always full-width.
**Effort:** Medium-high. Need breakpoint system, sidebar collapse, table-to-card transforms.

### 1.6 CLI Parity Footers
**Spec:** Every detail panel shows equivalent `shark ...` command with copy button.
**Status:** Not implemented anywhere.
**Effort:** Low. Static string templates per entity type.

### 1.7 Empty States (teach pattern)
**Spec:** Every page: headline + why-it-matters + two paths (dashboard button + CLI snippet) + docs link.
**Status:** Partial. Some pages have basic "No X yet" messages. None follow the full pattern with CLI snippets or docs links. `empty_shell.tsx` handles Phase 5/6 stubs well.
**Effort:** Low. Template component + per-page copy.

### 1.8 Optimistic Mutations
**Spec:** Every update reflects instantly, rolls back on error.
**Status:** Not implemented. All mutations wait for server response.
**Effort:** Medium. Need mutation wrapper that updates local state pre-flight.

---

## 2. Page-Level Gaps

### 2.1 Overview
**Status:** Most complete page. Real API calls, live polling, sparklines, donut, health, attention panel.
**Gaps:**
- Agent stats card uses MOCK (no `/admin/stats` agent data yet — backend gap)
- Several trend sparklines fall back to MOCK (sessions, mfa, failed, keys, agents — backend doesn't provide these in `/admin/stats/trends`)

### 2.2 Users
**Status:** Table + search + detail slide-over + batch actions all work.
**Gaps:**
- **Filters:** No filter by auth_method, role, org, MFA, verified status. Only text search exists.
- **Detail tabs:** Need to verify which tabs are wired. Spec requires: Profile (inline edit), Security (MFA/passkeys/OAuth accounts/sessions), Roles, Organizations, Agent Consents, Activity.
- **Missing actions:** Send Verification Email, Admin-triggered Reset Password, Disable MFA.
- **Last Active column:** Shows in table but `last_login_at` not tracked by backend.
- **Delete confirmation:** Should require typing user email, currently uses `window.confirm()`.

### 2.3 Sessions
**Status:** Full. Live polling, stats, filters, revoke, detail slide-over.
**Gaps:**
- **JWT mode extras:** No "Revoke by JTI" input field (spec: wired to `/admin/auth/revoke-jti`). No "Rotate signing keys" shortcut link.
- **JTI column:** Spec says show JTI short hash + copy for JWT-mode sessions.

### 2.4 Organizations
**Status:** Two-pane with members + roles. Real API.
**Gaps:**
- **Invitations tab:** Pending invites table (email, role, expires, token), resend, revoke. Not implemented.
- **SSO Enforcement tab:** Toggle require-SSO, bind domain. Not implemented.
- **Org-scoped Audit tab:** Filtered audit log for `org_id=...`. Not implemented.
- **Bulk invite CSV:** Spec mentions bulk member invite via CSV upload. Not implemented.

### 2.5 Applications
**Status:** Full CRUD with detail slide-over, tabs, secret rotation.
**Gaps:**
- **CLI parity footer:** No `shark app create --callback ...` snippet.
- **Loopback exception display:** Spec mentions showing `http://127.0.0.1:*` toggle.

### 2.6 Authentication
**Status:** Read-only config display from `/admin/config`.
**Gaps:**
- **JWT Mode toggle:** No radio to switch cookie ↔ JWT with warning about session invalidation.
- **OAuth provider test button:** No "Test" button that opens OAuth flow in new tab.
- **Passkey config section:** RP name, RP ID, origin, attestation, UV setting, registered count — unclear if displayed.
- **Email verification template preview:** Needs `/admin/email-preview/{template}` (backend missing too).
- **No edit capabilities:** Entire page is read-only. Spec notes edit is god-tier but JWT mode toggle is Phase 3.

### 2.7 SSO
**Status:** Full CRUD with create/edit modal.
**Gaps:**
- **Domain routing tester:** "Enter email → shows which connection routes." Not implemented.
- **Per-connection user count:** Backend query needed + display.
- **Connection detail view:** Linked users table, activity log. Not implemented.
- **"Discover" button:** For OIDC issuer, fetch `.well-known` metadata. Not implemented.

### 2.8 Roles & Permissions
**Status:** Two-pane with CRUD, permission attach/detach, inline edit.
**Gaps:**
- **Permission Explorer tab:** Full permission list with usage count (how many roles use each). Not implemented.
- **"Check Permission" tool:** Pick user → action + resource → allow/deny with rationale. Not implemented.
- **Reverse lookup:** Which roles/users have a given permission (backend needed too).

### 2.9 API Keys
**Status:** Full CRUD, rotation, revoke, detail slide-over.
**Gaps:**
- **curl snippet generator:** After creating/viewing key, show ready-to-paste curl. Not implemented.
- **Usage section in detail:** Recent audit log where `actor_type=api_key`. Not implemented.
- **CLI parity footer.**

### 2.10 Audit Log
**Status:** Full. Filters, live mode, CSV export, detail expansion.
**Gaps:**
- **Agent-specific filters:** Filter by `actor_type=agent`, filter by `agent_id`. May exist partially.
- **Delegation chain filter:** "User acted through agents" → trace `act` chain. Not implemented (Phase 6 dependency).
- **Extended event types:** `oauth.*`, `agent.*`, `vault.*`, `webhook.*`, `application.*` grouping in filter dropdown.

### 2.11 Webhooks
**Status:** Full CRUD with deliveries tab.
**Gaps:**
- **Replay button:** Replay individual delivery. Not implemented.
- **Signature verify tool:** Paste payload + signature → valid/invalid + computed HMAC. Not implemented.
- **Test fire:** Fire arbitrary event type with sample payload. Not implemented.

### 2.12 Dev Inbox
**Status:** Full. Table + detail preview.
**Gaps:**
- **"Open magic link" shortcut:** Extract magic link URL from email body, one-click open. Not implemented.
- **Banner text:** Should show "Dev inbox captures all outgoing mail. Switch email provider before production."

### 2.13 Signing Keys
**Status:** Read-only JWKS display.
**Gaps:**
- **Rotate now:** Button exists but shows "Rotation endpoint not available yet". Backend endpoint needed.
- **Retire key:** Remove from JWKS after grace period. Not implemented.
- **Download JWKS:** curl-ready snippet. Not implemented.

### 2.14 Settings
**Status:** Health, email config, session purge, audit purge, danger zone. All present.
**Gaps:**
- **Send test email:** Button exists but endpoint not available. Shows "Endpoint not available yet".
- **shark.email tier badge + quota:** Tier display, sent/daily_limit. Not shown.
- **Session mode display:** Cookie vs JWT indicator. Not shown.

### 2.15 Agents
**Status:** Partial — full UI with table, detail slide-over, tabs (overview/tokens/consents/usage/delegation), but ALL data is MOCK.
**Gaps:**
- No real API calls. Entire page is mock data.
- Depends on Phase 6 backend (Agent Auth). Expected — page serves as a visual shell/demo.

### 2.16 Device Flow
**Status:** Partial — UI for pending approvals + recent activity, all MOCK data.
**Gaps:**
- No real API. Approve/deny buttons not wired.
- Depends on Phase 6 backend.

---

## 3. Missing Phase Shells

Spec says all future phase pages should have nav entry + empty-state page in Phase 4. Currently missing:

| Phase | Page | Nav Entry | Empty State |
|-------|------|-----------|-------------|
| 7 | Proxy | No | No |
| 8 | OIDC Provider | No | No |
| 9 | Impersonation | No | No |
| 9 | Compliance | No | No |
| 9 | Migrations | No | No |
| 9 | Branding | No | No |
| 10 | Flow Builder | No | No |

Phase 5 (API Explorer, Session Debugger, Event Schemas) and Phase 6 (Consents, Tokens, Vault) shells exist via `empty_shell.tsx`.

---

## 4. Backend Endpoints Still Needed

These are called out in both specs as required but don't exist. Dashboard has UI stubs waiting.

| Endpoint | Spec Priority | Dashboard blocker? |
|----------|--------------|-------------------|
| `GET /admin/health` | R | Partial — overview/settings call it, may 404 |
| `GET /admin/config` | R | Auth page calls it, may 404 |
| `POST /admin/test-email` | G | Settings "Send test" disabled |
| `POST /admin/sessions/purge-expired` | R | Settings button shows "not available" |
| `POST /admin/audit-logs/purge` | R | Settings purge may not work |
| `GET /admin/email-preview/{template}` | G | Auth page template preview blocked |
| `GET /users/{id}/oauth-accounts` | R | User detail Security tab incomplete |
| `DELETE /users/{id}/oauth-accounts/{id}` | R | Can't unlink OAuth |
| `GET /users/{id}/passkeys` | R (store exists) | User detail Security tab incomplete |
| Users: `last_login_at` | R | Last Active column inaccurate |
| Users: filter by role/auth_method/org/mfa | R | User table filters missing |
| SSO: users-per-connection count | R | SSO table missing column |
| Permissions: reverse lookup | G | Permission Explorer blocked |
| Signing keys: rotate endpoint | R | Signing Keys rotate disabled |

---

## 5. Execution Plan — Phased

### Wave A — Global UX Foundation ✅ DONE
All 7 items shipped. New files: useKeyboardShortcuts, CommandPalette, CLIFooter, useURLParams. Modified 14 files.

- [x] Keyboard shortcuts (g+key nav, /, ?, Esc)
- [x] Command palette (Cmd+K fuzzy search)
- [x] CLI parity footers (6 detail panels)
- [x] Undo toast pattern (replaced confirm() in apps/orgs/sso/rbac)
- [x] Deep linking (URL params on users/audit/sessions/keys)
- [x] Missing phase shells (7 pages: Proxy→Flow Builder)
- [ ] Empty state teach pattern (TeachEmptyState component — started but not wired)

### Wave B — Settings + Auth Config (UI-only, no backend needed)
All read-only display improvements from existing `/admin/config` and `/admin/health` data.

- [ ] **Settings: session mode indicator** — show cookie/JWT badge from config data
- [ ] **Settings: shark.email tier badge** — display tier + sent/limit from health data
- [ ] **Auth: JWT mode toggle UI** — radio cookie|jwt with warning banner (writes to config not wired yet but UI ready)
- [ ] **Auth: passkey config section** — display RP name, RP ID, origin, UV setting from config
- [ ] **Signing Keys: download JWKS snippet** — curl command with copy button
- [ ] **Dev Inbox: "Open magic link" shortcut** — extract URL from email body, one-click button

### Wave C — Users + Sessions Enhancements
Filter UI, detail tab completions, JWT session features.

- [ ] **Users: filter dropdowns** — auth_method, MFA, verified status (client-side filter on loaded data)
- [ ] **Users: delete requires typing email** — replace confirm() with modal + email input
- [ ] **Users: profile actions** — Send Verification, Admin Reset Password, Disable MFA buttons (wired to existing endpoints where available)
- [ ] **Sessions: JTI column** — show short JTI hash + copy for JWT sessions
- [ ] **Sessions: Revoke by JTI input** — text input wired to `/admin/auth/revoke-jti`
- [ ] **Sessions: Rotate signing keys shortcut** — link to Signing Keys page

### Wave D — Organizations + SSO Completions
Multi-tab completions for org detail and SSO detail.

- [ ] **Orgs: Invitations tab** — pending invites table (email, role, expires), resend/revoke actions
- [ ] **Orgs: SSO Enforcement tab** — require-SSO toggle, domain binding display
- [ ] **Orgs: Audit tab** — per-org audit log filtered to org_id
- [ ] **SSO: domain routing tester** — "enter email → shows which connection routes" input
- [ ] **SSO: connection detail view** — linked users table, per-connection activity log

### Wave E — RBAC + API Keys + Webhooks Feature Gaps
Interactive tools and missing CRUD features.

- [ ] **RBAC: Permission Explorer tab** — all permissions + usage count table
- [ ] **RBAC: Check Permission tool** — pick user → action + resource → allow/deny result
- [ ] **API Keys: curl snippet generator** — auto-generated curl based on key scopes
- [ ] **API Keys: usage audit section** — recent audit entries for actor_type=api_key
- [ ] **Webhooks: replay button** — per-delivery replay action
- [ ] **Webhooks: test fire** — fire arbitrary event type with sample payload
- [ ] **Webhooks: signature verify tool** — paste payload + signature → valid/invalid + HMAC

### Wave F — Polish + Responsive
Cross-cutting quality improvements.

- [ ] **Responsive design** — @media queries, sidebar collapse at 768px, table→card at mobile
- [ ] **Optimistic mutations** — mutation wrapper for instant UI updates + rollback on error
- [ ] **Empty state teach pattern** — wire TeachEmptyState into remaining pages (orgs, rbac, webhooks, sso, api_keys)
- [ ] **Auth: email template preview** — rendered HTML preview (needs backend but UI can stub)

### Wave G — Backend-Blocked (ship when endpoints land)
These need Go backend work first. UI stubs can be built but won't function.

- [ ] `GET /admin/health` — may partially exist, verify and complete
- [ ] `GET /admin/config` — may partially exist, verify and complete
- [ ] `POST /admin/test-email` — Settings "Send test" button
- [ ] `POST /admin/sessions/purge-expired` — Settings purge action
- [ ] `POST /admin/audit-logs/purge` — Settings purge action
- [ ] `GET /users/{id}/oauth-accounts` + DELETE — User Security tab OAuth section
- [ ] `GET /users/{id}/passkeys` — User Security tab passkeys list
- [ ] Users: `last_login_at` field — accurate Last Active column
- [ ] Users: filter by role/auth_method/org/mfa — server-side query params
- [ ] SSO: users-per-connection count — aggregation query
- [ ] Signing keys: rotate endpoint — Signing Keys rotate action
- [ ] Permissions: reverse lookup — Permission Explorer backend
- [ ] `GET /admin/email-preview/{template}` — Auth page template preview
