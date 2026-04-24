/**
 * Tests for AgentsClient (F6).
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { AgentsClient } from "../src/agents.js";
import { SharkAPIError } from "../src/errors.js";

function makeResp(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

afterEach(() => vi.unstubAllGlobals());

const BASE = "https://auth.example.com";
const KEY = "sk_live_test";
const c = () => new AgentsClient({ baseUrl: BASE, adminKey: KEY });

const sampleAgent = {
  id: "agent_abc",
  name: "my-bot",
  description: "test agent",
  client_id: "shark_agent_xyz",
  client_type: "confidential",
  auth_method: "client_secret_basic",
  redirect_uris: [],
  grant_types: [],
  response_types: [],
  scopes: ["read:data"],
  token_lifetime: 3600,
  metadata: {},
  logo_uri: "",
  homepage_uri: "",
  active: true,
  created_at: "2026-04-24T10:00:00Z",
  updated_at: "2026-04-24T10:00:00Z",
};

// ---------------------------------------------------------------------------
// registerAgent
// ---------------------------------------------------------------------------

describe("AgentsClient.registerAgent()", () => {
  it("POST /api/v1/agents returns agent with client_secret", async () => {
    const createResp = { ...sampleAgent, client_secret: "secret_plaintext" };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(201, createResp)));

    const result = await c().registerAgent({ name: "my-bot", scopes: ["read:data"] });
    expect(result.id).toBe("agent_abc");
    expect(result.client_secret).toBe("secret_plaintext");

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${BASE}/api/v1/agents`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init.headers as Record<string, string>)["Authorization"]).toBe(`Bearer ${KEY}`);
  });

  it("throws SharkAPIError on 400 (missing name)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(makeResp(400, { error: { code: "invalid_request", message: "name is required" } }))
    );
    const err = await c().registerAgent({ name: "" }).catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).code).toBe("invalid_request");
    expect((err as SharkAPIError).status).toBe(400);
  });

  it("throws SharkAPIError on 401", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(401, {})));
    await expect(c().registerAgent({ name: "x" })).rejects.toBeInstanceOf(SharkAPIError);
  });
});

// ---------------------------------------------------------------------------
// listAgents
// ---------------------------------------------------------------------------

describe("AgentsClient.listAgents()", () => {
  it("GET /api/v1/agents returns agent list", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: [sampleAgent], total: 1 })));
    const result = await c().listAgents();
    expect(result.total).toBe(1);
    expect(result.data[0].name).toBe("my-bot");
  });

  it("appends active query param when provided", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: [], total: 0 })));
    await c().listAgents({ active: true, search: "bot" });
    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain("active=true");
    expect(url).toContain("search=bot");
  });

  it("throws SharkAPIError on 403", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(403, {})));
    const err = await c().listAgents().catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(403);
  });
});

// ---------------------------------------------------------------------------
// getAgent
// ---------------------------------------------------------------------------

describe("AgentsClient.getAgent()", () => {
  it("GET /api/v1/agents/{id} returns agent", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, sampleAgent)));
    const agent = await c().getAgent("agent_abc");
    expect(agent.client_id).toBe("shark_agent_xyz");
  });

  it("throws SharkAPIError on 404", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(404, { error: { code: "not_found", message: "not found" } })));
    const err = await c().getAgent("ghost").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(404);
  });
});

// ---------------------------------------------------------------------------
// revokeAgent
// ---------------------------------------------------------------------------

describe("AgentsClient.revokeAgent()", () => {
  it("DELETE /api/v1/agents/{id} returns void on 204", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(null, { status: 204 })));
    await expect(c().revokeAgent("agent_abc")).resolves.toBeUndefined();

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/v1/agents/agent_abc");
    expect((init as RequestInit).method).toBe("DELETE");
  });

  it("throws SharkAPIError on 404", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(404, { error: { code: "not_found", message: "not found" } })));
    const err = await c().revokeAgent("ghost").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(404);
  });

  it("throws SharkAPIError on 422", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(422, { error: { code: "api_error", message: "fail" } })));
    const err = await c().revokeAgent("agent_abc").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
  });
});
