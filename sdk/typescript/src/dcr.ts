/**
 * Dynamic Client Registration (RFC 7591) + Configuration Management (RFC 7592).
 *
 * Public client onboarding for the OAuth server. Each registered client gets
 * a `client_id`, optional `client_secret`, and a `registration_access_token`
 * which authenticates subsequent management requests to the
 * `registration_client_uri`.
 *
 * Backend routes (verified from `internal/api/router.go`):
 *   POST   /oauth/register                                   — register
 *   GET    /oauth/register/{client_id}                       — read
 *   PUT    /oauth/register/{client_id}                       — update
 *   DELETE /oauth/register/{client_id}                       — delete
 *   POST   /oauth/register/{client_id}/secret                — rotate secret
 *   DELETE /oauth/register/{client_id}/registration-token    — rotate registration token
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Input for {@link DcrClient.register}. */
export interface DcrRegisterInput {
  /** Human-readable client name (required). */
  client_name: string;
  /** Permitted redirect URIs. */
  redirect_uris?: string[];
  /** Allowed grant types, e.g. `["authorization_code", "refresh_token"]`. */
  grant_types?: string[];
  /** Allowed response types, e.g. `["code"]`. */
  response_types?: string[];
  /** Space-separated scope string. */
  scope?: string;
  /** `client_secret_basic`, `client_secret_post`, `none`, ... */
  token_endpoint_auth_method?: string;
  /** Additional RFC 7591 metadata forwarded verbatim. */
  [key: string]: unknown;
}

/** Result from `register` — server-issued credentials + echoed metadata. */
export interface DcrRegisterResult {
  client_id: string;
  client_secret?: string;
  client_id_issued_at?: number;
  client_secret_expires_at?: number;
  registration_access_token: string;
  registration_client_uri: string;
  client_name?: string;
  redirect_uris?: string[];
  grant_types?: string[];
  response_types?: string[];
  scope?: string;
  token_endpoint_auth_method?: string;
  [key: string]: unknown;
}

/** Result from `get` / `update`. */
export interface DcrClientMetadata {
  client_id: string;
  client_name?: string;
  redirect_uris?: string[];
  grant_types?: string[];
  response_types?: string[];
  scope?: string;
  token_endpoint_auth_method?: string;
  [key: string]: unknown;
}

/** Result from `rotateSecret` — includes the new `client_secret`. */
export interface DcrRotateSecretResult extends DcrClientMetadata {
  client_secret: string;
}

/** Result from `rotateRegistrationToken` — includes the new token. */
export interface DcrRotateTokenResult extends DcrClientMetadata {
  registration_access_token: string;
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

/** Options for {@link DcrClient}. */
export interface DcrClientOptions {
  /** No options today — kept for parity with sibling clients. */
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * Client for RFC 7591 Dynamic Client Registration and RFC 7592 management.
 *
 * @example
 * ```ts
 * const dcr = new DcrClient("https://auth.example.com");
 * const reg = await dcr.register({
 *   client_name: "My App",
 *   redirect_uris: ["https://app.example.com/cb"],
 *   grant_types: ["authorization_code", "refresh_token"],
 * });
 * const info = await dcr.get(reg.client_id, reg.registration_access_token);
 * ```
 */
export class DcrClient {
  private static readonly PREFIX = "/oauth/register";
  private readonly _base: string;

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  constructor(baseUrl: string, _opts: DcrClientOptions = {}) {
    this._base = baseUrl.replace(/\/+$/, "");
  }

  private _bearer(token: string): Record<string, string> {
    return { Authorization: `Bearer ${token}` };
  }

  private async _throw(status: number, text: string): Promise<never> {
    let code = "api_error";
    let message = text.slice(0, 300);
    try {
      const body = JSON.parse(text) as {
        error?: string | { code?: string; message?: string };
        error_description?: string;
        message?: string;
      };
      if (typeof body.error === "string") code = body.error;
      else if (body.error && typeof body.error === "object" && body.error.code)
        code = body.error.code;
      if (body.error_description) message = body.error_description;
      else if (body.message) message = body.message;
      else if (body.error && typeof body.error === "object" && body.error.message)
        message = body.error.message;
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  // ------------------------------------------------------------------
  // RFC 7591 — register
  // ------------------------------------------------------------------

  /**
   * Register a new OAuth client (RFC 7591).
   *
   * Returns at minimum `client_id`, `client_secret` (if confidential),
   * `registration_access_token`, and `registration_client_uri`.
   */
  async register(input: DcrRegisterInput): Promise<DcrRegisterResult> {
    const url = `${this._base}${DcrClient.PREFIX}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return resp.json<DcrRegisterResult>();
  }

  // ------------------------------------------------------------------
  // RFC 7592 — read / update / delete
  // ------------------------------------------------------------------

  /** Fetch the current client metadata (RFC 7592). */
  async get(
    clientId: string,
    registrationAccessToken: string,
  ): Promise<DcrClientMetadata> {
    const url = `${this._base}${DcrClient.PREFIX}/${encodeURIComponent(clientId)}`;
    const resp = await httpRequest(url, {
      method: "GET",
      headers: this._bearer(registrationAccessToken),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<DcrClientMetadata>();
  }

  /**
   * Replace client metadata via PUT (RFC 7592). RFC 7592 specifies a
   * full-replacement semantic — pass every metadata field you want kept.
   */
  async update(
    clientId: string,
    registrationAccessToken: string,
    fields: Record<string, unknown>,
  ): Promise<DcrClientMetadata> {
    const url = `${this._base}${DcrClient.PREFIX}/${encodeURIComponent(clientId)}`;
    const resp = await httpRequest(url, {
      method: "PUT",
      headers: { ...this._bearer(registrationAccessToken), "Content-Type": "application/json" },
      body: JSON.stringify(fields),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.json<DcrClientMetadata>();
  }

  /** Delete the registered client (RFC 7592). */
  async delete(clientId: string, registrationAccessToken: string): Promise<void> {
    const url = `${this._base}${DcrClient.PREFIX}/${encodeURIComponent(clientId)}`;
    const resp = await httpRequest(url, {
      method: "DELETE",
      headers: this._bearer(registrationAccessToken),
    });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // ------------------------------------------------------------------
  // SharkAuth extensions — secret + registration-token rotation
  // ------------------------------------------------------------------

  /**
   * Rotate the `client_secret`.
   *
   * Backend path: `POST /oauth/register/{client_id}/secret` (NOT `/rotate-secret`).
   * Returns the new client metadata including the rotated `client_secret`.
   */
  async rotateSecret(
    clientId: string,
    registrationAccessToken: string,
  ): Promise<DcrRotateSecretResult> {
    const url = `${this._base}${DcrClient.PREFIX}/${encodeURIComponent(clientId)}/secret`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: this._bearer(registrationAccessToken),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return resp.json<DcrRotateSecretResult>();
  }

  /**
   * Rotate the `registration_access_token`.
   *
   * Backend path: `DELETE /oauth/register/{client_id}/registration-token`
   * (verb is DELETE, NOT POST `/rotate-token`). The DELETE invalidates the
   * current token and the response carries the new `registration_access_token`.
   */
  async rotateRegistrationToken(
    clientId: string,
    registrationAccessToken: string,
  ): Promise<DcrRotateTokenResult> {
    const url = `${this._base}${DcrClient.PREFIX}/${encodeURIComponent(clientId)}/registration-token`;
    const resp = await httpRequest(url, {
      method: "DELETE",
      headers: this._bearer(registrationAccessToken),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return resp.json<DcrRotateTokenResult>();
  }
}
