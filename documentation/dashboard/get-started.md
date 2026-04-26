# Get Started — onboarding checklist

`admin/src/components/get_started.tsx`

The Get Started page is a two-tab guided checklist that helps new admins configure SharkAuth.
State (per-task completion status + active tab) persists in `localStorage` under
`shark_get_started_state_v1`.

---

## Tabs

| Tab | Task list | Description |
|-----|-----------|-------------|
| Proxy + hosted UI | `PROXY_TASKS` | SharkAuth as a reverse proxy; no SDK required |
| SDK | `SDK_TASKS` + `AGENT_TASKS` | Embed auth in your own UI; agent M2M onboarding |

---

## Proxy track (PROXY_TASKS)

Six tasks covering the full proxy setup path:

1. Configure proxy upstream
2. Add a protect rule
3. Pick login methods
4. Set post-login redirect
5. Brand the hosted login page
6. Test the proxy URL

Tasks with `autoCheck` flip to **done** automatically when SharkAuth detects the
corresponding state via `/admin/proxy/status`, `/admin/proxy/rules`, `/admin/apps`,
and `/admin/branding`.

---

## SDK track (SDK_TASKS)

Five tasks for embedding SharkAuth with the React / TypeScript SDK:

1. Create an Application
2. Install the SDK package
3. Mount SharkProvider
4. Wire a Sign-in button
5. Mount the callback route

---

## Agent Onboarding section (AGENT_TASKS) — added Wave 1 Edit 5

A collapsible sub-section rendered **below the SDK tasks** when the SDK tab is active.
The header row is a toggle; clicking it expands / collapses the four agent steps.

Task IDs share the same `tasks` map in localStorage so completion persists across sessions.

### Checklist items

| # | Task ID | Title | Key snippet / link |
|---|---------|-------|--------------------|
| 1 | `agent.register` | Create an agent | `shark agent register --name my-agent` · links to `/agents/new` |
| 2 | `agent.dpop` | Generate a DPoP keypair | `prover = DPoPProver.generate()` + curl alternative |
| 3 | `agent.delegation` | Exchange token via delegation (RFC 8693) | Python `client.oauth.token_exchange(...)` (Wave 2) + curl |
| 4 | `agent.audit` | Verify audit trail shows `act` claim + delegation chain | links to `/audit?delegation=true` |

### Copy-paste snippets

Each task opens the existing `TaskDrawer` (right-side panel) which shows the `cli`,
`code`, and `extraSnippets` fields via the `Snippet` component — same copy-button
pattern as all other tasks.

### Progress tracking

Agent tasks are included in the total / done / skipped progress counter when the SDK
tab is active. The progress bar and `{done}/{total} done` label reflect all SDK +
Agent tasks combined.

---

## Auto-verify

`useAutoChecks()` probes four admin endpoints on mount and exposes boolean flags.
Any task with a matching `autoCheck` key flips to **done** automatically.
Agent tasks do not currently have `autoCheck` keys (manual completion only).

---

## Smoke tests

`tests/smoke/test_w1_edit5_get_started_agent_track.py`

Covers:
- `GET /api/v1/agents` — 200 authenticated, 401 unauthenticated
- `GET /audit` — SPA reachable (200/30x)
- `GET /api/v1/admin/audit?delegation=true` — 401 unauthenticated
