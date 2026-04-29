# SharkAuth

## What This Is

A single Go binary that handles authentication end-to-end: signup, login, sessions, OAuth 2.1, MFA, passkeys, magic links, RBAC, SSO, M2M API keys, audit logs, agent delegation, token vault, reverse proxy — with an embedded React admin dashboard, TypeScript + Python SDKs, and a full CLI. Self-hosted for $0, cloud for ops convenience. `shark serve` and visit `:8080/admin`.

## Core Value

One binary replaces Auth0/Clerk at 98% less cost, with full feature parity on self-hosted. The binary is the product — cloud sells operational burden, not features.

## Requirements

### Validated (v0.9.0 — target ship 2026-04-29)

- [x] Email/password signup + login (Argon2id hashing)
- [x] Passkey/WebAuthn login (FIDO2-compliant, discoverable + non-discoverable)
- [x] Magic link login (email-based passwordless)
- [x] Server-side sessions (encrypted cookies, auth_method tracking)
- [x] Social OAuth — Google, GitHub, Apple, Discord
- [x] MFA — TOTP with recovery codes (Google Authenticator compatible)
- [x] RBAC (roles, permissions, middleware enforcement, /auth/check)
- [x] SSO — OIDC provider (SharkAuth as IdP) + SAML SP (Okta/Azure AD)
- [x] M2M API keys (scoped, rotatable, rate-limited)
- [x] Audit logs (filterable, exportable, retention-configurable)
- [x] Auth0 bcrypt→Argon2id transparent rehash on login
- [x] REST API for all features (~200 endpoints, OpenAPI spec at `/api/docs`)
- [x] TypeScript SDK (`@sharkauth/js`) + Python SDK (`sharkauth-py`)
- [x] Admin dashboard (React, embedded in binary)
- [x] OAuth 2.1 AS — auth code+PKCE, client credentials, DPoP, DCR, device flow, token exchange, introspection, revocation
- [x] Agent delegation (RFC 8693 token exchange, first-class agent identities)
- [x] Token vault (AES-256-GCM, Google/Slack/GitHub/Microsoft/Notion/Linear/Jira)
- [x] Reverse proxy with identity header injection and route rules engine
- [x] Auth flow engine (post-auth pipelines, 12 step types)
- [x] Full CLI (`shark user/sso/api-key/agent/session/audit/debug/admin/consents/org/vault/auth`)
- [x] YAML config with env var overrides
- [x] SQLite storage (embedded, zero-config)
- [x] Goreleaser + PyPI + npm distribution plan

### Active (v1.0 target)

- [ ] Flow builder dashboard UI (backend wired; UI cut at v0.9.0)
- [ ] Postgres mode
- [ ] SCIM provisioning
- [ ] React/Next.js component library (SDK hooks shipped; pre-built UI deferred)
- [ ] Hosted SharkAuth Cloud (blocked on Postgres, Q3 2026)

### Out of Scope (v0.9.0)

- Auth0 migration importer UI (bcrypt rehash works; bulk JSON import endpoint cut)
- Dashboard flow builder UI (API fully wired; UI deferred to v1.0)
- Tier/paywall UI (`BILLING_UI=false` flag; backend wired, UI hidden)

## Context

- **Current state:** ~38/54 capabilities shipped. 375+ smoke tests passing. v0.9.0 target 2026-04-29. Distribution via Goreleaser (binaries), PyPI (`sharkauth-py`), npm (`@sharkauth/js`).
- **Tech stack:** Go backend, SQLite (modernc/sqlite pure-Go), React 18 + Vite + TypeScript admin dashboard at `admin/src/` (embedded via go:embed at `internal/admin/dist/`), TypeScript SDK + Python SDK.
- **Key libraries:** go-webauthn/webauthn (passkeys), pquerna/otp (TOTP), gorilla/securecookie (sessions), ory/fosite (OAuth 2.1 AS).
- **Target audience:** Developers frustrated by Auth0/Clerk per-MAU pricing or vendor lock-in. Self-hosters who want full control. Agent/MCP developers needing delegation.
- **Pricing philosophy:** Self-hosted = $0 unlimited. Cloud tiers TBD (finalized Q3 2026 with partners). "You're paying for ops, not features."
- **Ship date:** v0.9.0 on 2026-04-29. v1.0 gated on two weeks of partner usage without breaking changes.

## Constraints

- **Timeline**: Ship by April 27, 2026 — 17 working days of evenings + weekends
- **Solo dev**: One person building everything (Go, Svelte, TypeScript, docs, landing page)
- **Single binary**: Everything must embed into one Go binary — no separate frontend deploy, no external dependencies at runtime
- **SQLite only**: No Postgres at launch. Zero-config is the selling point.
- **Feature parity**: Every feature must work on self-hosted. Cloud cannot gate features.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Argon2id for password hashing | Industry standard, better than bcrypt for new systems | — Pending |
| Server-side sessions over JWT | Simpler, revocable, no token refresh complexity | — Pending |
| SQLite over Postgres for v1 | Zero-config aligns with single-binary philosophy | — Pending |
| Passkeys bypass MFA | FIDO2 is phishing-resistant AAL2 per NIST SP 800-63-4 | — Pending |
| Generic OAuth handler pattern | One handler, provider registry — avoids per-provider sprawl | — Pending |
| Svelte for dashboard | Lightweight, compiles to small bundle, embeds well in Go | — Pending |
| sk_live_ prefix for API keys | Identifiable in logs and secret scanners | — Pending |
| Cursor-based pagination for audit logs | Better performance than offset for append-only tables | — Pending |
| 60% test coverage target | Realistic for sprint timeline, covers critical auth paths | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition:**
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone:**
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-25 — v0.9.0 pre-launch doc sweep*
