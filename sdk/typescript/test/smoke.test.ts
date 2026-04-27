/**
 * Smoke tests — one happy-path call per untested namespace.
 *
 * Verifies HTTP method + URL construction only. No backend behavior.
 * Pattern: vi.stubGlobal("fetch", ...) — same as users.test.ts.
 */

import { describe, it, expect, vi, afterEach } from "vitest";
import { SharkAPIError } from "../src/errors.js";

const BASE = "https://auth.example.com";
const KEY = "sk_live_test";

function makeResp(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

afterEach(() => vi.unstubAllGlobals());

function stubFetch(resp: Response) {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(resp));
}

function lastFetchCall(): [string, RequestInit] {
  return (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit];
}

// ---------------------------------------------------------------------------
// apiKeys
// ---------------------------------------------------------------------------

describe("ApiKeysClient.list()", () => {
  it("GET /api/v1/api-keys returns key list", async () => {
    const { ApiKeysClient } = await import("../src/apiKeys.js");
    stubFetch(makeResp(200, [{ id: "key_1", name: "ci" }]));

    const result = await new ApiKeysClient({ baseUrl: BASE, adminKey: KEY }).list();
    expect(result[0].id).toBe("key_1");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/api-keys");
    expect(init.method).toBe("GET");
  });
});

// ---------------------------------------------------------------------------
// apps
// ---------------------------------------------------------------------------

describe("AppsClient.list()", () => {
  it("GET /api/v1/admin/apps returns app list", async () => {
    const { AppsClient } = await import("../src/apps.js");
    stubFetch(makeResp(200, [{ id: "app_1", name: "My App", integration_mode: "custom" }]));

    const result = await new AppsClient({ baseUrl: BASE, adminKey: KEY }).list();
    expect(result[0].id).toBe("app_1");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/admin/apps");
    expect(init.method).toBe("GET");
  });
});

// ---------------------------------------------------------------------------
// audit
// ---------------------------------------------------------------------------

describe("AuditClient.list()", () => {
  it("GET /api/v1/audit-logs returns events", async () => {
    const { AuditClient } = await import("../src/audit.js");
    stubFetch(makeResp(200, { events: [], next_cursor: null }));

    const result = await new AuditClient({ baseUrl: BASE, adminKey: KEY }).list();
    expect(result.events).toEqual([]);

    const [url] = lastFetchCall();
    expect(url).toContain("/api/v1/audit-logs");
  });
});

// ---------------------------------------------------------------------------
// auth
// ---------------------------------------------------------------------------

describe("AuthClient.getMe()", () => {
  it("GET /api/v1/auth/me returns user", async () => {
    const { AuthClient } = await import("../src/auth.js");
    stubFetch(makeResp(200, { id: "usr_1", email: "a@b.com" }));

    const result = await new AuthClient(BASE).getMe();
    expect((result as Record<string, unknown>).email).toBe("a@b.com");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/auth/me");
    expect(init.method).toBe("GET");
  });
});

// ---------------------------------------------------------------------------
// consents
// ---------------------------------------------------------------------------

describe("ConsentsClient.list()", () => {
  it("GET /api/v1/auth/consents returns consent list", async () => {
    const { ConsentsClient } = await import("../src/consents.js");
    stubFetch(makeResp(200, [{ id: "con_1", app_id: "app_1", scopes: [] }]));

    const result = await new ConsentsClient(BASE).list();
    expect(result[0].id).toBe("con_1");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/auth/consents");
    expect(init.method).toBe("GET");
  });
});

// ---------------------------------------------------------------------------
// dcr
// ---------------------------------------------------------------------------

describe("DcrClient.get()", () => {
  it("GET /oauth/register/{client_id} returns client metadata", async () => {
    const { DcrClient } = await import("../src/dcr.js");
    const clientId = "c_abc";
    stubFetch(makeResp(200, { client_id: clientId, client_name: "agent" }));

    const result = await new DcrClient(BASE).get(clientId, "rat_token");
    expect(result.client_id).toBe(clientId);

    const [url, init] = lastFetchCall();
    expect(url).toContain(`/oauth/register/${clientId}`);
    expect(init.method).toBe("GET");
  });
});

// ---------------------------------------------------------------------------
// exchange
// ---------------------------------------------------------------------------

describe("exchangeToken()", () => {
  it("POST /api/v1/token-exchange returns new token", async () => {
    const { exchangeToken } = await import("../src/exchange.js");
    stubFetch(makeResp(200, { access_token: "new_tok", token_type: "Bearer" }));

    const result = await exchangeToken({
      authUrl: BASE,
      clientId: "cli_test",
      subjectToken: "old_tok",
      subjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
    });
    // exchangeToken maps access_token → accessToken (camelCase)
    expect(result.accessToken).toBe("new_tok");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/token");
    expect(init.method).toBe("POST");
  });
});

// ---------------------------------------------------------------------------
// magicLink
// ---------------------------------------------------------------------------

describe("MagicLinkClient.sendMagicLink()", () => {
  it("POST /api/v1/auth/magic-link/send constructs correct request", async () => {
    const { MagicLinkClient } = await import("../src/magicLink.js");
    stubFetch(makeResp(200, { sent: true, expires_at: "2026-04-28T00:00:00Z" }));

    await new MagicLinkClient({ baseUrl: BASE }).sendMagicLink("user@example.com");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/auth/magic-link/send");
    expect(init.method).toBe("POST");
    const body = JSON.parse(init.body as string) as { email: string };
    expect(body.email).toBe("user@example.com");
  });
});

// ---------------------------------------------------------------------------
// mfa
// ---------------------------------------------------------------------------

describe("MfaClient.enroll()", () => {
  it("POST /api/v1/auth/mfa/enroll returns secret + qr_url", async () => {
    const { MfaClient } = await import("../src/mfa.js");
    stubFetch(makeResp(200, { secret: "JBSWY3DPEHPK3PXP", qr_url: "otpauth://totp/shark:a@b.com?secret=X" }));

    const result = await new MfaClient(BASE).enroll();
    expect((result as Record<string, unknown>).secret).toBeTruthy();

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/auth/mfa/enroll");
    expect(init.method).toBe("POST");
  });
});

// ---------------------------------------------------------------------------
// oauth
// ---------------------------------------------------------------------------

describe("OAuthClient.introspectToken()", () => {
  it("POST introspect endpoint returns active=true", async () => {
    const { OAuthClient } = await import("../src/oauth.js");
    stubFetch(makeResp(200, { active: true, sub: "usr_1" }));

    const result = await new OAuthClient({ baseUrl: BASE, adminKey: KEY }).introspectToken("tok_abc");
    expect(result.active).toBe(true);

    const [, init] = lastFetchCall();
    expect(init.method).toBe("POST");
  });
});

// ---------------------------------------------------------------------------
// organizations
// ---------------------------------------------------------------------------

describe("OrganizationsClient.list()", () => {
  it("GET /api/v1/admin/organizations returns org list", async () => {
    const { OrganizationsClient } = await import("../src/organizations.js");
    stubFetch(makeResp(200, [{ id: "org_1", name: "Acme", slug: "acme", created_at: "2026-04-01T00:00:00Z" }]));

    const result = await new OrganizationsClient({ baseUrl: BASE, adminKey: KEY }).list();
    expect(result[0].id).toBe("org_1");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/admin/organizations");
    expect(init.method).toBe("GET");
  });
});

// ---------------------------------------------------------------------------
// rbac
// ---------------------------------------------------------------------------

describe("RbacClient.listRoles()", () => {
  it("GET /api/v1/roles returns role list", async () => {
    const { RbacClient } = await import("../src/rbac.js");
    stubFetch(makeResp(200, [{ id: "role_1", name: "admin", description: "" }]));

    const result = await new RbacClient({ baseUrl: BASE, adminKey: KEY }).listRoles();
    expect(result[0].id).toBe("role_1");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/roles");
    expect(init.method).toBe("GET");
  });
});

// ---------------------------------------------------------------------------
// sessions
// ---------------------------------------------------------------------------

describe("SessionsClient.list()", () => {
  it("GET /api/v1/auth/sessions returns session list", async () => {
    const { SessionsClient } = await import("../src/sessions.js");
    stubFetch(makeResp(200, [{ id: "sess_1", user_id: "usr_1", created_at: "2026-04-01T00:00:00Z", last_active_at: "2026-04-27T00:00:00Z", user_agent: "curl" }]));

    const result = await new SessionsClient(BASE).list();
    expect(result[0].id).toBe("sess_1");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/auth/sessions");
    expect(init.method).toBe("GET");
  });
});

// ---------------------------------------------------------------------------
// webhooks
// ---------------------------------------------------------------------------

describe("WebhooksClient.list()", () => {
  it("GET /api/v1/admin/webhooks returns webhook list", async () => {
    const { WebhooksClient } = await import("../src/webhooks.js");
    stubFetch(makeResp(200, [{ id: "wh_1", url: "https://hooks.example.com", events: ["user.created"], enabled: true, created_at: "2026-04-01T00:00:00Z" }]));

    const result = await new WebhooksClient({ baseUrl: BASE, adminKey: KEY }).list();
    expect(result[0].id).toBe("wh_1");

    const [url, init] = lastFetchCall();
    expect(url).toContain("/api/v1/admin/webhooks");
    expect(init.method).toBe("GET");
  });
});
