# sharkauth

```
 ███████╗██╗  ██╗ █████╗ ██████╗ ██╗  ██╗
 ██╔════╝██║  ██║██╔══██╗██╔══██╗██║ ██╔╝
 ███████╗███████║███████║██████╔╝█████╔╝
 ╚════██║██╔══██║██╔══██║██╔══██╗██╔═██╗
 ███████║██║  ██║██║  ██║██║  ██║██║  ██╗
 ╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝
```

### SharkAuth · auth for products that give customers their own agents

`Self-hosted, single-binary, OAuth 2.1 + RFC 8693 + DPoP. Five-layer revocation built in.`

[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue?style=flat-square)](LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/sharkauth/sharkauth/ci.yml?style=flat-square)](https://github.com/sharkauth/sharkauth/actions)
[![Release](https://img.shields.io/github/v/release/sharkauth/sharkauth?style=flat-square)](https://github.com/sharkauth/sharkauth/releases/latest)

**demo: 8-second cold-start to first agent token**

---

## Why SharkAuth

You are building a product where each customer gets their own agent — a research assistant, a coding tool, a customer-support bot. Your product issues tokens on behalf of those agents, delegates across agent hops, and needs to revoke everything for a customer in one call when they cancel.

No managed auth provider handles this today.

**Without SharkAuth:**

- **Vendor lock-in on token issuance.** Auth0/Clerk own your token format, rotation schedule, and revocation API. You cannot add DPoP binding or act-chain logging without their blessing.
- **No agent-native primitives.** RFC 8693 token exchange (act-on-behalf-of), DPoP key binding, and per-agent delegation policies are not in any SaaS auth product. You build it yourself or your agents run on bearer tokens with no proof of possession.
- **No act-on-behalf-of audit chain.** When agent A delegates to agent B delegates to agent C, who acted? Which token is live? Which hop caused the data exfil? Without structured delegation records you are answering these questions from logs, not from a purpose-built chain.

SharkAuth is the auth layer built for this exact problem. Single binary. Self-hosted. Ships agents as first-class identities.

---

## Quickstart

```bash
# Download the binary from GitHub Releases
# https://github.com/sharkauth/sharkauth/releases/latest
./shark serve

# Open http://localhost:8080/admin
# Paste the admin key printed to your terminal
# (also written to data/admin.key.firstboot)
```

That's it. No config file. No database setup. No environment variables required for first boot.

**demo: dashboard first-boot walkthrough — create user, register agent, mint first DPoP token**

---

## The Five Layers

Five-layer revocation is the moat. Each layer is independently callable.

| Layer | Name | Ships |
|-------|------|-------|
| L1 | Individual token revoke — kill one specific access or refresh token | v0.1 |
| L2 | Agent-wide revoke — kill all tokens for one agent identity | v0.1 |
| L3 | User-cascade revoke — kill all tokens for all agents owned by a user | v0.1 |
| L4 | Bulk pattern revoke — kill all tokens matching a scope/audience/tag pattern | v0.2 (W18-W19) |
| L5 | Vault disconnect cascade — revoke third-party provider tokens stored in vault + all derived agent tokens | v0.2 (W18-W19) |

When a customer cancels: one `bulk_revoke_by_pattern` call (L4) drops every agent token in their tenant. When a vault credential is compromised: L5 cascades from the vault credential outward to every token derived from it. Layers 1-3 ship now; 4-5 land W18-W19.

---

## Architecture

**diagram: human → app → agent → resource with shark at every boundary**

SharkAuth sits at every token boundary in your stack:

- Human authenticates to your app via OAuth 2.1 + PKCE
- Your app provisions an agent identity in SharkAuth via DCR (RFC 7591)
- Agent requests a DPoP-bound token (RFC 9449) — proof of key possession, not just bearer
- Agent acts on behalf of human via RFC 8693 token exchange — auditable `act` chain recorded
- Shark validates proof at every hop, logs every delegation, and holds all revocation authority

All config lives in SQLite. No YAML, no external dependencies, no runtime services beyond the binary.

---

## Use Cases

### AI SaaS shipping agents per customer

Your product gives each paying customer one or more agents. Each agent needs scoped credentials, key-bound tokens, and a revocation path that works at cancellation time. SharkAuth provisions agent identities on signup, issues DPoP-bound tokens on each agent invocation, and cascade-revokes on churn. Python SDK ships now; TS SDK in v0.2.

### Internal AI platform team (compliance + audit trails)

Your platform team runs LLM infrastructure for internal product teams. Security wants an audit record of every act-on-behalf-of hop: which agent accessed which resource, via which delegation chain, at which timestamp. SharkAuth's delegation chain canvas gives you that record. Every token exchange writes a structured audit event with the full `act` chain flattened for query.

### MCP server developers

If you are building an MCP server and need to gate tool calls behind scoped tokens with DPoP binding, SharkAuth is a drop-in OAuth 2.1 + DPoP authorization server. Register your MCP server as an application, issue client credentials per consumer, enforce scope per tool. This is a tertiary use case — the primary wedge is agent platforms — but MCP developers get the full protocol stack.

### Self-hosters wanting an Auth0 replacement with agent support

SharkAuth is a complete OAuth 2.1 authorization server: auth code + PKCE, client credentials, refresh rotation, DCR, device flow, introspection, revocation, JWKS. Passkeys, MFA/TOTP, magic links, SSO (SAML 2.0 + OIDC), RBAC, webhooks, and a full audit log ship in the same binary. The difference: it also handles agents.

---

## SDK

```bash
# Python SDK — dogfood mode (PyPI release after internal validation)
pip install git+https://github.com/sharkauth/sharkauth#subdirectory=sdk/python
# PyPI release coming W+1 after dogfood validation
```

```python
from shark_auth import Client, DPoPProver

client = Client(base_url="http://localhost:8080", token="sk_live_...")
prover = DPoPProver.generate()

# Mint a DPoP-bound agent token
token = client.oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="agent-123",
    client_secret="secret",
    scope="resource:read",
)

# Token exchange — act on behalf of a user (RFC 8693)
delegated = client.oauth.token_exchange(
    subject_token=token.access_token,
    scope="resource:read",
    dpop_prover=prover,
)
```

TypeScript SDK ships v0.2 (W18-W19).

---

## Roadmap

Transparent roadmap — YC reviewers and OSS contributors can see what's committed.

**v0.1 (now):**
- Identity hub (users, agents, applications, organizations)
- Token vault — managed third-party OAuth for agents (Google, Slack, GitHub, Microsoft, Notion, Linear, Jira)
- Full audit log with delegation chain canvas
- Auth flow builder (Auth0 Actions-style post-auth pipelines)
- Delegation chains + RFC 8693 act-chain flattening
- Python SDK (10 methods, dogfood mode)
- DCR (RFC 7591) + DPoP (RFC 9449)
- Five-layer revocation: L1-L3 fully live

**v0.2 (W18-W19):**
- Reverse proxy with identity header injection
- Bulk-pattern revoke (L4)
- Vault disconnect cascade (L5)
- TypeScript SDK (agent-native methods, mirrors Python surface)
- Auth flow builder UI editor

**v0.3 (W20+):**
- Hosted tier (managed SharkAuth, no infra)
- Claude Code skill — install SharkAuth and scaffold agent auth in one command
- MCP wrapper — expose SharkAuth as an MCP server for agent-native tooling

---

## Community + License

**License:** Apache-2.0. See [LICENSE](LICENSE).

**Discord:** [Join the SharkAuth Discord](https://discord.gg/sharkauth) — placeholder, link live at launch

**GitHub Discussions:** [Ask questions, share integrations](https://github.com/sharkauth/sharkauth/discussions)

**Contributing:** See [CONTRIBUTING.md](CONTRIBUTING.md) if it exists, else open a Discussion with your proposal first.

Build from source:

```bash
git clone https://github.com/sharkauth/sharkauth
cd sharkauth
go build -o shark ./cmd/shark
./shark serve
```

Requires Go 1.22+.

---

## Why Now

Every product is becoming an agent platform. Auth was already a differentiator — the teams that owned their auth stack moved faster, migrated without begging vendors, and didn't pay per-MAU rent on their own users. Agent auth is a moat: the team that ships RFC-correct DPoP binding, delegation chain auditing, and five-layer revocation as open infrastructure before Auth0 retrofits agent semantics owns the category for the next 18 months. That window is open now.
