/**
 * User admin API — tier management + read/list (v1.5).
 *
 * Routes (all admin-key authenticated):
 *   PATCH /api/v1/admin/users/{id}/tier   — set user tier
 *   GET   /api/v1/users/{id}              — get user by ID
 *   GET   /api/v1/users                   — list users (optional email filter)
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Accepted tier values. */
export type UserTier = "free" | "pro";

/**
 * User object as returned by the server. The full shape depends on which
 * fields are populated; we expose the common subset and allow extra keys
 * via an index signature.
 */
export interface User {
  id: string;
  email: string;
  name?: string;
  /** JSON blob with arbitrary user metadata including the `tier` key. */
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  [key: string]: unknown;
}

/** Response from `setUserTier`. */
export interface SetUserTierResult {
  user: User;
  tier: string;
}

/** Options for `listUsers`. */
export interface ListUsersOptions {
  /** Filter by email (exact or partial match — server behaviour). */
  email?: string;
  /** Maximum number of users to return. Server default: 50. */
  limit?: number;
  /** Pagination offset. */
  offset?: number;
}

/** Response from `listUsers`. */
export interface UserListResult {
  data: User[];
  total: number;
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link UsersClient}. */
export interface UsersClientOptions {
  /** Base URL of the SharkAuth server. */
  baseUrl: string;
  /** Admin API key (Bearer token). */
  adminKey: string;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Admin client for user management.
 */
export class UsersClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: UsersClientOptions) {
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
   * Set a user's tier.
   *
   * Only `"free"` and `"pro"` are accepted by the server. The tier
   * is baked into the next access token on refresh; existing tokens
   * retain the old tier until expiry.
   */
  async setUserTier(userId: string, tier: UserTier): Promise<SetUserTierResult> {
    const url = `${this._base}/api/v1/admin/users/${encodeURIComponent(userId)}/tier`;
    const resp = await httpRequest(url, {
      method: "PATCH",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify({ tier }),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<{ data: SetUserTierResult }>().data;
  }

  /** Fetch a single user by ID. */
  async getUser(userId: string): Promise<User> {
    const url = `${this._base}/api/v1/users/${encodeURIComponent(userId)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    // Server returns the user directly or wrapped; handle both shapes.
    const raw = resp.json<User | { data: User }>();
    if (raw && typeof raw === "object" && "data" in raw) {
      return (raw as { data: User }).data;
    }
    return raw as User;
  }

  /** List users, optionally filtered by email. */
  async listUsers(opts?: ListUsersOptions): Promise<UserListResult> {
    const qs = new URLSearchParams();
    if (opts?.email) qs.set("email", opts.email);
    if (opts?.limit != null) qs.set("limit", String(opts.limit));
    if (opts?.offset != null) qs.set("offset", String(opts.offset));
    const query = qs.toString();
    const url = `${this._base}/api/v1/users${query ? `?${query}` : ""}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<UserListResult>();
  }
}
