# Wave 2 SDK Methods — Documentation

## Method 4 — `AgentTokenClaims.delegation_chain()`

**File:** `sdk/python/shark_auth/claims.py`

Pure JWT parsing — no signature verification, no backend dependency. Decodes the base64url payload and walks the recursive `act` claim chain defined in RFC 8693.

### Types

```python
@dataclass
class ActorClaim:
    sub: str          # agent client_id at this hop
    iat: int          # delegated-at Unix timestamp
    scope: str | None # scopes at this hop (may differ from outer token)
    jkt: str | None   # cnf.jkt DPoP thumbprint bound at this hop
```

### Class: `AgentTokenClaims`

```python
class AgentTokenClaims:
    @classmethod
    def parse(cls, jwt_token: str) -> "AgentTokenClaims":
        """Decode a JWT payload without verifying the signature."""

    def delegation_chain(self) -> list[ActorClaim]:
        """Walk the `act` claim chain outermost-first. Returns [] for direct tokens."""

    def is_delegated(self) -> bool:
        """True if the token has at least one `act` hop."""

    def has_scope(self, scope: str) -> bool:
        """True if `scope` appears in the token's space-delimited scope string."""

    # Properties
    sub: str           # token subject
    iss: str           # issuer
    exp: int           # expiry (Unix timestamp)
    iat: int           # issued-at (Unix timestamp)
    scope: str | None  # space-delimited scope string
    jkt: str | None    # cnf.jkt from the top-level token
    raw: dict          # raw decoded payload
```

### Example

```python
from shark_auth.claims import AgentTokenClaims

claims = AgentTokenClaims.parse(sub_token.access_token)
print(claims.is_delegated())        # True
for hop in claims.delegation_chain():
    print(hop.sub, hop.scope, hop.jkt)
```

### Notes

- Import from `shark_auth.claims` directly. The top-level `shark_auth.AgentTokenClaims` is a different class (the verified-claims dataclass from `tokens.py`).
- Chain ordering: index 0 is the outermost (most recent) actor; last index is the innermost
- Safe to call on tokens without an `act` claim — returns an empty list

---

## Method 5 — `AgentsClient` extras

**File:** `sdk/python/shark_auth/agents.py`

Four new methods on `AgentsClient`. All use the existing `sk_live_*` admin API key.

### `list_tokens(agent_id) -> list[TokenInfo]`

Wraps `GET /api/v1/agents/{id}/tokens`.

```python
tokens = client.agents.list_tokens("agent_abc")
for tok in tokens:
    print(tok.token_id, tok.scope, tok.jkt)
```

**`TokenInfo` fields:** `token_id`, `agent_id`, `jkt`, `scope`, `expires_at`, `created_at`

### `revoke_all(agent_id) -> RevokeResult`

Wraps `POST /api/v1/agents/{id}/tokens/revoke-all`.

```python
result = client.agents.revoke_all("agent_abc")
print(result.revoked_count)  # int
```

**`RevokeResult` fields:** `revoked_count`, `agent_id`

### `rotate_secret(agent_id) -> AgentCredentials`

Wraps `POST /api/v1/agents/{id}/rotate-secret`. The new `client_secret` is returned once — copy it immediately.

```python
creds = client.agents.rotate_secret("agent_abc")
print(creds.client_secret)  # new secret — store securely
```

**`AgentCredentials` fields:** `agent_id`, `client_id`, `client_secret`, `rotated_at`

### `get_audit_logs(agent_id, *, limit=100, since=None) -> list[AuditEvent]`

Wraps `GET /api/v1/audit-logs?actor_id=<id>`.

```python
from datetime import datetime, timedelta
events = client.agents.get_audit_logs(
    "agent_abc",
    limit=20,
    since=datetime.utcnow() - timedelta(hours=1),
)
for ev in events:
    print(ev.event, ev.created_at)
```

**`AuditEvent` fields:** `id`, `event`, `actor_id`, `target_id`, `metadata`, `created_at`

---

## Method 6 — `UsersClient.revoke_agents()` (Layer 3 cascade)

**File:** `sdk/python/shark_auth/users.py`

```python
def revoke_agents(
    self,
    user_id: str,
    *,
    agent_ids: list[str] | None = None,
    reason: str | None = None,
) -> CascadeRevokeResult:
```

Wraps `POST /api/v1/users/{id}/revoke-agents`.

- No `agent_ids` → revokes ALL agents created by this user and all their consents
- With `agent_ids` → revokes only those agents (scoped to this user)

**`CascadeRevokeResult` fields:** `revoked_agent_ids`, `revoked_consent_count`, `audit_event_id`

```python
result = client.users.revoke_agents("usr_abc")
print(result.revoked_agent_ids)
print(result.audit_event_id)
```

**Auth:** Admin API key required. This is a destructive operation — never expose to session tokens.

---

## Method 7 — `UsersClient.list_agents()`

**File:** `sdk/python/shark_auth/users.py`

```python
def list_agents(
    self,
    user_id: str,
    *,
    filter: Literal["created", "authorized", "all"] = "all",
    limit: int = 100,
    offset: int = 0,
) -> AgentList:
```

Wraps `GET /api/v1/users/{id}/agents?filter=...`.

- `"created"` — agents where `created_by = user_id`
- `"authorized"` — agents this user has granted OAuth consent to
- `"all"` — union of the above

**`AgentList` fields:** `data` (list of agent dicts), `total`, `filter`

```python
result = client.users.list_agents("usr_abc", filter="created", limit=50)
for agent in result.data:
    print(agent["name"], agent["id"])
```

---

## Method 8 — `OAuthClient.bulk_revoke_by_pattern()`

**File:** `sdk/python/shark_auth/oauth.py`

Revoke all OAuth tokens whose `client_id` matches a SQLite GLOB pattern. Useful for emergency rollback of an entire agent class (e.g. all v3.2 instances).

### Types

```python
@dataclass
class BulkRevokeResult:
    revoked_count: int      # tokens revoked across all matching client_ids
    audit_event_id: str     # ID of the emitted audit event
    pattern_matched: str    # GLOB pattern echoed from the server
```

### Signature

```python
def bulk_revoke_by_pattern(
    self,
    *,
    client_id_pattern: str,   # SQLite GLOB — * any sequence, ? one char
    reason: str,
) -> BulkRevokeResult
```

**Auth:** Admin API key only. Wraps `POST /api/v1/admin/oauth/revoke-by-pattern`.

### Example

```python
from shark_auth import OAuthClient

oauth = OAuthClient(base_url="https://auth.example.com", token="sk_live_...")
result = oauth.bulk_revoke_by_pattern(
    client_id_pattern="shark_agent_v3.2_*",
    reason="emergency rollback 2026-04-26",
)
print(result.revoked_count)    # e.g. 47
print(result.audit_event_id)   # "audit_abc123"
print(result.pattern_matched)  # "shark_agent_v3.2_*"
```

---

## Method 9 — `VaultClient.disconnect()` + `VaultClient.fetch_token()`

**File:** `sdk/python/shark_auth/vault.py`

### Types

```python
@dataclass
class VaultDisconnectResult:
    connection_id: str
    revoked_agent_ids: list[str]       # agents cascade-revoked
    revoked_token_count: int
    cascade_audit_event_id: str | None

@dataclass
class VaultTokenResult:
    access_token: str
    token_type: str              # usually "Bearer"
    expires_at: str | None       # ISO-8601 or None
    provider: str | None         # e.g. "google_gmail"
```

### `VaultClient.disconnect()`

```python
def disconnect(
    self,
    connection_id: str,
    *,
    cascade_to_agents: bool = True,
) -> VaultDisconnectResult
```

Wraps `DELETE /api/v1/vault/connections/{id}`. When `cascade_to_agents=True` (default), the server also revokes tokens for any agent that has accessed this connection.

```python
from shark_auth import VaultClient

vault = VaultClient(base_url="https://auth.example.com", admin_key="sk_live_...")
result = vault.disconnect("conn_abc123", cascade_to_agents=True)
print(result.revoked_agent_ids)    # ["agent_x", "agent_y"]
print(result.revoked_token_count)  # 5
```

### `VaultClient.fetch_token()`

```python
def fetch_token(
    self,
    *,
    provider: str,
    bearer_token: str,
    prover: DPoPProver,
) -> VaultTokenResult
```

Wraps `GET /api/v1/vault/{provider}/token`. The server validates the DPoP proof bound to `bearer_token`, confirms the `vault:read` scope, and returns the decrypted 3rd-party token.

```python
from shark_auth import DPoPProver, OAuthClient, VaultClient

prover = DPoPProver.generate()
oauth = OAuthClient(base_url="https://auth.example.com")
token = oauth.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="shark_agent_...",
    client_secret="...",
    scope="vault:read",
)

vault = VaultClient(base_url="https://auth.example.com", admin_key="sk_live_...")
result = vault.fetch_token(
    provider="google_gmail",
    bearer_token=token.access_token,
    prover=prover,
)
print(result.access_token)  # decrypted Google access token
```

---

## Method 10 — `AgentsClient.rotate_dpop_key()`

**File:** `sdk/python/shark_auth/agents.py`

Rotate an agent's DPoP keypair binding. The caller generates a new keypair locally, supplies its public JWK, and the server atomically replaces the binding and revokes all tokens bound to the old key.

### Types

```python
@dataclass
class DPoPRotationResult:
    old_jkt: str              # thumbprint of the replaced key (empty if none was stored)
    new_jkt: str              # thumbprint of the new key (RFC 7638 SHA-256)
    revoked_token_count: int  # tokens revoked
    audit_event_id: str       # ID of "agent.dpop_key_rotated" audit event
```

### Signature

```python
def rotate_dpop_key(
    self,
    agent_id: str,
    *,
    new_public_key_jwk: dict,
    reason: str | None = None,
) -> DPoPRotationResult
```

**Auth:** Admin API key only. Wraps `POST /api/v1/agents/{id}/rotate-dpop-key`.

### Example

```python
from shark_auth import AgentsClient, DPoPProver

agents = AgentsClient(base_url="https://auth.example.com", token="sk_live_...")
new_prover = DPoPProver.generate()

result = agents.rotate_dpop_key(
    "agent_abc123",
    new_public_key_jwk=new_prover.public_jwk,
    reason="scheduled rotation 2026-04-26",
)
print(result.old_jkt, "->", result.new_jkt)
print(result.revoked_token_count)   # e.g. 3
print(result.audit_event_id)        # "audit_xyz789"
```

After rotation, all tokens bound to the old key are immediately revoked. Issue new tokens using the new prover's keypair.
