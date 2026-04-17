# SharkAuth Admin Dashboard — Feature & UX Specification (v2)

**Framework:** Svelte 5 + SvelteKit (static adapter) — embedded in Go binary via `go:embed`
**Path:** served at `/admin/*` from single binary
**Logo:** `sharky.svg` at repo root (square, black bg, shark + "shark" text). Use full logo in login screen; use icon-only crop in sidebar.
**Goal:** most ergonomic auth dashboard in market. Every API action reachable. Zero terminal needed. Faster than Auth0, prettier than Clerk, deeper than WorkOS.

---

## Context: What Changed Since v1

Dashboard spec v1 assumed backend state at end of Phase 1. Phase 2 + 3 shipped. Agent Auth spec (AGENT_AUTH.md) is next. Surface for dashboard grew significantly:

**Shipped (Phase 2):**
- Organizations (multi-tenant): users can belong to many orgs, org-scoped roles, invitations
- Webhooks: subscriptions, deliveries, HMAC, retry, test-fire
- Admin stats endpoint `/api/v1/admin/stats` + trends `/admin/stats/trends`
- Admin session list/revoke `/admin/sessions`
- Dev Inbox `/admin/dev/emails` (dev mode only)
- CLI `shark init` (1 question), `shark email setup`, `shark app create`

**Shipped (Phase 3):**
- JWT mode (cookie ↔ JWT switch via `auth.jwt.mode`)
- JWKS endpoint `/.well-known/jwks.json` + signing key rotation
- Per-JTI revocation (`/api/v1/admin/auth/revoke-jti`, `/api/v1/auth/revoke`)
- Applications table: `client_id`, `client_secret_hash`, redirect URI allowlist, CORS per-app
- Redirect validator unified across OAuth / magic-link / password-reset
- Org-scoped RBAC (`org_roles`, `org_members`, per-org permissions)

**Coming (Phase 5–10, pre-wire dashboard shells now):**

Phase 5 — SDK:
- API Explorer / playground (in-dashboard curl + TS SDK snippets per endpoint)
- Session debugger ("paste cookie/JWT → decode + validate")
- Webhook event schema reference browser

Phase 6 — Agent Auth:
- Agents (first-class `agent_` identities)
- OAuth 2.1 server (authorize, token, device, introspect, revoke, DCR, PAR)
- OAuth consents (per-user, per-agent, per-scope)
- Token Vault (third-party managed OAuth)
- Device-flow approval UI
- DPoP indicators

Phase 7/8 — Proxy + OIDC Provider:
- Proxy config + upstream health (shark proxy --upstream)
- Header injection rules editor
- OIDC Provider mode: clients list (this instance as IdP), federation pairings, `.well-known/openid-configuration` view

Phase 9 — Polish & Enterprise:
- Impersonation (admin → user; banner + session timer; all actions audit-flagged)
- Compliance toolkit (GDPR export, right-to-erasure, SOC2 access-review reports, session geography)
- Email provider presets UI (swap Resend/SES/SMTP without YAML)
- shark.email relay quota + upgrade
- Migration wizards (Auth0 / Clerk / Supabase importers w/ dry-run)
- Pre-built UI component config (branding, `<shark-sign-in>` theme editor)
- Dashboard editor (custom dashboards per-role — Phase 9 issue #68)
- `docs_url` surfaced on every error (Phase 9 cross-cutting)

Phase 10 — Moonshot:
- Visual flow builder (drag-and-drop auth flow, export YAML)

---

## Guiding Principles (revised)

1. **Progressive disclosure.** Overview surfaces what matters. Detail one click away.
2. **Act where you see.** Inline action on every row. No "go to settings to change this."
3. **Keyboard-first, mouse-friendly.** Cmd+K reaches everything. `g u` → Users, `g a` → Audit, etc.
4. **Optimistic UI.** Every mutation updates UI instantly, rolls back on error. No spinners except on initial load.
5. **Undo over confirm.** Destructive action → toast w/ 5s undo. Exception: delete user + delete agent (requires typing identifier).
6. **Live where it matters.** Audit log, active sessions, device-flow approvals, agent token usage → poll/SSE. User list static.
7. **Copy anywhere.** IDs, client secrets (once), curl snippets, error codes — one click.
8. **Empty states teach.** Every page has "how to create your first X" w/ copy-paste CLI or curl fallback.
9. **Zero terminal goal.** Every `shark` CLI verb has dashboard equivalent. Dashboard shows matching CLI command for every action (for scripting / CI export).
10. **No modals for multi-step flows.** Slide-overs (right panel) for details. Full-page wizards for onboarding. Modals only for confirms.

---

## Brand / Visual Direction

- **Logo usage:**
  - Login screen: centered `sharky.svg` (full square), 128×128, on solid bg matching theme.
  - Sidebar top: icon-only crop of shark (not full square), 32×32, click → Overview.
  - Favicon: 32×32 icon-only export.
- **Palette:** black/white contrast like the logo. Primary accent `#0066ff` (Shark blue, matches RM.md badge). Danger `#ff3b30`. Success `#00c853`. Warning `#ff9500`.
- **Typography:** Inter (system fallback: -apple-system, Segoe UI). Monospace: JetBrains Mono for IDs, tokens, curl.
- **Density:** compact by default (Linear-style), with "Comfortable" toggle in settings.
- **Dark mode:** default dark (logo was designed on black). Light mode switch in top bar.
- **Motion:** 120ms ease-out for panels, 60ms for hover. No bounces. Reduced-motion respected.

---

## Navigation Structure

Flat left sidebar. Collapsible to icons. Sections grouped by noun.

```
[shark icon → Overview]
------------------------
HUMANS
  Overview                   metrics, activity, health
  Users                      table, detail, CRUD
  Sessions                   active sessions, revoke
  Organizations              multi-tenant, per-org drill-down
------------------------
AGENTS                       (Phase 6 — shell visible, empty-state teaches)
  Agents                     first-class agent identities
  Consents                   per-user agent grants
  Tokens                     active OAuth tokens, revoke
  Vault                      third-party OAuth (Google, Slack, ...)
  Device Flow                pending approvals (live)
------------------------
ACCESS
  Applications               relying parties, redirect allowlist, CORS
  Authentication             OAuth providers, magic links, passkeys, email verify
  SSO                        SAML/OIDC connections, domain routing
  Roles & Permissions        global RBAC
  API Keys                   M2M keys
------------------------
OPERATIONS
  Audit Log                  full event stream
  Webhooks                   subscriptions, deliveries
  Dev Inbox                  captured emails (dev mode only)
  Signing Keys               JWKS, key rotation
  Proxy                      upstream config, header rules         (Phase 7)
  OIDC Provider              Shark as IdP, client clients, federation (Phase 8)
  Settings                   server config, email, danger zone
------------------------
DEVELOPERS                                                           (Phase 5)
  API Explorer               try endpoints, copy curl/TS snippets
  Session Debugger           decode cookie/JWT, validate
  Event Schemas              webhook + audit payload reference
------------------------
ENTERPRISE                                                           (Phase 9)
  Impersonation              active impersonation sessions, audit
  Compliance                 GDPR export/erase, SOC2 reports, geo
  Migrations                 Auth0 / Clerk / Supabase wizards
  Branding                   <shark-sign-in> theme, logos, copy
  Flow Builder               visual auth flows                     (Phase 10)
------------------------
[instance: v0.X.Y] [health: green] [env: dev|prod]
```

**Top bar (persistent):**
- Cmd+K global search / command palette
- `+ Quick create` (user, agent, API key, app, role, webhook, SSO conn)
- Notification bell (lockouts, failed webhooks, expiring keys/tokens, pending device approvals)
- Org switcher (if admin spans orgs — cloud only)
- Theme toggle, profile menu (impersonation indicator if active)

---

## Page Specifications

### 1. Overview

Landing page. "Is my system healthy? What happened? What needs attention?"

**Metric strip (top, 6 cards, ≤10ms refresh from `/admin/stats`):**

| Card | Value | Sub | Source |
|---|---|---|---|
| Users | total | +N last 7d | `stats.users` |
| Active Sessions | count | — | `stats.sessions.active` |
| MFA Adoption | pct | N of M | `stats.mfa` |
| Failed Logins 24h | count | trend ↑↓ | `stats.failed_logins_24h` |
| API Keys Active | count | N expiring 7d | `stats.api_keys` |
| Agents Active | count | N tokens live | **new (Phase 6)** |

Every card links to its page. Sparkline underneath pulled from `/admin/stats/trends` (already shipped).

**Attention panel (right rail):**
Surfaces anything needing admin eyeballs:
- Pending device-flow approvals (live)
- Failed webhook deliveries (last 24h)
- Expiring API keys / agent secrets / signing keys (< 7d)
- Unverified admin email on primary account
- Config warnings: "shark.email testing tier in use — switch before prod" / "no redirect allowlist on app X"
- Security anomalies: sudden scope escalation, bulk failed login single IP

**Auth method breakdown** (donut): sessions grouped by `auth_method` last 30d.

**Recent activity** (live feed, 20 events, `/api/v1/audit-logs?limit=20`): 10s poll (swap to SSE once server supports).

**System health:**
- DB size, uptime, version, migration cursor
- Config summary: SMTP status, OAuth providers enabled, SSO connections count, JWT mode, dev mode flag
- Source: `GET /api/v1/admin/health` (**still needed** — not yet shipped)

---

### 2. Users

**Table columns:**

| Col | Content | Sort | Filter |
|---|---|---|---|
| User | avatar + name + email | name, email | text search |
| Verified | badge | — | yes/no |
| MFA | shield icon | — | on/off |
| Auth Method | icon (last login) | — | password/oauth/passkey/magic/sso |
| Org(s) | chip w/ count | — | by org |
| Roles | chips, +N overflow | — | by role |
| Created | relative | yes | date range |
| Last Active | relative | yes | — |

**Backend needed still:** `last_login_at` on users. Filter by auth_method / role / org on `GET /users`.

**Row click → right slide-over, tabs:**

- **Profile:** inline-edit name, email, verified toggle, metadata JSON. Read-only: ID (copy), created/updated. Actions: Send Verification, Reset Password (admin-triggered), Disable MFA, Impersonate (Phase 9), Delete (requires typing email).
- **Security:** MFA status + enrolled date, passkeys list (rename/delete), OAuth accounts (unlink), active sessions (revoke one / all). Endpoints exist: `/users/{id}/sessions`, `DELETE /users/{id}/sessions`.
- **Roles & Permissions:** global roles assigned, effective permissions w/ source role, "check permission" tool.
- **Organizations:** list memberships w/ per-org role, invite, remove.
- **Agent Consents (Phase 6):** agents this user granted access to, per-agent scopes + RAR details, revoke → revokes all tied tokens.
- **Activity:** per-user audit log filtered to `actor_id=user_id OR target_id=user_id`.

**Batch actions (checkbox):** delete N, assign role, add to org, export CSV.

---

### 3. Sessions

Already shipped `/admin/sessions`. Table:

| Col | Content |
|---|---|
| User | avatar + email |
| Auth | icon + method |
| Mode | cookie / JWT badge |
| IP | + country (GeoIP future) |
| Device | parsed UA |
| MFA | passed / pending |
| Created | relative |
| Expires | relative, red if <24h |
| JTI | short hash + copy (JWT only) |
| Actions | Revoke |

**Aggregate cards above:** total, by method, MFA rate, by device type (future UA parsing).

**JWT-mode extras:** "Revoke by JTI" input (wired to `/admin/auth/revoke-jti`), "Rotate signing keys" shortcut to Signing Keys page.

---

### 4. Organizations *(new)*

Two-pane: left = list, right = detail.

**List columns:** name, slug, members count, roles count, SSO domain (if any), created. Filter by SSO-enabled, by member count.

**Detail tabs:**
- **Overview:** slug, display name, metadata, created_by. Inline edit.
- **Members:** table (user, role, invited_by, joined_at). Bulk invite CSV. Actions: change role, remove.
- **Invitations:** pending list (email, role, expires, token). Resend, revoke.
- **Roles & Permissions (org-scoped):** full CRUD, distinct from global roles. Uses org RBAC endpoints.
- **SSO Enforcement:** toggle require-SSO, bind domain.
- **Audit:** filtered to `org_id=...`.

Empty state: "No orgs yet. Create one via `shark org create` or +New Org."

---

### 5. Applications *(new)*

Relying parties. Covers redirect allowlist (Phase 3 shipped). Each app = `client_id` + secret.

**Table:** name, client_id (copy), redirect URIs (n), CORS origins (n), created, actions (Edit / Rotate / Delete).

**Create/Edit slide-over:**
- Name, description
- Allowed Callback URLs (list, exact-match; wildcard subdomain allowed `https://*.preview.vercel.app`; loopback `http://127.0.0.1:*` toggle)
- Allowed Logout URLs
- Allowed Origins (CORS)
- Default post-login URL
- On create: secret shown once in banner w/ copy; "will not be shown again."

**Rotate secret:** modal, shows new secret once, old invalidates immediately.

**CLI parity footer:** `shark app create --callback https://...` copy-paste snippet.

---

### 6. Authentication *(refactored)*

Config-style sections. Reads from `/admin/config` (**still needed** — surface non-sensitive YAML values).

- **Password Policy:** min length, complexity flags, lockout threshold + duration. (Phase 2 moved lockout to config — wire values here.)
- **OAuth Providers:** card per provider (Google, GitHub, Apple, Discord). Status, client_id (masked), scopes, redirect URL (computed). Test button → opens flow in new tab. **God-tier:** edit client_id/secret without restart (requires DB-backed provider config — deferred).
- **Magic Links:** token lifetime, redirect URL (derived from `app_url`), rate limit (read-only 1/60s).
- **Passkeys:** RP name, RP ID, origin, attestation pref, UV setting, registered count.
- **Email Verification:** SMTP status, template preview via `/admin/email-preview/{template}` (**still needed**).
- **JWT Mode:** radio cookie | jwt. Access lifetime, refresh lifetime, signing alg (ES256/RS256). Warning on switch: existing sessions invalidated. (Phase 3 config exposed — wire UI.)

---

### 7. SSO

(From v1, still applies.) Add: per-connection user count via `SELECT connection_id, COUNT(*) FROM sso_identities GROUP BY ...`. Domain routing tester: enter email → shows which connection routes.

---

### 8. Roles & Permissions

Two-pane: roles left, detail right. Global roles only on this page; org-roles live under Organizations > {org} > Roles.

**Role detail:** inline-edit name/description, permissions (attach/detach, create-and-attach inline), users-with-role list w/ assign.

**Permission Explorer tab:** all permissions + usage count. "Check Permission" tool: pick user → action + resource → allow/deny w/ rationale (which role granted).

Reverse lookup endpoints **still needed**: `GET /permissions/{id}/roles`, `/permissions/{id}/users`.

---

### 9. API Keys

(v1 spec still applies.) Add: agent-linked keys callout — if agent auth lands, keys and agent `client_secret`s merge into unified "credentials" model. Until then keep separate.

---

### 10. Agents *(new — Phase 6)*

First-class. Ship shell empty-state pre-Phase-6.

**Table columns:**

| Col | Content |
|---|---|
| Agent | logo_uri + name |
| Client ID | copy |
| Type | confidential / public badge |
| Grants | chips (client_credentials, auth_code, device, token_exchange) |
| Scopes | chips, +N overflow |
| Active Tokens | count → links to Tokens pre-filtered |
| Last Used | relative |
| Created By | user email |
| Status | Active / Disabled toggle |
| Actions | Edit, Rotate Secret, Revoke All Tokens, Delete |

**Create wizard (full-page, 4 steps):**
1. **Identity:** name, description, logo URL, homepage.
2. **Type:** confidential (server) / public (SPA/CLI) → informs auth_method picker.
3. **Grants + Scopes:** multi-select grants; scopes multi-select + freeform; resource (RFC 8707 audience).
4. **Credentials:** client_id generated; secret shown once (confidential); `jwks`/`jwks_uri` (private_key_jwt option); token lifetime slider (5–60 min).
5. Review → "Create" → secret reveal banner + curl snippet + TS SDK snippet.

**Detail tabs:**
- **Overview:** config (inline edit), CLI snippet `shark agent show <id>`.
- **Tokens:** active tokens for this agent, JTI, scope, user (if delegated), DPoP bound (✓/✗), expires, actions Revoke. Count + bulk revoke-all.
- **Consents:** users who granted, per-scope.
- **Audit:** `actor_type=agent AND actor_id=<id>`.
- **Usage:** token issuance rate chart, top scopes, top resources, error rate (7d / 30d).
- **Delegation:** graph — which agents this agent delegates to / from (RFC 8693 `act` chain).

**Delete** requires typing agent name. Revokes all tokens in same transaction.

---

### 11. OAuth Consents *(new — Phase 6)*

User-centric view of agent access grants. Cross-reference of `oauth_consents` table.

**Columns:** user, agent, scopes, RAR summary, granted_at, expires_at, revoked_at, actions.

**Filters:** by user, by agent, active / revoked, expiring soon.

**Row click → detail:** full RAR JSON viewer, delegation chain if any, tokens issued under this consent (→ Tokens page pre-filtered).

Revoke → revokes consent + all tied tokens in same transaction. Undo 5s.

---

### 12. Tokens *(new — Phase 6)*

Active OAuth tokens across agents + users. High-volume table.

**Columns:** JTI (short + copy), token type (access/refresh), client (agent), subject (user or —), scope, audience, DPoP ✓, expires, family_id, created, actions Revoke.

**Filters:** client_id, user_id, token_type, dpop_bound, expires_before, family_id (for tracing refresh chains).

**Row detail:** full JWT decoded (header + payload, sig hidden), `act` chain viz, `authorization_details` pretty-printed.

Bulk revoke by family_id (detects refresh chain reuse → whole family).

---

### 13. Vault *(new — Phase 6)*

Two sub-tabs.

**Providers (admin-managed):**
Cards: Google, Slack, GitHub, Microsoft, Notion, Linear, Jira (templates pre-filled). Each card:
- Status: Configured / Template available
- Client ID (masked), client secret (rotate)
- Default scopes (editable chips)
- Icon, display name
- Test connect button

**Connections (per-user, admin read-only, user-actionable in user profile):**
Table: user, provider, scopes, expires, last_refreshed, status (active / expired / refresh-failed), actions Disconnect.

Filters: by provider, expiring soon, failed refresh.

---

### 14. Device Flow *(new — Phase 6)*

Pending device-code approvals. Live (5s poll or SSE).

**Cards (not table — few at a time):** user_code (big, monospace), agent name + logo, scopes requested, resource, IP of requesting device, expires countdown, "Approve" / "Deny" buttons (admin can approve for testing). User-facing approval lives at `/oauth/device/verify?user_code=...`.

Empty state: "No pending device approvals. Agents using device flow will appear here."

---

### 15. Audit Log

(v1 still applies.) Additions:
- `actor_type` filter includes `agent`, `system`.
- Agent-specific filter: `agent_id`.
- Delegation chain filter: "user acted through agents" → traces `act` chain.
- Event types extended: `oauth.*`, `agent.*`, `vault.*`, `webhook.*`, `application.*`.
- Export CSV preserves filters.
- Live mode toggle already in v1.

---

### 16. Webhooks *(new — shipped Phase 2)*

**Table:** URL, events subscribed (chips), status (active / disabled), last delivery status, success rate 7d, actions.

**Create slide-over:** URL, events multi-select (user.created, user.deleted, session.created, mfa.enabled, oauth.token.issued, agent.registered, vault.connected, ...), secret (auto-gen, shown once), retry policy, description.

**Detail tabs:**
- **Config:** inline edit.
- **Deliveries:** table — event, timestamp, status code, attempts, duration, request body (expand), response body, replay button.
- **Signature verify:** paste a payload + signature → shows valid/invalid + computed HMAC (debug aid).
- **Test:** fire arbitrary event type w/ sample payload.

---

### 17. Dev Inbox *(new — dev mode only)*

Hidden when `cfg.Server.DevMode = false`. Uses `/admin/dev/emails`.

**List:** to, subject, type (magic_link / verify / password_reset), received.
**Detail:** full HTML preview + raw source toggle. "Open magic link" shortcut. Delete all button.

Banner: "Dev inbox captures all outgoing mail. Switch email provider before production: `shark email setup`."

---

### 18. Signing Keys *(new — Phase 3)*

For JWT mode. Reads `signing_keys` table + `/.well-known/jwks.json`.

**Table:** kid, algorithm, active (badge), created, rotated_at.

**Actions:**
- **Rotate now:** generates new key, marks old inactive (still verifies). Warning modal: "Clients caching JWKS may see transient failures within cache TTL."
- **Retire key** (only after grace period): removes from JWKS.
- **Download JWKS:** curl-ready snippet.

---

### 19. Settings

- **Server:** port (read-only), base_url (copy), app_url, secret status ("configured, 64 chars"), CORS origins.
- **Email:** provider, from, "Send test email" → `POST /admin/test-email` (**still needed**), shark.email tier badge + quota.
- **Session:** mode (cookie/JWT), lifetime, active count, "Purge expired" (**still needed**: `/admin/sessions/purge-expired`).
- **Audit Retention:** policy, row count, DB size, "Purge before date" (**still needed**).
- **Danger Zone:** rotate server secret (docs link to SECRETS.md), export all data (future), delete all users (types "DELETE ALL USERS").

---

### 20. API Explorer *(new — Phase 5)*

In-dashboard Postman. Every SharkAuth endpoint browsable.

**Layout:** left = endpoint tree (grouped: Auth, Users, Orgs, OAuth, Webhooks, Vault, Agents, Admin). Right = request builder.

**Request builder:**
- Method + path auto-filled
- Params form (typed from OpenAPI spec — Phase 5 emits one)
- Headers (auto-injects admin API key or session cookie)
- Body (JSON editor w/ schema validation + autocomplete)
- "Send" → shows status, headers, body (pretty + raw), timing
- **Snippets tab:** curl, TypeScript (`@sharkauth/js`), Python, Go — live-updating from current params.
- **Save request:** named collection per-admin. Shareable via URL (encoded, no secrets).

Rate-limit warning banner when approaching per-key ceiling.

### 21. Session Debugger *(new — Phase 5)*

Paste a session cookie or JWT → decoded + validated locally (no server roundtrip unless needed).

**Shows:**
- For cookies: decrypt via JWKS / secret (server call w/ admin key), show payload, expiry, user, MFA status.
- For JWTs: header (kid, alg), payload (claims pretty + explained: iss, sub, aud, exp, scope, `act` chain, `cnf.jkt`), signature valid? (fetches JWKS locally).
- Validation verdict: ✅ valid / ⚠️ expiring soon / ❌ reason (bad sig, expired, revoked JTI, wrong aud).
- "Revoke this JTI" shortcut.

### 22. Event Schemas *(new — Phase 5)*

Reference browser for webhook + audit event payloads.

**Per event:**
- Name, when fired, sample payload (JSON), field descriptions, SDK type snippet (TS interface).
- "Subscribe to this via webhook" → jumps to Webhooks create w/ event pre-selected.

### 23. Proxy *(new — Phase 7)*

Config + health for `shark proxy --upstream`.

**Sections:**
- Upstream URL (editable), health probe status, request volume 24h, error rate.
- **Header injection rules:** configurable mapping — which user/session fields become `X-Shark-*` headers. Default: `X-Shark-User-ID`, `X-Shark-User-Email`, `X-Shark-User-Roles`, `X-Shark-Org-ID`.
- **Path rules:** per-route auth requirements (public / authed / role-gated). Table edit.
- **Preview mode:** "test a request" → shows headers injected without hitting upstream.

### 24. OIDC Provider *(new — Phase 8)*

Shark acts as IdP ("Sign in with YourApp").

**Sections:**
- Discovery URL (`/.well-known/openid-configuration`) copy.
- **Clients (OIDC RPs):** table — client_id, name, redirect URIs, scopes, actions. Shares infra w/ Applications page but distinct semantic (OIDC RPs vs first-party apps). Link "Also appears in Applications."
- **Federation pairings:** other Shark instances / third-party IdPs to federate with. Upstream + downstream config.
- Claims mapping editor: which user fields → which OIDC claims.

### 25. Impersonation *(new — Phase 9)*

Admin → user sessions. Audit-heavy.

**Active table:** admin, impersonated user, started, expires, reason (required field), actions Revoke.

**Start flow:** pick user → type reason (required, becomes audit detail) → creates short-lived session → new tab opens as user w/ persistent red "Impersonating {email} — Stop" banner. All actions double-logged (`actor_id=admin, on_behalf_of=user`).

Global rate-limit: max concurrent impersonations per admin (config).

### 26. Compliance *(new — Phase 9)*

**GDPR:**
- Export user data (full JSON bundle, signed) — `POST /api/v1/users/{id}/export`.
- Right-to-erasure wizard — preview impact (sessions, audit refs, org memberships), then execute.

**SOC2 / Access Reviews:**
- "Who has admin access" report
- "Permission change history" (from audit)
- "Orphaned accounts" (no login in N days)
- Export PDF / CSV.

**Session Geography:**
- Map visualization of active sessions by GeoIP.
- Anomaly flags: impossible travel, unusual countries.

### 27. Migrations *(new — Phase 9)*

Wizards for Auth0, Clerk, Supabase.

**Per source:**
- Upload export file (JSON or CSV, source-specific).
- **Dry-run:** parses, maps fields, shows table of would-be-imported users + conflicts (duplicate emails). Warnings list.
- Select: preserve IDs? map providers? attach role? send welcome email?
- **Execute:** streams progress, writes audit events, final report w/ per-row status + download failed-rows CSV.

Password hashes preserved where compatible (bcrypt, scrypt — Shark auto-detects on next login and migrates to Argon2id transparently).

### 28. Branding *(new — Phase 9)*

Config for `<shark-sign-in>`, `<shark-user-button>`, email templates.

**Sections:**
- Logo upload (light + dark variants)
- Primary color + accent picker (live preview)
- Font stack
- Copy overrides (sign-in title, CTA labels — per locale)
- Email templates: live preview + WYSIWYG for subject/body, variables autocompleted from schema.
- **Preview pane:** live-rendered `<shark-sign-in>` using current config.
- "Export as npm package config" snippet.

### 29. Flow Builder *(new — Phase 10)*

Visual drag-and-drop auth flow editor. Nodes: Email+Password, Magic Link, OAuth (per provider), Passkey, MFA, Conditional (risk score, org), Redirect, Error. Edges = transitions.

Export YAML. Import existing flow from YAML. Versioned. Stage (draft) → publish. A/B test slot.

Out of scope for dashboard v1; reserve nav + "Phase 10" badge w/ waitlist signup.

---

## Global UX Patterns

### Command Palette (Cmd+K)

Fuzzy search across: users (email/name), agents (name/client_id), apps, orgs, roles, keys, SSO conns, webhooks, tokens (JTI), pages, actions (+new X), recent items.

Results grouped by type. Enter = select. `→` on a result = preview without navigating. Shift+Enter = open in new tab.

Action examples:
- "Create user" / "Create agent" / "Create app"
- "Revoke JTI ..." (paste JTI)
- "Impersonate ..." (type user)
- "Switch org ..."
- "Toggle dark mode"
- "Rotate signing keys"

### Optimistic Mutations

Every update: UI reflects change instantly. On failure: rollback + red toast + Retry button. Keeps dashboard feel faster than real network.

### Toasts

Bottom-right. Types: success, undo (countdown), error (w/ Retry + Copy error ID), info.
Stacked, max 3 visible, rest collapse to "+N more."

### Empty States (teach, don't scold)

Each page has:
1. Headline ("No agents yet.")
2. Why it matters (one sentence).
3. Two paths: dashboard "+Create" button and CLI snippet copy-paste.
4. Docs link.

### Loading

Skeleton screens matching real layout. Never spinners except for global full-page redirects.

### Inline Error

Field-level red border + message below. API errors → toast w/ error code + `docs_url` (Shark error responses ship `docs_url`).

### Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| Cmd+K | palette |
| g u / g o / g a / g k / g w / g d | Users / Orgs / Audit / Keys / Webhooks / Dev Inbox |
| g g | Agents |
| g v | Vault |
| g t | Tokens |
| / | focus table search |
| n | New (contextual) |
| e | Edit selected row |
| r | Refresh |
| ? | show shortcut cheatsheet |
| Esc | close panel/modal |

### Deep Linking

URL encodes state. Filter state + open panels restored on reload.
Examples:
- `/admin/users?search=alice&verified=true`
- `/admin/users/usr_abc/security`
- `/admin/agents/agent_xyz/tokens`
- `/admin/audit?actor_type=agent&agent_id=agent_xyz&from=2026-04-01`
- `/admin/tokens?family=fam_abc`

### Responsive

Desktop-first ≥1200px. ≥768px: sidebar → icons. <768px: sidebar → hamburger, tables → card list, slide-overs → full-screen.

### CLI Parity Footer

Every detail panel shows equivalent `shark ...` command at bottom w/ copy. Lets admin script anything done in UI.

---

## Authentication (dashboard itself)

**Bootstrap:** first run shows full-page login. Options:
- Paste admin API key (`sk_live_...`) — verifies against `/api/v1/admin/stats` w/ Bearer.
- "Use CLI" — prints `shark admin login` which writes short-lived JWT to local store.

Key kept in **sessionStorage** (cleared on tab close) — never localStorage. Optionally "remember device" stores encrypted key in IndexedDB w/ WebAuthn unlock (Phase 2 dashboard polish).

Admin session UI shows "Signed in as admin via API key ...xK9f" w/ sign-out button.

---

## Backend Endpoints — Status

Legend: ✅ shipped, 🟡 partial, ❌ still needed, 🔮 Phase 6 (agent auth).

| Endpoint | Status |
|---|---|
| `GET /api/v1/admin/stats` | ✅ |
| `GET /api/v1/admin/stats/trends` | ✅ |
| `GET /api/v1/admin/sessions` | ✅ |
| `DELETE /api/v1/admin/sessions/{id}` | ✅ |
| `POST /api/v1/admin/sessions/purge-expired` | ❌ |
| `GET /api/v1/admin/health` | ❌ |
| `GET /api/v1/admin/config` | ❌ |
| `GET /api/v1/admin/email-preview/{template}` | ❌ |
| `POST /api/v1/admin/test-email` | ❌ |
| `POST /api/v1/admin/audit-logs/purge` | ❌ |
| `GET /api/v1/users/{id}/sessions` | ✅ |
| `DELETE /api/v1/users/{id}/sessions` | ✅ |
| `GET /api/v1/users/{id}/oauth-accounts` | ❌ |
| `DELETE /api/v1/users/{id}/oauth-accounts/{id}` | ❌ |
| `GET /api/v1/users/{id}/passkeys` | 🟡 (store exists, endpoint needed) |
| Users: `last_login_at` | ❌ |
| Users: filter by role / auth_method / org / mfa | 🟡 |
| `/api/v1/organizations/*` | ✅ |
| `/api/v1/webhooks/*` | ✅ |
| `/api/v1/admin/apps/*` | ✅ |
| `/api/v1/admin/auth/revoke-jti` | ✅ |
| `GET /.well-known/jwks.json` | ✅ |
| Signing keys CRUD / rotate endpoint | ❌ |
| `POST /api/v1/admin/impersonate/{id}` | 🔮 (Phase 9) |
| SSO users-per-connection count | ❌ |
| Permissions reverse lookup | ❌ |
| OAuth 2.1: authorize, token, revoke, introspect, device, register | 🔮 |
| Agents CRUD | 🔮 |
| Consents list / revoke | 🔮 |
| Tokens list / revoke / family-revoke | 🔮 |
| Vault providers + connections | 🔮 |

---

## Information Architecture

```
Overview                (metrics, attention, activity, health)
Humans
  Users                 (table → detail tabs)
  Sessions              (active, revoke, aggregate)
  Organizations         (list → detail → members, invites, org roles, SSO)
Agents                  (Phase 6 — shells live now, empty-states teach)
  Agents
  Consents
  Tokens
  Vault (Providers | Connections)
  Device Flow
Access
  Applications          (relying parties, redirect allowlist)
  Authentication        (OAuth, magic, passkey, email, JWT mode)
  SSO                   (SAML/OIDC, domain routing)
  Roles & Permissions   (global RBAC + explorer)
  API Keys              (M2M)
Operations
  Audit Log             (filters, live mode, export)
  Webhooks              (subs, deliveries, replay, test)
  Dev Inbox             (dev mode only)
  Signing Keys          (JWKS, rotate)
  Settings              (server, email, session, retention, danger)
```

---

## Implementation Notes (Svelte)

- **Stack:** Svelte 5 (runes), SvelteKit static adapter, Vite, TypeScript.
- **Build output:** `admin/dist/` → embedded via `go:embed admin/dist/*` in `internal/api`, served under `/admin/*` by chi. Router handler returns `index.html` for all unknown `/admin/*` to support client-side routing.
- **Styling:** Tailwind v4 + shadcn-svelte components (copy-in, owned locally). No component library lock-in.
- **State:** svelte stores for session/auth + palette. TanStack Query (svelte) for server state (caching + optimistic).
- **Routing:** SvelteKit file-based routes.
- **Bundle target:** < 180KB gz (room for richer UI vs v1's 150KB; still ships in <1 RTT on LAN).
- **Icons:** lucide-svelte (tree-shaken).
- **Charts:** unovis or layerchart (svelte-native, small).
- **JWT decoding:** pure-JS in browser (no network) for token inspector.
- **Auth:** Bearer `sk_live_...` in sessionStorage → attached to fetch via interceptor. JWT-mode admin future.
- **Accessibility:** keyboard navigation for every action. ARIA labels on icon-only buttons. Color-contrast AA min.
- **i18n:** stub i18n setup day 1 (even if only English ships) to avoid retrofit later.
- **Tests:** Playwright for golden paths (login, create user, create app, rotate key, create agent shell).

---

## Phase 4 Execution Plan (Dashboard)

Suggested wave structure for `gsd:execute-phase`:

**Wave 1 — Foundation**
- Scaffold SvelteKit app under `admin/` (source), `admin/dist/` (build output)
- Wire `go:embed` + chi static handler + SPA fallback
- Base layout: sidebar (w/ sharky icon), top bar, theme toggle, login screen w/ full sharky.svg
- Auth: API key login, sessionStorage, fetch interceptor
- Cmd+K shell (empty)

**Wave 2 — Core pages (existing endpoints)**
- Overview (stats + trends + attention panel stubs)
- Users (table + detail tabs: Profile, Security, Roles, Organizations, Activity)
- Sessions
- API Keys
- Audit Log (filters + live mode + CSV export)
- Settings (read-only sections wiring existing config)

**Wave 3 — Phase 2/3 features**
- Organizations (two-pane list + detail)
- Applications (redirect allowlist CRUD)
- Webhooks (CRUD + deliveries + replay + test)
- Signing Keys (JWKS view — rotate endpoint if ready)
- Dev Inbox (conditional on dev mode)
- JWT mode toggle in Authentication page

**Wave 4 — Missing endpoints + polish**
- Ship `/admin/health`, `/admin/config`, `/admin/test-email`, purge endpoints, user oauth-accounts + passkeys endpoints
- Command palette fully wired (search users/agents/apps/actions)
- Keyboard shortcuts
- Empty states + CLI snippet footers
- Optimistic mutations + undo toasts
- Responsive passes

**Wave 5 — Agent shells (pre-Phase-6)**
- Render Agents / Consents / Tokens / Vault / Device Flow nav items
- Empty states w/ "Coming Phase 6" + link to AGENT_AUTH.md
- No backend calls; pure presentational shells
- Ensures Phase 6 backend can land w/ dashboard ready

**Wave 6 — Tests + docs**
- Playwright golden paths
- Screenshots for HN demo GIF
- Bundle size audit, a11y audit

---

## Future Phase Shells (build empty nav now, wire later)

Each downstream phase has at least a nav entry + empty-state page shipped in Phase 4 so Phase N backend lands against a dashboard already expecting it. Zero dashboard churn in Phases 5–10.

| Phase | Pages added | Shell lands in Phase 4? |
|---|---|---|
| 5 SDK | API Explorer, Session Debugger, Event Schemas | Yes — link to docs as placeholder |
| 6 Agent Auth | Agents, Consents, Tokens, Vault, Device Flow | Yes — teach empty-state |
| 7 Proxy | Proxy | Yes — "Not configured" state |
| 8 OIDC Provider | OIDC Provider | Yes — shows discovery URL if enabled |
| 9 Enterprise | Impersonation, Compliance, Migrations, Branding | Yes — dim w/ "Phase 9" badge |
| 10 Moonshot | Flow Builder | Yes — waitlist CTA |

Shell pages are cheap (single `<EmptyState>` component reused) but critical: they mean Phase 6+ is a backend-only phase from dashboard's POV. Wave 5 in execute plan covers agent shells; extend same wave to cover Phase 7–10 shells.

---

## Notes / Deferred

- **Cloud (Next.js clone):** do NOT build in this repo. Separate repo mirrors UX 1:1, talks to multi-tenant API. Note only.
- **Pre-built UI components (`<shark-sign-in>`):** Phase 9, separate npm package. Branding page feeds its config.
- **Dark mode first:** logo designed on black bg → default dark. Light mode must still pass contrast.
- **OpenAPI spec:** API Explorer (Phase 5) depends on OpenAPI emission. If not auto-generated by Phase 5 backend work, add sub-task: `chi-openapi` or hand-authored spec at `/openapi.json`.
- **GeoIP:** Compliance + Sessions benefit from GeoIP lookup. MaxMind GeoLite2 embedded optional (adds ~50MB) OR optional SaaS — Phase 9 decides.
- **Flow Builder:** nav entry only. Real build = Phase 10. Do not design components yet.

---

*v2 covers feature/UX logic + brand direction. Detailed component design (shadcn tokens, motion curves, exact spacing) defers to first Wave 1 PR review. Goal unchanged: most ergonomic auth dashboard in market, zero terminal needed, ships embedded in single Go binary.*
