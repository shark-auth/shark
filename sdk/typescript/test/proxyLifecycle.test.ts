/**
 * Tests for ProxyLifecycleClient (F2).
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { ProxyLifecycleClient } from "../src/proxyLifecycle.js";
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

function client() {
  return new ProxyLifecycleClient({ baseUrl: BASE, adminKey: KEY });
}

const runningStatus = {
  state: 1,
  state_str: "running",
  listeners: 2,
  rules_loaded: 17,
  started_at: "2026-04-24T10:09:45Z",
  last_error: "",
};

describe("ProxyLifecycleClient.getProxyStatus()", () => {
  it("GET /api/v1/admin/proxy/lifecycle returns status", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: runningStatus })));
    const s = await client().getProxyStatus();
    expect(s.state_str).toBe("running");
    expect(s.listeners).toBe(2);
  });

  it("throws SharkAPIError on 401", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(401, { error: { code: "unauthorized", message: "bad key" } })));
    await expect(client().getProxyStatus()).rejects.toBeInstanceOf(SharkAPIError);
  });
});

describe("ProxyLifecycleClient.startProxy()", () => {
  it("POST /api/v1/admin/proxy/start returns status", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: runningStatus })));
    const s = await client().startProxy();
    expect(s.state_str).toBe("running");

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${BASE}/api/v1/admin/proxy/start`);
    expect((init as RequestInit).method).toBe("POST");
  });

  it("throws SharkAPIError(409) on conflict (already running)", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(409, { error: { code: "proxy_start_failed", message: "already running" } })));
    const err = await client().startProxy().catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(409);
    expect((err as SharkAPIError).code).toBe("proxy_start_failed");
  });
});

describe("ProxyLifecycleClient.stopProxy()", () => {
  it("POST /api/v1/admin/proxy/stop returns stopped state", async () => {
    const stopped = { ...runningStatus, state: 0, state_str: "stopped" as const, listeners: 0, started_at: "" };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: stopped })));
    const s = await client().stopProxy();
    expect(s.state_str).toBe("stopped");
  });

  it("throws SharkAPIError on 403", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(403, { error: { code: "forbidden", message: "no" } })));
    const err = await client().stopProxy().catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(403);
  });
});

describe("ProxyLifecycleClient.reloadProxy()", () => {
  it("POST /api/v1/admin/proxy/reload returns status", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: runningStatus })));
    const s = await client().reloadProxy();
    expect(s.rules_loaded).toBe(17);

    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain("/proxy/reload");
  });

  it("throws SharkAPIError on 422", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(422, { error: { code: "proxy_reload_failed", message: "bind failed" } })));
    const err = await client().reloadProxy().catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).code).toBe("proxy_reload_failed");
  });
});
