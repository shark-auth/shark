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
// PKCE helpers — RFC 7636
// ---------------------------------------------------------------------------

/** PKCE triple as returned by {@link pkcePair}. */
export interface PkcePair {
  /** 43-char URL-safe random string (the "code_verifier"). */
  verifier: string;
  /** URL-safe base64 (no padding) of SHA-256(verifier). */
  challenge: string;
  /** Always `"S256"`. */
  method: "S256";
}

function base64UrlEncode(bytes: Uint8Array): string {
  // Browser-safe URL-safe base64 without padding.
  let s = "";
  for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
  // btoa is available in browsers and Node 16+.
  const b64 =
    typeof btoa === "function"
      ? btoa(s)
      : Buffer.from(bytes).toString("base64");
  return b64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function getRandomBytes(n: number): Uint8Array {
  const out = new Uint8Array(n);
  const cryptoObj =
    (typeof globalThis !== "undefined" &&
      (globalThis as { crypto?: { getRandomValues?: (a: Uint8Array) => Uint8Array } }).crypto) ||
    undefined;
  if (cryptoObj?.getRandomValues) {
    cryptoObj.getRandomValues(out);
    return out;
  }
  // Fallback to Node crypto (only when Web Crypto not present).
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const nodeCrypto = require("node:crypto") as typeof import("node:crypto");
  return new Uint8Array(nodeCrypto.randomBytes(n));
}

interface SubtleLike {
  digest(algo: string, data: ArrayBuffer): Promise<ArrayBuffer>;
  importKey(
    format: string,
    key: ArrayBuffer,
    algo: { name: string; hash: { name: string } },
    extractable: boolean,
    keyUsages: string[],
  ): Promise<unknown>;
  sign(algo: string, key: unknown, data: ArrayBuffer): Promise<ArrayBuffer>;
}

function getSubtle(): SubtleLike | undefined {
  const g = globalThis as { crypto?: { subtle?: unknown } };
  const s = g.crypto?.subtle;
  return s ? (s as SubtleLike) : undefined;
}

async function sha256(input: Uint8Array): Promise<Uint8Array> {
  const subtle = getSubtle();
  if (subtle) {
    const buf = input.buffer.slice(input.byteOffset, input.byteOffset + input.byteLength) as ArrayBuffer;
    const digest = await subtle.digest("SHA-256", buf);
    return new Uint8Array(digest);
  }
  // Node fallback
  const nodeCrypto = await import("node:crypto");
  return new Uint8Array(nodeCrypto.createHash("sha256").update(input).digest());
}

/**
 * Generate a PKCE (verifier, challenge, method) triple per RFC 7636.
 *
 * @returns `{ verifier, challenge, method: "S256" }`. Send `challenge` to
 *   `/oauth/authorize`, then `verifier` to `/oauth/token`.
 */
export async function pkcePair(): Promise<PkcePair> {
  const verifier = base64UrlEncode(getRandomBytes(32));
  const verifierBytes = new TextEncoder().encode(verifier);
  const digest = await sha256(verifierBytes);
  const challenge = base64UrlEncode(digest);
  return { verifier, challenge, method: "S256" };
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Token type hint for revocation. */
export type TokenTypeHint = "access_token" | "refresh_token";

/** Result from `getTokenAuthorizationCode` / `refreshToken`. */
export interface OAuthToken {
  access_token: string;
  token_type: string;
  expires_in?: number;
  scope?: string;
  refresh_token?: string;
  /** JWK thumbprint of a bound DPoP key, when present. */
  cnf_jkt?: string;
  /** Full server JSON response, for debugging or future claims. */
  raw: Record<string, unknown>;
}

/** Options for {@link OAuthClient.buildAuthorizeUrl}. */
export interface BuildAuthorizeUrlOptions {
  client_id: string;
  redirect_uri: string;
  scope?: string;
  state?: string;
  code_challenge?: string;
  code_challenge_method?: string;
  response_type?: string;
  /** Base URL of the SharkAuth server (e.g. `https://auth.example.com`). */
  base_url?: string;
  /** Additional query params forwarded verbatim. */
  extra?: Record<string, string | number | boolean>;
}

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

  // ------------------------------------------------------------------
  // Authorization-code + refresh grants (no DPoP)
  // ------------------------------------------------------------------

  private async _postToken(form: Record<string, string>): Promise<OAuthToken> {
    const url = `${this._base}/oauth/token`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: {},
      form,
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const data = resp.json<Record<string, unknown>>();
    const cnf = (data["cnf"] as { jkt?: string } | undefined) ?? undefined;
    return {
      access_token: String(data["access_token"]),
      token_type:
        typeof data["token_type"] === "string" ? data["token_type"] : "Bearer",
      expires_in:
        typeof data["expires_in"] === "number" ? data["expires_in"] : undefined,
      scope: typeof data["scope"] === "string" ? data["scope"] : undefined,
      refresh_token:
        typeof data["refresh_token"] === "string"
          ? data["refresh_token"]
          : undefined,
      cnf_jkt: cnf && typeof cnf.jkt === "string" ? cnf.jkt : undefined,
      raw: data,
    };
  }

  /**
   * Exchange an authorization code for tokens (RFC 6749 §4.1.3).
   *
   * @example
   * ```ts
   * const { verifier, challenge } = await pkcePair();
   * // ...redirect user to authorize URL with `challenge`...
   * const tok = await oauth.getTokenAuthorizationCode(
   *   "auth_xyz",
   *   "https://app.example.com/cb",
   *   verifier,
   *   "my-app",
   * );
   * ```
   */
  async getTokenAuthorizationCode(
    code: string,
    redirectUri: string,
    codeVerifier?: string,
    clientId?: string,
    clientSecret?: string,
  ): Promise<OAuthToken> {
    const form: Record<string, string> = {
      grant_type: "authorization_code",
      code,
      redirect_uri: redirectUri,
    };
    if (codeVerifier !== undefined) form["code_verifier"] = codeVerifier;
    if (clientId !== undefined) form["client_id"] = clientId;
    if (clientSecret !== undefined) form["client_secret"] = clientSecret;
    return this._postToken(form);
  }

  /**
   * Refresh an access token (RFC 6749 §6).
   *
   * @example
   * ```ts
   * const fresh = await oauth.refreshToken(oldTok.refresh_token!, undefined, "my-app");
   * ```
   */
  async refreshToken(
    refreshToken: string,
    scope?: string,
    clientId?: string,
    clientSecret?: string,
  ): Promise<OAuthToken> {
    const form: Record<string, string> = {
      grant_type: "refresh_token",
      refresh_token: refreshToken,
    };
    if (scope !== undefined) form["scope"] = scope;
    if (clientId !== undefined) form["client_id"] = clientId;
    if (clientSecret !== undefined) form["client_secret"] = clientSecret;
    return this._postToken(form);
  }

  /**
   * Pure URL builder for the `/oauth/authorize` redirect (no HTTP call).
   *
   * @example
   * ```ts
   * const { verifier, challenge } = await pkcePair();
   * const url = OAuthClient.buildAuthorizeUrl({
   *   client_id: "my-app",
   *   redirect_uri: "https://app.example.com/cb",
   *   scope: "openid profile",
   *   state: "xyz",
   *   code_challenge: challenge,
   *   base_url: "https://auth.example.com",
   * });
   * ```
   */
  static buildAuthorizeUrl(opts: BuildAuthorizeUrlOptions): string {
    const params = new URLSearchParams();
    params.set("response_type", opts.response_type ?? "code");
    params.set("client_id", opts.client_id);
    params.set("redirect_uri", opts.redirect_uri);
    if (opts.scope !== undefined) params.set("scope", opts.scope);
    if (opts.state !== undefined) params.set("state", opts.state);
    if (opts.code_challenge !== undefined) {
      params.set("code_challenge", opts.code_challenge);
      params.set(
        "code_challenge_method",
        opts.code_challenge_method ?? "S256",
      );
    }
    if (opts.extra) {
      for (const [k, v] of Object.entries(opts.extra)) params.set(k, String(v));
    }
    const prefix = (opts.base_url ?? "").replace(/\/+$/, "") + "/oauth/authorize";
    return `${prefix}?${params.toString()}`;
  }
}
