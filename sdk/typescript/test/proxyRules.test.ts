/**
 * Tests for ProxyRulesClient (F1).
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { ProxyRulesClient } from "../src/proxyRules.js";
import { SharkAPIError } from "../src/errors.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeResp(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function installFetch(mock: ReturnType<typeof vi.fn>) {
  vi.stubGlobal("fetch", mock);
}

afterEach(() => vi.unstubAllGlobals());

const BASE = "https://auth.example.com";
const KEY = "sk_live_test";

function client() {
  return new ProxyRulesClient({ baseUrl: BASE, adminKey: KEY });
}

const sampleRule = {
  id: "rule_abc",
  app_id: "app_1",
  name: "block-writes",
  pattern: "/api/writes/*",
  methods: ["POST"],
  require: "authenticated",
  allow: "",
  scopes: [],
  enabled: true,
  priority: 100,
  tier_match: "",
  m2m: false,
  created_at: "2026-04-24T10:00:00Z",
  updated_at: "2026-04-24T10:00:00Z",
};

// ---------------------------------------------------------------------------
// listRules
// ---------------------------------------------------------------------------

describe("ProxyRulesClient.listRules()", () => {
  it("GET /api/v1/admin/proxy/rules/db returns rule list", async () => {
    const mock = vi.fn().mockResolvedValueOnce(
      makeResp(200, { data: [sampleRule], total: 1 })
    );
    installFetch(mock);

    const result = await client().listRules();

    expect(result.total).toBe(1);
    expect(result.data[0].id).toBe("rule_abc");
    const [url, init] = mock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${BASE}/api/v1/admin/proxy/rules/db`);
    expect((init.headers as Record<string, string>)["Authorization"]).toBe(`Bearer ${KEY}`);
  });

  it("appends app_id query param when provided", async () => {
    installFetch(vi.fn().mockResolvedValueOnce(makeResp(200, { data: [], total: 0 })));

    await client().listRules({ appId: "app_123" });

    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain("app_id=app_123");
  });

  it("throws SharkAPIError on 401", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(
        makeResp(401, { error: { code: "unauthorized", message: "bad key" } })
      )
    );
    await expect(client().listRules()).rejects.toBeInstanceOf(SharkAPIError);
  });

  it("throws SharkAPIError on 403", async () => {
    installFetch(vi.fn().mockResolvedValueOnce(makeResp(403, { error: { code: "forbidden", message: "no" } })));
    const err = await client().listRules().catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(403);
  });
});

// ---------------------------------------------------------------------------
// createRule
// ---------------------------------------------------------------------------

describe("ProxyRulesClient.createRule()", () => {
  it("POST returns 201 with created rule", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(makeResp(201, { data: sampleRule }))
    );
    const result = await client().createRule({ name: "block-writes", pattern: "/api/writes/*" });
    expect(result.data.id).toBe("rule_abc");
  });

  it("throws SharkAPIError on 422", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(
        makeResp(422, { error: { code: "invalid_proxy_rule", message: "pattern must start with '/'" } })
      )
    );
    const err = await client().createRule({ name: "x", pattern: "bad" }).catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).code).toBe("invalid_proxy_rule");
    expect((err as SharkAPIError).status).toBe(422);
  });
});

// ---------------------------------------------------------------------------
// getRule
// ---------------------------------------------------------------------------

describe("ProxyRulesClient.getRule()", () => {
  it("GET /api/v1/admin/proxy/rules/db/{id} returns rule", async () => {
    installFetch(vi.fn().mockResolvedValueOnce(makeResp(200, { data: sampleRule })));
    const rule = await client().getRule("rule_abc");
    expect(rule.name).toBe("block-writes");
  });

  it("throws SharkAPIError(404) on not found", async () => {
    installFetch(vi.fn().mockResolvedValueOnce(makeResp(404, { error: { code: "not_found", message: "not found" } })));
    const err = await client().getRule("nope").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(404);
  });
});

// ---------------------------------------------------------------------------
// updateRule
// ---------------------------------------------------------------------------

describe("ProxyRulesClient.updateRule()", () => {
  it("PATCH returns updated rule", async () => {
    const updated = { ...sampleRule, enabled: false };
    installFetch(vi.fn().mockResolvedValueOnce(makeResp(200, { data: updated })));
    const result = await client().updateRule("rule_abc", { enabled: false });
    expect(result.data.enabled).toBe(false);
  });

  it("throws SharkAPIError on 400 (conflict require+allow)", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(makeResp(400, { error: { code: "invalid_request", message: "bad" } }))
    );
    const err = await client().updateRule("rule_abc", { require: "x", allow: "anonymous" }).catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(400);
  });
});

// ---------------------------------------------------------------------------
// deleteRule
// ---------------------------------------------------------------------------

describe("ProxyRulesClient.deleteRule()", () => {
  it("DELETE returns void on 204", async () => {
    installFetch(vi.fn().mockResolvedValueOnce(new Response(null, { status: 204 })));
    await expect(client().deleteRule("rule_abc")).resolves.toBeUndefined();
  });

  it("throws SharkAPIError on 404", async () => {
    installFetch(vi.fn().mockResolvedValueOnce(makeResp(404, { error: { code: "not_found", message: "not found" } })));
    const err = await client().deleteRule("nope").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(404);
  });
});
