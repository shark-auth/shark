/**
 * Tests for PaywallClient (F4).
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { PaywallClient } from "../src/paywall.js";
import { SharkAPIError } from "../src/errors.js";

afterEach(() => vi.unstubAllGlobals());

const BASE = "https://auth.example.com";
const c = () => new PaywallClient({ baseUrl: BASE });

// ---------------------------------------------------------------------------
// paywallURL — pure URL builder, no network
// ---------------------------------------------------------------------------

describe("PaywallClient.paywallURL()", () => {
  it("builds the correct URL with tier param", () => {
    const url = c().paywallURL({ appSlug: "my-app", tier: "pro" });
    expect(url).toBe(`${BASE}/paywall/my-app?tier=pro`);
  });

  it("includes return param when provided", () => {
    const url = c().paywallURL({ appSlug: "my-app", tier: "pro", returnUrl: "https://app.example.com/dash" });
    expect(url).toContain("return=");
    expect(url).toContain("tier=pro");
  });

  it("strips trailing slash from baseUrl", () => {
    const c2 = new PaywallClient({ baseUrl: "https://auth.example.com/" });
    const url = c2.paywallURL({ appSlug: "app", tier: "free" });
    expect(url).not.toContain("//paywall");
  });
});

// ---------------------------------------------------------------------------
// renderPaywall — fetches HTML
// ---------------------------------------------------------------------------

describe("PaywallClient.renderPaywall()", () => {
  it("returns HTML string on 200", async () => {
    const html = "<html><body>Upgrade to pro</body></html>";
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(new Response(html, { status: 200, headers: { "Content-Type": "text/html" } }))
    );
    const result = await c().renderPaywall({ appSlug: "my-app", tier: "pro" });
    expect(result).toBe(html);
  });

  it("sends GET to the correct URL", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(new Response("<html/>", { status: 200 }))
    );
    await c().renderPaywall({ appSlug: "my-app", tier: "pro", returnUrl: "/dash" });
    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain("/paywall/my-app");
    expect(url).toContain("tier=pro");
    expect(url).toContain("return=");
  });

  it("throws SharkAPIError on 404 (unknown app slug)", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("Not Found", { status: 404 })));
    const err = await c().renderPaywall({ appSlug: "ghost", tier: "pro" }).catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(404);
  });

  it("throws SharkAPIError on 400 (missing tier)", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("tier query param required", { status: 400 })));
    const err = await c().renderPaywall({ appSlug: "my-app", tier: "" }).catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(400);
  });
});

// ---------------------------------------------------------------------------
// previewPaywall
// ---------------------------------------------------------------------------

describe("PaywallClient.previewPaywall()", () => {
  it("returns URL string when format=url (no network)", async () => {
    const result = await c().previewPaywall({ appSlug: "my-app", tier: "pro", format: "url" });
    expect(result).toContain("/paywall/my-app");
    expect(result).toContain("tier=pro");
  });

  it("fetches HTML when format=html (default)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(new Response("<html>ok</html>", { status: 200 }))
    );
    const result = await c().previewPaywall({ appSlug: "my-app", tier: "pro" });
    expect(result).toContain("<html>");
  });
});
