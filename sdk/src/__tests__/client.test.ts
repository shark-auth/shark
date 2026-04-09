import { describe, it, expect, vi } from "vitest";
import { createSharkAuth, SharkError } from "../index.js";

function mockFetch(status: number, body: unknown, headers?: Headers) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    text: () => Promise.resolve(JSON.stringify(body)),
    json: () => Promise.resolve(body),
    headers: headers ?? new Headers(),
  });
}

describe("createSharkAuth", () => {
  it("creates an instance with all sub-clients", () => {
    const shark = createSharkAuth({ baseURL: "http://localhost:8080" });
    expect(shark.mfa).toBeDefined();
    expect(shark.passkey).toBeDefined();
    expect(shark.oauth).toBeDefined();
    expect(shark.magicLink).toBeDefined();
    expect(shark.login).toBeTypeOf("function");
    expect(shark.signup).toBeTypeOf("function");
    expect(shark.logout).toBeTypeOf("function");
    expect(shark.me).toBeTypeOf("function");
    expect(shark.check).toBeTypeOf("function");
  });
});

describe("auth", () => {
  it("signup returns user", async () => {
    const user = { id: "usr_123", email: "test@test.com", mfaEnabled: false };
    const fetch = mockFetch(201, user);
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.signup({ email: "test@test.com", password: "password123" });
    expect(result).toEqual(user);
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/signup",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
      }),
    );
  });

  it("login returns user when MFA not enabled", async () => {
    const user = { id: "usr_123", email: "test@test.com", mfaEnabled: false };
    const fetch = mockFetch(200, user);
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.login({ email: "test@test.com", password: "password123" });
    expect(result.mfaRequired).toBe(false);
    if (!result.mfaRequired) {
      expect(result.user.id).toBe("usr_123");
    }
  });

  it("login returns mfaRequired when MFA is enabled", async () => {
    const fetch = mockFetch(200, { mfaRequired: true });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.login({ email: "test@test.com", password: "password123" });
    expect(result.mfaRequired).toBe(true);
  });

  it("login throws SharkError on invalid credentials", async () => {
    const fetch = mockFetch(401, { error: "invalid_credentials", message: "Invalid email or password" });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await expect(shark.login({ email: "test@test.com", password: "wrong" }))
      .rejects.toThrow(SharkError);

    try {
      await shark.login({ email: "test@test.com", password: "wrong" });
    } catch (err) {
      expect(err).toBeInstanceOf(SharkError);
      expect((err as SharkError).status).toBe(401);
      expect((err as SharkError).code).toBe("invalid_credentials");
    }
  });

  it("me returns user", async () => {
    const user = { id: "usr_123", email: "test@test.com", mfaEnabled: false };
    const fetch = mockFetch(200, user);
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.me();
    expect(result.id).toBe("usr_123");
  });

  it("me throws 403 when MFA incomplete", async () => {
    const fetch = mockFetch(403, { error: "mfa_required", message: "MFA verification required" });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await expect(shark.me()).rejects.toThrow(SharkError);
  });

  it("logout calls POST /logout", async () => {
    const fetch = mockFetch(200, {});
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await shark.logout();
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/logout",
      expect.objectContaining({ method: "POST" }),
    );
  });
});

describe("check", () => {
  it("returns authenticated: true when session is valid", async () => {
    const user = { id: "usr_123", email: "test@test.com", mfaEnabled: false };
    const fetch = mockFetch(200, user);
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.check();
    expect(result.authenticated).toBe(true);
    expect(result.user?.id).toBe("usr_123");
  });

  it("returns authenticated: false on 401", async () => {
    const fetch = mockFetch(401, { error: "unauthorized", message: "No valid session" });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.check();
    expect(result.authenticated).toBe(false);
    expect(result.user).toBeNull();
  });

  it("returns authenticated: false on 403 (MFA incomplete)", async () => {
    const fetch = mockFetch(403, { error: "mfa_required", message: "MFA verification required" });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.check();
    expect(result.authenticated).toBe(false);
    expect(result.user).toBeNull();
  });

  it("rethrows non-auth errors", async () => {
    const fetch = mockFetch(500, { error: "internal_error", message: "Internal server error" });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await expect(shark.check()).rejects.toThrow(SharkError);
  });
});

describe("mfa", () => {
  it("enroll returns secret and qr_uri", async () => {
    const fetch = mockFetch(200, { secret: "JBSWY3DPEHPK3PXP", qr_uri: "otpauth://totp/..." });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.mfa.enroll();
    expect(result.secret).toBe("JBSWY3DPEHPK3PXP");
    expect(result.qr_uri).toContain("otpauth://");
  });

  it("verify returns recovery codes", async () => {
    const fetch = mockFetch(200, { mfa_enabled: true, recovery_codes: ["abc123", "def456"] });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.mfa.verify({ code: "123456" });
    expect(result.mfa_enabled).toBe(true);
    expect(result.recovery_codes).toHaveLength(2);
  });

  it("challenge upgrades session and returns user", async () => {
    const user = { id: "usr_123", email: "test@test.com", mfaEnabled: true };
    const fetch = mockFetch(200, user);
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.mfa.challenge({ code: "123456" });
    expect(result.id).toBe("usr_123");
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/mfa/challenge",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("challenge throws on invalid code", async () => {
    const fetch = mockFetch(401, { error: "invalid_code", message: "Invalid TOTP code" });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await expect(shark.mfa.challenge({ code: "000000" })).rejects.toThrow(SharkError);
  });

  it("useRecoveryCode works as MFA alternative", async () => {
    const user = { id: "usr_123", email: "test@test.com", mfaEnabled: true };
    const fetch = mockFetch(200, user);
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    const result = await shark.mfa.useRecoveryCode({ code: "abc123" });
    expect(result.id).toBe("usr_123");
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/mfa/recovery",
      expect.objectContaining({ method: "POST" }),
    );
  });
});

describe("oauth", () => {
  it("getURL returns correct redirect URL", () => {
    const shark = createSharkAuth({ baseURL: "http://localhost:8080" });

    expect(shark.oauth.getURL("google")).toBe("http://localhost:8080/api/v1/auth/oauth/google");
    expect(shark.oauth.getURL("github")).toBe("http://localhost:8080/api/v1/auth/oauth/github");
    expect(shark.oauth.getURL("apple")).toBe("http://localhost:8080/api/v1/auth/oauth/apple");
    expect(shark.oauth.getURL("discord")).toBe("http://localhost:8080/api/v1/auth/oauth/discord");
  });

  it("strips trailing slash from baseURL", () => {
    const shark = createSharkAuth({ baseURL: "http://localhost:8080/" });
    expect(shark.oauth.getURL("google")).toBe("http://localhost:8080/api/v1/auth/oauth/google");
  });
});

describe("magicLink", () => {
  it("send calls correct endpoint", async () => {
    const fetch = mockFetch(200, { message: "Check your email" });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await shark.magicLink.send({ email: "test@test.com" });
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/magic-link/send",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ email: "test@test.com" }),
      }),
    );
  });
});

describe("password reset", () => {
  it("sendPasswordReset calls correct endpoint", async () => {
    const fetch = mockFetch(200, { message: "If an account with that email exists..." });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await shark.sendPasswordReset({ email: "test@test.com" });
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/password/send-reset-link",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("resetPassword calls correct endpoint", async () => {
    const fetch = mockFetch(200, { message: "Password has been reset successfully" });
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await shark.resetPassword({ token: "abc123", password: "newpassword" });
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/password/reset",
      expect.objectContaining({ method: "POST" }),
    );
  });
});

describe("HttpClient", () => {
  it("includes credentials: include on every request", async () => {
    const fetch = mockFetch(200, {});
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await shark.me();
    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ credentials: "include" }),
    );
  });

  it("sets Content-Type: application/json", async () => {
    const fetch = mockFetch(200, {});
    const shark = createSharkAuth({ baseURL: "http://localhost:8080", fetch });

    await shark.logout();
    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({ "Content-Type": "application/json" }),
      }),
    );
  });
});
