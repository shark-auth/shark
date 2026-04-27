/**
 * Self-service session management — `/api/v1/auth/sessions`.
 *
 * Lists the authenticated user's active sessions and lets them revoke
 * individual sessions or all sessions.
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";
import { Session } from "./auth.js";

/** A session row as returned by `list`. */
export interface SessionRow {
  id?: string;
  session_id?: string;
  user_id?: string;
  /** Server marks the row representing the calling session. */
  current?: boolean;
  ip?: string;
  user_agent?: string;
  created_at?: string;
  expires_at?: string;
  [key: string]: unknown;
}

/** Options for {@link SessionsClient}. */
export interface SessionsClientOptions {
  /** Shared session — typically the one from {@link AuthClient}. */
  session?: Session;
}

/**
 * Wraps the per-user session-management endpoints. Requires the underlying
 * {@link Session} to carry a valid session cookie (typically planted by
 * `AuthClient.login`).
 */
export class SessionsClient {
  private static readonly PREFIX = "/api/v1/auth/sessions";
  private readonly _base: string;
  readonly session: Session;

  constructor(baseUrl: string, opts: SessionsClientOptions = {}) {
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

  /** List the authenticated user's active sessions. */
  async list(): Promise<SessionRow[]> {
    const resp = await this._request("GET", SessionsClient.PREFIX);
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const data = resp.json<unknown>();
    if (Array.isArray(data)) return data as SessionRow[];
    if (data && typeof data === "object") {
      const obj = data as { data?: unknown };
      if (Array.isArray(obj.data)) return obj.data as SessionRow[];
    }
    return [];
  }

  /** Revoke a specific session by id. */
  async revoke(sessionId: string): Promise<void> {
    const resp = await this._request(
      "DELETE",
      `${SessionsClient.PREFIX}/${encodeURIComponent(sessionId)}`,
    );
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  /**
   * Revoke every session except the current one.
   *
   * The user-facing surface does not expose a bulk-revoke endpoint, so this
   * iterates {@link list} and skips the row flagged `current`.
   */
  async revokeAll(): Promise<void> {
    const rows = await this.list();
    for (const row of rows) {
      if (row.current) continue;
      const sid = row.id ?? row.session_id;
      if (sid) await this.revoke(sid);
    }
  }
}
