# `OAuthClient.get_token_with_dpop()`

Request a DPoP-bound OAuth access token via POST `/oauth/token`.

---

## Method Signature

```python
def get_token_with_dpop(
    self,
    *,
    grant_type: str,
    dpop_prover: DPoPProver,
    client_id: str,
    client_secret: str | None = None,
    scope: str | None = None,
    **extra: Any,
) -> Token:
```

---

## Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `grant_type` | `str` | Yes | OAuth 2.1 grant type. One of `"client_credentials"`, `"authorization_code"`, `"refresh_token"`. |
| `dpop_prover` | `DPoPProver` | Yes | Holds the P-256 keypair used to sign the DPoP proof JWT. Generate with `DPoPProver.generate()`. |
| `client_id` | `str` | Yes | The client / agent identifier registered with SharkAuth. |
| `client_secret` | `str \| None` | No | Client secret. Omit for public clients. |
| `scope` | `str \| None` | No | Space-separated scopes to request. Omits the `scope` parameter when `None`. |
| `**extra` | `Any` | No | Additional form fields forwarded verbatim to the token endpoint (e.g. `code`, `redirect_uri`, `refresh_token`). |

---

## Return Value — `Token`

```python
@dataclass
class Token:
    access_token: str        # The DPoP-bound access token
    token_type: str          # "DPoP" (or "Bearer" for non-DPoP servers)
    expires_in: int | None   # Lifetime in seconds, or None
    scope: str | None        # Granted scopes, or None
    refresh_token: str | None  # Present only when server issues one
    cnf_jkt: str | None      # JWK thumbprint binding token to the keypair
    raw: dict                # Full server JSON response
```

`cnf_jkt` is the SHA-256 JWK thumbprint of the prover's public key.
A token with `cnf_jkt` set is cryptographically bound — an attacker who
steals the access token cannot use it without also possessing the matching
private key.

---

## Example Usage

```python
from shark_auth import DPoPProver, OAuthClient

# 1. Generate a fresh P-256 keypair (or load one with DPoPProver.from_pem())
prover = DPoPProver.generate()

# 2. Build the client
client = OAuthClient(base_url="https://auth.example.com")

# 3. Request a DPoP-bound token
token = client.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="my-agent",
    client_secret="s3cr3t",
    scope="read write",
)

# 4. Verify binding
assert token.cnf_jkt == prover.jkt  # True — proof of possession

# 5. Use the token in a downstream request
import requests
resp = requests.get(
    "https://api.example.com/resource",
    headers={
        "Authorization": f"DPoP {token.access_token}",
        "DPoP": prover.make_proof(htm="GET", htu="https://api.example.com/resource",
                                   access_token=token.access_token),
    },
)
```

---

## Error Handling

On any 4xx or 5xx response from the token endpoint, `OAuthError` is raised.

```python
from shark_auth.errors import OAuthError

try:
    token = client.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id="my-agent",
        client_secret="wrong",
    )
except OAuthError as e:
    print(e.error)              # "invalid_client"
    print(e.error_description)  # "bad secret" (server-provided, may be None)
    print(e.status_code)        # 401
```

### `OAuthError` fields

| Field | Type | Description |
|---|---|---|
| `error` | `str` | RFC 6749 error code, e.g. `"invalid_client"`, `"invalid_grant"`, `"server_error"`. |
| `error_description` | `str \| None` | Human-readable description from the server. |
| `status_code` | `int \| None` | HTTP status code (401, 400, 500, …). |
