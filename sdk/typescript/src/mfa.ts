/**
 * MFA (TOTP + recovery codes) — wraps `/api/v1/auth/mfa/*`.
 *
 * Includes a Web-Crypto-only RFC 6238 {@link computeTotp} so callers can
 * drive `enroll` -> `verify` end-to-end in tests without a third-party OTP lib.
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";
import { Session } from "./auth.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Response from `enroll`. */
export interface MfaEnrollResult {
  /** Base32-encoded TOTP secret. */
  secret: string;
  /** `otpauth://` provisioning URI suitable for QR-encoding. */
  qr_uri: string;
  /** Mirror of `qr_uri` for naming-compat with other SDKs. */
  otpauth_url: string;
  [key: string]: unknown;
}

/** Response from `verify`. */
export interface MfaVerifyResult {
  mfa_enabled: boolean;
  recovery_codes: string[];
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Base32 decode (RFC 4648) — minimal, no deps
// ---------------------------------------------------------------------------

const BASE32_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

function base32Decode(input: string): Uint8Array {
  // Normalize: uppercase, strip whitespace, drop padding.
  const cleaned = input.toUpperCase().replace(/\s+/g, "").replace(/=+$/, "");
  let bits = 0;
  let value = 0;
  const out: number[] = [];
  for (const ch of cleaned) {
    const idx = BASE32_ALPHABET.indexOf(ch);
    if (idx < 0) throw new Error(`invalid base32 character: ${ch}`);
    value = (value << 5) | idx;
    bits += 5;
    if (bits >= 8) {
      bits -= 8;
      out.push((value >>> bits) & 0xff);
    }
  }
  return Uint8Array.from(out);
}

// ---------------------------------------------------------------------------
// HMAC-SHA1 — Web Crypto with Node fallback
// ---------------------------------------------------------------------------

interface SubtleLike {
  importKey(
    format: string,
    key: ArrayBuffer,
    algo: { name: string; hash: { name: string } },
    extractable: boolean,
    keyUsages: string[],
  ): Promise<unknown>;
  sign(algo: string, key: unknown, data: ArrayBuffer): Promise<ArrayBuffer>;
}

function getSubtle(): SubtleLike | undefined {
  const g = globalThis as { crypto?: { subtle?: unknown } };
  const s = g.crypto?.subtle;
  return s ? (s as SubtleLike) : undefined;
}

async function hmacSha1(key: Uint8Array, message: Uint8Array): Promise<Uint8Array> {
  // Prefer Web Crypto (works in browsers and Node 16+).
  const subtle = getSubtle();

  if (subtle) {
    try {
      const keyBuf = key.buffer.slice(
        key.byteOffset,
        key.byteOffset + key.byteLength,
      ) as ArrayBuffer;
      const msgBuf = message.buffer.slice(
        message.byteOffset,
        message.byteOffset + message.byteLength,
      ) as ArrayBuffer;
      const cryptoKey = await subtle.importKey(
        "raw",
        keyBuf,
        { name: "HMAC", hash: { name: "SHA-1" } },
        false,
        ["sign"],
      );
      const sig = await subtle.sign("HMAC", cryptoKey, msgBuf);
      return new Uint8Array(sig);
    } catch {
      // fall through to Node fallback
    }
  }

  // Node fallback — only when Web Crypto is unavailable or rejected SHA-1.
  // `window` is browser-only; using `typeof` keeps the check tree-shakeable.
  const isBrowser =
    typeof (globalThis as { window?: unknown }).window !== "undefined";
  if (!isBrowser) {
    const nodeCrypto = await import("node:crypto");
    const h = nodeCrypto.createHmac("sha1", Buffer.from(key));
    h.update(Buffer.from(message));
    return new Uint8Array(h.digest());
  }

  throw new Error("HMAC-SHA1 unavailable: no Web Crypto subtle, no Node crypto");
}

// ---------------------------------------------------------------------------
// computeTotp — RFC 6238
// ---------------------------------------------------------------------------

/**
 * Compute a 6-digit RFC 6238 TOTP code.
 *
 * Uses HMAC-SHA1 over a 30-second time-step (the SharkAuth default). The
 * secret is base32-encoded, matching what the server returns from
 * {@link MfaClient.enroll} and embeds in the `otpauth://` QR URI.
 *
 * @param secret  Base32-encoded shared secret (case-insensitive, optional `=` padding).
 * @param t       Optional UNIX timestamp in seconds; defaults to `Date.now()/1000`.
 * @returns Zero-padded 6-digit code.
 */
export async function computeTotp(secret: string, t?: number): Promise<string> {
  const now = t ?? Date.now() / 1000;
  const key = base32Decode(secret);

  // Counter = floor(now / 30), packed as big-endian uint64.
  const counter = Math.floor(now / 30);
  const msg = new Uint8Array(8);
  // JavaScript bitwise ops are 32-bit; split into hi/lo for safety.
  const hi = Math.floor(counter / 0x100000000);
  const lo = counter >>> 0;
  msg[0] = (hi >>> 24) & 0xff;
  msg[1] = (hi >>> 16) & 0xff;
  msg[2] = (hi >>> 8) & 0xff;
  msg[3] = hi & 0xff;
  msg[4] = (lo >>> 24) & 0xff;
  msg[5] = (lo >>> 16) & 0xff;
  msg[6] = (lo >>> 8) & 0xff;
  msg[7] = lo & 0xff;

  const digest = await hmacSha1(key, msg);
  const offset = digest[digest.length - 1] & 0x0f;
  const code =
    (((digest[offset] & 0x7f) << 24) |
      ((digest[offset + 1] & 0xff) << 16) |
      ((digest[offset + 2] & 0xff) << 8) |
      (digest[offset + 3] & 0xff)) %
    1_000_000;
  return code.toString().padStart(6, "0");
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/** Options for {@link MfaClient}. */
export interface MfaClientOptions {
  /** Shared session — typically the one from {@link AuthClient}. */
  session?: Session;
}

/**
 * Manage TOTP MFA on the authenticated user's account.
 *
 * All endpoints assume the underlying {@link Session} carries a valid session
 * cookie (typically planted by `AuthClient.login`).
 */
export class MfaClient {
  private static readonly PREFIX = "/api/v1/auth/mfa";
  private readonly _base: string;
  readonly session: Session;

  constructor(baseUrl: string, opts: MfaClientOptions = {}) {
    this._base = baseUrl.replace(/\/+$/, "");
    this.session = opts.session ?? new Session();
  }

  private async _throw(status: number, text: string): Promise<never> {
    let code = "api_error";
    let message = text.slice(0, 300);
    try {
      const body = JSON.parse(text) as {
        error?: string | { code?: string; message?: string };
        message?: string;
      };
      if (typeof body.error === "string") code = body.error;
      else if (body.error && typeof body.error === "object" && body.error.code)
        code = body.error.code;
      if (body.message) message = body.message;
      else if (body.error && typeof body.error === "object" && body.error.message)
        message = body.error.message;
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  private async _request(
    method: string,
    path: string,
    json?: unknown,
  ): Promise<{ status: number; text: string; json<T>(): T }> {
    const url = `${this._base}${path}`;
    const headers: Record<string, string> = {};
    const cookie = this.session.cookieHeader();
    if (cookie) headers["Cookie"] = cookie;
    const reqOpts: Parameters<typeof httpRequest>[1] = { method, headers };
    if (json !== undefined) {
      headers["Content-Type"] = "application/json";
      reqOpts.body = JSON.stringify(json);
    }
    const resp = await httpRequest(url, reqOpts);
    const setCookie = resp.headers["set-cookie"];
    if (setCookie) this.session.ingestSetCookie(setCookie);
    return resp;
  }

  // ------------------------------------------------------------------
  // Enroll / Verify (full-session endpoints)
  // ------------------------------------------------------------------

  /**
   * Generate a TOTP secret + QR provisioning URI. Returns
   * `{ secret, qr_uri, otpauth_url }`. The server returns `secret` and `qr_uri`;
   * `otpauth_url` is mirrored from `qr_uri` for naming-compat with other SDKs.
   */
  async enroll(): Promise<MfaEnrollResult> {
    const resp = await this._request("POST", `${MfaClient.PREFIX}/enroll`);
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const data = resp.json<MfaEnrollResult & { otpauth_url?: string; qr_uri?: string }>();
    if (!data.otpauth_url && data.qr_uri) data.otpauth_url = data.qr_uri;
    return data;
  }

  /** Confirm enrollment with the first TOTP code. */
  async verify(code: string): Promise<MfaVerifyResult> {
    const resp = await this._request("POST", `${MfaClient.PREFIX}/verify`, { code });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<MfaVerifyResult>();
  }

  // ------------------------------------------------------------------
  // Login-time challenge (partial session) + disable + recovery codes
  // ------------------------------------------------------------------

  /**
   * Upgrade a partial (post-login) session by submitting a TOTP code.
   * The server upgrades the session's `mfa_passed` flag in place — no body returned.
   */
  async challenge(code: string): Promise<void> {
    const resp = await this._request("POST", `${MfaClient.PREFIX}/challenge`, { code });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  /**
   * Disable MFA for the authenticated user. The user must re-prove possession
   * of the device by supplying a current TOTP code.
   */
  async disable(code: string): Promise<void> {
    const resp = await this._request("DELETE", `${MfaClient.PREFIX}/`, { code });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  /**
   * Regenerate the current user's MFA recovery codes. Note: the server
   * exposes this as `GET` (not POST) — calling it invalidates the
   * previously-issued set and returns a fresh list.
   */
  async regenerateRecoveryCodes(): Promise<string[]> {
    const resp = await this._request("GET", `${MfaClient.PREFIX}/recovery-codes`);
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const data = resp.json<unknown>();
    if (Array.isArray(data)) return data.map(String);
    if (data && typeof data === "object") {
      const obj = data as Record<string, unknown>;
      const codes = obj["recovery_codes"] ?? obj["codes"];
      if (Array.isArray(codes)) return codes.map(String);
    }
    return [];
  }
}
