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

- Import from `shark_auth.claims` directly, or via `from shark_auth import DelegationTokenClaims`
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
