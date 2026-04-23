/**
 * Tests for decodeAgentToken.
 */

import { describe, it, expect } from "vitest";
import { decodeAgentToken } from "../src/tokens.js";
import { TokenError } from "../src/errors.js";

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

function b64url(s: string): string {
  return Buffer.from(s)
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

function makeJwt(payload: Record<string, unknown>): string {
  const header = b64url(JSON.stringify({ alg: "RS256", typ: "JWT" }));
  const body = b64url(JSON.stringify(payload));
  return `${header}.${body}.fakesig`;
}

const NOW = Math.floor(Date.now() / 1000);

const BASE_CLAIMS = {
  sub: "agent_abc",
  iss: "https://auth.example",
  aud: "https://api.my-app.example",
  exp: NOW + 3600,
  iat: NOW,
  scope: "vault:read agent:exec",
  act: { sub: "user_42" },
  cnf: { jkt: "abc123thumbprint" },
  authorization_details: [{ type: "vault_access", connection_id: "conn_x" }],
};

// ------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------

describe("decodeAgentToken()", () => {
  it("decodes a well-formed token and returns typed claims", () => {
    const tok = makeJwt(BASE_CLAIMS);
    const claims = decodeAgentToken(tok);

    expect(claims.sub).toBe("agent_abc");
    expect(claims.iss).toBe("https://auth.example");
    expect(claims.aud).toBe("https://api.my-app.example");
    expect(claims.exp).toBe(BASE_CLAIMS.exp);
    expect(claims.iat).toBe(BASE_CLAIMS.iat);
    expect(claims.scope).toBe("vault:read agent:exec");
    expect(claims.act).toEqual({ sub: "user_42" });
    expect(claims.cnf).toEqual({ jkt: "abc123thumbprint" });
    expect(claims.authorization_details).toEqual([
      { type: "vault_access", connection_id: "conn_x" },
    ]);
  });

  it("exposes cnf.jkt via the cnf field", () => {
    const tok = makeJwt(BASE_CLAIMS);
    const claims = decodeAgentToken(tok);
    expect(claims.cnf?.jkt).toBe("abc123thumbprint");
  });

  it("handles array audience", () => {
    const tok = makeJwt({
      ...BASE_CLAIMS,
      aud: ["https://api.example", "https://other.example"],
    });
    const claims = decodeAgentToken(tok);
    expect(Array.isArray(claims.aud)).toBe(true);
    expect(claims.aud).toContain("https://api.example");
  });

  it("preserves all claims in raw", () => {
    const tok = makeJwt({ ...BASE_CLAIMS, custom_field: "custom_value" });
    const claims = decodeAgentToken(tok);
    expect(claims.raw["custom_field"]).toBe("custom_value");
  });

  it("scope is undefined when not in token", () => {
    const { scope: _scope, ...noScope } = BASE_CLAIMS;
    const tok = makeJwt(noScope);
    const claims = decodeAgentToken(tok);
    expect(claims.scope).toBeUndefined();
  });

  it("act is undefined when not in token", () => {
    const { act: _act, ...noAct } = BASE_CLAIMS;
    const tok = makeJwt(noAct);
    const claims = decodeAgentToken(tok);
    expect(claims.act).toBeUndefined();
  });

  it("throws TokenError on empty string", () => {
    expect(() => decodeAgentToken("")).toThrow(TokenError);
  });

  it("throws TokenError on non-JWT string", () => {
    expect(() => decodeAgentToken("not-a-jwt")).toThrow(TokenError);
  });

  it("throws TokenError on invalid base64url payload", () => {
    expect(() => decodeAgentToken("header.!!!.sig")).toThrow(TokenError);
  });

  it("throws TokenError when required claims are missing", () => {
    const tok = makeJwt({ sub: "agent_abc" }); // missing aud, iss, exp, iat
    expect(() => decodeAgentToken(tok)).toThrow(TokenError);
  });

  it("throws TokenError on two-part token", () => {
    expect(() => decodeAgentToken("header.payload")).toThrow(TokenError);
  });

  it("throws TokenError on four-part token", () => {
    expect(() => decodeAgentToken("a.b.c.d")).toThrow(TokenError);
  });
});
