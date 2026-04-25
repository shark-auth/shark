# http.ts

**Path:** `sdk/typescript/src/http.ts`
**Type:** Internal fetch wrapper
**LOC:** 114

## Purpose
Minimal internal fetch wrapper used by all SDK modules (device flow, vault, exchange, every admin namespace). Adds default timeout, form encoding, response parsing — no third-party deps.

## Public API (internal — not in barrel)
- `function httpRequest(url: string, opts?: HttpRequestOptions): Promise<HttpResponse>`
- `interface HttpResponse`
  - `status: number`
  - `headers: Record<string, string>`
  - `text: string` — raw body
  - `json<T>(): T` — throws SyntaxError if body wasn't valid JSON
- `interface HttpRequestOptions`
  - `method?: string` (default `GET`)
  - `headers?: Record<string, string>`
  - `form?: Record<string, string>` — sent as `application/x-www-form-urlencoded`
  - `body?: string` — raw body
  - `timeoutMs?: number` (default 10 000)
- `interface StructuredError` — `{ error, error_description? }`

## Defaults injected
- `User-Agent: sharkauth-node/0.1.0`
- `Accept: application/json`
- 10 s timeout via `AbortController`

## Behavior
- `form` overrides `body` and sets `Content-Type` automatically.
- Body is consumed eagerly (`await response.text()`); `.json()` is a re-parser closure.
- Headers are flattened into a plain object (last-write-wins for duplicates).

## Internal dependencies
- None — uses native `fetch` (Node 18+) and `URLSearchParams`.

## Notes
- No retries here — caller (e.g. `VaultClient`) implements its own retry logic.
- Network errors propagate after `clearTimeout` to avoid leaks.
