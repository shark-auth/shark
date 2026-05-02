# My Agents — User Self-Service Tab

Route: `/admin/me-agents`  
Component: `admin/src/components/me_agents.tsx`  
Nav group: AGENTS (below "Agents")

## Overview

The **My Agents** tab gives any operator who has loaded an admin API key a
self-service view of every agent linked to their account. It replaces the need
to filter the global Agents list manually.

## Sections

### Created by me

Lists every agent whose `created_by_user_id` matches the current session. Each
row shows:

| Column | Value |
|--------|-------|
| Avatar | First letter of agent name |
| Name | Agent name + optional description |
| Client ID | Truncated (mid-ellipsis, 22 chars) |
| Created | Relative timestamp (`Xs ago`, `Xm ago`, …) |
| Status | `active` / `inactive` chip |
| Action | **Revoke** button (disabled if already inactive) |

Clicking **Revoke** opens a confirmation modal. On confirm the component calls
`PATCH /api/v1/agents/:id` with `{ active: false }`, which deactivates the
agent and revokes all its tokens.

### Authorized by me

Lists every agent the current user has granted OAuth consent to. Each row shows:

| Column | Value |
|--------|-------|
| Avatar | First letter of agent name |
| Name | Agent name + optional description |
| Provider | `provider` or `audience` field |
| Granted | Relative timestamp |
| Status | `active` / `inactive` chip |
| Action | **Revoke consent** button |

Clicking **Revoke consent** opens a confirmation modal. On confirm the component
calls `DELETE /api/v1/oauth/consents/:consent_id`. If the `consent_id` is not
available in the list payload the UI surfaces an error.

## Revoke All button

A **Revoke all my agents** button appears in the page header. It is disabled
when:
- No admin key is present in `localStorage` (`shark_admin_key`), or
- There are no agents linked to the account (total = 0).

On confirm it calls `POST /api/v1/me/agents/revoke-all` and refreshes both
sections.

## Backend Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/me/agents?filter=created` | Agents created by the current user |
| `GET` | `/api/v1/me/agents?filter=authorized` | Agents the current user has consented to |
| `PATCH` | `/api/v1/agents/:id` | Deactivate an agent (`active: false`) |
| `DELETE` | `/api/v1/oauth/consents/:consent_id` | Revoke a single consent grant |
| `POST` | `/api/v1/me/agents/revoke-all` | Revoke all agents + consents for current user |

## Empty State

When no agents exist in either section, a centered empty state is shown with
an Agent glyph and a suggestion to register via `/agents/new` or
`shark agent register`.

## CLI Parity

There is no CLI command for per-user agent listing. Use the dashboard or the SDK:

```python
from shark_auth import UsersClient
users = UsersClient("https://auth.example.com", token="sk_live_...")
result = users.list_agents("usr_abc", filter="created")
```

## Notes

- The dashboard uses an admin API key (stored in `localStorage`) rather than a
  session cookie, so `useAPI` is reused unchanged.
- The "Authorized by me" section only renders when at least one consent exists.
- Confirmation modals support keyboard dismissal via `Escape`.
