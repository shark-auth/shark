---

Phase 1 — Harden the foundation (2-3 days)

- Fix OIDC nonce validation (#42)
- Add request body limits (#43)
- Switch to slog (#46)
- Makefile + CI (#47)
- Tests across major components (#65) — start here, keep adding throughout

Ship this and you have a secure, testable, CI-gated codebase. Everything after  
 builds on solid ground.

Phase 2 — Make it usable (3-4 days)

- CLI subcommands (#53) — shark init && shark serve is the first thing anyone
  does
- Dev mode with email capture (#48) — removes SMTP friction for every developer
  trying Shark
- Admin stats endpoint (#44)
- Session management endpoints (#45)  


Now someone can install, run, and manage Shark from the terminal without reading
YAML docs.

Phase 3 — Dashboard (7-10 days)

- Svelte admin dashboard (existing DASHBOARD.md spec)
- Depends on: stats endpoint, session endpoints from Phase 2  


This is what turns Shark from an API into a product. The HN demo GIF comes from
this.

Phase 4 — SDK (5-7 days)

- TypeScript SDK (#54)
- React/Svelte/Vue wrappers  


Now developers can actually integrate. Without this, Shark is unusable for most
people.

Phase 5 — B2B unlock (7-10 days)

- Organizations (#50) — the #1 feature gap vs every competitor
- Webhooks (#51) — required for any real integration  


This is what converts "interesting OSS project" into "I can build my SaaS on  
 this."

Phase 6 — JWT + OAuth 2.1 infrastructure (5-7 days)

- Configurable session mode — cookie vs JWT (#67)
- JWKS endpoint, signing key management, key rotation
- This is the foundation that Agent Auth, OIDC Provider, and edge auth all share

Build this once, everything after uses it.

Phase 7 — Agent Auth (10-14 days)

- Full OAuth 2.1 server (#57) — client credentials, auth code + PKCE, device
  flow, token exchange, DCR
- Token Vault (#66) — managed third-party OAuth
- This is the headline feature. "First OSS auth with native MCP agent support."

Phase 8 — Proxy + OIDC Provider (5-7 days)

- Shark Proxy (#58) — shark proxy --upstream makes Shark usable without any SDK
- OIDC Provider mode (#60) — shares OAuth 2.1 infrastructure from Phase 7  


Phase 9 — Polish & enterprise (7-10 days)

- Impersonation (#59)
- Compliance toolkit (#61)
- Email provider presets (#52) + shark.email relay (#56)
- docs_url in error responses (#49)
- Migration tools — Auth0 (#55), Clerk (#63), Supabase (#64)
- Pre-built UI components + dashboard editor (#68)  


Phase 10 — Moonshot (when ready)

- Visual flow builder (#62)
- Remaining cloud-only issues (#27, #29, #30, #31)  


---

The critical path: Phases 1-4 are launch. You can post on HN after Phase 4.
Phases 5-7 are what make Shark dominant. Phase 5 is what keeps users. Phase 7 is
what makes news.

Start Phase 1 tomorrow. Ship daily. Go.
