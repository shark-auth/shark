# hosted_handlers.go

**Path:** `internal/api/hosted_handlers.go`
**Package:** `api`
**LOC:** 567
**Tests:** likely integration-tested

## Purpose
Renders the hosted-auth SPA HTML shell at `/hosted/{app_slug}/{page}`, the v1.5 paywall page at `/paywall/{app_slug}`, and serves the embedded JS bundle assets at `/admin/hosted/assets/*`. The shell inlines branding CSS vars, injects `window.__SHARK_HOSTED` config, and loads the immutable-cached JS bundle.

## Handlers exposed
- `handleHostedPage` (line 124) — GET `/hosted/{app_slug}/{page}`. Validates page (allowlist: `login|signup|magic|passkey|mfa|verify|error`), resolves app by slug, checks `IntegrationMode in {hosted, proxy}`, resolves merged branding, builds `hostedConfig` (app, branding, authMethods, oauthProviders, oauth params), renders shell. Supports `?preview=true` live-preview overrides for primary/secondary colors.
- `handlePaywallPage` (line 306) — GET `/paywall/{app_slug}`. Renders v1.5 upgrade page with sanitized tier label + return URL + upgrade href.
- `handleHostedAssets` (line 495) — GET `/admin/hosted/assets/*`. Serves embedded SPA bundle from `internal/admin.DistFS()`.

## Key types
- `hostedAppInfo` (line 82), `hostedBrandingInfo` (line 89), `hostedOAuthProvider` (line 97), `hostedOAuthParams` (line 105), `hostedConfig` (line 114)

## Helpers
- `findHostedBundle` (line 59) — locates `hosted-*.js` in embedded FS at init time; cached as `hostedBundleName`.
- `resolveAuthMethods` (line 525), `resolveOAuthProviders` (line 531)
- `sanitizeCSSValue` (line 557), `sanitizeTierLabel` (line 436), `sanitizeReturnURL` (line 451), `buildUpgradeHref` (line 474), `escapeQueryValue` (line 485)

## Package state
- `validHostedPages` (line 26, set), `cssColorRE` (line 39, regexp), `hostedBundleName` (line 44).

## Imports of note
- `internal/admin` — embedded SPA FS via `admin.DistFS()`
- `internal/storage` — `GetApplicationBySlug`, `ResolveBranding`

## Wired by
- `internal/api/router.go:756-757`, `:763`

## Notes
- Bundle is resolved once at `init()`; missing bundle logs a warning and serves a degraded shell that still loads.
- Paywall + hosted are public (no auth middleware).
