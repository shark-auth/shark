/**
 * Branding design tokens admin API (v1.5).
 *
 * Stores and retrieves a free-form JSON design-token object that drives
 * the paywall + hosted-login color/typography palette.
 *
 * Routes:
 *   GET   /api/v1/admin/branding              (read full branding row)
 *   PATCH /api/v1/admin/branding/design-tokens (overwrite design tokens)
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/**
 * Free-form design token map. The server stores whatever shape is sent;
 * the recommended structure is shown below but any JSON object is accepted.
 *
 * @example
 * ```ts
 * {
 *   colors: { primary: "#6366f1", background: "#ffffff" },
 *   typography: { font_family: "Inter, sans-serif" },
 *   spacing: { unit: "4px" },
 *   motion: { duration_ms: 150 },
 * }
 * ```
 */
export type DesignTokens = Record<string, unknown>;

/** Branding row as returned by GET /api/v1/admin/branding. */
export interface BrandingRow {
  id: string;
  app_id?: string;
  primary_color?: string;
  secondary_color?: string;
  font_family?: string;
  logo_url?: string;
  /** Design tokens blob — may be `{}` when not yet set. */
  design_tokens: DesignTokens;
  created_at: string;
  updated_at: string;
  [key: string]: unknown;
}

/** Response envelope from `setBranding`. */
export interface SetBrandingResult {
  /** Full branding row after the update. */
  branding: BrandingRow;
  /** The design tokens object echoed back. */
  design_tokens: DesignTokens;
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link BrandingClient}. */
export interface BrandingClientOptions {
  /** Base URL of the SharkAuth server. */
  baseUrl: string;
  /** Admin API key (Bearer token). */
  adminKey: string;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Admin client for branding design tokens.
 */
export class BrandingClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: BrandingClientOptions) {
    this._base = opts.baseUrl.replace(/\/+$/, "");
    this._key = opts.adminKey;
  }

  private _auth(): Record<string, string> {
    return { Authorization: `Bearer ${this._key}` };
  }

  private async _throw(status: number, text: string): Promise<never> {
    let code = "api_error";
    let message = text.slice(0, 300);
    try {
      const body = JSON.parse(text) as {
        error?: { code?: string; message?: string };
      };
      if (body.error?.code) code = body.error.code;
      if (body.error?.message) message = body.error.message;
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  /**
   * Fetch the current branding row (includes design tokens if set).
   *
   * @param appSlug  Optional application slug; omit for the global row.
   */
  async getBranding(appSlug?: string): Promise<BrandingRow> {
    let url = `${this._base}/api/v1/admin/branding`;
    if (appSlug) {
      url += `?app_slug=${encodeURIComponent(appSlug)}`;
    }
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<{ data: BrandingRow }>();
    return body.data;
  }

  /**
   * Overwrite the design tokens for the global branding row.
   *
   * This is a full replace: GET → merge client-side → call this method if you
   * need partial updates (the server does not deep-merge).
   *
   * Passing `null` or `{}` clears the token blob.
   */
  async setBranding(tokens: DesignTokens): Promise<SetBrandingResult> {
    const url = `${this._base}/api/v1/admin/branding/design-tokens`;
    const resp = await httpRequest(url, {
      method: "PATCH",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify({ design_tokens: tokens }),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<{ data: SetBrandingResult }>().data;
  }
}
