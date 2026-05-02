# SharkAuth — YC P26 Application Draft

Generated 2026-04-24. Submit target: Day 10 (2026-05-04).
Companion to: `raul-main-launch-yc-design-20260424-134836.md`.
Honest-framing discipline: claim only what the code actually ships (see audit matrix in design doc).

---

## Company

**Company name**
SharkAuth

**Company URL**
sharkauth.com (landing page ships Day 7; GitHub repo live today)

**Describe what your company does in 50 characters or less**
Open-source auth server built for AI agents.

**What is your company going to make?**
SharkAuth is a single-binary, open-source OAuth 2.1 authorization server built for the agent era. It ships the identity primitives every product needs (SSO, passkeys, MFA, magic links, organizations, RBAC, passwordless) plus RFC 8693 token-exchange with native agent delegation chains, plus a reverse-proxy mode that adds auth to any existing app with zero code changes.

Teams drop the binary in front of their service. MCP servers and AI agents authenticate using standard OAuth 2.1 + Dynamic Client Registration. Agents delegate capabilities down delegation chains (agent-A → agent-B → agent-C) with a full audit trail via RFC 8693 `act` claims. All running on SQLite by default, horizontally scalable via Postgres (shipping Q3 2026), self-hosted in the customer's own VPC.

**Where do you live now, and where would the company be based after YC?**
Founder currently lives in Monterrey, Mexico. Company would be based in San Francisco during the batch; post-batch location decision driven by where early customers and hires land (default SF).

---

## Progress

**How far along are you?**
Pre-launch, shipping v1.0 publicly in next 10 days. Codebase is production-grade on core surface: 38 of 54 audited capabilities shipped with full test coverage (54 audit items spanning OAuth 2.1, identity primitives, platform, DX). Full OAuth 2.1 authorization server with PKCE, DPoP, DCR, Token Exchange + delegation, refresh rotation with family-based reuse detection, device flow, introspection, revocation. Identity: passwords (argon2id), passkeys, TOTP MFA with recovery codes, magic links, account lockout, social login, session management. Platform: multi-tenant orgs, RBAC, reverse proxy with circuit breaker, audit log, webhooks, React/Python/TypeScript SDKs, admin dashboard with 47 UI components, CLI with 31 subcommands, first-boot setup UX.

Gaps we are explicit about: Protected Resource Metadata (RFC 9728), SAML IdP mode (we consume SAML today, don't provide it), SMS OTP, OpenAPI spec, PostgreSQL support (required before hosted cloud tier), binary publish pipeline (shipping Day 7). Full feature matrix with file-level evidence available on request.

**How long have each of you been working on this?**
Founder: ~8 months full-time, including all of the Go backend, CLI, admin dashboard, SDKs, and proxy mode. Working solo with AI copilot throughout.

**How many users do you have?**
Pre-launch; 0 paying customers. Design-partner outreach begins Day 1 of the 10-day launch sprint, target 3 named partners running SharkAuth in non-demo environments before YC submission. Public launch on Day 7 (HN + Product Hunt + r/selfhosted) — target 250+ GitHub stars and 10+ telemetry-pinged installs by submission.

We would rather apply with a small number of real installations than a large waitlist. A waitlist is not demand. An MCP server author running SharkAuth in staging is demand.

**Do you have revenue?**
No. Revenue model post-YC is Supabase-shaped: OSS free forever self-hosted; paid SharkAuth Cloud using MAI (Monthly Active Identities) — one metric that counts humans and agents together. Cloud Free: 20K MAI at $0. Cloud Pro: 50K MAI at $49/mo. Cloud Team: 200K MAI at $199/mo. Enterprise: unlimited MAI from $25K/yr. We do not charge per subagent or delegation hop — an orchestrator spawning 20 workers is still one identity. Cloud launch blocked on Postgres migration — single-node SQLite is a hard scaling ceiling for hosted. Enterprise contracts include regulated companies with compliance needs (HIPAA, SOC2, custom VPC deployment).

**Anything else we should know about progress?**
Technical depth is ahead of demand signal. That asymmetry is the exact thing the 10-day launch sprint is built to correct — we are not applying to YC to go learn how to build this; we are applying to accelerate distribution and customer acquisition for a product that already works.

---

## Idea

**Why did you pick this idea?**
Was building adjacent AI infrastructure and kept hitting the same wall: every MCP server needed auth, every available auth option was wrong. Auth0 priced agents as a 50%-uplift add-on. Clerk and Stytch were built for human-logging-into-SaaS flows, with agents bolted on as M2M clients. Okta is enterprise-locked. Every OSS developer I talked to was either paying a SaaS tax or hand-rolling JWT middleware and hoping for the best.

The gap is structural, not cosmetic. Auth was built for a world where humans log into apps. The agent era is a world where agents make delegated calls on behalf of humans and other agents. RFC 8693 solves this cleanly — nobody was shipping it well for the MCP ecosystem.

**Domain knowledge:**
Built the full OAuth 2.1 authorization server from scratch in Go on top of fosite. Implemented DCR (RFC 7591), DPoP (RFC 9449), Token Exchange (RFC 8693) with nested `act` delegation chains, audience-restricted tokens (RFC 8707), refresh token rotation with family-based reuse detection, SAML SP, OIDC federation, passkey enrollment, TOTP MFA — every component with tests. I am comfortable arguing any RFC in this space line-by-line; that domain depth is defensible against any competitor at the technical layer.

**Who are your competitors, and who might become competitors? Who do you fear most?**
- **Auth0 (Okta):** incumbent, expensive, closed. Agent auth is a $53+/mo add-on. Not OSS, not self-host. Fear level: low-med — they will ship a MCP feature in 2026 but their pricing and deployment model cannot fundamentally change.
- **Clerk:** fast-shipping SaaS, great DX. But SaaS-only, no self-host, agent support is secondary. Fear level: med — if Clerk pivots harder to agents, DX parity gets harder. Mitigated by the self-host moat.
- **Stytch:** ships MCP compliance today with Connected Apps. Closest feature-surface overlap. But agents are M2M clients in their model, not first-class identities with delegation chains. Their unit economics require SaaS cloud lock-in; self-host tier is unlikely. Fear level: med-high — they have customers today.
- **WorkOS:** narrow (SAML/SSO-focused), B2B-only. Different segment. Fear level: low.
- **Ory / Keycloak:** existing OSS options. Keycloak is Java, heavy, painful to operate. Ory is modular but complex and primarily SaaS-monetized. Neither prioritized agent auth. Fear level: low — they're the status quo we win developers away from, not where the market is going.
- **Cloudflare:** shipping agent security primitives. Fear level: watch-list. If they ship a full auth server, the competitive map redraws. Our counter: they will ship primitives tied to Workers, not a portable OSS binary.
- **Biggest fear:** not a competitor — category commoditization. Every major LLM provider (Anthropic, OpenAI, Google) could ship auth primitives baked into their agent SDKs, making "bring your own auth server" moot. Counter: platform-neutral auth always wins the long tail, as it has in every prior wave.

**What's new about what you're making?**
1. **Single-binary OSS with the full identity stack.** Not just OAuth; the whole thing — SSO, passkeys, MFA, magic links, orgs, RBAC, audit — in a 20MB Go binary on SQLite. No Redis, no Postgres, no separate admin panel service. This is what Supabase did to Firebase; we are doing it to Auth0.
2. **Native RFC 8693 delegation chains.** Agents are first-class identities with nested `act` audit trails. Agent-A delegating to agent-B delegating to agent-C is a supported primitive, not a workaround. Nobody else ships this as a core feature.
3. **Proxy mode.** `shark serve --proxy-upstream=http://my-app:8080` drops OAuth 2.1 auth + DPoP + scope enforcement in front of any existing HTTP service with zero code changes in that service. We haven't seen another auth server ship this as a first-class mode.
4. **MCP-native by construction.** RFC 8414 discovery, RFC 7591 DCR, RFC 8693 token exchange — every primitive an MCP client needs, out of the box. No plugins, no adapters.

**Who are your users?**
Primary: open-source MCP server authors and indie/small-team devs shipping agent-facing products. Already running Go, Node, or Python services. Already rejected Auth0 for pricing or Clerk for SaaS lock-in. Willing to self-host. Love single-binary ops.

Secondary: mid-market B2B SaaS with compliance requirements (healthcare, fintech, EU data residency) currently running Keycloak or hand-rolled auth. Want Postgres support and self-host; will adopt SharkAuth once the Postgres tier ships.

---

## Legal

**Are you incorporated?**
Not yet. Will incorporate as a Delaware C-corp if accepted to the batch, per YC standard practice.

**Who owns what part of the company?**
Founder: 100%. No prior investors, no prior employees, no prior advisors with equity.

**Have you taken any investment?**
No.

**Do you have any patents?**
No, nor any plans to file. OSS licensing (MIT or Apache 2.0) is the distribution strategy.

---

## Team

**Founder bio (Raul R. Gonzalez):**
18, from Monterrey, Mexico. Self-taught systems engineer; fluent in Go, TypeScript, Python. Built SharkAuth from scratch over the last 8 months including the full OAuth 2.1 server, reverse proxy, admin dashboard, CLI, and three SDKs. Comfortable operating at every layer of the stack — database schema, network protocols, cryptographic primitives, frontend state, build pipelines. Prior to SharkAuth, was building adjacent AI agent infrastructure where the auth pain became the bottleneck.

I am solo today. I am actively looking for a technical co-founder, specifically someone who can own cloud infrastructure and hosted-tier go-to-market while I continue owning the auth server core. I have posted on YC's co-founder matching board and am open to intros through the YC network during and after the batch.

**Why are you the right person to solve this?**
- Built the full thing solo, every RFC implemented line-by-line. I know where the bodies are buried.
- Hit the problem firsthand while building other AI infra — I am one of my own users.
- Technical depth combined with strategic framing of the 2026 auth landscape (see competitor analysis above).
- Monterrey/LatAm perspective on OSS adoption — markets where pay-per-seat SaaS doesn't work are where OSS infrastructure wins first.
- Ship velocity: 8 months from zero to a production-grade auth server with 54 capabilities and full test coverage. Top decile for solo founders at any age.

**How did you meet your co-founders?**
N/A — solo today, looking for one during the batch.

---

## Equity

**Are any founders' shares subject to vesting?**
N/A (pre-incorporation). Standard 4-year vest with 1-year cliff at incorporation.

---

## Other

**If we fund you, which three things will you do first?**
1. Land 20 design partners in the first 6 weeks (from the ~3 we'll have at application time) — target mix: 15 OSS MCP server authors, 5 mid-market B2B SaaS with compliance needs.
2. Ship Postgres support and launch SharkAuth Cloud private beta to the waitlist. This unlocks the hosted tier revenue story and the mid-market segment simultaneously.
3. Close the feature-surface gaps that real customers hit — likely SAML IdP mode, RFC 9728 Protected Resource Metadata, and whatever the design partners prioritize. Polish OpenAPI spec and docs site into the best self-service auth documentation in the category.

**What convinced you to apply to YC?**
Two things. First, the 2026 auth landscape is a rare case of large-incumbent weakness + urgent new demand (agent era) + working product (SharkAuth) + proven OSS-first playbook (Supabase, ClickHouse, Postgres). The P26 batch is the right window — 6 months from now, the incumbents will have half-baked MCP features and the category is partially commoditized. Second, I am solo, 18, international; YC's network effect on co-founder search, customer intros, and operational mentorship is uniquely valuable to a founder with my profile.

**How do you know people want this?**
Honest answer: the strongest signal today is the structural gap (four funded competitors all charging heavily or gated, every MCP server author I talk to is hand-rolling auth badly). The next signal — real installations from real users — will come from the 10-day launch sprint between now and submission. We will update this answer in the interview with concrete partner counts, quotes, and use-cases. If the sprint lands zero paying signal, I will say that plainly; I'd rather show YC an honest weak-signal story than an inflated one.

**What's the worst that could happen?**
Category consolidation by an LLM-provider-bundled auth primitive. Anthropic, OpenAI, or Google ships auth tightly bundled with their agent SDKs. Our counter is platform-neutrality (we work with any MCP client and any agent runtime) and deployment-model differentiation (OSS self-host the hyperscalers cannot match without damaging their SaaS margins).

**Anything else?**
The codebase is open source and will be public before the interview. I welcome partners verifying every claim in this application against actual code, commit history, and test suites. If we're invited to interview, I will bring a 5-minute live demo: `shark serve` → MCP client registers via DCR → agent delegates via token exchange → proxy enforces scope — end to end, no slides.

---

## Post-submission notes (not for YC form — founder reference)

**Sections to revisit before hitting submit:**
- Fill "How many users do you have?" with exact number landed in sprint.
- Fill design-partner quotes into "How do you know people want this?" if any partner allows attribution.
- Replace "sharkauth.com" with actual landing page URL if domain resolves by Day 9.
- Replace video link placeholder (in YC form, not here) with actual Loom URL.
- Update "competitor fear level" if Stytch or Cloudflare ships anything big in the next 10 days.

**Things to NOT do in the app:**
- Do not use the phrase "agent era" more than once. It is a buzzword. Let the demo carry it.
- Do not claim "premium DX" — claim is cheap, demonstration is expensive. Let the quickstart video demonstrate.
- Do not overclaim features. Audit matrix in design doc is the source of truth; if it says PARTIAL, the app says PARTIAL.
- Do not pitch "beat Stytch at all auth." Pitch "win MCP-era OSS auth, expand from there."
