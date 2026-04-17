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

Phase 4 — Dashboard (7-10 days)

- Svelte admin dashboard (existing DASHBOARD.md spec)
- nextjs/react dashboard mimiccking svelte but for cloud *do not code this here, just keep a note*
- Depends on: stats endpoint, session endpoints from Phase 2  


This is what turns Shark from an API into a product. The HN demo GIF comes from
this.

Phase 5 — SDK (5-7 days)

- TypeScript SDK (#54)
- React/Svelte/Vue wrappers  


Now developers can actually integrate. Without this, Shark is unusable for most
people.




Phase 6 — Agent Auth (10-14 days)

- Full OAuth 2.1 server (#57) — client credentials, auth code + PKCE, device
  flow, token exchange, DCR
- Token Vault (#66) — managed third-party OAuth
- This is the headline feature. "First OSS auth with native MCP agent support."

Phase 8 — Proxy + OIDC Provider (5-7 days)

- Shark Proxy (#58) — shark proxy --upstream makes Shark usable without any SDK
- OIDC Provider mode (#60) — shares OAuth 2.1 infrastructure from Phase 7  


Phase 9 — Polish & enterprise (7-10 days)

- Impersonation (#59)
- Compliance toolkit (#61)
- Email provider presets (#52) + shark.email relay (#56) (overlap with Phase 2)
- docs_url in error responses (#49)
- Migration tools — Auth0 (#55), Clerk (#63), Supabase (#64)
- Pre-built UI components + dashboard editor (#68)  


Phase 10 — Moonshot (when ready)

- Visual flow builder (#62)
- Remaining cloud-only issues (#27, #29, #30, #31)  


---

The critical path: Phases 1-4 are launch. You can post on HN after Phase 4.
Phases 5-7 are what make Shark dominant. Phase 5 is what keeps users. Phase 7 is
what makes news.

Start Phase 1 tomorrow. Ship daily. Go.
