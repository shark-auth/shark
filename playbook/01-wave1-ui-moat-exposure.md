# Wave 1 — Dashboard Moat Exposure

**Budget:** 5-6h CC · **Priority:** non-negotiable · **Outcome:** dashboard moat visibility 2/10 → 7/10

## Why this wave is first

Dashboard is the persistent moat surface. Every session sees it. YC partner installing shark sees the dashboard before they see the demo. The agent-native moat (DPoP RFC 9449, delegation chain RFC 8693, agent-aware audit) is functionally complete in the backend but nearly invisible in the UI today. A day-1 user reads "OAuth 2.1 clients for autonomous workloads" and a generic table of `client_id` + scopes — indistinguishable from Auth0.

**Worst offender:** `admin/src/components/get_started.tsx` is a generic OAuth onboarding checklist with zero agent-native content. Day-1 user post-magic-link lands here. This is the largest cliff in the dashboard.

**Hidden win:** `admin/src/components/audit.tsx` already has SSE live stream + `act` claim payload at 5/10. Needs polish, not rebuild.

## Constraints

- Monochrome/square/editable lock per `.impeccable.md` v3 (W17 lock). All edits must respect existing tokens.
- Smoke suite stays GREEN (375 PASS baseline).
- No new heavy dependencies. No chart libraries, no diagram engines.
- API surface unchanged — UI-only edits.

## Edit 1 — Agent detail drawer "Security" tab with DPoP keypair display

**File:** `admin/src/components/agents_manage.tsx` (`AgentDetail` component, around L250+)
**Effort:** ~40 min CC
**Why it moves the needle:** DPoP RFC 9449 is the cryptographic moat. Exposing `jkt` makes token-keypair binding concrete to the user.

**Change:**
- Add a new tab "Security" after the existing "Config" tab in the agent detail drawer.
- Tab content:
  - Field: `DPoP keypair: ECDSA P-256 · key_id: <from server>`
  - Field: `Thumbprint (jkt): <jkt>` with copy button (truncate display to first 8 + last 4 chars)
  - Section: `Rotation history` — collapsible, last 5 rotations with timestamp + actor
- Pull data from existing audit endpoints — no new backend.

**Acceptance:**
- Tab visible in detail drawer for any agent
- `jkt` thumbprint copy button puts full string on clipboard
- No smoke regression
- Visual style matches `.impeccable.md` v3 (mono, square, editable)

## Edit 2 — Audit detail inline delegation chain breadcrumb

**File:** `admin/src/components/audit.tsx` (event detail panel, around L350+)
**Effort:** ~50 min CC
**Why it moves the needle:** Makes RFC 8693 delegation chain visible. Currently buried inside collapsed `act` payload as raw JSON.

**Change:**
- For events where `actor_type === "agent"` and the JWT carries an `act` claim chain, render a horizontal breadcrumb above the existing JSON:
  ```
  [user] → [triage-agent (jkt:zQ7m...)] → [knowledge-agent (jkt:5kL9...)]
  ```
- Each segment is a clickable chip linking to that agent's detail drawer.
- Add a "Policy applied" row below the breadcrumb showing the `may_act` rule that permitted the delegation: `✓ permitted by policy: triage-agent → knowledge-agent`.

**Acceptance:**
- Breadcrumb renders only for agent-actor delegated events
- Clicking a chip navigates to `/agents/:id`
- `may_act` row reads from existing audit event metadata
- audit.tsx moat score 5/10 → 8/10

## Edit 3 — Agent drawer "Delegation Policies" tab

**File:** `admin/src/components/agents_manage.tsx` (`AgentDetail` tabs, around L200+)
**Effort:** ~60 min CC
**Why it moves the needle:** `may_act` enforcement is shark's authorization moat. No UI today = invisible. CLI bouncing kills onboarding flow.

**Change:**
- Add tab "Delegation Policies" after the new "Security" tab.
- Tab content:
  - Read-only summary: `This agent can delegate to: [<list of agent names>] (edit)`
  - Edit mode: checkbox grid of all other agents in the workspace, plus a `scope` input per row for downscoping
  - Save button calls existing `POST /api/v1/agents/{id}/policies` (already implemented per audit findings)
  - Empty state: "No delegation policies set. This agent cannot delegate to any other agent." with a "Configure" button

**Acceptance:**
- New policies persist via existing API
- Saving a policy fires audit event visible in audit.tsx (test cross-edit)
- Empty state renders for fresh agents
- No smoke regression

## Edit 4 — Overview agent-specific attention card

**File:** `admin/src/components/overview.tsx` (attention panel, around L220+)
**Effort:** ~45 min CC
**Why it moves the needle:** Currently the agents KPI is buried between "API keys" and generic metrics. Promote agent autonomy into first-class dashboard real estate.

**Change:**
- Add a new attention card titled "Agent Security" above the auth method breakdown.
- Card body, four metrics:
  - `Active token-exchange grants: <count>` (last 24h)
  - `Delegation chains: <count> (max depth: <n>)`
  - `DPoP binding: <%>` (percentage of agent tokens that are DPoP-bound)
  - `Expired DPoP keys: <count>` — yellow warn if > 0
- Click the card → navigate to `/agents`

**Acceptance:**
- Card visible on dashboard home
- All 4 metrics pull from existing audit + admin endpoints
- Numbers update on the existing SSE live stream
- Style matches the existing attention-panel grid

## Edit 5 — Get Started — SUPERSEDED BY WAVE 1.7

User directive: "redo the /get-started flow with /impeccable craft framing the focus on Agent integrations." This goes deeper than the original 90-min patch this section described.

**The original 90-min patch is now the FALLBACK** if Wave 1.7's full rebuild slips. Keep the spec below as documented fallback only.

**Primary plan:** see `01d-wave1-7-ui-cleanup-and-impeccable-rebuilds.md` Edit 4 — full `/impeccable` rebuild (~6-8h) with three tracks (Agent / Human / Both), agent track as the primary path, preserved auto-completion probes.

**Fallback (if Sunday afternoon hits with Wave 1.7 not started):** the 90-min add-agent-track patch documented below — works against current 12-step `get_started.tsx`, leaves human-OAuth content in place, just appends the agent section.

### Fallback spec (only if Wave 1.7 slips)

- Add a new collapsible section "Agent Onboarding" after the existing "SDK mounted" step.
- Four checklist items, each with inline copy-paste snippet:
  1. **Create an agent** — link `→ /agents/new` + CLI: `shark agent register --name my-agent`
  2. **Generate a DPoP keypair** — Python snippet `prover = DPoPProver.generate()` + curl alt
  3. **Exchange token via delegation (RFC 8693)** — Python snippet using `client.oauth.token_exchange(...)` (will land in Wave 2; until then, raw curl)
  4. **Verify audit trail shows `act` claim + delegation chain** — link `→ /audit?delegation=true`
- Each item completion writes to existing local-storage onboarding state (same pattern as existing checklist)
- Acceptance: section renders below SDK steps; snippets copy-paste-runnable; verification step links to audit page

## Test plan

After all 5 edits:

```bash
# from repo root
pnpm --filter admin test           # admin unit tests
pnpm --filter admin build          # build artifacts
go test ./...                      # backend unchanged but verify
./smoke.sh                         # full smoke suite (375 PASS expected)
```

Manual smoke:
- Fresh `shark serve` first-boot
- Magic-link login
- Open dashboard home → see "Agent Security" attention card
- Get Started → expand "Agent Onboarding" → verify 4 items render
- Create agent → open detail drawer → 4 tabs visible (Credentials, Config, Security, Delegation Policies)
- Run a token exchange via curl → check audit event shows breadcrumb + `may_act` row

## Definition of done for Wave 1

- All 5 edits merged to main
- Smoke suite GREEN (375 PASS)
- Dashboard moat visibility self-rated ≥7/10 (5-second test passes)
- Screenshot of new agent detail drawer suitable for launch post
- Screenshot of audit page with delegation breadcrumb visible

## Fallback if energy collapses

If only 3 of 5 edits ship, prioritize: Edit 5 (Get Started agent track) → Edit 1 (DPoP Security tab) → Edit 2 (delegation breadcrumb). These three carry the most moat-visibility weight per minute of work.
