/**
 * Admin API-key management.
 *
 * Wraps `/api/v1/api-keys` (mounted under the admin-key auth group; see
 * `internal/api/router.go`).
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** API key row as returned by the server. */
export interface ApiKey {
  id?: string;
  key_id?: string;
  name?: string;
  scopes?: string[];
  expires_at?: string;
  created_at?: string;
  /** Plaintext key — only returned at creation/rotation. */
  key?: string;
  [key: string]: unknown;
}

/** Input for {@link ApiKeysClient.create}. */
export interface CreateApiKeyInput {
  name: string;
  scopes?: string[];
  /** ISO 8601 expiry. */
  expires_at?: string;
}

/** Options for {@link ApiKeysClient}. */
export interface ApiKeysClientOptions {
  baseUrl: string;
  adminKey: string;
}

const PREFIX = "/api/v1/api-keys";

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/** Admin client for managing admin API keys. */
export class ApiKeysClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: ApiKeysClientOptions) {
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

  /**
   * Create a new admin API key.
   *
   * Returns a row containing both `key_id` (or `id`) AND the full `key`
   * value. The full key is only returned at creation — store it now.
   */
  async create(input: CreateApiKeyInput): Promise<ApiKey> {
    const body: Record<string, unknown> = { name: input.name };
    if (input.scopes !== undefined) body.scopes = input.scopes;
    if (input.expires_at !== undefined) body.expires_at = input.expires_at;
    const url = `${this._base}${PREFIX}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return ApiKeysClient._unwrap<ApiKey>(resp.json());
  }

  /** List API keys. */
  async list(): Promise<ApiKey[]> {
    const url = `${this._base}${PREFIX}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<ApiKey[] | { data?: ApiKey[]; api_keys?: ApiKey[] }>();
    if (Array.isArray(body)) return body;
    return body?.data ?? body?.api_keys ?? [];
  }

  /** Fetch a single API key. */
  async get(keyId: string): Promise<ApiKey> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(keyId)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return ApiKeysClient._unwrap<ApiKey>(resp.json());
  }

  /** Revoke (delete) an API key. */
  async revoke(keyId: string): Promise<void> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(keyId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  /**
   * Rotate an API key. Returns a row with the new `key` (and typically a
   * new `key_id`). Store the new key immediately — the server cannot
   * return it again.
   */
  async rotate(keyId: string): Promise<ApiKey> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(keyId)}/rotate`;
    const resp = await httpRequest(url, { method: "POST", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return ApiKeysClient._unwrap<ApiKey>(resp.json());
  }
}
