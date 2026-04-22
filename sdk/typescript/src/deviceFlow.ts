/**
 * RFC 8628 OAuth 2.0 Device Authorization Grant client.
 *
 * @example
 * ```ts
 * const flow = new DeviceFlow({ authUrl: "https://auth.example", clientId: "agent_xyz" });
 * const init = await flow.begin();
 * console.log(init.verificationUri, init.userCode);
 * const tokens = await flow.waitForApproval({ timeoutMs: 300_000 });
 * ```
 */

import { httpRequest } from "./http.js";
import { DPoPProver } from "./dpop.js";
import { DeviceFlowError } from "./errors.js";

const DEVICE_GRANT = "urn:ietf:params:oauth:grant-type:device_code";

/** Response to the device authorization request. */
export interface DeviceInit {
  deviceCode: string;
  userCode: string;
  verificationUri: string;
  verificationUriComplete?: string;
  expiresIn: number;
  interval: number;
}

/** Access token response returned by the auth server. */
export interface TokenResponse {
  accessToken: string;
  tokenType: string;
  expiresIn?: number;
  refreshToken?: string;
  scope?: string;
}

/** Options for constructing a {@link DeviceFlow}. */
export interface DeviceFlowOptions {
  /** Base URL of the SharkAuth server. */
  authUrl: string;
  /** OAuth 2.0 client ID. */
  clientId: string;
  /** Optional scope string (space-delimited). */
  scope?: string;
  /**
   * Optional {@link DPoPProver} — when supplied, a `DPoP` header is attached
   * to every token-endpoint request.
   */
  dpopProver?: DPoPProver;
  /** Override the device authorization path. Default: `/oauth/device_authorization`. */
  deviceAuthorizationPath?: string;
  /** Override the token path. Default: `/oauth/token`. */
  tokenPath?: string;
}

/** Options for {@link DeviceFlow.waitForApproval}. */
export interface WaitForApprovalOptions {
  /** Maximum time to wait in milliseconds. Default: 300 000 ms (5 minutes). */
  timeoutMs?: number;
  /**
   * Monotonic clock function returning milliseconds. Default: `Date.now`.
   * Injected for deterministic testing.
   */
  clock?: () => number;
  /**
   * Sleep function (receives milliseconds). Default: real setTimeout.
   * Injected for deterministic testing.
   */
  sleep?: (ms: number) => Promise<void>;
}

/**
 * Runs the RFC 8628 device authorization grant.
 *
 * Call {@link DeviceFlow.begin} first to obtain the `userCode` /
 * `verificationUri`, then call {@link DeviceFlow.waitForApproval} to poll
 * until the user approves or the flow fails.
 */
export class DeviceFlow {
  private readonly _authUrl: string;
  private readonly _clientId: string;
  private readonly _scope?: string;
  private readonly _dpopProver?: DPoPProver;
  private readonly _deviceAuthUrl: string;
  private readonly _tokenUrl: string;
  private _init: DeviceInit | null = null;

  constructor(opts: DeviceFlowOptions) {
    this._authUrl = opts.authUrl.replace(/\/+$/, "");
    this._clientId = opts.clientId;
    this._scope = opts.scope;
    this._dpopProver = opts.dpopProver;
    this._deviceAuthUrl =
      this._authUrl + (opts.deviceAuthorizationPath ?? "/oauth/device_authorization");
    this._tokenUrl = this._authUrl + (opts.tokenPath ?? "/oauth/token");
  }

  /**
   * Initiate the device authorization request.
   *
   * @returns {@link DeviceInit} containing the codes the user needs to visit.
   * @throws {@link DeviceFlowError} on network or server errors.
   */
  async begin(): Promise<DeviceInit> {
    const form: Record<string, string> = { client_id: this._clientId };
    if (this._scope) form["scope"] = this._scope;

    const resp = await httpRequest(this._deviceAuthUrl, {
      method: "POST",
      form,
    });

    if (resp.status !== 200) {
      throw new DeviceFlowError(
        `device_authorization failed: HTTP ${resp.status}: ${resp.text.slice(0, 200)}`
      );
    }

    let body: Record<string, unknown>;
    try {
      body = resp.json<Record<string, unknown>>();
    } catch {
      throw new DeviceFlowError(
        `device_authorization returned non-JSON: ${resp.text.slice(0, 200)}`
      );
    }

    const required = ["device_code", "user_code", "verification_uri", "expires_in"];
    const missing = required.filter((k) => !(k in body));
    if (missing.length > 0) {
      throw new DeviceFlowError(`device_authorization missing keys: ${missing.join(", ")}`);
    }

    const init: DeviceInit = {
      deviceCode: String(body["device_code"]),
      userCode: String(body["user_code"]),
      verificationUri: String(body["verification_uri"]),
      verificationUriComplete:
        typeof body["verification_uri_complete"] === "string"
          ? body["verification_uri_complete"]
          : undefined,
      expiresIn: Number(body["expires_in"]),
      interval: Number(body["interval"] ?? 5),
    };

    this._init = init;
    return init;
  }

  /**
   * Poll `/oauth/token` until the user approves or the flow errors.
   *
   * Handles `authorization_pending` (continue), `slow_down` (increase interval
   * by 5 s per RFC 8628 §3.5), `access_denied` and `expired_token` (throw).
   *
   * @throws {@link DeviceFlowError} on denial, expiry, or timeout.
   */
  async waitForApproval(opts: WaitForApprovalOptions = {}): Promise<TokenResponse> {
    if (this._init === null) {
      throw new DeviceFlowError("begin() must be called before waitForApproval()");
    }

    const {
      timeoutMs = 300_000,
      clock = () => Date.now(),
      sleep = (ms: number) => new Promise<void>((res) => setTimeout(res, ms)),
    } = opts;

    const deadline = clock() + timeoutMs;
    let intervalMs = Math.max(1000, this._init.interval * 1000);

    while (true) {
      if (clock() >= deadline) {
        throw new DeviceFlowError("device flow timed out before approval");
      }

      const form: Record<string, string> = {
        grant_type: DEVICE_GRANT,
        device_code: this._init.deviceCode,
        client_id: this._clientId,
      };

      const headers: Record<string, string> = {};
      if (this._dpopProver) {
        headers["DPoP"] = await this._dpopProver.createProof({
          method: "POST",
          url: this._tokenUrl,
        });
      }

      const resp = await httpRequest(this._tokenUrl, {
        method: "POST",
        form,
        headers,
      });

      let payload: Record<string, unknown>;
      try {
        payload = resp.json<Record<string, unknown>>();
      } catch {
        throw new DeviceFlowError(
          `token endpoint returned non-JSON: HTTP ${resp.status}: ${resp.text.slice(0, 200)}`
        );
      }

      if (resp.status === 200) {
        return {
          accessToken: String(payload["access_token"]),
          tokenType: typeof payload["token_type"] === "string"
            ? payload["token_type"]
            : "Bearer",
          expiresIn:
            typeof payload["expires_in"] === "number"
              ? payload["expires_in"]
              : undefined,
          refreshToken:
            typeof payload["refresh_token"] === "string"
              ? payload["refresh_token"]
              : undefined,
          scope:
            typeof payload["scope"] === "string" ? payload["scope"] : undefined,
        };
      }

      const err = typeof payload["error"] === "string" ? payload["error"] : null;
      if (err === "authorization_pending") {
        // continue
      } else if (err === "slow_down") {
        intervalMs += 5000;
      } else if (err === "access_denied") {
        throw new DeviceFlowError("user denied the authorization request");
      } else if (err === "expired_token") {
        throw new DeviceFlowError("device code expired before user approved");
      } else if (err === "invalid_client") {
        throw new DeviceFlowError("invalid client_id");
      } else {
        const desc =
          typeof payload["error_description"] === "string"
            ? payload["error_description"]
            : "";
        throw new DeviceFlowError(
          `token endpoint error: ${err ?? "unknown"} (HTTP ${resp.status}): ${desc}`
        );
      }

      const remaining = deadline - clock();
      if (remaining <= 0) {
        throw new DeviceFlowError("device flow timed out before approval");
      }
      await sleep(Math.min(intervalMs, remaining));
    }
  }
}
