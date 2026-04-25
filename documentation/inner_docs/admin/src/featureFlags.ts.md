# featureFlags.ts

**Path:** `admin/src/featureFlags.ts`

Compile-time feature flags for the admin dashboard. One exported constant per flag — tree-shaken at build time; no runtime config file needed.

## Exports

| Flag | Type | Default | Purpose |
|------|------|---------|---------|
| `BILLING_UI` | `boolean` | `false` | Gates tier/paywall admin surfaces |

## BILLING_UI

When `false` (v0.9.0 default):
- Tier column hidden from user list
- `TierSection` in user detail panel hidden
- Paywall-preview button hidden from Branding → Design Tokens
- Per-user tier badge hidden from overview cards

**Backend is unaffected.** `JWT.Tier`, the proxy rules engine ReqTier enforcement, and `GET /paywall/{slug}` all remain wired and functional. The flag hides only the admin-facing controls to avoid signalling a paid offering before pricing is finalised.

## Re-enable path

1. Set `BILLING_UI = true`
2. Confirm Stripe + MAU metering is live
3. Verify tier list (Starter/Growth/Scale) is server-driven via `GET /api/v1/admin/config` or similar
4. Rebuild admin bundle (`cd admin && npm run build`)
