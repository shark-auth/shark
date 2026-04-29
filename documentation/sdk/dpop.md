# DPoP

RFC 9449 sender-constrained tokens. Every Shark-issued agent token can be cryptographically bound to a specific keypair via the `cnf.jkt` claim. The holder must prove possession of the private key on every request by attaching a fresh DPoP proof JWT.

The SDK ships:

- `DPoPProver` — keypair holder + proof emitter
- `DPoPHTTPClient` (Python) — drop-in HTTP helper that auto-attaches proofs
- `SharkClient.fetch` (TypeScript) — same idea, `fetch`-shaped

## Generate or load a keypair

```python
from shark_auth import DPoPProver

# Fresh ECDSA P-256 keypair.
prover = DPoPProver.generate()

# Or load from a previously persisted PEM.
prover = DPoPProver.from_pem(open("agent.pem", "rb").read())

print(prover.jkt)               # SHA-256 thumbprint (RFC 7638)
print(prover.public_jwk)        # {"kty": "EC", "crv": "P-256", "x": "...", "y": "..."}
pem = prover.private_key_pem()  # PKCS#8 unencrypted
```

```typescript
import { DPoPProver } from "@sharkauth/sdk";

const prover = await DPoPProver.generate();
console.log(prover.jkt);
console.log(prover.publicJwk);
// PEM round-trips work in Node only — see browser caveat below.
```

ECDSA P-256 only. RSA and Ed25519 not supported by the server (yet).

## Emit a proof manually

A proof is a short-lived JWT bound to a specific HTTP method, URL, and (optionally) the access token via the `ath` claim.

```python
proof = prover.make_proof(
    htm="GET",
    htu="https://api.example.com/calendar/events",
    access_token=token.access_token,  # optional — adds ath claim
)
headers = {
    "Authorization": f"DPoP {token.access_token}",
    "DPoP": proof,
}
```

```typescript
const proof = await prover.createProof({
  method: "GET",
  url: "https://api.example.com/calendar/events",
  accessToken: token.accessToken,
});
const headers = {
  Authorization: `DPoP ${token.accessToken}`,
  DPoP: proof,
};
```

## Auto-DPoP HTTP helper (Python)

```python
from shark_auth import DPoPHTTPClient

http = DPoPHTTPClient(
    base_url="https://api.example.com",
    prover=prover,
    access_token=token.access_token,
)

resp = http.get("/calendar/events")
http.post("/calendar/events", json={"title": "Demo"})
http.delete("/calendar/events/123")
```

Handles `Authorization: DPoP <token>` and a fresh `DPoP` proof on every call. The proof is bound to the exact `method` + URL.

## Auto-DPoP fetch (TypeScript)

```typescript
import { SharkClient, DPoPProver } from "@sharkauth/sdk";

const prover = await DPoPProver.generate();
const c = new SharkClient({
  baseUrl: "https://auth.example.com",
  accessToken: token.accessToken,
  dpopProver: prover,
});

const resp = await c.fetch("https://api.example.com/calendar/events");
```

When `dpopProver` is set, `c.fetch` automatically:

- Sets `Authorization: DPoP <accessToken>`
- Generates a fresh proof bound to the request method + URL
- Sets `DPoP: <proof>`

## Token binding — `cnf.jkt`

When you request a token via `get_token_with_dpop` (Python) or via `/oauth/token` with a DPoP header (TS), the server embeds the prover's JWK thumbprint as `cnf.jkt` in the token. The resource server validates two things on every request:

1. The DPoP proof was signed by a key whose JWK thumbprint matches `cnf.jkt`.
2. The proof's `htm` / `htu` match the actual HTTP method and URL.

A stolen token is useless without the matching private key.

## Key rotation

If a private key may be compromised, rotate via the admin API:

```python
new_prover = DPoPProver.generate()
result = c.agents.rotate_dpop_key(
    "agent_abc",
    new_public_key_jwk=new_prover.public_jwk,
    reason="key compromise 2026-04-26",
)
print(result.old_jkt, "->", result.new_jkt)
print(result.revoked_token_count)  # tokens bound to old key are killed
```

```typescript
const result = await c.agents.rotateDpopKey(
  "agent_abc",
  newProver.publicJwk,
  "key compromise 2026-04-26"
);
```

After rotation, the agent must request fresh tokens with `new_prover`.

## Browser caveat

The TypeScript DPoP path uses `jose`'s `exportPKCS8` / `importPKCS8` for PEM round-trips, which depends on Node-only crypto APIs. Browser builds work for everything except DPoP key PEM persistence — which most browser-resident agents don't need (they regenerate keys per session).

The proof emission itself (`createProof`) works fine in browsers via Web Crypto.

## See also

- [Delegation and agents](./delegation-and-agents.md)
- [Token exchange](./token-exchange.md) — preserving DPoP binding across exchange
- [Existing reference: get_token_with_dpop](./python/get_token_with_dpop.md)
- [Existing reference: http_with_dpop](./python/http_with_dpop.md)
