# Wave 1.5 — Human↔Agent Linkage + Pre-launch Bug Fixes (Security Layer 3 of 5)

**Budget:** 7h CC (Bug C / proxy-hide moved to Wave 1.7 as coming-soon implementation per user direction) · **Outcome:** customer-fleet cascade revoke + two embarrassing-on-HN fixes (disable-doesn't-revoke, delete-orphan-tokens) · **Run after Wave 1, before Wave 1.7**

## Context — this is one layer of five

Per code audit, SharkAuth has 5 distinct revocation layers. 2 ship today, 3 are missing or buggy. Wave 1.5 ships layer 3 (customer-fleet cascade) AND fixes 2 pre-launch bugs that would surface as top HN comments.

| Layer | Status before Wave 1.5 | Status after Wave 1.5 |
|---|---|---|
| 1. Per-token revoke (RFC 7009) | ✅ ships | ✅ unchanged |
| 2. Per-agent revoke-all-tokens | ✅ ships | ✅ unchanged |
| 3. Per-customer cascade revoke | ❌ missing | ✅ ships |
| 4. Per-agent-type bulk by pattern | ❌ missing | ❌ defer to Wave 1.6 |
| 5. Per-vault-credential cascade | ❌ missing | ❌ defer to Wave 1.6 |

## Pre-launch bug fixes (must ship in this wave)

These bugs would surface as top HN comments. Fix before launch.

### Bug A — Disable-agent does NOT auto-revoke existing tokens (~20 min)

**File:** `internal/api/agent_handlers.go:297` (PATCH `/api/v1/agents/{id}` with `active=false`)

**Current behavior:** sets `active=false`, blocks new token issuance. Existing tokens keep working until they expire normally.

**Bug:** UI text at `admin/src/components/agents_manage.tsx:706` says "Deactivating will prevent new tokens and revoke all active tokens" — which is FALSE. Misleading. Top HN comment material.

**Fix:** when PATCH sets `active=false`, also call `RevokeOAuthTokensByClientID(ctx, agent.ClientID)` and audit `agent.deactivated_with_revocation`. Match the UI promise.

### Bug B — DELETE user creates orphan tokens (~30 min)

**File:** `internal/api/user_handlers.go:133-164` (DELETE `/api/v1/users/{id}`)

**Current behavior:** schema FK cascades agents/sessions, but oauth_tokens records still reference the deleted client_id with `revoked_at IS NULL`. Tokens remain technically valid until expiry. If the agent record is gone but token validation only checks `revoked_at`, the token may still pass introspection.

**Fix:** before `DeleteUser`, list user's agents and call `RevokeOAuthTokensByClientID` for each. Also revoke session tokens for the user. Then proceed with DeleteUser.

These two fixes are pre-conditions for the cascade revoke (Layer 3) being trustworthy. Ship them in the same wave.

### Bug C — Proxy hide MOVED TO WAVE 1.7

User clarified: don't hide proxy from sidebar. Keep the nav entry visible, route to a coming-soon placeholder, preserve the real component code in tree behind a feature flag. This is honest scoping in the UI itself, builds anticipation, doesn't surprise users who were told "proxy ships in v0.2."

See `01d-wave1-7-ui-cleanup-and-impeccable-rebuilds.md` Edit 2b for the coming-soon implementation. Wave 1.5 budget reduces to ~7h (was 7.5h).

Same pattern in Wave 1.7 also covers Compliance (rename → "Exporting logs") and Branding tabs — all three become coming-soon placeholders.

## Your own observation, captured verbatim (still the architectural insight)

> Linking humans to agents lets you have your app protected for rogue humans making agents do bad stuff. My goal is that SharkAuth lets you develop full-stack platforms where you need sandboxed agents or agents inside acting on behalf of users or just general secure unhackable agents.

This insight is correct AND limited. Cascade-by-customer is layer 3 of 5. Other threats need other layers. But customer-fleet cascade is the LAYER MOST UNIQUE TO SHARKAUTH — Auth0 / Clerk / WorkOS don't model agent-creation lineage at all. So Wave 1.5 ships the most-differentiated layer first.

## What ships today (foundation — already done)

Per code audit (`migrations/00010_oauth.sql`, `internal/oauth/exchange.go`):

- ✅ `agents.created_by TEXT REFERENCES users(id)` — every agent has a creator
- ✅ JWT `sub` claim preserves human identity through delegation chain
- ✅ `act` claim nests prior actor — RFC 8693 chain works end-to-end
- ✅ `oauth_consents` table records explicit user→agent authorization
- ✅ `oauth_authorization_codes.user_id` ties auth grants to humans
- ✅ Audit log records `actor_id` per event

## What's missing (close in this wave)

| Gap | File / Surface | Effort |
|---|---|---|
| API filter `?created_by_user_id=` on agent listing | `internal/storage/agents.go`, `internal/api/agent_handlers.go` | ~30 min |
| Endpoint `GET /api/v1/users/{id}/agents` (created + authorized) | `internal/api/user_handlers.go` (new route) | ~45 min |
| Endpoint `GET /api/v1/me/agents` (current user's agents) | same | ~30 min |
| User-facing "My Agents" UI tab | `admin/src/components/me_agents.tsx` (new) | ~90 min |
| Cascade revoke: `POST /api/v1/users/{id}/revoke-agents` | `internal/api/user_handlers.go` + storage helper | ~45 min |
| Audit event `user.cascade_revoked_agents` | `internal/api/audit_handlers.go` | ~15 min |
| Admin-side filter on `agents_manage.tsx` ("filter by creator") | `admin/src/components/agents_manage.tsx` | ~30 min |
| Smoke test for cascade-revoke flow | `tests/smoke/test_cascade_revoke.py` | ~30 min |

**Total: ~6h CC** within Wave 1.5 budget.

## Edit 1 — API filter + endpoints

**File:** `internal/storage/agents.go`

Extend `ListAgentsOpts`:

```go
type ListAgentsOpts struct {
    Limit            int
    Offset           int
    Search           string
    Active           *bool
    CreatedByUserID  *string  // NEW
    AuthorizedByUser *string  // NEW (joins via oauth_consents)
}
```

Update `ListAgents` query to apply filters when set.

**File:** `internal/api/user_handlers.go` (or new file)

```go
// GET /api/v1/users/{id}/agents?filter=created|authorized
// GET /api/v1/me/agents?filter=created|authorized
// Returns: { data: [...], total: N, filter: "created" }
```

Filter modes:
- `created` — agents where `created_by = user_id`
- `authorized` — agents where current `oauth_consents.user_id = user_id` and `revoked_at IS NULL`

Auth: admin API key OR session cookie (user must own the resource).

## Edit 2 — Cascade revoke

**File:** `internal/api/user_handlers.go`

```go
// POST /api/v1/users/{id}/revoke-agents
// Body (optional): { "agent_ids": ["agent-1","agent-2"], "reason": "rogue insider 2026-04-26" }
// If agent_ids omitted, revokes ALL agents created_by user_id AND all consents user_id has granted
// Returns: { revoked_agent_ids: [...], revoked_consent_count: N, audit_event_id: "..." }
```

Server-side action:
1. Soft-delete (set `active=false`) all agents where `created_by = user_id` (or in agent_ids list)
2. Set `revoked_at` on all `oauth_consents` rows where `user_id = ?`
3. Revoke all active tokens for those agents (via existing `RevokeAgentTokens` helper)
4. Write single audit event `user.cascade_revoked_agents` with metadata `{ revoked_agent_count, revoked_consent_count, reason, by_actor }`

**Auth requirement:** admin API key only. Cascade-revoke is destructive — never expose to a session token even for the user themselves (mitigates account-takeover blast radius).

## Edit 3 — UI: "My Agents" tab

**File:** `admin/src/components/me_agents.tsx` (new)

Tab on user profile (or dashboard sidebar entry) showing:

```
My Agents
─────────────────────────────────────────────
Created by me (3)
  · triage-agent      [active]   created 2 days ago    [revoke]
  · email-agent       [active]   created 2 days ago    [revoke]
  · followup-service  [active]   created 1 day ago     [revoke]

Authorized by me (1)
  · third-party-agent (acme.com)  granted 3 days ago    [revoke consent]
```

Header bar: red button "Revoke all my agents" (admin confirmation required).

Empty state: "No agents linked to your account yet. Register one at /agents/new or via `shark agent register`."

## Edit 4 — Admin filter on agents_manage.tsx

**File:** `admin/src/components/agents_manage.tsx`

Add a column "Created by" showing the user (link to user detail). Add a filter dropdown "Filter by creator: [all users / specific user]".

Single line addition: helps admin debug "which agents did user X create?" — same query path the cascade-revoke surface uses.

## Edit 5 — Smoke test

**File:** `tests/smoke/test_cascade_revoke.py` (new)

```python
def test_cascade_revoke_kills_all_agents_and_tokens():
    # 1. Register user U1
    # 2. U1 registers agents A1, A2, A3
    # 3. Each agent gets a token via client_credentials with DPoP
    # 4. Verify all 3 tokens work against /vault/{provider}/token
    # 5. POST /api/v1/users/{U1.id}/revoke-agents (admin auth)
    # 6. Verify all 3 tokens now return 401
    # 7. Verify all 3 agents have active=false
    # 8. Verify audit log has exactly one user.cascade_revoked_agents event
```

This smoke test IS the demo proof: the user can show this passing in the screencast.

## How this changes the pitch

**Before Wave 1.5:**
> SharkAuth is agent-native OAuth with DPoP and delegation chains.

**After Wave 1.5:**
> SharkAuth is the agent security platform for full-stack apps. Every agent action traces to an authorizing human with cryptographic proof at every hop. Revoke the human, every agent they spawned dies in the same transaction. Sandboxed by their human's privilege ceiling, can't escalate, and rogue insider actions are attributable in one query.

The second framing is what HN and YC respond to. Wave 1.5 is what makes the second framing TRUE.

## YC application impact

Add to "What's new about what you make?" section in 07-yc-application-strategy.md:

> Beyond agent primitives, SharkAuth solves the rogue-insider problem for agentic apps: every agent action traces to its authorizing human, and revoking the human cascades to every agent they spawned in a single transaction. No other auth product offers this — Auth0 doesn't model agent-creation lineage, Clerk doesn't model delegation chains, WorkOS doesn't model agent-aware revocation cascade.

## Definition of done for Wave 1.5

- All 5 edits merged to main
- Smoke test `test_cascade_revoke` GREEN
- Smoke suite remains 375+ PASS overall
- Demo screencast for the cascade revoke (separate from Wave 3 — for Twitter thread):
  - Show 3 agents tied to user U1
  - Show admin issuing cascade revoke
  - Show all 3 tokens 401-ing within 1 second
  - Show audit log entry
- Updates to HN post body and YC application copy reflect the new pitch
- README has a new section: "Security model for full-stack platforms"
