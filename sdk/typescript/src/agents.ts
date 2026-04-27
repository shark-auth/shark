/**
 * Agent admin API (v1.5).
 *
 * Register, list, and revoke (deactivate) OAuth agent clients.
 *
 * Routes (all admin-key authenticated):
 *   POST   /api/v1/agents                        — create / register agent
 *   GET    /api/v1/agents                        — list agents
 *   DELETE /api/v1/agents/{id}                   — deactivate (revoke) agent
 *   GET    /api/v1/agents/{id}                   — get agent by ID or clientID
 *   GET    /api/v1/agents/{id}/tokens            — list active tokens
 *   POST   /api/v1/agents/{id}/tokens/revoke-all — revoke all tokens
 *   POST   /api/v1/agents/{id}/rotate-secret     — rotate client secret
 *   POST   /api/v1/agents/{id}/rotate-dpop-key   — rotate DPoP keypair
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** A registered agent OAuth client. */
export interface Agent {
  id: string;
  name: string;
  description?: string;
  client_id: string;
  client_type: string;
  auth_method: string;
  redirect_uris: string[];
  grant_types: string[];
  response_types: string[];
  scopes: string[];
  token_lifetime: number;
  metadata: Record<string, unknown>;
  logo_uri?: string;
  homepage_uri?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * Response from `registerAgent`. Includes the one-time `client_secret`.
 * The secret is not stored server-side (only its hash is kept) — copy it now.
 */
export interface AgentCreateResult extends Agent {
  /** Plaintext client secret. Shown exactly once. */
  client_secret: string;
}

/** Input for registering a new agent. */
export interface RegisterAgentInput {
  /** Human-readable label. Required. */
  name: string;
  description?: string;
  /** `"confidential"` (default) or `"public"`. */
  client_type?: string;
  /** `"client_secret_basic"` (default). */
  auth_method?: string;
  redirect_uris?: string[];
  grant_types?: string[];
  response_types?: string[];
  /** OAuth scopes the agent is allowed to request. */
  scopes?: string[];
  /** Token lifetime in seconds. Default: 3600. */
  token_lifetime?: number;
  metadata?: Record<string, unknown>;
  logo_uri?: string;
  homepage_uri?: string;
}

/** A single active token record for an agent. */
export interface TokenInfo {
  token_id: string;
  agent_id: string;
  jkt?: string;
  scope?: string;
  expires_at?: string;
  created_at?: string;
}

/** Result of a revoke-all tokens operation. */
export interface RevokeAllResult {
  revoked_count: number;
  agent_id: string;
}

/** New credentials returned after a secret rotation. */
export interface AgentCredentials {
  agent_id: string;
  client_id: string;
  /** Plaintext client secret — copy now, shown once. */
  client_secret: string;
  rotated_at?: string;
}

/** Result of a DPoP key rotation operation. */
export interface DPoPRotationResult {
  old_jkt: string;
  new_jkt: string;
  revoked_token_count: number;
  audit_event_id: string;
}

/** Options for `listAgents`. */
export interface ListAgentsOptions {
  /** Filter by active/inactive state. Omit for all. */
  active?: boolean;
  /** Search term matched against name. */
  search?: string;
  limit?: number;
  offset?: number;
}

/** Response from `listAgents`. */
export interface AgentListResult {
  data: Agent[];
  total: number;
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link AgentsClient}. */
export interface AgentsClientOptions {
  /** Base URL of the SharkAuth server. */
  baseUrl: string;
  /** Admin API key (Bearer token). */
  adminKey: string;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Admin client for managing agent OAuth clients.
 */
export class AgentsClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: AgentsClientOptions) {
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
   * Register (create) a new agent client.
   *
   * The returned `client_secret` is shown exactly once — store it securely.
   */
  async registerAgent(input: RegisterAgentInput): Promise<AgentCreateResult> {
    const url = `${this._base}/api/v1/agents`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    if (resp.status !== 201) await this._throw(resp.status, resp.text);
    return resp.json<AgentCreateResult>();
  }

  /** List agents with optional filters. */
  async listAgents(opts?: ListAgentsOptions): Promise<AgentListResult> {
    const qs = new URLSearchParams();
    if (opts?.active != null) qs.set("active", String(opts.active));
    if (opts?.search) qs.set("search", opts.search);
    if (opts?.limit != null) qs.set("limit", String(opts.limit));
    if (opts?.offset != null) qs.set("offset", String(opts.offset));
    const query = qs.toString();
    const url = `${this._base}/api/v1/agents${query ? `?${query}` : ""}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<AgentListResult>();
  }

  /** Get a single agent by ID or clientID. */
  async getAgent(id: string): Promise<Agent> {
    const url = `${this._base}/api/v1/agents/${encodeURIComponent(id)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<Agent>();
  }

  /**
   * Deactivate (revoke) an agent by ID.
   *
   * This sets `active=false` and revokes all outstanding OAuth tokens for the
   * agent's `client_id`. Returns void on 204.
   */
  async revokeAgent(id: string): Promise<void> {
    const url = `${this._base}/api/v1/agents/${encodeURIComponent(id)}`;
    const resp = await httpRequest(url, {
      method: "DELETE",
      headers: this._auth(),
    });
    if (resp.status !== 204) await this._throw(resp.status, resp.text);
  }

  // ------------------------------------------------------------------
  // Token management
  // ------------------------------------------------------------------

  /** List active tokens for an agent. GET /api/v1/agents/{id}/tokens */
  async listTokens(agentId: string): Promise<TokenInfo[]> {
    const url = `${this._base}/api/v1/agents/${encodeURIComponent(agentId)}/tokens`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<{ data?: TokenInfo[] } | TokenInfo[]>();
    const items = Array.isArray(body) ? body : (body as { data?: TokenInfo[] }).data ?? [];
    return items.map((t) => ({
      token_id: (t as Record<string, string>).id ?? t.token_id,
      agent_id: t.agent_id ?? agentId,
      jkt: t.jkt,
      scope: t.scope,
      expires_at: t.expires_at,
      created_at: t.created_at,
    }));
  }

  /** Revoke all active tokens for an agent. POST /api/v1/agents/{id}/tokens/revoke-all */
  async revokeAllTokens(agentId: string): Promise<RevokeAllResult> {
    const url = `${this._base}/api/v1/agents/${encodeURIComponent(agentId)}/tokens/revoke-all`;
    const resp = await httpRequest(url, { method: "POST", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204) await this._throw(resp.status, resp.text);
    if (resp.status === 204 || !resp.text) {
      return { revoked_count: 0, agent_id: agentId };
    }
    const data = resp.json<{ revoked_count?: number; count?: number }>();
    return { revoked_count: data.revoked_count ?? data.count ?? 0, agent_id: agentId };
  }

  /** Rotate the agent's client secret. POST /api/v1/agents/{id}/rotate-secret */
  async rotateSecret(agentId: string): Promise<AgentCredentials> {
    const url = `${this._base}/api/v1/agents/${encodeURIComponent(agentId)}/rotate-secret`;
    const resp = await httpRequest(url, { method: "POST", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 201) await this._throw(resp.status, resp.text);
    const raw = resp.json<{ data?: AgentCredentials } & AgentCredentials>();
    const data = raw.data ?? raw;
    return {
      agent_id: data.agent_id ?? agentId,
      client_id: data.client_id ?? "",
      client_secret: data.client_secret ?? "",
      rotated_at: data.rotated_at,
    };
  }

  /**
   * Rotate an agent's DPoP keypair binding (admin-forced).
   * POST /api/v1/agents/{id}/rotate-dpop-key
   *
   * @param agentId         The `agent_*` identifier.
   * @param newPublicKeyJwk New public key as JWK. Server derives `cnf.jkt` via RFC 7638 SHA-256.
   * @param reason          Optional audit-log reason string.
   */
  async rotateDpopKey(
    agentId: string,
    newPublicKeyJwk: Record<string, unknown>,
    reason?: string,
  ): Promise<DPoPRotationResult> {
    const url = `${this._base}/api/v1/agents/${encodeURIComponent(agentId)}/rotate-dpop-key`;
    const body: Record<string, unknown> = { new_public_jwk: newPublicKeyJwk };
    if (reason !== undefined) body.reason = reason;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const data = resp.json<DPoPRotationResult>();
    return {
      old_jkt: data.old_jkt ?? "",
      new_jkt: data.new_jkt ?? "",
      revoked_token_count: data.revoked_token_count ?? 0,
      audit_event_id: data.audit_event_id ?? "",
    };
  }
}
