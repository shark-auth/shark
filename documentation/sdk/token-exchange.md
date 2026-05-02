# Token exchange

RFC 8693. Take a token, narrow it, hand it to a sub-agent. The new token's payload extends the parent's `act` chain — the resource server can prove who delegated to whom.

This is how SharkAuth implements "agent acts on behalf of user via sub-agents." Read [Delegation and agents](./delegation-and-agents.md) first if you haven't.

## Method shape

```python
def token_exchange(
    self,
    *,
    subject_token: str,
    dpop_prover: DPoPProver,
    scope: str | None = None,
    audience: str | None = None,
    actor_token: str | None = None,
    subject_token_type: str = "urn:ietf:params:oauth:token-type:access_token",
    requested_token_type: str = "urn:ietf:params:oauth:token-type:access_token",
    **extra,
) -> Token
```

```typescript
function exchangeToken(opts: {
  authUrl: string;
  clientId: string;
  clientSecret?: string;
  subjectToken: string;
  subjectTokenType?: string;
  actorToken?: string;
  actorTokenType?: string;
  scope?: string;
  requestedTokenType?: string;
  dpopProver?: DPoPProver;
  tokenPath?: string;
}): Promise<TokenResponse>
```

## Parameters

| Param                   | Required | Notes                                                  |
| ----------------------- | -------- | ------------------------------------------------------ |
| `subject_token`         | yes      | Token being exchanged (parent's `access_token`)        |
| `dpop_prover`           | yes (Py) | Same keypair preserves DPoP binding across exchange    |
| `scope`                 | no       | Subset of parent's scope. Server rejects expansion.    |
| `audience`              | no       | Restrict child token to a single resource server       |
| `actor_token`           | no       | Token of the agent doing the delegation — adds `act`   |
| `subject_token_type`    | no       | Default `urn:...:access_token`                         |
| `requested_token_type`  | no       | Default `urn:...:access_token`. Pass `:jwt` for JWT.   |

## Common patterns

### Downscope

```python
# parent has "calendar:read calendar:write"
read_only = oauth.token_exchange(
    subject_token=parent.access_token,
    dpop_prover=prover,
    scope="calendar:read",
)
```

### Audience-bind

```python
gmail_only = oauth.token_exchange(
    subject_token=parent.access_token,
    dpop_prover=prover,
    scope="gmail:read",
    audience="https://gmail.googleapis.com",
)
```

### Build `act` chain — orchestrator delegates to worker

```python
# Worker exchanges to record itself as the actor in the chain.
worker_token = oauth.token_exchange(
    subject_token=user_token.access_token,    # original human's token
    actor_token=orchestrator_token.access_token,  # who's delegating now
    dpop_prover=worker_prover,
    scope="files:read",
)

# Resource server reads:
#   sub = usr_alice
#   act = { sub: agent_orchestrator, act: { sub: usr_alice, ... } }
#   scope = "files:read"  (narrowed)
```

### Cross-key exchange (rotate the binding)

```python
new_prover = DPoPProver.generate()
fresh = oauth.token_exchange(
    subject_token=old.access_token,
    dpop_prover=new_prover,    # NEW keypair → new cnf.jkt
)
assert fresh.cnf_jkt == new_prover.jkt
```

The server emits the new token bound to the new key; the old token is unchanged but the new one carries a different `cnf.jkt`.

## Errors

| Error code        | When                                                       |
| ----------------- | ---------------------------------------------------------- |
| `invalid_token`   | `subject_token` revoked or expired                         |
| `invalid_scope`   | Requested scope exceeds parent's grant                     |
| `invalid_target`  | `audience` not allowed for this client                     |
| `unauthorized_client` | Client not allowed to use token-exchange grant         |

Python raises `OAuthError(error, error_description, status_code)`. TypeScript raises `TokenError`.

## Walking the chain

Once a child token exists, any party with the JWT can walk the chain:

```python
from shark_auth.claims import AgentTokenClaims

claims = AgentTokenClaims.parse(child.access_token)
print(claims.sub)              # original user
for hop in claims.delegation_chain():
    print(hop.sub, hop.scope, hop.jkt, hop.iat)
```

```typescript
// decodeAgentToken is not yet exported from @sharkauth/sdk.
// Decode manually with a JWT library (e.g. jose) or the Node buffer method:
const parts = child.accessToken.split(".");
const payload = JSON.parse(Buffer.from(parts[1], "base64url").toString());
console.log(payload.sub);      // original user
console.log(payload.act);      // actor chain
```

`delegation_chain()` returns hops outermost-first. An empty list means the token is direct (no delegation).

## See also

- [Delegation and agents](./delegation-and-agents.md) — flagship reference
- [Cookbook: multi-hop delegation](./cookbook/multi-hop-delegation.md)
- [DPoP](./dpop.md) — `cnf.jkt` binding mechanics
- [Existing reference: token_exchange](./python/token_exchange.md)
