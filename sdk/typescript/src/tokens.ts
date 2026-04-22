/**
 * Decode Shark-issued agent access tokens without signature verification.
 *
 * For use by agent runtimes that trust the token was already validated
 * elsewhere (e.g. edge middleware). Use the Python SDK's `decode_agent_token`
 * if you need full JWKS-backed verification.
 *
 * @example
 * ```ts
 * const claims = decodeAgentToken(jwtString);
 * console.log(claims.sub, claims.scope);
 * ```
 */

import { TokenError } from "./errors.js";

/** RFC 8693 actor claim. */
export interface ActorClaim {
  sub: string;
  [key: string]: unknown;
}

/** RFC 9449 confirmation claim. */
export interface ConfirmationClaim {
  /** DPoP JWK thumbprint bound to the token. */
  jkt?: string;
  [key: string]: unknown;
}

/** RFC 9396 authorization details entry. */
export interface AuthorizationDetail {
  type: string;
  [key: string]: unknown;
}

/**
 * Parsed claims from a Shark agent access token.
 *
 * Preserves all registered claims including RFC 8693 `act` (actor),
 * RFC 9449 `cnf` (confirmation), and RFC 9396 `authorization_details`.
 */
export interface AgentTokenClaims {
  /** Subject — typically the agent or user identifier. */
  sub: string;
  /** Audience — the intended resource server. */
  aud: string | string[];
  /** Issuer — the SharkAuth server URL. */
  iss: string;
  /** Expiration time (Unix timestamp, seconds). */
  exp: number;
  /** Issued-at time (Unix timestamp, seconds). */
  iat: number;
  /** Space-delimited scope string. */
  scope?: string;
  /** RFC 8693 actor claim. */
  act?: ActorClaim;
  /** RFC 9449 confirmation claim (contains `jkt` for DPoP binding). */
  cnf?: ConfirmationClaim;
  /** RFC 9396 authorization details. */
  authorization_details?: AuthorizationDetail[];
  /** All raw claims from the JWT payload. */
  raw: Record<string, unknown>;
}

/**
 * Split and base64url-decode a JWT without verifying its signature.
 *
 * Returns typed {@link AgentTokenClaims}. This function does **not**
 * verify the signature, expiry, issuer, or audience — use it only
 * when the token has already been verified by trusted infrastructure.
 *
 * @param token  Compact JWT string (`header.payload.signature`).
 * @throws {@link TokenError} if the token is malformed or the payload is not
 *   valid JSON.
 */
export function decodeAgentToken(token: string): AgentTokenClaims {
  if (typeof token !== "string" || token.trim() === "") {
    throw new TokenError("token must be a non-empty string");
  }

  const parts = token.split(".");
  if (parts.length !== 3) {
    throw new TokenError(
      `malformed JWT: expected 3 parts separated by '.', got ${parts.length}`
    );
  }

  const payloadB64 = parts[1];
  let payloadJson: string;
  try {
    payloadJson = _b64urlDecode(payloadB64);
  } catch {
    throw new TokenError("malformed JWT: payload is not valid base64url");
  }

  let raw: Record<string, unknown>;
  try {
    raw = JSON.parse(payloadJson) as Record<string, unknown>;
  } catch {
    throw new TokenError("malformed JWT: payload is not valid JSON");
  }

  // Validate required standard claims
  const missing = (["sub", "aud", "iss", "exp", "iat"] as const).filter(
    (k) => !(k in raw)
  );
  if (missing.length > 0) {
    throw new TokenError(`token missing required claims: ${missing.join(", ")}`);
  }

  return {
    sub: String(raw["sub"]),
    aud: Array.isArray(raw["aud"])
      ? (raw["aud"] as unknown[]).map(String)
      : String(raw["aud"]),
    iss: String(raw["iss"]),
    exp: Number(raw["exp"]),
    iat: Number(raw["iat"]),
    scope: typeof raw["scope"] === "string" ? raw["scope"] : undefined,
    act: isRecord(raw["act"]) ? (raw["act"] as ActorClaim) : undefined,
    cnf: isRecord(raw["cnf"]) ? (raw["cnf"] as ConfirmationClaim) : undefined,
    authorization_details: Array.isArray(raw["authorization_details"])
      ? (raw["authorization_details"] as AuthorizationDetail[])
      : undefined,
    raw,
  };
}

// ------------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------------

/** Decode a base64url string to UTF-8 text. */
function _b64urlDecode(input: string): string {
  // Re-add padding
  const padded = input + "=".repeat((4 - (input.length % 4)) % 4);
  const std = padded.replace(/-/g, "+").replace(/_/g, "/");
  return Buffer.from(std, "base64").toString("utf-8");
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}
