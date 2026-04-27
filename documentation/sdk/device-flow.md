# Device flow

RFC 8628 device authorization grant. Use when an agent has no browser and no inbound HTTP listener (CLI tools, CI runners, IoT).

The default backend route is `/oauth/device` (not the spec's `/oauth/device_authorization`). The SDK accounts for this — pass a custom path only if you've overridden the server config.

## Run a flow

```python
from shark_auth import DeviceFlow

flow = DeviceFlow(
    auth_url="https://auth.example.com",
    client_id="shark_agent_xxx",
    scope="calendar:read",
)
init = flow.begin()
print("Visit:", init.verification_uri)
print("Code:", init.user_code)
# Or print verification_uri_complete which embeds the code.
token = flow.wait_for_approval(timeout_s=300)
print(token.access_token)
```

```typescript
import { DeviceFlow } from "@sharkauth/sdk";

const flow = new DeviceFlow({
  authUrl: "https://auth.example.com",
  clientId: "shark_agent_xxx",
  scope: "calendar:read",
});
const init = await flow.begin();
console.log("Visit:", init.verificationUri, "code:", init.userCode);
const token = await flow.waitForApproval({ timeoutMs: 300_000 });
```

`begin()` returns `{device_code, user_code, verification_uri, verification_uri_complete?, expires_in, interval}`.

`wait_for_approval()` polls `/oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:device_code`, respecting the server's `interval`, and returns the `TokenResponse` once the user approves.

## DPoP-bound device flow

Pass a `DPoPProver` and the resulting token is sender-constrained.

```python
from shark_auth import DPoPProver, DeviceFlow

prover = DPoPProver.generate()
flow = DeviceFlow(
    auth_url="https://auth.example.com",
    client_id="shark_agent_xxx",
    scope="calendar:read",
    dpop_prover=prover,
)
init = flow.begin()
token = flow.wait_for_approval(timeout_s=300)
# token bound to prover.jkt
```

```typescript
const prover = await DPoPProver.generate();
const flow = new DeviceFlow({
  authUrl: "https://auth.example.com",
  clientId: "shark_agent_xxx",
  dpopProver: prover,
});
```

## Errors

`DeviceFlowError` (Python) / `DeviceFlowError` (TS) on:

- `expired_token` — user did not approve in time
- `access_denied` — user explicitly denied
- `slow_down` — surfaced when poll interval is too tight (the SDK auto-backs off)

## See also

- [OAuth clients](./oauth-clients.md)
- [DPoP](./dpop.md)
