/**
 * Magic link — send a sign-in link via email.
 *
 * Route (public — no admin key required):
 *   POST /api/v1/auth/magic-link/send
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Response from `sendMagicLink`. */
export interface SendMagicLinkResult {
  /** Server-supplied message (e.g. "The magic link has been sent"). */
  message: string;
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link MagicLinkClient}. */
export interface MagicLinkClientOptions {
  /** Base URL of the SharkAuth server. */
  baseUrl: string;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Send magic-link sign-in emails.
 *
 * @example
 * ```ts
 * const ml = new MagicLinkClient({ baseUrl: "https://auth.example.com" });
 * await ml.sendMagicLink("user@example.com");
 * ```
 */
export class MagicLinkClient {
  private readonly _base: string;

  constructor(opts: MagicLinkClientOptions) {
    this._base = opts.baseUrl.replace(/\/+$/, "");
  }

  private async _throw(status: number, text: string): Promise<never> {
    let code = "api_error";
    let message = text.slice(0, 300);
    try {
      const body = JSON.parse(text) as {
        error?: string | { code?: string; message?: string };
        message?: string;
      };
      if (typeof body.error === "string") code = body.error;
      else if (body.error?.code) code = body.error.code;
      if (body.message) message = body.message;
      else if (typeof body.error === "object" && body.error?.message)
        message = body.error.message;
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  /**
   * Send a magic-link email to the given address.
   *
   * The server applies per-email rate limiting (1 per 60 s). Always returns
   * success to avoid leaking information about account existence.
   *
   * @param email        Recipient email address.
   * @param redirectUri  Optional redirect URI embedded in the magic link
   *                     (must be on the server's allowlist).
   *
   * @example
   * ```ts
   * await ml.sendMagicLink("user@example.com", "https://app.example.com/auth/callback");
   * ```
   */
  async sendMagicLink(
    email: string,
    redirectUri?: string,
  ): Promise<SendMagicLinkResult> {
    const url = `${this._base}/api/v1/auth/magic-link/send`;
    const body: Record<string, string> = { email };
    if (redirectUri) body["redirect_uri"] = redirectUri;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<SendMagicLinkResult>();
  }

  /**
   * Verify a magic-link token (companion to {@link sendMagicLink}).
   *
   * Wraps `GET /api/v1/auth/magic-link/verify?token=...`. Returns the
   * authenticated user object on success. The server plants a session
   * cookie which the browser captures automatically; Node callers using
   * the {@link AuthClient} cookie jar should prefer `AuthClient.verifyMagicLink`
   * so the cookie is reused for follow-up calls.
   */
  async verify(token: string): Promise<Record<string, unknown>> {
    const qs = new URLSearchParams({ token }).toString();
    const url = `${this._base}/api/v1/auth/magic-link/verify?${qs}`;
    const resp = await httpRequest(url, { method: "GET" });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<Record<string, unknown>>();
  }
}
