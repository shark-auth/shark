# HANDOFF — SharkAuth Monday Launch Execution

**Date prepared:** 2026-04-26
**Launch target:** Monday 2026-04-27, 9am ET HN submission
**Originating session:** /office-hours with Raúl (founder), 2-3h conversation refining pitch + scope
**Recipient:** fresh agent / fresh CC session executing the playbook

---

## Read this first (in order)

1. **This file.** Full context for what you're about to do, why, and the rules you must follow.
2. `playbook/README.md` — index of all wave files + non-negotiable constraints + discipline directives D1-D7.
3. `playbook/00-design.md` — canonical design doc (locked, APPROVED).
4. `playbook/06-references.md` — Hang Huang transcript, YC/MX deadline math, the founder's existing context.
5. `playbook/07-yc-application-strategy.md` — YC chances, locked 50-word pitch, customer category appendix, rehearsed interview answers.

Do not read every wave file upfront. Read each wave file when you start dispatching a subagent for that wave.

## Your role: ORCHESTRATOR (not coder)

Per D7. Read this section before doing anything else.

**You do NOT write code directly.** Your job:
1. Read each wave file to understand acceptance criteria
2. Construct a subagent prompt using the template (below)
3. Dispatch a Sonnet subagent (`model: "sonnet"`, `isolation: "worktree"` if parallel)
4. Wait for subagent completion
5. Verify diff (`git diff --stat`) — confirm impl + tests + docs all present per D4
6. Run `./smoke.sh` directly via Bash tool (NOT ctx plugin per D2 rule 2)
7. If smoke 375+ PASS, merge. If not, re-dispatch with feedback.
8. Move to next wave

**The Sonnet subagent's job:** receives your prompt, reads the wave file + impacted code, writes impl + tests + docs, reports back. Subagent does NOT run smoke. Subagent does NOT use ctx plugin for smoke.

**Why:** Sonnet is faster + cheaper for well-scoped coding work. The wave files define tasks tightly enough. Orchestrator stays at strategic level: which wave next, scope decisions, integration verification, rolling back if a subagent ships broken work.

---

## Context: who is the founder, what did we decide together

- Raúl, 18, solo, Monterrey NL Mexico. CS second-semester student. Building SharkAuth alone with Claude Code over ~1 month.
- 100,000+ lines of Go + React + Python, RFC-correct (DPoP RFC 9449, RFC 8693 token exchange, agent-aware audit).
- Pre-launch as of session date. Zero users have run the binary. Hang Huang (InsForge CEO, YC P26) is the only external founder engaged so far ("love this a lot, send me launch posts").
- Three deadlines: Monday launch (~24-48h from session start), MX national funding (4 days), YC application (8 days).
- The session refined the pitch through three iterations:
  1. "Agent-native OAuth" → too technical, too narrow
  2. "Agent security platform for full-stack apps" → "platform" yellow flag, "full-stack apps" too broad
  3. **LOCKED:** "Auth for products that give customers their own agents." Five-layer depth-of-defense revocation model.

The founder explicitly directed: ship all by Monday, compressed scope where needed, with three discipline additions (D1 docs, D2 pytest, D3 /impeccable for new components).

---

## What is the moat (so you defend it correctly)

**Customer category:** products that give customers their own agents. ~50-100 named platforms today fit this wedge. See appendix in `07-yc-application-strategy.md`.

**Depth-of-defense (5 layers):**
1. Per-token revoke (RFC 7009 + family cascade) — ✅ ships today (`internal/oauth/revoke.go:15`)
2. Per-agent revoke-all-tokens — ✅ ships today (`internal/api/agent_handlers.go:380`)
3. Per-customer cascade revoke — Wave 1.5 (~6h)
4. Per-agent-type bulk by client_id pattern — Wave 1.6 (~3h, W+1 acceptable)
5. Per-vault-credential cascade — Wave 1.6 (~4h, W+1 acceptable)

**The Claude Code boundary:** SharkAuth does NOT replace agent-orchestration internals (LangChain, CrewAI, Claude Code's subagent dispatch). It handles agent-to-external-resource auth scoped to a specific customer's identity. If you find yourself building "subagent orchestration features" — STOP. That's not the moat.

**The launch story:** "auth for products that give customers their own agents," not "agent-native OAuth." The customer is platform engineers shipping AI products to end-users (Replit, Cursor, Lovable, Bolt, Vapi, Decagon, Zapier AI, etc.).

---

## Wave execution order (Monday launch path, compressed where flagged)

### Sat-Sun pre-launch waves

| # | Wave | File | Budget | Compressed? |
|---|---|---|---|---|
| 0 | Candidate-user list | `05-assignment.md` | 1.5h (founder, not agent) | — |
| 1 | UI moat exposure (5 surgical edits) | `01-wave1-ui-moat-exposure.md` | 5-6h | full |
| 1.5 | Customer cascade + 2 bug fixes | `01b-wave1-5-human-agent-linkage.md` | 7h | full |
| 1.7 | UI cleanup + coming-soon + get-started | `01d-wave1-7-ui-cleanup-and-impeccable-rebuilds.md` | **4-5h compressed** | YES |
| 1.8 | CLI output polish | `01e-wave1-8-cli-output-polish.md` | **1.5-2h compressed** | YES |
| 2 | Python SDK 10 methods | `02-wave2-sdk-killer-methods.md` | 8-13h | full |
| 3 | Demo + vault hop | `03-wave3-demo-command.md` | 7-11h | full |
| 4 | Launch posts + DMs (founder runs) | `04-wave4-launch.md` | 2h | — |

### Compressed scope for Wave 1.7 (Sat-Sun)

Per `README.md` budget reality. Ship these subset items:
- ✅ Edit 1 — Dev-email banner bug fix (~30 min) — DO NOT CUT, top HN comment risk
- ✅ Edit 2 — Coming-soon placeholders for Proxy + Compliance/"Exporting logs" + Branding (~1h) — DO NOT CUT
- 🔻 SKIP Edit 3 — Identity/Settings cleanup (defer to W+1 / Wave 1.7-extended)
- 🔻 FALLBACK Edit 4 — get-started: use 90-min add-agent-track patch (Wave 1 Edit 5 fallback spec) instead of full 6-8h impeccable rebuild. Full rebuild ships Wed-Thu post-launch alongside Video B.

### Compressed scope for Wave 1.8 (Sun)

- ✅ Surface 1 — `shark serve` boot sequence polish (~1.5h) — DO NOT CUT, screencast surface
- 🔻 DEFER Surfaces 2-5 to W+1 (slog handler refinement, per-CLI kv-output, demo command output polish, --verbose mode)

### Mon launch waves

| Wave | What | When | Who |
|---|---|---|---|
| 4 (founder) | Pre-flight smoke + fresh-machine demo test | Sun PM | Raúl |
| 4 (founder) | 30s screencast recording (Video A) | Sun PM | Raúl |
| 4 (founder) | Outreach DMs (10 candidate users from Wave 0 list) | Sun 8-11pm | Raúl |
| 4 (founder) | HN submission + Twitter + Discord | Mon 9am ET | Raúl |
| 4 (founder) | Comment-thread response (first 2h critical) | Mon 9-11am ET | Raúl |
| 4 (founder) | Reddit posts r/golang, r/selfhosted, r/LocalLLaMA | Mon 12-2pm ET | Raúl |

### Tue-Sat post-launch

| Wave | What | When |
|---|---|---|
| Wave 1.7-extended | Identity/Settings cleanup + full impeccable get-started rebuild | Tue-Wed |
| Wave 1.6 | Layers 4+5 (bulk-pattern revoke + vault cascade + DPoP rotation) | Tue-Wed |
| Wave 1.8-extended | CLI surfaces 2-5 polish | Wed |
| Wave 4.5 | Polished 90s Video B recording | Wed PM |
| Wave 4.5 | Real-agent screencasts (Claude Code, OpenClaw) | Tue-Thu |
| Wave 4.5 | Video B post + LinkedIn launch + Twitter pinned | Thu AM |
| YC submission | Application + Video B embed | Sat |

---

## Subagent prompt template (orchestrator uses for every wave dispatch)

Copy this block, fill the `<...>` placeholders from the wave file you're dispatching:

```
You are executing Wave <N> from the SharkAuth Monday-launch playbook.

CONTEXT:
- Repo: C:\Users\raulg\Desktop\projects\shark
- Branch: <current branch from git branch --show-current>
- Smoke baseline: 375+ PASS
- Read first: playbook/HANDOFF.md sections "Discipline directives" + "Specific gotchas"
- Then read fully: playbook/<wave-file>.md

YOUR TASK (verbatim from <wave-file>.md "Definition of done"):
<paste the wave file's "Definition of done" section here>

SPECIFIC EDITS to ship (verbatim from wave file):
<paste each "Edit N" section's "File:" + "Change:" + "Acceptance:" content>

NON-NEGOTIABLE RULES:
- D1: Update documentation/ for any new public surface in same diff
- D2: Write smoke tests in tests/smoke/ for any new logic. DO NOT RUN them yourself.
       The orchestrator runs ./smoke.sh after you report complete.
       NEVER use ctx plugin (mcp__plugin_context-mode_context-mode__*) for smoke.
- D3: For new .tsx components or HTML templates, follow /impeccable workflow
       (frontend-design skill discipline, monochrome/square/editable per .impeccable.md v3)
- D4: Ship BOTH impl files AND test files in the same diff. Don't ship test-only.
- D5: If you spawn parallel sub-subagents, each gets isolation: "worktree"
- D6: Stop after 2 failed attempts at any acceptance criterion. Report back.

WHEN COMPLETE, report in this exact format:
- "Wave <N> implementation complete"
- Files modified (impl): <list>
- Files modified (tests): <list>
- Files modified (docs): <list>
- One-line summary per file
- Any blockers / scope questions deferred to orchestrator
- "Ready for orchestrator smoke + merge"
```

When parallel-dispatching multiple waves (e.g., Wave 1.5 + 1.7 in parallel), each call uses `isolation: "worktree"` per D5. Sequential dispatch can use main repo.

## Discipline directives — apply to EVERY commit

These are non-negotiable. See `README.md` for full text. Summary:

- **D1 — Docs in the same PR.** Any new public surface (endpoint, SDK method, CLI command, dashboard page, config option) requires `documentation/` updates in the same PR. No code-without-docs merges.
- **D2 — Pytest smoke coverage.** Every wave that adds logic adds smoke tests in `tests/smoke/`. **TWO non-negotiable operational rules:**
  1. Pytest is **ORCHESTRATOR-ONLY**. Smoke NEVER runs inside an agent of any kind (subagent, worktree agent, dispatched agent). Per memory rule: pytest from agents kills shark instances on host machine. Only the human-supervised top-level orchestrator runs smoke.
  2. **NEVER run smoke via the ctx plugin** (any `mcp__plugin_context-mode_context-mode__*` tool — `ctx_execute`, `ctx_execute_file`, `ctx_batch_execute`, etc.). Smoke runs via direct Bash invocation only, in the orchestrator's main shell. Ctx plugin sandboxing alters behavior and conflicts with shark instance lifecycle management.

  **Correct invocation** (orchestrator main session, native Bash tool only):
  ```bash
  ./smoke.sh
  pytest tests/smoke/ -v
  ```

  Agent's job: WRITE the test files in `tests/smoke/`. Orchestrator's job: RUN them and verify pass count.
- **D3 — `/impeccable` for new components.** Every new `.tsx` component or HTML template goes through frontend-design / impeccable workflow. Skip allowed only for sub-30-line bugfix patches.
- **D4 — Verify impl + tests both committed.** Per memory, subagents can ship test files without impl. After any merge, run `git diff --stat HEAD~1` and confirm both impl AND test files appear. If only tests landed, BLOCK the merge and re-dispatch.
- **D5 — Worktrees for parallel agents.** >1 parallel agent on same repo MUST use `isolation: "worktree"`. Single agent in main repo is fine. Read-only Explore agents are exempt.
- **D6 — Stop conditions.** Stop and report back if: smoke drops below 375 PASS, a wave's acceptance fails after 2 attempts, an unscoped bug surfaces, scope expands beyond the wave file, or a destructive operation needs authorization.

- **D7 — Sonnet subagent protocol.** ALL coding work is dispatched via Agent tool with `model: "sonnet"`. Orchestrator does NOT write code directly. Subagent does NOT run smoke. See full text in README.md and the subagent prompt template above.

---

## Pre-execution checklist

**Important:** smoke commands here are run by the ORCHESTRATOR (human-supervised top session), NOT by the executing agent. If you ARE the executing agent, you do step 1, 3-5 yourself; you ASK the orchestrator to run steps 2 and 6 and report the result back to you.

```bash
# 1. (agent) Confirm baseline state — read-only
git status                              # should be clean (or aware of uncommitted scratch)
git branch --show-current               # confirm working branch
git log --oneline -5                    # most recent commit context

# 2. (orchestrator only) Smoke baseline
#    Direct Bash, NOT ctx plugin. NEVER from inside an agent.
./smoke.sh                              # must be 375+ PASS before starting any wave

# 3. (agent) Read the active wave file end-to-end before writing any code
# 4. (agent) Identify which existing files will be modified vs new files created
# 5. (agent) For new components: invoke /impeccable per D3

# 6. (orchestrator only) After wave complete:
./smoke.sh                              # direct Bash, must still be 375+ PASS
git diff --stat                         # verify impl + test + docs all present per D1, D2, D4
```

If smoke regresses at ANY wave boundary, STOP and report back per D6. Do not push red.

**If you are the executing agent and you find yourself wanting to run smoke yourself — STOP.** Write the test files. Hand back to the orchestrator with: "Tests written in `tests/smoke/test_X.py`. Please run smoke and confirm pass count before merging." That is the protocol.

---

## Specific gotchas to avoid (sourced from this session's exploration)

1. **Memory note "W18 recovery fixed dev-email tab" is misleading.** Code at `dev_email.tsx:390-406` is unchanged. The bug actually ships. Wave 1.7 Edit 1 fixes it for real. Do not skip thinking "memory says it's fixed."

2. **Proxy nav entry must NOT be hidden.** Founder explicitly directed coming-soon placeholder, NOT hidden. Real proxy code preserved as `proxy_manage_real.tsx` behind `VITE_FEATURE_PROXY=true` env flag. See Wave 1.7 Edit 2b.

3. **Identity/Settings duplicates.** Sessions & Tokens config is in BOTH tabs today. Source of truth is Settings. Identity is read-only mirror. Cleanup is in Wave 1.7 Edit 3 (deferred from compressed scope; ships Tue-Wed).

4. **Get-started.tsx is 1/10 today.** 12 OAuth-generic steps, only 2 agent-relevant. Compressed scope ships 90-min add-agent-track patch; full impeccable rebuild ships Tue-Wed post-launch.

5. **Vault feature ships and works.** `internal/vault/` is real. DPoP-bound retrieval audited. Demo's third-hop fetches from vault (Wave 3) — uses `GET /api/v1/vault/{provider}/token` which already ships per `vault_handlers.go:576-694`.

6. **Human↔agent linkage foundation exists.** `agents.created_by` FK, JWT `sub` preservation, `oauth_consents` table. What's missing: API filter `?created_by_user_id=`, `/api/v1/users/{id}/agents` endpoint, "My Agents" UI, cascade-revoke endpoint. All addressed in Wave 1.5.

7. **Branding lock:** `.impeccable.md` v3 (W17). Monochrome, square corners, editable feel. ALL UI changes respect this lock. No colors except in errors/warnings (where it earns information density).

---

## When in doubt — ask the founder, don't guess

The founder has been deep in this codebase for a month. They know the local conventions, the smoke quirks, the proxy bugs in `INVESTIGATE_REPORT.md`, the W18 recovery context. If a wave file is ambiguous or a memory note conflicts with code reality, ask before guessing.

Specific high-stakes questions to ask before acting:
- "Should I delete <file>?" — never delete without confirmation
- "This bug seems unscoped — should I fix it now or queue for W+1?"
- "Test failed because of <X>. Is this a real failure or a known flake?"
- "Wave file says <thing> but code shows <other thing>. Which is current?"

---

## Coordination protocol with founder

The founder will be:
- Sun afternoon-evening: doing Wave 0 (candidate-user list, 90 min) and recording Video A screencast
- Sun late: sending 10 outreach DMs
- Mon 9am-12pm ET: managing HN front page + Twitter thread + Discord posts
- Tue-Wed: post-launch DM conversion + Video B prep + Wave 1.7-extended

You (the orchestrator) are most useful Sat-Sun dispatching Sonnet subagents through Waves 1, 1.5, 1.7-compressed, 1.8-compressed, 2, 3 in series (or parallel where the wave files are independent — see D5 worktree rule). The subagents' output ships into the screencast Sun PM.

If something pivotal needs founder decision, surface it in your status update. Don't block silently. Don't try to solve scope questions without authorization.

**Parallel-vs-serial decision:** Most waves have implicit dependencies (Wave 2 SDK calls Wave 1.5's new endpoints; Wave 3 demo uses Wave 1.5's cascade and Wave 2's SDK methods). Default to serial. Parallel candidates: Wave 1 + Wave 1.7 (both UI-only, different files), Wave 1.5 + Wave 1.8 (different surfaces). If parallel, each subagent gets `isolation: "worktree"` mandatory.

---

## Final check before shipping

Before each wave's PR merges, verify:

- [ ] Code complete (impl files present per D4)
- [ ] Tests added (smoke coverage per D2)
- [ ] Docs updated (per D1)
- [ ] Smoke suite GREEN at 375+ PASS
- [ ] No new heavy deps added
- [ ] Monochrome/square/editable lock respected (D3 visual review for any new UI)
- [ ] No proxy / compliance / branding routes leaked into production build
- [ ] Wave acceptance criteria from the wave file met

If all six waves ship per spec by Sun PM:
- Founder records Video A
- Sends 10 outreach DMs
- Posts Mon 9am ET

If launch lands well, Tue-Sat ships extended scope + Video B + YC application Saturday.

---

## What done looks like

Monday end-of-day:
- HN post live, ranking visible
- Working binary that someone other than the founder has run successfully (posted publicly with a real handle)
- Smoke suite still 375+ PASS
- README has the killer 10-line snippet, demo screencast embedded, dashboard moat exposure visible in screenshots
- Coming-soon placeholders in proxy/compliance/branding (no broken features visible)
- The 5-layer revocation model: layers 1-3 ship live, layers 4-5 documented as W+1 with tracking issue link

Saturday end-of-week:
- YC application submitted with Video B embedded
- 4-5 of 5 layers shipped (Wave 1.6 lands Tue-Wed)
- Full impeccable get-started rebuild shipped (Wave 1.7-extended)
- Identity/Settings cleanup shipped (Wave 1.7-extended)
- 1+ named integrator publicly using SharkAuth
- Launch retrospective posted Friday (separate news cycle)

That's the win condition. Now go execute.

— Closing handoff. Office hours session ends here.
