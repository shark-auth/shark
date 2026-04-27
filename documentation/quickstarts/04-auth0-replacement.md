# Auth0 Replacement — Self-host auth with agent support

SharkAuth is a 30 MB single-binary auth server. No containers, no SaaS dependency, no per-MAU pricing. It covers the human auth primitives Auth0 / Clerk / WorkOS provide, and extends them with agent-native OAuth 2.1 that none of them support natively.

## What ships today

### Human authentication

| Feature | SharkAuth | Auth0 |
|---|---|---|
| Email + password signup/login | ✓ | ✓ |
| Magic link sign-in | ✓ | ✓ (paid) |
| Password reset | ✓ | ✓ |
| Email verification | ✓ | ✓ |
| Session management | ✓ | ✓ |
| MFA (TOTP) | planned v0.2 | ✓ |
| Social login (Google, GitHub) | planned v0.2 | ✓ |

### OAuth provider

SharkAuth acts as an OAuth 2.1 authorization server for your applications:

- Authorization code flow (PKCE)
- Client credentials (M2M)
- Device authorization grant (RFC 8628) — for CLI tools and TV apps
- Token introspection (RFC 7662)
- Token revocation (RFC 7009)
- JWKS endpoint (`/.well-known/jwks.json`)
- OpenID Connect discovery (`/.well-known/openid-configuration`)

### Agent-native primitives (Auth0 cannot match)

| Feature | SharkAuth | Auth0 |
|---|---|---|
| DPoP (RFC 9449) — token bound to keypair | ✓ | ✗ |
| Token exchange (RFC 8693) — delegation chains | ✓ | ✗ |
| Five-layer revocation model | ✓ | ✗ (one layer) |
| Per-agent audit trail with `act` chain | ✓ | ✗ |
| Cascade revoke entire customer fleet in one call | ✓ | ✗ |

## Step 1 — Install and start

```bash
pip install shark-auth   # Python SDK
# Binary:
curl -sSL https://github.com/your-org/shark/releases/latest/download/shark-linux-amd64 -o shark
chmod +x shark && ./shark serve
```

**screenshot: first-boot dashboard showing human auth and agent sections**

## Step 2 — Human auth: signup, login, magic link

### Signup

```bash
POST /api/v1/auth/register
{
  "email": "user@example.com",
  "password": "...",
  "name": "Alice"
}
```

### Login

```bash
POST /api/v1/auth/login
{
  "email": "user@example.com",
  "password": "..."
}
# Returns: { "access_token": "...", "refresh_token": "...", "user": {...} }
```

### Magic link

```python
from shark_auth import MagicLinkClient

ml = MagicLinkClient(base_url="http://localhost:8080", token="sk_live_...")
ml.send(email="user@example.com", redirect_url="https://app.example.com/auth/callback")
```

The user clicks the link → SharkAuth validates the one-time token → redirects to your `redirect_url` with a session token.

See [`/documentation/api/auth.md`](/documentation/api/auth.md) for the full human auth API.

## Step 3 — Use SharkAuth as the OAuth provider for your apps

Register your frontend or mobile app as an Application:

```bash
shark app create --name "my-react-app" --redirect-uris "https://app.example.com/callback"
```

Your app initiates the authorization code flow (PKCE):

```
GET /oauth/authorize
  ?client_id=app_xxx
  &redirect_uri=https://app.example.com/callback
  &response_type=code
  &code_challenge=<PKCE challenge>
  &code_challenge_method=S256
  &scope=openid profile email
```

Exchange code for tokens:

```bash
POST /oauth/token
  grant_type=authorization_code
  code=<code>
  client_id=app_xxx
  code_verifier=<PKCE verifier>
  redirect_uri=https://app.example.com/callback
```

## Step 4 — Add agent identities alongside human users

Once your human auth is working, adding agent support is one SDK call per customer:

```python
from shark_auth import Client

admin = Client(base_url="http://localhost:8080", token="sk_live_...")

# After a human user authenticates, provision their agent
agent = admin.agents.register_agent(
    app_id="app_my_react_app",
    name=f"agent-{user_id}",
    scopes=["data:read", "data:write"],
)
```

This is the step Auth0 doesn't have. From here, see [01 — Customer Agents](./01-customer-agents.md) for the full agent provisioning flow.

## Step 5 — Admin: manage users via SDK

```python
from shark_auth import Client

admin = Client(base_url="http://localhost:8080", token="sk_live_...")

# List users
users = admin.users.list_users()

# Update a user
admin.users.update_user("usr_abc", name="Alice Smith", email_verified=True)

# Set tier (for paywalled features)
admin.users.set_user_tier("usr_abc", tier="pro")

# Delete a user
admin.users.delete_user("usr_abc")
```

## Migration from Auth0 (high level)

1. Export Auth0 users (Management API: `GET /api/v2/users`)
2. Import via SharkAuth admin API: `POST /api/v1/admin/users` per user
3. Update your apps' token endpoint to point at SharkAuth
4. Update JWKS URL in your resource servers to `http://your-shark:8080/.well-known/jwks.json`
5. Update logout endpoints

Detailed migration guide with Auth0 schema mapping is deferred to v0.2. The high-level path is straightforward — SharkAuth speaks the same OAuth 2.1 / OIDC wire protocol.

## No vendor lock-in

- All data in a single SQLite file (default) or Postgres (v0.2)
- Standard OAuth 2.1 / OIDC wire protocol — your apps don't need the SharkAuth SDK
- Self-host anywhere: bare VM, Docker, Fly.io, Railway, your own Kubernetes
- Binary is 30 MB, no runtime dependencies

## Next steps

- Full agent provisioning flow: [01 — Customer Agents](./01-customer-agents.md)
- Human auth API reference: [`/documentation/api/auth.md`](/documentation/api/auth.md)
- OAuth provider reference: [`/documentation/api/oauth.md`](/documentation/api/oauth.md)
- CLI reference: [`/documentation/cli/`](/documentation/cli/)
