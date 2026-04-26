# SharkAuth Launch Playbook

Generated: 2026-04-26 · Source: `/office-hours` session · Status: APPROVED

This playbook is the executable plan for the Monday launch. Read in order.
Total budget: **47-65h CC** spread over ~6 days (Sat-Sat). **This is over original 24-48h calendar window. Honest scope conversation needed before Sunday afternoon — see "Budget reality" below.**

Mandatory by Monday launch: ~37-49h (Waves 0, 1, 1.5, 1.7, 1.8, 2, 3, 4)
Wed-Thu post-launch: +8-12h (Wave 4.5 video B + real-agent screencasts)
W+1: +7h (Wave 1.6 — Layers 4+5 of security model)

## Budget reality (read before executing)

Waves 1.7 and 1.8 added significant scope after the original plan. The total mandatory work (~37-49h) exceeds what a solo founder can ship in 24-48h. Two real choices:

**Option A — Push the launch.** Move launch from Monday to Tuesday or Wednesday. Buy 24-48 more hours. Land Wave 1.7 + 1.8 properly. Risk: YC application deadline closes 6 days from launch instead of 8 days, and MX funding deadline may slip.

**Option B — Cut Wave 1.7 / 1.8 scope to fit.** Specific cuts available within each wave (see those files). Realistic compressed Sat-Mon work:
- Wave 0: 1.5h
- Wave 1: 5-6h
- Wave 1.5: 7h
- Wave 1.7 compressed: 4-5h (skip Identity/Settings cleanup, use 90-min get-started patch instead of impeccable rebuild, ship coming-soon placeholders + dev-email fix only)
- Wave 1.8 compressed: 1.5-2h (Surface 1 only — `shark serve` boot polish)
- Wave 2: 6-10h
- Wave 3: 7-11h
- Wave 4: 2h
- Total compressed: ~34-44h still tight but feasible with split sleep schedule

**Recommended position:** Option B for the Monday launch. Option A's date-slip risks YC application timing. Compressed Wave 1.7 still ships the dev-email fix (top HN comment risk) and the coming-soon placeholders (honest scoping). Compressed Wave 1.8 still ships the boot-sequence polish (highest-leverage CLI surface).

The Identity/Settings cleanup and the full get-started impeccable rebuild become **Wave 1.7-extended** — ship Tue-Thu post-launch, alongside Video B recording. This makes "1 week into SharkAuth — here's what shipped this week" a richer Thursday story.

**Drop priority within mandatory wave list (use only if Sunday hits and budget breaks):**
1. Wave 1.7 Identity/Settings cleanup — defer to W+1
2. Wave 1.7 get-started rebuild — fall back to 90-min patch (Wave 1 Edit 5 fallback spec)
3. Wave 1.8 Surfaces 2-5 — keep only Surface 1 (boot polish)
4. Wave 4.5 video B — defer to Friday-Saturday post-launch
5. Wave 1.6 — already deferred to W+1 by design

Do NOT cut: Wave 0, Wave 1 moat exposure, Wave 1.5 Layer 3 cascade, dev-email bug fix in Wave 1.7, coming-soon placeholders in Wave 1.7, Wave 2 SDK, Wave 3 demo, Wave 4 launch posts.

**Drop priority if budget tight:** Wave 4.5 → Wave 3 simplifies to bash → Wave 2 narrows to 3 SDK methods → Wave 1.5 LAST (it unlocks the architectural pitch — protect it).

## Read order

| # | File | Purpose | Budget |
|---|------|---------|--------|
| 0 | [00-design.md](00-design.md) | Full design doc — context, premises, success criteria | reference |
| 1 | [01-wave1-ui-moat-exposure.md](01-wave1-ui-moat-exposure.md) | 5 dashboard UI edits — make moat visible | 5-6h |
| 1.5 | [01b-wave1-5-human-agent-linkage.md](01b-wave1-5-human-agent-linkage.md) | Layer 3: customer-fleet cascade + 2 pre-launch bug fixes | 7h |
| 1.6 | [01c-wave1-6-bulk-pattern-and-vault-cascade.md](01c-wave1-6-bulk-pattern-and-vault-cascade.md) | Layer 4 + 5: bulk-pattern revoke + vault disconnect cascade | 7h W+1 |
| 1.7 | [01d-wave1-7-ui-cleanup-and-impeccable-rebuilds.md](01d-wave1-7-ui-cleanup-and-impeccable-rebuilds.md) | Dev-email fix + coming-soon placeholders + Identity/Settings cleanup + get-started impeccable rebuild | 10-12h |
| 1.8 | [01e-wave1-8-cli-output-polish.md](01e-wave1-8-cli-output-polish.md) | CLI / terminal output polish — beautiful logs, screenshot-shareable | 4-6h |
| 2 | [02-wave2-sdk-killer-methods.md](02-wave2-sdk-killer-methods.md) | Python SDK 5 methods + README snippet | 6-10h |
| 3 | [03-wave3-demo-command.md](03-wave3-demo-command.md) | `shark demo delegation-with-trace` (with vault hop) | 7-11h |
| 4 | [04-wave4-launch.md](04-wave4-launch.md) | Screencast + posts + DMs | 2h |
| 4.5 | [04b-wave4-5-real-agent-screencasts.md](04b-wave4-5-real-agent-screencasts.md) | Polished 90s video B (YC app) + real-agent screencasts | 8-12h opt |
| 5 | [05-assignment.md](05-assignment.md) | **DO THIS FIRST** — 90 min candidate-user list | 1.5h |
| 6 | [06-references.md](06-references.md) | Hang Huang transcript, YC/MX context, resources | reference |
| 7 | [07-yc-application-strategy.md](07-yc-application-strategy.md) | YC chances + application script + video script | reference |
| 8 | [08-launch-scope-cuts.md](08-launch-scope-cuts.md) | What's NOT in launch — proxy deferred, hide nav | 30 min |

## Critical sequence

1. **Wave 0 (assignment, 90 min before any code):** generate `.planning/launch-targets.md`. 10 named MCP/agent builders. Read [05-assignment.md](05-assignment.md). Without this list, the demo lands in a void.
2. **Wave 1 (UI moat exposure, 5-6h):** dashboard moat-visibility 2/10 → 7/10. Highest leverage. Persistent surface. Every session sees it.
3. **Wave 1.5 (human↔agent linkage, 6h):** the SECURITY MOAT. Cascade revoke. Rogue-insider attribution. This unlocks the architectural pitch ("agent security platform for full-stack apps") instead of the technical pitch ("agent OAuth"). Don't skip — it changes the story HN reads.
4. **Wave 2 (SDK killer methods, 6-10h):** README 10-liner that Auth0 cannot match. Technical credibility for HN/MCP Discord readers.
5. **Wave 3 (demo with vault hop, 7-11h):** viral artifact. Now includes vault retrieval at third hop — closes the loop end-to-end. Skip/simplify to bash if energy collapses.
6. **Wave 4 (launch, 2h):** Sunday night. Record screencast on clean install. Post Monday 9am ET. DM Hang Huang.
7. **Wave 4.5 (real-agent screencasts, optional 3-4h):** Claude Code + OpenClaw + Cursor demos for Tue-Fri Twitter. Stretches launch into a 5-day campaign. Skip if energy collapses — primary demo is sufficient.

## Non-negotiable constraints

- **Smoke suite stays GREEN.** Baseline 375 PASS post Phase 6.5. Never merge red.
- **Monochrome/square/editable lock** per `.impeccable.md` v3 (W17). UI changes respect Phase 7 Dashboard rebuild.
- **No new heavy deps.** No chart libs, no diagram engines, no external runtime.
- **Hard scope cuts allowed in this order:** Wave 4 polish → Wave 3 HTML→bash fallback → Wave 2 docstring polish. Wave 1 is non-negotiable; the dashboard is the moat surface.

## Discipline directives (apply to EVERY wave from this point forward)

These rules apply to every change shipped from any wave. Non-negotiable.

### D1 — Documentation must be updated in the same PR

Any change touching public surface (HTTP endpoint, SDK method, CLI command, dashboard page, configuration option, environment variable) MUST update `documentation/` in the same PR.

- New endpoint → new doc page under `documentation/api/`
- New SDK method → docstring + example added to `documentation/sdk/python/` (or `documentation/sdk/typescript/` if TS)
- New CLI command → man-page-style entry in `documentation/cli/`
- New dashboard page → user-flow walkthrough in `documentation/dashboard/`
- Behavior change to existing surface → update existing doc with version note ("Changed in v0.2: …")
- New config option → add to `documentation/config-reference.md`

The PR is not done until docs land. Code-without-docs PRs do not merge.

### D2 — Pytest smoke coverage required for any new logic

`tests/smoke/` is the source of truth for acceptance. Any wave that adds new logic adds at least one smoke test that exercises the happy path AND one negative path (auth fail, scope fail, etc.).

**Two critical operational rules — both non-negotiable:**

1. **Pytest is ORCHESTRATOR-ONLY.** Smoke suite NEVER runs inside an agent (subagent, worktree agent, dispatched agent of any kind). Only the orchestrator (the human-supervised top-level CC session) runs `./smoke.sh` or `pytest tests/smoke/`. Agents that try to run smoke kill shark instances on the host machine and break running state.
2. **NEVER run smoke via ctx plugin** (any `mcp__plugin_context-mode_context-mode__*` tool — `ctx_execute`, `ctx_execute_file`, `ctx_batch_execute`, etc.). Smoke runs via direct Bash invocation only, in the orchestrator's main shell. Ctx plugin sandboxing alters behavior and conflicts with shark instance lifecycle management.

**The correct way to run smoke (orchestrator's main session, native Bash tool):**

```bash
./smoke.sh                              # full suite
pytest tests/smoke/ -v                  # if pytest is preferred shape
pytest tests/smoke/test_specific.py -v  # single test file
```

If a wave adds new tests, the AGENT writes the test files (within `tests/smoke/`) but DOES NOT execute them. The orchestrator runs the suite after the agent reports completion, verifies pass count, and only then merges.

For each wave:
- Wave 1.5 — smoke for cascade-revoke (`test_cascade_revoke.py`), disable-revokes-tokens, delete-user-revokes-tokens
- Wave 1.6 — smoke for bulk-pattern-revoke, vault-disconnect-cascade, dpop-key-rotation
- Wave 1.7 — smoke for dev-email-banner-state, identity/settings-no-duplicate-config, get-started-track-completion-probes
- Wave 1.8 — smoke for boot-sequence-no-stack-trace, slog-format-stable, demo-command-output-stable
- Wave 2 — smoke for each of the 10 SDK methods (mock + real binary)
- Wave 3 — smoke for `shark demo delegation-with-trace` happy path + error path

Every wave's "Definition of Done" includes "smoke suite GREEN at 375+ PASS." Number can grow but never shrink.

### D3 — `/impeccable` workflow for every NEW component or screen

Per user direction during scope refinement: every new component or screen ships through `/impeccable` craft discipline (frontend-design skill workflow).

Mandatory for: new `*.tsx` components in `admin/src/components/`, new HTML templates (e.g., demo report), new dashboard pages, any user-facing surface.

Optional for: bugfix patches under 30 lines that don't introduce new surfaces, internal refactors, backend-only changes.

When dispatched, the workflow:
1. Skill invocation (`/impeccable` or `frontend-design`)
2. Provide design intent (audience, purpose, constraints — including monochrome/square/editable lock)
3. Skill produces design proposal — review/iterate
4. Generate component code following `.impeccable.md v3` lock
5. Verify against existing dashboard tokens
6. Smoke + visual review

### D4 — Verify all impl files committed (per memory feedback)

Memory rule: subagents can ship test files without their implementation. Always verify. After any agent merge:
- Run `git diff --stat HEAD~1` against the merge
- Confirm both impl files (`internal/`, `admin/src/`, `sdk/`) AND test files (`tests/smoke/`) appear
- If only tests landed without impl, BLOCK the merge and re-dispatch

### D5 — Worktrees for parallel agents

Memory rule: >1 parallel agent on same repo MUST use `isolation: "worktree"` to prevent collisions on smoke/dist/DB. Single agent in main repo is fine.

If the executing agent dispatches sub-agents in parallel, each sub-agent gets a worktree. Read-only Explore agents are exempt (no writes).

### D6 — Stop conditions

Stop and report back to the orchestrator (don't push through) if:
- Smoke suite drops below 375 PASS at any point
- A wave's acceptance criteria can't be met after 2 attempts
- A bug surfaces that wasn't in the playbook (the full security model, the dev-email bug were both surfaced this way — flag promptly)
- The scope expands beyond what's documented in the wave file
- You hit a destructive-operation question that wasn't pre-authorized (deleting branches, force-push, dropping tables, removing tracked files)

### D7 — Sonnet subagent protocol for ALL coding work

**The orchestrator (top-level CC session) does NOT write code directly.** Every coding task — bugfixes, new features, refactors, UI changes, SDK methods, smoke test writing — is dispatched to a Sonnet subagent via the Agent tool with a clear directive.

**Roles:**

- **Orchestrator** (top-level CC session): reads playbook, dispatches Sonnet subagents, verifies their output, runs smoke (per D2), reviews diffs, merges PRs, coordinates between waves. Does NOT write code.
- **Sonnet subagent** (dispatched via Agent tool with `model: "sonnet"`): receives a clear directive, reads the relevant wave file + impacted code, writes code (impl + tests + docs per D1/D2/D4), reports back. Does NOT run smoke. Does NOT use ctx plugin for smoke.

**Why this rule:**
- Sonnet is faster + cheaper than Opus for well-scoped coding work, and the wave files define each task tightly enough that Sonnet can execute without strategic-level reasoning
- Orchestrator stays at strategic level (which wave next, scope decisions, integration verification, rolling back if a subagent ships broken work)
- Cleaner audit trail: each wave's code change is one subagent dispatch, traceable back to one prompt

**Dispatch protocol:**

```
Agent tool call:
  description: "Wave 1.5 — customer cascade revoke"
  subagent_type: "general-purpose"  (or specialized agent if applicable)
  model: "sonnet"
  isolation: "worktree"  (if running in parallel with other agents)
  prompt: <see HANDOFF.md "Subagent prompt template">
```

**Subagent prompt template** (orchestrator constructs per wave):

```
You are executing Wave <N> from the SharkAuth Monday-launch playbook.

CONTEXT:
- Repo: C:\Users\raulg\Desktop\projects\shark
- Branch: <current branch>
- Smoke baseline: 375+ PASS
- Read first: playbook/HANDOFF.md (full context + discipline directives D1-D7)
- Then read: playbook/<wave file>

YOUR TASK:
<Acceptance criteria from the wave file's "Definition of done" section>
<Any specific edits the wave file enumerates>

RULES (non-negotiable):
- Implement BOTH impl files AND smoke tests in tests/smoke/ AND documentation/ updates per D1+D2+D4
- DO NOT run smoke yourself — write tests, hand back to orchestrator who runs them
- DO NOT use ctx plugin (mcp__plugin_context-mode_context-mode__*) for smoke
- Follow .impeccable.md v3 lock for any UI work (monochrome/square/editable)
- For new components, invoke /impeccable workflow per D3
- If you cannot meet acceptance after 2 attempts, STOP and report back

WHEN COMPLETE, report:
- List of files modified (impl, tests, docs)
- One-line summary per file of what changed
- Any blockers encountered or scope questions deferred
- "Ready for orchestrator smoke + merge"
```

**Parallel agents → worktrees mandatory** (D5). If the orchestrator dispatches two Sonnet agents simultaneously (e.g., Wave 1.5 + Wave 1.7 in parallel), each gets `isolation: "worktree"`. Single-agent serial execution can use the main repo.

**Verification cycle (orchestrator's job between subagent dispatches):**

1. Subagent reports complete
2. Orchestrator reads `git diff --stat` — verifies impl + tests + docs per D4
3. Orchestrator runs `./smoke.sh` directly via Bash (not ctx plugin per D2 rule 2)
4. If smoke 375+ PASS and diff verified, merge to main
5. Update progress, dispatch next wave's subagent

**The orchestrator NEVER:** writes code, runs smoke from agent context, dispatches via ctx plugin, merges without diff+smoke verification, lets a subagent ship test-only without impl.

**The subagent NEVER:** runs smoke, uses ctx plugin for build/test, dispatches its own sub-subagents without orchestrator authorization, merges its own work.

## Demand evidence at start

Zero users have run the binary as of 2026-04-26. The launch posture is honest: ship the artifact, then earn demand. The assignment exists to convert "zero" into "one named integrator running shark Sunday night."

## Targets

- Monday 12:00 PT: launch live, demo video posted
- Monday EOD: ≥1 named integrator with working setup, posted publicly
- Tuesday EOD: ≥3 GitHub issues from real users
- 4 days: MX funding application uses launch traction story
- 8 days: YC application embeds demo video + first-week numbers

## Skill chain after this

When you're ready to plan execution detail per wave, use:

- `/plan-eng-review` — lock architecture, edge cases, tests, performance per wave
- `/executing-plans` — run a wave with review checkpoints
- `/ship` — when wave 1 lands, create the PR with smoke proof

The canonical design doc is also at `~/.gstack/projects/shark-auth-shark/raul-main-design-20260426-024944.md` — those skills find it automatically.
