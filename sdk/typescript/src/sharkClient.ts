/**
 * High-level client for agents with automatic DPoP signing.
 */

import { DPoPProver } from "./dpop.js";

/** Options for {@link SharkClient}. */
export interface SharkClientOptions {
  /** Bearer access token for the calling agent. */
  accessToken: string;
  /**
   * Optional {@link DPoPProver} — when supplied, the client will
   * auto-sign every outgoing request with a DPoP proof JWT.
   */
  dpopProver?: DPoPProver;
  /** Custom User-Agent header. */
  userAgent?: string;
}

/**
 * A DPoP-aware HTTP client for SharkAuth agents.
 *
 * Wraps the built-in `fetch` API to automatically inject
 * `Authorization: DPoP <token>` and `DPoP: <proof>` headers.
 *
 * @example
 * ```ts
 * const prover = await DPoPProver.generate();
 * const client = new SharkClient({ accessToken: "agent_abc", dpopProver: prover });
 *
 * const res = await client.fetch("https://api.example/data");
 * console.log(await res.json());
 * ```
 */
export class SharkClient {
  private readonly _accessToken: string;
  private readonly _dpopProver?: DPoPProver;
  private readonly _userAgent: string;

  constructor(opts: SharkClientOptions) {
    this._accessToken = opts.accessToken;
    this._dpopProver = opts.dpopProver;
    this._userAgent = opts.userAgent ?? "@sharkauth/node/0.1.0";
  }

  /**
   * Perform an HTTP request with automatic DPoP signing.
   *
   * Signature and headers are generated based on the method and URL
   * provided in the arguments.
   *
   * @param input   URL or Request object.
   * @param init    Standard fetch options.
   */
  async fetch(input: string | URL | Request, init?: RequestInit): Promise<Response> {
    const method = init?.method ?? "GET";
    const url = input instanceof Request ? input.url : String(input);

    const headers = new Headers(init?.headers);
    headers.set("User-Agent", this._userAgent);
    headers.set("Accept", "application/json");

    if (this._dpopProver) {
      const proof = await this._dpopProver.createProof({
        method,
        url,
        accessToken: this._accessToken,
      });
      headers.set("Authorization", `DPoP ${this._accessToken}`);
      headers.set("DPoP", proof);
    } else {
      headers.set("Authorization", `Bearer ${this._accessToken}`);
    }

    return fetch(input, {
      ...init,
      headers,
    });
  }
}
