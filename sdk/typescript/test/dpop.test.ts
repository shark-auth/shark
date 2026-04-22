/**
 * Tests for DPoPProver (RFC 9449).
 *
 * Mirrors the Python SDK test suite where applicable.
 */

import { describe, it, expect } from "vitest";
import { decodeProtectedHeader, decodeJwt } from "jose";
import { DPoPProver } from "../src/dpop.js";
import { DPoPError } from "../src/errors.js";

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

function b64urlDecode(s: string): Buffer {
  const padded = s + "=".repeat((4 - (s.length % 4)) % 4);
  return Buffer.from(padded.replace(/-/g, "+").replace(/_/g, "/"), "base64");
}

// ------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------

describe("DPoPProver.generate()", () => {
  it("emits a DPoP JWT with correct header fields", async () => {
    const prover = await DPoPProver.generate();
    const proof = await prover.createProof({
      method: "POST",
      url: "https://auth.example/oauth/token",
    });

    const header = decodeProtectedHeader(proof) as Record<string, unknown>;
    expect(header["typ"]).toBe("dpop+jwt");
    expect(header["alg"]).toBe("ES256");

    const jwk = header["jwk"] as Record<string, unknown>;
    expect(jwk["kty"]).toBe("EC");
    expect(jwk["crv"]).toBe("P-256");
    // Must not contain private key material
    expect(Object.keys(jwk).sort()).toEqual(["crv", "kty", "x", "y"].sort());
  });

  it("emits a DPoP JWT with correct payload claims", async () => {
    const prover = await DPoPProver.generate();
    const proof = await prover.createProof({
      method: "POST",
      url: "https://auth.example/oauth/token",
    });

    const payload = decodeJwt(proof);
    expect(payload["htm"]).toBe("POST");
    expect(payload["htu"]).toBe("https://auth.example/oauth/token");
    expect(typeof payload["jti"]).toBe("string");
    expect((payload["jti"] as string).length).toBeGreaterThanOrEqual(16);
    expect(typeof payload["iat"]).toBe("number");
    expect(payload["ath"]).toBeUndefined();
    expect(payload["nonce"]).toBeUndefined();
  });

  it("sets the ath claim when accessToken is provided", async () => {
    const prover = await DPoPProver.generate();
    const token = "abc.def.ghi";
    const proof = await prover.createProof({
      method: "GET",
      url: "https://api.example/data",
      accessToken: token,
    });

    const payload = decodeJwt(proof);
    // Expected: base64url(sha256(token))
    const encoder = new TextEncoder();
    const digest = await crypto.subtle.digest("SHA-256", encoder.encode(token));
    const expected = Buffer.from(digest)
      .toString("base64")
      .replace(/\+/g, "-")
      .replace(/\//g, "_")
      .replace(/=/g, "");

    expect(payload["ath"]).toBe(expected);
  });

  it("sets the nonce claim when nonce is provided", async () => {
    const prover = await DPoPProver.generate();
    const proof = await prover.createProof({
      method: "POST",
      url: "https://auth.example/t",
      nonce: "server-nonce-1",
    });

    const payload = decodeJwt(proof);
    expect(payload["nonce"]).toBe("server-nonce-1");
  });

  it("normalises htm to uppercase", async () => {
    const prover = await DPoPProver.generate();
    const proof = await prover.createProof({
      method: "post",
      url: "https://x.example/t",
    });
    const payload = decodeJwt(proof);
    expect(payload["htm"]).toBe("POST");
  });

  it("generates unique JTIs across many proofs", async () => {
    const prover = await DPoPProver.generate();
    const jtis = new Set<string>();
    for (let i = 0; i < 50; i++) {
      const proof = await prover.createProof({
        method: "POST",
        url: "https://x.example/t",
      });
      jtis.add(String(decodeJwt(proof)["jti"]));
    }
    expect(jtis.size).toBe(50);
  });

  it("jkt is stable across multiple proofs", async () => {
    const prover = await DPoPProver.generate();
    const t1 = prover.jkt;
    await prover.createProof({ method: "POST", url: "https://x.example/t" });
    expect(prover.jkt).toBe(t1);
    // SHA-256 thumbprint = 32 bytes = 43 unpadded base64url chars
    const raw = b64urlDecode(t1);
    expect(raw.byteLength).toBe(32);
  });
});

describe("DPoPProver.fromPem() round-trip", () => {
  it("preserves the thumbprint across export / import", async () => {
    const p1 = await DPoPProver.generate();
    const pem = await p1.privateKeyPem();
    const p2 = await DPoPProver.fromPem(pem);

    expect(p1.jkt).toBe(p2.jkt);
    // Both can produce valid proofs
    const proof = await p2.createProof({
      method: "POST",
      url: "https://x.example/t",
    });
    const header = decodeProtectedHeader(proof) as Record<string, unknown>;
    expect(header["alg"]).toBe("ES256");
  });

  it("rejects a non-P-256 PEM", async () => {
    // Generate a P-384 key using SubtleCrypto directly
    const { privateKey } = await crypto.subtle.generateKey(
      { name: "ECDSA", namedCurve: "P-384" },
      true,
      ["sign", "verify"]
    );
    const pkcs8 = await crypto.subtle.exportKey("pkcs8", privateKey);
    const pem = `-----BEGIN PRIVATE KEY-----\n${Buffer.from(pkcs8)
      .toString("base64")
      .match(/.{1,64}/g)!
      .join("\n")}\n-----END PRIVATE KEY-----`;

    await expect(DPoPProver.fromPem(pem)).rejects.toBeInstanceOf(DPoPError);
  });

  it("rejects empty method", async () => {
    const prover = await DPoPProver.generate();
    await expect(
      prover.createProof({ method: "", url: "https://x.example/t" })
    ).rejects.toBeInstanceOf(DPoPError);
  });

  it("rejects empty url", async () => {
    const prover = await DPoPProver.generate();
    await expect(
      prover.createProof({ method: "POST", url: "" })
    ).rejects.toBeInstanceOf(DPoPError);
  });
});

describe("DPoPProver publicJwk", () => {
  it("returns a copy — mutation does not affect the prover", async () => {
    const prover = await DPoPProver.generate();
    const jwk = prover.publicJwk;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (jwk as any)["crv"] = "P-384";
    // Internal state unaffected
    expect(prover.publicJwk["crv"]).toBe("P-256");
  });
});
