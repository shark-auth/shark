# Delegation and agents

This is the moat. Read it twice.

SharkAuth lets a software agent act on a user's behalf with cryptographic proof of the chain of authorization. Three primitives compose:

1. **Agent registration** — every machine principal is a first-class OAuth client.
2. **DPoP `cnf.jkt` binding** — every token is sender-constrained to a specific keypair (RFC 9449). Token theft is useless without the matching private key.
3. **`act` claim chains via Token Exchange** — RFC 8693 produces a child token whose payload includes nested `act` objects walking back to the original human. The resource server can prove "this request is on behalf of user X via agents A → B → C" without trusting anyone in the chain.

The shipped surface that makes this work:

- `AgentsClient` — register/list/get/revoke agents and their tokens
- `DPoPProver` — generate keypairs, emit proofs, expose JWK + JKT
- `OAuthClient.get_token_with_dpop()` — DPoP-bound `client_credentials`
- `OAuthClient.token_exchange()` — RFC 8693 with `act`
- `AgentTokenClaims.delegation_chain()` — pure JWT walk of `act` lineage
- `UsersClient.revoke_agents()` — Layer 3 cascade revocation
- `OAuthClient.bulk_revoke_by_pattern()` — GLOB-pattern emergency revoke
- `AgentsClient.rotate_dpop_key()` — admin-forced key rotation

## Mental model

```
human user (sub: usr_alice)
    │
    └─ grants consent → may_act
           │
           └─ agent A (client_id: shark_agent_alice_calendar)
                   │
                   └─ token_exchange( actor: A, subject: A's parent token )
                           │
                           └─ agent B (sub-agent invoked by A)
                                   │
                                   └─ resource server reads `act` chain
                                       and enforces narrowest-scope rule
```

Each token's payload carries `act: { sub: agent_b, cnf: { jkt: ... }, act: { sub: agent_a, ... } }`. The resource server walks the chain via `AgentTokenClaims.delegation_chain()` and applies its own policy.

## Registering an agent

Admin-key authenticated. Request the scopes the agent will ever need; tokens are downscoped from this list.

```python
from shark_auth import Client

c = Client("https://auth.example.com", "sk_live_admin")
agent = c.agents.register_agent(
    app_id="app_calendar",
    name="alice-calendar-agent",
    scopes=["calendar:read", "calendar:write", "vault:read"],
)
print(agent["client_id"])      # shark_agent_xxx
print(agent["client_secret"])  # ONE-TIME — store now
```

```typescript
import { SharkClient } from "@sharkauth/sdk";

const c = new SharkClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_admin" });
const agent = await c.agents.registerAgent({
  name: "alice-calendar-agent",
  scopes: ["calendar:read", "calendar:write", "vault:read"],
  metadata: { app_id: "app_calendar" },
});
console.log(agent.client_id, agent.client_secret);
```

| Field         | Notes                                           |
| ------------- | ----------------------------------------------- |
| `name`        | Required                                        |
| `scopes`      | Maximum scope set the agent can ever request    |
| `metadata`    | Free-form dict; conventionally include `app_id` |
| `client_secret` | Returned exactly once at creation             |
| `redirect_uris` | Optional — needed for non-`client_credentials` grants |

## Generating a DPoP keypair

The keypair never leaves the agent process. The server only ever sees the public JWK and its SHA-256 thumbprint (`jkt`).

```python
from shark_auth import DPoPProver

prover = DPoPProver.generate()
print(prover.jkt)             # binds future tokens
pem = prover.private_key_pem() # persist if the agent restarts
```

```typescript
import { DPoPProver } from "@sharkauth/sdk";

const prover = await DPoPProver.generate();
console.log(prover.jkt);
```

ECDSA P-256 only. Persist the PEM if you need the same agent identity across restarts.

> Browser caveat: `dpop.ts` uses `jose`'s `exportPKCS8` / `importPKCS8` for PEM round-trips, which is Node-only. In-browser agents typically just regenerate keypairs per session, so this rarely matters.

## Requesting an agent token

```python
from shark_auth import OAuthClient

oauth = OAuthClient("https://auth.example.com")
parent = oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id=agent["client_id"],
    client_secret=agent["client_secret"],
    scope="calendar:read calendar:write vault:read",
)
assert parent.cnf_jkt == prover.jkt   # token bound to this keypair
print(parent.access_token)
```

```typescript
import { OAuthClient } from "@sharkauth/sdk";

const oauth = new OAuthClient("https://auth.example.com");
// In TS, use the lower-level token endpoint with DPoP via SharkClient.fetch
// or directly via fetch — there's no get_token_with_dpop helper yet (P1).
// For now, use exchangeToken or the agent's own /oauth/token POST manually.
```

The TS SDK does not yet ship a `getTokenWithDpop` helper (parity gap — see `sdk/HANDOFF.md`). Workaround: use `exchangeToken` from a parent token, or POST `/oauth/token` directly through `SharkClient.fetch` with a DPoP prover attached.

The returned `Token` carries `access_token`, `token_type: "DPoP"`, `cnf_jkt` matching `prover.jkt`, and a `raw` dict with the full server response.

## Token exchange — building the `act` chain

Take a parent token, narrow it, hand it to a sub-agent. The new token has the parent's `act` lineage extended.

```python
child = oauth.token_exchange(
    subject_token=parent.access_token,
    dpop_prover=prover,
    scope="calendar:read",                             # narrower
    audience="https://calendar.example.com",
    actor_token=parent.access_token,                   # parent agent identity
)
assert child.cnf_jkt == parent.cnf_jkt                 # same keypair
```

```typescript
import { exchangeToken } from "@sharkauth/sdk";

const child = await exchangeToken({
  authUrl: "https://auth.example.com",
  clientId: agent.client_id,
  subjectToken: parent.access_token,
  scope: "calendar:read",
  dpopProver: prover,
});
```

| Param               | Required | Notes                                                  |
| ------------------- | -------- | ------------------------------------------------------ |
| `subject_token`     | yes      | The token being exchanged                              |
| `dpop_prover`       | yes (Python) / optional (TS) | Same keypair as parent for binding |
| `scope`             | no       | Narrower than parent — server rejects scope expansion  |
| `audience`          | no       | Restricts the new token to a single resource server    |
| `actor_token`       | no       | Adds `act` claim documenting who delegated             |
| `subject_token_type`| no       | Default `urn:ietf:params:oauth:token-type:access_token`|

Server raises `OAuthError` (Python) / `TokenError` (TS) on `invalid_scope` (asked for more than parent has) or `invalid_token` (parent revoked).

## Walking the chain on the resource server

Pure JWT — no signature verification. Use this in middleware that has already verified the signature out-of-band.

```python
from shark_auth import AgentTokenClaims

claims = AgentTokenClaims.parse(child.access_token)
print(claims.sub)            # the original subject (e.g. usr_alice)
print(claims.scope)          # current narrowed scope
print(claims.jkt)            # current cnf.jkt
print(claims.is_delegated()) # True

for hop in claims.delegation_chain():
    print(hop.sub, hop.scope, hop.jkt, hop.iat)
```

```typescript
import { decodeAgentToken } from "@sharkauth/sdk";

const claims = decodeAgentToken(child.accessToken);
for (const hop of claims.act_chain ?? []) {
  console.log(hop.sub, hop.scope, hop.cnf?.jkt);
}
```

The chain is returned outermost-first (most recent actor) → innermost (original delegator). The token's top-level `sub` is the user that authorized the entire chain.

> The Python SDK ships full JWT verification (`decode_agent_token` against the server's JWKS); the TS SDK currently only decodes (no verify). Use a real JWT library in TS until P1 lands.

## Revocation — five layers

Defense-in-depth. Five SDK methods you'll reach for on the bad day:

| Layer | Action                                  | Method                                                       |
| ----- | --------------------------------------- | ------------------------------------------------------------ |
| 1     | Revoke a single token                   | `OAuthClient.revoke_token()`                                 |
| 2     | Revoke all tokens for one agent         | `AgentsClient.revoke_all(agent_id)`                          |
| 3     | Cascade-revoke all agents owned by user | `UsersClient.revokeUserAgents(user_id)` (TS) / `revoke_agents()` (Py) |
| 4     | Bulk revoke by client_id GLOB pattern   | `OAuthClient.bulk_revoke_by_pattern("shark_agent_v3.2_*")`   |
| 5     | Disconnect vault → cascade agent tokens | `VaultClient.disconnect(connection_id, cascade_to_agents=True)` |

```python
# Layer 3 — user lost their device
result = c.users.revoke_agents("usr_alice", reason="device-lost")
print(result.revoked_agent_ids, result.revoked_consent_count, result.audit_event_id)
```

```typescript
// Layer 3 — user lost their device
const result = await c.users.revokeUserAgents("usr_alice", { reason: "device-lost" });
```
# Layer 4 — emergency rollback of a buggy agent version
result = c.oauth.bulk_revoke_by_pattern(
    client_id_pattern="shark_agent_v3.2_*",
    reason="rollback v3.2",
)
print(result.revoked_count, result.audit_event_id)
```

See `documentation/quickstarts/10-five-layer-revocation.md` for the full philosophy.

## DPoP key rotation

When an agent's private key may be compromised, rotate. The server records the new `cnf.jkt` and revokes all tokens bound to the old key.

```python
from shark_auth import DPoPProver

new_prover = DPoPProver.generate()
result = c.agents.rotate_dpop_key(
    "agent_abc",
    new_public_key_jwk=new_prover.public_jwk,
    reason="scheduled rotation 2026-04-26",
)
print(result.old_jkt, "->", result.new_jkt)
print(result.revoked_token_count, result.audit_event_id)
```

```typescript
const newProver = await DPoPProver.generate();
const result = await c.agents.rotateDpopKey(
  "agent_abc",
  newProver.publicJwk,
  "scheduled rotation"
);
```

The agent must reacquire tokens with the new prover — old tokens are dead.

## Listing tokens / audit

```python
tokens = c.agents.list_tokens("agent_abc")
events = c.agents.get_audit_logs("agent_abc", limit=50)
```

```typescript
const tokens = await c.agents.listTokens("agent_abc");
const events = await c.agents.getAuditLogs("agent_abc", { limit: 50 });
```

`AuditEvent` carries `id`, `event` (e.g. `agent.token_issued`), `actor_id`, `target_id`, `metadata`, `created_at`.

## Listing agents per user

```python
result = c.users.list_agents("usr_alice", filter="created", limit=100)
print(result.total, result.filter)
for a in result.data:
    print(a["client_id"], a["name"])
```

```typescript
const result = await c.users.listUserAgents("usr_alice");
```

`filter` accepts `"created"` (agents the user created), `"authorized"` (agents the user has consented to), or `"all"`.

## Recipes

- [Agent acts on user's Gmail](./cookbook/agent-acts-on-gmail.md) — vault + DPoP end-to-end
- [Multi-hop delegation chain](./cookbook/multi-hop-delegation.md) — orchestrator → worker → tool

## See also

- [DPoP primitives](./dpop.md) — `DPoPProver` deep dive
- [Token exchange](./token-exchange.md) — RFC 8693 details and edge cases
- [Vault](./vault.md) — fetching third-party tokens via DPoP-bound agent tokens
- [Quickstart: five-layer revocation](../quickstarts/10-five-layer-revocation.md)
- [Quickstart: delegation chains](../quickstarts/11-delegation-chains.md)
