# SharkAuth API — Webhooks

SharkAuth webhooks deliver real-time event notifications to your HTTP endpoint when platform events occur (user created, session revoked, MFA enabled, etc.). The signature scheme is Stripe-compatible, making it easy to reuse existing webhook verification middleware.

---

## Subscription Model

A webhook subscription binds a **URL**, one or more **event filters**, and a **signing secret**.

**Create a subscription** (`POST /api/v1/webhooks`):

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer sk_live_..." \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://app.example.com/webhooks/sharkauth",
    "events": ["user.created", "session.revoked"],
    "description": "Production event sink"
  }'
```

**Response `201 Created`** (secret shown **once** at create time — store it securely):

```json
{
  "id": "wh_abc123",
  "url": "https://app.example.com/webhooks/sharkauth",
  "events": ["user.created", "session.revoked"],
  "enabled": true,
  "description": "Production event sink",
  "secret": "whsec_a1b2c3d4e5f6...",
  "created_at": "2024-04-01T10:00:00Z",
  "updated_at": "2024-04-01T10:00:00Z"
}
```

After creation, `GET/PATCH /api/v1/webhooks/{id}` responses **never** include the secret. If the secret is lost, delete the subscription and create a new one.

---

## Available Events

Retrieve the canonical list at runtime:

```bash
curl http://localhost:8080/api/v1/webhooks/events \
  -H "Authorization: Bearer sk_live_..."
```

**Response**:
```json
{
  "events": [
    "mfa.enabled",
    "org.created",
    "org.deleted",
    "org.member.added",
    "session.created",
    "session.revoked",
    "system.audit_log",
    "user.created",
    "user.deleted",
    "user.updated",
    "webhook.test"
  ]
}
```

| Event | Fires When |
|---|---|
| `user.created` | A new user account is created (signup, admin create, SSO first login) |
| `user.updated` | User profile or metadata is updated |
| `user.deleted` | User account is deleted |
| `session.created` | A new user session is established |
| `session.revoked` | A session is revoked (logout, admin revoke, expiry cascade) |
| `mfa.enabled` | A user enables TOTP MFA |
| `org.created` | A new organization is created |
| `org.deleted` | An organization is deleted |
| `org.member.added` | A member joins an organization |
| `system.audit_log` | An audit log event is emitted (fan-out for audit integrations) |
| `webhook.test` | Synthetic test event fired from `POST /api/v1/webhooks/{id}/test` |

---

## Delivery Payload

Every delivery is a POST to your endpoint with `Content-Type: application/json`:

```json
{
  "event": "user.created",
  "created_at": "2024-04-01T10:05:00Z",
  "data": {
    "id": "usr_01HZ2XKABCDEF",
    "email": "alice@example.com",
    "created_at": "2024-04-01T10:05:00Z"
  }
}
```

---

## HMAC Signature Verification

Every delivery includes an `X-Shark-Signature` header:

```
X-Shark-Signature: t=1712000100,v1=3d6e...
```

The signature is **Stripe-compatible**: `HMAC-SHA256(secret, "<timestamp>.<raw-body>")`.

### Verification Algorithm

1. Split the header on `,` to get `t=<timestamp>` and `v1=<hex-signature>`.
2. Extract the Unix timestamp from `t=`.
3. Reject if `|now - timestamp| > 300` seconds (replay protection).
4. Compute `HMAC-SHA256(secret, "<timestamp>.<raw-body>")` where `raw-body` is the exact request body bytes.
5. Compare your computed digest to the `v1=` value using a constant-time comparison.

### Python Example

```python
import hashlib
import hmac
import time

def verify_shark_webhook(payload: bytes, header: str, secret: str, tolerance: int = 300) -> bool:
    """Verify a SharkAuth webhook signature."""
    parts = {k: v for k, v in (item.split("=", 1) for item in header.split(","))}
    timestamp = parts.get("t")
    signature = parts.get("v1")
    if not timestamp or not signature:
        return False

    # Replay protection
    if abs(time.time() - int(timestamp)) > tolerance:
        return False

    # Compute expected signature
    signed_payload = f"{timestamp}.".encode() + payload
    expected = hmac.new(secret.encode(), signed_payload, hashlib.sha256).hexdigest()

    # Constant-time comparison
    return hmac.compare_digest(expected, signature)


# Usage (Flask example)
from flask import Flask, request, abort

app = Flask(__name__)
WEBHOOK_SECRET = "whsec_a1b2c3d4e5f6..."

@app.route("/webhooks/sharkauth", methods=["POST"])
def handle_webhook():
    payload = request.get_data()
    sig_header = request.headers.get("X-Shark-Signature", "")
    if not verify_shark_webhook(payload, sig_header, WEBHOOK_SECRET):
        abort(400, "Invalid signature")
    event = request.get_json()
    print(f"Received event: {event['event']}")
    return "", 200
```

### Node.js / TypeScript Example

```typescript
import crypto from "crypto";
import express from "express";

function verifySharkWebhook(
  payload: Buffer,
  header: string,
  secret: string,
  toleranceSeconds = 300
): boolean {
  const parts = Object.fromEntries(
    header.split(",").map((p) => p.split("=") as [string, string])
  );
  const { t: timestamp, v1: signature } = parts;
  if (!timestamp || !signature) return false;

  // Replay protection
  if (Math.abs(Date.now() / 1000 - Number(timestamp)) > toleranceSeconds) {
    return false;
  }

  // Compute expected signature
  const signedPayload = Buffer.concat([
    Buffer.from(`${timestamp}.`),
    payload,
  ]);
  const expected = crypto
    .createHmac("sha256", secret)
    .update(signedPayload)
    .digest("hex");

  // Constant-time comparison
  return crypto.timingSafeEqual(
    Buffer.from(expected, "hex"),
    Buffer.from(signature, "hex")
  );
}

// Usage (Express example)
const app = express();
const WEBHOOK_SECRET = "whsec_a1b2c3d4e5f6...";

app.post(
  "/webhooks/sharkauth",
  express.raw({ type: "application/json" }),
  (req, res) => {
    const sigHeader = req.headers["x-shark-signature"] as string;
    if (!verifySharkWebhook(req.body, sigHeader, WEBHOOK_SECRET)) {
      return res.status(400).send("Invalid signature");
    }
    const event = JSON.parse(req.body.toString());
    console.log("Received event:", event.event);
    res.sendStatus(200);
  }
);
```

> **Important**: Use `express.raw()` (or equivalent) — not `express.json()` — to preserve the raw body bytes for signature verification. Parsing JSON before verification will cause the signature check to fail if the serialization differs.

---

## Retry Behavior

- SharkAuth considers a delivery **successful** when your endpoint responds with any `2xx` status code within the configured timeout.
- Failed deliveries (non-2xx or timeout) are **not** automatically retried by the dispatcher in the current release (v0.9). Use the replay endpoint for manual retries.
- The delivery log retains all delivery attempts for inspection.

---

## Delivery Log

Retrieve the delivery history for a webhook subscription:

```bash
curl "http://localhost:8080/api/v1/webhooks/wh_abc123/deliveries?limit=50" \
  -H "Authorization: Bearer sk_live_..."
```

**Response** (keyset-paginated):
```json
{
  "data": [
    {
      "id": "del_01HZ2XKABCDEF",
      "webhook_id": "wh_abc123",
      "event": "user.created",
      "payload": "{\"event\":\"user.created\",...}",
      "status_code": 200,
      "success": true,
      "created_at": "2024-04-01T10:05:01Z"
    }
  ],
  "next_cursor": "2024-04-01T10:05:01Z|del_01HZ2XKABCDEF"
}
```

**Pagination**: Pass `cursor=<next_cursor>` on subsequent requests. `limit` defaults to 50, max 200.

---

## Replay a Delivery

Re-enqueue a past delivery using its original payload:

```bash
curl -X POST \
  "http://localhost:8080/api/v1/webhooks/wh_abc123/deliveries/del_01HZ2XKABCDEF/replay" \
  -H "Authorization: Bearer sk_live_..."
```

**Response `202 Accepted`**:
```json
{
  "message": "Delivery replay enqueued",
  "new_delivery_id": "del_02HZ2XKABCDEF",
  "event": "user.created"
}
```

The replay is byte-faithful to the original payload. The `webhook_id` in the URL must match the delivery's `webhook_id` — cross-webhook replay is rejected with `404`.

---

## Test Endpoint

Fire a synthetic event to verify your endpoint's network reachability and signature handling:

```bash
curl -X POST http://localhost:8080/api/v1/webhooks/wh_abc123/test \
  -H "Authorization: Bearer sk_live_..." \
  -H "Content-Type: application/json" \
  -d '{ "event_type": "user.created" }'
```

If `event_type` is omitted, the default `webhook.test` event is sent.

**Response `202 Accepted`**:
```json
{
  "message": "Test event enqueued",
  "delivery_id": "del_03HZ2XKABCDEF",
  "event": "user.created"
}
```

---

## Update a Subscription

```bash
curl -X PATCH http://localhost:8080/api/v1/webhooks/wh_abc123 \
  -H "Authorization: Bearer sk_live_..." \
  -H "Content-Type: application/json" \
  -d '{
    "events": ["user.created", "user.updated", "session.revoked"],
    "enabled": true
  }'
```

All fields are optional (partial patch). Setting `"enabled": false` pauses delivery without deleting the subscription.
