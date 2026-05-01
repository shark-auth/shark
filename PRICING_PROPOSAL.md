# SharkAuth — Product Definition & Agent-Centric Pricing Proposal

**Status:** DRAFT v1.0
**Date:** 2026-05-01
**Purpose:** Pricing framework for $1B revenue ambition. For review by agents, investors, and partners.

---

## What SharkAuth Is

SharkAuth is an open-source OAuth 2.1 authorization server built for the agentic era. It treats AI agents as first-class identities with native delegation primitives, proof-of-possession tokens, and a unified audit trail that tracks every hop from user to resource.

**The one-line pitch:**
> Auth for humans and the agents they ship. One platform. 40MB binary. Self-hosted free.

**The differentiated pitch:**
> SharkAuth ships the authorization primitives the OAuth standards already specified for agent delegation — and that no competitor has shipped as core infrastructure: RFC 8693 token exchange with nested act chains, RFC 9449 DPoP-bound tokens, five-layer cascade revocation, and a token vault where agents never see refresh tokens.

**What it is NOT:** A better Auth0. A human-auth bolt-on. A JWT middleware library. SharkAuth is purpose-built for products where agents act on behalf of users — with the audit trail, revocation blast radius, and credential vault that agents require.

**Who uses it:** AI coding assistants, customer support agents, sales/outbound AI, voice AI, workflow automation AI, custom-agent platforms, vertical AI (healthcare, legal, finance), browser-based agents. See `documentation/inner_docs/general_product_guide.md` §5 for the full 10-vertical target list.

**Current status:** v0.9.0 shipped 2026-05-01. MIT license. Self-hosted free today. Hosted cloud tier launches Q3 2026 after Postgres migration.

---

## The Agent Auth Moat — Five Layers

The competitive moat is not any single feature. It is the combination of five revocation blast-radius layers, the token vault, and the trust chain created by DPoP + act chains + grant correlation in one system.

### Layer 1 — Token (RFC 7009)

**Threat:** Agent's token leaks via prompt injection, log bleed, or network interception.
**Response:** Revoke token + refresh family immediately. Bearer tokens are useless without the DPoP private key.
**Economic value:** Token exfiltration yields nothing. No incident response required.

### Layer 2 — One Agent

**Threat:** One specific agent instance is compromised (RCE, key extraction).
**Response:** `agents.revoke_all(agent_id)` — kill every token that agent ever held.
**Economic value:** Surgical kill vs. manual rotation across every downstream service. Minutes vs. hours. $3K-$10K/hour IR cost reduction.

### Layer 3 — Customer Fleet

**Threat:** Customer churns, goes rogue, or a departing employee provisioned agents before exit.
**Response:** `users.revoke_agents(user_id)` — one call cascades to revoke every agent that customer ever spawned.
**Economic value:**
- Instant revocation: prevents unauthorized usage during churn disputes ($10K-$40K/month per enterprise contract)
- GDPR Article 17 compliance: failure to revoke within 72 hours = up to 4% of global annual turnover fine
- Rogue insider containment: average insider threat incident = $4.2M (Ponemon 2024)

### Layer 4 — Agent Type Pattern

**Threat:** Buggy agent template v3.2 deployed across 500 customers. Critical vulnerability.
**Response:** Bulk revoke by `client_id` pattern — single call kills every agent matching `*_v3.2*` across ALL customers.
**Economic value:**
- Without: 500 tenants × 5 min manual coordination = 41 hours during active exploit = $123K-$410K IR cost
- With: minutes = <$10K IR cost
- The IR cost gap per major incident: **$100K-$400K saved per vulnerability**

### Layer 5 — Vault Credential

**Threat:** Customer's external OAuth (Google, Slack) is compromised at the provider.
**Response:** `vault.disconnect(connection_id, cascade=true)` — disconnects the root credential and kills every derived agent token that ever used it.
**Economic value:**
- Google Drive breach average: $2.1M
- Third-party OAuth abuse without cascade: spreading credential compromise across every agent using that connection
- Vault disconnect contains the blast radius at the root

### The Token Vault

The vault is not a convenience feature. It is security infrastructure:
- Stores encrypted refresh tokens server-side. Agents never see refresh tokens.
- Agents get short-lived access tokens via `vault.fetch_token()`. The refresh token never leaves the vault.
- Server-side auto-refresh handles token rotation transparently.
- One encrypted credential server-side vs. 1,000 refresh tokens scattered across 1,000 agents.

**Build cost vs. buy:** To build a vault with equivalent security properties (AES-256-GCM, auto-rotation, DPoP integration, RFC 7009 cascade) costs $460K-920K one-time. SharkAuth includes it.

**Feature comparison:**

| Feature | SharkAuth | Auth0 | Clerk | Stytch |
|---|:---:|:---:|:---:|:---:|
| Broker third-party OAuth for agents | ✅ | ❌ | ❌ | ❌ |
| Agent never sees refresh token | ✅ | N/A | N/A | N/A |
| Auto-refresh server-side | ✅ | N/A | N/A | N/A |
| Cascade revoke (Layer 3) | ✅ | ❌ | ❌ | ❌ |
| Pattern bulk revoke (Layer 4) | ✅ | ❌ | ❌ | ❌ |
| Vault disconnect cascade (Layer 5) | ✅ | ❌ | ❌ | ❌ |
| DPoP-bound tokens | ✅ | ❌ | ❌ | ❌ |
| RFC 8693 act chain audit | ✅ | ❌ | ❌ | ❌ |
| Grant_id correlated audit | ✅ | ❌ | ❌ | ❌ |

---

## Why the Human-Auth Pricing Model Is Wrong

Every auth vendor prices human auth by MAU. Auth0, Clerk, Stytch, WorkOS — all MAU-based pricing.

**For human auth, MAU is correct:**
- One human ≈ one identity ≈ one risk unit
- Humans authenticate ~once per session
- The blast radius of a compromised human token is bounded by that human's permissions

**For agent auth, MAU is wrong:**
- One customer may have 500 agents acting on behalf of 50 users
- Each agent is a separate trust lineage and blast radius
- One compromised agent can access 50 users' worth of data
- One vault credential compromise cascades to ALL agents using that connection

**The unit in agent auth is not the human. It is the AGENT.**

---

## The Competitive Set

The correct competitive set for SharkAuth pricing is NOT Clerk at $25/seat. It is:

| Alternative | Cost | What you get | What you lose |
|---|---|---|---|
| Custom build (internal auth team) | $500K-2M one-time | Custom delegation infrastructure | 6-18 months of dev time, ongoing maintenance |
| Ory Permissions | $0.14/aDAU + $0.007/token | Zanzibar-style fine-grained permissions | No act chains, no vault, no cascade revoke |
| HashiCorp Vault | $0.47/resource/mo | Secrets management for agents | No OAuth delegation, no DPoP, no audit correlation |
| AWS Secrets Manager | $0.40/secret + $0.05/10K API calls | Ephemeral credentials for agents | No OAuth, no act chains, no delegation |
| Incident response (breach without cascade revoke) | $180K-500K per engagement | IR when something goes wrong | Prevention is not included |
| Data breach (credential sprawl) | $2-4M per event | — | Prevention costs more upfront |
| Auth0 AI Agents add-on | +50% on base price | Token vault (limited), some CIBA | No act chains, no cascade revoke, no self-host |

At these prices, **$25/agent/month for full five-layer revocation + DPoP + act chains + vault is a 10-50x discount** relative to the alternatives.

---

## Proposed Pricing Model

### Self-Hosted OSS — Free Forever

- Core OAuth 2.1, OIDC, DPoP, RFC 8693 token exchange
- Basic `may_act_grants`, Layer 1-3 revocation
- SQLite, single-tenant, community support
- **Purpose:** Developer adoption. Win the narrative. No friction.

---

### Cloud Agent Pro — Per Agent / Per Month

Pricing unit is **agents** (blast radius unit), not human MAU.

| Tier | Price | Agents | Vault Connections | Revocation Layers | Support |
|---|---|---|---|---|---|
| **Starter** | $25/agent/mo | Up to 10 | 3 included | Layers 1-3 | Email |
| **Growth** | $20/agent/mo | Up to 100 | 10 included | Layers 1-4 | Priority |
| **Scale** | $15/agent/mo | Unlimited | Unlimited | Layers 1-5 | SLA + Slack |

**Why per-agent decreases with volume?** Higher agent count customers have better security hygiene and lower per-agent breach risk. Volume discount reflects reduced risk surface per agent at scale. Also: the more agents, the more critical cascade revoke becomes — that's when Layer 3-4 earn their keep.

**Vault Connections (add-on):** $30/connection/month beyond the included count.
- Included in tier (3/10/unlimited)
- Each connection = one external OAuth integration (Google Workspace, Slack, GitHub, Linear, Jira, etc.)
- Infisical charges $18/identity for secrets management (lesser security properties)
- AWS Secrets Manager: $0.40/secret + per-API fees

**Example customer math:**

| Customer type | Agents | Vault connections | Monthly cost |
|---|---|---|---|
| Indie developer / small team | 5 | 2 | $125/mo |
| Startup shipping agent product | 25 | 5 | $625/mo |
| Mid-market with agent platform | 100 | 12 | $2,350/mo |
| Enterprise agent platform | 500 | 30 | $8,500/mo |
| Large enterprise | 2,000 | 100 | $32,500/mo |

**Revenue model at scale:**

| Tier | Avg ARR | Target customers | ARR |
|---|---|---|---|
| Starter (10 agents avg) | $3,000/yr | 5,000 | $15M |
| Growth (50 agents avg) | $12,000/yr | 2,000 | $24M |
| Scale (200 agents avg) | $36,000/yr | 500 | $18M |
| Enterprise | $100,000/yr | 200 | $20M |
| **Total addressable** | | | **$77M ARR** |

Path to $1B: vertical expansion into healthcare AI agents (HIPAA compliance requires this infrastructure), legal AI agents, financial AI agents — plus geographic expansion and enterprise seat expansions.

---

### Enterprise — Custom, $50K-500K/year

- Unlimited agents, unlimited connections
- Private deployment (their VPC, their cloud)
- Custom revocation patterns via API
- Dedicated customer success
- SLA (99.99% uptime)
- Compliance: SOC2, HIPAA, custom

**Revenue math:**
- 100 enterprise customers at $100K ACV = $10M ARR
- 1,000 enterprise customers at $100K ACV = $100M ARR

---

## Why $1B Is Achievable

**The category is nascent.** Every AI product that ships agents will need this infrastructure. The market is not saturated — it's early.

**The alternatives are either nonexistent, expensive, or both.** Custom builds cost $500K-2M. Auth0 charges 50% uplift with none of the features. Ory/Keycloak have no act chains. AWS Secrets Manager has no OAuth delegation.

**The five-layer revocation is a category-defining moat.** No one else has it. It took 6 months of shipped Go code to build. A competitor replicating it requires 6-12 months minimum, and SharkAuth won't be standing still.

**The token vault is a $460K-920K build sold as a $30/mo add-on.** Customers get enterprise-grade credential infrastructure at SMB prices.

**The unit economics improve with enterprise scale.** At 1,000 enterprise customers at $100K ACV, you're at $100M ARR with 60% gross margins (hosted Postgres, minimal infra). Add 10,000 mid-market at $12K ACV = additional $120M ARR.

**The path to $1B is not mass-market $19/mo pricing.** It's 20,000 mid-market teams at $1K-10K ACV + 1,000 enterprise customers at $100K-500K ACV. That's the Supabase model applied to agent auth.

---

## Immediate Actions

1. **Remove** "$19/mo vs Auth0 $2,500/mo" from all docs and landing copy
2. **Update** YC application pricing paragraph: "OSS free forever; hosted pricing based on per-agent-month + vault connections, starting $25/agent/mo"
3. **Keep** BILLING_UI=false until Stripe is wired and tier model is finalized
4. **Ship** Postgres support Q3 2026 to unblock cloud tier
5. **Define** exact vault connection pricing ($30/connection/mo is the working number)

---

## Open Questions

1. **Are you comfortable pricing per agent vs. per human MAU?** This is the foundational question. If yes, the $1B path is clear. If no, we need to discuss why.
2. **What is the minimum viable cloud tier for launch?** The minimal cloud offering needs: hosted SharkAuth on Postgres, custom domain, email support, 3 vault connections, Layers 1-3. What's the minimum price point for that?
3. **Enterprise pricing — will customers pay $100K/year?** The comparable is not Auth0's B2C pricing — it's the $500K custom build and $180K IR engagement. At those numbers, $100K/year is a 5-10x discount.
4. **Vertical focus: which agent domain first?** Healthcare AI agents have the strongest compliance pull (HIPAA requires exactly this audit trail). Legal AI agents are similarly compliance-driven. Financial AI agents (Hebbia, Bloomberg terminal integrations) have the highest ACV potential.

---

## Sources

- `documentation/inner_docs/general_product_guide.md` — product definition, 54-capability audit, competitive matrix
- `gstack/01-launch-design.md` — demand evidence, target user, narrowest wedge
- `gstack/04-autoplan-synthesis.md` — TD3 pricing decision (deferred)
- `gstack/05-revisions-cold-dm-kit.md` — locked decisions, compressed calendar
- `documentation/inner_docs/TIER_PAYWALL_DORMANT.md` — current billing backend status
- `documentation/sdk/vault.md` — token vault architecture
- `documentation/sdk/cookbook/multi-hop-delegation.md` — RFC 8693 act chains
- IBM/Ponemon 2024 Cost of Data Breach Report — breach cost benchmarks
- HashiCorp Vault pricing — secrets management comparable
- Ory Network pricing — permissions/authorization comparable
