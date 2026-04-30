# SharkAuth Copy Grounding Guide

This document provides the foundational facts, technical specifications, and product positioning for SharkAuth. Use this as the "Source of Truth" when writing landing page copy, documentation, or marketing materials.

---

## 1. Core Identity
- **Definition**: SharkAuth is an open-source identity and authentication platform purpose-built for AI agents and human users.
- **Tagline**: "The first open-source identity platform built for the agentic world."
- **Primary Value Prop**: It bridges the gap between human identity and autonomous agent authority.
- **License**: MIT (Open Source).
- **Architecture**: A single, lightweight Go binary (~40MB) with an embedded SQLite database. **Zero infrastructure dependencies.**

## 2. Technical Facts (The "No-Hallucination" Zone)
- **Programming Language**: Written in Go (Golang).
- **Database**: Embedded SQLite by default (WAL mode). **Fact**: It does NOT require Postgres or Redis.
- **Cryptography**: 
    - Password Hashing: Argon2id (Auth0-compatible migration supported).
    - Signing: ES256 (ECDSA P-256) for OAuth2 tokens.
    - Encryption: AES-256-GCM for Token Vault.
- **Standards Implemented**:
    - **OAuth 2.1**: Modern secure defaults (no implicit flow).
    - **DPoP (RFC 9449)**: Cryptographic binding of tokens to agent keys.
    - **Token Exchange (RFC 8693)**: Native agent-to-agent delegation with `act` claims.
    - **Passkeys (FIDO2)**: WebAuthn support for humans.

## 3. Segment-Specific Grounding

### A. For YC & Angel Investors (The "Venture" Hook)
- **The Shift**: In 2026, software doesn't just have users; it has agents. Existing auth (Auth0/Clerk) treats agents as "Machine-to-Machine" (M2M) with no lineage.
- **The Moat**: **Lineage-as-a-Service.** Every agent action traces back to the human who authorized it.
- **Blast Radius Control**: If an agent is prompt-injected, you can revoke *just that agent's access* without killing the human's session.
- **Economic Value**: Reduces liability for companies shipping autonomous agents. Prevents "agent-driven data exfiltration."

### B. For OSS Contributors (The "Builder" Hook)
- **The Tech Stack**: Clean, idiomatic Go. 100k+ lines of code implementing complex RFCs correctly.
- **Simplicity**: `go install` and you have a full OIDC IdP. No Docker-compose hell.
- **Hackability**: Everything is in one binary. SQLite means contributors don't need to set up a DB cluster to help.
- **Mission**: Replacing the "proprietary auth tax" with a better, faster, open-source alternative.

### C. For SaaS Teams (The "Product" Hook)
- **One Trust Chain**: Use one platform for your "Login with Google" AND your "Agent-to-Agent Delegation."
- **Token Vault**: Stop building custom Slack/GitHub/Google integration logic. Shark handles the tokens, the refresh, and the encryption.
- **Audit Logs**: Compliance-ready audit trails that make sense in a multi-agent world.

## 4. The "Translation Layer" (RFCs to English)

When writing for non-protocol experts, use these analogies to explain the "Why":

| Technical Spec | The "Plain English" Analogy | Value Proposition |
| :--- | :--- | :--- |
| **DPoP (RFC 9449)** | **The "ID-Verified Credit Card"** | Standard tokens are like cash; if you drop them, anyone can spend them. DPoP tokens are locked to the agent's unique signature. Stolen tokens are useless. |
| **Token Exchange (RFC 8693)** | **The "Valet Key"** | Instead of giving an agent your "Master Key" (full access), you give them a limited "Valet Key" that only works for a specific task (e.g., "Book this flight") and expires quickly. |
| **`act` Claims** | **The "Notarized Power of Attorney"** | A transparent record of who authorized whom. It's not just "Agent X is acting," it's "Agent X is acting for User Y, authorized on Z date." |
| **Single Binary** | **"App Store Simplicity"** | No complex setup. If you can run a file, you can run a production auth server. No "assembly required." |

## 5. Problem / Solution Pairs
... (rest of section) ...
## 5. Hallucination Guard (What SharkAuth IS NOT)
- **NOT a wrapper around Auth0/Clerk**: It is a full, ground-up implementation of OAuth 2.1.
- **NOT just a dev tool**: It's built for production (Audit logs, MFA, SSO, RBAC).
- **NOT limited to SQLite**: While SQLite is default, it's designed for high-concurrency (WAL mode).

## 6. Tone & Voice
- **Direct & Technical**: Lead with RFCs and specs to show authority.
- **No Fluff**: Focus on "shipped code" and "single binary."
- **Future-Proof**: Frame it as "Auth for the Agentic Era."

---
*Grounding data last verified: April 30, 2026*
