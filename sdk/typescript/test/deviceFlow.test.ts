/**
 * Tests for DeviceFlow (RFC 8628).
 *
 * Uses a minimal HTTP mock via vitest's `vi.fn()` + global fetch stubbing.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { DeviceFlow } from "../src/deviceFlow.js";
import { DeviceFlowError } from "../src/errors.js";

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

// Deterministic clock/sleep helpers (clock in ms, matches DeviceFlow API)
function fakeClock() {
  let t = 0;
  return {
    clock: () => t,
    sleep: async (ms: number) => {
      t += ms;
    },
    advance: (ms: number) => {
      t += ms;
    },
  };
}

// ------------------------------------------------------------------
// Test data
// ------------------------------------------------------------------

const DEVICE_INIT_BODY = {
  device_code: "dc_123",
  user_code: "WDJB-MJHT",
  verification_uri: "https://auth.example/device",
  verification_uri_complete: "https://auth.example/device?user_code=WDJB-MJHT",
  expires_in: 600,
  interval: 5,
};

const SUCCESS_TOKEN_BODY = {
  access_token: "at_success",
  token_type: "Bearer",
  expires_in: 3600,
  refresh_token: "rt_1",
  scope: "resource:read",
};

// ------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------

describe("DeviceFlow.begin()", () => {
  afterEach(restoreFetch);

  it("parses the device authorization response", async () => {
    const mock: FetchMock = vi.fn().mockResolvedValueOnce(
      makeFetchResponse(200, DEVICE_INIT_BODY)
    );
    installFetch(mock);

    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    const init = await flow.begin();

    expect(init.deviceCode).toBe("dc_123");
    expect(init.userCode).toBe("WDJB-MJHT");
    expect(init.verificationUri).toBe("https://auth.example/device");
    expect(init.verificationUriComplete).toContain("WDJB-MJHT");
    expect(init.interval).toBe(5);
    expect(init.expiresIn).toBe(600);
  });

  it("throws DeviceFlowError on non-200", async () => {
    installFetch(vi.fn().mockResolvedValueOnce(makeFetchResponse(500, {})));
    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    await expect(flow.begin()).rejects.toBeInstanceOf(DeviceFlowError);
  });

  it("throws DeviceFlowError on missing required keys", async () => {
    installFetch(
      vi.fn().mockResolvedValueOnce(
        makeFetchResponse(200, { device_code: "dc", user_code: "UC" }) // missing verification_uri, expires_in
      )
    );
    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    await expect(flow.begin()).rejects.toBeInstanceOf(DeviceFlowError);
  });
});

describe("DeviceFlow.waitForApproval()", () => {
  afterEach(restoreFetch);

  it("succeeds after two authorization_pending responses", async () => {
    const { clock, sleep } = fakeClock();

    const responses = [
      makeFetchResponse(200, DEVICE_INIT_BODY),
      makeFetchResponse(400, { error: "authorization_pending" }),
      makeFetchResponse(400, { error: "authorization_pending" }),
      makeFetchResponse(200, SUCCESS_TOKEN_BODY),
    ];
    let idx = 0;
    installFetch(vi.fn().mockImplementation(async () => responses[idx++]));

    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    await flow.begin();
    const tok = await flow.waitForApproval({
      timeoutMs: 60_000,
      clock,
      sleep,
    });

    expect(tok.accessToken).toBe("at_success");
    expect(tok.tokenType).toBe("Bearer");
    expect(tok.refreshToken).toBe("rt_1");
    expect(tok.scope).toBe("resource:read");
  });

  it("increases interval by 5 s on slow_down", async () => {
    const sleepDurations: number[] = [];
    const { clock, sleep: baseSleep } = fakeClock();
    const sleep = async (ms: number) => {
      sleepDurations.push(ms);
      await baseSleep(ms);
    };

    const responses = [
      makeFetchResponse(200, { ...DEVICE_INIT_BODY, interval: 2 }),
      makeFetchResponse(400, { error: "slow_down" }),
      makeFetchResponse(200, { access_token: "at_ok", token_type: "Bearer" }),
    ];
    let idx = 0;
    installFetch(vi.fn().mockImplementation(async () => responses[idx++]));

    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    await flow.begin();
    await flow.waitForApproval({ timeoutMs: 120_000, clock, sleep });

    // After slow_down: interval was 2 s → 2000 + 5000 = 7000 ms
    expect(sleepDurations).toEqual([7000]);
  });

  it("throws on access_denied", async () => {
    const { clock, sleep } = fakeClock();
    const responses = [
      makeFetchResponse(200, DEVICE_INIT_BODY),
      makeFetchResponse(400, { error: "access_denied" }),
    ];
    let idx = 0;
    installFetch(vi.fn().mockImplementation(async () => responses[idx++]));

    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    await flow.begin();
    await expect(
      flow.waitForApproval({ timeoutMs: 60_000, clock, sleep })
    ).rejects.toThrow(/denied/);
  });

  it("throws on expired_token", async () => {
    const { clock, sleep } = fakeClock();
    const responses = [
      makeFetchResponse(200, DEVICE_INIT_BODY),
      makeFetchResponse(400, { error: "expired_token" }),
    ];
    let idx = 0;
    installFetch(vi.fn().mockImplementation(async () => responses[idx++]));

    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    await flow.begin();
    await expect(
      flow.waitForApproval({ timeoutMs: 60_000, clock, sleep })
    ).rejects.toThrow(/expired/);
  });

  it("throws on timeout", async () => {
    const { clock, sleep } = fakeClock();
    const responses = [
      makeFetchResponse(200, { ...DEVICE_INIT_BODY, interval: 5 }),
      makeFetchResponse(400, { error: "authorization_pending" }),
    ];
    let idx = 0;
    installFetch(vi.fn().mockImplementation(async () => responses[idx++]));

    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    await flow.begin();
    // timeout_ms = 3000 ms; first sleep would be 5000 ms — exceeds deadline
    await expect(
      flow.waitForApproval({ timeoutMs: 3000, clock, sleep })
    ).rejects.toThrow(/timed out/);
  });

  it("throws when waitForApproval called before begin", async () => {
    const flow = new DeviceFlow({
      authUrl: "https://auth.example",
      clientId: "agent_abc",
    });
    await expect(flow.waitForApproval()).rejects.toThrow(/begin/);
  });
});
