---

Phase 1 — Harden the foundation (2-3 days)

- Fix OIDC nonce validation (#42)
- Add request body limits (#43)
- Switch to slog (#46)
- Makefile + CI (#47)
- Tests across major components (#65) — start here, keep adding throughout

Done

Phase 2 — Make it usable (3-4 days)

- Organizations (#50) — the #1 feature gap vs every competitor
- Webhooks (#51) — required for any real integration
- CLI subcommands (#53) — shark init && shark serve is the first thing anyone
- shark init is now **1 question** (base_url). secret auto-generated, email
  defaults to shark.email testing tier. Server boots end-to-end with zero
  extra config. Production switch via `shark email setup` or YAML edit.
- Dev mode with email capture (#48) — removes SMTP friction for every developer
  trying Shark and scaffolding of shark.email
- Admin stats endpoint (#44)
- Session management endpoints (#45)  

Now someone can install, run, and manage Shark from the terminal without reading
YAML docs.


- Done

Phase 3 — JWT + OAuth 2.1 infrastructure (5-7 days)
- RBAC on orgs.
- JWT auth system.
- Configurable session mode — cookie vs JWT (#67)
- This is the foundation that Agent Auth, OIDC Provider, and edge auth all share

Build this once, everything after uses it.

Done.

### Redirect URI allowlist (Shark Cloud blocker)

Today: `server.base_url` is single string. Used for magic-link/email-verify URLs,
OAuth provider callback construction, cookie Secure flag. There is NO per-client
`redirect_uri` allowlist anywhere — `social.redirect_url`, `magic_link.redirect_url`,
`password_reset.redirect_url` are single global YAML strings. `cors_origins` is a
list but only governs browser CORS preflight, not OAuth redirects.

Why dodgeable today: OAuth flow is server-initiated. Caller hits
`/api/v1/auth/oauth/{provider}/start`, shark redirects to Google/etc, final landing
= single config value. No user-controlled `redirect_uri` query param → no open
redirect surface.

Why blocker for Cloud:
1. Cloud must expose `/oauth/authorize` so third-party apps consume Shark as IdP
   (SDK.md "Shark as IdP" path) → user-supplied `redirect_uri` enters request
2. Multi-tenant: each tenant's app has own callback URLs, can't be one global string
3. SPA/mobile SDKs pass own `redirect_uri` (deep links, localhost dev ports)
4. Without exact-match allowlist → open redirect → auth code theft → account takeover
   (OAuth 2.1 §3.1.2 mandate)

Compare:
- Auth0: app config "Allowed Callback URLs" (comma list per client_id)
- Clerk: instance "Allowed redirect URLs" + per-instance origin allowlist
- Keycloak: client "Valid Redirect URIs" (wildcard supported)
- Cognito: app client "Callback URLs" exact list

Build:
- `applications` table: `id`, `tenant_id`, `client_id`, `client_secret_hash`,
  `allowed_callback_urls []string`, `allowed_logout_urls []string`,
  `allowed_origins []string`
- `/authorize` validator: exact-match `redirect_uri` against allowlist, reject with
  `invalid_request` on miss (never redirect to bad URI — render error page instead,
  per OAuth 2.1)
- Wildcard support: `https://*.preview.vercel.app` style (subdomain only, no path)
- Loopback exception: `http://127.0.0.1:*` allowed if any loopback URL registered
  (RFC 8252 native app pattern)
- Same allowlist enforced on: post-magic-link redirect, post-password-reset redirect,
  post-social-login redirect → unify under single `redirect_validator` package

### Ergonomic / configless reflection

Phase 2 already collapsed install to **1 question** (`base_url`). Secret is
auto-generated, email defaults to shark.email testing tier → `shark init &&
shark serve` boots end-to-end with zero extra setup. The configless ladder for
self-host now reads:

| Step | What user types | What runs |
|------|-----------------|-----------|
| 1 | `shark serve --dev` | ephemeral secret, dev.db, dev inbox — no YAML |
| 2 | `shark init && shark serve` | 1 question (base_url), shark.email testing tier |
| 3 | `shark email setup` | swap to resend/smtp/ses for production sending |
| 4 | `shark app create --callback ...` | register relying party for OAuth |

Pain remaining: YAML still grows fast once orgs/SSO/OAuth land. Users copy-paste
50-line configs.

Moves toward configless:

1. **Derive everything from `base_url` + `app_url`** (already drafted in SDK.md):
   - `cors_origins` auto-includes `app_url`
   - `magic_link.redirect_url` defaults to `{app_url}/auth/callback`
   - `password_reset.redirect_url` defaults to `{app_url}/auth/reset-password`
   - `passkeys.rp_id` derived from `base_url` host
   - User overrides only when non-default. Goal: 90% of installs are 3 lines.

2. **Dashboard-first config for redirect allowlists**: Cloud should never make
   tenants edit YAML for callback URLs. Add through UI → write to `applications`
   table → no restart needed. Self-host can do same via `shark app create
   --callback https://...`.

3. **Auto-detect localhost in dev**: When `--dev` flag set, allow `http://localhost:*`
   and `http://127.0.0.1:*` for any registered app without explicit listing. Kills
   90% of "why doesn't OAuth work locally" issues.

4. **Smart defaults from environment**: Detect Vercel/Netlify/Fly env vars
   (`VERCEL_URL`, `RAILWAY_PUBLIC_DOMAIN`, etc.) → auto-populate `base_url` and
   default callback. Zero-config for PaaS deploys.

5. **Single `app_url` superseding scattered redirect configs**: Replace
   `social.redirect_url` + `magic_link.redirect_url` + `password_reset.redirect_url`
   with one `auth.app_url`. Per-flow override only when user explicitly opts in.
   Reduces config surface by 60%.

6. **Keep `shark init` at 1 question**: Resist adding prompts. New config knobs
   should derive from `base_url` or get optional follow-up commands (`shark email
   setup`, `shark app create`) — never additional init questions. Every extra
   prompt halves completion rate. Write inferred defaults as commented-out YAML
   so users see what was set without docs lookup.

7. **Per-tenant API for Cloud**: Tenants never touch YAML. All config (callback URLs,
   CORS, providers, branding) lives in DB, edited via dashboard or
   `POST /api/v1/admin/apps`. YAML stays self-host-only for instance-level secrets.

Net: self-host = 1-question init + `shark app create` once per relying party.
Cloud = zero YAML, all dashboard, shark.email auto-provisioned per tenant.

Phase 4 — Dashboard (7-10 days) — Done

- Svelte admin dashboard (existing DASHBOARD.md spec)
- nextjs/react dashboard mimiccking svelte but for cloud *do not code this here, just keep a note*
- Depends on: stats endpoint, session endpoints from Phase 2  


This is what turns Shark from an API into a product. The HN demo GIF comes from
this.

Phase 4.5 — Dashboard Polish (3-5 days) — Done (except responsive)

**Spec:** `specdashby.md` — full gap analysis + phased execution plan (Waves A–G)

- **Wave A ✅** — Global UX: Cmd+K, keyboard shortcuts, undo toasts, deep linking, CLI footers, phase shells
- **Wave B ✅** — Settings + Auth config: session mode badge, shark.email tier, JWT toggle UI, passkey config, JWKS download, magic link shortcut
- **Wave C ✅** — Users + Sessions: filter dropdowns, type-email delete, JTI column, revoke-by-JTI input, profile actions
- **Wave D ✅** — Orgs + SSO: invitations tab, SSO enforcement, org audit, domain routing tester, connection detail
- **Wave E ✅** — RBAC + Keys + Webhooks: permission explorer, check-permission tool, curl snippets, replay, test fire, sig verify
- **Wave F ✅** — Teach empty states, user profile actions, API key audit, email preview stub, SSO detail. Responsive deferred.
- **Wave G ✅** — Backend: test-email endpoint, admin passkeys endpoint, signing key rotation, last_login_at migration, user filters, SSO user counts, smoke test sections 14+15

Remaining from specdashby.md: responsive design only. See specdashby.md for details.

Phase 5 — OAuth 2.1 + Agent Auth — Done

**Plan:** `docs/superpowers/plans/2026-04-18-oauth21-agent-auth.md`
**Spec:** `AGENT_AUTH.md`

Build core features FIRST, SDK on top LATER. Agents are first-class citizens.

- ✅ Full OAuth 2.1 Authorization Server — fosite-based, ES256 signing
- ✅ Agent identity (first-class, not user extension)
- ✅ Auth Code + PKCE, Client Credentials, Refresh Token Rotation
- ✅ Dynamic Client Registration (RFC 7591) — MCP discovery
- ✅ Device Authorization Grant (RFC 8628) — headless agents
- ✅ Token Exchange (RFC 8693) — agent-to-agent delegation chains
- ✅ DPoP (RFC 9449) — proof-of-possession tokens
- ✅ Token Introspection + Revocation (RFC 7662, 7009)
- ✅ Resource Indicators (RFC 8707) — audience binding
- ✅ AS Metadata (RFC 8414) — MCP compatibility
- ✅ Consent UI (server-rendered + React)
- ✅ Agent management dashboard + consent management + device flow React
- ✅ Smoke tests: sections 26-42 (AS metadata, agent CRUD, all grants, DPoP, introspection, revocation, DCR, resource indicators, ES256 JWKS, consents) — 181 PASS, 0 FAIL

Phase 5.5 — Token Vault — Done

**Plan:** `docs/superpowers/plans/2026-04-18-token-vault.md`

- ✅ Managed third-party OAuth tokens (Google, Slack, GitHub, Microsoft, Notion, Linear, Jira)
- ✅ Agents request tokens via `/api/v1/vault/{provider}/token` (OAuth Bearer delegation)
- ✅ Auto-refresh on retrieval, AES-256-GCM encryption at rest (FieldEncryptor)
- ✅ 9 pre-built provider templates (snake_case API)
- ✅ Dashboard UI: split grid, create wizard, rotate secret, type-to-confirm delete, audit tab
- ✅ Smoke tests sections 43-48 (provider CRUD, templates, connect, bearer auth, connections, audit) — 222 PASS, 0 FAIL

Phase 6 — Proxy + Visual Flow Builder — Done

**Plan:** `docs/superpowers/plans/2026-04-18-proxy.md`
**Plan:** `docs/superpowers/plans/2026-04-18-visual-flow-builder.md`

- ✅ Shark Proxy (#58) — embedded `shark serve --proxy-upstream` + standalone `shark proxy`
- ✅ Route-level rules engine (path wildcards, method filters, require/allow)
- ✅ Circuit breaker: L1 JWT local verify (agents never down), L2 session cache, L3 health monitor
- ✅ Identity header injection (X-User-ID, X-Agent-ID, X-User-Roles, X-Auth-Method, X-Shark-Cache-Age)
- ✅ Admin API: /admin/proxy/status + /rules + /simulate + SSE status stream
- ✅ Proxy dashboard: 3-gauge circuit strip, URL simulator hero, rules table, read-only MVP
- ✅ Auth Flow Builder — 12 step types (6 wired, 6 stubbed), conditional branches, priority + conditions
- ✅ Flow integration: signup / login / oauth_callback / password_reset / magic_link hooks
- ✅ Flow dashboard: palette + canvas + config + Preview dry-run + History tab
- ✅ Smoke tests sections 49-54 (proxy disabled admin 404s, flow CRUD, dry-run timeline, signup block/disable/runs) — 244 PASS, 0 FAIL

Phase 6.5 — Dashboard Gap Fix — Done

**Spec:** `DASHBOARD_GAPS.md` — full audit + ranked plan + wave breakdown

Backend smoke 375 PASS. Dashboard ~90% wired after Waves A-G. All closed.

Phase 6.6 — Dashboard Deep Audit + Shape Fixes — Done (2026-04-20)

**Trigger:** Smoke green but user reported "malfunctioning." Deep investigation via parallel Sonnet subagents found 24 real bugs (P0/P1/P2) spanning shape mismatches, routing 404s, crashes, hardcoded lies, mock residue. All 24 shipped.

Highlights:
- 3 routing 404s fixed (`/admin/audit`, `/admin/orgs/{id}/roles`, `/admin/orgs/{id}/invitations`)
- 5 JSON shape mismatches fixed (sessions.data, users camelCase→snake_case, proxy PascalCase→snake_case, overview stats triple-bug, orgs members/metadata)
- 1 crash fixed (organizations.tsx ReferenceError)
- 3 authflow stubs wired (assign_role, add_to_org, require_mfa_challenge + new POST /auth/flow/mfa/verify)
- adminConfigSummary expanded (passkey/password_policy/jwt/magic_link/session_mode/social_providers)
- Real CSV audit export + pagination + date-range UI
- 8 new API endpoints (revoke-all sessions, rotate agent secret, batch permission usage, webhook events, delete org member, dev-mode gate, etc.)
- New migration `00002_audit_logs_extended_filters.sql`
- 8 swallowed `_ = store.X` errors now `slog.Warn`
- Dead `agents.tsx` stub deleted

Full detail: `CHANGELOG.internal.md` → Phase 6.6.
Handoff: `HANDOFF.md`.

Phase 7 — SDK (5-7 days)

**Plan:** TBD (builds on OAuth 2.1 flows)
**Spec:** `SDK.md`

- TypeScript SDK (#54) — now a thin OAuth 2.1 client
- React/Svelte/Vue/Next.js wrappers
- Node/Python/Go admin SDKs
- SDK builds on top of real OAuth flows, not custom cookie logic

Phase 8 — OIDC Provider + Polish (5-7 days)

- OIDC Provider mode (#60) — shares OAuth 2.1 infrastructure from Phase 5
- Impersonation (#59)
- Compliance toolkit (#61)
- Email provider presets (#52) + shark.email relay (#56)
- docs_url in error responses (#49)
- Migration tools — Auth0 (#55), Clerk (#63), Supabase (#64)

Phase 9 — Enterprise (P2 from AGENT_AUTH.md)

- Rich Authorization Requests (RFC 9396)
- Pushed Authorization Requests (RFC 9126)
- Step-up authorization flow
- Consent management UI (advanced)
- Agent analytics dashboard

Phase 10 — Moonshot (when ready)

- Pre-built UI components + dashboard editor (#68)
- Remaining cloud-only issues (#27, #29, #30, #31)  


---

The critical path: Phases 1-4 are launch. You can post on HN after Phase 4.
Phases 5-7 are what make Shark dominant. Phase 5 is what keeps users. Phase 7 is
what makes news.

Start Phase 1 tomorrow. Ship daily. Go.
