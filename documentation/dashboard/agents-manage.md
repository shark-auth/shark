# Agents — Admin Management Page

Route: `/admin/agents`  
Component: `admin/src/components/agents_manage.tsx`  
Nav group: AGENTS

## Overview

The **Agents** page is the admin view of all OAuth 2.1 client registrations for
autonomous/agent workloads. It provides a table of every registered agent with
inline filtering, a slide-over detail panel, and a create flow with one-time
secret reveal.

## Table Columns

| Column | Description |
|--------|-------------|
| Avatar | First letter of agent name |
| Name | Agent name + optional description (truncated at 280 px) |
| Client ID | Middle-truncated to 24 chars |
| Grants | Count of configured grant types |
| Scopes | Count of configured scopes |
| Created | Relative timestamp |
| **Created by** | Email/username of the registering user (W1.5 Edit 4 — see below) |
| Status | `active` / `inactive` chip |

## Toolbar

### Search

Filters the visible rows by agent name or `client_id` (case-insensitive
substring match, client-side).

### Status filter

Segmented control: **All** / **Active** / **Inactive** — filters client-side
against the already-fetched list.

### Creator filter (W1.5 Edit 4)

A `<select>` dropdown populated from `GET /api/v1/users?limit=200`. Selecting a
user rerequests agents from the server using the `created_by_user_id` query
parameter:

```
GET /api/v1/agents?limit=200&created_by_user_id=<user_id>
```

The dropdown displays `email`, falling back to `username`, `name`, or raw `id`.
Selecting the default **All creators** option removes the filter and fetches the
full agent list.

### Created by column (W1.5 Edit 4)

Each row in the agent table now shows which user registered the agent. The value
is resolved from the `created_by_user_id` or `created_by` field on the agent
object, looked up against a `userMap` built from the same users fetch.

The cell renders as a ghost button that, when clicked, navigates to the Users
page pre-filtered by that user's ID:

```
setPage('users', { search: uid })
```

If no creator is recorded the cell shows `—`.

## Slide-over Detail Panel

Clicking any row opens the `AgentDetail` slide-over with six tabs:

| Tab | Content |
|-----|---------|
| Config | Editable name, description, redirect URIs, grant types, scopes, secret rotation |
| Tokens | Active tokens with per-agent revoke-all |
| Consents | Users who have authorized this agent |
| Audit | Last 30 audit events |
| Security | DPoP keypair + rotation history |
| Delegation Policies | Which agents this agent may delegate to |

## Create Flow

**New agent** button opens a slide-over (`CreateAgentSlideOver`) with fields
for name, description, redirect URIs, grant types, and scopes. On success the
response includes a `client_secret` displayed once (copy-to-clipboard) before
the panel closes.

## Backend Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/agents?limit=200` | All registered agents |
| `GET` | `/api/v1/agents?limit=200&created_by_user_id=<id>` | Filter agents by creator |
| `POST` | `/api/v1/agents` | Register a new agent |
| `PATCH` | `/api/v1/agents/:id` | Update agent fields |
| `DELETE` | `/api/v1/agents/:id` | Deactivate agent + revoke tokens |
| `POST` | `/api/v1/agents/:id/tokens/revoke-all` | Revoke all tokens for an agent |
| `POST` | `/api/v1/agents/:id/rotate-secret` | Rotate client secret |
| `GET` | `/api/v1/agents/:id/tokens?limit=50` | List active tokens |
| `GET` | `/api/v1/agents/:id/audit?limit=30` | Audit log for agent |
| `GET` | `/api/v1/agents/:id/policies` | Delegation policies |
| `POST` | `/api/v1/agents/:id/policies` | Save delegation policies |
| `GET` | `/api/v1/users?limit=200` | User list for creator dropdown |

## CLI Parity

`shark agent list` is not implemented in the CLI. Use the dashboard or the SDK:

```python
from shark_auth import AgentsClient
agents = AgentsClient("https://auth.example.com", token="sk_live_...")
agents.list_agents()
```
