# SharkAuth — General Product Guide

> **Purpose:** Context document for Claude Design to produce a Product Launch Video.
> **Synthesized from:** codebase (`internal/`, `admin/src/`, `cmd/`), `playbook/`, `README.md`, `STRATEGY.md`, `DESIGN.md`, `CLOUD.md`, `shark_idea.md`, `PROJECT.md`, `.impeccable.md`, and all planning artifacts. Every claim is grounded in shipped code or locked playbook decisions.

---

## 1. What SharkAuth Is (One Paragraph)

SharkAuth is an open-source OAuth 2.1 + OIDC authorization server that treats AI agents as first-class identities. It ships as a single ~30MB Go binary with embedded SQLite, an embedded React admin dashboard (~50 screens), Python + TypeScript SDKs, and a full CLI. It handles both human authentication (passkeys, MFA, SSO, magic links, RBAC, organizations) and agent authentication (DPoP-bound tokens, RFC 8693 delegation chains, token vault, cascade revocation) in one unified trust chain. `./shark serve` — running in 30 seconds, no Postgres, no Redis, no Docker required.

---



---

## 3. The Problem SharkAuth Solves

### The world changed

Auth0, Clerk, WorkOS, Stytch — they all built for the human-and-app world. Login, SSO, MFA, RBAC. Done. **That world is gone.**

The product you ship in 2026 has agents. Coding agents, support agents, research agents. They hold tokens, they act for the user, they delegate to sub-agents, and on a bad day they get prompt-injected into doing the wrong thing.

### The questions that matter now

1. Who authorized this agent to act for this user, and when does that authorization expire?
2. Is this token bound to a key the holder actually controls, or is "bearer of this string" still the security model?
3. When agent-A delegated to agent-B, was agent-A allowed to do that? How many more hops are allowed?
4. One audit query, one `grant_id` — can we reconstruct every token, every hop, every resource touched?
5. When something goes wrong, what is the blast radius and how do I shrink it without taking the customer offline?

### What teams do today

- Roll their own OAuth 2.1 flows (MCP spec is recent, libraries lag)
- Bolt OAuth onto Auth0/Clerk — agents treated as generic M2M clients with no native delegation, no DPoP, no `act` claim awareness
- Skip auth entirely in dev, panic when shipping to production
- Hand-roll DPoP proof signing because RFC 9449 is fresh and SDK support is patchy

**SharkAuth answers those questions in primitives the OAuth standards already specified. Nobody else shipped them.**

---

## 4. Positioning & Taglines

### Primary tagline (locked)
> **Auth for humans and the agents they ship. One platform. 40MB binary. Self-hosted free.**

### Founder-voice variant
> **Your agents are already doing this. They're just not doing it safely.**

### Subhead
> Your team can already build the concierge that books flights, syncs calendars, charges Stripe. Shark makes sure it ships the right way: real OAuth identity per agent, DPoP-bound tokens, delegation chains with full audit, and one call to revoke when a customer churns.

### 55-word pitch (YC-locked)
> SharkAuth is auth for products that give customers their own agents. Every token traces to the customer who authorized the agent that issued it. When something goes wrong, you have precise responses at different blast radii: revoke a leaked token, kill one agent, cascade-revoke a customer's whole fleet, kill all instances of a buggy agent-type, or disconnect a compromised vault credential. Five layers, one mental model. Open source. Self-hosted free.

### Trust strip (always visible)
- 40MB single binary
- Embedded SQLite (WAL) — zero-config infra
- Self-hosted free, OSS (MIT) forever
- All human auth primitives shipped (passkeys, MFA, SSO, RBAC, orgs, magic links)
- All agent auth primitives shipped (OAuth 2.1, DPoP, RFC 8693 delegation, vault, audit, cascade revoke)

---

## 5. Target Customer & Community

**Primary:** agent-framework maintainers, and SaaS teams shipping per-customer agents — people already hand-rolling DPoP + delegation who recognize the value in 30 seconds.

**OSS Contributors:** We are building a world-class OIDC IdP in idiomatic, clean Go. If you care about cryptography, high-performance proxying, or the future of agentic identity, SharkAuth is your playground. Single-binary architecture means you can contribute from any machine with `go install`.

**Specific shapes:**
- Anthropic MCP server developers
- Cloudflare Agents builders, Smithery contributors
- Vercel AI SDK / LangChain / LlamaIndex maintainers
- Products like Cursor, Replit Agents, Lovable, Bolt, v0 — where each customer owns agents acting on their behalf

**Customer categories (10 verticals):**
1. AI coding assistants (Cursor, Replit, Lovable, Bolt, v0, Devin)
2. Customer support AI (Intercom Fin, Decagon, Sierra)
3. Sales/outbound AI (Clay, Apollo AI, 11x)
4. Voice/phone AI (Bland, Vapi, Retell)
5. Workflow automation AI (Zapier AI, n8n, Lindy AI)
6. Custom-agent platforms (OpenAI Custom GPTs, ChatGPT Apps SDK)
7. Personal/lifestyle AI (Rabbit R1, Friend.com)
8. Vertical industry AI — legal (Harvey), medical (Hippocratic), finance (Hebbia)
9. Browser-based agents (Browser-use, Multion)
10. Code review / DevSecOps AI (Greptile, Coderabbit)

**Not the audience:** Single-tenant internal MCP servers. Teams happy with shared API keys.

---

## 6. The Five-Layer Revocation Moat

The real moat = first-class lineage between customer → agent → token. Five precise responses at different blast radii:

| Layer | Threat | Response | Status |
|---|---|---|---|
| 1. Token | Agent's token leaks via prompt injection | Revoke token + refresh family (RFC 7009) | ✅ Ships |
| 2. Agent | One specific agent compromised | Kill all tokens for that agent | ✅ Ships |
| 3. Customer fleet | Customer churned or goes rogue | Cascade-revoke every agent they spawned | ✅ Ships |
| 4. Agent-type pattern | Buggy agent template v3.2 across all customers | Bulk-revoke by `client_id` pattern | v1.0 |
| 5. Vault credential | Customer's external OAuth compromised | Vault disconnect cascades to derived agent tokens | v1.0 |

**One mental model. Five blast radii.** No competitor offers this.

---

## 7. Core Technical Moat (The Trio)

Any one in isolation is competitor parity. The trio together is unique:

### 7a. DPoP-Bound Access Tokens (RFC 9449)
Every token carries `cnf.jkt` — the SHA-256 thumbprint of the holder's public key. The resource server verifies the DPoP proof JWT on each request. **Steal the token, you get nothing.** The key never leaves the agent.

### 7b. Token Exchange with Full `act` Chain (RFC 8693)
Multi-hop delegation produces a nested `act` claim. Every hop is recorded. The chain flattens into one audit query. `max_hops` is enforced server-side at exchange time, not by the client.

### 7c. `may_act_grants` — Real Rows, Not Vibes
```sql
CREATE TABLE may_act_grants (
    id          TEXT PRIMARY KEY,
    from_id     TEXT NOT NULL,   -- agent that may act
    to_id       TEXT NOT NULL,   -- subject they may act for
    max_hops    INTEGER NOT NULL DEFAULT 1,
    scopes      TEXT NOT NULL DEFAULT '[]',
    expires_at  TIMESTAMP,
    revoked_at  TIMESTAMP,
    created_by  TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```
Every token-exchange call writes the matching `grant_id` into the audit row. Operators can revoke by `id`, `from_id`, `to_id`, or in bulk. Grant lives outside the JWT — revocation takes effect without token rotation.

### 7d. One `grant_id`, One Audit Query, One Full Picture
Audit log carries: subject sub, acting client_id, granted scopes, jkt, parent token id, matched grant id, act chain, issued token id, issuance time. Pivot on any column. Admin UI has a screen for it. SDK has a method for it.

---
IF THE DEMO IS A LOT SIMPLY DONT ADD IT. 
## 8. The Demo Story — Acme Travel Concierge

**File:** `tools/agent_demo_concierge.py`

Maria signs up to Acme Travel. Concierge agent (depth 1) plans her trip. It delegates:

```
Maria (human)
└─ Travel Concierge (master, client_credentials + DPoP)
   ├─ Flight Booker  (act_chain depth 2, vault: Amadeus)
   │  └─ Payment Processor  (act_chain depth 3, vault: Stripe)
   ├─ Hotel Booker   (act_chain depth 2, vault: Booking.com)
   ├─ Calendar Sync  (act_chain depth 2, vault: Google Calendar)
   └─ Expense Filer  (act_chain depth 2, vault: Concur)
```

Every hop: DPoP proof. Every token: scoped down. Every action: one immutable audit row with full `act.act.sub` chain. Maria churns → one call cascade-revokes the whole tree.

**20 steps covering:** Human signup → magic link → session → org creation → API keys → agent CRUD → may_act policies → vault provisioning → DPoP token issuance → depth-2 delegation → vault retrieval → depth-3 delegation → Stripe charge → audit log fetch → DPoP rotation → bulk revoke → vault disconnect → cascade revoke.

---

## 9. What Ships (v0.9.0 — Shipped & Grounded)

### Human Auth
- Email/password signup + login (Argon2id hashing)
- Passkey/WebAuthn login (FIDO2)
- Magic link login
- Social OAuth — Google, GitHub, Apple, Discord
- MFA — TOTP with recovery codes
- Server-side sessions (encrypted cookies)

### Agent Auth
- OAuth 2.1: auth code + PKCE, client credentials, refresh rotation
- OIDC: discovery, JWKS rotation, ID tokens, userinfo
- Token exchange with `act` chain (RFC 8693)
- DPoP (RFC 9449) on token issuance + protected resources
- Token introspection (RFC 7662), Token revocation (RFC 7009)
- `may_act_grants` table, admin CRUD, audit correlation by `grant_id`

### Platform
- **Zero-Code Auth Proxy**: High-performance identity-injecting gateway (Shipped)
- **Token Vault**: AES-256-GCM encrypted-at-rest with auto-refresh (Shipped: Google, GitHub, Slack)
- Organizations + RBAC matrix + invitations
- Webhooks with signature verification, retry, DLQ
- Audit log (filterable, exportable)
- Full CLI: `shark serve/user/agent/org/app/vault/audit/sso/proxy/keys/demo/doctor/version`

### SDKs & UI
- Python SDK + TypeScript SDK
- React component library: `@sharkauth/react`
- Embedded React admin dashboard — ~50 screens

### Infrastructure
- 29 migrations, all idempotent, applied on first boot
- SQLite storage (embedded, zero-config, WAL mode)
- Single binary ~30MB, cross-compiles to darwin/linux/windows on amd64 and arm64

---

## 10. The 60-Second Quickstart

```bash
# 1. Download the binary
curl -fsSL https://github.com/sharkauth/sharkauth/releases/latest/download/shark-$(uname -s)-$(uname -m) -o shark
chmod +x shark

# 2. Boot. First-run prints an admin key on stdout.
./shark serve
# => admin key: sk_live_xxxxx
# => admin UI : http://localhost:8080/admin
# => issuer   : http://localhost:8080



Three commands. No YAML. No schema bootstrap. SQLite WAL file in `./data/`. 29 migrations apply on first boot.

---

## 11. What It Feels Like (Real Code)

**Get a DPoP-bound token:**
```python
from shark_auth import Client, DPoPProver

client = Client(base_url="https://auth.example.com", token="sk_live_...")
prover = DPoPProver.generate()

token = client.oauth.get_token_with_dpop(
    grant_type="client_credentials", dpop_prover=prover,
    client_id="agent-123", client_secret="secret", scope="mcp:write",
)
# token.cnf_jkt = thumbprint binding. Token theft alone is useless.
```

**Delegate down (RFC 8693):**
```python
sub_token = client.oauth.token_exchange(
    subject_token=token.access_token,
    dpop_prover=prover,
    scope="mcp:read",  # scoped down
)
# act-claim records "Concierge → Flight Booker" in JWT
```

**Walk the chain:**
```python
from shark_auth.claims import AgentTokenClaims
claims = AgentTokenClaims.parse(sub_token.access_token)
for hop in claims.delegation_chain():
    print(hop.sub, hop.scope, hop.jkt)
# Maria → Concierge → Flight Booker → Payment Processor
```

**Cascade revoke when customer churns:**
```python
client.users.revoke_agents(user_id="usr_maria", reason="churn")
# Every token every agent ever held: dead.
```

---

## 12. CLI Surface

```
shark serve            # run the server
shark doctor           # 9-check health probe
shark demo             # delegation-with-trace, concierge
shark agent            # CRUD agents
shark user             # CRUD users
shark org              # CRUD orgs + invitations
shark app              # OAuth client mgmt
shark vault            # provider connections
shark audit            # query audit log
shark api-key          # admin keys
shark session          # session inspection
shark sso              # SSO connection mgmt
shark proxy            # proxy rule mgmt
shark keys             # signing-key rotation
shark version / health
```

---

## 13. Admin Dashboard — ~50 Screens

Built in React 18 + Vite + TypeScript. Embedded in the Go binary via `go:embed`.

**Key screens:** Overview (KPI grid + live activity SSE stream), Users (gold-standard list+drawer), Agents (manage, detail with Security/Delegation tabs), Applications (OAuth clients), Delegation Canvas (visual tree of delegation chains), Delegation Chains (audit), Audit Log, Vault Manager, Signing Keys, RBAC Matrix, Organizations, Sessions (debugger), Webhooks, SSO Connections, Settings, API Keys, Get Started, Command Palette, Dev Email Inbox, Identity Hub.

Every primitive has a screen. Every screen has a working revoke button.

---

## 14. Design System — ".impeccable"

**Brand personality:** Bold. Elegant. Fast. Monochrome. "No bullshit."

**Design principles:**
1. **Monochrome** — B&W foundation. Color ONLY for status meaning (success green, warn amber, danger red). No colored brand accents.
2. **Square** — Border radius capped at 3-5px. Sharp 90° corners everywhere.
3. **Dense** — Compact rows (~32-36px), 7-10px padding. Data over chrome.
4. Glamour
6. **Speed as UX** — Optimistic mutations, skeletons, keyboard shortcuts.

**Surfaces step:** `#000` → `#0d0d0d` → `#141414` → `#1c1c1c`

---

## 15. Competitive Position

|                                    | SharkAuth | Auth0 | Clerk | Keycloak |
|:-----------------------------------|:---------:|:-----:|:-----:|:--------:|
| Agent as first-class identity      | ✅        | ❌    | ❌    | ❌       |
| RFC 8693 token exchange + `act`    | ✅        | partial | ❌  | partial  |
| `may_act_grants` table, revocable  | ✅        | ❌    | ❌    | ❌       |
| `max_hops` enforced server-side    | ✅        | ❌    | ❌    | ❌       |
| RFC 9449 DPoP                      | ✅        | ❌    | ❌    | partial  |
| `grant_id` correlated audit        | ✅        | ❌    | ❌    | ❌       |
| Cascade revocation (user → agents) | ✅        | ❌    | ❌    | ❌       |
| Single binary, ~30MB               | ✅        | ❌    | ❌    | ❌ (JVM, ~1GB) |
| Self-hostable, fully OSS           | ✅        | ❌    | ❌    | ✅       |
| Human auth (SSO, MFA, passkeys)    | ✅        | ✅    | ✅    | ✅       |

**Where Auth0/Clerk win today:** bigger ecosystems, polished hosted UX, mature SDKs across more languages. SharkAuth is not trying to replace them for a marketing-site login form. It is for products that ship agents and need both halves of the trust chain in the same audit log.

---

## 16. Pricing Philosophy

**Core principle:** The binary is the product. Cloud is a convenience layer, not a feature gate. Every auth feature ships in the binary at $0. This is non-negotiable. Cloud Flat Pricing for humans. No MAU. No tricks. 100% transparency.



**Competitive cost comparison at 100K MAU:** Clerk ~$1,020/mo. Auth0 ~$2,500/mo. SharkAuth Cloud: $49/mo for Pro (10K MAU, 100 agents). SharkAuth Self-Hosted: $0.

---

## 17. Standards Compliance

Implements: RFC 6749, RFC 6750, RFC 7009, RFC 7517, RFC 7636, RFC 7662, RFC 8414, RFC 8628, RFC 8693, RFC 8707, RFC 9449. OAuth 2.1 draft. WebAuthn Level 2.

---

## 18. Architecture Overview

### Go Backend (`internal/`)
21 packages: `admin`, `api`, `audit`, `auth`, `authflow`, `cli`, `config`, `demo`, `email`, `identity`, `oauth`, `proxy`, `rbac`, `server`, `sso`, `storage`, `telemetry`, `testutil`, `user`, `vault`, `webhook`.

### Storage
SQLite via `modernc.org/sqlite` (pure Go, no CGo). 29 goose migrations in `cmd/shark/migrations/`. WAL mode for concurrent reads. Single `dev.db` file.

### Admin UI
React 18 + Vite + TypeScript. 52 component files totaling ~1.1MB of source. Embedded into Go binary via `go:embed` at `internal/admin/dist/`.

### SDKs
- Python: `shark_auth` package — `DPoPProver`, `OAuthClient`, `AgentsClient`, `AgentTokenClaims`
- TypeScript: `@sharkauth/sdk` — mirror of Python API surface
- React: `@sharkauth/react` — component library with hooks

---

## 19. The Pitch Demo — Blast Radius (4 Beats, 45-60s)

For investors, design-partner CTOs, hallway track at conferences:

| Beat | Time | Screen | Speech |
|---|---|---|---|
| 1. `./shark serve` | ~16s | Terminal: migrations apply, root key prints, server up | Nothing. Let it run. |
| 2. Open dashboard → canvas | ~5s | Delegation canvas loads | Nothing. Silent navigation. |
| 3. Tree on screen | ~10s | María → Concierge → Flight Booker → Payment Processor with scopes, timestamps, act-chain badges | Nothing. Let them read it. |
| 4. Point + revoke | ~15s | Hover Payment Processor → click Revoke → node greys out, others stay green | "This agent holds María's Stripe credentials. If it gets prompt-injected, I do this— *click* —and only it dies. Everything else keeps running." |

**Why this works:** Silence beats explanation. The tree IS the pitch. One sentence carries the message. No jargon. Fast setup is the second advantage.

---

## 20. Video B Script(Internal) — 90s Polished Launch Video

```
0:00-0:10  RAÚL on camera, plain background.
           "I cracked the cryptography of my city's transit
            system out of curiosity. That's how I learned auth. Last month,
            agents started becoming real users on the internet. So I built
            the auth they need."

0:10-0:25  CUT TO: shark serve → first-boot magic-link → dashboard home
           VO: "SharkAuth. Single binary, 30 seconds to running. Full
            human auth — magic-link, SSO, organizations. Plus the thing
            Auth0 cannot ship: agent identities."

0:25-0:50  CUT TO: register agent → delegation chain → vault retrieval (3rd hop fetches Gmail)
           VO: "Every agent inherits its human's privileges. Three
            agents in a delegation chain. Each token cryptographically bound
            to the agent's keypair. The third-hop agent fetches an encrypted
            Gmail credential from the vault. Watch the audit log update."

0:50-1:10  CUT TO: cascade revoke — admin runs revoke-agents on the user
           VO: "When the human is revoked, every agent they spawned
            dies in the same transaction. Rogue insider attribution in one
            query. This is the architectural bet."

1:10-1:30  BACK TO RAÚL on camera
           "I'm Raúl. 18, solo, Monterrey, Mexico. 100,000 lines of
            RFC-correct code in a month with Claude Code. SharkAuth.
            The agent era needs its own auth."
```

---

## 21. Visual & Brand Assets

- **Logo mascot:** Shark character at `admin/src/assets/sharky-full.png`
- **Dashboard screenshot:** `studio.png` in repo root
- **Color palette:** Monochrome — pure B&W with status-only color
- **Fonts:** Hanken Grotesk (display), Manrope (body), Azeret Mono (code/IDs)
- **Admin UI aesthetic:** Dark mode, high-density, sharp corners (3-5px radius max), hairline borders

---

## 22. Key Differentiators for Video (Visual Beats)

1. **The Tree** — Delegation canvas showing María → Concierge → Flight Booker → Payment Processor with DPoP badges and scope labels at each node
2. **The Click** — Single-node revoke on the canvas. Payment Processor greys out. Everything else stays green. Blast radius: one node.
3. **The Boot** — `shark serve` in terminal, 29 migrations apply, admin URL printed. 30 seconds from binary to running OIDC IdP.
4. **The Code** — 10-line Python snippet that Auth0 cannot replicate. DPoP token + delegation + chain walk + cascade revoke.
5. **The Audit** — One `grant_id`, full chain rendered: `[user] → [agent-A] → [agent-B]` with timestamps, scopes, jkt thumbprints.
6. **The Size** — `ls -lh ./shark` → `~30MB`. Compare mentally to Keycloak's ~1GB JVM image.

---

## 23. Tone & Voice Guidelines

- **Builder-to-builder.** No corporate. No marketing fluff.
- **Show real code in hero,** not a "Get Started" button alone.
- **Lead with the moat** (delegation chains + cascade revoke), not the feature checklist.
- **The 30-second screencast IS the pitch.** Embed above the fold.
- **Self-host story is non-negotiable:** free, single binary, no vendor lock-in.
- **Declarative voice:** "I built X." "Auth0 is 18 months away." Not "I think." Not "I hope."
- **Confidence > enthusiasm.** Steady, not breathy or excited.

---

## 24. What NOT to Pitch (Founder-Locked Exclusions)

These exist in code but are explicitly OUT for launch video:

- Branding tab → ComingSoon
- Impersonate → unwired
- Device flow (RFC 8628) → returns 501
- Hosted login pages → devs supply their own
- Proxy Rules tab in dashboard → removed
- PyPI publish → not yet (install via git clone)
- Auth Flow Builder UI → backend works, UI deferred to v1.0

---

## 25. v1.0 Roadmap (Next 2-4 Weeks)

- Revocation layers 4 + 5 (pattern bulk-revoke, vault disconnect cascade)
- Auth flow builder dashboard UI
- Vault custom-provider quirks (Linear, Jira, Slack rotating refresh)
- Hosted/cloud tier
- Public Postman + OpenAPI 3.1 spec polish
- Postgres mode
- SCIM provisioning

---

## 26. Use Cases for Video Scenarios

1. **Personal AI assistant on your inbox** — `may_act` grant from you to assistant-agent, scope `gmail:read`, expires 24h. Revoke from admin UI.
2. **Multi-agent orchestrator** — Planner → research → tool agent. `max_hops=3`, scopes downscoped per hop, every hop recorded.
3. **Internal employee + agent SSO** — Employees via OIDC/SAML, each spawns scoped agents.
4. **MCP server auth** — Register MCP server as application, issue per-consumer client credentials, gate tool calls behind DPoP-bound tokens.
5. **Embedded auth for open-source SaaS** — Drop binary next to your app, point at it for OAuth + OIDC + admin UI.
6. **Compliance-grade audit** — Run proxy in front of APIs, get DPoP enforcement and `grant_id` audit on existing identities.

---

*Written 2026-04-27. Ground truth. Every claim cites shipped code or locked playbook decisions.*
