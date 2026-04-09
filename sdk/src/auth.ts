import type { HttpClient } from "./client.js";
import type {
  ChangePasswordParams,
  LoginMFAResult,
  LoginParams,
  LoginResult,
  PasswordResetParams,
  PasswordResetSendParams,
  SignupParams,
  User,
} from "./types.js";

export class AuthClient {
  constructor(private http: HttpClient) {}

  /** Create a new account and automatically sign in. */
  async signup(params: SignupParams): Promise<User> {
    return this.http.post<User>("/api/v1/auth/signup", params);
  }

  /**
   * Sign in with email and password.
   *
   * If MFA is enabled on the account, the result will have `mfaRequired: true`.
   * You must then call `mfa.challenge()` or `mfa.useRecoveryCode()` to complete login.
   *
   * @example
   * ```ts
   * const result = await auth.login({ email, password });
   * if (result.mfaRequired) {
   *   // show TOTP input
   *   const user = await auth.mfa.challenge({ code: totpCode });
   * } else {
   *   // fully authenticated
   *   console.log(result.user);
   * }
   * ```
   */
  async login(params: LoginParams): Promise<LoginResult | LoginMFAResult> {
    const data = await this.http.post<User & { mfaRequired?: boolean }>(
      "/api/v1/auth/login",
      params,
    );

    if (data.mfaRequired) {
      return { mfaRequired: true };
    }

    return { mfaRequired: false, user: data };
  }

  /** Sign out and destroy the current session. */
  async logout(): Promise<void> {
    await this.http.post("/api/v1/auth/logout");
  }

  /** Get the currently authenticated user. Throws if not authenticated or MFA incomplete. */
  async me(): Promise<User> {
    return this.http.get<User>("/api/v1/auth/me");
  }

  /** Send a password reset email. Always succeeds (doesn't leak email existence). */
  async sendPasswordReset(params: PasswordResetSendParams): Promise<void> {
    await this.http.post("/api/v1/auth/password/send-reset-link", params);
  }

  /** Reset password using a token from the reset email. */
  async resetPassword(params: PasswordResetParams): Promise<void> {
    await this.http.post("/api/v1/auth/password/reset", params);
  }

  /** Change password for the currently authenticated user. Requires MFA-completed session. */
  async changePassword(params: ChangePasswordParams): Promise<void> {
    await this.http.post("/api/v1/auth/password/change", params);
  }
}
