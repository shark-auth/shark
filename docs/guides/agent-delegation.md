# Agent-to-Agent Delegation (RFC 8693)

Chain agents A→B→C with downscoped tokens and a cryptographically verifiable audit trail. SharkAuth implements RFC 8693 Token Exchange natively — every hop is a standard `POST /oauth/token` call.

**Audience:** Builders composing multi-agent systems who need traceable, least-privilege delegation.

**Reference:** Live flow in [`demos/DEMO_02_DELEGATION_CHAIN.md`](../../demos/DEMO_02_DELEGATION_CHAIN.md).

---

## How it works

Each agent holds a token issued on behalf of the original subject. When it delegates to a downstream agent, it exchanges its token at the AS and the AS:

1. Verifies the subject token's signature and expiry
2. Checks that the acting agent is listed in `may_act` (if present)
3. Enforces scope narrowing — the new token cannot exceed the subject token's scopes
4. Builds a nested `act` claim recording the full delegation chain
5. Issues a new audience-bound token

The result: the deepest token in the chain carries a complete, tamper-evident record of every agent that touched it, all the way back to the original subject.

---

## Prerequisites

```bash
# Start shark in dev mode
shark serve --dev
```

Register three agents (replace names and secrets for your system):

```bash
# Platform / orchestrator — receives the initial broad-scope token
curl -sS -X POST http://localhost:8080/oauth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "client_name": "orchestrator",
    "grant_types": ["client_credentials"],
    "token_endpoint_auth_method": "client_secret_basic",
    "scope": "ticket:read ticket:resolve email:draft kb:read"
  }' | jq '{client_id, client_secret}'

# Agent B — sub-agent, receives narrowed scope
curl -sS -X POST http://localhost:8080/oauth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "client_name": "email-agent",
    "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
    "token_endpoint_auth_method": "client_secret_basic",
    "scope": "email:draft"
  }' | jq '{client_id, client_secret}'

# Agent C — tool, receives further narrowed scope
curl -sS -X POST http://localhost:8080/oauth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "client_name": "gmail-tool",
    "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
    "token_endpoint_auth_method": "client_secret_basic",
    "scope": "email:send"
  }' | jq '{client_id, client_secret}'
```

Set env vars from the responses:

```bash
export ORCH_ID="shark_dcr_aaa"      ; export ORCH_SECRET="cs_live_..."
export EMAIL_ID="shark_dcr_bbb"     ; export EMAIL_SECRET="cs_live_..."
export GMAIL_ID="shark_dcr_ccc"     ; export GMAIL_SECRET="cs_live_..."
```

---

## Token exchange call shape

The grant type is `urn:ietf:params:oauth:grant-type:token-exchange` (RFC 8693 §2.1).

The **acting agent** authenticates with its own credentials (`-u client_id:secret`). The **subject token** is the token being delegated from.

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded
Authorization: Basic <base64(acting_client_id:acting_client_secret)>

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token=<token-to-delegate-from>
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&scope=<requested-scope>          # must be a subset of subject_token's scopes
&audience=<resource-identifier>   # RFC 8707 audience binding
```

Optional fields:
- `actor_token` / `actor_token_type` — explicit actor credential (rarely needed; Basic auth is sufficient)

---

## Hop 0 — Orchestrator gets a broad-scope token

```bash
ORCH_TOKEN=$(curl -sS -X POST http://localhost:8080/oauth/token \
  -u "${ORCH_ID}:${ORCH_SECRET}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "grant_type=client_credentials" \
  -d "scope=ticket:read ticket:resolve email:draft kb:read" \
  -d "audience=core-api" \
  | jq -r '.access_token')
```

Decode it to confirm no `act` claim yet:

```bash
python3 -c "
import base64, json, sys
parts = '${ORCH_TOKEN}'.split('.')
padded = parts[1] + '=' * (-len(parts[1]) % 4)
print(json.dumps(json.loads(base64.urlsafe_b64decode(padded)), indent=2))
"
```

```json
{
  "iss": "http://localhost:8080",
  "sub": "shark_dcr_aaa",
  "aud": "core-api",
  "scope": "ticket:read ticket:resolve email:draft kb:read",
  "exp": 1745000000,
  "iat": 1744999100
}
```

---

## Hop 1 — Orchestrator delegates to email-agent (scope narrows)

```bash
EMAIL_TOKEN=$(curl -sS -X POST http://localhost:8080/oauth/token \
  -u "${EMAIL_ID}:${EMAIL_SECRET}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${ORCH_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=email:draft" \
  -d "audience=gmail-vault" \
  | jq -r '.access_token')
```

Decode the delegated token:

```json
{
  "iss": "http://localhost:8080",
  "sub": "shark_dcr_aaa",
  "aud": "gmail-vault",
  "scope": "email:draft",
  "act": {
    "sub": "shark_dcr_bbb"
  },
  "exp": 1745000000,
  "iat": 1744999200
}
```

Key observations:
- `sub` is still the original subject (`orchestrator`) — the identity chain is preserved
- `scope` narrowed from `ticket:read ticket:resolve email:draft kb:read` → `email:draft`
- `act.sub` identifies who is currently acting (`email-agent`)
- `aud` is bound to `gmail-vault` — this token cannot be reused at another resource

---

## Hop 2 — Email-agent delegates to gmail-tool (3-deep chain)

```bash
GMAIL_TOKEN=$(curl -sS -X POST http://localhost:8080/oauth/token \
  -u "${GMAIL_ID}:${GMAIL_SECRET}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${EMAIL_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=email:send" \
  -d "audience=smtp-relay" \
  | jq -r '.access_token')
```

Decode:

```json
{
  "iss": "http://localhost:8080",
  "sub": "shark_dcr_aaa",
  "aud": "smtp-relay",
  "scope": "email:send",
  "act": {
    "sub": "shark_dcr_ccc",
    "act": {
      "sub": "shark_dcr_bbb"
    }
  },
  "exp": 1745000000,
  "iat": 1744999300
}
```

The `act` claim now nests two levels deep. Any resource server can walk the chain:

```
sub: orchestrator  (original user/system)
act.sub: gmail-tool  (current actor)
act.act.sub: email-agent  (delegated from)
```

Implemented at `internal/oauth/exchange.go` L315-322 (`buildActClaim()`).

---

## `act` claim chain in Python

If you're using the shark Python SDK:

```python
from shark_auth import decode_agent_token

claims = decode_agent_token(
    token=gmail_token,
    jwks_url="http://localhost:8080/oauth/jwks",
    issuer="http://localhost:8080",
    audience="smtp-relay",
)

print(f"Original subject: {claims.sub}")
print(f"Current actor:    {claims.act['sub'] if claims.act else 'none'}")
print(f"Full act chain:   {claims.act}")
print(f"Scopes:           {claims.scope}")
```

`AgentTokenClaims.act` is typed as `Optional[Dict[str, Any]]` — walk it recursively to reconstruct the full chain.

---

## Scope downscoping enforcement

SharkAuth rejects any exchange that attempts to escalate scopes. Attempting to request a scope not present in the subject token:

```bash
curl -sS -X POST http://localhost:8080/oauth/token \
  -u "${EMAIL_ID}:${EMAIL_SECRET}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${EMAIL_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=crm:write" \
  -d "audience=salesforce-api"
```

```json
{
  "error": "invalid_scope",
  "error_description": "requested scope exceeds subject token scope"
}
```

Enforced by `scopesSubset()` at `internal/oauth/exchange.go` L104-118.

---

## `may_act` — restrict who can delegate from a token

When issuing a token, annotate it with which clients may exchange it:

```bash
# Issue a token with may_act restriction
# (set via the admin API or by including may_act in the client's registered metadata)
```

If the subject token includes `may_act: ["crm-agent"]`, any attempt by a different client to exchange it is blocked:

```bash
curl -sS -X POST http://localhost:8080/oauth/token \
  -u "${EMAIL_ID}:${EMAIL_SECRET}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${CRM_ONLY_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=crm:write" \
  -d "audience=salesforce-api"
```

```json
{
  "error": "access_denied",
  "error_description": "acting agent is not permitted by may_act"
}
```

Enforced by `isMayActAllowed()` at `internal/oauth/exchange.go` L120-126.

---

## Audit log — query the delegation trail

Every token exchange emits an `oauth.token.exchanged` event. Query the audit log with your admin API key (printed on `shark serve --dev` startup):

```bash
ADMIN_KEY="sk_live_..."

# All token exchange events
curl -sS "http://localhost:8080/api/v1/audit-logs?event_type=oauth.token.exchanged" \
  -H "Authorization: Bearer ${ADMIN_KEY}" | jq '.logs[] | {actor_id, target_id, action, created_at}'

# Filter by session or org
curl -sS "http://localhost:8080/api/v1/audit-logs?session_id=sess_xxx" \
  -H "Authorization: Bearer ${ADMIN_KEY}" | jq .

# Filter by actor
curl -sS "http://localhost:8080/api/v1/audit-logs?actor_id=${EMAIL_ID}" \
  -H "Authorization: Bearer ${ADMIN_KEY}" | jq .

# Paginate with cursor
curl -sS "http://localhost:8080/api/v1/audit-logs?limit=50&cursor=<next_cursor>" \
  -H "Authorization: Bearer ${ADMIN_KEY}" | jq '{logs: .logs | length, next_cursor}'
```

Supported query parameters:

| param | description |
|-------|-------------|
| `actor_id` | filter by who performed the action |
| `actor_type` | `user`, `agent`, `system` |
| `org_id` | filter by organization |
| `session_id` | filter by session |
| `from` | RFC3339 start timestamp |
| `to` | RFC3339 end timestamp |
| `limit` | 1–200, default 50 |
| `cursor` | opaque pagination cursor from `next_cursor` |

Export to CSV for incident review:

```bash
curl -sS -X POST http://localhost:8080/api/v1/audit-logs/export \
  -H "Authorization: Bearer ${ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "from": "2026-04-29T00:00:00Z",
    "to": "2026-04-29T23:59:59Z",
    "actor_id": "'${EMAIL_ID}'"
  }' -o delegation-audit.csv

head -3 delegation-audit.csv
# id,created_at,actor_id,actor_type,action,target_type,target_id,org_id,session_id,...
```

Every exchange also records `DelegationSubject` and `DelegationActor` in the `oauth_tokens` table (`internal/oauth/exchange.go` L187-198).

---

## Resource server: verify a delegated token

Any resource server receiving a delegated token should:

1. Validate the JWT signature against `GET /oauth/jwks`
2. Check `aud` matches the server's identifier
3. Check `scope` contains the required scope
4. Optionally: inspect `act` to enforce chain policies (e.g. only allow tokens delegated from known orchestrators)

```python
from shark_auth import decode_agent_token

def require_scope(token: str, required_scope: str, audience: str) -> dict:
    claims = decode_agent_token(
        token=token,
        jwks_url="http://localhost:8080/oauth/jwks",
        issuer="http://localhost:8080",
        audience=audience,
    )
    scopes = set((claims.scope or "").split())
    if required_scope not in scopes:
        raise PermissionError(f"token missing scope: {required_scope}")
    return claims
```

---

## Known limitations (v0.1)

- **DPoP per-hop re-binding at exchange is not yet enforced.** The acting agent does not currently need to present a DPoP proof at exchange time; `cnf.jkt` in the delegated token reflects the subject token's binding. Full per-hop DPoP re-binding is a post-v0.1 enhancement.
- **`may_act` must be set via token metadata at issuance time.** There is no admin UI for this yet; it must be embedded in the client registration or via a future API endpoint.
