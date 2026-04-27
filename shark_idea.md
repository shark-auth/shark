# SharkAuth — What It Is

> Source-of-truth document for landing page (sharkauth.com) and `documentation/` deploy (sharkauth.com/docs).
> Synthesized from codebase + `playbook/` + `docs/` on 2026-04-27.
> Every claim cites code or playbook. No invention.

---

## 1. The 55-Word Pitch (locked, founder voice)

> SharkAuth is auth for products that give customers their own agents. Every token traces to the customer who authorized the agent that issued it. When something goes wrong, you have precise responses at different blast radii: revoke a leaked token, kill one agent, cascade-revoke a customer's whole fleet, kill all instances of a buggy agent-type, or disconnect a compromised vault credential. Five layers, one mental model. Open source. Self-hosted free.

Source: `playbook/07-yc-application-strategy.md` (verbatim).

---

## 2. One-Line Positions (lead candidates — unified platform framing)

**Primary (recommended):**
> **Auth for humans and the agents they ship. One platform. 30MB binary. Self-hosted free.**

**Founder-voice variant (verbatim, `playbook/15-concierge-demo-plan.md`):**
> **Your agents are already doing this. They're just not doing it safely.**

**Subhead under either:**
> Your team can already build the concierge that books flights, syncs calendars, charges Stripe. Shark makes sure it ships the right way: real OAuth identity per agent, DPoP-bound tokens, delegation chains with full audit, and one call to revoke when a customer churns.
>
> One binary. SQLite. Self-host in 60 seconds. All human auth primitives included — passkeys, MFA, SSO, RBAC, orgs.

**Why unified-platform leads (not five-layer revoke):**
- Buyer mental model = "I ship agents + I have humans. Auth0 covers humans only. DIY agent auth is broken."
- Five-layer revoke = fear/enterprise-security tone. Wrong altitude. Demoted to "Security model" receipts.
- Permission-grant framing > threat framing. "You can already do this. Shark makes it right."

**Trust strip (always visible):**
- 30MB single binary
- Embedded SQLite — no Redis, no Postgres
- Self-hosted free, OSS forever
- All human auth primitives shipped (passkeys, MFA, SSO, RBAC, orgs, magic links)
- All agent auth primitives shipped (OAuth 2.1, DPoP, RFC 8693 delegation, vault, audit, cascade revoke)

---

## 3. Who It's For

**Target customer (locked):**
MCP server builders, agent-framework maintainers, and SaaS teams shipping per-customer agents — people already hand-rolling DPoP + delegation and recognizing the moat in 30 seconds.

**Specific shapes:**
- Anthropic MCP server developers
- Cloudflare Agents builders, Smithery contributors
- Vercel AI SDK / LangChain / LlamaIndex maintainers
- Lovable / Cursor-style products where each customer owns agents that act on their behalf

**Not the audience:**
Single-tenant internal MCP servers. Teams happy with shared API keys.

Source: `playbook/01-wave1-ui-moat-exposure.md`, `playbook/07-yc-application-strategy.md`, `docs/agent-platform-quickstart.md`.

---

## 4. The Wedge (smallest version someone pays for)

**"Drop-in agent auth for MCP servers."**
One command spins up auth + working delegation chain + DPoP-bound tokens. Killer artifact: 30-second screencast — 3 agents, depth-3 delegation chain, cryptographic proofs visible.

Source: `playbook/04-wave4-launch.md`, `playbook/15-concierge-demo-plan.md`.

---

## 5. The Five-Layer Revocation Moat (lead with this)

Real moat = first-class lineage between customer → agent → token. Five precise responses at different blast radii:

| Layer | Action | When to use |
|---|---|---|
| 1 | Revoke a single token | Token leaked in a log |
| 2 | Kill one agent | One agent went rogue |
| 3 | Cascade-revoke a customer's whole fleet | Customer churned or compromised |
| 4 | Kill all instances of a buggy agent-type | Bad release pattern-wide |
| 5 | Disconnect a compromised vault credential | Upstream provider key rotated |

**One mental model. Five blast radii.** No competitor offers this.

---

## 6. What Ships (locked features)

| Feature | RFC | Source |
|---|---|---|
| OAuth 2.1 + PKCE | 6749 + 7636 | `internal/oauth/server.go` |
| DPoP proof-of-possession (per-request, per-hop) | 9449 | `internal/auth/dpop.go`, `cnf.jkt` claim |
| Token Exchange w/ delegation chains (`act` claim) | 8693 | `internal/oauth/token_exchange.go` |
| Dynamic Client Registration | 7591/7592 | `/api/v1/admin/apps` |
| Token Introspection | 7662 | `/oauth/introspect` |
| Token Revocation | 7009 | `/oauth/revoke` |
| Authorization Server Metadata | 8414 | `/.well-known/oauth-authorization-server` |
| Resource Indicators (audience-bound) | 8707 | token issuance path |
| JWKS | 7517 | `/.well-known/jwks.json` |
| SSO + WebAuthn passkeys + MFA recovery | — | `oauth_accounts`, `passkey_credentials`, `mfa_recovery_codes` |
| Organizations + RBAC + invitations | — | `organizations`, `org_roles`, `organization_invitations` |
| Token Vault (encrypted-at-rest, AES-256-GCM, lazy refresh) | — | `internal/vault/vault.go` |
| Audit log w/ delegation chain metadata | — | `internal/audit/audit.go`, `/api/v1/audit-logs` |
| Webhooks (signed, retry, DLQ) | — | `webhooks`, `webhook_deliveries` |
| Reverse proxy (identity injection, circuit breaker, DPoP validation) | — | `internal/proxy/` |
| Single-binary deploy w/ embedded React admin | — | `cmd/shark/main.go` |

**Endpoints:** ~60 routes across OAuth, admin, agents, vault, organizations, audit, webhooks, proxy. See `internal/api/router.go`.

**Storage:** 26 goose migrations (`cmd/shark/migrations/`). modernc.org/sqlite, embedded. No external deps.

**Binary size:** ~30MB. Boots cold. `shark serve`. Done.

---

## 7. CLI Surface

```
shark serve            # run the server
shark doctor           # 9-check health probe (admin key, DB, signing keys, ...)
shark demo             # subcommands below
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
shark reset            # nuke local DB (dev only)
shark keys             # signing-key rotation
shark version / health
```

**Top demo:** `shark demo delegation-with-trace` — 3-hop chain, DPoP at every hop, vault retrieval, self-contained HTML report.

Source: `cmd/shark/cmd/demo.go`, `cmd/shark/main.go`.

---

## 8. What It Feels Like (real code, no invention)

**Get a DPoP-bound token (RFC 9449):**
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

Source: `documentation/sdk/python/get_token_with_dpop.md`, `token_exchange.md`, `sdk_methods_w2.md`.

---

## 9. The Demo Story (Acme Travel Concierge)

**File:** `tools/agent_demo_concierge.py` + `docs/demos/concierge.md`.

Maria signs up to Acme Travel. Concierge agent (depth 1) plans her trip. It delegates:
- → Flight Booker (depth 2) — calls Amadeus via vault
- → Hotel Booker (depth 2) — calls Booking.com via vault
- → Calendar Sync (depth 2) — Google Calendar
- → Payment Processor (depth 3, called by Flight Booker) — Stripe

Every hop: DPoP proof. Every token: scoped down. Every action: one immutable audit row with full `act.act.sub` chain. Maria churns → one call cascade-revokes the whole tree.

**This is the proof. The screencast is the wedge.**

---

## 10. Positioning vs Auth0 / WorkOS / Clerk / Stytch

| | SharkAuth | Auth0 / WorkOS / Clerk |
|---|---|---|
| Per-customer agent identity | First-class | Hack on top of M2M |
| DPoP (RFC 9449) | Native, per-hop | None |
| Token exchange w/ delegation (RFC 8693) | Native, audited | None |
| Cascade revoke (5 layers) | One call | Manual per token |
| Self-host | Free, single binary | Enterprise tier or unsupported |
| External deps | None (embedded SQLite) | Postgres + Redis usually |
| Per-MAU pricing | Free forever | Yes |
| Token vault for upstream APIs | Built-in, AES-256-GCM | Build it yourself |

---

## 11. Three Onboarding Paths (landing page CTAs)

1. **Customer Agents** — SaaS shipping per-tenant agents
   `documentation/quickstarts/01-customer-agents.md`
2. **MCP Server** — OAuth 2.1 gating for Model Context Protocol servers
   `documentation/quickstarts/02-mcp-server.md`
3. **Auth0 Replacement** — self-hosted, agent-native, drop-in
   `documentation/quickstarts/04-auth0-replacement.md`

Bonus paths: `03-internal-platform.md`, `10-five-layer-revocation.md`, `11-delegation-chains.md`.

---

## 12. Do NOT Pitch (founder-locked exclusions)

These exist in code but are explicitly OUT for launch — don't mention on landing page or docs hero:

- **Branding tab** → ComingSoon (CRUD exists, not battle-tested)
- **Impersonate** → unwired, not launched
- **Device flow (RFC 8628)** → returns 501; removed from `/.well-known/oauth-authorization-server`
- **Hosted login pages** → devs supply their own redirect URLs
- **Proxy Rules tab in dashboard** → tab removed from `/applications` (proxy backend ships, dashboard tab cut)
- **PyPI publish** → not yet (install via git clone)

Source: `playbook/PRE_LAUNCH_CHECKLIST.md`, `playbook/08-launch-scope-cuts.md`.

---

## 13. Known Backend Bugs (W+1, do not demo)

- `POST /api/v1/agents/{id}/policies` → 404 (route not registered). `shark demo delegation-with-trace` partially affected.
- `POST /users/{id}/revoke-agents` → returns `invalid_client` error envelope mismatch.
- `GET /users/{id}/agents?filter=created|authorized` → empty/500 on JOIN.
- `GET /me/agents` → empty list (created_by lineage mismatch).
- Proxy: 2 P0 + 4 P1 + 6 P2 in `internal/proxy/` (see `INVESTIGATE_REPORT.md`).

**Implication for landing page:** lead with delegation chains + cascade revoke + DPoP. Don't lead with proxy or bulk-revoke filters yet.

Source: `playbook/POST_LAUNCH_BUGS.md`.

---

## 14. Distribution Levers (post-launch, not for v0.1 hero)

- **Claude Code skill** (`skills/shark-cc/`) — install via `claude mcp add`
- **`shark serve --mcp`** — SharkAuth itself as an MCP server (W19+)
- **Demo screencast** as the wedge, not a doc

Source: `playbook/09-post-launch-harness-skill-eureka.md`, memory `project_eureka_harness_skill`.

---

## 15. Standards Compliance Block (footer / trust strip)

Implements: RFC 6749, RFC 6750, RFC 7009, RFC 7517, RFC 7591, RFC 7592, RFC 7636, RFC 7662, RFC 8414, RFC 8628, RFC 8693, RFC 8707, RFC 9449. OAuth 2.1 draft. WebAuthn Level 2.

---

## 16. Tone for Landing Page Copy

- Builder-to-builder. No corporate. No marketing fluff.
- Show real code in hero, not a "Get Started" button alone.
- Lead with the moat (delegation chains + cascade revoke), not the feature checklist.
- The 30-second screencast IS the pitch. Embed above the fold.
- Self-host story is non-negotiable: free, single binary, no vendor lock-in.

---

## 17. Landing Page Skeleton (unified-platform-led)

1. **Hero**
   - H1: *Auth for humans and the agents they ship.*
   - Subhead: unified-platform paragraph (see § 2)
   - Three trust pills: `30MB binary` · `Self-host in 60s` · `Free forever`
   - Primary CTA: `shark serve` install one-liner (copy button)
   - Secondary CTA: 30-second concierge screencast embed

2. **The "you can already do this" beat**
   - Acme Travel concierge demo, 3-hop delegation chain
   - Real audit log render — Maria → Concierge → Flight Booker → Payment Processor
   - One paragraph: "Your team can already build this. Shark makes it ship right."

3. **One platform, two halves**
   - Side-by-side block:
     - **For your humans:** passkeys, MFA, SSO, magic links, orgs, RBAC, invitations, sessions
     - **For your agents:** OAuth 2.1 per agent, DPoP-bound tokens, RFC 8693 delegation, token vault, cascade revoke, audit chain
   - Tagline: *Auth0 covers half. We cover both.*

4. **What it feels like** (real code, see § 8)
   - Tab 1: Get a DPoP token (10 lines)
   - Tab 2: Delegate down (token exchange)
   - Tab 3: Walk the chain
   - Tab 4: Cascade revoke

5. **Self-host story** (own the section)
   - One binary download. SQLite embedded. React admin embedded. No Postgres. No Redis. No SaaS bill.
   - `curl … | sh` → `shark serve` → admin URL printed → first agent token in <60s
   - Footprint diagram: 30MB vs Auth0's stack

6. **Three quickstart CTAs**
   - Customer Agents (SaaS shipping per-tenant agents)
   - MCP Server (OAuth 2.1 for Model Context Protocol)
   - Auth0 Replacement (drop-in, self-hosted)

7. **Vs Auth0 / WorkOS / Clerk / Stytch** (table from § 10)

8. **Security model — receipts** (the demoted five-layer section)
   - Five-layer revoke as proof, not pitch
   - DPoP per-hop binding diagram
   - Audit chain example
   - Title: *When something goes wrong, you have five precise responses.*

9. **Standards trust strip footer** (RFC list from § 15)

**Above-the-fold rule:** screencast + unified-platform tagline + install one-liner. Nothing else competes.

**What's NOT above the fold (intentional demotion):**
- Five-layer revoke → § 8 (receipts)
- RFC alphabet → footer
- Feature checklist → § 3 (one platform, two halves)
- Comparison tables → § 7

---

*Written 2026-04-27. Ground truth. Update on every locked-feature change.*
