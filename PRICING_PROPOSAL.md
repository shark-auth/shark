# SharkAuth — Product Definition & Pricing Proposal

**Status:** REVISED v3.0
**Date:** 2026-05-01
**Revision note:** Modelo unificado MAI (Monthly Active Identities). Humanos y agentes son la misma unidad de medida. No se cobra por subagentes ni hops de delegación.

---

## What SharkAuth Is

SharkAuth is an open-source OAuth 2.1 authorization server built for the agentic era. It treats AI agents as first-class identities with native delegation primitives, proof-of-possession tokens, and a unified audit trail that tracks every hop from user to resource.

**The pitch:**
> Auth for products where agents act on behalf of users.

**The one-line:**
> Auth for humans and the agents they ship. One platform. 40MB binary. Self-hosted free.

**What it is NOT:** A better Auth0. A human-auth bolt-on. A JWT middleware library. SharkAuth is purpose-built for products where agents act on behalf of users — with the audit trail, revocation blast radius, and credential vault that agents require.

**Who uses it:** AI coding assistants, customer support agents, sales/outbound AI, voice AI, workflow automation AI, custom-agent platforms, vertical AI (healthcare, legal, finance), browser-based agents.

**Current status:** v0.9.0 shipped 2026-05-01. MIT license. Self-hosted free. Hosted cloud tier launches Q3 2026 after Postgres migration.

---

## The Agent Auth Moat — Five Layers

The competitive moat is the combination of five revocation blast-radius layers, the token vault, and the trust chain created by DPoP + act chains + grant correlation in one system.

### Layer 1 — Token (RFC 7009)

**Threat:** Agent's token leaks via prompt injection, log bleed, or network interception.
**Response:** Revoke token + refresh family immediately. DPoP binding substantially reduces replay value — a stolen token is useless without the private key.
**Economic value:** Surgical revocation vs. waiting for token expiry. No manual rotation across downstream services.

### Layer 2 — One Agent

**Threat:** One specific agent instance is compromised (RCE, key extraction).
**Response:** `agents.revoke_all(agent_id)` — kill every token that agent ever held.
**Economic value:** Minutes vs. hours of manual IR work. $3K-$10K/hour IR cost reduction.

### Layer 3 — Customer Fleet

**Threat:** Customer churns, goes rogue, or a departing employee provisioned agents before exit.
**Response:** `users.revoke_agents(user_id)` — one call cascades to revoke every agent that customer ever spawned.
**Economic value:**
- Instant revocation prevents unauthorized usage during churn disputes
- Rogue insider containment without manual deprovisioning across every agent

### Layer 4 — Agent Type Pattern

**Threat:** Buggy agent template v3.2 deployed across 500 customers. Critical vulnerability.
**Response:** Bulk revoke by `client_id` pattern — single call kills every agent matching `*_v3.2*` across ALL customers.
**Economic value:**
- Without: 500 tenants × 5 min manual coordination = 41 hours during active exploit
- With: minutes

### Layer 5 — Vault Credential

**Threat:** Customer's external OAuth (Google, Slack) is compromised at the provider.
**Response:** `vault.disconnect(connection_id, cascade=true)` — disconnects the root credential and kills every derived agent token.
**Economic value:** Contains blast radius at the root credential. Without this, a third-party OAuth compromise spreads across every agent that ever used that connection.

### The Token Vault

The vault is security infrastructure, not a convenience feature:
- Stores encrypted refresh tokens server-side. Agents never see refresh tokens.
- Server-side auto-refresh handles token rotation transparently.
- One encrypted credential server-side vs. N refresh tokens scattered across N agents.
- Replaces months of custom OAuth token lifecycle and security engineering work.

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
| Self-hostable, open-source | ✅ | ❌ | ❌ | ❌ |

---

## Competitive Reality

The category is not empty. This must inform how we position and price.

**Auth0** is pushing "Auth0 for AI Agents" with Token Vault, CIBA, and async authorization. They charge +50% on base price for the AI Agents add-on. They have a token vault (limited), some CIBA, and are connecting agents to Google/Slack/GitHub. This validates the category exists. It also means we cannot say "nobody sees this" — we can say: "SharkAuth is the open-source/self-hosted agent-auth stack with RFC-native delegation and revocation primitives."

**Stytch** explicitly includes "AI agents" in their free tier at 10,000 MAU. This validates that agents belong in the auth pricing conversation. It also means per-agent pricing as a headline is harder to sustain when Stytch is effectively giving agent auth away in their free tier.

**WorkOS** has AuthKit free up to 1M MAU, then $2,500/mo per additional million. Clerk has a generous free tier and Pro at $20/mo. These anchor expectations low for self-serve pricing.

**Ory** and **HashiCorp Vault** support the infra/security-style pricing model (per-resource, per-check, usage-based). This is the most credible analog for vault and permission-layer pricing — and supports usage-based overage rather than pure per-agent flat pricing.

**The correct competitive set for SharkAuth is therefore:**
- Custom builds ($500K-2M one-time, 6-18 months)
- Ory Permissions ($0.14/aDAU + $0.007/token) for fine-grained permissions without delegation
- AWS Secrets Manager ($0.40/secret + $0.05/10K API calls) for vault-lite without OAuth
- Auth0 AI Agents add-on (+50% on base) for cloud-only agent vault
- Self-hosted Keycloak (Java, ~1GB, significant ops burden) as the self-hosted alternative with no agent-native primitives

---

## Why MAI (Monthly Active Identities)

Every auth vendor prices human auth by MAU. For human auth, MAU is correct: one human ≈ one risk unit.

For agent auth, MAU is incomplete:
- One customer may have 500 agents acting for 50 users
- Each agent is a separate trust lineage and blast radius
- One compromised vault credential cascades to ALL agents using it

**The unit in agent auth is the IDENTITY — human or agent.** A developer with SSH access, a support agent bot, and a data-pipeline worker are all identities that need the same auth primitives: registration, keys, tokens, revocation, audit.

**MAI counts them as one metric.** No distinction between "human MAU" and "agent count." No double-counting. No surprise bills because your orchestator spawned 20 sub-tasks.

**Subagents and delegation hops do NOT count as additional MAI.** A token-exchange chain `user → orchestrator → worker` is a delegation feature, not three billable identities. The orchestrator is one MAI. The worker is one MAI if it has its own client_id and DPoP keys. The intermediate token is just a hop.

---

## Pricing for Launch

**Principle:** Maximize adoption and conversation. Do not extract value before trust is established. In auth and security, people pay for confidence — not just features.

**Core thesis:** MAI (Monthly Active Identities) replaces MAU. One metric for humans and agents. Delegation chains are unlimited features; only persistent identities count toward the limit.

### Public Launch Pricing

| Plan | Price | MAI (Monthly Active Identities) | Delegation Depth | Vault Connections | Support |
|---|---|---|---|---|---|
| **Self-hosted OSS** | Free forever | Unlimited | Unlimited | Unlimited | Community |
| **Cloud Free** | $0/mo | Up to 20,000 | Max depth 2 | 3 | Community |
| **Cloud Pro** | $49/mo | Up to 50,000 | Max depth 4 | 10 | Priority email |
| **Cloud Team** | $199/mo | Up to 200,000 | Max depth 7 | 25 | Priority + Slack |
| **Enterprise** | Custom, from $25K/yr | Unlimited | Unlimited | Unlimited | SLA + dedicated CS |

**What counts as 1 MAI:** Any identity that authenticates within the billing window — human user login, agent client_credentials call, device flow activation. DPoP key registration counts as identity creation; subsequent token exchanges on the same client_id do not.

**What does NOT count as additional MAI:**
- Delegation hops in an `act_chain` (intermediate tokens)
- Subagents created transiently by an orchestrator (no persistent client_id)
- Token refresh on an existing grant
- Introspection calls

**Delegation depth limit (Free tier):** Max 2 hops (user → agent). Pro unlocks depth 4 (user → agent → sub-agent → worker). Team unlocks depth 7. Enterprise unlimited. This is the soft gate on the free tier — generous enough for demos, meaningful enough to upgrade for complex multi-agent flows.

### Overage Pricing

| Plan | MAI overage | Vault connection overage |
|---|---|---|
| **Cloud Free** | Hard cap at 20,000 MAI | Hard cap at 3 |
| **Cloud Pro** | $2 per 1,000 MAI/mo above 50K | $10 per connection/mo above 10 |
| **Cloud Team** | $1 per 1,000 MAI/mo above 200K | $5 per connection/mo above 25 |
| **Enterprise** | Custom | Custom |

**Example customer math:**

| Customer type | Humans | Agents | Total MAI | Plan | Monthly cost |
|---|---|---|---|---|---|
| Indie dev / small team | 3 | 2 | 5 | Cloud Free | $0 |
| Startup shipping agent product | 15 | 35 | 50 | Cloud Pro | $49/mo |
| Mid-market with agent platform | 80 | 420 | 500 | Cloud Pro + overage | $49 + $0 = $49 |
| Growing platform | 200 | 1,800 | 2,000 | Cloud Pro + overage | $49 + $0 = $49 |
| Large platform | 1,000 | 9,000 | 10,000 | Cloud Pro + overage | $49 + $0 = $49 |
| Enterprise agent fleet | 5,000 | 45,000 | 50,000 | Cloud Team | $199/mo |
| Large enterprise | 20,000 | 180,000 | 200,000 | Cloud Team + overage | $199 + $0 = $199 |
| Hyperscale | 50,000 | 450,000 | 500,000 | Enterprise | Custom |

*Note: All examples under 50K MAI fit in Pro ($49/mo). Team ($199/mo) covers up to 200K MAI. This is intentionally generous — the goal is adoption, not squeezing every dollar at low scale.*

### Enterprise Pricing

- Unlimited MAI, unlimited delegation depth
- Private deployment (their VPC, their cloud)
- Custom revocation patterns via API
- Dedicated customer success
- SLA (99.99% uptime)
- Compliance: SOC2, HIPAA, custom
- Starts at $25K/year

**Why $25K/yr minimum:** The comparable is not Auth0's B2C pricing — it is the $500K custom build and $180K IR engagement. At those numbers, $25K/year is a discount. For regulated companies (healthcare, legal, finance), this is a rounding error compared to compliance costs.

---

## The YC / Investor Framing

For YC and investor conversations:

> SharkAuth is open-source and free to self-host. The hosted product monetizes via Cloud plans starting at $49/mo for startups and $199/mo for teams, with enterprise contracts starting at $25K/year. The pricing unit is MAI — Monthly Active Identities — which counts humans and agents as a single metric. No distinction, no double-counting, no surprise bills when an orchestrator spawns sub-tasks.

> The open-source/self-hosted tier wins the developer narrative. The cloud tier monetizes teams that do not want to operate infrastructure. Enterprise monetizes regulated companies that need SLA, compliance, and private deployment.

> The category is validated: Auth0 charges +50% for their AI Agents add-on. Stytch includes agents in their free tier. WorkOS bundles agent auth into AuthKit. SharkAuth's differentiation is RFC-native delegation chains (act chains), five-layer cascade revocation, and a token vault where agents never see refresh tokens — all in a single self-hostable binary.

---

## What to Avoid Saying

| Original claim | Problem | Replace with |
|---|---|---|
| "No competitor has shipped this" | Auth0 has shipped AI agent messaging + token vault | "No open-source or self-hosted auth server packages these as core infrastructure" |
| "GDPR Article 17 failure = 4% global turnover fine" | Legally sloppy; ties our feature to a specific regulatory outcome | Remove or soften to: "regulatory compliance for delegated access" |
| "Token exfiltration yields nothing" | Too absolute; if key is also compromised, risk remains | "DPoP substantially reduces the replay value of stolen tokens" |
| "Vault is a $460K-920K build" | Sounds made up without a defensible model | "Replaces months of custom OAuth token lifecycle and security engineering" |
| "$1B revenue ambition" | Investor-brained for a public pricing doc | Remove from public-facing doc; keep internal |
| "$25/agent/mo as public headline" | Scares early devs; Stytch gives agent auth in free tier | MAI-based pricing: $0 for 20K MAI, $49 for 50K |
| "We charge per subagent / delegated actor" | Contradicts MAI thesis; subagents are feature, not unit | "Delegation chains are unlimited; only persistent identities count toward MAI" |

---

## What to Put on the Launch Page

```
## Pricing

SharkAuth is open-source and free to self-host.

Cloud is coming soon — join the waitlist.

### Self-hosted
$0 forever
- MIT licensed
- OAuth 2.1 / OIDC
- RFC 8693 token exchange
- RFC 9449 DPoP
- Agent delegation chains (unlimited depth)
- Token vault
- SQLite single-binary deploy

### Cloud Free
$0/mo
- For demos and experiments
- 20,000 MAI
- Delegation depth: 2 hops
- 3 vault connections
- Community support

### Cloud Pro
$49/mo
- For startups shipping real agentic apps
- 50,000 MAI
- Delegation depth: 4 hops
- 10 vault connections
- Priority email support

### Cloud Team
$199/mo
- For production teams
- 200,000 MAI
- Delegation depth: 7 hops
- 25 vault connections
- Advanced audit export
- Priority support

### Enterprise
Custom, starts at $25K/yr
- Unlimited MAI
- Unlimited delegation depth
- Private deployment / VPC
- SSO / SAML
- SLA
- Compliance support
- Custom revocation policies
```

**MAI = Monthly Active Identities.** One human login and one agent client_credentials call are the same unit. An orchestrator spawning 20 sub-tasks is still one identity. You only count persistent identities with their own keys — not delegation hops.

---

## Immediate Actions

1. **Remove** all references to "$19/mo" and "$29/mo" from docs and landing copy
2. **Remove** "$1B revenue ambition" from public-facing documents
3. **Adopt** MAI pricing model: Cloud Free 20K MAI → Pro 50K MAI $49 → Team 200K MAI $199 → Enterprise from $25K/yr
4. **Update** all pricing tables to show MAI + delegation depth + vault connections
5. **Update** the YC application pricing paragraph to reflect MAI model
6. **Keep** BILLING_UI=false until Stripe is wired and cloud tier is ready
7. **Ship** Postgres support Q3 2026 to unblock cloud tier

---

## Open Questions

1. **Free tier MAI limit — is 20K right?** At 20K MAI, a startup can run 10K humans + 10K agents. That covers most early-stage products. Too generous? Maybe. But Stytch gives 10K MAU free and WorkOS gives 1M. We need to match or exceed for developer mindshare.
2. **Delegation depth as tier gate — does it convert?** Free at depth 2 means simple user→agent flows work. Pro at depth 4 unlocks orchestrator patterns. Team at depth 7 covers almost all real-world multi-agent graphs. Is this a meaningful upgrade driver?
3. **Enterprise minimum** — is $25K/yr the right floor, or should it be $10K/yr to capture more SMB-adjacent regulated companies?
4. **Vertical focus** — healthcare AI agents (HIPAA) and legal AI agents (compliance-driven audit) are the strongest early enterprise targets. Which first?

---

## Sources

- Auth0 pricing page — AI Agents add-on, B2C/B2B tiers
- Stytch pricing page — free tier includes 10K MAU + AI agents
- WorkOS pricing page — AuthKit free up to 1M MAU
- Clerk pricing page — Pro from $20/mo
- Ory Network pricing — aDAU + M2M token model
- HashiCorp Vault / HCP pricing — enterprise secrets management
- AWS Secrets Manager pricing — per-secret + per-API-call model
- IBM/Ponemon 2024 Cost of Data Breach Report
- SharkAuth codebase — `internal/oauth/`, `internal/proxy/`, `internal/vault/`, `documentation/sdk/vault.md`, `documentation/sdk/cookbook/multi-hop-delegation.md`
- SharkAuth launch docs — `gstack/01-launch-design.md`, `gstack/05-revisions-cold-dm-kit.md`
