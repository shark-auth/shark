# Shark Strategic Analysis — Differentiation & Monetization

**Date:** 2026-04-14
**Status:** Pre-launch strategy revision

---

## The Brutal Honest Assessment

### What We Have
A genuinely impressive Go auth backend — 10+ auth methods, solid security primitives, clean architecture. The "single binary, zero config" story is real and compelling.

### What's Wrong

**1. The Pricing Model Is Broken**

Current tiers ($19/$49/$149) have three fatal problems:

- **Too cheap to signal quality.** A $19/mo auth system reads as a toy. Auth is security-critical infrastructure — buyers expect to pay more, and cheap pricing makes them *nervous*, not excited. WorkOS gives basic auth away for free to 1M MAU because they know the money is elsewhere.

- **No B2B revenue lever.** SSO (SAML + OIDC) is built in — this is the feature WorkOS charges $125/connection/month for, Clerk charges $75/connection for, and Auth0 gates behind $240/mo minimum. We're giving it away at every tier. The "SSO tax" is the most hated thing in auth, but it works — it's where the money is.

- **MAU-based pricing on a single-binary product creates cognitive dissonance.** If someone can run it for free with unlimited users, why pay $19/mo for 50K MAU? "You pay for ops" is correct but weak — most teams that can run a single Go binary can do basic ops.

**2. "Ops convenience" is a weak cloud value prop.** Running a single Go binary with SQLite is trivially easy. docker run + a volume mount. Caddy/nginx in front, certbot for TLS. The operational burden we're selling against is nearly zero for our own product. Ironic: we made the self-hosted story *too good*.

**3. Missing the 2026 gold rush.** AI agent authentication (MCP, OAuth 2.1 for machines) is the fastest-growing segment. Auth0 launched "Auth0 for AI Agents," Okta announced "Agent Gateway." Nobody in OSS has nailed this yet. We have M2M API keys already — we're 60% of the way there.

**4. No organizations model.** The single biggest feature gap. Every B2B SaaS needs multi-tenancy for their *end users*. Without Organizations, Shark can't serve B2B SaaS companies — the highest-paying segment.

---

## The Differentiation Strategy

### Positioning: "The Auth System That Grows With You"

Stop competing with Auth0/Clerk on their terms (managed cloud, prebuilt UI). Own three positions nobody else holds simultaneously:

1. **The only auth system that runs from a 20MB binary to planet-scale cloud with zero config changes**
2. **The first OSS auth system with native AI agent/MCP authentication**
3. **Enterprise SSO without the enterprise tax**

---

## Feature Roadmap

### Tier 1: Ship-Now Differentiators (v0.1 - v0.3)

#### A. Organizations & Team Management
Multi-tenancy for your users' users. Every B2B SaaS needs this.
- `org_` prefixed IDs, org membership, org-level roles
- Org invitations (email + link), org switching
- Per-org SSO enforcement ("Acme Corp requires SAML")
- Org-level audit logs
- *Why it matters:* #1 reason teams stay on Clerk/Auth0. Without it, Shark is only usable for B2C.

#### B. Webhooks & Event System
- `POST` to customer URLs on auth events (user.created, session.started, mfa.enrolled, etc.)
- Retry with exponential backoff, delivery logs, signature verification
- SVix-style webhook management
- *Why it matters:* Makes auth "integrate-able." Without webhooks, every integration requires polling.

#### C. `shark` CLI Tool
- `shark init` — generates config + first admin key
- `shark serve` — runs the server
- `shark users list`, `shark users create`, `shark keys rotate`
- `shark migrate auth0 export.json` — import Auth0 users
- `shark health` — check deployment status
- *Why it matters:* DX is how OSS projects win. Ory's terrible CLI is one of its biggest complaints. Make `shark` the `kubectl` of auth.

### Tier 2: The 2026 Differentiators (v0.4 - v0.6)

#### D. AI Agent Authentication (MCP + OAuth 2.1)
Blue ocean feature. Nobody in OSS does this well.
- OAuth 2.1 server with PKCE (Shark as authorization server)
- MCP-compatible token endpoint for AI agents
- Agent identity tokens (distinct from human sessions)
- Scoped agent permissions ("this agent can read users but not delete")
- Agent session audit trail (which agent did what)
- Agent-to-agent delegation chains
- *Implementation:* Build on existing M2M API key system. Add OAuth 2.1 authorization code + client credentials flows. Add `agent_` prefixed identities.
- *Runs on SQLite WAL:* Token storage is just another table. Challenge state goes through existing cache system.

#### E. Edge-Ready Auth Verification
- Generate a signed, self-contained verification bundle that edge workers can validate without calling home
- `shark edge-bundle generate` — creates a signed JWKS + policy bundle
- Cloudflare Workers / Vercel Edge / Deno Deploy can verify sessions locally
- Periodic sync (every 60s) for revocation lists
- *Why it matters:* Cookie-based sessions are our strength, but require a round-trip to Shark for verification. Edge bundles solve this without going full JWT.

#### F. Shark Proxy (Sidecar Mode)
- `shark proxy --upstream http://localhost:3000`
- Sits in front of any app, handles all auth, injects `X-User-ID`, `X-User-Email`, `X-User-Roles` headers
- Zero-code auth for any backend (Go, Python, Node, Rails, PHP)
- Like Ory Oathkeeper but actually simple
- *Why it matters:* Makes Shark usable with *any* language without an SDK. Drop-in auth for existing apps.

#### G. Impersonation & Support Mode
- Admin can generate a time-limited impersonation session for any user
- All actions during impersonation flagged in audit log
- Auto-expires after configurable duration
- *Why it matters:* Every SaaS support team needs this. Auth0 charges enterprise tier for it.

### Tier 3: Killer Features (v0.7+)

#### H. Shark Connect (OAuth 2.1 / OIDC Provider Mode)
- Shark becomes a full OpenID Connect provider
- Your users' apps can "Sign in with [YourApp]" — like "Sign in with Google"
- Federation: Shark instances can trust each other
- *Why it matters:* Turns Shark from "auth for one app" into "identity provider for an ecosystem"

#### I. Compliance Toolkit
- GDPR data export (one-click user data package)
- Right to erasure (cascade delete with audit trail)
- SOC2 evidence collection (auto-generate access review reports from audit logs)
- Session geography tracking (which countries are sessions active from)
- *Runs on SQLite:* All metadata, no external services needed

#### J. Shark Studio (Visual Auth Flow Builder)
- Drag-and-drop auth flow customization in the dashboard
- "After signup -> require email verification -> show MFA enrollment -> redirect to onboarding"
- "If SSO domain match -> skip password -> route to SAML IdP"
- Export flows as YAML config
- *Why it matters:* Auth0's "Actions" and "Rules" are their stickiest features. This is the OSS equivalent.

---

## Revised Pricing Model

### The Problem With Current Model

Pricing cloud as a hosting service. But hosting a Go binary is trivial. Cloud must be priced as a **platform** that does things self-hosted can't easily do.

### New Model: "Free Core, Paid Platform"

**The Rule stays:** The binary is the product. Every auth feature ships in the binary.

**What changes:** Cloud sells *platform capabilities* — things that genuinely require managed infrastructure.

### Self-Hosted (Free Forever)
- Every auth feature, unlimited users
- SQLite or Postgres
- Svelte dashboard embedded
- CLI tools
- Community support (Discord + GitHub)
- **This never changes.**

### Cloud Tiers

| | **Starter** | **Pro** | **Business** | **Enterprise** |
|---|---|---|---|---|
| **Price** | **Free** | **$49/mo** | **$249/mo** | **Custom** |
| **MAU** | 5,000 | 50,000 | 500,000 | Unlimited |
| **Overage** | -- | $0.005/MAU | $0.003/MAU | Negotiated |
| **SSO Connections** | 1 OIDC | 3 (SAML+OIDC) | Unlimited | Unlimited |
| **Organizations** | 3 | 50 | Unlimited | Unlimited |
| **Webhooks** | 1 endpoint | 10 endpoints | Unlimited | Unlimited |
| **Audit Retention** | 7 days | 90 days | 1 year | Custom |
| **Agent/MCP Auth** | 5 agents | 100 agents | Unlimited | Unlimited |
| **Edge Bundles** | -- | 1 region | Multi-region | Global |
| **Support** | Community | Email (48h) | Priority (4h) + Slack | Dedicated + SLA |
| **Dashboard Seats** | 1 | 5 | 20 | Unlimited |
| **Compliance Reports** | -- | -- | SOC2 evidence | Full compliance |
| **Uptime SLA** | -- | 99.5% | 99.9% | 99.99% |
| **Impersonation** | -- | -- | Yes | Yes |
| **Custom Domain** | -- | Yes | Yes | Yes |

### Why This Works

1. **Free cloud tier is the funnel.** Most developers evaluate auth by signing up, not downloading a binary. 5K MAU costs nearly nothing (SQLite WAL per tenant on a $5 VPS handles thousands of free tenants) but captures the developer comparing against Clerk's free tier.

2. **$49 Pro is the sweet spot.** Cheaper than Auth0's cheapest paid ($240), cheaper than Clerk Pro ($25 + overages that balloon), expensive enough to signal quality. At 50K MAU this is 50-90% cheaper than every competitor.

3. **$249 Business captures B2B SaaS.** WorkOS charges $125/connection/mo for SSO. We offer unlimited SSO + Organizations + compliance for $249 flat. Absurdly good deal for any company with 10+ enterprise customers.

4. **Enterprise is where real money lives.** Custom pricing for companies processing millions of auth events. Auth0 makes their real money here ($30K+/yr).

5. **SSO connections and Organizations are natural upgrade triggers.** Startup starts with Pro, signs first enterprise customer who requires SAML -> needs Business tier. Organic upgrade, not artificial gate.

### Reconciling "Cloud Sells Ops, Not Features"

The philosophy evolves, the spirit stays:

- **Every auth feature** (password, passkeys, MFA, RBAC, SSO, magic links, M2M keys, audit logs) is free in self-hosted. Always.
- **Cloud platform features** (managed edge bundles, auto-scaling, multi-region, compliance reports, team dashboard seats, managed webhook delivery) genuinely require infrastructure — not arbitrary gates.
- **Cloud limits** (3 orgs, 1 SSO connection on free) are *resource limits*, not feature locks. Self-hosted has no limits.

A developer who self-hosts can set up their own webhooks (cron + curl), their own monitoring, their own SSO config. Cloud makes it push-button. That IS ops convenience — just more specific.

---

## The Funnel: GitHub Star to Paying Customer

```
AWARENESS
|  - GitHub README with killer demo GIF
|  - "shark vs auth0" / "shark vs clerk" comparison pages
|  - HN/Reddit launch post: "I built Auth0 in a single Go binary"
|  - "The Auth Pricing Calculator" -- interactive tool showing cost vs competitors
|  - "Add auth to your app in 5 minutes" tutorial series
|
ACTIVATION
|  - `curl -fsSL https://sharkauth.com/install | sh` (installs CLI)
|  - `shark init && shark serve` -- running in 10 seconds
|  - OR: Sign up at cloud.sharkauth.com (free tier, no credit card)
|  - Interactive getting-started wizard in dashboard
|  - First "hello world" auth in under 2 minutes
|
ENGAGEMENT
|  - TypeScript SDK: `npm i @sharkauth/js` + 3 lines of code
|  - Shark Proxy: zero-code auth for any backend
|  - Dashboard shows real-time auth events
|  - Weekly digest email: "147 signups, 3 SSO connections configured"
|
CONVERSION (Self-Hosted -> Cloud)
|  - Trigger: Team grows, ops becomes a headache
|  - Trigger: First enterprise customer asks for SSO + compliance
|  - Trigger: Need webhooks / multi-region / edge auth
|  - `shark cloud migrate` -- one command to move to cloud
|  - Data import is seamless (same schema, same API)
|
EXPANSION
|  - Starter -> Pro: Hit 5K MAU or need custom domain
|  - Pro -> Business: First enterprise SSO customer
|  - Business -> Enterprise: Compliance requirements / volume
|  - Add-on: Agent Auth pack, extra edge regions
```

---

## DX Improvements (Non-Negotiable)

1. **The 10-Second Start.** `shark init && shark serve` must produce a working auth system with a dashboard you can click through. Not "read this YAML file" — interactive init that asks 3 questions and generates everything.

2. **The 2-Minute Integration.** `npm i @sharkauth/js` + copy-paste a code snippet from the dashboard -> working auth in any frontend. Dashboard generates framework-specific code snippets (React, Svelte, Vue, vanilla JS).

3. **The Copy-Paste Dashboard.** Every entity (user ID, API key, role name) one-click copyable. Every API endpoint has a "Try it" button. Every config change shows equivalent YAML/CLI command.

4. **Error Messages That Teach.** Every error response includes a `docs_url` pointing to the relevant doc page. `{"error": "mfa_required", "message": "This account has MFA enabled", "docs_url": "https://sharkauth.com/docs/mfa"}`.

5. **Migration Tools.** `shark migrate auth0`, `shark migrate clerk`, `shark migrate supabase`. Import users with password hashes intact. Zero switching cost = market share theft.

6. **Local Dev Mode.** `shark serve --dev` disables email verification, prints magic links to stdout, auto-accepts OAuth callbacks, uses in-memory database. Never need a real email service during development.

---

## Value Propositions

**Indie devs / side projects:**
"Full Auth0 features in a 20MB binary. `shark serve` and you're done. Free forever."

**Startups:**
"Stop paying Clerk $1,000/mo for 100K users. Shark Cloud: $49/mo for the same thing, or $0 self-hosted."

**B2B SaaS:**
"Enterprise SSO for every customer without the $125/connection tax. Organizations, RBAC, audit logs — $249/mo flat."

**AI/agent builders:**
"The first auth system with native MCP agent authentication. Your AI agents get real identities, scoped permissions, and an audit trail."

**Platform teams:**
"One binary. SQLite to Postgres. Self-hosted to cloud. Same API, same config, same SDK. Grows with you from prototype to IPO."

---

## Competitive Matrix

| | Auth0 | Clerk | WorkOS | Supabase | Ory | **Shark** |
|---|---|---|---|---|---|---|
| Self-hosted | No | No | No | Yes | Yes | **Yes** |
| Single binary | -- | -- | -- | No (14 svc) | No (4 svc) | **Yes** |
| Free SSO | No ($240+) | No ($75/conn) | No ($125/conn) | No (Team) | Yes | **Yes** |
| Organizations | Yes | Yes | Yes | No | Yes | **Planned** |
| Agent/MCP Auth | Launching | No | No | No | No | **Planned** |
| Edge Auth | No | Yes | No | No | No | **Planned** |
| Auth Proxy/Sidecar | No | No | No | No | Oathkeeper | **Planned** |
| SQLite support | No | No | No | No | No | **Yes** |
| < 5KB client SDK | No | No | No | No | No | **Yes** |
| Zero-config start | No | No | No | No | No | **Yes** |
| OIDC Provider mode | Yes | No | No | No | Yes (Hydra) | **Planned** |
| Migration tools | -- | -- | -- | -- | -- | **Planned** |

---

## Priority Order

| Priority | Feature | Why | Effort |
|---|---|---|---|
| **NOW** | Dashboard + TS SDK | Can't launch without these | 2-3 weeks |
| **v0.2** | Organizations | Unlocks B2B market (highest-paying) | 1-2 weeks |
| **v0.2** | Webhooks | Required for any real integration | 3-5 days |
| **v0.3** | CLI tool (`shark init/serve/migrate`) | DX differentiator | 1 week |
| **v0.3** | Free cloud tier | Funnel entry point | 1-2 weeks |
| **v0.3** | Migration tools (Auth0, Clerk) | Steal market share | 1 week |
| **v0.4** | Agent/MCP Auth (OAuth 2.1 server) | Blue ocean, first-mover | 2-3 weeks |
| **v0.4** | Shark Proxy (sidecar mode) | Zero-code auth for any language | 1 week |
| **v0.5** | Edge auth bundles | Performance differentiator | 1-2 weeks |
| **v0.5** | OIDC Provider mode | Platform play | 2 weeks |
| **v0.6** | Compliance toolkit | Enterprise sales | 1-2 weeks |
| **v0.6** | Visual flow builder | Stickiness feature | 2-3 weeks |

---

## Cloud on SQLite WAL — It's a Feature, Not a Limitation

The constraint (SQLite WAL per tenant) is actually a selling point:

- **Physical tenant isolation.** Each tenant gets their own `.db` file. No `WHERE tenant_id = ?` on every query. No risk of cross-tenant data leaks. *Stronger* isolation than Postgres with row-level security.
- **Trivial backups.** `cp tenant_acme.db tenant_acme.db.backup` — it's a file. No pg_dump, no connection strings.
- **Instant provisioning.** New tenant = copy template database. Under 1ms.
- **Automatic resource isolation.** One tenant's heavy query doesn't block another.
- **Near-zero cost per tenant.** Idle SQLite file uses zero CPU, zero memory, negligible disk. 10,000 free-tier tenants on a $20/mo machine.

Only limitation: no cross-tenant queries (solved with a separate analytics store).

**Marketing angle:** "Every customer gets their own database. Not a row filter — a real, physical database. Your data never touches anyone else's." A security selling point no competitor matches at this price.

---

## TL;DR

1. **Add a free cloud tier** — it's the funnel, not self-hosted
2. **Raise prices** — $49/$249/Custom, not $19/$49/$149
3. **Build Organizations** — unlocks B2B market that actually pays
4. **Build Agent/MCP Auth** — first-mover in biggest auth trend of 2026
5. **Build Shark Proxy** — zero-code auth makes every backend dev a potential user
6. **Market SSO as "no enterprise tax"** — compelling reason to switch alone
7. **Lean into SQLite-per-tenant** as a security feature, not an architecture compromise
8. **Make DX religious** — 10-second start, 2-minute integration, errors that teach

The features are 80% built. The positioning and pricing need surgery.

---

*This document should drive roadmap decisions from v0.1 onward.*
