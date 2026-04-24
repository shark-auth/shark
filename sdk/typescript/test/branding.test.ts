/**
 * Tests for BrandingClient (F3).
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { BrandingClient } from "../src/branding.js";
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
const c = () => new BrandingClient({ baseUrl: BASE, adminKey: KEY });

const sampleBranding = {
  id: "branding_1",
  primary_color: "#6366f1",
  secondary_color: "#4f46e5",
  font_family: "Inter, sans-serif",
  logo_url: null,
  design_tokens: { colors: { primary: "#6366f1" } },
  created_at: "2026-04-24T10:00:00Z",
  updated_at: "2026-04-24T10:00:00Z",
};

describe("BrandingClient.getBranding()", () => {
  it("GET /api/v1/admin/branding returns branding row", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: sampleBranding })));
    const b = await c().getBranding();
    expect(b.id).toBe("branding_1");
    expect(b.design_tokens).toEqual({ colors: { primary: "#6366f1" } });

    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toBe(`${BASE}/api/v1/admin/branding`);
  });

  it("appends app_slug query param when provided", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: sampleBranding })));
    await c().getBranding("my-app");
    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain("app_slug=my-app");
  });

  it("throws SharkAPIError on 401", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(401, { error: { code: "unauthorized", message: "bad key" } })));
    await expect(c().getBranding()).rejects.toBeInstanceOf(SharkAPIError);
  });
});

describe("BrandingClient.setBranding()", () => {
  it("PATCH /api/v1/admin/branding/design-tokens returns updated branding", async () => {
    const tokens = { colors: { primary: "#ff0000" } };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        makeResp(200, { data: { branding: sampleBranding, design_tokens: tokens } })
      )
    );

    const result = await c().setBranding(tokens);
    expect(result.design_tokens).toEqual(tokens);

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${BASE}/api/v1/admin/branding/design-tokens`);
    expect((init as RequestInit).method).toBe("PATCH");
    const parsed = JSON.parse((init as RequestInit).body as string) as { design_tokens: unknown };
    expect(parsed.design_tokens).toEqual(tokens);
  });

  it("throws SharkAPIError on 400", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(400, { error: { code: "invalid_request", message: "bad json" } })));
    const err = await c().setBranding({}).catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(400);
  });

  it("throws SharkAPIError on 403", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(403, {})));
    const err = await c().setBranding({ x: 1 }).catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(403);
  });
});
