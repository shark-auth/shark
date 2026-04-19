# SharkAuth

## What This Is

A single Go binary that handles authentication end-to-end: signup, login, sessions, OAuth, MFA, passkeys, magic links, RBAC, SSO, M2M API keys, audit logs — with Auth0 migration, an embedded Svelte admin dashboard, and a TypeScript SDK. Self-hosted for $0, cloud for ops convenience. `shark serve` and visit `:8080/admin`.

## Core Value

One binary replaces Auth0/Clerk at 98% less cost, with full feature parity on self-hosted. The binary is the product — cloud sells operational burden, not features.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Email/password signup + login (Argon2id hashing)
- [ ] Passkey/WebAuthn login (FIDO2-compliant, discoverable + non-discoverable)
- [ ] Magic link login (email-based passwordless)
- [ ] Server-side sessions (encrypted cookies, auth_method tracking)
- [ ] Social OAuth — Google, GitHub, Apple, Discord (generic handler pattern)
- [ ] MFA — TOTP with recovery codes (Google Authenticator compatible)
- [ ] RBAC (roles, permissions, middleware enforcement, /auth/check)
- [ ] SSO — OIDC provider (SharkAuth as IdP)
- [ ] SSO — SAML SP (beta — Okta/Azure AD)
- [ ] M2M API keys (scoped, rotatable, rate-limited)
- [ ] Audit logs (every auth event, filterable, exportable, retention-configurable)
- [ ] Auth0 user import (JSON export, bcrypt→argon2id transparent rehash)
- [ ] REST API for all features
- [ ] TypeScript SDK (`@sharkauth/js` — fetch-based, zero-dep, isomorphic)
- [ ] Admin dashboard (Svelte, embedded in binary)
- [ ] Automated test suite (Go unit + integration, SDK vitest, CI with gosec)
- [ ] YAML config with env var overrides
- [ ] SQLite storage (embedded, zero-config)
- [ ] Docker image (<30MB, one container)
- [ ] Landing page updates (sharkauth.com — pricing, features, comparison)

### Out of Scope

- Organizations/multi-tenancy — post-launch, highest priority after v1
- OIDC client federation — post-launch
- Agent identity / MCP auth — emerging standard, not table stakes yet
- Clerk/Firebase/Cognito migration — later
- Postgres mode — later
- React/Next.js component library — SDK first, pre-built UI follows

## Context

- **Existing code:** Partial Go scaffold exists (router, auth handlers, password hashing, sessions, config, DB layer, OAuth stubs, middleware). Spec v2 expands significantly beyond what's built.
- **Tech stack:** Go backend, SQLite (mattn/go-sqlite3), React 18 + Vite + TypeScript admin dashboard at `admin/src/` (embedded via go:embed at `internal/admin/dist/`), TypeScript SDK (tsup build). (Original spec called for Svelte; React was chosen during Phase 4 build.)
- **Key libraries:** go-webauthn/webauthn (passkeys), pquerna/otp (TOTP), gorilla/securecookie (sessions).
- **Target audience:** Developers who currently use Auth0/Clerk and are frustrated by per-MAU pricing or vendor lock-in. Also self-hosters who want full control.
- **Pricing philosophy:** Self-hosted = $0 unlimited. Cloud tiers: $19/$49/$149/mo. "You're paying for ops, not features" — the day that stops being true, we lose positioning.
- **Sprint:** 17 days (evenings 3h + three weekends 8-11h). Ship date: April 27, 2026.

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
*Last updated: 2026-04-05 after initialization*
