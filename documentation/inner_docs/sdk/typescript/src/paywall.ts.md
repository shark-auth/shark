# paywall.ts

**Path:** `sdk/typescript/src/paywall.ts`
**Type:** Helper — paywall URL builder + HTML renderer
**LOC:** 119

## Purpose
The paywall is the public, server-rendered upsell page shown when a caller's tier doesn't satisfy a `tier_match` rule. This client builds its URL and (optionally) fetches the HTML for dashboard preview panes.

## Public API
- `class PaywallClient`
  - `constructor(opts: PaywallClientOptions)` — only `baseUrl`; no auth
  - `paywallURL(opts: PaywallOptions): string` — pure builder, no network
  - `renderPaywall(opts: PaywallOptions): Promise<string>` — GET, returns HTML
  - `previewPaywall(opts: PaywallPreviewOptions): Promise<string>` — `format: "url" | "html"`

## PaywallOptions
- `appSlug: string` — branding owner
- `tier: string` — denied rule's required tier
- `returnUrl?: string` — post-payment redirect

## URL shape
`${baseUrl}/paywall/{appSlug}?tier=<tier>[&return=<url>]`

## Constructor options
- `baseUrl: string` — trailing slashes stripped

## Error mapping
- Non-200 from `renderPaywall` → `SharkAPIError(text, "paywall_error", status)`.

## Internal dependencies
- `http.ts`, `errors.ts`

## Notes
- The paywall endpoint is **public** — no admin key or bearer token attached.
- `previewPaywall({ format: "url" })` short-circuits to the synchronous builder (still returns Promise).
- Designed for `<iframe>` embedding in the admin dashboard preview pane.
