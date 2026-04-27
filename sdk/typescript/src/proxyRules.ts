/**
 * Proxy rules CRUD — DB-backed rule management (v1.5).
 *
 * All methods target `/api/v1/admin/proxy/rules/db` and `/api/v1/admin/proxy/rules/import`.
 * Auth is via an admin API key passed as a Bearer token.
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** A single DB-backed proxy rule as returned by the server. */
export interface ProxyRule {
  id: string;
  app_id: string;
  name: string;
  pattern: string;
  methods: string[];
  require: string;
  allow: string;
  scopes: string[];
  enabled: boolean;
  priority: number;
  tier_match: string;
  m2m: boolean;
  created_at: string;
  updated_at: string;
}

/** Input for creating a new proxy rule. */
export interface CreateProxyRuleInput {
  /** Scope the rule to one application. Omit for global. */
  app_id?: string;
  /** Human-readable label. Required. */
  name: string;
  /** chi-style path pattern, e.g. `/api/writes/*`. Required. */
  pattern: string;
  /** HTTP methods to match. Empty = any method. */
  methods?: string[];
  /** Require grammar string, e.g. `authenticated`, `role:admin`. */
  require?: string;
  /** Set to `"anonymous"` to allow unauthenticated access. */
  allow?: string;
  /** Additional scope AND-check. */
  scopes?: string[];
  /** Defaults to true. */
  enabled?: boolean;
  /** Higher priority wins. Defaults to 0. */
  priority?: number;
  /** Tier required, e.g. `"pro"`. */
  tier_match?: string;
  /** Machine-to-machine flag — denies non-agent callers. */
  m2m?: boolean;
}

/** Partial update for an existing proxy rule. All fields optional. */
export type UpdateProxyRuleInput = Partial<CreateProxyRuleInput>;

/** Response envelope returned by list. */
export interface ProxyRuleListResult {
  data: ProxyRule[];
  total: number;
}

/** Per-row error in an import result. */
export interface ImportRuleError {
  index: string;
  name: string;
  error: string;
}

/** Response returned by `importProxyRulesYAML`. */
export interface ImportResult {
  imported: number;
  errors: ImportRuleError[];
}

/** Response shape for create/get/update that may include an engine refresh error. */
export interface ProxyRuleMutationResult {
  data: ProxyRule;
  /** Non-empty when the DB write succeeded but the live-engine refresh failed. */
  engine_refresh_error?: string;
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link ProxyRulesClient}. */
export interface ProxyRulesClientOptions {
  /** Base URL of the SharkAuth server, e.g. `https://auth.example.com`. */
  baseUrl: string;
  /** Admin API key (Bearer token). */
  adminKey: string;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Admin client for managing DB-backed proxy rules.
 */
export class ProxyRulesClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: ProxyRulesClientOptions) {
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

  /** List all proxy rules, optionally filtered by app ID. */
  async listRules(opts?: { appId?: string }): Promise<ProxyRuleListResult> {
    let url = `${this._base}/api/v1/admin/proxy/rules/db`;
    if (opts?.appId) {
      url += `?app_id=${encodeURIComponent(opts.appId)}`;
    }
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<ProxyRuleListResult>();
  }

  /** Create a new proxy rule. Returns the created rule. */
  async createRule(spec: CreateProxyRuleInput): Promise<ProxyRuleMutationResult> {
    const url = `${this._base}/api/v1/admin/proxy/rules/db`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(spec),
    });
    if (resp.status !== 201) await this._throw(resp.status, resp.text);
    return resp.json<ProxyRuleMutationResult>();
  }

  /** Fetch a single rule by ID. */
  async getRule(id: string): Promise<ProxyRule> {
    const url = `${this._base}/api/v1/admin/proxy/rules/db/${encodeURIComponent(id)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<{ data: ProxyRule }>().data;
  }

  /** Partially update a rule. Only supplied fields are mutated. */
  async updateRule(id: string, patch: UpdateProxyRuleInput): Promise<ProxyRuleMutationResult> {
    const url = `${this._base}/api/v1/admin/proxy/rules/db/${encodeURIComponent(id)}`;
    const resp = await httpRequest(url, {
      method: "PATCH",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(patch),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<ProxyRuleMutationResult>();
  }

  /** Delete a rule by ID. Returns void on 204. */
  async deleteRule(id: string): Promise<void> {
    const url = `${this._base}/api/v1/admin/proxy/rules/db/${encodeURIComponent(id)}`;
    const resp = await httpRequest(url, {
      method: "DELETE",
      headers: this._auth(),
    });
    if (resp.status !== 204) await this._throw(resp.status, resp.text);
  }

}
