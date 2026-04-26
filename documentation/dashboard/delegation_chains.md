# Delegation Chains

## What it shows

The **Delegation Chains** page (`/admin/delegation-chains`) gives you a standalone view of every
token-exchange delegation chain visible in your audit log. A "delegation chain" is the ordered list
of principals that caused a particular token-exchange event: typically a human user at the root,
followed by one or more agent hops, each delegating authority to the next via the RFC 8693 `act`
(actor) claim mechanism.

Example chain:

```
user (demo@example.com)  →  user-proxy (jkt:7zKt)  →  email-svc (jkt:Tyhw)
                                                      →  followup-svc
```

The page groups audit events by the root subject of their `act_chain` field, so each row in the
list represents one delegation chain with all its hops collapsed into a compact breadcrumb.

---

## Data sources

The page fetches two audit-log queries in parallel:

| Query | Purpose |
|---|---|
| `GET /api/v1/audit-logs?action=oauth.token.exchanged&limit=50` | Token exchange events — core delegation hops |
| `GET /api/v1/audit-logs?action=vault.token.retrieved&limit=50` | Terminal vault-retrieval events — final hops |

Both queries respect the selected time-window filter (1h / 24h / 7d / all).

Grouping is done client-side: events are grouped by the root subject of the `act_chain` array
(the first element's `sub` or `label`). If no `act_chain` is present, the legacy `oauth.act.sub`
single-hop field is used as a fallback.

> **Wave 1.6 note:** A dedicated `/api/v1/audit-logs/chains` aggregator endpoint would return
> pre-grouped chain objects from the server, enabling true server-side pagination and cross-action
> grouping. The current client-side grouping works for the initial launch; the backend endpoint
> is the recommended next improvement.

---

## When it fills

The page shows data as soon as token-exchange audit events exist. This happens when:

1. An agent calls the token-exchange endpoint (`/oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`).
2. A vault credential retrieval is performed on behalf of a delegated actor.

In a fresh deployment with no agent activity, the page shows an empty state with instructions
for generating sample data.

---

## Generating sample data

### Option 1 — CLI demo

```sh
shark demo delegation-with-trace
```

This creates a synthetic multi-hop delegation chain and emits the corresponding audit events.

### Option 2 — Agent demo tester

```sh
python tools/agent_demo_tester.py
```

The tester authenticates as a demo user, exchanges a token to act as `user-proxy`, then
exchanges again to act as `email-svc`, producing a 3-hop chain visible immediately on the
Delegation Chains page.

---

## Page layout

```
┌────────────────────────────────────────────────────────────────────┐
│  Delegation Chains                                                 │
│  [1h] [24h] [7d] [all]   Filter by actor…              N chains   │
├────────────────────────────────────────────────────────────────────┤
│  CHAIN — 2026-04-26 14:23:01                      3 events        │
│  user (demo@…)  →  user-proxy (jkt:7zKt)  →  email-svc (jkt:Tyhw)│
│  Last action: vault.token.retrieved · target=conn_xyz              │
│  [audit_abc…] [audit_def…] [audit_ghi…]                           │
├────────────────────────────────────────────────────────────────────┤
│  CHAIN — 2026-04-26 14:21:47                      1 event         │
│  user (admin@…)  →  bot-1 (jkt:Mrj8)                              │
│  Last action: oauth.token.exchanged                                │
└────────────────────────────────────────────────────────────────────┘
```

Clicking a chain row opens a right-side detail drawer showing:

- **Chain tree** — ASCII-art ordered hop list
- **Cnf.jkt verification** — per-hop check that the `cnf.jkt` thumbprint is consistent; drifting
  jkt values are flagged with a red ✗
- **JWT claims per hop** — collapsible JSON for each segment's known claims (`sub`, `cnf.jkt`, label)
- **Audit events table** — all events in the chain, click any row to jump to the Audit Log page
  filtered to that event ID
- **Export chain JSON** — downloads the raw audit events for the chain as `.json`

---

## Visual design

Follows `.impeccable.md` v3 strictly:

- Monochrome (white/black/gray only; color used only for status indicators).
- Square corners (border-radius ≤ 4px throughout).
- Editable feel: all IDs are copy-on-click; agent chips navigate to the Agents page.
- Uses only existing CSS variables: `--surface-0/1/2/3`, `--hairline`, `--hairline-strong`,
  `--fg-dim`, `--fg-muted`, `--fg-faint`, `--font-mono`.

---

## Navigation

The page appears in the left sidebar under **AGENTS** → **Delegation Chains**.
URL path: `/admin/delegation-chains`.

---

## Per-agent Delegations tab

Open any agent's drawer (Agents page → click an agent) → "Delegations" tab.

Shows a mini-graph centered on the agent:
- **Inbound** (top): agents that delegated TO this agent (their tokens were the subject_token of an exchange where this agent was the actor)
- **Outbound** (bottom): agents this agent delegated TO

Edges labeled with timestamp + scope at the time of the exchange.
Click any node to jump to that agent's drawer.

Pure inline SVG, no chart libraries, monochrome per .impeccable.md v3 lock.
