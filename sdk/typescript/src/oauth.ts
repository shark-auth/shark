/**
 * OAuth 2.1 token utilities — revocation (RFC 7009) and introspection (RFC 7662).
 *
 * Routes (no auth required beyond the token itself):
 *   POST /oauth/revoke      — RFC 7009 token revocation
 *   POST /oauth/introspect  — RFC 7662 token introspection
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Token type hint for revocation. */
export type TokenTypeHint = "access_token" | "refresh_token";

/** Response from `introspectToken`. Active tokens include claims; inactive return `{ active: false }`. */
export interface IntrospectResult {
  /** Whether the token is currently active. */
  active: boolean;
  /** Token scope (if active). */
  scope?: string;
  /** Client ID that obtained the token (if active). */
  client_id?: string;
  /** Subject (user or agent ID, if active). */
  sub?: string;
  /** Expiry Unix timestamp (if active). */
  exp?: number;
  /** Issuer (if active). */
  iss?: string;
  /** Token type (if active). */
  token_type?: string;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link OAuthClient}. */
export interface OAuthClientOptions {
  /** Base URL of the SharkAuth server. */
  baseUrl: string;
  /** Admin API key (Bearer token) used for introspection. */
  adminKey?: string;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Client for RFC 7009 token revocation and RFC 7662 token introspection.
 *
 * @example
 * ```ts
 * const oauth = new OAuthClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_..." });
 *
 * // Revoke a token
 * await oauth.revokeToken("my_access_token");
 *
 * // Introspect a token
 * const info = await oauth.introspectToken("my_access_token");
 * if (info.active) console.log("token belongs to", info.sub);
 * ```
 */
export class OAuthClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: OAuthClientOptions) {
    this._base = opts.baseUrl.replace(/\/+$/, "");
    this._key = opts.adminKey ?? "";
  }

  private async _throw(status: number, text: string): Promise<never> {
    let code = "api_error";
    let message = text.slice(0, 300);
    try {
      const body = JSON.parse(text) as {
        error?: string | { code?: string; message?: string };
        error_description?: string;
      };
      if (typeof body.error === "string") code = body.error;
      else if (body.error?.code) code = body.error.code;
      if (body.error_description) message = body.error_description;
      else if (typeof body.error === "object" && body.error?.message)
        message = body.error.message;
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  /**
   * Revoke a token (RFC 7009).
   *
   * The server always returns 200 regardless of whether the token existed,
   * to avoid leaking information.
   *
   * @param token        The access or refresh token to revoke.
   * @param tokenTypeHint  Optional hint — `"access_token"` or `"refresh_token"`.
   *
   * @example
   * ```ts
   * await oauth.revokeToken("eyJhbGci...", "access_token");
   * ```
   */
  async revokeToken(token: string, tokenTypeHint?: TokenTypeHint): Promise<void> {
    const url = `${this._base}/oauth/revoke`;
    const form: Record<string, string> = { token };
    if (tokenTypeHint) form["token_type_hint"] = tokenTypeHint;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: this._key
        ? { Authorization: `Bearer ${this._key}` }
        : {},
      form,
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
  }

  /**
   * Introspect a token (RFC 7662).
   *
   * Requires admin credentials. Returns `{ active: false }` for invalid
   * or expired tokens.
   *
   * @param token  The token to introspect.
   *
   * @example
   * ```ts
   * const info = await oauth.introspectToken("eyJhbGci...");
   * console.log(info.active, info.sub, info.scope);
   * ```
   */
  async introspectToken(token: string): Promise<IntrospectResult> {
    const url = `${this._base}/oauth/introspect`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: this._key
        ? { Authorization: `Bearer ${this._key}` }
        : {},
      form: { token },
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<IntrospectResult>();
  }
}
