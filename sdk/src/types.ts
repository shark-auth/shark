// ── Configuration ──

export interface SharkAuthConfig {
  /** Base URL of the SharkAuth server (e.g. "https://auth.myapp.com" or "http://localhost:8080") */
  baseURL: string;
  /** Custom fetch implementation. Defaults to globalThis.fetch. */
  fetch?: typeof fetch;
}

// ── User ──

export interface User {
  id: string;
  email: string;
  emailVerified: boolean;
  name?: string;
  avatarUrl?: string;
  mfaEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

// ── Auth ──

export interface SignupParams {
  email: string;
  password: string;
  name?: string;
}

export interface LoginParams {
  email: string;
  password: string;
}

export interface LoginResult {
  mfaRequired: false;
  user: User;
}

export interface LoginMFAResult {
  mfaRequired: true;
}

export interface PasswordResetSendParams {
  email: string;
}

export interface PasswordResetParams {
  token: string;
  password: string;
}

export interface ChangePasswordParams {
  current_password: string;
  new_password: string;
}

// ── MFA ──

export interface MFAEnrollResult {
  secret: string;
  qr_uri: string;
}

export interface MFAVerifyResult {
  mfa_enabled: true;
  recovery_codes: string[];
}

export interface MFACodeParams {
  code: string;
}

// ── Passkeys ──

export interface PasskeyCredential {
  id: string;
  name?: string;
  transports: string;
  backed_up: boolean;
  created_at: string;
  last_used_at?: string;
}

export interface PasskeyRegisterResult {
  credential_id: string;
  name: string;
}

// ── OAuth ──

export type OAuthProvider = "google" | "github" | "apple" | "discord";

// ── Magic Link ──

export interface MagicLinkSendParams {
  email: string;
}

// ── Errors ──

export interface SharkErrorBody {
  error: string;
  message: string;
}
