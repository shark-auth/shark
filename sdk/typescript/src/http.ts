/**
 * Minimal fetch wrapper used internally by the SDK.
 *
 * Parses structured JSON error envelopes and applies a default timeout via
 * AbortController. Uses the built-in `fetch` on Node 18+; no extra deps.
 */

const DEFAULT_TIMEOUT_MS = 10_000;
const USER_AGENT = "sharkauth-node/0.1.0";

/** Shape of a structured error envelope returned by SharkAuth endpoints. */
export interface StructuredError {
  error: string;
  error_description?: string;
}

/** Parsed HTTP response (body already consumed). */
export interface HttpResponse {
  status: number;
  headers: Record<string, string>;
  /** Raw text body. */
  text: string;
  /** Parsed JSON body, or null if body is not valid JSON. */
  json: <T = unknown>() => T;
}

/** Options for {@link httpRequest}. */
export interface HttpRequestOptions {
  method?: string;
  headers?: Record<string, string>;
  /** Form-encoded body (sends as application/x-www-form-urlencoded). */
  form?: Record<string, string>;
  /** JSON body. */
  body?: string;
  timeoutMs?: number;
}

/**
 * Perform an HTTP request using the Node 18+ built-in `fetch`.
 *
 * @param url    Target URL.
 * @param opts   Request options.
 * @returns Parsed {@link HttpResponse}.
 */
export async function httpRequest(
  url: string,
  opts: HttpRequestOptions = {}
): Promise<HttpResponse> {
  const {
    method = "GET",
    headers: extraHeaders = {},
    form,
    body: rawBody,
    timeoutMs = DEFAULT_TIMEOUT_MS,
  } = opts;

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  const headers: Record<string, string> = {
    "User-Agent": USER_AGENT,
    Accept: "application/json",
    ...extraHeaders,
  };

  let bodyInit: string | undefined;

  if (form) {
    bodyInit = new URLSearchParams(form).toString();
    headers["Content-Type"] = "application/x-www-form-urlencoded";
  } else if (rawBody !== undefined) {
    bodyInit = rawBody;
  }

  let response: Response;
  try {
    response = await fetch(url, {
      method: method.toUpperCase(),
      headers,
      body: bodyInit,
      signal: controller.signal,
    });
  } catch (err) {
    clearTimeout(timer);
    throw err;
  }
  clearTimeout(timer);

  const text = await response.text();
  const respHeaders: Record<string, string> = {};
  response.headers.forEach((value, key) => {
    respHeaders[key] = value;
  });

  let parsed: unknown = null;
  let parseError: unknown = null;
  try {
    parsed = JSON.parse(text);
  } catch (e) {
    parseError = e;
  }

  return {
    status: response.status,
    headers: respHeaders,
    text,
    json<T = unknown>(): T {
      if (parseError !== null) {
        throw new SyntaxError(`Response body is not valid JSON: ${text.slice(0, 200)}`);
      }
      return parsed as T;
    },
  };
}
