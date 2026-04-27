# Webhooks

Subscribe to server events, verify signatures on incoming deliveries, replay old deliveries. Wraps `/api/v1/admin/webhooks/*`.

## Construct

```python
from shark_auth import WebhooksClient
wh = WebhooksClient("https://auth.example.com", admin_api_key="sk_live_admin")
```

```typescript
import { WebhooksClient } from "@sharkauth/sdk";
const wh = new WebhooksClient({ baseUrl: "https://auth.example.com", adminKey: "sk_live_admin" });
```

## Register

```python
created = wh.register(
    url="https://app.example.com/hooks",
    events=["user.created", "user.deleted", "agent.token_issued"],
    secret="whsec_xxxx",   # optional; server can auto-generate
)
print(created["id"])
```

```typescript
const created = await wh.register({
  url: "https://app.example.com/hooks",
  events: ["user.created", "user.deleted", "agent.token_issued"],
  secret: "whsec_xxxx",
});
```

`events` accepts globs: `user.*`, `agent.*`, `*` for everything.

## List / get / update / delete

```python
hooks = wh.list()
one   = wh.get("wh_abc")
wh.update("wh_abc", events=["user.created"], enabled=True)
wh.delete("wh_abc")
```

```typescript
const hooks = await wh.list();
const one   = await wh.get("wh_abc");
await wh.update("wh_abc", { events: ["user.created"], enabled: true });
await wh.delete("wh_abc");
```

Backend uses `PATCH /{id}` for update (NOT PUT) — the SDK matches.

## Test fire

```python
wh.send_test("wh_abc", event_type="user.created")
```

```typescript
await wh.sendTest("wh_abc", { eventType: "user.created" });
```

Backend route: `POST /api/v1/admin/webhooks/{id}/test` (NOT `/send-test`).

## Past deliveries / replay

```python
deliveries = wh.list_deliveries("wh_abc", limit=50)
wh.replay("wh_abc", delivery_id=deliveries[0]["id"])
```

```typescript
const deliveries = await wh.listDeliveries("wh_abc", { limit: 50 });
await wh.replay("wh_abc", deliveries[0].id);
```

## Verify incoming signatures

The signature header has two formats. The SDK helper handles both with constant-time comparison.

### Stripe-style: `t=<unix_ts>,v1=<hex_hmac>`

Signed payload is `f"{ts}.".encode() + raw_body`. The SDK enforces a 300-second tolerance window by default.

### Raw hex digest

`HMAC-SHA256(secret, raw_body)` as hex. Optionally prefixed with `sha256=`.

```python
from shark_auth import verify_signature

@app.post("/hooks")
def receive(req):
    raw = req.body  # bytes — DO NOT json.loads first
    sig = req.headers["Shark-Signature"]
    if not verify_signature(raw, sig, secret="whsec_xxxx"):
        return 401
    payload = json.loads(raw)
    # ... handle payload ...
```

```typescript
import { verifySignature } from "@sharkauth/sdk";

app.post("/hooks", async (req, res) => {
  const raw = await readRawBody(req);
  const sig = req.headers["shark-signature"];
  const ok = await verifySignature(raw, sig, "whsec_xxxx");
  if (!ok) return res.status(401).send("bad sig");
  const payload = JSON.parse(raw.toString());
  // ...
});
```

| Param                | Default | Notes                                          |
| -------------------- | ------- | ---------------------------------------------- |
| `payload`            | -       | Raw bytes — must match what the server signed  |
| `header_signature`   | -       | Verbatim header value                          |
| `secret`             | -       | Same value passed to `register(secret=...)`    |
| `tolerance_seconds`  | 300     | Replay window for the timestamped form         |

The TS implementation uses `globalThis.crypto.subtle` and works in browser + Node 18+ without dependencies.

## Common events

See [Audit logs](./audit-logs.md#common-event-types) for the canonical event-name list. Webhooks fire on the same event stream.

## See also

- [Audit logs](./audit-logs.md) — same event stream, query API instead of push
