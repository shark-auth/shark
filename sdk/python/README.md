# shark-auth

Python SDK for [SharkAuth](https://github.com/shark-auth/shark) agent-auth primitives.

Implements the four primitives most agent builders reach for:

1. **`DPoPProver`** — RFC 9449 DPoP proof JWTs (no more hand-rolled JWK x/y / `ath` / `jti`).
2. **`DeviceFlow`** — RFC 8628 device authorization grant with `slow_down` + `expired_token` handling.
3. **`VaultClient`** — fetch auto-refreshed 3rd-party OAuth credentials from Shark's Token Vault.
4. **`decode_agent_token`** — verify a Shark-issued agent access token (signature, exp, iss, aud) with a cached JWKS.

Targets Python 3.9+.

## Install

    pip install shark-auth

## Quickstart 1 — DPoP-bound device flow

Get a DPoP-bound agent token and call a resource server with it.

```python
from shark_auth import DPoPProver, DeviceFlow
import requests

prover = DPoPProver.generate()

flow = DeviceFlow(
    auth_url="https://auth.example",
    client_id="agent_abc",
    scope="resource:read",
    dpop_prover=prover,
)

init = flow.begin()
print(f"Visit {init.verification_uri_complete or init.verification_uri} "
      f"and enter code {init.user_code}")

token = flow.wait_for_approval(timeout_s=300)

# Call a resource with DPoP + bearer
proof = prover.make_proof(
    "GET",
    "https://api.example/data",
    access_token=token.access_token,
)
r = requests.get(
    "https://api.example/data",
    headers={
        "Authorization": f"DPoP {token.access_token}",
        "DPoP": proof,
    },
)
r.raise_for_status()
print(r.json())
```

## Quickstart 2 — DPoPProver standalone

```python
from shark_auth import DPoPProver

# Fresh P-256 keypair
prover = DPoPProver.generate()

# Persist the private key
pem = prover.private_key_pem()
# ...store `pem` safely, then later:
prover = DPoPProver.from_pem(pem)

# Build a proof for a token-endpoint request
proof = prover.make_proof(htm="POST", htu="https://auth.example/oauth/token")

# Include the confirmation claim in your authorization request if the
# server supports `dpop_jkt`:
print("jkt:", prover.jkt)
```

## Quickstart 3 — VaultClient

Pull a fresh 3rd-party access token the Vault refreshed for you.

```python
from shark_auth import VaultClient, VaultError

vault = VaultClient(
    auth_url="https://auth.example",
    access_token=agent_access_token,
)

try:
    fresh = vault.get_fresh_token(connection_id="conn_abc")
except VaultError as e:
    print(f"vault error: {e} (status={e.status_code})")
    raise

print(fresh.provider, fresh.scopes, fresh.access_token[:12] + "...")
```

## Quickstart 4 — decode_agent_token

Resource servers verify agent tokens locally via cached JWKS.

```python
from shark_auth import decode_agent_token, TokenError

try:
    claims = decode_agent_token(
        token=jwt_string,
        jwks_url="https://auth.example/.well-known/jwks.json",
        expected_issuer="https://auth.example",
        expected_audience="https://api.my-app.example",
    )
except TokenError as e:
    return 401, str(e)

print(claims.sub, claims.scope, claims.agent_id)
print("DPoP-bound jkt:", claims.jkt)
print("RFC 8693 actor:", claims.act)
print("RFC 9396 authz details:", claims.authorization_details)
```

## Publishing (maintainers)

    python -m build
    python -m twine upload dist/*

## License

MIT — see [LICENSE](LICENSE).
