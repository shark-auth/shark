/**
 * DPoP proof JWT generation per RFC 9449.
 *
 * Holds an ECDSA P-256 keypair and emits DPoP proof JWTs bound to an HTTP
 * method, URL, and optionally an access token (via the `ath` claim).
 *
 * @example
 * ```ts
 * const prover = await DPoPProver.generate();
 * const proof = await prover.createProof({ method: "POST", url: "https://auth.example/token" });
 * ```
 */

import {
  exportPKCS8,
  exportJWK,
  generateKeyPair,
  importPKCS8,
  type KeyLike,
  type JWK,
} from "jose";
import { SignJWT } from "jose";
import { DPoPError } from "./errors.js";

/** Options for {@link DPoPProver.createProof}. */
export interface CreateProofOptions {
  /** HTTP method (e.g. `"POST"`). Normalized to uppercase. */
  method: string;
  /** Target URL (without query/fragment per RFC 9449 §4.2). */
  url: string;
  /**
   * Optional server-provided DPoP nonce (from `DPoP-Nonce` response header).
   * Included in the proof as the `nonce` claim.
   */
  nonce?: string;
  /**
   * If provided, the proof binds to the token via the `ath` claim
   * (`base64url(sha256(accessToken))`).
   */
  accessToken?: string;
  /** Override `iat` (for testing only). */
  iat?: number;
  /** Override `jti` (for testing only). */
  jti?: string;
}

/** @internal */
interface EcPublicJwk {
  kty: "EC";
  crv: "P-256";
  x: string;
  y: string;
}

/**
 * RFC 9449 DPoP prover.
 *
 * Typically constructed via {@link DPoPProver.generate} (fresh P-256 key)
 * or {@link DPoPProver.fromPem} (load an existing PEM-encoded private key).
 */
export class DPoPProver {
  private readonly _privateKey: KeyLike;
  private readonly _publicJwk: EcPublicJwk;
  private readonly _thumbprint: string;

  private constructor(
    privateKey: KeyLike,
    publicJwk: EcPublicJwk,
    thumbprint: string
  ) {
    this._privateKey = privateKey;
    this._publicJwk = publicJwk;
    this._thumbprint = thumbprint;
  }

  // ------------------------------------------------------------------
  // Constructors
  // ------------------------------------------------------------------

  /**
   * Generate a fresh ECDSA P-256 keypair and return a new `DPoPProver`.
   */
  static async generate(): Promise<DPoPProver> {
    const { privateKey } = await generateKeyPair("ES256", {
      crv: "P-256",
      extractable: true,
    });
    return DPoPProver._fromPrivateKey(privateKey);
  }

  /**
   * Load a PEM-encoded PKCS#8 private key and return a new `DPoPProver`.
   *
   * @param pem  PKCS#8 PEM string (or `Buffer`).
   * @throws {@link DPoPError} if the key is not a P-256 EC key.
   */
  static async fromPem(pem: string | Buffer): Promise<DPoPProver> {
    const pemStr = Buffer.isBuffer(pem) ? pem.toString("utf-8") : pem;
    let privateKey: KeyLike;
    try {
      privateKey = await importPKCS8(pemStr, "ES256");
    } catch (err) {
      throw new DPoPError(
        `Failed to load private key: ${err instanceof Error ? err.message : String(err)}`
      );
    }
    return DPoPProver._fromPrivateKey(privateKey);
  }

  /** @internal */
  private static async _fromPrivateKey(privateKey: KeyLike): Promise<DPoPProver> {
    let rawJwk: JWK;
    try {
      rawJwk = await exportJWK(privateKey);
    } catch (err) {
      throw new DPoPError(
        `Failed to export JWK: ${err instanceof Error ? err.message : String(err)}`
      );
    }

    if (rawJwk.kty !== "EC" || rawJwk.crv !== "P-256") {
      throw new DPoPError("Private key must be ECDSA P-256 (secp256r1)");
    }
    if (typeof rawJwk.x !== "string" || typeof rawJwk.y !== "string") {
      throw new DPoPError("Private key missing EC public coordinates x/y");
    }

    const publicJwk: EcPublicJwk = {
      kty: "EC",
      crv: "P-256",
      x: rawJwk.x,
      y: rawJwk.y,
    };

    const thumbprint = await _jwkThumbprint(publicJwk);
    return new DPoPProver(privateKey, publicJwk, thumbprint);
  }

  // ------------------------------------------------------------------
  // Accessors
  // ------------------------------------------------------------------

  /**
   * The RFC 7638 JWK thumbprint (SHA-256, base64url) bound to this keypair.
   * Include as `dpop_jkt` in authorization requests that support it.
   */
  get jkt(): string {
    return this._thumbprint;
  }

  /**
   * The public key as a JWK (copy — no private material).
   */
  get publicJwk(): EcPublicJwk {
    return { ...this._publicJwk };
  }

  /**
   * Export the private key as an unencrypted PKCS#8 PEM string.
   *
   * Store this securely; it is sufficient to reconstruct the prover via
   * {@link DPoPProver.fromPem}.
   */
  async privateKeyPem(): Promise<string> {
    return exportPKCS8(this._privateKey);
  }

  // ------------------------------------------------------------------
  // Proof emission
  // ------------------------------------------------------------------

  /**
   * Build and sign a DPoP proof JWT.
   *
   * @param opts  Proof options including `method` and `url` (required).
   * @returns Compact serialised DPoP JWT string.
   * @throws {@link DPoPError} if `method` or `url` is empty.
   */
  async createProof(opts: CreateProofOptions): Promise<string> {
    const { method, nonce, accessToken, iat, jti } = opts;
    let { url } = opts;

    if (!method) throw new DPoPError("method is required");
    if (!url) throw new DPoPError("url is required");

    // RFC 9449 §4.2: The HTU claim MUST NOT contain any query or fragment.
    if (url.includes("?")) url = url.split("?")[0]!;
    if (url.includes("#")) url = url.split("#")[0]!;

    const claims: Record<string, unknown> = {
      jti: jti ?? _randomJti(),
      htm: method.toUpperCase(),
      htu: url,
      iat: iat ?? Math.floor(Date.now() / 1000),
    };

    if (nonce !== undefined) {
      claims["nonce"] = nonce;
    }
    if (accessToken !== undefined) {
      claims["ath"] = await _ath(accessToken);
    }

    try {
      return await new SignJWT(claims)
        .setProtectedHeader({
          typ: "dpop+jwt",
          alg: "ES256",
          jwk: this._publicJwk,
        })
        .sign(this._privateKey);
    } catch (err) {
      throw new DPoPError(
        `Failed to sign DPoP proof: ${err instanceof Error ? err.message : String(err)}`
      );
    }
  }
}

// ------------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------------

/** Compute `base64url(sha256(accessToken))` for the `ath` claim. */
async function _ath(accessToken: string): Promise<string> {
  const encoder = new TextEncoder();
  const digest = await crypto.subtle.digest(
    "SHA-256",
    encoder.encode(accessToken)
  );
  return _b64url(new Uint8Array(digest));
}

/** RFC 7638 JWK thumbprint for a P-256 EC public key. */
async function _jwkThumbprint(jwk: EcPublicJwk): Promise<string> {
  // Canonical form: only required members, lexicographic key order.
  const canonical = JSON.stringify({
    crv: jwk.crv,
    kty: jwk.kty,
    x: jwk.x,
    y: jwk.y,
  });
  const encoder = new TextEncoder();
  const digest = await crypto.subtle.digest("SHA-256", encoder.encode(canonical));
  return _b64url(new Uint8Array(digest));
}

/** base64url-encode a byte array without padding. */
function _b64url(bytes: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

/** Generate a random base64url JTI (~128 bits of entropy). */
function _randomJti(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return _b64url(bytes);
}
