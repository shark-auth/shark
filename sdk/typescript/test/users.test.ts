/**
 * Tests for UsersClient (F5).
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { UsersClient } from "../src/users.js";
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
const c = () => new UsersClient({ baseUrl: BASE, adminKey: KEY });

const sampleUser = {
  id: "usr_abc",
  email: "raul@example.com",
  name: "Raul",
  metadata: { tier: "pro" },
  created_at: "2026-04-24T10:00:00Z",
  updated_at: "2026-04-24T10:00:00Z",
};

// ---------------------------------------------------------------------------
// setUserTier
// ---------------------------------------------------------------------------

describe("UsersClient.setUserTier()", () => {
  it("PATCH /api/v1/admin/users/{id}/tier returns user + tier", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(makeResp(200, { data: { user: sampleUser, tier: "pro" } }))
    );

    const result = await c().setUserTier("usr_abc", "pro");
    expect(result.tier).toBe("pro");
    expect(result.user.id).toBe("usr_abc");

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/admin/users/usr_abc/tier");
    expect((init as RequestInit).method).toBe("PATCH");
    const body = JSON.parse((init as RequestInit).body as string) as { tier: string };
    expect(body.tier).toBe("pro");
    expect((init.headers as Record<string, string>)["Authorization"]).toBe(`Bearer ${KEY}`);
  });

  it("throws SharkAPIError on 400 (invalid tier)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(makeResp(400, { error: { code: "invalid_tier", message: "tier must be free or pro" } }))
    );
    const err = await c().setUserTier("usr_abc", "pro").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).code).toBe("invalid_tier");
    expect((err as SharkAPIError).status).toBe(400);
  });

  it("throws SharkAPIError(404) when user not found", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(makeResp(404, { error: { code: "not_found", message: "not found" } }))
    );
    const err = await c().setUserTier("ghost", "free").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(404);
  });

  it("throws SharkAPIError on 401", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(401, { error: { code: "unauthorized", message: "bad key" } })));
    await expect(c().setUserTier("usr_abc", "pro")).rejects.toBeInstanceOf(SharkAPIError);
  });
});

// ---------------------------------------------------------------------------
// getUser
// ---------------------------------------------------------------------------

describe("UsersClient.getUser()", () => {
  it("GET /api/v1/users/{id} returns user", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, sampleUser)));
    const user = await c().getUser("usr_abc");
    expect(user.email).toBe("raul@example.com");
  });

  it("handles data-wrapped response shape", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: sampleUser })));
    const user = await c().getUser("usr_abc");
    expect(user.id).toBe("usr_abc");
  });

  it("throws SharkAPIError on 404", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(404, { error: { code: "not_found", message: "not found" } })));
    const err = await c().getUser("ghost").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(404);
  });
});

// ---------------------------------------------------------------------------
// listUsers
// ---------------------------------------------------------------------------

describe("UsersClient.listUsers()", () => {
  it("GET /api/v1/users returns user list", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: [sampleUser], total: 1 })));
    const result = await c().listUsers();
    expect(result.total).toBe(1);
    expect(result.data[0].id).toBe("usr_abc");
  });

  it("appends email query param when provided", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { data: [], total: 0 })));
    await c().listUsers({ email: "raul@example.com" });
    const [url] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string];
    expect(url).toContain("email=raul%40example.com");
  });

  it("throws SharkAPIError on 403", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(403, {})));
    const err = await c().listUsers().catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(403);
  });
});
