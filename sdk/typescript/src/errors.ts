/**
 * Exception hierarchy for @sharkauth/node.
 *
 * All errors extend {@link SharkAuthError} so callers can catch by class.
 */

/** Base class for all SharkAuth SDK errors. */
export class SharkAuthError extends Error {
  constructor(message: string) {
    super(message);
    this.name = this.constructor.name;
    // Maintain proper prototype chain in transpiled ES5+
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Error constructing or signing a DPoP proof JWT. */
export class DPoPError extends SharkAuthError {}

/** Error during RFC 8628 device authorization flow. */
export class DeviceFlowError extends SharkAuthError {}

/** Error interacting with the Shark Token Vault. */
export class VaultError extends SharkAuthError {
  /** HTTP status code from the vault endpoint, if available. */
  readonly statusCode: number | undefined;

  constructor(message: string, statusCode?: number) {
    super(message);
    this.statusCode = statusCode;
  }
}

/** Error decoding or verifying a Shark-issued agent token. */
export class TokenError extends SharkAuthError {}

/**
 * Error returned by admin API calls when the server responds with a non-2xx
 * status. Carries the server's `error.code` string and the HTTP status code
 * so callers can branch on either.
 */
export class SharkAPIError extends SharkAuthError {
  /** Server-supplied error code, e.g. `"invalid_proxy_rule"`. */
  readonly code: string;
  /** HTTP status code (e.g. 400, 401, 404). */
  readonly status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}
