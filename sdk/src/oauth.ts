import type { HttpClient } from "./client.js";
import type { OAuthProvider } from "./types.js";

export class OAuthClient {
  constructor(private http: HttpClient) {}

  /**
   * Get the OAuth redirect URL for a provider.
   * Navigate the user to this URL to start the OAuth flow.
   *
   * @example
   * ```ts
   * window.location.href = auth.oauth.getURL("google");
   * ```
   */
  getURL(provider: OAuthProvider): string {
    return this.http.url(`/api/v1/auth/oauth/${provider}`);
  }
}
