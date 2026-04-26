# Agent Detail Drawer

The agent detail drawer slides in from the right when you click any row in the Agents table. It provides five tabs.

## Tabs

| Tab | Purpose |
|-----|---------|
| Config | Edit name, description, redirect URIs, grant types, scopes, and rotate the client secret. |
| Tokens | List and revoke active tokens issued to this agent. |
| Consents | Users who have authorized this agent via OAuth consent. |
| Audit | Time-ordered audit log of events for this agent. |
| Security | DPoP keypair details and key-rotation history (added W1-Edit1). |
| Delegation Policies | Configure which other agents this agent may delegate to, with optional scope downscoping (added W1-Edit3). |

---

## Security Tab

The Security tab surfaces DPoP (Demonstrating Proof-of-Possession, RFC 9449) metadata for the agent.

### DPoP Keypair section

Displays a read-only summary of the agent's bound public key:

```
ECDSA P-256 · key_id: <value from server>
Thumbprint (jkt): <first 8 chars>...<last 4 chars>  [copy button]
```

- **Algorithm**: Always ECDSA P-256 for DPoP.
- **key_id**: The server-assigned key identifier (`dpop_key_id` / `key_id` field on the agent object).
- **Thumbprint (jkt)**: The JWK thumbprint bound to tokens issued with DPoP (`dpop_jkt` / `jkt` field). Displayed as a truncated string (first 8 + last 4 characters) for readability. The copy button places the **full** thumbprint string on the clipboard.

If the agent has not yet performed a DPoP flow, both fields render as `—`.

### Rotation history section

A collapsible panel showing the last 5 key-rotation events drawn from the existing audit endpoint (`GET /api/v1/agents/:id/audit`). Events are filtered to those whose `action` contains `key_rotation`, `dpop`, or `rotate_key`.

Each row shows:
- Severity dot (danger / warn / info)
- Action string
- Actor (user or service that triggered the rotation)
- Relative timestamp

If no matching audit events exist the panel renders "No key rotation events found in audit log."

---

## Data contract

The Security tab consumes these fields from `GET /api/v1/agents/:id`:

| Field | Type | Description |
|-------|------|-------------|
| `dpop_jkt` (or `jkt`) | `string \| null` | JWK thumbprint of the bound DPoP key. |
| `dpop_key_id` (or `key_id`) | `string \| null` | Server-assigned key identifier. |

Rotation history comes from `GET /api/v1/agents/:id/audit?limit=50` (existing endpoint, no new backend required).

---

---

## Delegation Policies Tab

The Delegation Policies tab controls agent-to-agent delegation: which other registered agents the current agent is permitted to delegate work to, and any scope restrictions applied to those delegations.

### Read-only summary

When at least one policy is configured the tab displays:

```
This agent can delegate to: [agent-a] [agent-b]  (edit)
```

Each target agent is shown as a chip. Clicking **edit** enters edit mode.

### Empty state

If no delegation policies have been set:

```
No delegation policies set. This agent cannot delegate to any other agent.
[Configure]
```

Clicking **Configure** enters edit mode.

### Edit mode

A checkbox grid lists every other agent in the workspace. Each row contains:

| Control | Description |
|---------|-------------|
| Checkbox | Enable/disable delegation to this agent |
| Agent name + description | Identity of the target agent |
| Scope input | Optional scope string to downscope delegated tokens (disabled when checkbox is unchecked) |

**Save policies** calls `POST /api/v1/agents/{id}/policies` with the full policies array. Saving an empty list removes all delegations. The backend fires an audit event automatically via existing audit middleware.

### Data contract

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `GET /api/v1/agents/{id}/policies` | GET | Fetch existing delegation policies for display |
| `POST /api/v1/agents/{id}/policies` | POST | Replace the full policy set; body: `{ "policies": [{ "delegate_to_id": "<uuid>", "scope": "<optional>" }] }` |

Both endpoints are pre-existing (per audit findings); no new backend changes are needed.

---

## Smoke tests

`tests/smoke/test_w1_edit1_dpop_security_tab.py` covers:

1. **Happy path** — agent detail returns 200 with DPoP fields present (may be `null` for a newly-created agent); audit endpoint is reachable and returns a `data` list.
2. **Negative path** — unknown agent ID returns 404 on both the detail and audit endpoints.

`tests/smoke/test_w1_edit3_delegation_policies.py` covers:

1. **Happy path** — `POST /api/v1/agents/{id}/policies` with a valid policy returns 2xx; subsequent GET reflects saved policies; empty-list POST clears all.
2. **Negative paths** — POST without auth returns 401; POST with malformed body returns 4xx; GET/POST on an unknown agent ID returns 404.
