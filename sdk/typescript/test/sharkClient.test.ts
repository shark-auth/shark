/**
 * Tests for SharkClient namespace composition (F7).
 *
 * Verifies that all v1.5 namespaces are accessible and wire the correct
 * URL + Authorization header when called through the composed client.
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { SharkClient } from "../src/sharkClient.js";
import { ProxyRulesClient } from "../src/proxyRules.js";
import { ProxyLifecycleClient } from "../src/proxyLifecycle.js";
import { BrandingClient } from "../src/branding.js";
import { PaywallClient } from "../src/paywall.js";
import { UsersClient } from "../src/users.js";
import { AgentsClient } from "../src/agents.js";

afterEach(() => vi.unstubAllGlobals());

const BASE = "https://auth.example.com";
const ACCESS = "agent_token_123";
const ADMIN_KEY = "sk_live_admin_key";

function makeResp(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("SharkClient — namespace types", () => {
  it("exposes proxyRules as ProxyRulesClient", () => {
    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    expect(client.proxyRules).toBeInstanceOf(ProxyRulesClient);
  });

  it("exposes proxyLifecycle as ProxyLifecycleClient", () => {
    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    expect(client.proxyLifecycle).toBeInstanceOf(ProxyLifecycleClient);
  });

  it("exposes branding as BrandingClient", () => {
    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    expect(client.branding).toBeInstanceOf(BrandingClient);
  });

  it("exposes paywall as PaywallClient", () => {
    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    expect(client.paywall).toBeInstanceOf(PaywallClient);
  });

  it("exposes users as UsersClient", () => {
    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    expect(client.users).toBeInstanceOf(UsersClient);
  });

  it("exposes agents as AgentsClient", () => {
    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    expect(client.agents).toBeInstanceOf(AgentsClient);
  });
});

describe("SharkClient.proxyRules.listRules() — URL + auth header", () => {
  it("calls the correct endpoint with the admin key", async () => {
    const mock = vi.fn().mockResolvedValueOnce(makeResp(200, { data: [], total: 0 }));
    vi.stubGlobal("fetch", mock);

    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    await client.proxyRules.listRules();

    const [url, init] = mock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${BASE}/api/v1/admin/proxy/rules/db`);
    expect((init.headers as Record<string, string>)["Authorization"]).toBe(`Bearer ${ADMIN_KEY}`);
  });
});

describe("SharkClient.proxyLifecycle.startProxy() — URL + method", () => {
  it("calls the correct POST endpoint", async () => {
    const statusBody = {
      state: 1, state_str: "running", listeners: 1, rules_loaded: 5,
      started_at: "2026-04-24T10:00:00Z", last_error: "",
    };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: statusBody })));

    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    const s = await client.proxyLifecycle.startProxy();

    expect(s.state_str).toBe("running");
    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${BASE}/api/v1/admin/proxy/start`);
    expect((init as RequestInit).method).toBe("POST");
  });
});

describe("SharkClient.branding.setBranding() — payload", () => {
  it("sends design_tokens in request body", async () => {
    const tokens = { colors: { primary: "#abc" } };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        makeResp(200, { data: { branding: {}, design_tokens: tokens } })
      )
    );

    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    await client.branding.setBranding(tokens);

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    const body = JSON.parse((init as RequestInit).body as string) as { design_tokens: unknown };
    expect(body.design_tokens).toEqual(tokens);
  });
});

describe("SharkClient.paywall.paywallURL() — URL builder (no fetch)", () => {
  it("builds paywall URL with tier", () => {
    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    const url = client.paywall.paywallURL({ appSlug: "my-app", tier: "pro" });
    expect(url).toBe(`${BASE}/paywall/my-app?tier=pro`);
  });
});

describe("SharkClient.users.setUserTier() — URL + body", () => {
  it("PATCHes the correct tier endpoint", async () => {
    const user = { id: "usr_1", email: "x@example.com", created_at: "", updated_at: "" };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: { user, tier: "free" } })));

    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    const result = await client.users.setUserTier("usr_1", "free");

    expect(result.tier).toBe("free");
    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain("/admin/users/usr_1/tier");
  });
});

describe("SharkClient.agents.registerAgent() — one-time secret", () => {
  it("returns agent with client_secret", async () => {
    const created = {
      id: "agent_1", name: "bot", client_id: "c1", client_secret: "s3cr3t",
      client_type: "confidential", auth_method: "client_secret_basic",
      redirect_uris: [], grant_types: [], response_types: [], scopes: [],
      token_lifetime: 3600, metadata: {}, active: true,
      created_at: "", updated_at: "",
    };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(201, created)));

    const client = new SharkClient({ accessToken: ACCESS, adminKey: ADMIN_KEY, baseUrl: BASE });
    const result = await client.agents.registerAgent({ name: "bot" });
    expect(result.client_secret).toBe("s3cr3t");
  });
});

describe("SharkClient.fetch() — DPoP signing", () => {
  it("adds Bearer Authorization header without DPoP prover", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(new Response("{}", { status: 200 }));
    vi.stubGlobal("fetch", mockFetch);

    const client = new SharkClient({ accessToken: ACCESS, baseUrl: BASE });
    await client.fetch(`${BASE}/some/resource`);

    const [, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    const h = init.headers as Headers;
    expect(h.get("Authorization")).toBe(`Bearer ${ACCESS}`);
  });
});
