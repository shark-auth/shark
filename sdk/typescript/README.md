# @sharkauth/sdk

TypeScript SDK for [SharkAuth](https://github.com/shark-auth/shark) agent-auth primitives and v1.5 admin APIs.

Targets Node 18+ (uses built-in `fetch` and `crypto.subtle`).

> **New to SharkAuth?** The [Hello Agent walkthrough](../../docs/hello-agent.md)
> takes you from zero to a working DPoP-bound agent token in 15 minutes
> using this SDK.

## Install

```sh
npm install @sharkauth/sdk
```

## Exported primitives

**Auth & OAuth**
- `AuthClient` — signup, login, sessions, password reset, magic links
- `OAuthClient` — token, revoke, introspect, refresh, PKCE, authorize URL builder
- `DeviceFlow` — RFC 8628 device authorization grant
- `DPoPProver` — RFC 9449 DPoP proof JWTs (ES256 via `jose`)

**Admin & Platform**
- `SharkClient` — composer client with 18 namespaces (users, agents, orgs, RBAC, audit, webhooks, proxy, branding, paywall, etc.)
- `ApiKeysClient`, `AgentsClient`, `AuditClient`, `BrandingClient`, `ConsentsClient`, `DcrClient`, `MagicLinkClient`, `MayActClient`, `OrganizationsClient`, `PaywallClient`, `ProxyLifecycleClient`, `ProxyRulesClient`, `RbacClient`, `SessionsClient`, `UsersClient`, `WebhooksClient`

**Utilities**
- `exchangeToken` — RFC 8693 token exchange (free function)
- `pkcePair` — PKCE verifier/challenge generator
- `computeTotp` — TOTP code generator
- `verifySignature` — async webhook signature verifier

See `src/index.ts` for the full public API.

## Quickstart 1 — DPoP-bound device flow

```ts
import { DPoPProver, DeviceFlow } from "@sharkauth/sdk";

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
import { DPoPProver } from "@sharkauth/sdk";

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
import { VaultClient, VaultError } from "@sharkauth/sdk";

const vault = new VaultClient({
  authUrl: "https://auth.example",
  accessToken: agentAccessToken,
  // Optional: auto-refresh the agent token on 401
  onRefresh: async () => await refreshAgentToken(),
});

try {
  const fresh = await vault.fetchToken("conn_abc");
  console.log(fresh.provider, fresh.scopes, fresh.accessToken.slice(0, 12) + "...");
} catch (e) {
  if (e instanceof VaultError) {
    console.error(`Vault error: ${e.message} (status=${e.statusCode})`);
  }
  throw e;
}
```

## Quickstart 4 — SharkClient composer

Most admin and platform operations go through the composer client:

```ts
import { SharkClient } from "@sharkauth/sdk";

const c = new SharkClient({
  baseUrl: "https://auth.example",
  adminKey: "sk_live_admin",
});

const users = await c.users.listUsers();
const agent = await c.agents.registerAgent({
  name: "my-agent",
  scopes: ["vault:read"],
});
```

## Error classes

All errors extend `SharkAuthError`:

| Class | Thrown by |
|---|---|
| `DPoPError` | `DPoPProver` |
| `DeviceFlowError` | `DeviceFlow` |
| `VaultError` | `VaultClient` (has `.statusCode`) |
| `TokenError` | Token parsing / verification failures |

## License

MIT — see [LICENSE](LICENSE).
