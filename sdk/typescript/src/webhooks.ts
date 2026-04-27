/**
 * Webhooks admin client + HMAC-SHA256 signature verification helper.
 *
 * Endpoints under `/api/v1/admin/webhooks/` (verified in
 * `internal/api/router.go` lines 510-522):
 *
 *  - POST   /                                — create
 *  - GET    /                                — list
 *  - GET    /{id}                            — get
 *  - PATCH  /{id}                            — update
 *  - DELETE /{id}                            — delete
 *  - POST   /{id}/test                       — fire a test delivery
 *  - GET    /{id}/deliveries                 — list past deliveries
 *  - POST   /{id}/deliveries/{deliveryId}/replay — replay a delivery
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Webhook subscription row as returned by the server. */
export interface Webhook {
  id: string;
  url: string;
  events: string[];
  enabled?: boolean;
  description?: string;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
}

/** Webhook delivery record. */
export interface WebhookDelivery {
  id: string;
  webhook_id?: string;
  event_type?: string;
  status?: string;
  status_code?: number;
  attempts?: number;
  delivered_at?: string;
  created_at?: string;
  [key: string]: unknown;
}

/** Input for {@link WebhooksClient.register}. */
export interface RegisterWebhookInput {
  url: string;
  events: string[];
  secret?: string;
  description?: string;
  enabled?: boolean;
  [key: string]: unknown;
}

/** Input for {@link WebhooksClient.update} — partial. */
export type UpdateWebhookInput = Partial<RegisterWebhookInput>;

/** Options for {@link WebhooksClient}. */
export interface WebhooksClientOptions {
  baseUrl: string;
  adminKey: string;
}

// ---------------------------------------------------------------------------
// Module-level signature verification helper
// ---------------------------------------------------------------------------

const HEX_CHARS = "0123456789abcdef";

function bytesToHex(bytes: ArrayBuffer | Uint8Array): string {
  const view = bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  let out = "";
  for (let i = 0; i < view.length; i++) {
    const b = view[i] ?? 0;
    out += HEX_CHARS[(b >> 4) & 0xf];
    out += HEX_CHARS[b & 0xf];
  }
  return out;
}

function utf8Encode(input: string | Uint8Array): Uint8Array {
  if (input instanceof Uint8Array) return input;
  return new TextEncoder().encode(input);
}

function concatBytes(a: Uint8Array, b: Uint8Array): Uint8Array {
  const out = new Uint8Array(a.length + b.length);
  out.set(a, 0);
  out.set(b, a.length);
  return out;
}

/** Constant-time hex string comparison. */
function timingSafeEqualHex(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) {
    diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }
  return diff === 0;
}

async function hmacSha256Hex(secret: Uint8Array, data: Uint8Array): Promise<string> {
  // Copy into fresh ArrayBuffers to satisfy lib.dom BufferSource typing
  // across Node and DOM lib variants.
  const secretBuf = secret.buffer.slice(
    secret.byteOffset,
    secret.byteOffset + secret.byteLength,
  );
  const dataBuf = data.buffer.slice(
    data.byteOffset,
    data.byteOffset + data.byteLength,
  );
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const subtle: any = (globalThis as any).crypto?.subtle;
  if (!subtle) throw new Error("Web Crypto subtle API not available");
  const key = await subtle.importKey(
    "raw",
    secretBuf,
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const sigBuf = (await subtle.sign("HMAC", key, dataBuf)) as ArrayBuffer;
  return bytesToHex(sigBuf);
}

/**
 * Verify an HMAC-SHA256 signature on an incoming webhook payload.
 *
 * Two header formats are supported:
 *
 *   1. Stripe-style: `t=<unix_ts>,v1=<hex_hmac>` (extra `vN=` segments
 *      ignored). Signed payload is `${ts}.<payload>`. Timestamp is
 *      enforced against `toleranceSeconds` to thwart replay attacks.
 *   2. Raw hex digest of HMAC-SHA256(secret, payload). Optional
 *      `sha256=` prefix is stripped. No timestamp enforced.
 *
 * Uses Web Crypto (`crypto.subtle`) — works in browsers and Node 18+.
 *
 * @param payload          Raw request body bytes or string.
 * @param headerSignature  Signature header value sent by SharkAuth.
 * @param secret           Shared HMAC secret.
 * @param toleranceSeconds Max age of the timestamp (Stripe-style only). Default 300.
 * @returns `true` if valid, `false` otherwise.
 */
export async function verifySignature(
  payload: Uint8Array | string,
  headerSignature: string,
  secret: string,
  toleranceSeconds: number = 300,
): Promise<boolean> {
  if (!headerSignature || typeof headerSignature !== "string") return false;
  const secretBytes = utf8Encode(secret);
  const payloadBytes = utf8Encode(payload);

  // Format 1 — comma-separated key=value pairs.
  if (headerSignature.includes("=") && headerSignature.includes(",")) {
    const parts: Record<string, string> = {};
    for (const seg of headerSignature.split(",")) {
      const trimmed = seg.trim();
      const eq = trimmed.indexOf("=");
      if (eq > 0) {
        parts[trimmed.slice(0, eq).trim()] = trimmed.slice(eq + 1).trim();
      }
    }
    const ts = parts["t"];
    const v1 = parts["v1"];
    if (ts === undefined || v1 === undefined) return false;
    const tsInt = Number.parseInt(ts, 10);
    if (!Number.isFinite(tsInt)) return false;
    if (toleranceSeconds > 0) {
      const nowSec = Math.floor(Date.now() / 1000);
      if (Math.abs(nowSec - tsInt) > toleranceSeconds) return false;
    }
    const signed = concatBytes(utf8Encode(`${ts}.`), payloadBytes);
    const expected = await hmacSha256Hex(secretBytes, signed);
    return timingSafeEqualHex(expected, v1.toLowerCase());
  }

  // Format 2 — raw hex digest, optionally prefixed with "sha256=".
  let candidate = headerSignature.trim();
  if (candidate.toLowerCase().startsWith("sha256=")) {
    candidate = candidate.slice("sha256=".length);
  }
  const expected = await hmacSha256Hex(secretBytes, payloadBytes);
  return timingSafeEqualHex(expected, candidate.toLowerCase());
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

const PREFIX = "/api/v1/admin/webhooks";

/** Admin client for webhook subscriptions. */
export class WebhooksClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: WebhooksClientOptions) {
    this._base = opts.baseUrl.replace(/\/+$/, "");
    this._key = opts.adminKey;
  }

  private _auth(): Record<string, string> {
    return { Authorization: `Bearer ${this._key}` };
  }

  private async _throw(status: number, text: string): Promise<never> {
    let code = "api_error";
    let message = text.slice(0, 300);
    try {
      const body = JSON.parse(text) as {
        error?: { code?: string; message?: string } | string;
        message?: string;
      };
      if (typeof body.error === "object" && body.error) {
        if (body.error.code) code = body.error.code;
        if (body.error.message) message = body.error.message;
      } else if (typeof body.error === "string") {
        message = body.error;
      } else if (body.message) {
        message = body.message;
      }
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  // --------------------------------------------------------------- CRUD

  /** Create a webhook subscription. */
  async register(input: RegisterWebhookInput): Promise<Webhook> {
    const url = `${this._base}${PREFIX}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    const raw = resp.json<Webhook | { data: Webhook }>();
    if (raw && typeof raw === "object" && "data" in raw) return (raw as { data: Webhook }).data;
    return raw as Webhook;
  }

  /** List all webhook subscriptions. */
  async list(): Promise<Webhook[]> {
    const url = `${this._base}${PREFIX}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<Webhook[] | { data?: Webhook[] }>();
    if (Array.isArray(body)) return body;
    return body?.data ?? [];
  }

  /** Fetch a single webhook subscription. */
  async get(webhookId: string): Promise<Webhook> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(webhookId)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const raw = resp.json<Webhook | { data: Webhook }>();
    if (raw && typeof raw === "object" && "data" in raw) return (raw as { data: Webhook }).data;
    return raw as Webhook;
  }

  /** Patch a webhook subscription (PATCH /{id}). */
  async update(webhookId: string, fields: UpdateWebhookInput): Promise<Webhook> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(webhookId)}`;
    const resp = await httpRequest(url, {
      method: "PATCH",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(fields),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const raw = resp.json<Webhook | { data: Webhook }>();
    if (raw && typeof raw === "object" && "data" in raw) return (raw as { data: Webhook }).data;
    return raw as Webhook;
  }

  /** Delete a webhook subscription. */
  async delete(webhookId: string): Promise<void> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(webhookId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // --------------------------------------------------------- Test + deliveries

  /** Fire a synthetic test event (POST /{id}/test). */
  async sendTest(
    webhookId: string,
    opts?: { eventType?: string },
  ): Promise<Record<string, unknown>> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(webhookId)}/test`;
    const body: Record<string, unknown> = {};
    if (opts?.eventType !== undefined) body.event_type = opts.eventType;
    const hasBody = Object.keys(body).length > 0;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: hasBody
        ? { ...this._auth(), "Content-Type": "application/json" }
        : this._auth(),
      body: hasBody ? JSON.stringify(body) : undefined,
    });
    if (![200, 201, 202].includes(resp.status))
      await this._throw(resp.status, resp.text);
    try {
      return resp.json<Record<string, unknown>>();
    } catch {
      return {};
    }
  }

  /** Replay a previous delivery. */
  async replay(webhookId: string, deliveryId: string): Promise<Record<string, unknown>> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(
      webhookId,
    )}/deliveries/${encodeURIComponent(deliveryId)}/replay`;
    const resp = await httpRequest(url, { method: "POST", headers: this._auth() });
    if (![200, 201, 202].includes(resp.status))
      await this._throw(resp.status, resp.text);
    try {
      return resp.json<Record<string, unknown>>();
    } catch {
      return {};
    }
  }

  /** List past deliveries for a webhook. */
  async listDeliveries(
    webhookId: string,
    opts?: { limit?: number },
  ): Promise<WebhookDelivery[]> {
    const limit = opts?.limit ?? 50;
    const qs = new URLSearchParams({ limit: String(limit) });
    const url = `${this._base}${PREFIX}/${encodeURIComponent(
      webhookId,
    )}/deliveries?${qs.toString()}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<WebhookDelivery[] | { data?: WebhookDelivery[] }>();
    if (Array.isArray(body)) return body;
    return body?.data ?? [];
  }
}
