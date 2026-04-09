import type { SharkAuthConfig, SharkErrorBody } from "./types.js";

export class SharkError extends Error {
  public readonly status: number;
  public readonly code: string;

  constructor(status: number, body: SharkErrorBody) {
    super(body.message);
    this.name = "SharkError";
    this.status = status;
    this.code = body.error;
  }
}

export class HttpClient {
  private baseURL: string;
  private fetchFn: typeof fetch;

  constructor(config: SharkAuthConfig) {
    // Strip trailing slash
    this.baseURL = config.baseURL.replace(/\/+$/, "");
    this.fetchFn = config.fetch ?? globalThis.fetch.bind(globalThis);
  }

  async request<T>(
    method: string,
    path: string,
    body?: unknown,
    headers?: Record<string, string>,
  ): Promise<T> {
    const url = `${this.baseURL}${path}`;

    const init: RequestInit = {
      method,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...headers,
      },
    };

    if (body !== undefined) {
      init.body = JSON.stringify(body);
    }

    const res = await this.fetchFn(url, init);

    if (!res.ok) {
      let errorBody: SharkErrorBody;
      try {
        errorBody = await res.json();
      } catch {
        errorBody = { error: "unknown", message: res.statusText };
      }
      throw new SharkError(res.status, errorBody);
    }

    // Handle empty responses (logout, etc.)
    const text = await res.text();
    if (!text) return {} as T;
    return JSON.parse(text) as T;
  }

  get<T>(path: string, headers?: Record<string, string>): Promise<T> {
    return this.request<T>("GET", path, undefined, headers);
  }

  post<T>(path: string, body?: unknown, headers?: Record<string, string>): Promise<T> {
    return this.request<T>("POST", path, body, headers);
  }

  patch<T>(path: string, body?: unknown, headers?: Record<string, string>): Promise<T> {
    return this.request<T>("PATCH", path, body, headers);
  }

  del<T>(path: string, body?: unknown, headers?: Record<string, string>): Promise<T> {
    return this.request<T>("DELETE", path, body, headers);
  }

  /** Returns the full URL for a given path (used for OAuth redirects). */
  url(path: string): string {
    return `${this.baseURL}${path}`;
  }
}
