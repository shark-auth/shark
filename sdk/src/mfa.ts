import type { HttpClient } from "./client.js";
import type {
  MFACodeParams,
  MFAEnrollResult,
  MFAVerifyResult,
  User,
} from "./types.js";

export class MFAClient {
  constructor(private http: HttpClient) {}

  /**
   * Start MFA enrollment. Returns the TOTP secret and a QR code URI.
   * Requires a fully authenticated session (MFA already passed or not enabled).
   */
  async enroll(): Promise<MFAEnrollResult> {
    return this.http.post<MFAEnrollResult>("/api/v1/auth/mfa/enroll");
  }

  /**
   * Confirm MFA setup with the first TOTP code from the authenticator app.
   * Returns recovery codes — the user should save these.
   */
  async verify(params: MFACodeParams): Promise<MFAVerifyResult> {
    return this.http.post<MFAVerifyResult>("/api/v1/auth/mfa/verify", params);
  }

  /**
   * Submit a TOTP code to complete login when MFA is required.
   * Call this after `login()` returns `{ mfaRequired: true }`.
   * On success, the session is upgraded and the user is returned.
   */
  async challenge(params: MFACodeParams): Promise<User> {
    return this.http.post<User>("/api/v1/auth/mfa/challenge", params);
  }

  /**
   * Use a recovery code to complete login when MFA is required.
   * Call this after `login()` returns `{ mfaRequired: true }` if the user lost their device.
   * Recovery codes are single-use.
   */
  async useRecoveryCode(params: MFACodeParams): Promise<User> {
    return this.http.post<User>("/api/v1/auth/mfa/recovery", params);
  }

  /**
   * Disable MFA. Requires a current TOTP code for confirmation.
   * Requires a fully authenticated session.
   */
  async disable(params: MFACodeParams): Promise<void> {
    await this.http.del("/api/v1/auth/mfa", params);
  }

  /**
   * Regenerate recovery codes. Invalidates all previous codes.
   * Requires a fully authenticated session with MFA enabled.
   */
  async regenerateRecoveryCodes(): Promise<{ recovery_codes: string[] }> {
    return this.http.get<{ recovery_codes: string[] }>("/api/v1/auth/mfa/recovery-codes");
  }
}
