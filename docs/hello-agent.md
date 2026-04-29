# Hello Agent â€” 15-minute walkthrough

Stand up SharkAuth, register an OAuth-2.1 agent, mint a DPoP-bound access
token from Python, and verify claims against the server â€” end to end, in
under fifteen minutes.

## What you'll build

A working local SharkAuth deployment with one registered agent. A Python
script that mints a `client_credentials` access token, attaches an RFC
9449 DPoP proof JWT, and decodes the token's claims locally with three
lines of SDK code. At the end you will have the exact four primitives
every agent builder reaches for: agent registration, DPoP proof emission,
token minting, and JWKS-based JWT verification.

## Prerequisites

- **Python 3.9+** and `pip` (or `uv`) on your `$PATH`
- **Go 1.22+** â€” you are going to build `shark` from source
- A terminal, `curl`, and `jq`
- 15 minutes

A one-shot automated run of this whole walkthrough lives at
[`examples/hello_agent.sh`](../examples/hello_agent.sh). Everything below
is the same path step-by-step.

---

## Step 1 â€” Install SharkAuth (2 min)

Clone and build the binary from source. Installers (`curl | sh`,
Homebrew, `go install`) are coming with the public launch.

```bash
git clone https://github.com/shark-auth/shark && cd shark
go build -o bin/shark ./cmd/shark
./bin/shark version
```

## Step 2 â€” Start the server (1 min)

`--dev` boots an ephemeral SQLite store, auto-creates a default OAuth
application, and prints a one-time admin API key. No config file or setup
wizard required in dev mode â€” just run and go.

```bash
./bin/shark serve --dev
```

Expected output (truncated):

```
migrations complete
dev mode: using in-db dev inbox for email capture

  ============================================================
    Default application created
    client_id:     shark_app_...
    client_secret: ...                (shown once â€” save it)
  ============================================================

  ADMIN API KEY (shown once â€” save it now)
    sk_live_...

SharkAuth starting                    addr=:8080 dev_mode=true
admin dashboard                        url=http://localhost:8080/admin
health check                           url=http://localhost:8080/healthz
```

Leave this running in one terminal. In a second terminal:

```bash
curl -s http://localhost:8080/healthz
# {"status":"ok"}
```

## Step 3 â€” Register an agent (2 min)

Agents are the OAuth 2.1 clients that obtain access tokens on behalf of
your software. Register one using **RFC 7591 Dynamic Client Registration**
against the unauthenticated `/oauth/register` endpoint:

```bash
curl -sS -X POST http://localhost:8080/oauth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "client_name":"hello-agent",
    "grant_types":["client_credentials"],
    "token_endpoint_auth_method":"client_secret_basic",
    "scope":"openid"
  }' | jq .
```

Response fields that matter:

| field | meaning |
|-------|---------|
| `client_id` | agent identifier, e.g. `shark_dcr_abc...` |
| `client_secret` | long random secret â€” **save it, it is shown once** |
| `registration_access_token` | bearer for future RFC 7592 updates/rotation |
| `registration_client_uri` | `GET`/`PUT`/`DELETE` endpoint for this client |
| `grant_types` | the grants this client may use at `/oauth/token` |

Stash two env vars:

```bash
export CID='shark_dcr_...'
export CSECRET='...'
```

*(If you prefer the CLI, `./bin/shark app create --name hello-agent --json`
registers a regular OAuth Application â€” good for the Authorization Code
flow but not for `client_credentials`. DCR is the agent-native path.)*

## Step 4 â€” Install the Python SDK (1 min)

`shark-auth` is not yet on PyPI â€” install directly from the repo:

```bash
python3 -m venv .venv && source .venv/bin/activate
pip install git+https://github.com/shark-auth/shark#subdirectory=sdk/python
# PyPI release coming after dogfood validation
```

Or install editable from a local clone:

```bash
pip install -e sdk/python
```

Or with `uv`:

```bash
uv venv && source .venv/bin/activate
uv pip install git+https://github.com/shark-auth/shark#subdirectory=sdk/python
```

Smoke-test the import:

```bash
python -c "import shark_auth; print(shark_auth.__version__)"
# 0.1.0
```

## Step 5 â€” Write your first agent (5 min)

A ~30-line script that mints a token and emits a DPoP proof. The full
runnable version lives at
[`examples/hello_agent.py`](../examples/hello_agent.py).

```python
import requests
from shark_auth import DPoPProver

AUTH = "http://localhost:8080"
CID = "shark_dcr_..."      # from Step 3
CSECRET = "..."            # from Step 3

# 1. Mint an access token via client_credentials.
resp = requests.post(
    f"{AUTH}/oauth/token",
    data={"grant_type": "client_credentials", "scope": "openid"},
    auth=(CID, CSECRET),
    timeout=10,
)
resp.raise_for_status()
token = resp.json()["access_token"]

# 2. Emit a DPoP proof bound to a protected resource call.
prover = DPoPProver.generate()
proof = prover.make_proof(
    htm="GET",
    htu="https://api.example/data",
    access_token=token,
)
print("jkt:", prover.jkt)
print("proof (first 60):", proof[:60], "...")
```

`DPoPProver.generate()` creates a fresh ECDSA P-256 keypair. `make_proof`
returns a JWT with an `ath` claim (SHA-256 of the access token), a `jti`
for replay protection, and the public key embedded in the header
(`jwk`). Resource servers verify the proof and that its `jkt` thumbprint
matches the `cnf.jkt` in the access token.

Why `client_credentials` and not Device Flow here? CI-friendly: no human
has to type a code into a browser. Device Flow shines when the same agent
is paired to a specific *user*; see `DeviceFlow` in the SDK for that.

## Step 6 â€” Call a protected resource (2 min)

Attach both headers â€” `Authorization: DPoP <token>` and `DPoP: <proof>` â€”
to your resource request:

```python
r = requests.get(
    "https://api.example/data",
    headers={"Authorization": f"DPoP {token}", "DPoP": proof},
    timeout=10,
)
```

SharkAuth issues **real RFC 7519 JWTs** out of the box, signed with the
ES256 key advertised at `/.well-known/jwks.json`. Resource servers can
verify them locally with three lines of SDK code â€” no introspection
round-trip required:

```python
from urllib.parse import urljoin
from shark_auth import decode_agent_token

claims = decode_agent_token(
    token,
    urljoin(AUTH + "/", ".well-known/jwks.json"),
    expected_issuer=AUTH,
    expected_audience=CID,
)
print("sub:", claims.sub, "exp:", claims.exp, "scope:", claims.scope)
```

`decode_agent_token` fetches and caches the JWKS, verifies the signature
+ `exp`/`nbf`, and checks `iss` and `aud` against the values you pass in.

If you prefer the round-trip (e.g. to centralize revocation checks),
RFC 7662 introspection still works against the same `/oauth/introspect`
endpoint and returns `{active, sub, iss, exp, scope, ...}`.

## Step 7 â€” Retrieve a 3rd-party credential via Vault (2 min)

The **Token Vault** stores user-granted OAuth connections (Google,
GitHub, Slack, ...) and hands back refreshed access tokens on demand so
your agent never sees the end user's refresh token.

```python
from shark_auth import VaultClient

vault = VaultClient(auth_url=AUTH, access_token=delegated_agent_token)
fresh = vault.get_fresh_token(connection_id="google")
print(fresh.provider, fresh.scopes, fresh.access_token[:12] + "...")
```

Two caveats the walkthrough can't fully automate:

- The vault endpoint requires a **user-bound** token (the agent acting on
  behalf of a specific user), not a raw `client_credentials` token. Grab
  one through Device Flow or Token Exchange with a user delegation.
- A provider template must be configured first. The admin dashboard at
  `http://localhost:8080/admin` walks you through adding Google / GitHub
  / custom providers.

See [`AGENT_AUTH.md`](../AGENT_AUTH.md) Token Vault section for the
end-to-end delegation flow.

---

## What just happened

You stood up an OAuth 2.1 authorization server, registered an agent over
RFC 7591, minted a JWT access token over `client_credentials`, built an
RFC 9449 DPoP proof JWT that binds that token to your ECDSA keypair, and
verified the token's claims locally with `decode_agent_token` against
the server's published JWKS. That's the full spec-compliant agent-auth
loop with zero introspection round-trips.

Architecture deep-dive: [`AGENT_AUTH.md`](../AGENT_AUTH.md).

## Next steps

- **Device Flow** â€” headless agents that need user approval:
  `shark_auth.DeviceFlow`.
- **RFC 8693 Token Exchange** â€” delegation + `act` (actor) chains for
  multi-agent systems. See `AGENT_AUTH.md` â†’ Token Exchange.
- **Custom claim injection** â€” extend the JWT body with deployment-
  specific extras (planned, see Implementation Status in
  [`AGENT_AUTH.md`](../AGENT_AUTH.md)).
- **Vault provider setup** â€” admin dashboard â†’ Vault â†’ Providers.
- **RFC compliance matrix** â€” [`AGENT_AUTH.md`](../AGENT_AUTH.md)
  Implementation Status section.
