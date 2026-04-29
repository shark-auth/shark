# Errors and retries

Exception hierarchy, what each one means, and the retry strategy you actually want.

## Exception tree

Both SDKs share the same conceptual hierarchy.

### Python

```
SharkAuthError
├── DPoPError              # constructing or signing DPoP proof
├── DeviceFlowError        # RFC 8628 errors
├── VaultError             # vault HTTP failures (status_code attr)
├── TokenError             # decoding / verifying agent JWTs
├── OAuthError             # OAuth token endpoint 4xx/5xx (RFC 6749 fields)
└── SharkAPIError          # generic admin-API failures (code, status attrs)
```

### TypeScript

```
SharkAuthError
├── DPoPError
├── DeviceFlowError
├── VaultError
├── TokenError
└── SharkAPIError          # admin-API failures
```

The TypeScript SDK does not yet export a dedicated `OAuthError` type — `TokenError` covers the token-endpoint cases. (Parity gap; tracked in `sdk/HANDOFF.md`.)

## OAuthError fields (Python)

```python
try:
    oauth.token_exchange(...)
except OAuthError as e:
    print(e.error)              # RFC 6749 error code, e.g. "invalid_scope"
    print(e.error_description)  # human-readable
    print(e.status_code)        # HTTP status
```

## VaultError fields

```python
try:
    vault.fetch_token(provider="google_gmail", bearer_token=t, prover=p)
except VaultError as e:
    if e.status_code == 401:
        # token expired — refresh and retry once
    elif e.status_code == 403:
        # missing vault:read scope — fail loud
    elif e.status_code == 404:
        # no connection — prompt user to connect
```

## SharkAPIError fields

```python
try:
    c.organizations.create(name="...")
except SharkAPIError as e:
    print(e.code, e.status, str(e))
```
```typescript
try {
  await c.organizations.create("...");
} catch (e) {
  if (e instanceof SharkAPIError) {
    console.log(e.code, e.status);
  }
}
```

## Retry strategy

The SDKs do **not** retry automatically (except `VaultClient.exchange` / `fetch_token` via the `onRefresh` callback in TypeScript). You decide.

### What to retry

| Status      | Retry?            | How                                |
| ----------- | ----------------- | ---------------------------------- |
| 200 / 2xx   | n/a               | -                                  |
| 400         | No                | Caller bug                         |
| 401         | Yes (once)        | Refresh token, retry               |
| 403         | No                | Missing scope / wrong principal    |
| 404         | No                | Resource gone                      |
| 409         | Maybe             | Idempotent operation? Re-fetch state |
| 422         | No                | Validation error                   |
| 429         | Yes (with backoff)| Honor `Retry-After`                |
| 5xx         | Yes (with backoff)| Exponential, capped at ~3 attempts |
| Network     | Yes (with backoff)| Same                               |

### Backoff

A reasonable default: exponential with jitter, capped.

```python
import time, random
from shark_auth.errors import SharkAuthError

def with_retry(fn, attempts=3, base=0.5):
    for i in range(attempts):
        try:
            return fn()
        except SharkAuthError as e:
            status = getattr(e, "status_code", 0) or getattr(e, "status", 0)
            if status in (401, 429) or 500 <= (status or 0) < 600:
                if i == attempts - 1:
                    raise
                time.sleep(base * (2 ** i) + random.random() * 0.1)
            else:
                raise
```

```typescript
async function withRetry<T>(fn: () => Promise<T>, attempts = 3): Promise<T> {
  for (let i = 0; i < attempts; i++) {
    try {
      return await fn();
    } catch (e: any) {
      const status = e?.status ?? e?.statusCode ?? 0;
      const retryable = status === 401 || status === 429 || (status >= 500 && status < 600);
      if (!retryable || i === attempts - 1) throw e;
      await new Promise(r => setTimeout(r, 500 * 2 ** i + Math.random() * 100));
    }
  }
  throw new Error("unreachable");
}
```

### What not to retry

- Token revocation (`oauth.revoke_token`) — server returns 200 even if token didn't exist; failures are network-only and OK to retry.
- Audit purge — guard with `dry_run=True` first.
- DPoP key rotation — irreversible.
- Webhook test fire — duplicates are fine, but pile up in delivery logs.

## DPoP-specific failures

`DPoPError` from `make_proof` / `createProof` usually means one of:

- The keypair is malformed (caller passed a non-P-256 PEM).
- `htu` URL is malformed.
- Signing failed in Web Crypto (TypeScript) — typically permission/CSP issue in a browser.

All four cases are programmer error, not transient — do not retry.

## Surfacing errors to users

The server returns RFC 6749 / 7807 style bodies:

```json
{ "error": "invalid_scope", "error_description": "scope exceeds parent grant" }
```

The SDK lifts these into the exception message. Log `error_description` (when present) — it's the user-facing copy.

## See also

- [DPoP](./dpop.md) — `DPoPError` causes
- [Token exchange](./token-exchange.md) — common `OAuthError` codes
- [Vault](./vault.md) — 401/403/404 paths
