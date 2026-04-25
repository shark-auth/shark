// Compile-time feature flags for the admin UI.
//
// BILLING_UI: gates the tier/paywall surfaces in the admin dashboard.
// Backend tier wiring (JWT.Tier, rules engine, /paywall/{slug}) ships
// regardless — only the admin-facing controls are hidden so the v0.9.0
// launch does not signal a paid offering before pricing is finalised.
// Re-enable post-launch once Stripe + MAU metering land and the tier
// list (Starter/Growth/Scale) is server-driven.
export const BILLING_UI = false;
