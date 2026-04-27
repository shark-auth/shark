/**
 * High-level client for agents with automatic DPoP signing,
 * and admin namespaces for proxy rules, lifecycle, branding, paywall,
 * users, and agents (v1.5).
 */

import { DPoPProver } from "./dpop.js";
import { ProxyRulesClient } from "./proxyRules.js";
import { ProxyLifecycleClient } from "./proxyLifecycle.js";
import { BrandingClient } from "./branding.js";
import { PaywallClient } from "./paywall.js";
import { UsersClient } from "./users.js";
import { AgentsClient } from "./agents.js";
import { AuthClient } from "./auth.js";
import { MfaClient } from "./mfa.js";
import { SessionsClient } from "./sessions.js";
import { ConsentsClient } from "./consents.js";
import { DcrClient } from "./dcr.js";
import { OrganizationsClient } from "./organizations.js";
import { AppsClient } from "./apps.js";
import { ApiKeysClient } from "./apiKeys.js";
import { RbacClient } from "./rbac.js";
import { AuditClient } from "./audit.js";
import { WebhooksClient } from "./webhooks.js";

/** Options for {@link SharkClient}. */
export interface SharkClientOptions {
  /** Bearer access token for the calling agent. Optional in admin-only mode. */
  accessToken?: string;
  /**
   * Optional {@link DPoPProver} — when supplied, the client will
   * auto-sign every outgoing request with a DPoP proof JWT.
   */
  dpopProver?: DPoPProver;
  /** Custom User-Agent header. */
  userAgent?: string;
  /**
   * Admin API key used by the admin namespaces
   * (proxy rules, lifecycle, branding, users, agents, organizations,
   * apps, api-keys, rbac, audit, webhooks).
   * Required if any admin namespace is used.
   */
  adminKey?: string;
  /**
   * Base URL of the SharkAuth server, e.g. `https://auth.example.com`.
   * Required if any admin namespace or `paywall` is used.
   */
  baseUrl?: string;
}

/**
 * A DPoP-aware HTTP client for SharkAuth agents, extended with v1.5 admin namespaces.
 *
 * Wraps the built-in `fetch` API to automatically inject
 * `Authorization: DPoP <token>` and `DPoP: <proof>` headers.
 *
 * Admin namespaces (`proxyRules`, `proxyLifecycle`, `branding`, `users`, `agents`)
 * use the `adminKey` option; `paywall` uses `baseUrl` with no auth.
 *
 * @example
 * ```ts
 * const client = new SharkClient({
 *   accessToken: "agent_abc",
 *   adminKey: "sk_live_xyz",
 *   baseUrl: "https://auth.example.com",
 * });
 *
 * const rules = await client.proxyRules.listRules();
 * const status = await client.proxyLifecycle.getProxyStatus();
 * ```
 */
export class SharkClient {
  private readonly _accessToken: string;
  private readonly _dpopProver?: DPoPProver;
  private readonly _userAgent: string;

  // Admin namespaces (v1.5)
  readonly proxyRules: ProxyRulesClient;
  readonly proxyLifecycle: ProxyLifecycleClient;
  readonly branding: BrandingClient;
  readonly paywall: PaywallClient;
  readonly users: UsersClient;
  readonly agents: AgentsClient;

  // Pass-A human-auth namespaces (Phase 7)
  readonly auth: AuthClient;
  readonly mfa: MfaClient;
  readonly sessions: SessionsClient;
  readonly consents: ConsentsClient;
  readonly dcr: DcrClient;

  // Pass-B admin namespaces (Phase 7)
  readonly organizations: OrganizationsClient;
  readonly apps: AppsClient;
  readonly apiKeys: ApiKeysClient;
  readonly rbac: RbacClient;
  readonly audit: AuditClient;
  readonly webhooks: WebhooksClient;

  constructor(opts: SharkClientOptions) {
    this._accessToken = opts.accessToken ?? "";
    this._dpopProver = opts.dpopProver;
    this._userAgent = opts.userAgent ?? "@sharkauth/node/0.1.0";

    const baseUrl = opts.baseUrl ?? "";
    const adminKey = opts.adminKey ?? opts.accessToken ?? "";

    this.proxyRules = new ProxyRulesClient({ baseUrl, adminKey });
    this.proxyLifecycle = new ProxyLifecycleClient({ baseUrl, adminKey });
    this.branding = new BrandingClient({ baseUrl, adminKey });
    this.paywall = new PaywallClient({ baseUrl });
    this.users = new UsersClient({ baseUrl, adminKey });
    this.agents = new AgentsClient({ baseUrl, adminKey });

    this.auth = new AuthClient(baseUrl);
    this.mfa = new MfaClient(baseUrl);
    this.sessions = new SessionsClient(baseUrl);
    this.consents = new ConsentsClient(baseUrl);
    this.dcr = new DcrClient(baseUrl);

    this.organizations = new OrganizationsClient({ baseUrl, adminKey });
    this.apps = new AppsClient({ baseUrl, adminKey });
    this.apiKeys = new ApiKeysClient({ baseUrl, adminKey });
    this.rbac = new RbacClient({ baseUrl, adminKey });
    this.audit = new AuditClient({ baseUrl, adminKey });
    this.webhooks = new WebhooksClient({ baseUrl, adminKey });
  }

  /**
   * Perform an HTTP request with automatic DPoP signing.
   *
   * Signature and headers are generated based on the method and URL
   * provided in the arguments.
   *
   * @param input   URL or Request object.
   * @param init    Standard fetch options.
   */
  async fetch(input: string | URL | Request, init?: RequestInit): Promise<Response> {
    const method = init?.method ?? "GET";
    const url = input instanceof Request ? input.url : String(input);

    const headers = new Headers(init?.headers);
    headers.set("User-Agent", this._userAgent);
    headers.set("Accept", "application/json");

    if (this._dpopProver) {
      const proof = await this._dpopProver.createProof({
        method,
        url,
        accessToken: this._accessToken,
      });
      headers.set("Authorization", `DPoP ${this._accessToken}`);
      headers.set("DPoP", proof);
    } else {
      headers.set("Authorization", `Bearer ${this._accessToken}`);
    }

    return fetch(input, {
      ...init,
      headers,
    });
  }
}
