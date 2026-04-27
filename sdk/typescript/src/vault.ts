/**
 * Shark Token Vault client — fetch fresh 3rd-party OAuth credentials.
 *
 * @example
 * ```ts
 * const vault = new VaultClient({
 *   authUrl: "https://auth.example",
 *   accessToken: agentToken,
 * });
 * const fresh = await vault.exchange("conn_abc");
 * console.log(fresh.accessToken);
 * ```
 */

import { httpRequest } from "./http.js";
import { VaultError } from "./errors.js";

/** A fresh, server-refreshed 3rd-party access token from the vault. */
export interface VaultToken {
  accessToken: string;
  expiresAt?: number;
  provider?: string;
  scopes: string[];
}

/** Result of disconnecting a vault connection. */
export interface VaultDisconnectResult {
  connection_id: string;
  revoked_agent_ids: string[];
  revoked_token_count: number;
  cascade_audit_event_id?: string;
}

/** A vault connection record returned by listConnections. */
export interface VaultConnection {
  id: string;
  provider?: string;
  user_id?: string;
  created_at?: string;
  [key: string]: unknown;
}

/** Response from listConnections. */
export interface VaultConnectionListResult {
  data: VaultConnection[];
  total: number;
}

/** Options for listConnections. */
export interface ListConnectionsOptions {
  limit?: number;
  offset?: number;
  user_id?: string;
}

/** Options for constructing a {@link VaultClient}. */
export interface VaultClientOptions {
  /** Base URL of the SharkAuth server. */
  authUrl: string;
  /** Bearer access token for the calling agent. */
  accessToken: string;
  /**
   * Callback invoked when the server returns 401.
   * Should return a fresh access token. Called at most `maxRetries` times.
   */
  onRefresh?: () => Promise<string>;
  /** Maximum number of 401-retry attempts. Default: 2. */
  maxRetries?: number;
  /** Override the connections path. Default: `/admin/vault/connections`. */
  connectionsPath?: string;
  /** Admin API key for admin-scoped operations (disconnect, listConnections). */
  adminKey?: string;
}

/**
 * Fetches auto-refreshed 3rd-party tokens from the Shark Token Vault.
 *
 * On a `401` response, {@link VaultClient} will call the user-supplied
 * `onRefresh` callback (if any) to get a fresh agent token and retry the
 * request. Retries are bounded by `maxRetries` (default 2) to prevent
 * infinite loops.
 */
export class VaultClient {
  private readonly _authUrl: string;
  private _accessToken: string;
  private readonly _onRefresh?: () => Promise<string>;
  private readonly _maxRetries: number;
  private readonly _connectionsPath: string;
  private readonly _adminKey?: string;

  constructor(opts: VaultClientOptions) {
    this._authUrl = opts.authUrl.replace(/\/+$/, "");
    this._accessToken = opts.accessToken;
    this._onRefresh = opts.onRefresh;
    this._maxRetries = opts.maxRetries ?? 2;
    this._connectionsPath = opts.connectionsPath ?? "/admin/vault/connections";
    this._adminKey = opts.adminKey;
  }

  private _adminAuth(): Record<string, string> {
    const key = this._adminKey ?? this._accessToken;
    return { Authorization: `Bearer ${key}` };
  }

  /**
   * @deprecated Use {@link fetchToken} instead — same shape, preferred name matching Python SDK.
   */
  async exchange(referenceToken: string): Promise<VaultToken> {
    return this.fetchToken(referenceToken);
  }

  private async _exchangeWithRetry(
    connectionId: string,
    attempt: number
  ): Promise<VaultToken> {
    const url = `${this._authUrl}${this._connectionsPath}/${connectionId}/token`;
    const resp = await httpRequest(url, {
      method: "GET",
      headers: { Authorization: `Bearer ${this._accessToken}` },
    });

    if (resp.status === 200) {
      let body: Record<string, unknown>;
      try {
        body = resp.json<Record<string, unknown>>();
      } catch {
        throw new VaultError(
          `vault response is not valid JSON: ${resp.text.slice(0, 200)}`
        );
      }

      let scopes: string[];
      const rawScopes = body["scopes"] ?? body["scope"];
      if (typeof rawScopes === "string") {
        scopes = rawScopes.split(/\s+/).filter(Boolean);
      } else if (Array.isArray(rawScopes)) {
        scopes = rawScopes.map(String);
      } else {
        scopes = [];
      }

      return {
        accessToken: String(body["access_token"]),
        expiresAt:
          typeof body["expires_at"] === "number" ? body["expires_at"] : undefined,
        provider:
          typeof body["provider"] === "string" ? body["provider"] : undefined,
        scopes,
      };
    }

    if (resp.status === 401) {
      if (this._onRefresh && attempt < this._maxRetries) {
        // Invoke user-supplied refresh callback and retry with the fresh token.
        this._accessToken = await this._onRefresh();
        return this._exchangeWithRetry(connectionId, attempt + 1);
      }
      throw new VaultError("agent not authorized (401)", 401);
    }

    if (resp.status === 403) {
      throw new VaultError("missing scope for vault access (403)", 403);
    }

    if (resp.status === 404) {
      throw new VaultError(`connection not found: ${connectionId}`, 404);
    }

    throw new VaultError(
      `vault request failed: HTTP ${resp.status}: ${resp.text.slice(0, 200)}`,
      resp.status
    );
  }

  // ------------------------------------------------------------------
  // Admin — disconnect + list connections
  // ------------------------------------------------------------------

  /**
   * Disconnect a vault connection.
   * DELETE /api/v1/admin/vault/connections/{id}
   *
   * @param connectionId The `conn_*` identifier.
   * @param cascade      When true (default) cascade-revokes agent tokens bound to this connection.
   */
  async disconnect(connectionId: string, cascade = true): Promise<VaultDisconnectResult> {
    const qs = cascade ? "?cascade=true" : "?cascade=false";
    const url = `${this._authUrl}/api/v1/admin/vault/connections/${encodeURIComponent(connectionId)}${qs}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._adminAuth() });
    if (resp.status === 404) throw new VaultError(`connection not found: ${connectionId}`, 404);
    if (resp.status === 401) throw new VaultError("not authorized (401)", 401);
    if (resp.status === 403) throw new VaultError("forbidden — admin key required (403)", 403);
    if (resp.status !== 200 && resp.status !== 204) {
      throw new VaultError(`disconnect failed: HTTP ${resp.status}: ${resp.text.slice(0, 200)}`, resp.status);
    }
    if (resp.status === 204 || !resp.text) {
      return { connection_id: connectionId, revoked_agent_ids: [], revoked_token_count: 0 };
    }
    const body = resp.json<VaultDisconnectResult>();
    return {
      connection_id: body.connection_id ?? connectionId,
      revoked_agent_ids: body.revoked_agent_ids ?? [],
      revoked_token_count: body.revoked_token_count ?? 0,
      cascade_audit_event_id: body.cascade_audit_event_id,
    };
  }

  /**
   * List vault connections (admin scope).
   * GET /api/v1/vault/connections
   */
  async listConnections(params?: ListConnectionsOptions): Promise<VaultConnectionListResult> {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.offset != null) qs.set("offset", String(params.offset));
    if (params?.user_id) qs.set("user_id", params.user_id);
    const query = qs.toString();
    const url = `${this._authUrl}/api/v1/admin/vault/connections${query ? `?${query}` : ""}`;
    const resp = await httpRequest(url, { headers: this._adminAuth() });
    if (resp.status !== 200) {
      throw new VaultError(`listConnections failed: HTTP ${resp.status}: ${resp.text.slice(0, 200)}`, resp.status);
    }
    const raw = resp.json<VaultConnectionListResult | VaultConnection[]>();
    if (Array.isArray(raw)) return { data: raw, total: raw.length };
    return raw as VaultConnectionListResult;
  }

  // ------------------------------------------------------------------
  // fetchToken — preferred name (mirrors Python vault.fetch_token)
  // ------------------------------------------------------------------

  /**
   * Retrieve a fresh access token for the given stored connection.
   *
   * Preferred over {@link exchange} — matches Python SDK's `fetch_token` name.
   * Supports an optional DPoP proof header for DPoP-bound connections.
   *
   * @param referenceToken  Connection ID / reference token identifying the stored connection.
   * @param dpop            Optional DPoP proof header value.
   */
  async fetchToken(referenceToken: string, dpop?: string): Promise<VaultToken> {
    if (!referenceToken) throw new VaultError("referenceToken (connection ID) is required");
    // For DPoP-less requests delegate to the retry-capable private helper.
    if (!dpop) return this._exchangeWithRetry(referenceToken, 0);
    // DPoP path — single attempt (caller manages token refresh for DPoP flows).
    const url = `${this._authUrl}${this._connectionsPath}/${referenceToken}/token`;
    const resp = await httpRequest(url, {
      method: "GET",
      headers: { Authorization: `Bearer ${this._accessToken}`, DPoP: dpop },
    });
    if (resp.status === 200) {
      let body: Record<string, unknown>;
      try { body = resp.json<Record<string, unknown>>(); }
      catch { throw new VaultError(`vault response is not valid JSON: ${resp.text.slice(0, 200)}`); }
      let scopes: string[];
      const rawScopes = body["scopes"] ?? body["scope"];
      if (typeof rawScopes === "string") scopes = rawScopes.split(/\s+/).filter(Boolean);
      else if (Array.isArray(rawScopes)) scopes = rawScopes.map(String);
      else scopes = [];
      return {
        accessToken: String(body["access_token"]),
        expiresAt: typeof body["expires_at"] === "number" ? body["expires_at"] : undefined,
        provider: typeof body["provider"] === "string" ? body["provider"] : undefined,
        scopes,
      };
    }
    if (resp.status === 401) throw new VaultError("agent not authorized (401)", 401);
    if (resp.status === 403) throw new VaultError("missing scope for vault access (403)", 403);
    if (resp.status === 404) throw new VaultError(`connection not found: ${referenceToken}`, 404);
    throw new VaultError(`vault request failed: HTTP ${resp.status}: ${resp.text.slice(0, 200)}`, resp.status);
  }
}
