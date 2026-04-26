# Wave 2 — Python SDK Killer Methods (expanded post-1.5/1.6 layer additions)

**Budget:** 8-13h CC (was 6-10h, +2-3h for 5 additional methods covering Layers 3/4/5 + DPoP rotation + customer-agent listing) · **Outcome:** 10-line README snippet that Auth0 cannot match + complete coverage of the depth-of-defense model from the SDK side

## The killer 10-liner (the goal)

This is what lands at the top of the README. It is the artifact technical readers see in 30 seconds. Every method below exists to make this snippet runnable.

```python
from shark_auth import Client, DPoPProver

client = Client(base_url="https://auth.example.com", token="sk_live_...")
prover = DPoPProver.generate()

# Auth0 cannot do this: DPoP-bound token in one line
token = client.oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="agent-123",
    client_secret="secret",
    scope="mcp:write",
)

# Delegation (RFC 8693) — scoped sub-token bound to the same key
sub_token = client.oauth.token_exchange(
    subject_token=token.access_token,
    scope="mcp:read",
    dpop_prover=prover,
)

# Use it — DPoP proof auto-signed per request
result = client.http.get_with_dpop("/resource", token=sub_token.access_token, prover=prover)
```

## SDK gap matrix (per audit)

| Capability | Backend | Python SDK before | Python SDK after Wave 2 |
|---|---|---|---|
| Agent CRUD | ✓ Full | ⚠️ Partial 60% | ✓ Full |
| DPoP keygen + signing | ✓ Full | ✓ Full (already) | ✓ Full |
| Token request M2M | ✓ Full | ❌ None | ✓ Full |
| Token refresh | ✓ Full | ❌ None | ✓ Full |
| Token exchange RFC 8693 | ✓ Full | ❌ None (constant exists, no method) | ✓ Full |
| Scoped sub-tokens | ✓ Full | ❌ None | ✓ Full |
| Policy helpers | ✓ Introspect | ⚠️ 40% | ✓ Full |

## The 5 methods

### Method 1 — `OAuthClient.get_token_with_dpop()`

**Location:** `sdk/python/shark_auth/oauth_client.py` (or current OAuthClient module)

```python
def get_token_with_dpop(
    self,
    *,
    grant_type: str,
    dpop_prover: DPoPProver,
    client_id: str,
    client_secret: str | None = None,
    scope: str | None = None,
    **extra: Any,
) -> Token:
    """
    Request an OAuth token with a DPoP proof header.

    Wraps POST /oauth/token. The DPoP proof is signed with the prover's keypair
    and bound to the request method+URL. The returned token has cnf.jkt matching
    the prover's public-key thumbprint — token theft alone is useless.

    Supports: client_credentials, authorization_code, refresh_token grants.
    """
```

**Behavior:**
- Builds a DPoP proof: `prover.make_proof(htm="POST", htu=token_endpoint)`
- POSTs `{grant_type, client_id, client_secret, scope, **extra}` form-encoded
- Sets header `DPoP: <proof>`
- Parses response into `Token` dataclass: `(access_token, token_type, expires_in, scope, refresh_token, cnf_jkt)`
- Raises `OAuthError` on 4xx/5xx with parsed `error` and `error_description`

**Tests:**
- Unit: mock token endpoint, verify DPoP header signed correctly, verify Token dataclass populated
- Smoke: against real `shark serve`, request M2M token, assert returned `cnf.jkt` matches `prover.thumbprint()`

### Method 2 — `OAuthClient.token_exchange()`

**Location:** same module as Method 1

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
) -> Token:
    """
    Exchange a token for a delegated sub-token (RFC 8693).

    The returned token has an `act` claim recording the actor chain.
    Scope can only be narrower than subject_token's scope (server enforces).
    """
```

**Behavior:**
- POSTs to `/oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`
- Includes DPoP proof header
- Returns Token with `act` chain visible via Method 4 helper

**Tests:**
- Unit: mock exchange endpoint, verify request body shape
- Smoke: end-to-end 3-hop chain (user → agent-A → agent-B), assert each token's `act` claim shows full lineage

### Method 3 — `Client.http.get_with_dpop` / `post_with_dpop` / `delete_with_dpop`

**Location:** `sdk/python/shark_auth/http_client.py` (or wherever the protected-resource helper lives)

```python
def get_with_dpop(self, path: str, *, token: str, prover: DPoPProver, **kwargs) -> Response: ...
def post_with_dpop(self, path: str, *, token: str, prover: DPoPProver, json: Any = None, **kwargs) -> Response: ...
def delete_with_dpop(self, path: str, *, token: str, prover: DPoPProver, **kwargs) -> Response: ...
```

**Behavior:**
- Generates a fresh DPoP proof per request: binds `htm=<METHOD>`, `htu=<full URL>`, `ath=<sha256 hash of token>`
- Sends `Authorization: DPoP <token>` + `DPoP: <proof>`
- Returns the underlying httpx/requests Response untouched

**Tests:**
- Unit: verify proof has correct `htm`, `htu`, `ath`, fresh `jti` per call
- Integration: against real shark, hit a DPoP-protected endpoint, assert 200

### Method 4 — `AgentTokenClaims.delegation_chain()`

**Location:** `sdk/python/shark_auth/claims.py` (or a new `agent_claims.py`)

```python
@dataclass
class ActorClaim:
    sub: str  # agent client_id
    iat: int  # delegated-at timestamp
    scope: str | None
    jkt: str | None  # confirmation thumbprint at this hop

class AgentTokenClaims:
    @classmethod
    def parse(cls, jwt_token: str) -> "AgentTokenClaims": ...
    
    def delegation_chain(self) -> list[ActorClaim]:
        """Walk the `act` claim chain bottom-up, returning each hop in order."""
    
    def is_delegated(self) -> bool: ...
    def has_scope(self, scope: str) -> bool: ...
```

**Behavior:**
- Parses JWT (no signature verification — the resource server does that)
- Recursively walks `act` claim, building list of `ActorClaim`
- Returns `[]` if no `act` claim (token is direct, not delegated)

**Tests:**
- Unit: parse fixture JWTs with 0/1/2/3 hops, assert chain length and ordering

### Method 5 — `AgentsClient` extras

**Location:** `sdk/python/shark_auth/agents.py`

Add to the existing `AgentsClient`:

```python
def list_tokens(self, agent_id: str) -> list[TokenInfo]:
    """List active tokens for an agent. GET /api/v1/agents/{id}/tokens"""

def revoke_all(self, agent_id: str) -> RevokeResult:
    """Revoke all active tokens for an agent. POST /api/v1/agents/{id}/revoke-all"""

def rotate_secret(self, agent_id: str) -> AgentCredentials:
    """Rotate the agent's client secret. POST /api/v1/agents/{id}/rotate-secret"""

def get_audit_logs(self, agent_id: str, *, limit: int = 100, since: datetime | None = None) -> list[AuditEvent]:
    """Fetch audit events filtered to this agent. GET /api/v1/audit-logs?actor_id=<id>"""
```

**Behavior:**
- Standard admin-API wrappers, all use the existing `sk_live_*` auth header
- Return typed dataclasses

**Tests:**
- Unit: mock each endpoint, verify request shape and response parsing
- Smoke: register a test agent, exercise each method, verify side effects via audit

## Methods 6-10 — Depth-of-defense layer coverage

These methods expose Layers 3/4/5 of the security model + DPoP rotation + customer-agent listing. Without them, the "five layers" pitch is server-only — the SDK can't drive the demo or Twitter screencasts that show layers 3/4/5 in action.

### Method 6 — `UsersClient.revoke_agents()` (Layer 3 — customer-fleet cascade)

**Location:** `sdk/python/shark_auth/users.py` (new file or extend existing)

```python
def revoke_agents(
    self,
    user_id: str,
    *,
    agent_ids: list[str] | None = None,
    reason: str | None = None,
) -> CascadeRevokeResult:
    """
    Cascade-revoke all agents owned by a user (Layer 3 of the depth-of-defense model).
    
    If agent_ids is provided, revokes only those (still scoped to this user's agents).
    If None, revokes ALL agents created_by=user_id AND all consents granted by user_id.
    
    Returns: { revoked_agent_ids, revoked_consent_count, audit_event_id }
    
    Auth: requires admin API key. Cascade revocation is destructive — never expose to a
    session token even for the user themselves.
    """
```

**Behavior:** wraps `POST /api/v1/users/{id}/revoke-agents` (ships in Wave 1.5). Returns typed result. Smoke test: register user → register 3 agents under that user → cascade revoke → assert all 3 token-revoked + audit event fired.

### Method 7 — `UsersClient.list_agents()` (customer-agent listing)

**Location:** `sdk/python/shark_auth/users.py`

```python
def list_agents(
    self,
    user_id: str,
    *,
    filter: Literal["created", "authorized", "all"] = "all",
    limit: int = 100,
    offset: int = 0,
) -> AgentList:
    """
    List agents tied to a user.
    - filter='created': agents where created_by = user_id
    - filter='authorized': agents this user has granted consent to via oauth_consents
    - filter='all': union of the above
    """
```

**Behavior:** wraps `GET /api/v1/users/{id}/agents?filter=...` (ships in Wave 1.5). Powers the "My Agents" UI tab and gives SDK users programmatic access for their own customer-fleet listings.

### Method 8 — `OAuthClient.bulk_revoke_by_pattern()` (Layer 4 — agent-type bulk revoke)

**Location:** extend `sdk/python/shark_auth/oauth_client.py`

```python
def bulk_revoke_by_pattern(
    self,
    *,
    client_id_pattern: str,
    reason: str,
) -> BulkRevokeResult:
    """
    Revoke all tokens whose client_id matches a SQLite GLOB pattern.
    
    Pattern syntax: * = any sequence, ? = single char.
    Example: 'shark_agent_v3.2_*' kills all v3.2 instances across all customers.
    
    Returns: { revoked_count, audit_event_id, pattern_matched }
    Auth: admin API key only.
    """
```

**Behavior:** wraps `POST /api/v1/admin/oauth/revoke-by-pattern` (ships in Wave 1.6). Smoke test: register 3 agents matching pattern + 2 not matching → revoke pattern → assert 3 revoked, 2 unaffected.

### Method 9 — `VaultClient.disconnect()` (Layer 5 — vault-cascade)

**Location:** `sdk/python/shark_auth/vault.py` (new file)

```python
def disconnect(
    self,
    connection_id: str,
    *,
    cascade_to_agents: bool = True,
) -> VaultDisconnectResult:
    """
    Disconnect a vault connection.
    
    If cascade_to_agents=True (default), also revokes tokens for any agent that has
    ever fetched from this vault connection (Layer 5 cascade).
    
    Returns: {
        connection_id,
        revoked_agent_ids: list[str],
        revoked_token_count: int,
        audit_event_id,
    }
    """

def fetch_token(
    self,
    *,
    provider: str,
    bearer_token: str,
    prover: DPoPProver,
) -> VaultTokenResult:
    """
    Fetch a fresh OAuth token from the vault using a DPoP-bound delegated agent token.
    
    GET /api/v1/vault/{provider}/token with Authorization: DPoP <bearer_token>.
    Server validates DPoP proof, vault:read scope, and tok.UserID binding before
    returning the encrypted-at-rest token.
    
    Returns: { access_token, token_type, expires_at, provider_name }
    """
```

**Behavior:** `disconnect()` wraps `DELETE /api/v1/vault/connections/{id}` (Wave 1.6 adds the cascade).
`fetch_token()` wraps the existing `GET /api/v1/vault/{provider}/token` — already ships per Phase 5.5 audit. Critical for the demo: this is the method `delegation_with_vault_trace.py` calls in the third-hop fetch.

### Method 10 — `AgentsClient.rotate_dpop_key()` (DPoP key rotation primitive)

**Location:** extend `sdk/python/shark_auth/agents.py`

```python
def rotate_dpop_key(
    self,
    agent_id: str,
    *,
    new_public_key_jwk: dict,  # caller provides new pubkey JWK
    reason: str | None = None,
) -> DPoPRotationResult:
    """
    Rotate an agent's DPoP keypair binding (admin-forced).
    
    Caller supplies the new public key JWK; server records it as the new
    cnf.jkt for future token issuance and invalidates tokens bound to the old jkt.
    
    Returns: {
        old_jkt, new_jkt, revoked_token_count, audit_event_id
    }
    Auth: admin API key only.
    """
```

**Behavior:** wraps `POST /api/v1/agents/{id}/rotate-dpop-key`. Per code audit, this endpoint is **MISSING TODAY** — needs server-side implementation in Wave 1.6 alongside the storage-level versioning.

## Updated SDK gap matrix (post-Wave 2 expanded)

| Capability | Backend ships | SDK after expanded Wave 2 |
|---|---|---|
| Agent CRUD | ✓ | ✓ Full |
| DPoP keygen + signing | ✓ | ✓ Full |
| Token request (M2M) | ✓ | ✓ |
| Token refresh | ✓ | ✓ |
| Token exchange RFC 8693 | ✓ | ✓ |
| Scoped sub-tokens | ✓ | ✓ |
| Policy helpers | ✓ | ✓ |
| **Layer 3 — customer cascade** | Wave 1.5 | ✓ Method 6 |
| **Layer 3 — list-agents-by-user** | Wave 1.5 | ✓ Method 7 |
| **Layer 4 — bulk pattern revoke** | Wave 1.6 | ✓ Method 8 |
| **Layer 5 — vault disconnect cascade** | Wave 1.6 | ✓ Method 9 |
| **Vault fetch-token (DPoP-bound)** | ships today | ✓ Method 9 |
| **DPoP key rotation** | Wave 1.6 | ✓ Method 10 |

## README update

`README.md` (root of `sdk/python/` and the repo) leads with:

```markdown
## Agent auth in 10 lines

[killer 10-liner from the top of this file]

Why this matters: every line above is something Auth0 / Clerk / WorkOS cannot ship today.
- DPoP RFC 9449 — token bound to the agent's keypair. Theft alone is useless.
- Token exchange RFC 8693 — delegated sub-tokens with auditable `act` chain.
- Per-request proof — `htm`+`htu`+`ath` binding, no replay.

Try it: `pip install shark-auth && shark serve` → see the demo at /demo.
```

## Examples

`examples/agent_delegation.py` — end-to-end runnable script:

```python
"""
Demonstrates: register agent, get DPoP-bound token, exchange for sub-token,
call protected endpoint, verify delegation chain in audit log.

Run: python examples/agent_delegation.py
"""
# ~30 LOC, runnable against `shark serve` with default first-boot
```

## Definition of done for Wave 2

- 5 methods implemented with type hints + docstrings + one usage example each
- All methods have unit tests (mock endpoints) AND smoke tests (real binary)
- README updated with killer 10-liner at the top
- `examples/agent_delegation.py` ships and runs against fresh `shark serve`
- Python package builds cleanly (`pnpm` or `python -m build` per existing CI)
- Smoke suite GREEN (375 PASS)
- Tagged release ready: `shark-auth==<launch_version>` on PyPI (or rc tag if not pushing yet)

## TypeScript SDK

Ship Python first per audit recommendation (closer to done). Port to TS in W18 with a "now in TypeScript too" follow-up post — turns one launch into two news cycles.
