# @sharkauth/node

TypeScript SDK for [SharkAuth](https://github.com/shark-auth/shark) agent-auth primitives.

Implements the four primitives most agent builders reach for:

1. **`DPoPProver`** — RFC 9449 DPoP proof JWTs (ES256 via `jose`).
2. **`DeviceFlow`** — RFC 8628 device authorization grant with `slow_down` + `expired_token` handling.
3. **`VaultClient`** — fetch auto-refreshed 3rd-party OAuth credentials from Shark's Token Vault.
4. **`decodeAgentToken`** — decode a Shark-issued agent token payload (no signature verification).

Targets Node 18+ (uses built-in `fetch` and `crypto.subtle`).

> **New to SharkAuth?** The [Hello Agent walkthrough](../../docs/hello-agent.md)
> takes you from zero to a working DPoP-bound agent token in 15 minutes
> using this SDK.

## Install

```sh
npm install @sharkauth/node
```

## Quickstart 1 — DPoP-bound device flow

```ts
import { DPoPProver, DeviceFlow } from "@sharkauth/node";

const prover = await DPoPProver.generate();

const flow = new DeviceFlow({
  authUrl: "https://auth.example",
  clientId: "agent_abc",
  scope: "resource:read",
  dpopProver: prover,
});

const init = await flow.begin();
console.log(`Visit ${init.verificationUriComplete ?? init.verificationUri}`);
console.log(`User code: ${init.userCode}`);

const token = await flow.waitForApproval({ timeoutMs: 300_000 });

// Call a resource server with DPoP + bearer
const proof = await prover.createProof({
  method: "GET",
  url: "https://api.example/data",
  accessToken: token.accessToken,
});

const res = await fetch("https://api.example/data", {
  headers: {
    Authorization: `DPoP ${token.accessToken}`,
    DPoP: proof,
  },
});
console.log(await res.json());
```

## Quickstart 2 — DPoPProver standalone

```ts
import { DPoPProver } from "@sharkauth/node";

// Generate a fresh P-256 keypair
const prover = await DPoPProver.generate();

// Persist the private key (PKCS#8 PEM)
const pem = await prover.privateKeyPem();
// ...store `pem` securely, then later:
const restoredProver = await DPoPProver.fromPem(pem);

// Build a proof for a token-endpoint request
const proof = await prover.createProof({
  method: "POST",
  url: "https://auth.example/oauth/token",
});

// Include the confirmation claim in your authorization request
console.log("jkt:", prover.jkt);
```

## Quickstart 3 — VaultClient

```ts
import { VaultClient, VaultError } from "@sharkauth/node";

const vault = new VaultClient({
  authUrl: "https://auth.example",
  accessToken: agentAccessToken,
  // Optional: auto-refresh the agent token on 401
  onRefresh: async () => await refreshAgentToken(),
});

try {
  const fresh = await vault.exchange("conn_abc");
  console.log(fresh.provider, fresh.scopes, fresh.accessToken.slice(0, 12) + "...");
} catch (e) {
  if (e instanceof VaultError) {
    console.error(`Vault error: ${e.message} (status=${e.statusCode})`);
  }
  throw e;
}
```

## Quickstart 4 — decodeAgentToken

Decode a Shark-issued agent JWT payload without signature verification (for
use in contexts where trust has already been established, e.g. after edge
middleware validation).

```ts
import { decodeAgentToken, TokenError } from "@sharkauth/node";

try {
  const claims = decodeAgentToken(jwtString);
  console.log(claims.sub, claims.scope);
  console.log("DPoP-bound jkt:", claims.cnf?.jkt);
  console.log("RFC 8693 actor:", claims.act);
  console.log("RFC 9396 authz details:", claims.authorization_details);
} catch (e) {
  if (e instanceof TokenError) {
    return { status: 401, body: String(e) };
  }
  throw e;
}
```

## Error classes

All errors extend `SharkAuthError`:

| Class | Thrown by |
|---|---|
| `DPoPError` | `DPoPProver` |
| `DeviceFlowError` | `DeviceFlow` |
| `VaultError` | `VaultClient` (has `.statusCode`) |
| `TokenError` | `decodeAgentToken` |

## License

MIT — see [LICENSE](LICENSE).
