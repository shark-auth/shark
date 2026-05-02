<p align="center">
  <img src="admin/src/assets/sharky-full.png" width="220" alt="SharkAuth" />
</p>

<h1 align="center">SharkAuth</h1>

<p align="center">
  <strong>The open-source identity platform built for AI agents.</strong><br />
  One ~29 MB binary. OAuth 2.1, OIDC, RFC 8693 Token Exchange, and DPoP. Zero config.
</p>

<p align="center">
  <a href="https://github.com/shark-auth/shark/releases/latest"><img src="https://img.shields.io/badge/version-v0.1.0-blue?style=flat-square" alt="Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License"></a>
  <a href="#"><img src="https://img.shields.io/badge/go-1.22%2B-00ADD8?style=flat-square" alt="Go"></a>
  <a href="#"><img src="https://img.shields.io/badge/SQLite-embedded-003B57?style=flat-square" alt="SQLite"></a>
</p>

---

## Auth was built for humans clicking buttons. Your agents need something better.

When an AI agent delegates to a sub-agent, the trust chain breaks. Bearer tokens leak. Revocation becomes a mess. Auditors ask "which agent did what?" and you have no answer.

**SharkAuth fixes this.** It treats agents as first-class identities with native delegation, cryptographically bound tokens, and a unified audit trail that tracks every hop from user to resource.

### Install in 10 seconds

```bash
curl -fsSL sharkauth.com/get | sh

# Or with Go 1.22+
go install github.com/shark-auth/shark/cmd/shark@latest
```

```bash
# Boot. SQLite-embedded, zero config.
shark serve
# => admin UI : http://localhost:8080/admin
# => issuer   : http://localhost:8080
```

---

## Why teams choose SharkAuth

### 1. Agent Delegation That Actually Works

Real delegated authority using **RFC 8693 Token Exchange**. SharkAuth issues `may_act_grants` that are revocable, time-limited, and hop-constrained. No more prompt-level "trust me" delegation.

<p align="center">
  <img src="images/del_chains1.png" width="600" alt="Delegation Chain Audit" />
  <br /><em>Audit every hop: User → Researcher → Tool Agent.</em>
</p>

### 2. Tokens Bound to Keys, Not Bearers (RFC 9449 DPoP)

Bearer tokens are a liability. SharkAuth ships **Demonstrating Proof-of-Possession** by default. Every token is cryptographically bound to the agent's private key. Stolen via prompt injection or log leak? Useless without the key.

### 3. Complete Provenance in One Query

One `grant_id` correlates every token, every hop, and every resource touched. Reconstruct the full lifecycle of an agent's authority instantly. No more "the agent did it" dead ends.

### 4. One Binary. Zero Dependencies. Anywhere.

SharkAuth is a single static Go binary with embedded SQLite WAL. No Postgres, no Redis, no Docker, no Helm charts. It cold-starts in 38 ms and runs on a Raspberry Pi as easily as it runs in Kubernetes.

### 5. Open Source, Zero Lock-In (MIT)

100% open source. No telemetry, no vendor lock-in, no "free until we change our mind." Your data never leaves your infrastructure. Your auth stack is yours forever.

---

## What you get

| Category | Highlights |
| :--- | :--- |
| **Agent Auth** | RFC 8693 Token Exchange, RFC 9449 DPoP, `may_act_grants`, cascade revocation, full act chains (depth ≥ 7 observed) |
| **Human Auth** | Passkeys (FIDO2), Magic Links, MFA (TOTP), Enterprise SSO (SAML 2.0, OIDC), Argon2id passwords |
| **Platform** | Zero-Code Auth Proxy, Multi-tenant Orgs, Wildcard RBAC, HMAC-signed Webhooks, grant_id-indexed Audit Logs |
| **Admin UI** | React dashboard embedded in the binary. One-click revocation for every session, token, and grant |

---

## Get started

### Docker (fastest)

```bash
docker run -p 8080:8080 -v shark-data:/data ghcr.io/shark-auth/shark
```

### Dev mode (no config needed)

```bash
shark serve --dev
```

Magic links print to stdout. In-memory database. Perfect for rapid prototyping.

### TypeScript SDK

```typescript
import { SharkClient } from "@sharkauth/sdk";

const shark = new SharkClient({ baseUrl: "http://localhost:8080" });

// Sign in with DPoP protection
const session = await shark.signIn({ email: "alice@co.io", password: "..." });
```

### Python SDK

```python
from sharkauth import SharkClient

shark = SharkClient(base_url="http://localhost:8080")
session = shark.sign_in(email="alice@co.io", password="...")
```

**→ [Read the full docs](https://sharkauth.com/docs)**

---

## What's coming

- **Visual Flow Builder** — Drag-and-drop auth flows (MFA → SSO → Org Select)
- **Shark Cloud** — Managed infrastructure, free to enterprise. [Join the waitlist](https://sharkauth.com/waitlist)
- **Postgres Mode** — Optional external DB for planet-scale deployments
- **Shark Email** — Built-in delivery for magic links and MFA codes

---

## Community

- **Discord**: [discord.gg/sharkauth](https://discord.gg/sharkauth) — ask questions, share deployments
- **Twitter**: [@raulgooo](https://twitter.com/raulgooo) — updates and agent identity threads
- **Docs**: [sharkauth.com/docs](https://sharkauth.com/docs)
- **Issues & PRs**: [github.com/shark-auth/shark/issues](https://github.com/shark-auth/shark/issues)

Built by [Raúl](https://github.com/raulgooo) in Monterrey, Mexico. MIT licensed.

---

<p align="center">
  <strong>If your product ships agents, the auth stack starts here.</strong>
</p>

<p align="center">
  <a href="https://github.com/shark-auth/shark/stargazers">⭐ Star this repo</a> ·
  <a href="https://sharkauth.com/waitlist">☁️ Join Cloud Waitlist</a> ·
  <a href="https://discord.gg/sharkauth">💬 Join Discord</a>
</p>
