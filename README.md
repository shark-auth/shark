<p align="center">
  <img src="admin/src/assets/sharky-full.png" alt="SharkAuth" width="180">
</p>

<h3 align="center">Open-source auth for the agent-on-behalf-of-user world.</h3>
<p align="center">Real delegation. Real DPoP. Real audit. One ~40MB Go binary, SQLite-embedded, runs on a Raspberry Pi.</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License"></a>
  <a href="https://github.com/sharkauth/sharkauth/releases/latest"><img src="https://img.shields.io/badge/version-v0.9.0-blue?style=flat-square" alt="Version"></a>
  <a href="#"><img src="https://img.shields.io/badge/go-1.22%2B-00ADD8?style=flat-square" alt="Go"></a>
  <a href="#"><img src="https://img.shields.io/badge/SQLite-embedded-003B57?style=flat-square" alt="SQLite"></a>
</p>

---

## What is this

An OAuth 2.1 + OIDC authorization server that treats agents as first-class identities. RFC 8693 token exchange with verifiable `act` claim chains, RFC 9449 DPoP key-binding, an issuer-issued `may_act_grants` table that is revocable and audit-correlated, and a built-in admin UI that has a working revoke button on every primitive it shows.

It compiles to a single ~40MB binary. No Postgres. No Redis. No Docker required. `./shark serve` and you have an OIDC IdP running on `:8080`.

```
[ user ]
   │  OAuth 2.1 + PKCE  (or passkey, or magic link, or SSO)
   ▼
[ SharkAuth ]──────────────  may_act_grants  (operator-issued, max_hops, expires_at, revocable)
   │  DPoP-bound access token (cnf.jkt)
   ▼
[ agent-A ]
   │  RFC 8693 token exchange   subject_token = agent-A
   │  act chain recorded        scope downscoped
   ▼
[ agent-B ]                     act: { sub: agent-A, act: { sub: user } }
   │
   ▼
[ resource ]   audit row carries grant_id, jkt, full act chain
```

> Drop a real screenshot here once captured: `![Delegation canvas](docs/assets/delegation-canvas.png)`
> The screen lives at `admin/src/components/delegation_canvas.tsx` and renders the full hop chain with revoke buttons inline.

---

## Why this exists

Auth0, Clerk, WorkOS, Stytch, Keycloak. They all built for the human-and-app world. Login, SSO, MFA, RBAC. Done.

That world is gone. The product you ship in 2026 has agents. Coding agents, support agents, research agents. They hold tokens, they act for the user, they delegate to sub-agents, and on a bad day they get prompt-injected into doing the wrong thing.

The questions that matter now:

- Who authorized this agent to act for this user, and when does that authorization expire?
- Is this token bound to a key the holder actually controls, or is "the bearer of this string" still the security model?
- When agent-A delegated to agent-B, was agent-A allowed to do that? How many more hops are allowed?
- One audit query, one `grant_id`, and we should be able to reconstruct every token, every hop, every resource touched.
- When something goes wrong, what is the blast radius and how do I shrink it without taking the customer offline?

SharkAuth answers those questions in primitives the OAuth standards already specified. Nobody else shipped them.

You are the user. Your agent acts on your behalf. Both should be auditable end to end. That is the whole pitch.

---

## The 60-second quickstart

```bash
# 1. Download the binary (Linux/macOS, arm64 + amd64)
curl -fsSL https://github.com/sharkauth/sharkauth/releases/latest/download/shark-$(uname -s)-$(uname -m) -o shark
chmod +x shark

# 2. Boot. First-run prints an admin key on stdout.
./shark serve
# => admin key: sk_live_xxxxx
# => admin UI : http://localhost:8080/admin
# => issuer   : http://localhost:8080

# 3. Run the bundled demo: full delegation chain end to end.
./shark demo concierge
# user → research-agent → tool-agent
# prints DPoP-bound tokens, the act chain, and the matching audit rows.
```

That is the wow moment. Three commands. No YAML. No schema bootstrap. SQLite WAL file in `./data/`. 29 migrations apply on first boot.

Already running it? Skip to [Code samples](#code-samples).

---

## What makes it different

### `may_act_grants` are real rows, not vibes

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

Every token-exchange call that traverses a grant writes the matching `grant_id` into the audit row. Operators can revoke by `id`, by `from_id`, by `to_id`, or in bulk. The grant lives outside the JWT, so revocation does not require token rotation to take effect.

Source of truth: `internal/oauth/exchange.go`, migration `00028_may_act_grants.sql`.

### DPoP-bound access tokens (RFC 9449)

Every token carries `cnf.jkt`, the SHA-256 thumbprint of the holder's public key. The resource server verifies the DPoP proof JWT on each request. Steal the token, you get nothing. The key never leaves the agent.

### Token Exchange with full `act` chain (RFC 8693)

Multi-hop delegation produces a nested `act` claim. Every hop is recorded. The chain flattens into one audit query. `max_hops` is enforced at exchange time, not by the client.

### One `grant_id`, one audit query, one full picture

Audit log carries: subject sub, acting client_id, granted scopes, jkt, parent token id, the matched grant id, the act chain, the issued token id, the issuance time. Pivot on any of those columns. The admin UI has a screen for it, the SDK has a method for it.

### Five layers of revocation, each independently callable

| Layer | Threat | Response |
|:------|:-------|:---------|
| Token | Agent's token leaks via prompt injection | Revoke the token + its refresh family |
| Agent | One agent is compromised | Kill all tokens for that agent |
| User | User goes rogue or cancels | Cascade-revoke every agent they spawned |
| Pattern | Buggy agent template across all customers | Bulk-revoke by `client_id` pattern |
| Vault | Customer's external OAuth credential is compromised | Vault disconnect cascades to derived agent tokens |

Layers 1-3 ship in v0.9. Layers 4-5 ship in v1.0.

### One ~40MB binary, no ceremony

```
$ ls -lh ./shark
-rwxr-xr-x  ~42M  ./shark
```

That includes the Go runtime, the SQLite engine, the embedded React admin UI, the migrations, the JWKS rotation worker, the proxy engine, and every SSO connector. Cross-compiles cleanly to darwin/linux/windows on amd64 and arm64.

### Battery-included admin UI

Every primitive has a screen. Every screen has a working revoke. Highlights: applications, agents, delegation canvas, delegation chain audit, vault manager, signing keys, RBAC matrix, organizations, sessions debugger, audit log, proxy config, branding, command palette. Source: `admin/src/components/`.

---

## Code samples

### Python: agent acts for a user, then delegates to a sub-agent

```python
from shark_auth import Client, DPoPProver

client = Client(base_url="http://localhost:8080", token="sk_live_...")
prover = DPoPProver.generate()

# 1. Agent gets a DPoP-bound access token (cnf.jkt locked to its key)
token = client.oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="research-agent",
    client_secret="...",
    scope="mcp:write",
)

# 2. Agent exchanges that token down to a narrower scope for a sub-agent.
#    The act chain is recorded server-side. max_hops on the may_act grant
#    decides whether this is allowed.
sub_token = client.oauth.token_exchange(
    subject_token=token.access_token,
    scope="mcp:read",
    dpop_prover=prover,
)

# 3. Every call to the resource carries a fresh DPoP proof JWT.
resp = client.http.get_with_dpop(
    "/resource",
    token=sub_token.access_token,
    prover=prover,
)
```

### TypeScript: same flow, browser-safe primitives

```typescript
import { SharkClient, DPoPProver } from "@sharkauth/sdk";

const client = new SharkClient({ baseUrl: "http://localhost:8080", adminKey: "sk_live_..." });
const prover = await DPoPProver.generate();

const token = await client.oauth.getTokenWithDpop({
  grantType: "client_credentials",
  dpopProver: prover,
  clientId: "research-agent",
  clientSecret: "...",
  scope: "mcp:write",
});

const subToken = await client.oauth.tokenExchange({
  subjectToken: token.accessToken,
  scope: "mcp:read",
  dpopProver: prover,
});
```

### curl: prove it works without an SDK

```bash
# Issue a may_act grant: research-agent may act for alice, max 2 hops.
curl -X POST http://localhost:8080/admin/may-act-grants \
  -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"from_id":"research-agent","to_id":"alice","max_hops":2,"scopes":["mcp:read","mcp:write"],"expires_at":"2026-12-31T00:00:00Z"}'

# Inspect the issuer metadata. Look for token_endpoint, exchange grant types,
# DPoP signing alg list, and the JWKS URL.
curl -s http://localhost:8080/.well-known/openid-configuration | jq
```

Full SDK docs: [`documentation/sdk/index.md`](documentation/sdk/index.md). Delegation deep-dive: [`documentation/sdk/delegation-and-agents.md`](documentation/sdk/delegation-and-agents.md). DPoP primitives: [`documentation/sdk/dpop.md`](documentation/sdk/dpop.md). Token exchange: [`documentation/sdk/token-exchange.md`](documentation/sdk/token-exchange.md).

---

## Comparison

|                                    | SharkAuth | Auth0 | Clerk | Keycloak |
|:-----------------------------------|:---------:|:-----:|:-----:|:--------:|
| Agent as first-class identity      | yes       | no    | no    | no       |
| RFC 8693 token exchange + `act`    | yes       | partial (paid tier) | no | partial (extension) |
| `may_act_grants` table, revocable  | yes       | no    | no    | no       |
| `max_hops` enforced server-side    | yes       | no    | no    | no       |
| RFC 9449 DPoP                      | yes       | no    | no    | partial  |
| `grant_id` correlated audit        | yes       | no    | no    | no       |
| Cascade revocation (user → agents) | yes       | no    | no    | no       |
| Single binary, ~40MB               | yes       | no    | no    | no (JVM, ~1GB image) |
| Self-hostable, fully OSS           | yes       | no    | no    | yes      |
| Human auth (SSO, MFA, passkeys)    | yes       | yes   | yes   | yes      |
| Polished hosted-tier UX            | not yet   | yes   | yes   | no       |
| Per-MAU pricing on the hosted tier | n/a       | yes   | yes   | n/a      |

Where Auth0 and Clerk win today: bigger ecosystems, polished hosted UX, mature SDKs across more languages, real customer support. SharkAuth is not trying to replace them for a marketing-site login form. It is for products that ship agents and need both halves of the trust chain in the same audit log.

---

## Use cases

- **Personal AI assistant on your inbox.** Issue a `may_act` grant from `you` to `assistant-agent`, scope `gmail:read`, expires in 24h. Revoke the grant from the admin UI when you stop trusting it.
- **Multi-agent orchestrator.** Planner agent delegates to a research agent, research agent delegates to a tool agent. `max_hops=3`, scopes downscoped at each hop, every hop recorded.
- **Internal employee + agent SSO.** Employees log in via OIDC federation or SAML. Each employee spawns one or more agents that inherit a scoped subset of their authority.
- **MCP server auth.** Register the MCP server as an application, issue per-consumer client credentials, gate tool calls behind DPoP-bound scoped tokens.
- **Embedded auth for an open-source SaaS.** Drop the binary next to your app, point your app at it for OAuth + OIDC + admin UI. No tenant of your tenants ends up paying Auth0 retail.
- **Compliance-grade audit for an existing OAuth deployment.** Run the proxy in front of your APIs, get DPoP enforcement and `grant_id` audit on top of identities you already have.

---

## Status: v0.9.0 (launch week, 2026-04-29)

What works today:

- OAuth 2.1: auth code + PKCE, client credentials, device flow (RFC 8628), refresh rotation
- OIDC: discovery, JWKS rotation, ID tokens, userinfo
- Token exchange with `act` chain (RFC 8693)
- DPoP (RFC 9449) on token issuance + protected resources
- Dynamic Client Registration (RFC 7591/7592)
- `may_act_grants` table, admin CRUD, audit correlation by `grant_id`
- Human auth: password, MFA/TOTP, recovery codes, magic link, passkeys/WebAuthn
- SSO: SAML 2.0, OIDC federation
- Organizations + RBAC matrix
- Token vault (Google, GitHub verified end-to-end; Slack, Microsoft, Notion experimental)
- Reverse proxy mode with identity header injection
- Audit log, webhooks with signature verification, branding, hosted login pages
- 252 backend routes wired into the chi router (`internal/api/router.go`)
- 29 migrations, all idempotent, applied on first boot
- Python SDK + TypeScript SDK, ~75% endpoint coverage on the agent platform + OAuth core (`documentation/sdk/index.md`)
- React component library: `@sharkauth/react`
- Embedded React admin UI, ~50 screens

What is on the v1.0 list (next 2-4 weeks):

- Revocation layers 4 + 5 (pattern bulk-revoke, vault disconnect cascade)
- Auth flow builder UI
- Vault custom-provider quirks (Linear, Jira, Slack rotating refresh)
- Hosted/cloud tier
- Public Postman + OpenAPI 3.1 spec polish

This README documents what ships today. Nothing aspirational.

---

## Install

```bash
# Binary
curl -fsSL https://github.com/sharkauth/sharkauth/releases/latest/download/shark-$(uname -s)-$(uname -m) -o shark
chmod +x shark
./shark serve

# Docker
docker run -p 8080:8080 -v $(pwd)/data:/data ghcr.io/sharkauth/sharkauth:latest

# Build from source
git clone https://github.com/sharkauth/sharkauth
cd sharkauth/admin && pnpm install && pnpm build && cd ..
go build -o shark ./cmd/shark
./shark serve
```

Requires Go 1.22+ and Node 20+ to build from source.

---

## Docs and community

- SDK index: [`documentation/sdk/index.md`](documentation/sdk/index.md)
- Delegation guide: [`documentation/sdk/delegation-and-agents.md`](documentation/sdk/delegation-and-agents.md)
- Token exchange: [`documentation/sdk/token-exchange.md`](documentation/sdk/token-exchange.md)
- DPoP primitives: [`documentation/sdk/dpop.md`](documentation/sdk/dpop.md)
- Vault: [`documentation/sdk/vault.md`](documentation/sdk/vault.md)
- OpenAPI 3.1 spec: `documentation/openapi.yaml`
- Issues: https://github.com/sharkauth/sharkauth/issues
- Discussions: https://github.com/sharkauth/sharkauth/discussions

If you are an early integrator and want help wiring it up, open an issue. There is exactly one person at the wheel and they answer.

---

## License

MIT. See [LICENSE](LICENSE).

<p align="center">
Built by <a href="https://github.com/raulgooo">Raúl</a> in Monterrey, Mexico.<br>
If your product ships agents, the auth stack starts here.
</p>
