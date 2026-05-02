# OAuth clients

Token endpoint, revocation, introspection, refresh, PKCE, authorize-URL builder. Wraps `/oauth/*`.

`OAuthClient` is intentionally low-level — most agent flows compose its primitives. For the higher-level patterns see [Delegation and agents](./delegation-and-agents.md).

## Construct

```python
from shark_auth import OAuthClient

oauth = OAuthClient("https://auth.example.com")

# For introspection (admin scope), pass an admin key.
oauth_admin = OAuthClient("https://auth.example.com", token="sk_live_admin")
```

```typescript
import { OAuthClient } from "@sharkauth/sdk";

const oauth = new OAuthClient({ baseUrl: "https://auth.example.com" });
const oauthAdmin = new OAuthClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_admin" });
```

## Build authorize URL

Pure URL builder. No HTTP call.

```python
from shark_auth import OAuthClient, pkce_pair

verifier, challenge, _ = pkce_pair()
url = OAuthClient.build_authorize_url(
    client_id="my-app",
    redirect_uri="https://app.example.com/cb",
    scope="openid profile",
    state="csrf-xyz",
    code_challenge=challenge,
    base_url="https://auth.example.com",
)
# Redirect the user-agent to `url`. Stash `verifier` in your session.
```

```typescript
import { OAuthClient, pkcePair } from "@sharkauth/sdk";

const { verifier, challenge } = await pkcePair();
const url = OAuthClient.buildAuthorizeUrl({
  client_id: "my-app",
  redirect_uri: "https://app.example.com/cb",
  scope: "openid profile",
  state: "csrf-xyz",
  code_challenge: challenge,
  base_url: "https://auth.example.com",
});
```

## PKCE pair

```python
verifier, challenge, method = pkce_pair()  # method always "S256"
```

```typescript
const { verifier, challenge, method } = await pkcePair();
```

43-char URL-safe base64 verifier; SHA-256 challenge.

## Authorization code → token

```python
token = oauth.get_token_authorization_code(
    code="auth_xyz",
    redirect_uri="https://app.example.com/cb",
    code_verifier=verifier,
    client_id="my-app",
)
print(token.access_token, token.refresh_token)
```

```typescript
const token = await oauth.getTokenAuthorizationCode(
  "auth_xyz",
  "https://app.example.com/cb",
  verifier,
  "my-app",
);
```

## Refresh

```python
new = oauth.refresh_token(
    old.refresh_token,
    client_id="my-app",
)
```

```typescript
const newToken = await oauth.refreshToken(old.refreshToken, "my-app");
```

## DPoP-bound token request

```python
from shark_auth import DPoPProver

prover = DPoPProver.generate()
token = oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="shark_agent_xxx",
    client_secret="...",
    scope="calendar:read",
)
assert token.cnf_jkt == prover.jkt
```

See [DPoP](./dpop.md) for the keypair lifecycle.

## Revoke (RFC 7009)

```python
oauth.revoke_token("eyJhbGci...", token_type_hint="access_token")
```

```typescript
await oauth.revokeToken("eyJhbGci...", "access_token");
```

Always returns 200, regardless of whether the token existed.

## Introspect (RFC 7662)

Requires admin auth.

```python
info = oauth_admin.introspect_token("eyJhbGci...")
if info["active"]:
    print(info["sub"], info["scope"], info["exp"])
```

```typescript
const info = await oauthAdmin.introspectToken("eyJhbGci...");
if (info.active) console.log(info.active, info.sub, info.scope);
```

For invalid/expired tokens the server returns `{"active": false}` — not an error.

## Bulk revoke by GLOB pattern

Layer-4 emergency revocation. Kills every token whose `client_id` matches a SQLite GLOB.

```python
result = oauth.bulk_revoke_by_pattern(
    client_id_pattern="shark_agent_v3.2_*",
    reason="emergency rollback",
)
print(result.revoked_count, result.audit_event_id, result.pattern_matched)
```

`*` matches any sequence, `?` matches one char. Audit event captures the pattern + reason.

## Token exchange

Covered in detail in [Token exchange](./token-exchange.md). Short version:

```python
child = oauth.token_exchange(
    subject_token=parent.access_token,
    dpop_prover=prover,
    scope="calendar:read",                 # narrower
    audience="https://calendar.example.com",
    actor_token=parent.access_token,       # adds act claim
)
```

```typescript
import { exchangeToken } from "@sharkauth/sdk";

const child = await exchangeToken({
  authUrl: "https://auth.example.com",
  clientId: "shark_agent_xxx",
  subjectToken: parent.access_token,
  scope: "calendar:read",
  dpopProver: prover,
});
```

## See also

- [Dynamic client registration](./dcr.md) — register clients programmatically
- [Device flow](./device-flow.md) — RFC 8628
- [Delegation and agents](./delegation-and-agents.md) — flagship use case
