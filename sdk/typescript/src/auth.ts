/**
 * Human-auth client — signup / login / logout / me / password / email verify / magic link.
 *
 * Wraps the public `/api/v1/auth/...` routes. No DPoP required — these
 * endpoints use cookie-based sessions plus optional bearer JWT.
 *
 * Cookie handling: in browsers, pass `credentials: "include"` is implicit
 * via the underlying fetch; this client supplies `credentials: "include"`
 * automatically and additionally captures `set-cookie` headers when running
 * on Node (where browser cookie jars do not exist) so subsequent calls to
 * the same client carry the session forward.
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Minimal user shape returned by signup/login/me. The full payload is
 * server-defined; we expose the common subset and forward the rest. */
export interface AuthUser {
  id: string;
  email: string;
  name?: string;
  email_verified?: boolean;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
}

/** Response from `signup`. */
export interface SignupResult extends AuthUser {}

/** Response from `login`. May indicate MFA challenge required. */
export interface LoginResult {
  /** True when the server requires an MFA code before the session is fully authed. */
  mfaRequired?: boolean;
  /** Populated for fully-authenticated logins (no MFA pending). */
  user?: AuthUser;
  [key: string]: unknown;
}

/** Response from `getMe`. */
export interface MeResult extends AuthUser {}

/** Response from `verifyMagicLink`. */
export interface MagicLinkVerifyResult extends AuthUser {}

/** Optional fields for `signup` beyond the required email + password. */
export interface SignupOptions {
  /** Display name. Mapped to the server's `name` field. */
  name?: string;
  /** Additional fields forwarded verbatim. */
  extra?: Record<string, unknown>;
}

// ---------------------------------------------------------------------------
// Session shim — minimal cookie jar for Node interop
// ---------------------------------------------------------------------------

/**
 * Minimal shared session — collects `set-cookie` headers from one response
 * and replays them as `Cookie` headers on later requests. Suitable for
 * Node test harnesses where no browser cookie jar exists.
 *
 * In a browser, pass an empty `Session` (or none) — the browser's cookie
 * jar plus `credentials: "include"` already handles session continuity.
 */
export class Session {
  private _cookies: Record<string, string> = {};

  /** Set or replace a cookie value by name. */
  setCookie(name: string, value: string): void {
    this._cookies[name] = value;
  }

  /** Apply Set-Cookie headers from a response into this session. */
  ingestSetCookie(setCookie: string | string[] | undefined): void {
    if (!setCookie) return;
    const list = Array.isArray(setCookie) ? setCookie : [setCookie];
    for (const raw of list) {
      // First segment is `name=value`; remaining are attributes (Path=, etc.).
      const head = raw.split(";")[0];
      const eq = head.indexOf("=");
      if (eq <= 0) continue;
      const name = head.slice(0, eq).trim();
      const value = head.slice(eq + 1).trim();
      if (name) this._cookies[name] = value;
    }
  }

  /** Render the cookie jar as a `Cookie:` header value, or undefined if empty. */
  cookieHeader(): string | undefined {
    const entries = Object.entries(this._cookies);
    if (entries.length === 0) return undefined;
    return entries.map(([k, v]) => `${k}=${v}`).join("; ");
  }
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/** Options for {@link AuthClient}. */
export interface AuthClientOptions {
  /** Pre-existing {@link Session} to share with sibling clients (MFA, sessions, consents). */
  session?: Session;
}

/**
 * Public human-auth client. Wraps the password / magic-link / email-verify /
 * password-reset endpoints under `/api/v1/auth/`.
 *
 * Stateful — the client holds a {@link Session} that captures the
 * `shark_session` cookie planted by `login`/`signup`, so follow-up calls
 * (`getMe`, `changePassword`, `requestEmailVerification`) authenticate
 * automatically. Browser callers can ignore the cookie jar; the browser's
 * own cookie store + `credentials: "include"` already handles continuity.
 */
export class AuthClient {
  private static readonly PREFIX = "/api/v1/auth";
  private readonly _base: string;
  /** Shared session token jar (Node interop). Public so siblings can read it. */
  readonly session: Session;

  constructor(baseUrl: string, opts: AuthClientOptions = {}) {
    this._base = baseUrl.replace(/\/+$/, "");
    this.session = opts.session ?? new Session();
  }

  // ------------------------------------------------------------------
  // Internal — error envelope + cookie-aware request
  // ------------------------------------------------------------------

  private async _throw(status: number, text: string): Promise<never> {
    let code = "api_error";
    let message = text.slice(0, 300);
    try {
      const body = JSON.parse(text) as {
        error?: string | { code?: string; message?: string };
        message?: string;
      };
      if (typeof body.error === "string") code = body.error;
      else if (body.error && typeof body.error === "object" && body.error.code)
        code = body.error.code;
      if (body.message) message = body.message;
      else if (body.error && typeof body.error === "object" && body.error.message)
        message = body.error.message;
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  private async _request(
    method: string,
    path: string,
    opts: {
      json?: unknown;
      query?: Record<string, string>;
      okStatuses?: number[];
    } = {},
  ): Promise<{ status: number; text: string; json<T>(): T }> {
    let url = `${this._base}${path}`;
    if (opts.query) {
      const qs = new URLSearchParams(opts.query).toString();
      if (qs) url += `?${qs}`;
    }
    const headers: Record<string, string> = {};
    const cookie = this.session.cookieHeader();
    if (cookie) headers["Cookie"] = cookie;

    const reqOpts: Parameters<typeof httpRequest>[1] = { method, headers };
    if (opts.json !== undefined) {
      headers["Content-Type"] = "application/json";
      reqOpts.body = JSON.stringify(opts.json);
    }

    const resp = await httpRequest(url, reqOpts);

    // Capture Set-Cookie for Node interop.
    const setCookie = resp.headers["set-cookie"];
    if (setCookie) this.session.ingestSetCookie(setCookie);

    return resp;
  }

  // ------------------------------------------------------------------
  // Signup / Login / Logout
  // ------------------------------------------------------------------

  /**
   * Create a new user account.
   *
   * On success the server plants the session cookie which is captured by the
   * shared {@link Session} for follow-up authenticated calls.
   *
   * Note: the backend signup struct uses `name`, not `full_name` — this method
   * accepts `name` and forwards it transparently.
   */
  async signup(
    email: string,
    password: string,
    options: SignupOptions = {},
  ): Promise<SignupResult> {
    const body: Record<string, unknown> = { email, password };
    if (options.name !== undefined) body["name"] = options.name;
    if (options.extra) Object.assign(body, options.extra);
    const resp = await this._request("POST", `${AuthClient.PREFIX}/signup`, { json: body });
    if (resp.status !== 200 && resp.status !== 201) await this._throw(resp.status, resp.text);
    return resp.json<SignupResult>();
  }

  /**
   * Authenticate with email + password. The session cookie is captured into
   * the shared {@link Session}. When MFA is enabled the server returns
   * `{ mfaRequired: true }`; the caller must follow up with `MfaClient.challenge`.
   */
  async login(email: string, password: string): Promise<LoginResult> {
    const resp = await this._request("POST", `${AuthClient.PREFIX}/login`, {
      json: { email, password },
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<LoginResult>();
  }

  /** Revoke the current session. */
  async logout(): Promise<void> {
    const resp = await this._request("POST", `${AuthClient.PREFIX}/logout`);
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // ------------------------------------------------------------------
  // /me
  // ------------------------------------------------------------------

  /** Return the authenticated user. */
  async getMe(): Promise<MeResult> {
    const resp = await this._request("GET", `${AuthClient.PREFIX}/me`);
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<MeResult>();
  }

  /** Delete the authenticated user. Requires verified email. */
  async deleteMe(): Promise<void> {
    const resp = await this._request("DELETE", `${AuthClient.PREFIX}/me`);
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // ------------------------------------------------------------------
  // Password management
  // ------------------------------------------------------------------

  /** Change the authenticated user's password. */
  async changePassword(oldPassword: string, newPassword: string): Promise<void> {
    const resp = await this._request("POST", `${AuthClient.PREFIX}/password/change`, {
      json: { current_password: oldPassword, new_password: newPassword },
    });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  /** Send a password reset email. Always returns success regardless of account existence. */
  async requestPasswordReset(email: string): Promise<void> {
    const resp = await this._request(
      "POST",
      `${AuthClient.PREFIX}/password/send-reset-link`,
      { json: { email } },
    );
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  /** Complete a password reset with the token from the reset email. */
  async confirmPasswordReset(token: string, newPassword: string): Promise<void> {
    const resp = await this._request("POST", `${AuthClient.PREFIX}/password/reset`, {
      json: { token, password: newPassword },
    });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // ------------------------------------------------------------------
  // Email verification
  // ------------------------------------------------------------------

  /** Send the email-verification link to the authenticated user. */
  async requestEmailVerification(): Promise<void> {
    const resp = await this._request("POST", `${AuthClient.PREFIX}/email/verify/send`);
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  /** Consume an email-verification token. */
  async consumeEmailVerification(token: string): Promise<void> {
    const resp = await this._request("GET", `${AuthClient.PREFIX}/email/verify`, {
      query: { token },
    });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // ------------------------------------------------------------------
  // Magic link verify (companion to MagicLinkClient.sendMagicLink)
  // ------------------------------------------------------------------

  /**
   * Consume a magic-link token and return the user object. On success the
   * server plants the session cookie on this client's {@link Session}.
   */
  async verifyMagicLink(token: string): Promise<MagicLinkVerifyResult> {
    const resp = await this._request("GET", `${AuthClient.PREFIX}/magic-link/verify`, {
      query: { token },
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<MagicLinkVerifyResult>();
  }
}
