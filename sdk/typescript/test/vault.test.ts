/**
 * Tests for VaultClient.
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { VaultClient } from "../src/vault.js";
import { VaultError } from "../src/errors.js";

// ------------------------------------------------------------------
// Fetch mock helpers
// ------------------------------------------------------------------

type FetchMock = ReturnType<typeof vi.fn>;

function makeFetchResponse(status: number, body: unknown): Response {
  const json = JSON.stringify(body);
  return new Response(json, {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function installFetch(mock: FetchMock) {
  vi.stubGlobal("fetch", mock);
}

function restoreFetch() {
  vi.unstubAllGlobals();
}

// ------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------

describe("VaultClient.exchange()", () => {
  afterEach(restoreFetch);

  it("returns a fresh token on 200 (array scopes)", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(
        makeFetchResponse(200, {
          access_token: "ya29.fresh",
          expires_at: 1999999999,
          provider: "google",
          scopes: ["gmail.readonly", "calendar.events"],
        })
      )
    );

    const vault = new VaultClient({
      authUrl: "https://auth.example",
      accessToken: "agent_token_xyz",
    });
    const tok = await vault.exchange("conn_abc");

    expect(tok.accessToken).toBe("ya29.fresh");
    expect(tok.expiresAt).toBe(1999999999);
    expect(tok.provider).toBe("google");
    expect(tok.scopes).toEqual(["gmail.readonly", "calendar.events"]);
  });

  it("handles scope as a space-delimited string", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(
        makeFetchResponse(200, {
          access_token: "t",
          scope: "read write",
          provider: "github",
        })
      )
    );
    const vault = new VaultClient({
      authUrl: "https://auth.example",
      accessToken: "at",
    });
    const tok = await vault.exchange("conn_x");
    expect(tok.scopes).toEqual(["read", "write"]);
  });

  it("throws VaultError(404) on not found", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(makeFetchResponse(404, { error: "not_found" }))
    );
    const vault = new VaultClient({
      authUrl: "https://auth.example",
      accessToken: "at",
    });
    const err = await vault.exchange("missing").catch((e) => e);
    expect(err).toBeInstanceOf(VaultError);
    expect((err as VaultError).statusCode).toBe(404);
  });

  it("throws VaultError(401) on unauthorized", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(makeFetchResponse(401, { error: "unauthorized" }))
    );
    const vault = new VaultClient({
      authUrl: "https://auth.example",
      accessToken: "at",
    });
    const err = await vault.exchange("conn").catch((e) => e);
    expect(err).toBeInstanceOf(VaultError);
    expect((err as VaultError).statusCode).toBe(401);
  });

  it("throws VaultError(403) on forbidden", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(makeFetchResponse(403, { error: "forbidden" }))
    );
    const vault = new VaultClient({
      authUrl: "https://auth.example",
      accessToken: "at",
    });
    const err = await vault.exchange("conn").catch((e) => e);
    expect(err).toBeInstanceOf(VaultError);
    expect((err as VaultError).statusCode).toBe(403);
  });

  it("throws VaultError on empty referenceToken", async () => {
    const vault = new VaultClient({
      authUrl: "https://auth.example",
      accessToken: "at",
    });
    await expect(vault.exchange("")).rejects.toBeInstanceOf(VaultError);
  });

  it("auto-retries once on 401 when onRefresh is provided", async () => {
    const responses = [
      makeFetchResponse(401, { error: "unauthorized" }),
      makeFetchResponse(200, {
        access_token: "fresh",
        scopes: [],
      }),
    ];
    let idx = 0;
    installFetch(vi.fn().mockImplementation(async () => responses[idx++]));

    let refreshCalled = 0;
    const vault = new VaultClient({
      authUrl: "https://auth.example",
      accessToken: "stale_token",
      onRefresh: async () => {
        refreshCalled++;
        return "new_token";
      },
      maxRetries: 1,
    });

    const tok = await vault.exchange("conn_abc");
    expect(tok.accessToken).toBe("fresh");
    expect(refreshCalled).toBe(1);
  });

  it("does not loop infinitely — stops at maxRetries", async () => {
    // Always returns 401 — use mockImplementation so each call gets a fresh Response
    installFetch(vi.fn().mockImplementation(async () => makeFetchResponse(401, {})));

    let refreshCalls = 0;
    const vault = new VaultClient({
      authUrl: "https://auth.example",
      accessToken: "at",
      onRefresh: async () => {
        refreshCalls++;
        return "new_token";
      },
      maxRetries: 2,
    });

    await expect(vault.exchange("conn")).rejects.toBeInstanceOf(VaultError);
    // 2 retries = 2 refresh calls
    expect(refreshCalls).toBe(2);
  });
});
