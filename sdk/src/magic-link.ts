import type { HttpClient } from "./client.js";
import type { MagicLinkSendParams } from "./types.js";

export class MagicLinkClient {
  constructor(private http: HttpClient) {}

  /**
   * Send a magic link to the given email address.
   * The user clicks the link in their email to authenticate.
   *
   * Always succeeds (doesn't reveal whether the email exists).
   * Rate-limited to 1 request per 60 seconds per email.
   */
  async send(params: MagicLinkSendParams): Promise<void> {
    await this.http.post("/api/v1/auth/magic-link/send", params);
  }
}
