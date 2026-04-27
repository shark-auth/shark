/**
 * Smoke test for MayActClient — mock-based, no live server.
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { MayActClient } from "../src/mayAct.js";

function makeResp(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

afterEach(() => vi.unstubAllGlobals());

const BASE = "https://auth.example.com";
const KEY = "sk_live_test";
const c = () => new MayActClient({ baseUrl: BASE, adminKey: KEY });

describe("MayActClient.find()", () => {
  it("builds query string and unwraps {grants:[...]}", async () => {
    const grants = [
      { id: "mag_a", from_id: "agent-a", to_id: "u-1", max_hops: 1, scopes: [], created_at: "now" },
    ];
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(makeResp(200, { grants }))
    );
    const out = await c().find({ from_id: "agent-a", include_revoked: true });
    expect(out).toEqual(grants);

    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain(`${BASE}/api/v1/admin/may-act?`);
    expect(url).toContain("from_id=agent-a");
    expect(url).toContain("include_revoked=true");
    expect(url).not.toContain("to_id=");
  });

  it("returns [] when grants key missing", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, {})));
    const out = await c().find({ to_id: "u-9" });
    expect(out).toEqual([]);
    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain("to_id=u-9");
    expect(url).not.toContain("include_revoked");
  });
});
