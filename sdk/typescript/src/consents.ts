/**
 * Self-service OAuth consent management — `/api/v1/auth/consents`.
 *
 * Lets users review and revoke the OAuth grants they've issued to agents.
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";
import { Session } from "./auth.js";

/** A consent row as returned by `list`. */
export interface ConsentRow {
  id?: string;
  user_id?: string;
  client_id?: string;
  scopes?: string[] | string;
  granted_at?: string;
  [key: string]: unknown;
}

/** Options for {@link ConsentsClient}. */
export interface ConsentsClientOptions {
  /** Shared session — typically the one from {@link AuthClient}. */
  session?: Session;
}

/**
 * Wraps the per-user OAuth-consent endpoints. Requires the underlying
 * {@link Session} to carry a valid session cookie (typically planted by
 * `AuthClient.login`).
 */
export class ConsentsClient {
  private static readonly PREFIX = "/api/v1/auth/consents";
  private readonly _base: string;
  readonly session: Session;

  constructor(baseUrl: string, opts: ConsentsClientOptions = {}) {
    this._base = baseUrl.replace(/\/+$/, "");
    this.session = opts.session ?? new Session();
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
  ): Promise<{ status: number; text: string; json<T>(): T }> {
    const url = `${this._base}${path}`;
    const headers: Record<string, string> = {};
    const cookie = this.session.cookieHeader();
    if (cookie) headers["Cookie"] = cookie;
    const resp = await httpRequest(url, { method, headers });
    const setCookie = resp.headers["set-cookie"];
    if (setCookie) this.session.ingestSetCookie(setCookie);
    return resp;
  }

  /** List active OAuth consents granted by the authenticated user. */
  async list(): Promise<ConsentRow[]> {
    const resp = await this._request("GET", ConsentsClient.PREFIX);
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const data = resp.json<unknown>();
    if (Array.isArray(data)) return data as ConsentRow[];
    if (data && typeof data === "object") {
      const obj = data as { data?: unknown };
      if (Array.isArray(obj.data)) return obj.data as ConsentRow[];
    }
    return [];
  }

  /** Revoke a specific consent grant by id. */
  async revoke(consentId: string): Promise<void> {
    const resp = await this._request(
      "DELETE",
      `${ConsentsClient.PREFIX}/${encodeURIComponent(consentId)}`,
    );
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }
}
