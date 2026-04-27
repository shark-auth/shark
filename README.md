<p align="center">
  <img src="admin/src/assets/sharky-full.png" alt="SharkAuth" width="200">
</p>

<h3 align="center">Auth for apps that ship agents to customers.</h3>
<p align="center">Self-hosted · Single binary · SQLite · OAuth 2.1 · DPoP · Delegation chains</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License"></a>
  <a href="https://github.com/sharkauth/sharkauth/actions"><img src="https://img.shields.io/github/actions/workflow/status/sharkauth/sharkauth/ci.yml?style=flat-square" alt="Build"></a>
  <a href="https://github.com/sharkauth/sharkauth/releases/latest"><img src="https://img.shields.io/github/v/release/sharkauth/sharkauth?style=flat-square" alt="Release"></a>
</p>

---

## What is this

SharkAuth is an open-source auth server for products where **each customer gets their own AI agents**  and those agents need scoped credentials, delegation chains, and revocation that actually works.

One ~30MB binary. No config files. No external database. `shark serve` and you're running.



## Quickstart

```bash
# Download from GitHub Releases (macOS, Linux, Windows)
curl -fsSL https://github.com/sharkauth/sharkauth/releases/latest/download/shark-$(uname -s)-$(uname -m) -o shark
chmod +x shark
./shark serve

# Open http://localhost:8080/admin
# Paste the admin key printed in your terminal
```

Docker image ships in v0.2.

**That's it.** No YAML. No Postgres. No env vars. First boot creates everything.

```bash
# Run the concierge demo — full delegation chain end-to-end
shark demo concierge
```

Shows user → research-agent → tool-agent with `act` chain + audit trail.

---

## Why this exists

You're building a product where each customer gets agents — a coding assistant, a support bot, a research tool. Those agents hold your customer's credentials, call APIs on their behalf, and delegate to sub-agents.

Today, you hand-roll all of this:

- Per-tenant token scoping
- Agent credential rotation
- A vault for customer OAuth tokens (Gmail, Slack, GitHub)
- Revocation when something goes wrong

And your hand-rolled stack is probably missing layers. SharkAuth handles the full chain — from human login to the agent's third-hop delegated sub-token — in one server, one audit log, one revocation model.

---

## Agent auth in 10 lines

```python
from shark_auth import Client, DPoPProver

client = Client(base_url="http://localhost:8080", token="sk_live_...")
prover = DPoPProver.generate()

# DPoP-bound token — theft alone is useless without the private key
token = client.oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="agent-123",
    client_secret="secret",
    scope="mcp:write",
)

# Delegate to a sub-agent with narrower scope (RFC 8693)
sub_token = client.oauth.token_exchange(
    subject_token=token.access_token,
    scope="mcp:read",
    dpop_prover=prover,
)

# Every request carries cryptographic proof of possession
resp = client.http.get_with_dpop("/resource", token=sub_token.access_token, prover=prover)
```

Every line above is something Auth0, Clerk, and WorkOS cannot do today. Not because they're slow — because their token model wasn't built for agents.

---

## What ships

### Human auth (complete)
- Password + MFA/TOTP + recovery codes
- Magic link (passwordless)
- Passkeys / WebAuthn
- SSO: SAML 2.0 + OIDC federation
- OAuth 2.1: auth code + PKCE, client credentials, device flow, refresh rotation
- Dynamic Client Registration (RFC 7591)
- Organizations + RBAC
- Session management + admin dashboard

### Agent auth (the part that's new)
- **DPoP (RFC 9449)** — every token is cryptographically bound to the agent's keypair. Steal the token, it's useless without the key.
- **Token exchange (RFC 8693)** — agents delegate to sub-agents with `act` claim chains. Every hop is audited.
- **Agent identities** — agents are first-class entities, not hacked-on machine clients.
- **Delegation policies** — `may_act` rules control which agents can delegate to which.
- **Token vault** — encrypted storage for customer OAuth tokens. Verified end-to-end: Google (Gmail/Drive/Calendar), GitHub. Experimental in v0.1: Slack, Microsoft (multi-tenant), Notion. Linear and Jira land in v0.2 with full provider-quirk handling. Custom OAuth 2.0 providers supported for textbook auth-code flows; non-standard token responses, post-exchange steps, and rotating-refresh quirks ship in v0.2 (see `playbook/CUSTOM_VAULT_LIMITATIONS.md`). Agents retrieve tokens via DPoP-bound requests scoped to the delegation chain.

### Revocation (depth-of-defense)

When something goes wrong, you need the right blast radius:

| Layer | Threat | Response |
|:------|:-------|:---------|
| **Token** | Agent's token leaks via prompt injection | Revoke that specific token + its refresh family |
| **Agent** | One agent is compromised | Kill all tokens for that agent |
| **Customer** | Customer goes rogue or cancels | Cascade-revoke every agent they spawned — one call |
| **Pattern** | Buggy agent template v3.2 across all customers | Bulk-revoke by `client_id` pattern |
| **Vault** | Customer's external OAuth credential compromised | Vault disconnect cascades to every derived agent token |

Five layers. Each independently callable. Layers 1–3 ship in v0.1. Layers 4–5 ship in v0.2.

**Custom OAuth providers (v0.1):** standard auth-code flows with RFC 6749 token responses work via the admin "Add provider" form. Non-standard quirks (custom token parsing, required extra params, post-exchange steps) ship in v0.2 — see `playbook/CUSTOM_VAULT_LIMITATIONS.md`.

### Audit
- Every token issuance, exchange, and revocation is logged
- Delegation chain canvas: visual trace of `[user] → [agent-A] → [agent-B]`
- Full `act` claim chain flattened for query

---

## Architecture

```
Human ─── OAuth 2.1 + PKCE ──→ Your App
                                   │
                              DCR (RFC 7591)
                                   │
                                   ▼
                               SharkAuth ◄── admin dashboard
                                   │          SQLite (embedded)
                              DPoP-bound      audit log
                              token issued    delegation policies
                                   │
                                   ▼
                            Agent (your code)
                                   │
                         token exchange (RFC 8693)
                         scope downscoped
                         act chain recorded
                                   │
                                   ▼
                          Sub-agent / Resource
```

- **Single binary** — Go, ~30MB, cross-compiles to macOS/Linux/Windows (arm64 + amd64)
- **SQLite** — embedded, WAL mode, zero ops. All state in `./data/`
- **No external dependencies** — no Redis, no Postgres, no message queue
- **Admin dashboard** — built-in React admin UI, embedded in the binary

---

## SDK

### Python

_Available on PyPI — install snippet below works once published; tracking ship today._

```bash
pip install shark-auth
```

```python
from shark_auth import Client, DPoPProver, decode_agent_token

client = Client(base_url="http://localhost:8080", token="sk_live_...")

# Register an agent
agent = client.agents.register_agent(
    app_id="my-app",
    name="research-bot",
    scopes=["mcp:read", "mcp:write"],
)

# Verify tokens locally (3 lines)
claims = decode_agent_token(
    token,
    jwks_url="http://localhost:8080/.well-known/jwks.json",
    expected_issuer="http://localhost:8080",
    expected_audience=agent["client_id"],
)
```

See [`examples/`](examples/) for runnable scripts.

### TypeScript

```bash
npm install @sharkauth/node
```

```typescript
import { Client, DPoPProver } from "@sharkauth/node";

const client = new Client({ baseUrl: "http://localhost:8080", token: "sk_live_..." });
const prover = await DPoPProver.generate();

// DPoP-bound token
const token = await client.oauth.getTokenWithDpop({
  grantType: "client_credentials",
  dpopProver: prover,
  clientId: "agent-123",
  clientSecret: "secret",
  scope: "mcp:write",
});

// Delegate to sub-agent (RFC 8693)
const subToken = await client.oauth.tokenExchange({
  subjectToken: token.accessToken,
  scope: "mcp:read",
  dpopProver: prover,
});
```

Coverage: ~40% of admin surface in v0.1; full parity in v0.2.

---

## Comparison

| | SharkAuth | Auth0 | Clerk | WorkOS |
|:--|:--|:--|:--|:--|
| Self-hosted, single binary | ✅ | ❌ | ❌ | ❌ |
| Agent as first-class identity | ✅ | ❌ | ❌ | ❌ |
| DPoP (RFC 9449) | ✅ | ❌ | ❌ | ❌ |
| Token exchange with `act` chain (RFC 8693) | ✅ | ❌ | ❌ | ❌ |
| Delegation chain audit | ✅ | ❌ | ❌ | ❌ |
| Cascade revocation (user → agents → tokens) | ✅ | ❌ | ❌ | ❌ |
| Token vault for customer OAuth credentials | ✅ | ❌ | ❌ | ❌ |
| Human auth (SSO, MFA, passkeys, magic link) | ✅ | ✅ | ✅ | ✅ |
| Per-MAU pricing | Free | $$$  | $$$  | $$$  |

SharkAuth is not trying to replace Auth0 for a standard SaaS login page. It's built for the products that also need **agent auth on top of human auth** — and need both in the same trust chain.

---

## Who is this for

**AI SaaS shipping agents per customer.** You give each paying customer one or more agents. Agents need scoped credentials, key-bound tokens, and a revocation path that works at cancellation time.

**Platform teams running LLM infrastructure.** Security wants an audit record of every delegation hop: which agent accessed which resource, via which chain, at which timestamp.

**MCP server developers.** You need to gate tool calls behind scoped tokens with DPoP binding. SharkAuth is a drop-in OAuth 2.1 + DPoP authorization server. Register your server as an application, issue client credentials per consumer, enforce scope per tool.

**Self-hosters who want auth they own.** Complete OAuth 2.1 authorization server with SSO, passkeys, MFA, RBAC, organizations, webhooks, audit log, in a binary you control. No per-MAU rent.

---

## Roadmap

**v0.1 (now):**
- Full human + agent auth stack
- DPoP + token exchange + delegation chains
- Token vault (7 providers)
- Revocation layers 1–3
- Python SDK
- Admin dashboard + audit log

**v0.2:**
- Revocation layers 4–5 (pattern + vault cascade)
- Reverse proxy with identity header injection
- Auth flow builder UI
- Vault custom-provider quirks (Linear, Jira, Slack quirks, rotating refresh)

**v0.3:**
- Hosted/cloud tier
- Claude Code skill + MCP server wrapper

Full roadmap: [STRATEGY.md](STRATEGY.md)

---

## Build from source

```bash
git clone https://github.com/sharkauth/sharkauth
cd sharkauth

# Build frontend + backend
cd admin && npm install && npm run build && cd ..
go build -o shark ./cmd/shark

./shark serve
```

Requires Go 1.22+ and Node 20+.

---

## Contributing

SharkAuth is open source under the [MIT License](LICENSE).

- **Issues:** [GitHub Issues](https://github.com/sharkauth/sharkauth/issues) — bug reports and feature requests
- **Discussions:** [GitHub Discussions](https://github.com/sharkauth/sharkauth/discussions) — questions, integrations, proposals
- **Discord:** [Join](https://discord.gg/sharkauth)

If you want to be an early integrator, open an issue or DM. I'll help wire it up.

---

<p align="center">
Built by <a href="https://github.com/raulgooo">Raúl</a> in Monterrey, Mexico.<br>
If your product ships agents to customers, their auth stack starts here.
</p>
