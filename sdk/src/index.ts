import { HttpClient } from "./client.js";
import { AuthClient } from "./auth.js";
import { MFAClient } from "./mfa.js";
import { PasskeyClient } from "./passkey.js";
import { OAuthClient } from "./oauth.js";
import { MagicLinkClient } from "./magic-link.js";
import type { SharkAuthConfig, User } from "./types.js";
import { SharkError } from "./client.js";

export class SharkAuth {
  private authClient: AuthClient;

  /** Multi-factor authentication (TOTP). */
  public readonly mfa: MFAClient;
  /** Passkey / WebAuthn authentication. */
  public readonly passkey: PasskeyClient;
  /** Social OAuth login (Google, GitHub, Apple, Discord). */
  public readonly oauth: OAuthClient;
  /** Passwordless magic link login. */
  public readonly magicLink: MagicLinkClient;

  constructor(config: SharkAuthConfig) {
    const http = new HttpClient(config);
    this.authClient = new AuthClient(http);
    this.mfa = new MFAClient(http);
    this.passkey = new PasskeyClient(http);
    this.oauth = new OAuthClient(http);
    this.magicLink = new MagicLinkClient(http);

    // Bind auth methods to this instance
    this.signup = this.authClient.signup.bind(this.authClient);
    this.login = this.authClient.login.bind(this.authClient);
    this.logout = this.authClient.logout.bind(this.authClient);
    this.me = this.authClient.me.bind(this.authClient);
    this.sendPasswordReset = this.authClient.sendPasswordReset.bind(this.authClient);
    this.resetPassword = this.authClient.resetPassword.bind(this.authClient);
    this.changePassword = this.authClient.changePassword.bind(this.authClient);
  }

  /** Create a new account and automatically sign in. */
  signup: AuthClient["signup"];

  /**
   * Sign in with email and password.
   *
   * Returns `{ mfaRequired: false, user }` on success, or
   * `{ mfaRequired: true }` if MFA verification is needed.
   *
   * When MFA is required, call `shark.mfa.challenge({ code })` to complete login.
   *
   * @example
   * ```ts
   * const result = await shark.login({ email, password });
   * if (result.mfaRequired) {
   *   const user = await shark.mfa.challenge({ code: totpCode });
   * } else {
   *   console.log("Logged in:", result.user);
   * }
   * ```
   */
  login: AuthClient["login"];

  /** Sign out and destroy the current session. */
  logout: AuthClient["logout"];

  /** Get the current user. Throws `SharkError` (403) if MFA is incomplete. */
  me: AuthClient["me"];

  /** Send a password reset email. */
  sendPasswordReset: AuthClient["sendPasswordReset"];

  /** Reset password using a token from the reset email. */
  resetPassword: AuthClient["resetPassword"];

  /** Change password (requires current password + MFA-completed session). */
  changePassword: AuthClient["changePassword"];

  /**
   * Check if the user is currently authenticated.
   * Returns the user if authenticated, or `null` if not (without throwing).
   */
  async check(): Promise<{ authenticated: true; user: User } | { authenticated: false; user: null }> {
    try {
      const user = await this.authClient.me();
      return { authenticated: true, user };
    } catch (err) {
      if (err instanceof SharkError && (err.status === 401 || err.status === 403)) {
        return { authenticated: false, user: null };
      }
      throw err;
    }
  }
}

/**
 * Create a SharkAuth client instance.
 *
 * @example
 * ```ts
 * import { createSharkAuth } from "@sharkauth/js";
 *
 * const shark = createSharkAuth({
 *   baseURL: "https://auth.myapp.com",
 * });
 *
 * // Sign up
 * const user = await shark.signup({ email: "user@example.com", password: "secret123" });
 *
 * // Login with MFA handling
 * const result = await shark.login({ email, password });
 * if (result.mfaRequired) {
 *   const user = await shark.mfa.challenge({ code: "123456" });
 * }
 *
 * // Passkey login
 * const user = await shark.passkey.login();
 *
 * // OAuth redirect
 * window.location.href = shark.oauth.getURL("google");
 *
 * // Magic link
 * await shark.magicLink.send({ email: "user@example.com" });
 *
 * // Check auth state (never throws on 401/403)
 * const { authenticated, user } = await shark.check();
 * ```
 */
export function createSharkAuth(config: SharkAuthConfig): SharkAuth {
  return new SharkAuth(config);
}

// Re-export types and error
export { SharkError } from "./client.js";
export type {
  SharkAuthConfig,
  User,
  SignupParams,
  LoginParams,
  LoginResult,
  LoginMFAResult,
  PasswordResetSendParams,
  PasswordResetParams,
  ChangePasswordParams,
  MFAEnrollResult,
  MFAVerifyResult,
  MFACodeParams,
  PasskeyCredential,
  PasskeyRegisterResult,
  OAuthProvider,
  MagicLinkSendParams,
} from "./types.js";
