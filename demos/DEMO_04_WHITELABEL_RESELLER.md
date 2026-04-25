# DEMO 04 — White-Label Auth-as-a-Service Reseller
## "One Founder Sells Branded Auth to 100 Indie SaaS Creators"

**Duration:** 10 minutes live  
**Persona:** Mira — founder of LaunchPad, a no-code platform for solo SaaS makers  
**Theme:** One Shark binary powers 100 tenants, each with their own logo, palette, and auth flow. Zero per-tenant code.

---

## 1. Story — The Mira Persona

Mira built LaunchPad to help indie SaaS makers ship fast. She has 100 paying tenants: fintech apps, wellness trackers, dev tools, pet-tech, enterprise dashboards. Each tenant demands their own login page — their colors, their font, their logo. Today she has three bad options:

- **Auth0:** $240+/mo per-tenant, white-label costs extra (Enterprise tier). 100 tenants = $24,000+/mo in auth fees alone.
- **Clerk:** $20+/mo per app. No-brand removal is Pro ($25/mo). 100 apps × $25 = $2,500/mo + satellite domains at $10/mo each = $3,500+/mo with no reseller model.
- **SuperTokens self-hosted:** Free binary, but branding is per-tenant CSS customization — no design-token pipeline, no CLI extraction, manual every time.
- **FrontEgg:** Targets enterprise SaaS (not indie resellers), opaque pricing, requires their hosted cloud.
- **Build it yourself:** 6–18 months, $200k+ in eng cost.

**Shark gives Mira a fourth option:** one binary ($0 infra licensing), multi-tenant orgs, `shark branding set` per tenant, Flow Builder per tenant, design tokens auto-pushed from tenant's deployed frontend via `shark init` (partial — see Honest Status below). She charges tenants $49/mo for "Auth Included." At 100 tenants: **$4,900 MRR from auth alone** with near-zero marginal cost.

**Concrete numbers:**
- 100 tenants × $49/mo = $4,900 MRR
- Shark infra cost: ~$20/mo VPS (single binary, SQLite or Postgres)
- Auth profit margin: ~99%
- Setup per new tenant: ~3 minutes (`shark app create` + `shark branding set --from-file`)

---

## 2. Architecture

```
                        ┌─────────────────────────────────────┐
                        │         shark (single binary)        │
                        │                                      │
  :8080 ──── Admin UI  │  internal/admin/admin.go             │
  :8080 ──── API       │  internal/api/router.go              │
  :8080 ──── Hosted UI │  /hosted/{app_slug}/{page}           │
  :8081 ──── Proxy     │  internal/proxy/                     │
                        └─────────────────────────────────────┘
                                        │
                   ┌────────────────────┼────────────────────┐
                   │                    │                     │
            Org: pink-fintech    Org: green-wellness   Org: dark-devtool
            App slug: fintra     App slug: wellbeing    App slug: devhub
            Branding: #EC4899    Branding: #10B981      Branding: #1F2937
            Flow: signup→        Flow: signup→           Flow: signup→
              email_verify→        MFA→redirect            email_verify→
              MFA_enroll→                                   redirect
              paywall→redirect
```

**Design-token pipeline:**
1. `shark branding set <slug> --from-file tenant-tokens.json` → PATCH `/api/v1/admin/branding/design-tokens`
2. Tokens stored in `branding.design_tokens` (JSON blob, migration 00025)
3. `GET /hosted/{slug}/login` → `handleHostedPage` reads branding row, inlines CSS vars into HTML shell
4. SPA receives `window.__SHARK_HOSTED.branding` with `primary_color`, `secondary_color`, `font_family`, `logo_url`
5. Hosted page renders natively branded — no iframe, no "Powered by X" badge

**Custom domain routing (demo uses slugs; production path):**
- Today: `/hosted/{app_slug}/{page}` — slug-routed, per-app branding
- Production: custom domain → Shark proxy SNI routing → same hosted handler
- TLS: terminate at reverse proxy (Caddy/nginx) or future `--tls-cert` flag on Shark listener

---

## 3. Shark Feature Map

| Demo Step | Feature | File / Endpoint |
|-----------|---------|-----------------|
| Create tenant org | `POST /api/v1/admin/organizations` | `internal/api/admin_organization_handlers.go` |
| Create app per tenant | `shark app create --slug fintra` | `cmd/shark/cmd/app_create.go` |
| Push branding tokens | `shark branding set fintra --from-file tokens.json` → `PATCH /api/v1/admin/branding/design-tokens` | `cmd/shark/cmd/branding.go`, `internal/api/branding_handlers.go` |
| Serve branded login | `GET /hosted/{slug}/login` inlines CSS vars | `internal/api/hosted_handlers.go`, `admin/src/hosted/routes/login.tsx` |
| Hosted pages roster | login, signup, mfa, magic, passkey, verify, forgot_password, reset_password, error | `admin/src/hosted/routes/` |
| Flow Builder — create flow | `POST /api/v1/admin/flows` with step array | `internal/api/flow_handlers.go` |
| Flow steps (all wired) | require_email_verification, require_mfa_enrollment, require_mfa_challenge, redirect, webhook, set_metadata, assign_role, add_to_org, conditional | `internal/authflow/steps.go`, `internal/authflow/engine.go` |
| Conditional branches | DSL: `{"user_has_role": "pro"}` | `internal/authflow/conditions.go` |
| Webhook to billing | `webhook` step → POST to LaunchPad master | `internal/authflow/steps.go:executeWebhook` |
| Live branding edit | PATCH tokens → next page load reflects | `internal/api/branding_handlers.go` |
| Design token DB schema | `branding.design_tokens` JSON blob | `cmd/shark/migrations/00025_branding_design_tokens.sql` |
| Per-app branding override | `applications.branding_override` column | `cmd/shark/migrations/00017_branding_and_email_templates.sql` |
| Single binary serve | `shark serve` starts admin+API+hosted+proxy | `cmd/shark/cmd/serve.go` |

---

## 4. Demo Script (Exact CLI + Clicks)

### Setup (pre-demo, hidden)
```bash
shark serve  # single binary, already running on :8080/:8081

# 5 tenant orgs already created via:
shark app create --slug fintra    --name "FinTra"
shark app create --slug wellbeing --name "Wellbeing Hub"
shark app create --slug devhub    --name "DevHub"
shark app create --slug petpals   --name "PetPals"
shark app create --slug corpblue  --name "CorpBlue"

# Branding pushed from demos/whitelabel/tenants/*.json (see §6)
for f in demos/whitelabel/tenants/*.json; do
  slug=$(basename $f .json)
  shark branding set $slug --from-file $f
done
```

### Live Walkthrough (10 min)

**[0:00 – 1:00] The Problem**
- Mira shows her Slack: "Hey, can you make the login page green for my wellness app?"
- "Every week, 3 tenants ask for this. Auth0 charges Enterprise for white-label. Here's what I built."

**[1:00 – 2:00] The Binary**
```bash
$ ps aux | grep shark
# → one process: shark serve
$ curl localhost:8080/health
# → {"status":"ok","binary":"shark","version":"..."}
```
"One process. All 100 tenants. $20/mo VPS."

**[2:00 – 4:30] Five Distinct Branded Login Pages**

Open browser tabs (pre-loaded):
```
http://localhost:8080/hosted/fintra/login      # pink #EC4899, font: DM Sans
http://localhost:8080/hosted/wellbeing/login   # green #10B981, font: Inter
http://localhost:8080/hosted/devhub/login      # dark #1F2937, font: JetBrains Mono
http://localhost:8080/hosted/petpals/login     # yellow #F59E0B, font: Nunito
http://localhost:8080/hosted/corpblue/login    # corporate #1D4ED8, font: IBM Plex Sans
```
Flip through tabs. **Five completely different designs. Same binary. Same code. Zero per-tenant engineering.**

**[4:30 – 6:30] Flow Builder — TenantA (FinTra) Custom Signup Flow**

Open Admin → Flows → "New Flow" → trigger: signup

Drag and connect nodes in the UI (or show JSON):
```json
{
  "name": "FinTra Signup",
  "trigger": "signup",
  "steps": [
    {"type": "require_email_verification", "order": 1},
    {"type": "require_mfa_enrollment",     "order": 2},
    {"type": "webhook",                    "order": 3,
     "config": {"url": "http://localhost:3000/billing/new-user", "method": "POST"}},
    {"type": "set_metadata",               "order": 4,
     "config": {"key": "tier", "value": "free"}},
    {"type": "redirect",                   "order": 5,
     "config": {"url": "https://fintra.app/onboarding"}}
  ]
}
```
"Every new FinTra user: verify email → enroll MFA → fire billing webhook to LaunchPad → land on onboarding."

**[6:30 – 7:30] Same Flow, Different Brand — DevHub**

Copy flow, apply to devhub slug. Show `GET /hosted/devhub/login` — dark mode, mono font, identical flow logic.

"Reusing a flow across tenants is a copy + slug assignment. No code change."

**[7:30 – 9:00] Live Branding Edit — WoW Moment**

```bash
$ shark branding set wellbeing \
    --token colors.primary=#6D28D9 \
    --token typography.font_family=Georgia
# → design tokens updated
```
Refresh `http://localhost:8080/hosted/wellbeing/login` in <2 seconds.
**Purple. Georgia serif. Instant.** No redeploy. No cache bust. One CLI command.

**[9:00 – 10:00] The Pitch**

"Mira charges $49/mo per tenant. 100 tenants. $4,900 MRR. Her auth cost: $20/mo.  
Auth0 would charge her $24,000+/mo for the same. Clerk: $3,500+/mo.  
Shark: one binary, self-hosted, infinite margin."

---

## 5. Implementation Plan

### Files to create for demo

```
demos/whitelabel/
├── tenants/
│   ├── fintra.json        # {"colors":{"primary":"#EC4899"},"typography":{"font_family":"DM Sans"},"logo_url":"..."}
│   ├── wellbeing.json     # {"colors":{"primary":"#10B981"},"typography":{"font_family":"Inter"},...}
│   ├── devhub.json        # {"colors":{"primary":"#1F2937","secondary":"#F8FAFC"},"typography":{"font_family":"JetBrains Mono"},...}
│   ├── petpals.json       # {"colors":{"primary":"#F59E0B"},"typography":{"font_family":"Nunito"},...}
│   └── corpblue.json      # {"colors":{"primary":"#1D4ED8"},"typography":{"font_family":"IBM Plex Sans"},...}
├── flows/
│   ├── fintra-signup.json       # full flow JSON for demo
│   ├── devhub-signup.json       # simpler flow, dark-mode brand
│   └── wellbeing-login.json     # MFA-only, no paywall
├── launchpad-master/
│   └── server.ts          # tiny Express server listening for webhook POSTs, logs "New user: {email} on {tenant}"
└── setup.sh               # full demo setup script: app create + branding push + flow create
```

### Mock frontend dirs for `shark init` (future)
```
demos/whitelabel/mock-frontends/
├── fintra-frontend/        # index.html with pink tailwind classes, DM Sans google font link
├── wellbeing-frontend/     # green palette, Inter
└── devhub-frontend/        # dark bg, JetBrains Mono
```
These would be scraped by the future `shark init --from-url` CSS extraction feature.

---

## 6. Honest Status — What Works vs. What's Stubbed

| Feature | Status | Notes |
|---------|--------|-------|
| `shark app create` / slug routing | **WORKS** | `cmd/shark/cmd/app_create.go`, `internal/api/application_slug.go` |
| `POST /api/v1/admin/organizations` | **WORKS** | `internal/api/admin_organization_handlers.go` |
| `shark branding set --token` / `--from-file` | **WORKS** | `cmd/shark/cmd/branding.go` → `PATCH /api/v1/admin/branding/design-tokens` |
| Branding DB storage (`design_tokens` JSON) | **WORKS** | Migration 00025; `branding.design_tokens` column |
| `GET /hosted/{slug}/{page}` with inlined CSS | **WORKS** | `internal/api/hosted_handlers.go` inlines `primary_color`, `secondary_color`, `font_family`, `logo_url` |
| Hosted pages: login, signup, mfa, magic, passkey, verify, error | **WORKS** | `admin/src/hosted/routes/` |
| Flow engine: all 12 step types | **WORKS** | `internal/authflow/steps.go` + `engine.go` |
| Conditional branches (DSL) | **WORKS** | `internal/authflow/conditions.go` |
| Webhook step (billing hook) | **WORKS** | `executeWebhook` in steps.go |
| Admin Flow Builder UI (drag-drop) | **WORKS** (admin UI) | Admin dashboard flow editor |
| Live branding edit < 2s | **WORKS** | No cache layer; next request reads DB |
| Per-app branding override | **SCHEMA EXISTS** | `applications.branding_override` column; full per-app routing not yet wired in hosted handler — today branding is global, not per-slug |
| `shark init` palette/CSS extraction from frontend | **STUBBED** | `cmd/shark/cmd/init.go` only asks for base URL; CSS scraping, Tailwind/token extraction not implemented — demo uses manual `shark branding set --from-file` |
| Custom domain → per-tenant TLS routing | **NOT YET** | Slug routing works; custom domain SNI not wired; use nginx/Caddy reverse proxy for demo |
| Per-tenant branding (slug-scoped, not global) | **PARTIAL** | `branding.scope` column exists (`global`/org); slug-scoped branding reads need wiring in hosted handler |

**Demo workaround for per-slug branding:** In the demo, create 5 separate Shark instances on different ports, OR use the `branding_override` JSON on each application row to carry per-app tokens. The hosted handler reads `applications.branding_override` — confirm in `hosted_handlers.go` before demo day.

---

## 7. The Wow Moment

**5 visually distinct login pages — pink fintech, green wellness, dark dev tool, yellow pet-tech, corporate blue — all served by ONE binary, generated from 5 JSON files, with ZERO per-tenant engineering.**

Then: one CLI command changes WellBeing's color from green to purple. Refresh. Done. Under 2 seconds. No redeploy. No rebuild. No code.

This is the moment competitors cannot match:
- Auth0: white-label requires Enterprise contract negotiation
- Clerk: each app is a separate dashboard project; no reseller model; $10/mo per satellite domain
- SuperTokens: self-hosted but no design-token CLI pipeline; manual CSS per tenant
- FrontEgg: cloud-only, priced for enterprise, no indie reseller story

---

## 8. Sellable Angle

**One line:** "Shark is the only auth server that lets one developer sell fully white-labeled login pages to 100 tenants — one binary, one command per tenant, infinite margin."

**Three customer types:**

| Customer | Pain | Shark Unlock |
|----------|------|--------------|
| **Dev-tools agencies** (Vercel-for-X shops) | Each client demands their own branded auth. Today: fork, deploy, maintain separately. | `shark app create` + `shark branding set` = 3 min per client. One binary serves all. |
| **No-code platforms** (Lovable, v0, bolt.new clones) | Users ship apps that need auth. Platform can't ask users to integrate Auth0 themselves. | Shark embedded in platform: every generated app gets `/hosted/{slug}/login` auto-configured. |
| **Vertical-SaaS incubators** (YC for fitness apps, fintech accelerators) | Portfolio companies share infra but need distinct branding + custom auth flows (e.g., KYC step for fintech). | One Shark binary per cohort. Flow Builder lets incubator configure KYC/MFA per portfolio company without touching code. |

---

## 9. UAT Checklist (12 items)

- [ ] **U1** `shark app create --slug fintra` returns 201 with slug in response
- [ ] **U2** `shark branding set fintra --from-file demos/whitelabel/tenants/fintra.json` returns "design tokens updated"
- [ ] **U3** `GET /hosted/fintra/login` returns 200 HTML with `--color-primary: #EC4899` in `<style>`
- [ ] **U4** `GET /hosted/wellbeing/login` returns 200 HTML with `--color-primary: #10B981` (different from U3)
- [ ] **U5** All 5 tenant login pages load with visually distinct palettes (manual browser check)
- [ ] **U6** `POST /api/v1/admin/flows` with fintra-signup.json steps creates flow and returns 201
- [ ] **U7** Flow engine executes `require_email_verification` → blocks unverified user
- [ ] **U8** Flow engine executes `webhook` step → LaunchPad mock server receives POST within 2s
- [ ] **U9** `shark branding set wellbeing --token colors.primary=#6D28D9` updates color; next page load reflects new color
- [ ] **U10** `GET /hosted/devhub/login` renders dark background (`--color-secondary: #F8FAFC` text on `#1F2937` bg)
- [ ] **U11** `GET /hosted/fintra/signup` and `GET /hosted/fintra/mfa` load with same fintra branding (consistent across pages)
- [ ] **U12** LaunchPad `server.ts` webhook receiver logs `{"tenant":"fintra","email":"test@example.com","event":"signup"}` when flow fires
- [ ] **U13** (Bonus) Simultaneous requests to 5 different tenant slugs all return correct branding without cross-contamination
- [ ] **U14** (Bonus) `shark branding get fintra --json` returns current token blob including updated color from U9

---

## 10. Competitive Intelligence Summary

| Provider | White-Label? | Reseller Model? | Self-Host? | Price at 100 Tenants |
|----------|-------------|-----------------|------------|----------------------|
| **Auth0** | Enterprise only | No | No | ~$24,000+/mo |
| **Clerk** | Pro+ ($25/mo/app) | No | No | ~$2,500–$3,500+/mo |
| **SuperTokens** | Manual CSS | No | Yes (self-host) | Free binary + ops cost; no token pipeline |
| **FrontEgg** | Yes (enterprise pitch) | No | No | Opaque / enterprise |
| **Shark** | Native, CLI-driven | **YES** | **YES** | $0 licensing + $20/mo VPS |

---

*Plan only — no implementation in this file. See `demos/whitelabel/` for tenant JSON fixtures and setup scripts.*
