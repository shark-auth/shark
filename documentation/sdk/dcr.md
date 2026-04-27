# Dynamic client registration

RFC 7591 (register) + RFC 7592 (configuration management). Wraps `/oauth/register*`.

Lets clients register themselves without an admin pre-creating an entry. Each registration returns a `client_id`, optional `client_secret`, and a `registration_access_token` used to authenticate subsequent management requests.

## Construct

```python
from shark_auth import DCRClient
dcr = DCRClient("https://auth.example.com")
```

```typescript
import { DcrClient } from "@sharkauth/sdk";
const dcr = new DcrClient({ baseUrl: "https://auth.example.com" });
```

## Register

```python
reg = dcr.register(
    client_name="My App",
    redirect_uris=["https://app.example.com/cb"],
    grant_types=["authorization_code", "refresh_token"],
    response_types=["code"],
    scope="openid profile",
    token_endpoint_auth_method="client_secret_basic",
)
client_id = reg["client_id"]
secret    = reg.get("client_secret")
rat       = reg["registration_access_token"]
```

```typescript
const reg = await dcr.register({
  clientName: "My App",
  redirectUris: ["https://app.example.com/cb"],
  grantTypes: ["authorization_code", "refresh_token"],
  responseTypes: ["code"],
  scope: "openid profile",
  tokenEndpointAuthMethod: "client_secret_basic",
});
```

| Field                         | Type     | Required |
| ----------------------------- | -------- | -------- |
| `client_name`                 | string   | yes      |
| `redirect_uris`               | string[] | for code grants |
| `grant_types`                 | string[] | no, server defaults |
| `response_types`              | string[] | no       |
| `scope`                       | string   | no       |
| `token_endpoint_auth_method`  | string   | no, defaults to `client_secret_basic` |

Response includes everything sent plus `client_id`, `client_secret` (confidential clients only), `registration_access_token`, `registration_client_uri`.

## Read / update / delete

All of these require the `registration_access_token` returned at registration.

```python
info = dcr.get(client_id, registration_access_token=rat)
updated = dcr.update(client_id, registration_access_token=rat, redirect_uris=[...])
dcr.delete(client_id, registration_access_token=rat)
```

```typescript
const info = await dcr.get(clientId, rat);
const updated = await dcr.update(clientId, rat, { redirectUris: [...] });
await dcr.delete(clientId, rat);
```

## Rotate secret

```python
new_creds = dcr.rotate_secret(client_id, registration_access_token=rat)
print(new_creds["client_secret"])
```

```typescript
const newCreds = await dcr.rotateSecret(clientId, rat);
```

Backend route: `POST /oauth/register/{id}/secret` (NOT `/rotate-secret`).

## Rotate registration access token

```python
new_rat = dcr.rotate_registration_token(client_id, registration_access_token=rat)
```

```typescript
const newRat = await dcr.rotateRegistrationToken(clientId, rat);
```

Backend route: `DELETE /oauth/register/{id}/registration-token` (yes, DELETE — the SDK matches the wired verb). The previous RAT is invalidated; store the new one.

## See also

- [OAuth clients](./oauth-clients.md) — using a registered client
- [Delegation and agents](./delegation-and-agents.md) — registering machine principals via `AgentsClient` instead
