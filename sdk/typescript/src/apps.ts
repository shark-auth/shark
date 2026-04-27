/**
 * Application management — admin API.
 *
 * Wraps `/api/v1/admin/apps` (see `internal/api/router.go`).
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Application row as returned by the server. */
export interface App {
  id: string;
  name: string;
  integration_mode?: string;
  redirect_uris?: string[];
  client_id?: string;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
}

/** Result from {@link AppsClient.rotateSecret}. */
export interface AppRotateSecretResult {
  client_id?: string;
  client_secret?: string;
  [key: string]: unknown;
}

/** Input for {@link AppsClient.create}. */
export interface CreateAppInput {
  name: string;
  integration_mode?: string;
  redirect_uris?: string[];
  [key: string]: unknown;
}

/** Input for {@link AppsClient.update}. */
export type UpdateAppInput = Partial<CreateAppInput>;

/** Options for {@link AppsClient}. */
export interface AppsClientOptions {
  baseUrl: string;
  adminKey: string;
}

const PREFIX = "/api/v1/admin/apps";

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/** Admin client for managing OAuth/embed applications. */
export class AppsClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: AppsClientOptions) {
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
        error?: { code?: string; message?: string } | string;
        message?: string;
      };
      if (typeof body.error === "object" && body.error) {
        if (body.error.code) code = body.error.code;
        if (body.error.message) message = body.error.message;
      } else if (typeof body.error === "string") {
        message = body.error;
      } else if (body.message) {
        message = body.message;
      }
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  private static _unwrap<T>(body: unknown): T {
    if (
      body &&
      typeof body === "object" &&
      "data" in (body as Record<string, unknown>) &&
      Object.keys(body as Record<string, unknown>).length <= 2
    ) {
      return (body as { data: T }).data;
    }
    return body as T;
  }

  // --------------------------------------------------------------- CRUD

  /** Create an application. */
  async create(input: CreateAppInput): Promise<App> {
    const body: Record<string, unknown> = {
      integration_mode: "custom",
      ...input,
    };
    const url = `${this._base}${PREFIX}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return AppsClient._unwrap<App>(resp.json());
  }

  /** List apps. */
  async list(): Promise<App[]> {
    const url = `${this._base}${PREFIX}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<App[] | { data?: App[]; apps?: App[] }>();
    if (Array.isArray(body)) return body;
    return body?.data ?? body?.apps ?? [];
  }

  /** Fetch a single app. */
  async get(appId: string): Promise<App> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(appId)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return AppsClient._unwrap<App>(resp.json());
  }

  /** Update an app (PATCH). */
  async update(appId: string, fields: UpdateAppInput): Promise<App> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(appId)}`;
    const resp = await httpRequest(url, {
      method: "PATCH",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(fields),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return AppsClient._unwrap<App>(resp.json());
  }

  /** Delete an app. */
  async delete(appId: string): Promise<void> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(appId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // --------------------------------------------------------- Operations

  /**
   * Rotate the app's client secret.
   *
   * Returns the freshly issued secret in the response body — store it
   * immediately, the server cannot return it again.
   */
  async rotateSecret(appId: string): Promise<AppRotateSecretResult> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(appId)}/rotate-secret`;
    const resp = await httpRequest(url, { method: "POST", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return AppsClient._unwrap<AppRotateSecretResult>(resp.json());
  }

  /** Get the embed snippet text for the application. */
  async getSnippet(appId: string): Promise<string> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(appId)}/snippet`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const ctype = (resp.headers["content-type"] || "").toLowerCase();
    if (ctype.includes("application/json")) {
      let body: unknown;
      try {
        body = resp.json();
      } catch {
        return resp.text;
      }
      if (body && typeof body === "object") {
        const obj = body as Record<string, unknown>;
        for (const k of ["snippet", "embed", "html", "code"] as const) {
          if (typeof obj[k] === "string") return obj[k] as string;
        }
        const data = obj.data;
        if (typeof data === "string") return data;
        if (data && typeof data === "object") {
          const dobj = data as Record<string, unknown>;
          for (const k of ["snippet", "embed", "html", "code"] as const) {
            if (typeof dobj[k] === "string") return dobj[k] as string;
          }
        }
      }
      return resp.text;
    }
    return resp.text;
  }

}
