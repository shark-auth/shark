/**
 * Tests for AuthClient — P2 additions: check() and revokeSelf().
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { AuthClient } from "../src/auth.js";
import { SharkAPIError } from "../src/errors.js";

function makeResp(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

afterEach(() => vi.unstubAllGlobals());

const BASE = "https://auth.example.com";
const c = () => new AuthClient(BASE);

// ---------------------------------------------------------------------------
// P2 smoke tests — check, revokeSelf
// ---------------------------------------------------------------------------

describe("AuthClient.check()", () => {
  it("POST /api/v1/auth/check sends action+resource and returns response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, { allowed: true })));
    const result = await c().check("read", "documents:123");
    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${BASE}/api/v1/auth/check`);
    expect((init as RequestInit).method).toBe("POST");
    const body = JSON.parse((init as RequestInit).body as string) as { action: string; resource: string };
    expect(body.action).toBe("read");
    expect(body.resource).toBe("documents:123");
    expect((result as { allowed: boolean }).allowed).toBe(true);
  });

  it("throws SharkAPIError on 403", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(403, { error: "forbidden" })));
    const err = await c().check("write", "secrets").catch((e) => e);
    expect(err).toBeInstanceOf(SharkAPIError);
    expect((err as SharkAPIError).status).toBe(403);
  });
});

describe("AuthClient.revokeSelf()", () => {
  it("POST /api/v1/auth/revoke returns void on 200", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(makeResp(200, {})));
    await expect(c().revokeSelf()).resolves.toBeUndefined();
    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
    expect(url).toBe(`${BASE}/api/v1/auth/revoke`);
    expect((init as RequestInit).method).toBe("POST");
  });
});
