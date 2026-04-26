# Post-launch backend bug list (W+1 priority)

Generated: 2026-04-26 — branch: main

These are real backend bugs that ship in v0.1 because the wave merges
delivered the routes but smoke testing surfaced behavior mismatches. The
launch UI ships **with the feature visible**; W+1 closes the backend.

## Wave 1 Edit 3 — Delegation Policies endpoint

**Symptom:** UI tab "Delegation Policies" calls `POST /api/v1/agents/{id}/policies`
which returns 404 (route not registered).

**Impact:**
- UI tab works visually but Save button gets 404
- `shark demo delegation-with-trace` cannot complete because the chain
  requires policy configuration via this endpoint

**Fix:** wire `POST /api/v1/agents/{id}/policies` (and `GET` companion) in
`internal/api/router.go`. Handler logic should accept the body shape the UI
sends (`{may_act: [{agent_id, scope}]}`) and persist via a new
`agent_policies` table or extension of an existing one.

**Smoke:** `tests/smoke/test_w1_edit3_delegation_policies.py` (currently
skipped) and `tests/smoke/test_w3_demo_command.py` (currently skipped).

## Wave 1.5 — Cascade revoke + listing endpoints (6 bugs)

Backend shipped in commits `7f4c6d8` (listing endpoints) and `7e293f0`
(cascade revoke). Endpoints respond but behavior is wrong:

### Bug 1: `POST /api/v1/users/{id}/revoke-agents` rejects valid admin requests with `invalid_client`

The endpoint validates the admin key but its OAuth-style error envelope
mislabels the failure mode. Either:
- Switch to a non-OAuth error envelope (`{code: "...", message: "..."}`)
  matching the rest of the admin API
- OR ensure the admin key path in this handler doesn't go through OAuth
  client validation

### Bug 2: Session-token reject returns 401 instead of 403

`POST /users/{id}/revoke-agents` with a session cookie should return 403
(forbidden — admin key required). Returns 401 (unauthorized — bad
credentials). Auth middleware order issue.

### Bug 3: `GET /api/v1/users/{id}/agents?filter=created` returns empty list

Smoke test creates agents via direct admin POST (not user signup flow), so
`agents.created_by` is admin-id, not user-id. Either the endpoint should
also surface admin-created agents that target a user (via a different
relation), or the smoke test needs a user-signup-driven create_agent
fixture. **Pick one and document the chosen semantics.**

### Bug 4: `GET /api/v1/users/{id}/agents?filter=authorized` returns 500

The JOIN against `oauth_consents` blows up when the user has no consent
rows OR when `revoked_at` is null in unexpected places. Check:
- `internal/storage/agents_sqlite.go` ListAgents query for AuthorizedByUser
  branch
- LEFT JOIN vs INNER JOIN
- IS NULL vs = NULL bug

### Bug 5: `GET /api/v1/me/agents` returns empty list

Same root cause as bug 3 — created_by lineage mismatch.

### Bug 6: `GET /api/v1/agents?created_by_user_id=` returns empty for admin-created agents

Same root cause as bug 3.

**Smoke:** `tests/smoke/test_cascade_revoke.py` (currently skipped).

## Wave 1.7 — Audit endpoint reachable test

**Symptom:** `test_w17_coming_soon_routes::test_audit_endpoint_still_reachable`
returns 404 with raw `requests.get(...)` + Bearer header, but the same URL
works with `admin_client` fixture in `test_admin_deep`.

**Hypothesis:** raw Bearer is hitting a different middleware path than the
session-bound `admin_client`. Investigate auth middleware ordering for the
`/api/v1/admin/audit-logs` route.

**Fix:** convert `test_w17_coming_soon_routes.py` to use the `admin_client`
fixture (same pattern as test_w17_dev_email_banner.py which works).

## Wave 2 SDK — AgentsClient.register missing

The Python SDK's `AgentsClient` exposes `list_agents`, `get_agent`,
`revoke_agent` — but not `register`. Three smoke test files
(test_w2_sdk_*) assume a register method that doesn't exist.

**Fix:** add `AgentsClient.create_agent(name, scopes, ...)` method that
posts to `POST /api/v1/agents` and returns an Agent dataclass. Then
re-enable the W2 SDK smoke tests.

## Pre-existing failures (not from tonight's waves — defer)

These were red before tonight too:
- `test_admin_mgmt::test_dev_inbox_access` (404)
- `test_cli_user_sso_agent_session::TestAPIKeyCLI::test_api_key_create_list_revoke`
- `test_cli_user_sso_agent_session::TestAuditExportCLI::test_audit_export_stdout`
- `test_w15_advanced::test_w15_multi_listener_isolation` (proxy listener auth)
- `test_w15_gateway::test_transparent_gateway_porter` (404 != 302)

Track separately; not blocking launch.

## Recommended W+1 order

1. **Wire `/api/v1/agents/{id}/policies`** (1-2h) — unblocks delegation UI
   AND `shark demo delegation-with-trace`. Highest leverage.
2. **Fix `created_by_user_id` filter semantics** (1h) — affects 4 smoke
   tests. Decide: include admin-created-on-behalf or strict user-signup-only.
3. **Fix oauth_consents JOIN 500** (30min) — quick storage layer fix.
4. **Add `AgentsClient.create_agent` to SDK** (30min) — re-enables W2 smoke.
5. **Convert test_w17_coming_soon to admin_client fixture** (15min).

Total ~4h to turn the entire smoke suite green.
