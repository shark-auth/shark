# Shark Agent Auth — The Definitive Spec

**Date:** 2026-04-15
**Status:** MVP shipping (Waves A-E complete, 95% feature parity). Phase 6 next.
**Goal:** Make Shark the best agent auth system in OSS, and the only one that's MCP-native, single-binary, and SQLite-compatible.

---

## Why This Matters

Agents are proliferating faster than the auth primitives they need. Teams shipping AI agents today reach for patterns that predate the threat model: shared API keys, service accounts, or hand-rolled JWTs with whatever scope checks happen to survive the next refactor. None of those give you per-agent identity, consent, or an audit trail that can name which agent touched which resource.

Nobody in OSS has shipped a complete answer. The commercial options each have a gap:

- **Auth0** — agent auth is a 50% pricing add-on ($53+/month). Not OSS. Not self-hosted.
- **Okta Agent Gateway** — enterprise plan only, pricing undisclosed, not GA for most use cases.
- **Stytch** — strong MCP compliance, but agents are M2M clients, not first-class identities. No Token Vault.
- **Arcade** — excellent tool-calling and Token Vault story, but not a general-purpose auth server. No OAuth 2.1 AS.

Shark is the only self-hosted, single-binary, OSS auth system where agents are first-class OAuth clients with their own credentials, scopes, audit trail, and Token Vault — running on SQLite with zero external dependencies.

---

## Implementation Status (2026-04-20)

> **Want to try it?** See the [Hello Agent walkthrough](docs/hello-agent.md) — 15 minutes from clone to a working DPoP-bound agent token.

**Waves A–E complete.** All core agent-auth primitives ship in the single binary, tested and smoke-covered.

**Shipped RFCs:**
- RFC 6749 + 7636 — OAuth 2.1 + PKCE (authorization code + client credentials + refresh with rotation)
- RFC 8414 — OAuth Authorization Server Metadata discovery
- RFC 7591/7592 — Dynamic Client Registration + management
- RFC 7662 — Token Introspection
- RFC 7009 — Token Revocation
- RFC 8628 — Device Authorization Grant (headless agents)
- RFC 9449 — DPoP proof-of-possession (JWK validation, JTI replay cache, thumbprint verify, ath claim)
- RFC 8693 — Token Exchange with delegation chains (act claim) + scope narrowing
- RFC 8707 — Resource Indicators (audience-bound tokens)

**Token Vault:** encrypted-at-rest OAuth token storage for third-party APIs, provider templates (Google, Slack, GitHub), auto-refresh with 30s leeway, per-user per-provider connections, admin dashboard CRUD.

**Agent identity:** agents are first-class OAuth clients with their own credentials, scopes, and audit trail. Not user extensions. Agent CRUD via `/api/v1/agents` + dashboard.

**Not yet shipped (tracked for post-launch):**
- RFC 9396 Rich Authorization Requests — schema field exists, no request validation yet
- RFC 9126 Pushed Authorization Requests
- Step-up authorization

---

## Competitive Gap Analysis

| Capability | Auth0 | Okta | Stytch | Arcade | **Shark** |
|---|---|---|---|---|---|
| Agent as first-class identity | No (user extension) | Yes | No (M2M client) | No (tool-calling only) | **Yes** |
| MCP-native | No | Yes (Agent Gateway) | Yes (Connected Apps) | Yes (runtime) | **Yes** |
| OAuth 2.1 compliant | OAuth 2.0 + CIBA | Yes | Yes | Partial | **Yes** |
| Token Vault (managed OAuth for 3rd-party APIs) | Yes (30+ apps) | Yes | No | Yes (zero-exposure) | **Yes (v1)** |
| Human-in-the-loop consent | CIBA | TBD | OAuth consent UI | JIT approval gates | **Yes** |
| Agent-to-agent delegation | No | No | Published guide | No | **Yes (RFC 8693)** |
| Device flow (headless agents) | No | Unknown | No | No | **Yes (RFC 8628)** |
| DPoP (proof-of-possession) | No | Unknown | No | No | **Yes (RFC 9449)** |
| Per-tool permissions (RAR) | No | Dynamic policy | Scopes only | Per-tool scoping | **Partial (RFC 9396, schema ships, validation post-launch)** |
| Audit trail (agent-specific) | Basic | Yes | Per-token | Per-action | **Yes (unified)** |
| Self-hosted / OSS | No | No | No | No | **Yes** |
| Single binary | No | No | No | No | **Yes** |
| Pricing | 50% add-on ($53+/mo) | Enterprise (undisclosed) | Per-MAU | $25/mo + usage | **Free (self-hosted)** |

---

## Architecture Overview

Shark becomes a full OAuth 2.1 Authorization Server with agent-native extensions. All of this runs in the same binary, stores state in SQLite WAL, and requires zero external dependencies.

```
                    MCP Client (AI Agent)
                           |
                    [discovers via RFC 9728]
                           |
                    [registers via RFC 7591 / CIMD]
                           |
            +--------------+---------------+
            |              |               |
    Client Credentials  Auth Code+PKCE  Device Flow
    (autonomous agent)  (user delegates) (headless agent)
            |              |               |
            v              v               v
        +--------------------------------------+
        |     Shark OAuth 2.1 Token Endpoint   |
        |  - JWT access tokens (ES256 signed)  |
        |  - Refresh token rotation            |
        |  - DPoP proof validation             |
        |  - Resource indicator binding        |
        |  - Token exchange (delegation)       |
        +--------------------------------------+
                           |
                    [JWT with aud, scope,
                     act (delegation), cnf (DPoP)]
                           |
                    Resource Server / MCP Server
                           |
                    [validates via JWKS]
                    [introspects if needed]
```

---

## Data Model

### agents table
```sql
CREATE TABLE agents (
    id                  TEXT PRIMARY KEY,        -- agent_xxxx (nanoid)
    name                TEXT NOT NULL,
    description         TEXT DEFAULT '',
    client_id           TEXT UNIQUE NOT NULL,     -- public identifier
    client_secret_hash  TEXT,                     -- SHA-256, NULL for public clients
    client_type         TEXT NOT NULL DEFAULT 'confidential',  -- confidential | public
    auth_method         TEXT NOT NULL DEFAULT 'client_secret_basic',
                        -- client_secret_basic | client_secret_post | private_key_jwt | none
    jwks                TEXT,                     -- JSON: public keys for private_key_jwt
    jwks_uri            TEXT,                     -- URL to fetch JWKS (alternative to inline)
    redirect_uris       TEXT NOT NULL DEFAULT '[]',  -- JSON array
    grant_types         TEXT NOT NULL DEFAULT '["client_credentials"]',  -- JSON array
    response_types      TEXT NOT NULL DEFAULT '["code"]',  -- JSON array
    scopes              TEXT NOT NULL DEFAULT '[]',  -- JSON array of allowed scopes
    token_lifetime      INTEGER DEFAULT 900,      -- access token lifetime in seconds (default 15min)
    metadata            TEXT DEFAULT '{}',        -- arbitrary JSON (agent version, capabilities, etc.)
    logo_uri            TEXT,
    homepage_uri        TEXT,
    active              INTEGER NOT NULL DEFAULT 1,
    created_by          TEXT REFERENCES users(id),  -- human who registered this agent
    created_at          TIMESTAMP NOT NULL,
    updated_at          TIMESTAMP NOT NULL
);
```

### oauth_authorization_codes table
```sql
CREATE TABLE oauth_authorization_codes (
    code_hash           TEXT PRIMARY KEY,         -- SHA-256 of the code
    client_id           TEXT NOT NULL,
    user_id             TEXT NOT NULL REFERENCES users(id),
    redirect_uri        TEXT NOT NULL,
    scope               TEXT NOT NULL DEFAULT '',
    code_challenge      TEXT NOT NULL,            -- PKCE challenge
    code_challenge_method TEXT NOT NULL DEFAULT 'S256',
    resource            TEXT,                     -- RFC 8707 audience
    authorization_details TEXT,                   -- RFC 9396 RAR (JSON)
    nonce               TEXT,                     -- OIDC nonce
    expires_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP NOT NULL
);
```

### oauth_tokens table
```sql
CREATE TABLE oauth_tokens (
    id                  TEXT PRIMARY KEY,         -- token_xxxx
    jti                 TEXT UNIQUE NOT NULL,      -- JWT ID (for revocation tracking)
    client_id           TEXT NOT NULL,
    agent_id            TEXT REFERENCES agents(id),
    user_id             TEXT REFERENCES users(id), -- NULL for client_credentials
    token_type          TEXT NOT NULL,             -- access | refresh
    token_hash          TEXT UNIQUE NOT NULL,      -- SHA-256 (for refresh tokens)
    scope               TEXT NOT NULL DEFAULT '',
    audience            TEXT,                      -- RFC 8707 resource
    authorization_details TEXT,                    -- RFC 9396 RAR (JSON)
    dpop_jkt            TEXT,                      -- DPoP key thumbprint (RFC 9449)
    delegation_subject  TEXT,                      -- original subject in token exchange
    delegation_actor    TEXT,                      -- acting party in token exchange
    family_id           TEXT,                      -- refresh token family (for rotation/reuse detection)
    expires_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP NOT NULL,
    revoked_at          TIMESTAMP                  -- NULL = active, set = revoked
);
CREATE INDEX idx_oauth_tokens_family ON oauth_tokens(family_id);
CREATE INDEX idx_oauth_tokens_client ON oauth_tokens(client_id);
CREATE INDEX idx_oauth_tokens_jti ON oauth_tokens(jti);
```

### oauth_consents table
```sql
CREATE TABLE oauth_consents (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id),
    client_id           TEXT NOT NULL,
    scope               TEXT NOT NULL,
    authorization_details TEXT,                    -- RFC 9396 RAR (JSON)
    granted_at          TIMESTAMP NOT NULL,
    expires_at          TIMESTAMP,                 -- NULL = permanent until revoked
    revoked_at          TIMESTAMP
);
CREATE UNIQUE INDEX idx_oauth_consents_user_client ON oauth_consents(user_id, client_id) WHERE revoked_at IS NULL;
```

### oauth_device_codes table
```sql
CREATE TABLE oauth_device_codes (
    device_code_hash    TEXT PRIMARY KEY,          -- SHA-256
    user_code           TEXT UNIQUE NOT NULL,      -- 8-char human-readable (XXXX-XXXX)
    client_id           TEXT NOT NULL,
    scope               TEXT NOT NULL DEFAULT '',
    resource            TEXT,
    user_id             TEXT REFERENCES users(id), -- set when user approves
    status              TEXT NOT NULL DEFAULT 'pending', -- pending | approved | denied | expired
    last_polled_at      TIMESTAMP,                -- for slow_down enforcement
    poll_interval       INTEGER NOT NULL DEFAULT 5, -- seconds
    expires_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP NOT NULL
);
```

### oauth_dcr_clients table (Dynamic Client Registration)
```sql
CREATE TABLE oauth_dcr_clients (
    client_id           TEXT PRIMARY KEY,
    registration_token_hash TEXT UNIQUE NOT NULL,  -- SHA-256 of registration_access_token
    client_metadata     TEXT NOT NULL,             -- full JSON metadata from registration
    created_at          TIMESTAMP NOT NULL,
    expires_at          TIMESTAMP                  -- client_secret_expires_at
);
```

### signing_keys table
```sql
CREATE TABLE signing_keys (
    kid                 TEXT PRIMARY KEY,          -- key ID
    algorithm           TEXT NOT NULL,             -- ES256 | RS256 | EdDSA
    private_key_enc     BLOB NOT NULL,             -- AES-256-GCM encrypted private key
    public_key          TEXT NOT NULL,             -- PEM or JWK JSON
    active              INTEGER NOT NULL DEFAULT 1, -- 1 = use for signing, 0 = verify only (rotating out)
    created_at          TIMESTAMP NOT NULL,
    rotated_at          TIMESTAMP                  -- when this key stopped being used for signing
);
```

---

## API Endpoints

### Discovery (MCP-required)

| Endpoint | RFC | Description |
|----------|-----|-------------|
| `GET /.well-known/oauth-authorization-server` | RFC 8414 | Authorization server metadata (MCP clients discover this) |
| `GET /.well-known/oauth-protected-resource` | RFC 9728 | Protected resource metadata (if Shark hosts MCP servers) |
| `GET /.well-known/jwks.json` | - | Public signing keys for token verification |

### Core OAuth 2.1

| Endpoint | RFC | Description |
|----------|-----|-------------|
| `GET /oauth/authorize` | RFC 6749 | Authorization endpoint (renders consent UI) |
| `POST /oauth/authorize` | RFC 6749 | Authorization decision (user approves/denies) |
| `POST /oauth/token` | RFC 6749 | Token endpoint (all grant types) |
| `POST /oauth/revoke` | RFC 7009 | Token revocation |
| `POST /oauth/introspect` | RFC 7662 | Token introspection |
| `POST /oauth/register` | RFC 7591 | Dynamic client registration |
| `GET/PUT/DELETE /oauth/register/{client_id}` | RFC 7592 | Client registration management |
| `POST /oauth/device` | RFC 8628 | Device authorization (headless agents) |
| `GET /oauth/device/verify` | RFC 8628 | Device code verification page (user enters code) |
| `POST /oauth/par` | RFC 9126 | Pushed authorization requests |

### Agent Management

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/agents` | Register an agent (admin) |
| `GET /api/v1/agents` | List agents |
| `GET /api/v1/agents/{id}` | Get agent details |
| `PATCH /api/v1/agents/{id}` | Update agent |
| `DELETE /api/v1/agents/{id}` | Deactivate agent (revokes all tokens) |
| `GET /api/v1/agents/{id}/tokens` | List active tokens for agent |
| `POST /api/v1/agents/{id}/tokens/revoke-all` | Revoke all tokens for agent |
| `GET /api/v1/agents/{id}/audit` | Agent-specific audit trail |

### Consent Management

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/auth/consents` | List user's active agent consents |
| `DELETE /api/v1/auth/consents/{id}` | Revoke consent (revokes all associated tokens) |

---

## Grant Types — Full Specification

### 1. Client Credentials (autonomous agent)

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(client_id:client_secret)

grant_type=client_credentials
&scope=users:read audit:read
&resource=https://api.example.com
```

Response:
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "users:read audit:read"
}
```

No refresh token. Agent requests a new access token when it expires.

### 2. Authorization Code + PKCE (human delegates to agent)

Step 1 — Agent redirects user:
```
GET /oauth/authorize
  ?response_type=code
  &client_id=agent_client_id
  &redirect_uri=http://localhost:8080/callback
  &scope=files:read files:write
  &resource=https://api.example.com
  &code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
  &code_challenge_method=S256
  &state=random_state_value
```

Step 2 — User sees consent screen, approves.

Step 3 — Agent exchanges code for tokens:
```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=SplxlOBeZQQYbYS6WxSbIA
&redirect_uri=http://localhost:8080/callback
&client_id=agent_client_id
&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk
```

Response:
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "xRkY...",
  "scope": "files:read files:write"
}
```

### 3. Device Authorization (headless agent)

Step 1 — Agent requests device code:
```
POST /oauth/device
Content-Type: application/x-www-form-urlencoded

client_id=agent_client_id
&scope=deploy:create
```

Response:
```json
{
  "device_code": "GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS",
  "user_code": "SHARK-ABCD",
  "verification_uri": "https://auth.example.com/oauth/device/verify",
  "verification_uri_complete": "https://auth.example.com/oauth/device/verify?user_code=SHARK-ABCD",
  "expires_in": 900,
  "interval": 5
}
```

Step 2 — Agent displays to user: "Visit https://auth.example.com/oauth/device/verify and enter code SHARK-ABCD"

Step 3 — Agent polls token endpoint:
```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:device_code
&device_code=GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS
&client_id=agent_client_id
```

Responses during polling: `{"error": "authorization_pending"}` or `{"error": "slow_down"}`

After user approves: standard token response.

### 4. Token Exchange (agent-to-agent delegation)

Agent A has a token and wants Agent B to act on its behalf:

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(agent_b_client_id:agent_b_secret)

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token=eyJ...agent_a_token
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&actor_token=eyJ...agent_b_token
&actor_token_type=urn:ietf:params:oauth:token-type:access_token
&scope=deploy:create
&resource=https://api.example.com
```

Response — JWT with delegation chain:
```json
{
  "sub": "usr_alice",
  "client_id": "agent_b_client_id",
  "scope": "deploy:create",
  "act": {
    "sub": "agent_b",
    "act": {
      "sub": "agent_a"
    }
  }
}
```

The resource server sees: "Agent B is acting on behalf of Agent A, who was acting on behalf of Alice. The allowed scope is deploy:create only."

### 5. Refresh Token (rotation)

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=xRkY...
&client_id=agent_client_id
```

Response includes a NEW refresh token. The old one is immediately invalidated. If the old token is reused (replay attack), the entire token family is revoked.

---

## JWT Access Token Format

```json
{
  "header": {
    "typ": "at+jwt",
    "alg": "ES256",
    "kid": "key-2026-04"
  },
  "payload": {
    "iss": "https://auth.example.com",
    "sub": "usr_alice",
    "aud": "https://api.example.com",
    "client_id": "agent_client_id",
    "scope": "files:read files:write",
    "exp": 1713200400,
    "iat": 1713199500,
    "jti": "token_abc123",
    "agent_id": "agent_xxxx",
    "agent_name": "DeployBot",
    "act": {
      "sub": "agent_xxxx"
    },
    "authorization_details": [
      {
        "type": "file_access",
        "locations": ["https://storage.example.com/project-x"],
        "actions": ["read", "list"]
      }
    ],
    "cnf": {
      "jkt": "0ZcOCORZNYy-DWpqq30jZyJGHTN0d2HglBV3uiguA4I"
    }
  }
}
```

Key claims:
- `agent_id` + `agent_name` — Shark-specific, identifies which agent
- `act` — delegation chain (RFC 8693)
- `authorization_details` — fine-grained permissions (RFC 9396)
- `cnf.jkt` — DPoP key binding (RFC 9449)
- `aud` — audience restriction (RFC 8707)

---

## DPoP (Demonstration of Proof-of-Possession)

Prevents stolen tokens from being used by attackers. The agent generates a keypair and proves possession on every request.

**At token request:**
```
POST /oauth/token
DPoP: eyJ...dpop_proof_jwt
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&scope=read
```

DPoP proof contains: `htm` (HTTP method), `htu` (endpoint URL), `iat`, `jti`, and the public key in the header.

**At resource server:**
```
GET /api/data
Authorization: DPoP eyJ...access_token
DPoP: eyJ...dpop_proof_jwt_with_ath
```

The proof now includes `ath` (access token hash). Resource server verifies the proof signature matches the `cnf.jkt` in the access token.

**Implementation priority:** P1. Not required by MCP spec but dramatically improves security for agent tokens that may be long-lived or high-privilege.

---

## MCP Compatibility Checklist

To be a fully MCP-compatible authorization server:

- [x] OAuth 2.1 with Authorization Code + PKCE (S256 mandatory)
- [x] Client Credentials grant
- [x] Authorization Server Metadata at `/.well-known/oauth-authorization-server` (RFC 8414)
- [x] `code_challenge_methods_supported` includes `S256` in metadata
- [x] Resource Indicators (RFC 8707) — accept `resource` param, bind token audience
- [x] Dynamic Client Registration at `/oauth/register` (RFC 7591)
- [x] Client ID Metadata Documents (CIMD) — fetch and validate client metadata from HTTPS URLs
- [x] Step-up authorization — return `403` with `WWW-Authenticate: Bearer error="insufficient_scope", scope="needed_scopes"`
- [x] Bearer token validation per OAuth 2.1 Section 5.2
- [x] Refresh token rotation for public clients
- [x] HTTPS everywhere (except localhost in dev)
- [ ] Protected Resource Metadata (RFC 9728) — only if Shark also hosts MCP servers

---

## Security Model

### Token Lifetimes
| Token | Lifetime | Notes |
|-------|----------|-------|
| Access token | 5-15 min (configurable per agent) | Short = less damage if stolen |
| Refresh token | 7-14 days | Rotated on every use |
| Authorization code | 60 seconds | Single-use |
| Device code | 15 minutes | User must approve within this window |
| DPoP proof | Reject if iat > 60s ago | Prevents replay |
| PAR request_uri | 90 seconds | Short-lived reference |

### Rate Limits
| Endpoint | Limit | Notes |
|----------|-------|-------|
| `POST /oauth/token` | 50/min per client_id | Burst tolerance for client_credentials |
| `GET /oauth/authorize` | 10/min per IP | Prevent auth flooding |
| `POST /oauth/introspect` | 200/min per caller | High because called per API request |
| `POST /oauth/revoke` | 20/min per client_id | |
| `POST /oauth/register` | 5/min per IP | Prevent mass registration |
| `POST /oauth/device` | 10/min per client_id | |
| Device polling | Enforce `interval` param | Return `slow_down` on violation |

### Abuse Prevention
- Refresh token reuse detection: revoke entire token family on replay
- Failed auth attempts: exponential backoff (1s, 2s, 4s, 8s... cap 5min)
- JTI-based replay detection for DPoP proofs
- Client reputation: new/unverified agents get stricter rate limits
- Anomaly monitoring: sudden scope escalation, unexpected IP, high-volume token requests

### Agent-Specific Security
- Agents are first-class identities — not shoehorned into user or service account models
- Every agent action creates an audit log entry with `actor_type: "agent"` and `agent_id`
- Delegation chains are tracked: "Agent B acting on behalf of Agent A on behalf of User Alice"
- Consent is per-agent, per-scope — users can revoke individual agent access
- Token exchange enforces scope narrowing — delegated tokens cannot have MORE permissions than the original
- `may_act` claim prevents unauthorized delegation — only specified agents can receive delegated tokens

---

## Audit Trail (Agent-Aware)

Every OAuth operation creates an audit event:

```json
{
  "id": "aud_xxxx",
  "timestamp": "2026-04-15T10:30:00Z",
  "action": "oauth.token.issued",
  "actor_type": "agent",
  "actor_id": "agent_xxxx",
  "actor_name": "DeployBot",
  "user_id": "usr_alice",
  "grant_type": "authorization_code",
  "scopes": ["deploy:create"],
  "resource": "https://api.example.com",
  "delegation_chain": ["usr_alice", "agent_xxxx", "agent_yyyy"],
  "ip": "203.0.113.42",
  "user_agent": "SharkSDK/1.0"
}
```

Events tracked:
- `agent.registered` / `agent.updated` / `agent.deactivated`
- `oauth.authorize.started` / `oauth.authorize.approved` / `oauth.authorize.denied`
- `oauth.token.issued` / `oauth.token.refreshed` / `oauth.token.exchanged`
- `oauth.token.revoked` / `oauth.token.introspected`
- `oauth.consent.granted` / `oauth.consent.revoked`
- `oauth.device.requested` / `oauth.device.approved` / `oauth.device.denied`
- `oauth.dcr.registered` / `oauth.dcr.updated` / `oauth.dcr.deleted`

Filter audit logs by `actor_type=agent` to see all agent activity. Filter by `agent_id` for a specific agent's history. Filter by `delegation_chain` to trace delegation paths.

---

## Go Implementation Stack

| Component | Library | Why |
|-----------|---------|-----|
| OAuth 2.1 core | **ory/fosite** | Battle-tested, extensible handler architecture, used by Hydra/OpenAI. Implement storage interfaces for SQLite. |
| JWT / JWK / JWS | **lestrrat-go/jwx** | Best Go JWx library. Auto-refreshing JWKS cache. Key generation + rotation. |
| DPoP | **Custom on top of jwx** | No mature Go DPoP library. ~200 LOC to validate DPoP proofs using jwx. |
| Device Flow | **Custom fosite handler** | Fosite doesn't include device flow. Implement as custom grant handler (~300 LOC). |
| Token Exchange | **Custom fosite handler** | Implement RFC 8693 as custom grant handler (~400 LOC). |
| DCR | **Custom fosite handler** | Implement RFC 7591 registration endpoint. |
| CIMD | **Custom** | Fetch + validate JSON from HTTPS URL. ~150 LOC. |

### Why fosite over building from scratch?
- Token generation, validation, storage interfaces already solved
- PKCE, authorization code, client credentials, refresh token grants built-in
- Extensible: add custom grant types via `oauth2.TokenEndpointHandler` interface
- Same library Ory Hydra uses — proven at scale
- Pure Go, no CGO — compatible with Shark's single-binary constraint

### Why NOT Hydra?
- Hydra is a full server — we need a library we embed in our binary
- Hydra requires Postgres — we need SQLite
- Hydra has its own HTTP server — we already have chi
- fosite gives us the internals without the opinions

---

## Implementation Priority

### Phase 1 — MCP-Compatible Core (P0)
1. Authorization Server Metadata (`/.well-known/oauth-authorization-server`)
2. JWKS endpoint + key generation/rotation (`/.well-known/jwks.json`)
3. Authorization Code + PKCE grant
4. Client Credentials grant
5. Token endpoint with JWT access tokens (ES256)
6. Refresh token with rotation + family-based reuse detection
7. Resource Indicators (RFC 8707) — `resource` param + audience binding
8. Dynamic Client Registration (RFC 7591)
9. Agent CRUD API
10. Consent screen UI
11. Audit trail for all OAuth events

### Phase 2 — Advanced Grants (P1)
12. Device Authorization Grant (RFC 8628)
13. Token Exchange (RFC 8693) — agent-to-agent delegation
14. Token Introspection (RFC 7662)
15. Token Revocation (RFC 7009)
16. DPoP (RFC 9449) — proof-of-possession
17. Client ID Metadata Documents (CIMD)

### Phase 3 — Enterprise Features (P2)
18. Rich Authorization Requests (RFC 9396)
19. Pushed Authorization Requests (RFC 9126)
20. Step-up authorization flow
21. Consent management UI (users review/revoke agent access)
22. Agent analytics dashboard (token usage, scope patterns)

---

## What This Gives Shark

**For the OSS community:**
The only self-hosted auth system where AI agents are first-class citizens. Not an afterthought, not an add-on, not an enterprise upsell. `shark serve` and you have a full OAuth 2.1 authorization server with MCP compatibility.

**For Shark Cloud:**
Agent auth is the natural upsell. Free tier: 5 agents. Pro: 100 agents. Business: unlimited. Enterprise: managed Token Vault, compliance reports, anomaly detection.

**For the market narrative:**
"Shark is the auth system built for the agent era. While Auth0 charges 50% more and Okta locks you into enterprise pricing, Shark gives you MCP-native agent auth in a 20MB binary. Free forever."

---

## Token Vault (Managed Third-Party OAuth)

Auth0's biggest agent differentiator — and the reason they charge a 50% add-on. Shark builds it on top of existing OAuth client infrastructure (social login already does the same flow).

### What It Does

Shark manages OAuth tokens for third-party APIs on behalf of agents and users. The agent never handles raw third-party credentials — it asks Shark "give me a token for Alice's Google Calendar" and Shark handles the entire lifecycle: OAuth flow, token storage, refresh, and delivery.

Without a Token Vault, every agent developer must:
1. Build their own OAuth flow for each third-party API
2. Store tokens securely (encrypted at rest, rotation, etc.)
3. Handle refresh token rotation per provider
4. Handle token expiry mid-operation
5. Deal with provider-specific OAuth quirks

### Schema

```sql
CREATE TABLE vault_providers (
    id              TEXT PRIMARY KEY,         -- vp_xxxx
    name            TEXT NOT NULL,            -- "google_calendar", "slack", "github"
    display_name    TEXT NOT NULL,            -- "Google Calendar"
    auth_url        TEXT NOT NULL,
    token_url       TEXT NOT NULL,
    client_id       TEXT NOT NULL,
    client_secret_enc BLOB NOT NULL,          -- AES-256-GCM encrypted
    scopes          TEXT NOT NULL DEFAULT '[]', -- JSON: default scopes to request
    icon_url        TEXT,
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL
);

CREATE TABLE vault_connections (
    id              TEXT PRIMARY KEY,         -- vc_xxxx
    provider_id     TEXT NOT NULL REFERENCES vault_providers(id),
    user_id         TEXT NOT NULL REFERENCES users(id),
    access_token_enc  BLOB NOT NULL,          -- AES-256-GCM encrypted
    refresh_token_enc BLOB,                   -- AES-256-GCM encrypted (may be NULL)
    token_type      TEXT NOT NULL DEFAULT 'Bearer',
    scopes          TEXT NOT NULL DEFAULT '[]', -- JSON: granted scopes
    expires_at      TIMESTAMP,
    metadata        TEXT DEFAULT '{}',         -- provider-specific (e.g., Google workspace ID)
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    UNIQUE(provider_id, user_id)
);
CREATE INDEX idx_vault_connections_user ON vault_connections(user_id);
```

### How It Works

**Setup (admin):**
1. Admin registers a provider: `POST /api/v1/vault/providers` with OAuth client credentials
2. Shark stores the provider config (client_id, client_secret encrypted, scopes, endpoints)

**User connects:**
1. Agent (or app) redirects user: `GET /api/v1/vault/connect/{provider}?redirect_uri=...`
2. Shark runs standard OAuth flow with the third-party (authorize → callback → exchange code)
3. Shark stores the access + refresh token encrypted in `vault_connections`
4. User is redirected back to the app

**Agent requests token:**
1. Agent calls: `GET /api/v1/vault/{provider}/token` (with agent's OAuth access token, which identifies the user via delegation)
2. Shark checks: does this user have a connection to this provider?
3. If token is expired → Shark auto-refreshes using the stored refresh token
4. Returns a fresh, short-lived access token to the agent
5. Agent uses the token to call the third-party API directly

**The agent never sees the refresh token. The agent never stores credentials. Shark handles everything.**

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/vault/providers` | Register a third-party OAuth provider (admin) |
| `GET /api/v1/vault/providers` | List available providers |
| `PATCH /api/v1/vault/providers/{id}` | Update provider config |
| `DELETE /api/v1/vault/providers/{id}` | Remove provider |
| `GET /api/v1/vault/connect/{provider}` | Start OAuth flow to connect user to provider |
| `GET /api/v1/vault/callback/{provider}` | OAuth callback (stores tokens) |
| `GET /api/v1/vault/{provider}/token` | Get fresh access token for current user+provider |
| `GET /api/v1/vault/connections` | List user's connected providers |
| `DELETE /api/v1/vault/connections/{id}` | Disconnect (revoke + delete tokens) |

### Why This Is Not Massive

Shark already has:
- OAuth client infrastructure for social login (Google, GitHub, Apple, Discord)
- AES-256-GCM field encryption (`internal/auth/fieldcrypt.go`)
- Token exchange patterns in the OAuth provider code
- The same authorize → callback → store flow

The Token Vault is the same pattern with two differences:
1. Instead of creating a Shark session on callback, store the third-party token
2. Add an endpoint to retrieve/refresh stored tokens on demand

The new code is: ~2 tables, ~6 endpoints, ~400 LOC of handler logic, reusing existing OAuth client + encryption infrastructure.

### Pre-Built Provider Templates

Ship with templates for the most common agent integrations:
- Google (Calendar, Drive, Gmail, Sheets)
- Slack
- GitHub
- Microsoft (Outlook, OneDrive, Teams)
- Notion
- Linear
- Jira

Each template pre-fills: auth_url, token_url, default scopes, icon. Admin just adds their client_id + client_secret.

### Cloud Pricing Tier

| Tier | Connected Providers |
|------|-------------------|
| Free | 3 providers |
| Pro | 20 providers |
| Business | Unlimited |
| Self-hosted | Unlimited (always) |

### Audit Events

- `vault.provider.created` / `vault.provider.updated` / `vault.provider.deleted`
- `vault.connected` — user connected to a provider
- `vault.disconnected` — user disconnected
- `vault.token.retrieved` — agent requested a token (log agent_id, provider, user_id)
- `vault.token.refreshed` — Shark auto-refreshed an expired token
- `vault.token.failed` — refresh failed (revoked by user at provider, expired refresh token)

---

## Configurable Session Mode (Cookie vs JWT)

Since the JWT infrastructure (signing keys, JWKS, verification) is built for agent auth, expose it for human sessions too.

```yaml
auth:
  session_mode: "cookie"       # cookie (default) | jwt
  jwt:
    access_lifetime: "15m"
    refresh_lifetime: "7d"
    signing_algorithm: "ES256"  # ES256 | RS256
```

**`cookie` mode (default):** Current behavior. Server-side sessions, encrypted cookies, 5KB SDK. Best for monoliths, SSR apps, simple setups.

**`jwt` mode:** Stateless JWT access tokens + refresh tokens. JWKS endpoint for verification. Best for microservices, edge validation, polyglot backends where multiple services verify auth without calling Shark.

Both modes share the same signing key infrastructure, JWKS endpoint, and user model. The SDK auto-detects which mode the server runs and adjusts (cookie mode uses `credentials: 'include'`, JWT mode uses `Authorization: Bearer`).

This kills a common objection: "I need JWTs for my architecture, so Shark won't work." Now it works for both.

---

*This spec should be reviewed after Organizations (#50) ships, as org-scoped agent permissions depend on the org model.*
