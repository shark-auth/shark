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

  constructor(opts: VaultClientOptions) {
    this._authUrl = opts.authUrl.replace(/\/+$/, "");
    this._accessToken = opts.accessToken;
    this._onRefresh = opts.onRefresh;
    this._maxRetries = opts.maxRetries ?? 2;
    this._connectionsPath = opts.connectionsPath ?? "/admin/vault/connections";
  }

  /**
   * Retrieve a fresh access token for the given stored connection.
   *
   * This is the primary API. The name matches the task spec (`exchange`).
   *
   * @param referenceToken  Connection ID / reference token identifying the stored connection.
   * @returns {@link VaultToken} with a fresh `accessToken` from the vault.
   * @throws {@link VaultError} on 404 (not found), 401/403 (auth), or other errors.
   */
  async exchange(referenceToken: string): Promise<VaultToken> {
    if (!referenceToken) {
      throw new VaultError("referenceToken (connection ID) is required");
    }
    return this._exchangeWithRetry(referenceToken, 0);
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
}
