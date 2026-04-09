import type { HttpClient } from "./client.js";
import type { PasskeyCredential, PasskeyRegisterResult, User } from "./types.js";

/**
 * Base64url encoding/decoding for WebAuthn ArrayBuffer <-> string conversion.
 * These work in both browser and Node.js 18+ (via globalThis.btoa/atob).
 */
function toBase64URL(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function fromBase64URL(str: string): ArrayBuffer {
  const base64 = str.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

export class PasskeyClient {
  constructor(private http: HttpClient) {}

  /**
   * Register a new passkey for the current user.
   * This triggers the browser's WebAuthn dialog (Touch ID, Windows Hello, security key, etc).
   * Requires an authenticated session.
   */
  async register(): Promise<PasskeyRegisterResult> {
    // Step 1: Get registration options from server
    const { publicKey, challengeKey } = await this.http.post<{
      publicKey: PublicKeyCredentialCreationOptions;
      challengeKey: string;
    }>("/api/v1/auth/passkey/register/begin");

    // Step 2: Decode base64url fields to ArrayBuffers for the browser API
    const options: PublicKeyCredentialCreationOptions = {
      ...publicKey,
      challenge: fromBase64URL(publicKey.challenge as unknown as string),
      user: {
        ...publicKey.user,
        id: fromBase64URL(publicKey.user.id as unknown as string),
      },
    };

    if (publicKey.excludeCredentials) {
      options.excludeCredentials = publicKey.excludeCredentials.map((cred) => ({
        ...cred,
        id: fromBase64URL(cred.id as unknown as string),
      }));
    }

    // Step 3: Browser WebAuthn dialog
    const credential = (await navigator.credentials.create({
      publicKey: options,
    })) as PublicKeyCredential | null;

    if (!credential) {
      throw new Error("Passkey registration was cancelled");
    }

    const response = credential.response as AuthenticatorAttestationResponse;

    // Step 4: Send attestation to server
    const body = {
      id: credential.id,
      rawId: toBase64URL(credential.rawId),
      type: credential.type,
      response: {
        attestationObject: toBase64URL(response.attestationObject),
        clientDataJSON: toBase64URL(response.clientDataJSON),
      },
    };

    return this.http.post<PasskeyRegisterResult>(
      "/api/v1/auth/passkey/register/finish",
      body,
      { "X-Challenge-Key": challengeKey },
    );
  }

  /**
   * Sign in with a passkey.
   * This triggers the browser's WebAuthn dialog for authentication.
   * Passkey login bypasses MFA (passkeys satisfy AAL2).
   *
   * @param email - Optional. If provided, limits to that user's passkeys.
   *                If omitted, uses discoverable credentials (resident keys).
   */
  async login(email?: string): Promise<User> {
    // Step 1: Get assertion options from server
    const beginBody = email ? { email } : undefined;
    const { publicKey, challengeKey } = await this.http.post<{
      publicKey: PublicKeyCredentialRequestOptions;
      challengeKey: string;
    }>("/api/v1/auth/passkey/login/begin", beginBody);

    // Step 2: Decode base64url fields
    const options: PublicKeyCredentialRequestOptions = {
      ...publicKey,
      challenge: fromBase64URL(publicKey.challenge as unknown as string),
    };

    if (publicKey.allowCredentials) {
      options.allowCredentials = publicKey.allowCredentials.map((cred) => ({
        ...cred,
        id: fromBase64URL(cred.id as unknown as string),
      }));
    }

    // Step 3: Browser WebAuthn dialog
    const credential = (await navigator.credentials.get({
      publicKey: options,
    })) as PublicKeyCredential | null;

    if (!credential) {
      throw new Error("Passkey authentication was cancelled");
    }

    const response = credential.response as AuthenticatorAssertionResponse;

    // Step 4: Send assertion to server
    const body = {
      id: credential.id,
      rawId: toBase64URL(credential.rawId),
      type: credential.type,
      response: {
        authenticatorData: toBase64URL(response.authenticatorData),
        clientDataJSON: toBase64URL(response.clientDataJSON),
        signature: toBase64URL(response.signature),
        userHandle: response.userHandle ? toBase64URL(response.userHandle) : undefined,
      },
    };

    return this.http.post<User>(
      "/api/v1/auth/passkey/login/finish",
      body,
      { "X-Challenge-Key": challengeKey },
    );
  }

  /** List the current user's registered passkeys. */
  async list(): Promise<PasskeyCredential[]> {
    const data = await this.http.get<{ credentials: PasskeyCredential[] }>(
      "/api/v1/auth/passkey/credentials",
    );
    return data.credentials;
  }

  /** Delete a passkey by ID. */
  async remove(credentialId: string): Promise<void> {
    await this.http.del(`/api/v1/auth/passkey/credentials/${credentialId}`);
  }

  /** Rename a passkey. */
  async rename(credentialId: string, name: string): Promise<PasskeyCredential> {
    return this.http.patch<PasskeyCredential>(
      `/api/v1/auth/passkey/credentials/${credentialId}`,
      { name },
    );
  }
}
