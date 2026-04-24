/**
 * Paywall helper (v1.5).
 *
 * The paywall is a public, server-rendered HTML page served by the proxy when
 * a caller's tier doesn't match the required tier. The SDK provides:
 *
 *   - `paywallURL`    — URL builder (no network call)
 *   - `renderPaywall` — fetches the rendered HTML string
 *   - `previewPaywall`— same as render but accepts a `format` flag
 *
 * Route (public, no auth):
 *   GET /paywall/{app_slug}?tier=<tier>&return=<url>
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Options for building or fetching a paywall page. */
export interface PaywallOptions {
  /** Application slug that owns the branding/palette. */
  appSlug: string;
  /** Tier the denied rule required, e.g. `"pro"`. */
  tier: string;
  /** URL to return the caller to after payment. Defaults to `/`. */
  returnUrl?: string;
}

/** Options for `previewPaywall`. */
export interface PaywallPreviewOptions extends PaywallOptions {
  /**
   * `"html"` (default) — returns rendered HTML string.
   * `"url"`  — returns only the composed URL, no network call.
   */
  format?: "html" | "url";
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link PaywallClient}. */
export interface PaywallClientOptions {
  /** Base URL of the SharkAuth server. */
  baseUrl: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function buildPaywallURL(base: string, opts: PaywallOptions): string {
  const qs = new URLSearchParams({ tier: opts.tier });
  if (opts.returnUrl) {
    qs.set("return", opts.returnUrl);
  }
  return `${base}/paywall/${encodeURIComponent(opts.appSlug)}?${qs.toString()}`;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Helper client for the paywall page.
 *
 * The paywall endpoint is public (no auth required); this client exposes
 * a URL builder and an HTML-fetching helper for dashboard preview use.
 */
export class PaywallClient {
  private readonly _base: string;

  constructor(opts: PaywallClientOptions) {
    this._base = opts.baseUrl.replace(/\/+$/, "");
  }

  /**
   * Build the paywall URL without making a network request.
   * Suitable for embedding in an `<iframe>` or opening in a new tab.
   */
  paywallURL(opts: PaywallOptions): string {
    return buildPaywallURL(this._base, opts);
  }

  /**
   * Fetch the rendered paywall HTML for the given options.
   * Returns the raw HTML string.
   */
  async renderPaywall(opts: PaywallOptions): Promise<string> {
    const url = buildPaywallURL(this._base, opts);
    const resp = await httpRequest(url, {
      headers: { Accept: "text/html" },
    });
    if (resp.status !== 200) {
      throw new SharkAPIError(
        resp.text.slice(0, 300),
        "paywall_error",
        resp.status
      );
    }
    return resp.text;
  }

  /**
   * Preview the paywall.
   *
   * - `format: "url"` — returns the composed URL string, no network call.
   * - `format: "html"` (default) — fetches and returns the HTML string.
   */
  async previewPaywall(opts: PaywallPreviewOptions): Promise<string> {
    if (opts.format === "url") {
      return buildPaywallURL(this._base, opts);
    }
    return this.renderPaywall(opts);
  }
}
