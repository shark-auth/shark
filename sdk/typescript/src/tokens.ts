/**
 * Verify and decode Shark-issued agent access tokens via JWKS.
 */

import * as jose from "jose";
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
  raw: jose.JWTPayload;
}

/** Options for {@link verifyAgentToken}. */
export interface VerifyOptions {
  /** Base URL of the SharkAuth server (used to construct JWKS URL). */
  authUrl: string;
  /** Expected issuer (`iss` claim). Typically matches `authUrl`. */
  expectedIssuer: string;
  /** Expected audience (`aud` claim). */
  expectedAudience: string | string[];
  /** Clock leeway in seconds. Default: 0. */
  leeway?: number;
}

/**
 * Verify a Shark agent JWT signature against the server's JWKS and check standard claims.
 *
 * This is the secure-by-default way to process tokens in Node.js.
 * It caches the JWKS automatically via `jose.createRemoteJWKSet`.
 */
export async function verifyAgentToken(
  token: string,
  opts: VerifyOptions
): Promise<AgentTokenClaims> {
  const jwksUrl = new URL("/.well-known/jwks.json", opts.authUrl).toString();
  const JWKS = jose.createRemoteJWKSet(new URL(jwksUrl));

  try {
    const { payload } = await jose.jwtVerify(token, JWKS, {
      issuer: opts.expectedIssuer,
      audience: opts.expectedAudience,
      clockTolerance: opts.leeway ?? 0,
    });

    return {
      sub: String(payload.sub),
      aud: Array.isArray(payload.aud) ? payload.aud : String(payload.aud),
      iss: String(payload.iss),
      exp: Number(payload.exp),
      iat: Number(payload.iat),
      scope: typeof payload.scope === "string" ? payload.scope : undefined,
      act: isRecord(payload.act) ? (payload.act as unknown as ActorClaim) : undefined,
      cnf: isRecord(payload.cnf) ? (payload.cnf as unknown as ConfirmationClaim) : undefined,
      authorization_details: Array.isArray(payload.authorization_details)
        ? (payload.authorization_details as AuthorizationDetail[])
        : undefined,
      raw: payload,
    };
  } catch (err) {
    if (err instanceof jose.errors.JWTExpired) {
      throw new TokenError("token expired");
    }
    if (err instanceof jose.errors.JWSSignatureVerificationFailed) {
      throw new TokenError("invalid signature");
    }
    if (err instanceof jose.errors.JWTClaimValidationFailed) {
      throw new TokenError(`claim validation failed: ${err.claim} ${err.code}`);
    }
    throw new TokenError(`token verification failed: ${String(err)}`);
  }
}

/**
 * Split and base64url-decode a JWT WITHOUT verifying its signature.
 *
 * @deprecated Use {@link verifyAgentToken} for production workloads.
 * This remains for use by agent runtimes that trust the token was already
 * validated elsewhere (e.g. edge middleware).
 */
export function decodeAgentToken(token: string): AgentTokenClaims {
  try {
    const payload = jose.decodeJwt(token);
    return {
      sub: String(payload.sub),
      aud: Array.isArray(payload.aud) ? payload.aud : String(payload.aud),
      iss: String(payload.iss),
      exp: Number(payload.exp),
      iat: Number(payload.iat),
      scope: typeof payload.scope === "string" ? payload.scope : undefined,
      act: isRecord(payload.act) ? (payload.act as unknown as ActorClaim) : undefined,
      cnf: isRecord(payload.cnf) ? (payload.cnf as unknown as ConfirmationClaim) : undefined,
      authorization_details: Array.isArray(payload.authorization_details)
        ? (payload.authorization_details as AuthorizationDetail[])
        : undefined,
      raw: payload,
    };
  } catch (err) {
    throw new TokenError(`malformed JWT: ${String(err)}`);
  }
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}
