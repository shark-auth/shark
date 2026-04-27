/**
 * MayAct grant client — admin API.
 *
 * Wraps `/api/v1/admin/may-act` (list/create/revoke). Operator-issued
 * delegation grants letting subject `from_id` act on behalf of `to_id`,
 * verified during RFC 8693 token exchange.
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

export interface MayActGrant {
  id: string;
  from_id: string;
  to_id: string;
  max_hops: number;
  scopes: string[];
  expires_at?: string | null;
  revoked_at?: string | null;
  created_by?: string;
  created_at: string;
}

export interface MayActFindOptions {
  from_id?: string;
  to_id?: string;
  include_revoked?: boolean;
}

export interface MayActCreateInput {
  from_id: string;
  to_id: string;
  max_hops?: number;
  scopes?: string[];
  expires_at?: string;
}

export interface MayActClientOptions {
  baseUrl: string;
  adminKey: string;
}

const PREFIX = "/api/v1/admin/may-act";

/** Admin client for managing may_act_grants rows. */
export class MayActClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: MayActClientOptions) {
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

  /** List grants matching the filter. Returns the unwrapped grants array. */
  async find(opts: MayActFindOptions = {}): Promise<MayActGrant[]> {
    const qs = new URLSearchParams();
    if (opts.from_id) qs.set("from_id", opts.from_id);
    if (opts.to_id) qs.set("to_id", opts.to_id);
    if (opts.include_revoked) qs.set("include_revoked", "true");
    const url = qs.toString()
      ? `${this._base}${PREFIX}?${qs.toString()}`
      : `${this._base}${PREFIX}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<{ grants?: MayActGrant[] }>();
    return Array.isArray(body?.grants) ? body!.grants : [];
  }

  /** Create a new grant. */
  async create(input: MayActCreateInput): Promise<MayActGrant> {
    const url = `${this._base}${PREFIX}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    if (resp.status !== 200 && resp.status !== 201) {
      await this._throw(resp.status, resp.text);
    }
    return resp.json<MayActGrant>();
  }

  /** Revoke a grant by ID. Returns the updated row. */
  async revoke(grantId: string): Promise<MayActGrant> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(grantId)}`;
    const resp = await httpRequest(url, {
      method: "DELETE",
      headers: this._auth(),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<MayActGrant>();
  }
}
