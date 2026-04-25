# Tier / Paywall — Dormant Backend, Hidden UI

**Status:** dormant — wired backend, UI hidden behind `BILLING_UI` flag.
**Decision date:** 2026-04-25 (4 days pre-launch).
**Owner of revival:** post-launch billing milestone (post Q3 2026 PostgreSQL unblock).

---

## What this doc explains

The shark binary contains a partially-wired tier/paywall system that ships
in v0.9.0 but is **not** customer-facing. This document records what is
wired, what is missing, and *why* it ships dormant rather than being ripped
out — so a future maintainer does not (a) mistake the dead UI for a bug
and re-enable it prematurely, or (b) rip out useful infra thinking it is
abandoned.

If you are looking at `users.tsx → TierSection`, `branding.tsx →
PaywallPreviewEditor`, or the `tier_match` field in `proxy_config.tsx`
and wondering "is this shipped?" — the answer is *no, not at launch*.

---

## Why dormant, not deleted

1. **Pricing is not finalised.** YC application narrative explicitly says
   "OSS free forever self-host; hosted pricing finalized during batch."
   See `gstack/05-revisions-cold-dm-kit.md` decision TD3.
2. **Cloud is blocked on Postgres support (Q3 2026).** Until cloud ships,
   no paying customer exists to gate. See `CLOUD.md` Phase-1 roadmap.
3. **The pricing-doc tier list does not match the implemented tier list.**
   `shark_pricing_philosophy.md` defines Starter / Growth / Scale (cloud)
   plus Self-Hosted. The backend models only `"free"` and `"pro"`.
   Shipping a `free | pro` toggle now would signal a price commitment we
   have not made.
4. **The free/pro engine wiring is genuinely correct and cheap to keep.**
   Ripping it would force a re-implementation in 3 months.

---

## What is wired (and stays wired)

| Layer | Component | File:Line |
|---|---|---|
| JWT mint | `Manager.mint` reads `users.metadata["tier"]`, defaults `"free"`, sets `claims.Tier` | `internal/auth/jwt/manager.go:48,62-66,89-92,114` |
| JWT path identity | `JWTResolver.Resolve` populates `Identity.Tier` from claims | `internal/api/proxy_resolvers.go:66-77` |
| Session-cookie path identity | `LiveResolver.Resolve` reads tier from `users.metadata` JSON | `internal/api/proxy_resolvers.go:142-167` |
| Rules engine | `tier:X` predicate emits `DecisionPaywallRedirect` on mismatch | `internal/proxy/rules.go:57-61, 442-458, 556-561` |
| Proxy HTTP | redirects `DecisionPaywallRedirect` → `/paywall/{slug}?tier=X&return=…` | `internal/proxy/proxy.go:248-261` |
| Paywall page | renders 402 upgrade page with brand tokens | `internal/api/hosted_handlers.go:293-430` |
| Router | `GET /paywall/{app_slug}` mounted | `internal/api/router.go:760-763` |
| Admin endpoint | `PATCH /admin/users/{id}/tier` writes metadata | `internal/api/router.go:699-701` + `internal/storage/user_tier.go` |
| CLI | `shark user tier`, `shark paywall preview` | `cmd/shark/cmd/user_tier.go`, `cmd/shark/cmd/paywall.go` |
| Migration | `tier_match` column on `proxy_rules` | `internal/testutil/migrations/00023_proxy_rules_tier_match.sql` |
| Tests | engine + paywall + CLI flow | `internal/proxy/rules_test.go:636`, `internal/api/hosted_handlers_test.go:293`, `cmd/shark/cmd/user_tier_test.go` |

This means an operator can — *via CLI or API* — define a proxy rule with
`tier_match: "pro"`, flip a user's tier with `shark user tier set <id> pro`,
and see the paywall redirect work end-to-end. Power users can. The admin
dashboard cannot, by design, until billing is real.

---

## What is hidden in the admin UI

Gated by `admin/src/featureFlags.ts → BILLING_UI = false`:

- `admin/src/components/users.tsx:931-935` — `TierSection` (per-user tier toggle)
- `admin/src/components/branding.tsx:73-75` — paywall preview surface tab
- `admin/src/components/branding.tsx:660` — `PaywallPreviewEditor` render
- `admin/src/components/proxy_config.tsx:742` — `tier:X` chip in rule list
- `admin/src/components/proxy_config.tsx:1000-1003` — `tier_match` form field

To revive: flip `BILLING_UI = true`. Visual surfaces reappear with no
backend change. Test the paywall preview in dev with
`shark paywall preview --slug <app>` first.

---

## What is NOT wired (and why we are not building it now)

| Capability | Status | Why deferred |
|---|---|---|
| Stripe / payment provider | absent | Cloud blocked on Postgres; no paying customer until Q3 2026 |
| MAU counting / overage metering | absent | Pricing doc says `$0.003/MAU overage` — meaningless without billing rail |
| Self-serve upgrade flow | absent | Paywall CTA carries `?upgrade=tier` but no checkout handler |
| Multi-tier model (Starter/Growth/Scale) | hardcoded `free|pro` | Pricing not finalised; expanding to 4 tiers would lock numbers prematurely |
| Server-driven tier list endpoint | absent | Frontend hardcodes `['free','pro']` — fix in same PR as Stripe wiring |
| Webhook → `SetUserTier("pro")` on `checkout.session.completed` | absent | Depends on Stripe |
| Cookie-path tier silent-fallback risk | **already fixed** — `LiveResolver` populates `tier` (audit miss; verify before re-asserting) | — |

---

## When to revive

Trigger conditions, in order:

1. PostgreSQL support lands (CLOUD.md Phase 2).
2. First cloud beta customer signs up via waitlist.
3. Stripe account + billing entity decisions are finalised (org name, EIN, tax setup).
4. Pricing doc is converted from philosophy to commitment (numbers locked).

Reasonable revival sequence:

1. Server-driven `GET /api/v1/billing/tiers` endpoint returning the tier
   list. Frontend reads from this — drop the hardcoded `['free','pro']`.
2. Stripe Checkout session creator + `checkout.session.completed` webhook
   that calls `storage.SetUserTier`.
3. Replace `TierSection` hardcoded list with API call. Add downgrade
   confirmation + force-logout button.
4. `PaywallPreviewEditor` — load tier list from same API; add error/loading
   states; verify `/paywall/{slug}` renders correctly post-Stripe link.
5. `proxy_config.tsx tier_match` — convert freetext to dropdown sourced
   from API. Add validation.
6. MAU counter (separate decision: per-app or global?). Wire to overage
   billing once Stripe is live.
7. Flip `BILLING_UI = true`, delete the flag and this doc.

---

## Pointers

- Pricing philosophy: `shark_pricing_philosophy.md` (root)
- Strategy + revised pricing: `STRATEGY.md` (root)
- Cloud roadmap: `CLOUD.md` (root)
- Locked decision TD3 (defer pricing): `gstack/05-revisions-cold-dm-kit.md`
- Engine semantics: `internal/proxy/rules.go` + `documentation/inner_docs/internal/proxy/rules.go.md`
- Storage tier API: `internal/storage/user_tier.go` + its inner doc
- Paywall renderer: `internal/api/hosted_handlers.go` + its inner doc
- CLI: `cmd/shark/cmd/user_tier.go.md`, `cmd/shark/cmd/paywall.go.md`
