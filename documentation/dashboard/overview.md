# Overview Dashboard

## Summary

The Overview page (`/`) is the dashboard home. It provides a live snapshot of
users, sessions, MFA adoption, API keys, and agent activity. The right-hand
column contains the **Attention Panel** — a sticky sidebar surfacing items that
warrant immediate review.

---

## Attention Panel

The Attention Panel sits in the 340 px right column. It is sticky so it stays
visible while the user scrolls the main content area.

### Agent Security card (W1-Edit4)

The **Agent Security** card sits at the top of the Attention Panel, above the
health summary. Clicking anywhere on the card navigates to `/agents`.

#### Metrics displayed

| Metric | Description |
|---|---|
| Active token-exchange grants | Count of `token.exchange` audit events in the last 24 h |
| Delegation chains | Count of events carrying an `act_chain` or `oauth.act` delegation; `max depth` = longest chain seen |
| DPoP binding % | Percentage of active agent tokens that carry a `dpop_jkt` / `cnf.jkt` claim |
| Expired DPoP keys | Count of DPoP-bound tokens whose `expires_at` is in the past — shown in amber if > 0 |

#### Data sources (no new backend)

All four metrics are derived from existing endpoints:

| Endpoint | Used for |
|---|---|
| `GET /api/v1/admin/audit-logs?action=token.exchange&from=<24h ago>&limit=200` | Token-exchange grant count; delegation chain analysis (`act_chain`, `oauth.act`) |
| `GET /api/v1/agents?limit=200` | Agent list for token iteration |
| `GET /api/v1/agents/{id}/tokens` | Per-agent token list; DPoP binding (`dpop_jkt` / `cnf.jkt`) and expiry check |

#### Live updates

The `useAgentSecurityMetrics` hook polls every **60 seconds** — matching the
cadence of the existing SSE live stream. No new WebSocket or SSE connection is
opened.

#### Visual style

- Monochrome palette: `var(--surface-*)`, `var(--hairline*)`, `var(--fg*)`, `var(--warn)`
- Square corners (no `border-radius` on the card body rows)
- Metric values in `var(--font-mono)` at 13 px / weight 600
- Expired DPoP key count rendered in `var(--warn, #d97706)` when > 0
- `chip agent sm` badge labelled "live" in the card header

---

## Top-level metrics grid

Six stat tiles across the full width (hidden when the Magical Moment hero is
shown):

| Tile | Source field |
|---|---|
| Users | `stats.users.total` |
| Active sessions | `stats.sessions.active` |
| MFA adoption | `stats.mfa.pct * 100` % |
| Failed logins 24h | `stats.failedLogins24h.count` |
| API keys active | `stats.apiKeys.count` |
| Agents active | `agentsRaw.total` |

---

## Auth method breakdown (30d)

Donut chart populated from `GET /api/v1/admin/stats/trends?days=14`
(`auth_methods` array). Colors: password → `#e4e4e4`, oauth → `#888`,
passkey → `#555`, magic_link → `#3a3a3a`.

---

## Real-time activity stream

Powered by `GET /api/v1/admin/logs/stream` (SSE). Up to 50 events buffered
in memory. Automatic exponential-backoff reconnect on error.
