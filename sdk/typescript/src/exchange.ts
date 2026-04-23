/**
 * RFC 8693 Token Exchange implementation for agent-to-agent delegation.
 */

import { httpRequest } from "./http.js";
import { TokenError } from "./errors.js";
import { TokenResponse } from "./deviceFlow.js";
import { DPoPProver } from "./dpop.js";

const TOKEN_EXCHANGE_GRANT = "urn:ietf:params:oauth:grant-type:token-exchange";
const ACCESS_TOKEN_TYPE = "urn:ietf:params:oauth:token-type:access_token";
const REFRESH_TOKEN_TYPE = "urn:ietf:params:oauth:token-type:refresh_token";
const JWT_TOKEN_TYPE = "urn:ietf:params:oauth:token-type:jwt";

/** Options for {@link exchangeToken}. */
export interface TokenExchangeOptions {
  /** Base URL of the SharkAuth server. */
  authUrl: string;
  /** OAuth 2.0 client ID of the acting agent. */
  clientId: string;
  /** Client secret (for confidential clients). */
  clientSecret?: string;
  /**
   * The token representing the identity being acted upon.
   * Typically a JWT access token issued by SharkAuth.
   */
  subjectToken: string;
  /**
   * The type of the subject token.
   * Default: `urn:ietf:params:oauth:token-type:access_token`.
   */
  subjectTokenType?: string;
  /**
   * Optional token representing the actor's own identity.
   * If omitted, the client's own credentials (secret) are used.
   */
  actorToken?: string;
  /**
   * The type of the actor token.
   * Default: `urn:ietf:params:oauth:token-type:access_token`.
   */
  actorTokenType?: string;
  /** Optional space-delimited scope string for the new token. */
  scope?: string;
  /**
   * The type of token requested.
   * Default: `urn:ietf:params:oauth:token-type:access_token`.
   */
  requestedTokenType?: string;
  /**
   * Optional {@link DPoPProver} — when supplied, the exchange will
   * result in a DPoP-bound token, and a `DPoP` header is attached
   * to the request.
   */
  dpopProver?: DPoPProver;
  /** Override the token endpoint path. Default: `/oauth/token`. */
  tokenPath?: string;
}

/**
 * Perform an RFC 8693 Token Exchange.
 *
 * This enables "Act On Behalf Of" delegation chains. An agent can exchange
 * a user's token (the subject) for a new token that identifies the agent
 * as the actor acting on that user's behalf.
 *
 * @example
 * ```ts
 * const tokens = await exchangeToken({
 *   authUrl: "https://auth.example",
 *   clientId: "agent_worker",
 *   subjectToken: userAccessToken,
 *   scope: "files:read",
 * });
 * console.log(tokens.accessToken);
 * ```
 */
export async function exchangeToken(
  opts: TokenExchangeOptions
): Promise<TokenResponse> {
  const tokenUrl = opts.authUrl.replace(/\/+$/, "") + (opts.tokenPath ?? "/oauth/token");

  const form: Record<string, string> = {
    grant_type: TOKEN_EXCHANGE_GRANT,
    client_id: opts.clientId,
    subject_token: opts.subjectToken,
    subject_token_type: opts.subjectTokenType ?? ACCESS_TOKEN_TYPE,
  };

  if (opts.clientSecret) {
    form["client_secret"] = opts.clientSecret;
  }
  if (opts.actorToken) {
    form["actor_token"] = opts.actorToken;
    form["actor_token_type"] = opts.actorTokenType ?? ACCESS_TOKEN_TYPE;
  }
  if (opts.scope) {
    form["scope"] = opts.scope;
  }
  if (opts.requestedTokenType) {
    form["requested_token_type"] = opts.requestedTokenType;
  }

  const headers: Record<string, string> = {};
  if (opts.dpopProver) {
    headers["DPoP"] = await opts.dpopProver.createProof({
      method: "POST",
      url: tokenUrl,
    });
  }

  const resp = await httpRequest(tokenUrl, {
    method: "POST",
    form,
    headers,
  });

  let payload: Record<string, unknown>;
  try {
    payload = resp.json<Record<string, unknown>>();
  } catch {
    throw new TokenError(
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

  const err = typeof payload["error"] === "string" ? payload["error"] : "unknown";
  const desc =
    typeof payload["error_description"] === "string"
      ? payload["error_description"]
      : "";

  throw new TokenError(
    `token exchange failed: ${err} (HTTP ${resp.status}): ${desc}`
  );
}
