# Audit Log — Event Detail Panel

## Overview

The Audit page (`/audit`) streams every authentication, RBAC change, and admin
action in real time. Clicking any row opens the **Event Detail** side panel.

---

## Delegation Chain Breadcrumb (W1-Edit2)

### What it shows

For audit events where `actor_type === "agent"` **and** the JWT carried an `act`
claim chain, the detail panel renders a horizontal breadcrumb **above** the raw
JSON payload:

```
[user]  →  [triage-agent (jkt:zQ7m…)]  →  [knowledge-agent (jkt:5kL9…)]
```

Each segment is a clickable chip that navigates to `/agents/:id` (the Agent
Detail drawer).  Non-agent events (human users, admin, system) are unaffected —
the breadcrumb is not rendered at all.

### Policy applied row

Immediately below the breadcrumb, a **"permitted by policy"** row shows the
`may_act` rule that authorised the delegation:

```
✓ permitted by policy: triage-agent → knowledge-agent
```

The value is read from the audit event's own metadata — no extra backend
requests are made.

---

## Data shape the breadcrumb consumes

The component reads from `event._raw` (the raw API payload). Two shapes are
supported:

### Preferred — `act_chain` array (server-serialised)

```json
{
  "actor_type": "agent",
  "actor_email": "knowledge-agent@agents.local",
  "act_chain": [
    { "sub": "usr_7fK2a8", "label": "amelia@nimbus.sh" },
    { "sub": "agt_triage_01", "label": "triage-agent", "jkt": "zQ7m..." }
  ],
  "may_act_rule": "triage-agent → knowledge-agent"
}
```

Each node:

| Field   | Required | Description                                    |
|---------|----------|------------------------------------------------|
| `sub`   | yes      | Opaque subject identifier (agent ID or user ID)|
| `label` | no       | Human-readable display name                    |
| `jkt`   | no       | JWK thumbprint of the DPoP key (first 4 chars shown)|

### Legacy — `oauth.act` single-hop

```json
{
  "actor_type": "agent",
  "oauth": {
    "act": { "sub": "usr_7fK2a8", "email": "amelia@nimbus.sh" }
  }
}
```

The component synthesises a two-segment breadcrumb from this shape automatically.

### `may_act_rule` fallback

If `may_act_rule` / `policy_may_act` are absent from the raw payload the
component derives the rule string as `<last act_chain node> → <terminal agent>`.

---

## Navigation

Clicking an agent chip sets `window.location.hash` to `/agents/:id`, which
opens the Agent Detail drawer.  The originating-user chip (index 0) is not
clickable (no `/agents/` route for human accounts).

---

## Visual style

- Monochrome palette using `var(--surface-*)`, `var(--hairline*)`, `var(--fg*)` tokens
- Square corners (`border-radius: 3px`) throughout
- Font: `var(--font-mono)` for all identifiers
- Success tick uses `var(--success, #22c55e)` with graceful fallback

---

## Missing-field rendering (no fabrication)

The Event Detail drawer renders only fields actually returned by the backend.
Fields the backend has not populated are shown as an em-dash placeholder:

```
<span className="faint">—</span>
```

Specifically:

| Field        | Source                                   | If absent |
|--------------|------------------------------------------|-----------|
| `request_id` | `event._raw.request_id` (DB column)      | `—`        |
| `session_id` | `event._raw.session_id` (DB column)      | `—`        |
| `act_chain`  | `parseMeta(event._raw).act_chain` (JSON) | section hidden |
| `may_act_rule` | `parseMeta(event._raw).may_act_rule`   | row hidden |
| `oauth.*`    | `parseMeta(event._raw).oauth.*`          | `—`        |

`parseMeta()` safely parses the `metadata` TEXT column (stored as a JSON
string by the backend). All sub-field access goes through this helper.

**No `Math.random()` fallbacks exist in the drawer.** Previous fabricated
values (random `req_*` / `sess_*` IDs that changed on every render) have been
removed.

For the canonical list of which `action` values populate which metadata fields
see [documentation/api/audit.md](../api/audit.md).
