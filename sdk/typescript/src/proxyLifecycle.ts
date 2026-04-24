/**
 * Proxy lifecycle control (v1.5).
 *
 * Start, stop, reload the embedded reverse-proxy without restarting the server,
 * and query its current state.
 *
 * Routes:
 *   GET  /api/v1/admin/proxy/lifecycle
 *   POST /api/v1/admin/proxy/start
 *   POST /api/v1/admin/proxy/stop
 *   POST /api/v1/admin/proxy/reload
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Current state of the embedded reverse-proxy. */
export interface ProxyStatus {
  /** Integer enum from `proxy.State`. Prefer `state_str` for UI. */
  state: number;
  /** Human-readable state label. */
  state_str: "stopped" | "running" | "reloading" | "unknown";
  /** Number of actively bound listener ports. */
  listeners: number;
  /** Total rules compiled across all listeners. */
  rules_loaded: number;
  /** RFC3339 UTC timestamp; empty string when stopped. */
  started_at: string;
  /** Most recent error recorded by the Manager; empty string on success. */
  last_error: string;
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link ProxyLifecycleClient}. */
export interface ProxyLifecycleClientOptions {
  /** Base URL of the SharkAuth server. */
  baseUrl: string;
  /** Admin API key (Bearer token). */
  adminKey: string;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Admin client for controlling proxy lifecycle.
 */
export class ProxyLifecycleClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: ProxyLifecycleClientOptions) {
    this._base = opts.baseUrl.replace(/\/+$/, "");
    this._key = opts.adminKey;
  }

  private _auth(): Record<string, string> {
    return { Authorization: `Bearer ${this._key}` };
  }

  private async _parseStatus(text: string, status: number): Promise<ProxyStatus> {
    if (status !== 200) {
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
    const body = JSON.parse(text) as { data: ProxyStatus };
    return body.data;
  }

  /** Get the current proxy state without modifying it. */
  async getProxyStatus(): Promise<ProxyStatus> {
    const resp = await httpRequest(`${this._base}/api/v1/admin/proxy/lifecycle`, {
      headers: this._auth(),
    });
    return this._parseStatus(resp.text, resp.status);
  }

  /** Start the proxy. Returns the new state. 409 if already running. */
  async startProxy(): Promise<ProxyStatus> {
    const resp = await httpRequest(`${this._base}/api/v1/admin/proxy/start`, {
      method: "POST",
      headers: this._auth(),
    });
    return this._parseStatus(resp.text, resp.status);
  }

  /** Stop the proxy. Idempotent — stopping a stopped proxy returns 200. */
  async stopProxy(): Promise<ProxyStatus> {
    const resp = await httpRequest(`${this._base}/api/v1/admin/proxy/stop`, {
      method: "POST",
      headers: this._auth(),
    });
    return this._parseStatus(resp.text, resp.status);
  }

  /**
   * Reload the proxy (stop + start in one critical section).
   * Also re-publishes all DB rules to the live engine.
   */
  async reloadProxy(): Promise<ProxyStatus> {
    const resp = await httpRequest(`${this._base}/api/v1/admin/proxy/reload`, {
      method: "POST",
      headers: this._auth(),
    });
    return this._parseStatus(resp.text, resp.status);
  }
}
