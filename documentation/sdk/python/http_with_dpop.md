# HTTP Helpers with DPoP — `get_with_dpop` / `post_with_dpop` / `delete_with_dpop`

## Overview

SharkAuth resource endpoints require **DPoP-bound access tokens** (RFC 9449).
Each HTTP request must carry:

- `Authorization: DPoP <access_token>` — the token obtained via Method 1 or 2.
- `DPoP: <proof-JWT>` — a **fresh, per-request** signed proof that binds the
  request to the keypair used during token issuance.

`DPoPHTTPClient` handles both headers automatically. You never construct a DPoP
proof manually when using these helpers.

---

## Method Signatures

```python
class DPoPHTTPClient:
    def __init__(self, base_url: str, *, timeout: float = 30.0, session=None): ...

    def get_with_dpop(
        self, path: str, *, token: str, prover: DPoPProver, **kwargs
    ) -> requests.Response: ...

    def post_with_dpop(
        self, path: str, *, token: str, prover: DPoPProver,
        json: Any = None, **kwargs
    ) -> requests.Response: ...

    def delete_with_dpop(
        self, path: str, *, token: str, prover: DPoPProver, **kwargs
    ) -> requests.Response: ...

    def close(self) -> None: ...
```

`**kwargs` are forwarded directly to `requests.Session.request` (e.g. `params`,
`headers`, `data`, `timeout`).  Any `headers` you pass are merged with —
not replacing — the DPoP headers.

---

## Why DPoP Per Request (RFC 9449)

DPoP (Demonstrating Proof of Possession) solves token theft.  A stolen bearer
token can be replayed from any client.  With DPoP:

1. The **token is bound** to a public key at issuance (the `jkt` claim in the
   token's payload references the prover's JWK thumbprint).
2. Every request carries a **fresh proof JWT** signed by the matching private
   key (the `htm`/`htu` claims tie the proof to this specific method + URL).
3. The server verifies both the token binding and the proof signature —
   a stolen token is useless without the private key.

Each proof contains a unique `jti` (JWT ID) generated per call.  Servers
**must** reject replayed proofs, so `DPoPProver` never reuses a `jti`.

---

## The `ath` Claim — Binding the Proof to a Token

RFC 9449 §4.3 defines the **`ath` claim** (access token hash):

```
ath = BASE64URL(SHA-256(ASCII(access_token)))
```

Including `ath` in the proof cryptographically links the proof to the specific
access token being presented.  Without `ath`, an attacker who intercepts the
proof JWT could potentially use it with a different token.

`DPoPHTTPClient` computes `ath` automatically from the `token` argument and
passes it to `DPoPProver.make_proof(access_token=token)`, which handles the
SHA-256 + base64url encoding internally.

---

## Example Usage — the Killer 10-liner

```python
from shark_auth import Client, OAuthClient
from shark_auth.dpop import DPoPProver

# 1. Generate (or load) your keypair — keep the prover alive for the session.
prover = DPoPProver.generate()

# 2. Build a client — .http is a DPoPHTTPClient pointed at the same server.
client = Client(base_url="https://auth.example.com", token="sk_live_...")

# 3. Exchange credentials for a DPoP-bound access token (Method 1).
oauth = OAuthClient(base_url=client.base_url)
token_resp = oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="agt_abc123",
    client_secret="cs_...",
)
access_token = token_resp.access_token

# 4. Hit any DPoP-protected endpoint — proof is built and attached automatically.
resp = client.http.get_with_dpop("/api/v1/auth/me", token=access_token, prover=prover)
resp.raise_for_status()
print(resp.json())  # {"sub": "agt_abc123", ...}
```

---

## When to Use This Helper vs Raw `requests`/`httpx`

| Situation | Use |
|---|---|
| Calling a SharkAuth DPoP-protected endpoint | `client.http.get_with_dpop(...)` |
| Calling the SharkAuth **admin** API (plain bearer) | `client.proxy_rules`, `client.agents`, etc. |
| Calling a **third-party** API with no DPoP requirement | Raw `requests.get(...)` |
| Constructing a custom DPoP proof with a nonce | `DPoPProver.make_proof(htm=..., htu=..., nonce=...)` directly |

The helper exists precisely for the first case — shark-protected resources that
validate both the `Authorization: DPoP` header and the per-request proof.  For
everything else, reach for plain `requests` or whichever HTTP library fits
your stack.

---

## Context Manager Usage

`DPoPHTTPClient` supports the context manager protocol for clean session teardown:

```python
with DPoPHTTPClient("https://auth.example.com") as http:
    resp = http.get_with_dpop("/api/v1/auth/me", token=at, prover=prover)
```

When you use `Client`, call `client.close()` to release the shared session.
