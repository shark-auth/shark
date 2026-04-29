# MCP Server Auth in 5 Minutes

Add token-gated auth to any MCP server using SharkAuth as the OAuth 2.1 AS. This guide covers the full cold-start path: DCR â†’ device flow â†’ scope-protected resource. No pre-shared secrets required.

**Audience:** MCP server authors who want standards-based auth without running a cloud service.

---

## What you need

- Go 1.22+ (to build shark) or a prebuilt binary
- Python 3.10+ for the example MCP server
- `curl`, `jq`

Estimated time: 5 minutes.

---

## 1. Install shark

### From source

```bash
git clone https://github.com/shark-auth/shark
cd sharkauth
go build -o bin/shark ./cmd/shark
```

### Prebuilt (GitHub Releases)

```bash
# Linux / macOS
curl -Lo shark https://github.com/shark-auth/shark/releases/latest/download/shark_$(uname -s)_$(uname -m)
chmod +x shark
sudo mv shark /usr/local/bin/shark
```

Verify:

```bash
shark version
```

---

## 2. Start the AS in dev mode

```bash
shark serve --dev
```

`--dev` skips config entirely: ephemeral SQLite in `dev.db`, permissive CORS, built-in email inbox at `/admin/inbox`. No setup wizard needed.

Expected output:

```
2026-04-29T10:00:00Z INF SharkAuth starting  addr=:8080 dev_mode=true
2026-04-29T10:00:00Z INF admin dashboard     url=http://localhost:8080/admin
2026-04-29T10:00:00Z INF health check        url=http://localhost:8080/healthz
2026-04-29T10:00:00Z INF default app created client_id=shark_app_abc123
```

The AS is now live at `http://localhost:8080`. Discovery metadata:

```bash
curl -s http://localhost:8080/.well-known/oauth-authorization-server | jq '{issuer,registration_endpoint,token_endpoint,dpop_signing_alg_values_supported}'
```

```json
{
  "issuer": "http://localhost:8080",
  "registration_endpoint": "http://localhost:8080/oauth/register",
  "token_endpoint": "http://localhost:8080/oauth/token",
  "dpop_signing_alg_values_supported": ["ES256"]
}
```

---

## 3. Register your MCP server as an OAuth application (DCR)

RFC 7591 dynamic client registration â€” no admin credentials needed at registration time.

```bash
curl -sS -X POST http://localhost:8080/oauth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "client_name": "my-mcp-server",
    "grant_types": ["urn:ietf:params:oauth:grant-type:device_code", "client_credentials"],
    "token_endpoint_auth_method": "client_secret_basic",
    "scope": "mcp:read mcp:write"
  }' | jq .
```

Response:

```json
{
  "client_id": "shark_dcr_abc123",
  "client_secret": "cs_live_...",
  "registration_access_token": "rat_...",
  "registration_client_uri": "http://localhost:8080/oauth/register/shark_dcr_abc123",
  "grant_types": ["urn:ietf:params:oauth:grant-type:device_code", "client_credentials"],
  "client_name": "my-mcp-server"
}
```

Save the credentials:

```bash
export MCP_CLIENT_ID="shark_dcr_abc123"
export MCP_CLIENT_SECRET="cs_live_..."
```

---

## 4. Write a minimal protected MCP server

This is the part your MCP server needs to implement. Two requirements from RFC 9728:

1. `GET /.well-known/oauth-protected-resource` â€” return AS pointer
2. On unauthenticated requests â€” return `401` + `WWW-Authenticate: Bearer resource_metadata=<url>`

```python
#!/usr/bin/env python3
# mcp_server.py â€” minimal protected MCP server
import json, os
from http.server import BaseHTTPRequestHandler, HTTPServer
import urllib.request, urllib.error

SHARK_URL = os.environ.get("SHARK_URL", "http://localhost:8080")
RESOURCE   = os.environ.get("MCP_RESOURCE", "mcp://localhost:9000/my-server")
PORT       = int(os.environ.get("PORT", "9000"))
JWKS_URL   = f"{SHARK_URL}/oauth/jwks"

RESOURCE_METADATA = {
    "resource": RESOURCE,
    "authorization_servers": [SHARK_URL],
    "scopes_supported": ["mcp:read", "mcp:write"],
}

def verify_token(token: str) -> bool:
    """Verify token via AS introspection endpoint."""
    req = urllib.request.Request(
        f"{SHARK_URL}/oauth/introspect",
        data=f"token={token}".encode(),
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "Authorization": f"Basic {__import__('base64').b64encode(f'{os.environ[\"MCP_CLIENT_ID\"]}:{os.environ[\"MCP_CLIENT_SECRET\"]}'.encode()).decode()}",
        },
        method="POST",
    )
    try:
        resp = json.loads(urllib.request.urlopen(req).read())
        return bool(resp.get("active"))
    except Exception:
        return False

class MCPHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        print(f"[mcp] {fmt % args}")

    def send_json(self, code, body):
        data = json.dumps(body).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path == "/.well-known/oauth-protected-resource":
            self.send_json(200, RESOURCE_METADATA)
            return

        # Check Authorization header
        auth = self.headers.get("Authorization", "")
        if not auth.startswith("Bearer "):
            metadata_url = f"http://localhost:{PORT}/.well-known/oauth-protected-resource"
            self.send_response(401)
            self.send_header("WWW-Authenticate", f'Bearer resource_metadata="{metadata_url}"')
            self.send_header("Content-Length", "0")
            self.end_headers()
            return

        token = auth.removeprefix("Bearer ")
        if not verify_token(token):
            self.send_json(401, {"error": "invalid_token"})
            return

        # Dispatch to your MCP tools here
        if self.path == "/tools/hello":
            self.send_json(200, {"result": "Hello from protected MCP tool!"})
        else:
            self.send_json(404, {"error": "not_found"})

if __name__ == "__main__":
    print(f"[mcp] listening on :{PORT}")
    print(f"[mcp] resource: {RESOURCE}")
    print(f"[mcp] AS: {SHARK_URL}")
    HTTPServer(("", PORT), MCPHandler).serve_forever()
```

Run it:

```bash
SHARK_URL=http://localhost:8080 \
MCP_CLIENT_ID=$MCP_CLIENT_ID \
MCP_CLIENT_SECRET=$MCP_CLIENT_SECRET \
MCP_RESOURCE=mcp://localhost:9000/my-server \
python3 mcp_server.py
```

Confirm discovery:

```bash
curl -s http://localhost:9000/.well-known/oauth-protected-resource | jq .
```

```json
{
  "resource": "mcp://localhost:9000/my-server",
  "authorization_servers": ["http://localhost:8080"],
  "scopes_supported": ["mcp:read", "mcp:write"]
}
```

Unauthenticated request returns `401` with the metadata hint:

```bash
curl -si http://localhost:9000/tools/hello | head -5
# HTTP/1.1 401 Unauthorized
# WWW-Authenticate: Bearer resource_metadata="http://localhost:9000/.well-known/oauth-protected-resource"
```

---

## 5. Agent connects via Device Authorization Flow

RFC 8628 device flow lets a headless agent get a user-approved token without a browser redirect.

### Step 1 â€” Request device code

```bash
curl -sS -X POST http://localhost:8080/oauth/device \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "client_id=${MCP_CLIENT_ID}&scope=mcp:read"
```

```json
{
  "device_code": "GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS",
  "user_code": "SHARK-ABCD",
  "verification_uri": "http://localhost:8080/oauth/device/verify",
  "verification_uri_complete": "http://localhost:8080/oauth/device/verify?user_code=SHARK-ABCD",
  "expires_in": 900,
  "interval": 5
}
```

### Step 2 â€” User approves

Display to the user (or log it):

```
Visit http://localhost:8080/oauth/device/verify
Enter code: SHARK-ABCD
```

In dev mode, visit the URL in a browser, enter the code, and click Approve.

### Step 3 â€” Poll for token

```bash
DEVICE_CODE="GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS"

while true; do
  RESP=$(curl -sS -X POST http://localhost:8080/oauth/token \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
    -d "device_code=${DEVICE_CODE}" \
    -d "client_id=${MCP_CLIENT_ID}")

  ERROR=$(echo "$RESP" | jq -r '.error // empty')
  if [ -z "$ERROR" ]; then
    echo "Token: $(echo $RESP | jq -r '.access_token')"
    break
  elif [ "$ERROR" = "authorization_pending" ]; then
    sleep 5
  elif [ "$ERROR" = "slow_down" ]; then
    sleep 10
  else
    echo "Error: $ERROR"
    break
  fi
done
```

After approval:

```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "mcp:read"
}
```

---

## 6. Call the protected resource

```bash
ACCESS_TOKEN="eyJ..."

curl -sS http://localhost:9000/tools/hello \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" | jq .
```

```json
{
  "result": "Hello from protected MCP tool!"
}
```

---

## 7. Scope-protected resource example

Expand the `do_GET` handler in your MCP server to check specific scopes from the introspection response:

```python
def get_token_claims(token: str) -> dict | None:
    req = urllib.request.Request(
        f"{SHARK_URL}/oauth/introspect",
        data=f"token={token}".encode(),
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "Authorization": f"Basic {__import__('base64').b64encode(f'{os.environ[\"MCP_CLIENT_ID\"]}:{os.environ[\"MCP_CLIENT_SECRET\"]}'.encode()).decode()}",
        },
        method="POST",
    )
    try:
        resp = json.loads(urllib.request.urlopen(req).read())
        return resp if resp.get("active") else None
    except Exception:
        return None

# In do_GET, after verifying the token:
claims = get_token_claims(token)
if claims is None:
    self.send_json(401, {"error": "invalid_token"})
    return

token_scopes = set((claims.get("scope") or "").split())

if self.path == "/tools/read-only":
    if "mcp:read" not in token_scopes:
        self.send_json(403, {"error": "insufficient_scope", "scope": "mcp:read"})
        return
    self.send_json(200, {"data": "read result"})

elif self.path == "/tools/write-op":
    if "mcp:write" not in token_scopes:
        self.send_json(403, {"error": "insufficient_scope", "scope": "mcp:write"})
        return
    self.send_json(200, {"data": "write result"})
```

---

## Next steps

- **DPoP binding** â€” pass `resource=mcp://localhost:9000/my-server` at token request and a `DPoP:` proof header for audience-bound, key-confirmed tokens. See [RFC 9449](https://www.rfc-editor.org/rfc/rfc9449).
- **Token exchange** â€” chain agents with downscoped tokens. See [agent-delegation.md](./agent-delegation.md).
- **Production config** â€” configure `server.base_url` via the dashboard (Settings) or `shark admin config`, switch email provider. See README.
- **Dashboard** â€” `http://localhost:8080/admin` â€” view registered clients, audit logs, active sessions.

## Troubleshooting

**`authorization_pending` never clears.** Open `http://localhost:8080/oauth/device/verify` in a browser, enter the `user_code`, and click Approve. Shark does not auto-approve in dev mode.

**`invalid_client` on device code request.** The `client_id` must be registered with `grant_types` containing `urn:ietf:params:oauth:grant-type:device_code`.

**`401` from introspection.** Pass `Authorization: Basic <base64(client_id:client_secret)>` â€” introspection requires a registered client to authenticate.

**MCP server not returning `WWW-Authenticate`.** RFC 9728 requires this on every unauthenticated request. MCP-aware clients rely on it for discovery.
