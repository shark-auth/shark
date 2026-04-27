# SharkAuth API — OAuth 2.1 Endpoints

All OAuth endpoints live under `/oauth/*` and follow RFC 6749 / RFC 9449 semantics. Errors use the RFC 6749 §5.2 shape `{error, error_description, error_uri}`, **not** the admin envelope. Compatible with any standard OAuth 2.1 client library.

---

## Dynamic Client Registration

### `POST /oauth/register` — Register a client (RFC 7591)

**Auth**: None (open registration — protect with network policy in production)

**Request**:
```json
{
  "client_name": "my-billing-agent",
  "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
  "token_endpoint_auth_method": "client_secret_basic",
  "scope": "openid billing:read"
}
```

**Response `201 Created`**:
```json
{
  "client_id": "shark_dcr_abc123",
  "client_secret": "cs_live_...",
  "client_name": "my-billing-agent",
  "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
  "token_endpoint_auth_method": "client_secret_basic",
  "scope": "openid billing:read",
  "registration_access_token": "rat_...",
  "registration_client_uri": "http://localhost:8080/oauth/register/shark_dcr_abc123"
}
```

**Error `400`**:
```json
{ "error": "invalid_client_metadata", "error_description": "redirect_uris required for authorization_code grant" }
```

---

### `GET /oauth/register/{client_id}` — Read client (RFC 7592)

**Auth**: `Authorization: Bearer <registration_access_token>`

**Response `200`**: Client metadata object (same shape as register response, without `client_secret`).

---

### `PUT /oauth/register/{client_id}` — Update client (RFC 7592)

**Auth**: `Authorization: Bearer <registration_access_token>`

**Request**: Full client metadata object (replacement, not partial patch).

**Response `200`**: Updated client metadata.

---

### `DELETE /oauth/register/{client_id}` — Delete client (RFC 7592)

**Auth**: `Authorization: Bearer <registration_access_token>`

**Response `204 No Content`**

---

### `POST /oauth/register/{client_id}/secret` — Rotate client secret

**Auth**: `Authorization: Bearer <registration_access_token>`

**Response `200`**:
```json
{ "client_id": "shark_dcr_abc123", "client_secret": "cs_live_newvalue..." }
```

---

### `DELETE /oauth/register/{client_id}/registration-token` — Rotate registration access token

**Auth**: `Authorization: Bearer <registration_access_token>`

**Response `200`**:
```json
{ "registration_access_token": "rat_newvalue..." }
```

---

## Authorization Code Flow

### `GET /oauth/authorize` — Authorization endpoint

**Auth**: Optional session cookie (user must be logged in to approve)

**Query parameters**:

| Parameter | Required | Description |
|---|---|---|
| `response_type` | Yes | Must be `code` |
| `client_id` | Yes | Registered client ID |
| `redirect_uri` | Yes | Must match registered redirect URIs |
| `scope` | Yes | Space-separated scopes (e.g. `openid profile email`) |
| `state` | Recommended | CSRF protection value |
| `code_challenge` | Yes (PKCE) | Base64url-encoded SHA-256 of verifier |
| `code_challenge_method` | Yes (PKCE) | Must be `S256` |

**Response**: Redirects to consent page or `redirect_uri` with `code=...&state=...`.

**Error**: Redirects to `redirect_uri` with `error=access_denied&error_description=...`.

---

### `POST /oauth/authorize` — Submit consent decision

**Auth**: Optional session cookie

**Body** (`application/x-www-form-urlencoded`): Consent form fields set by the hosted consent page.

**Response**: Redirects to `redirect_uri?code=<auth-code>&state=<state>`.

---

## Token Endpoint

### `POST /oauth/token` — Issue tokens (all grants)

**Auth**: Client credentials via `Authorization: Basic base64(client_id:client_secret)` or `client_id`/`client_secret` in body.

**Content-Type**: `application/x-www-form-urlencoded`

#### Grant: `authorization_code`

**Request**:
```bash
curl -X POST http://localhost:8080/oauth/token \
  -u "shark_dcr_abc123:cs_live_..." \
  -d "grant_type=authorization_code" \
  -d "code=<auth-code>" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "code_verifier=<pkce-verifier>"
```

**Response `200`**:
```json
{
  "access_token": "eyJhbGciOiJFUzI1NiIsImtpZCI6Imt...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "rt_...",
  "scope": "openid profile email"
}
```

#### Grant: `refresh_token`

```bash
curl -X POST http://localhost:8080/oauth/token \
  -u "shark_dcr_abc123:cs_live_..." \
  -d "grant_type=refresh_token" \
  -d "refresh_token=rt_..."
```

#### Grant: `client_credentials`

```bash
curl -X POST http://localhost:8080/oauth/token \
  -u "shark_dcr_abc123:cs_live_..." \
  -d "grant_type=client_credentials" \
  -d "scope=billing:read"
```

**Response `200`**:
```json
{
  "access_token": "eyJhbGciOiJFUzI1NiIsImtpZCI6Imt...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "billing:read"
}
```

#### Grant: `urn:ietf:params:oauth:grant-type:token-exchange` (RFC 8693)

Used by agents to delegate on behalf of a user or to narrow scope.

```bash
curl -X POST http://localhost:8080/oauth/token \
  -u "shark_dcr_abc123:cs_live_..." \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=<user-access-token>" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=openid billing:read"
```

**Response `200`**: Same shape as `client_credentials` response. The resulting token carries the user subject and agent delegation chain in the `act` claim.

#### DPoP-bound tokens (RFC 9449)

Include a `DPoP` proof header on the token request:

```bash
curl -X POST http://localhost:8080/oauth/token \
  -H "DPoP: <dpop-proof-jwt>" \
  -u "shark_dcr_abc123:cs_live_..." \
  -d "grant_type=client_credentials" \
  -d "scope=billing:read"
```

The issued access token will contain a `cnf.jkt` claim (public key thumbprint). Subsequent resource requests must include a fresh DPoP proof matching the key.

**Error `400`**:
```json
{ "error": "invalid_grant", "error_description": "The provided authorization grant is invalid." }
```

---

## Token Revocation

### `POST /oauth/revoke` — Revoke a token (RFC 7009)

**Auth**: Client credentials

**Request** (`application/x-www-form-urlencoded`):
```
token=<access-or-refresh-token>&token_type_hint=access_token
```

**Response `200`**: Empty body. Always returns 200 even if token was already revoked or unknown (per RFC 7009 §2.2).

---

## Token Introspection

### `POST /oauth/introspect` — Introspect a token (RFC 7662)

**Auth**: Client credentials

**Request** (`application/x-www-form-urlencoded`):
```
token=<access-token>&token_type_hint=access_token
```

**Response `200` — active token**:
```json
{
  "active": true,
  "sub": "usr_01HZ2XKABCDEF",
  "client_id": "shark_dcr_abc123",
  "scope": "openid billing:read",
  "exp": 1714000000,
  "iat": 1713996400,
  "token_type": "Bearer"
}
```

**Response `200` — revoked/expired token**:
```json
{ "active": false }
```

> **Notable behavior**: Revoked tokens return `active: false` — the response body never reveals _why_ (revoked vs. expired) to avoid information leakage.

---

## UserInfo

### `GET /oauth/userinfo` — UserInfo endpoint (OIDC Core §5.3)

**Auth**: `Authorization: Bearer <access-token>` with `openid` scope

**Response `200`**:
```json
{
  "sub": "usr_01HZ2XKABCDEF",
  "email": "alice@example.com",
  "email_verified": true,
  "name": "Alice Example"
}
```

---

## JWKS

### `GET /.well-known/jwks.json` — JSON Web Key Set (RFC 7517)

**Auth**: None (public)

**Response `200`** (`Cache-Control: public, max-age=300`):
```json
{
  "keys": [
    {
      "kty": "EC",
      "use": "sig",
      "alg": "ES256",
      "kid": "2024-01-15T00:00:00Z",
      "crv": "P-256",
      "x": "...",
      "y": "..."
    }
  ]
}
```

Returns active and recently-retired signing keys (within 2× access token TTL window) so in-flight tokens remain verifiable during key rotation.

---

## Discovery

### `GET /.well-known/oauth-authorization-server` — RFC 8414 Authorization Server Metadata

**Auth**: None (public)

**Response `200`**:
```json
{
  "issuer": "https://auth.yourdomain.com",
  "authorization_endpoint": "https://auth.yourdomain.com/oauth/authorize",
  "token_endpoint": "https://auth.yourdomain.com/oauth/token",
  "jwks_uri": "https://auth.yourdomain.com/.well-known/jwks.json",
  "registration_endpoint": "https://auth.yourdomain.com/oauth/register",
  "introspection_endpoint": "https://auth.yourdomain.com/oauth/introspect",
  "revocation_endpoint": "https://auth.yourdomain.com/oauth/revoke",
  "grant_types_supported": ["authorization_code", "client_credentials", "refresh_token", "urn:ietf:params:oauth:grant-type:token-exchange"],
  "response_types_supported": ["code"],
  "code_challenge_methods_supported": ["S256"],
  "dpop_signing_alg_values_supported": ["ES256", "RS256"]
}
```

---

## Device Authorization

### `POST /oauth/device` — Device Authorization Request (RFC 8628)

**Auth**: Client credentials

**Response `200`**:
```json
{
  "device_code": "GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS",
  "user_code": "WDJB-MJHT",
  "verification_uri": "https://auth.yourdomain.com/oauth/device/verify",
  "verification_uri_complete": "https://auth.yourdomain.com/oauth/device/verify?user_code=WDJB-MJHT",
  "expires_in": 1800,
  "interval": 5
}
```

### `GET /oauth/device/verify` — Display verification page (RFC 8628)

**Auth**: Optional session cookie

### `POST /oauth/device/verify` — Submit device approval (RFC 8628)

**Auth**: Optional session cookie

---

> **Note on device flow in v0.1**: The device authorization grant endpoints are registered and functional. However, the default scope policy and consent UI for headless flows are experimental. Monitor `GET /api/v1/admin/oauth/device-codes` to triage pending device flows from the admin dashboard.
