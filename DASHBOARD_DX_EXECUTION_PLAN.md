# Dashboard DX Execution Plan

**Source:** `DASHBOARD_DX_REVIEW.md` (scorecard, decisions, rationale)
**Start:** 2026-04-20
**Ship target:** 2026-04-27 (7 days)
**Progress log:** `DASHBOARD_DX_PROGRESS.md` — **every task state change MUST append a row**
**Status file:** `DASHBOARD_DX_STATUS.json` — current task + machine-readable state
**Commit convention:** `feat(dx): T<id> — <subject>` · `fix(dx): T<id> — <subject>`
**Branch:** continue on `claude/admin-vendor-assets-fix` OR cut new `feat/dashboard-dx` — executor decides on Day 1 and logs choice

---

## Execution Protocol (read first, every session)

1. **Resume:** open `DASHBOARD_DX_STATUS.json` → find `current_task` + `next_action`. If none, pick next `pending` task from table below whose `blocked_by` are all `done`.
2. **Claim:** in `DASHBOARD_DX_PROGRESS.md`, append row: `| <ts> | T<id> | claim | <agent> | - |`.
3. **Work:** follow the task spec (files, endpoints, acceptance). No scope drift.
4. **Verify:** run the task's `verify` block. If fail, log `verify_fail`, fix, re-run.
5. **Commit:** atomic commit with convention above. One task = one commit (unless task explicitly multi-step).
6. **Log done:** append row `| <ts> | T<id> | done | <agent> | <sha> |` AND update `DASHBOARD_DX_STATUS.json`.
7. **Handoff:** if stopping mid-task, append `| <ts> | T<id> | pause | <agent> | <notes+sha> |` with enough context for next agent to resume cold.
8. **Crash recovery:** if `STATUS.json` says `in_progress` but last log row is `claim`, assume interrupted. Re-read task spec, re-diff working tree against last `done` sha, decide resume vs restart.

**Never:**
- Skip progress log entries.
- Commit multiple tasks in one commit.
- Start T<N+1> before T<N> is logged `done` (unless independence noted in Deps).
- Implement beyond the task's acceptance criteria.

---

## Dependency Graph

```
T01 ── P0 credibility (hardcoded lies, Wave A) ── no deps
T02 ── Wave A overview mocks                     ── no deps (parallel with T01)
T03 ── Signing-key rotate wire                   ── no deps (parallel)
T04 ── Backend POST /admin/users                 ── no deps
T05 ── Frontend create-user slide-over           ── T04
T06 ── Org invitations slide-over                ── T05 (shares component)
T07 ── alert() → toast migration                 ── no deps (parallel)
T08 ── Fix 6 silent 404s in organizations.tsx    ── no deps (parallel)
T09 ── Overview hero tile (magical moment)       ── T01, T02
T10 ── Proxy empty-state wizard                  ── T09
T11 ── /admin/get-started page + first-login redir ── T09, T10
T12 ── Session Debugger ship (client-side JWT)   ── no deps
T13 ── Compliance demotion (audit export + GDPR stub) ── no deps
T14 ── Hide 7 remaining phase-gates behind setting── T12, T13
T15 ── Bootstrap token backend + frontend        ── no deps
T16 ── Ship-readiness checklist (Overview card + topbar score) ── T01, T02
T17 ── Help menu + floating ? button + Cmd+K help commands ── no deps
T18 ── Feedback button (screenshot + mailto/issue) ── no deps
T19 ── api_keys scope picker at create           ── no deps
T20 ── webhooks last-5-deliveries panel + retry  ── no deps
T21 ── rbac permission matrix                    ── no deps
T22 ── vault add/remove toast (silent catch fix) ── no deps
T23 ── <CLIFooter/> blanket add to 6 pages       ── no deps
T24 ── Final ship review (re-score, smoke, eng review) ── all
```

Parallelizable waves (safe to run as independent subagents):

- **Wave α** (Day 1): T01 + T02 + T03 + T07 + T08 + T12 + T13 + T22 + T23 — pure-frontend, no shared files outside layout/components.
- **Wave β** (Day 2): T04 (backend blocker) → T05 — sequential.
- **Wave γ** (Day 3): T06 + T19 + T20 + T21 — independent screens.
- **Wave δ** (Day 4): T09 → T10 → T11 — magical moment chain.
- **Wave ε** (Day 5): T14 + T15 + T16 — independent.
- **Wave ζ** (Day 6): T17 + T18 — independent.
- **Wave η** (Day 7): T24 ship review.

**File-lock warning:** T01 and T02 both touch `layout.tsx` / `overview.tsx`. Serialize on same file. Use git branch or directory lock.

---

## Task Specs

Each task has: ID · Files · Backend? · Acceptance · Verify · Subagent prompt.

---

### T01 — Fix hardcoded credibility lies

**Files:** `admin/src/components/layout.tsx`
**Backend:** no
**Lines:** 22 (fake sidebar badges), 103 (v0.8.2), 182-186 (healthy · dev), 284 (nimbus-prod)
**Acceptance:**
- `layout.tsx:284` env chip reads from `adminConfig.env` OR `window.location.hostname` fallback. Drop `nimbus-prod` string entirely.
- `layout.tsx:103` version reads from `/admin/health` `version` field. Loading state = blank until fetched.
- `layout.tsx:22` NAV array — remove `badge: '4'` on device-flow, `badge: '1.2k'` on tokens. Replace with real counts wired from `/admin/stats` (device pending + active tokens) OR delete badge property. Pick: delete.
- `layout.tsx:182-186` footer — status dot + label reads from `/admin/health` (healthy|degraded|down) + `adminConfig.dev_mode ? 'dev' : 'prod'`.
**Verify:** load `/admin`, DOM shows real env + version + no fake badges + live health. `grep -n 'nimbus-prod\|v0.8.2\|1.2k' admin/src/components/layout.tsx` returns empty.
**Subagent prompt (ready to paste):**

> Edit `/home/raul/Desktop/projects/shark/admin/src/components/layout.tsx` per task T01 in `DASHBOARD_DX_EXECUTION_PLAN.md`. Acceptance in plan. Read the 4 line ranges, wire to `adminConfig` (available via prop drill from App.tsx — may need to pass down) and `useAPI('/admin/health')` (import from `./api`). Do not implement features beyond the acceptance. Commit: `fix(dx): T01 — wire env/version/health, remove fake badges`. After commit, append progress row.

---

### T02 — Wave A overview mocks

**Files:** `admin/src/components/overview.tsx`
**Backend:** no
**Lines:** 49 (MOCK.stats.agents fallback), 60-65 (mapTrends — already correct, verify), ~370 (MOCK.attention)
**Acceptance:**
- L49: already derives from `/agents?limit=1` total — verify no MOCK fallback reached.
- L60-65: verify signups-only sparkline renders, others render `null` (no sparkline rather than fabricated).
- L~370 attention panel: replace MOCK.attention with warnings from `/admin/health` response (fields: smtp_unconfigured, expiring_keys, degraded_migration).
**Verify:** `grep -n 'MOCK' admin/src/components/overview.tsx` returns zero lines. Dashboard renders attention warnings from health endpoint.
**Subagent prompt:** identical pattern. Commit: `fix(dx): T02 — wire overview attention + drop MOCK sparkline residue`.

---

### T03 — Signing-key rotate button

**Files:** `admin/src/components/signing_keys.tsx`
**Backend:** no (endpoint `POST /admin/auth/rotate-signing-key` already shipped)
**Acceptance:** rotate button is not disabled. Click → confirm modal → `API.post('/admin/auth/rotate-signing-key')` → toast success + refresh JWKS. Include warning about JWKS cache TTL.
**Verify:** rotate once in dev mode, `/.well-known/jwks.json` shows new kid.
**Commit:** `fix(dx): T03 — wire signing-key rotate button`.

---

### T04 — Backend POST /admin/users

**Files:** `internal/api/admin_user_handlers.go` (new or existing), `internal/api/router.go`, `internal/storage/storage.go` (may need CreateAdmin-origin user method)
**Backend:** yes
**Acceptance:**
- New route `POST /api/v1/admin/users` (admin-key auth).
- Body: `{email, password?, name?, role_ids?, org_ids?}`. Password optional → if omitted, create unverified user + send magic link invite.
- Returns 201 with user shape matching `GET /admin/users/{id}`.
- Audit log entry: `admin.user.create`.
- Smoke test added covering both password + invite paths.
**Verify:** `curl -X POST /api/v1/admin/users -H 'Authorization: Bearer <admin>' -d '{"email":"t@t.io"}'` returns 201 + user. Smoke passes.
**Commit:** `feat(dx): T04 — POST /admin/users handler + invite path`.

---

### T05 — Create-user slide-over (frontend)

**Files:** `admin/src/components/users.tsx` (replace alert at L65)
**Backend:** depends on T04
**Acceptance:**
- Delete `alert()` at L65.
- Build `<CreateUserSlideover/>` component: email (required), password (optional, reveal-once on create), name, role picker (multiselect from `/roles`), org picker (multiselect from `/admin/organizations`).
- Toast on success + refresh list + auto-select new row.
- Respond to `?new=1` URL param by opening slide-over.
- Keyboard shortcut `n` opens slide-over (already wired via `usePageActions.onNew`).
**Verify:** Create a user with password; log in as that user via login form; succeeds. Create a user with invite; check Dev Inbox for magic link email.
**Commit:** `feat(dx): T05 — create-user slide-over`.

---

### T06 — Org invitations slide-over

**Files:** `admin/src/components/organizations.tsx` (invitations tab)
**Backend:** verify endpoints A5 + A6 shipped in T08 fix; if missing, escalate to backend sub-task T06b.
**Acceptance:**
- Remove "use CLI" text.
- `<InviteSlideover/>` (can share base component with T05): email, role-in-org, expiry.
- Pending list shows real invites with resend + revoke buttons.
**Verify:** invite an email, see pending row, click resend → Dev Inbox shows new email.
**Commit:** `feat(dx): T06 — org invitations slide-over`.

---

### T07 — alert() → toast migration

**Files:** `admin/src/components/signing_keys.tsx:96`, `admin/src/components/dev_inbox.tsx:58` (users.tsx:65 covered by T05)
**Backend:** no
**Acceptance:** zero `alert(` calls in admin/src/components (except test fixtures). All replaced with `useToast().error` / `.info`.
**Verify:** `grep -rn 'alert(' admin/src/components` returns empty.
**Commit:** `fix(dx): T07 — remove remaining alert() calls`.

---

### T08 — Fix 6 silent 404s in organizations.tsx

**Files:** `admin/src/components/organizations.tsx` lines 238, 459, 558, 609, 616 (A1 webhooks replay is T20 concern)
**Backend:** verify each endpoint exists; if not, ship missing handler.
**Acceptance:**
- Each of A2 (DELETE org), A3 (POST org roles), A4 (PATCH org), A5 (DELETE invite), A6 (POST invite resend) either succeeds or toasts a structured error.
- No `catch {}` silent swallow.
- Buttons disabled while request in flight.
**Verify:** smoke test covers each endpoint with expected status.
**Commit:** `fix(dx): T08 — organizations silent 404s + error surfaces`.

---

### T09 — Overview hero tile (magical moment trigger)

**Files:** `admin/src/components/overview.tsx`
**Backend:** needs `/admin/proxy/status` (already exists, returns 404 when disabled)
**Acceptance:**
- When `statsRaw.users.total === 0` AND `proxyStatus` is 404/disabled, replace metric cards with `<MagicalMomentTile/>`.
- Tile: headline "Add auth to any app in 60s", subhead with proxy pitch, URL input (upstream), primary button "Configure proxy" → navigates to `/admin/proxy?new=1`.
- Tile hides once either condition flips.
**Verify:** fresh instance shows tile; after enabling proxy, metrics return.
**Commit:** `feat(dx): T09 — overview magical-moment hero tile`.

---

### T10 — Proxy empty-state wizard

**Files:** `admin/src/components/proxy_config.tsx`
**Backend:** uses existing `/admin/proxy/*` (status, simulate) + requires Wave D CRUD from DASHBOARD_GAPS (verify shipped per commit ffdf540)
**Acceptance:**
- When `/admin/proxy/status` returns 404 (disabled), render 3-step wizard: (1) paste upstream URL + select route pattern, (2) configure auth requirement (open|require|optional), (3) open preview in new tab.
- Wizard persists via POST rule endpoints (verify shipped).
- On completion, close wizard, show standard proxy page.
**Verify:** fresh instance → wizard → enter localhost:3000 → creates rule → preview opens.
**Commit:** `feat(dx): T10 — proxy empty-state onboarding wizard`.

---

### T11 — `/admin/get-started` page + first-login redirect

**Files:** `admin/src/components/App.tsx` (route), new `admin/src/components/get_started.tsx`
**Backend:** no
**Acceptance:**
- New route `get-started`. First login (after bootstrap T15 OR no prior admin action) auto-redirects here.
- Combines Overview tile CTA + Proxy wizard in linear flow.
- "Skip to dashboard" link always visible.
- After completing or skipping, sets `sessionStorage.getItem('shark_admin_onboarded') = '1'` and doesn't auto-redirect again.
**Verify:** fresh login → `/admin/get-started`, second login goes to `/admin/overview`.
**Commit:** `feat(dx): T11 — dedicated get-started page + first-login redirect`.

---

### T12 — Session Debugger (client-side JWT)

**Files:** `admin/src/components/empty_shell.tsx` (remove SessionDebugger export), new `admin/src/components/session_debugger.tsx`, update `App.tsx` import
**Backend:** no (JWKS via existing `/.well-known/jwks.json`)
**Acceptance:**
- Paste JWT or session cookie → decode header + payload (pure client, no network).
- Validate signature against JWKS fetched from `/.well-known/jwks.json`.
- Show exp/nbf validity, alg, kid, claims.
- Copy decoded JSON button.
**Verify:** paste a known-valid session JWT → renders valid. Paste a tampered one → renders invalid.
**Commit:** `feat(dx): T12 — ship session debugger (client-side JWT)`.

---

### T13 — Compliance page demotion

**Files:** `admin/src/components/empty_shell.tsx` (remove CompliancePage export), new `admin/src/components/compliance.tsx`, update `App.tsx`, `layout.tsx` NAV (drop ph:9 on compliance)
**Backend:** uses existing audit export + placeholder for GDPR
**Acceptance:**
- Page has two tabs: Audit Export (wires to existing `POST /audit-logs/export`) + GDPR (stub: "request deletion via `shark users delete --email X --purge`" with copyable command + `/permissions/{id}/users` if useful for access review).
- No backend changes required.
**Commit:** `feat(dx): T13 — ship Compliance (audit export + GDPR stub)`.

---

### T14 — Hide remaining phase-gates behind settings toggle

**Files:** `admin/src/components/layout.tsx` NAV rendering, `admin/src/components/settings.tsx` (add toggle), `localStorage` key `shark_show_preview`
**Backend:** no
**Acceptance:**
- 7 remaining stubs (APIExplorer, EventSchemas, OIDCProvider, Impersonation, Migrations, Branding, Tokens, Flow Builder — keep Flow since not ph-gated and already shipped per commit 989c319) — hide from sidebar unless `localStorage.shark_show_preview === '1'`.
- Settings page has toggle "Show preview features (Phase N)".
- Default off.
**Verify:** clean session → sidebar shows only buildable items. Toggle on → stubs reappear.
**Commit:** `feat(dx): T14 — hide unbuilt phase-gates by default`.

---

### T15 — Bootstrap token login

**Files:** backend (`cmd/shark/*.go` startup, `internal/api/admin_bootstrap_handlers.go` new, router), frontend (`admin/src/components/login.tsx`)
**Backend:** yes
**Acceptance:**
- On `shark serve` startup, if no admin session has ever been used (check admin audit log), mint a one-time `bootstrap_token` (random 32-byte). Print to stdout: `Open http://<host>:<port>/admin/?bootstrap=<tok>`.
- New route `GET /admin/bootstrap/consume?token=<tok>` returns a short-lived admin session + redirects to `/admin/get-started`.
- Token single-use, expires 10 min.
- Frontend `login.tsx` reads `?bootstrap=` from URL → POSTs to consume → stores resulting admin key → redirects.
- Fallback: "Where is my key?" small link always visible below input. Expands inline hint with `shark admin-key show` command.
**Verify:** `shark serve` on fresh DB prints URL. Opening URL logs in without pasting a key. Second startup doesn't reprint.
**Commit:** `feat(dx): T15 — bootstrap token login flow`.

---

### T16 — Ship-readiness checklist

**Files:** `admin/src/components/overview.tsx` (add card), `admin/src/components/layout.tsx` (add topbar score badge)
**Backend:** no (derive from existing `/admin/health` + `/admin/stats` + `/admin/config`)
**Acceptance:**
- Overview card "Ready to ship: N/8" with items: SMTP configured, first app created, first user, signing key healthy, webhook test fired, redirect whitelist has ≥1 entry, branding set (stub — just check config), audit reviewed (>0 events).
- Click row → navigates to relevant page.
- Topbar badge: score % (compute from same signals). Click → modal with same breakdown.
**Commit:** `feat(dx): T16 — ship-readiness checklist + score`.

---

### T17 — Help menu + floating ? + Cmd+K help

**Files:** `admin/src/components/layout.tsx` (profile menu), new `admin/src/components/HelpButton.tsx`, `admin/src/components/CommandPalette.tsx`
**Backend:** no
**Acceptance:**
- Profile menu → Help → Docs (link to repo README until docs site ships) / GitHub issues / Changelog (CHANGELOG.internal.md or repo) / Report bug (mailto or GH issue prefill).
- Floating `?` bottom-right, always visible, opens same menu.
- Cmd+K commands added: `help`, `docs`, `bug`, `changelog`, `github`.
**Verify:** Cmd+K → type "bug" → enters issue prefill mailto.
**Commit:** `feat(dx): T17 — help menu + floating ? + Cmd+K help commands`.

---

### T18 — Feedback button

**Files:** overlaps with T17. Can bundle.
**Backend:** no
**Acceptance:**
- "Report bug" opens form: textarea + auto-attached: current page, current version, console errors (last 10), user agent.
- Submit = opens prefilled GH issue OR mailto with all fields. User picks outlet (config in settings).
**Commit:** `feat(dx): T18 — feedback form with auto-context capture`.

---

### T19 — API keys scope picker

**Files:** `admin/src/components/api_keys.tsx`
**Backend:** no (existing POST /api-keys accepts scopes)
**Acceptance:** Create modal adds scope checkbox grid grouped by resource (users, orgs, sessions, audit, webhooks, etc.). Default "full admin". Derive scope list from backend or hardcode match.
**Commit:** `feat(dx): T19 — api-keys scope picker at create`.

---

### T20 — Webhooks last-5-deliveries panel + retry

**Files:** `admin/src/components/webhooks.tsx`
**Backend:** uses `/webhooks/{id}/deliveries` (if shipped) or needs new route. Verify first. A1 replay needs backend.
**Acceptance:**
- Webhook detail panel has "Recent deliveries" table: timestamp, event, status code, response time, retry button.
- Retry calls `POST /webhooks/{id}/deliveries/{delivery_id}/replay` (ship backend if missing — A1 from DASHBOARD_GAPS).
- Failed deliveries highlighted.
**Commit:** `feat(dx): T20 — webhook delivery history panel + retry`.

---

### T21 — RBAC permission matrix

**Files:** `admin/src/components/rbac.tsx`
**Backend:** existing role/permission endpoints + `/permissions/{id}/roles`/`/users` (shipped per commit 6ffaa5c)
**Acceptance:** Grid: rows = permissions, cols = roles. Cell = toggle (role has permission?). Edits call existing assign/unassign endpoints. Optimistic UI.
**Commit:** `feat(dx): T21 — rbac permission matrix grid`.

---

### T22 — Vault silent catch → toast

**Files:** `admin/src/components/vault_manage.tsx`
**Backend:** no
**Acceptance:** Wrap add/remove with try/catch that toasts on failure. No silent swallow. 2 spots per audit.
**Commit:** `fix(dx): T22 — vault add/remove error surfaces`.

---

### T23 — <CLIFooter/> blanket add

**Files:** `admin/src/components/{consents_manage,vault_manage,dev_inbox,proxy_config,settings,authentication}.tsx`
**Backend:** no
**Acceptance:** Each page has `<CLIFooter command="shark <appropriate-verb>"/>` at bottom.
- consents: `shark consents list --user <id>`
- vault: `shark vault providers list`
- dev-inbox: `shark dev inbox tail`
- proxy: `shark proxy rules list`
- settings: `shark admin config dump`
- authentication: `shark auth config show`
**Commit:** `feat(dx): T23 — CLI parity footers on 6 pages`.

---

### T24 — Final ship review

**Files:** none (meta)
**Acceptance:**
- Re-score all 8 DX dimensions. Target: overall ≥ 7/10. Getting Started ≥ 8.
- Run smoke test suite. 100% pass.
- Run `/plan-eng-review` for architecture + test coverage check.
- Update `DASHBOARD_DX_REVIEW.md` scorecard with post-ship numbers.
- Tag release `v0.9.0-dx` (or next appropriate).
- Append final log row: `| <ts> | T24 | SHIPPED | - | <tag> |`.
**Commit:** `chore(dx): T24 — ship v0.9.0-dx`.

---

## Progress Log File Format (`DASHBOARD_DX_PROGRESS.md`)

Initialize with:

```
# Dashboard DX Progress Log

Append-only. Every state change = one row. Never edit prior rows.

| ts (UTC ISO) | task | event | agent | commit/notes |
|--------------|------|-------|-------|--------------|
```

Events: `claim`, `in_progress`, `pause`, `verify_fail`, `blocked`, `done`, `skipped`.

---

## Status File Format (`DASHBOARD_DX_STATUS.json`)

```json
{
  "updated_at": "2026-04-20T00:00:00Z",
  "branch": "claude/admin-vendor-assets-fix",
  "current_task": null,
  "current_agent": null,
  "tasks": {
    "T01": {"status": "pending", "commit": null, "blocked_by": []},
    "T02": {"status": "pending", "commit": null, "blocked_by": []},
    "T03": {"status": "pending", "commit": null, "blocked_by": []},
    "T04": {"status": "pending", "commit": null, "blocked_by": []},
    "T05": {"status": "pending", "commit": null, "blocked_by": ["T04"]},
    "T06": {"status": "pending", "commit": null, "blocked_by": ["T05"]},
    "T07": {"status": "pending", "commit": null, "blocked_by": []},
    "T08": {"status": "pending", "commit": null, "blocked_by": []},
    "T09": {"status": "pending", "commit": null, "blocked_by": ["T01","T02"]},
    "T10": {"status": "pending", "commit": null, "blocked_by": ["T09"]},
    "T11": {"status": "pending", "commit": null, "blocked_by": ["T09","T10"]},
    "T12": {"status": "pending", "commit": null, "blocked_by": []},
    "T13": {"status": "pending", "commit": null, "blocked_by": []},
    "T14": {"status": "pending", "commit": null, "blocked_by": ["T12","T13"]},
    "T15": {"status": "pending", "commit": null, "blocked_by": []},
    "T16": {"status": "pending", "commit": null, "blocked_by": ["T01","T02"]},
    "T17": {"status": "pending", "commit": null, "blocked_by": []},
    "T18": {"status": "pending", "commit": null, "blocked_by": []},
    "T19": {"status": "pending", "commit": null, "blocked_by": []},
    "T20": {"status": "pending", "commit": null, "blocked_by": []},
    "T21": {"status": "pending", "commit": null, "blocked_by": []},
    "T22": {"status": "pending", "commit": null, "blocked_by": []},
    "T23": {"status": "pending", "commit": null, "blocked_by": []},
    "T24": {"status": "pending", "commit": null, "blocked_by": ["T01","T02","T03","T04","T05","T06","T07","T08","T09","T10","T11","T12","T13","T14","T15","T16","T17","T18","T19","T20","T21","T22","T23"]}
  },
  "notes": []
}
```

Valid statuses: `pending`, `in_progress`, `blocked`, `done`, `skipped`.

---

## Subagent Dispatch Template

When dispatching a subagent for a task, include this preamble in the prompt:

```
Execute task T<ID> from /home/raul/Desktop/projects/shark/DASHBOARD_DX_EXECUTION_PLAN.md.
Read the task spec, follow the Execution Protocol.

Before starting:
- Read DASHBOARD_DX_STATUS.json, confirm T<ID> status is `pending` and blocked_by are all `done`.
- Append to DASHBOARD_DX_PROGRESS.md: `| <ts> | T<ID> | claim | <your-id> | - |`.
- Update STATUS.json: current_task=T<ID>, task status=in_progress.

Work:
- Edit only files listed in the task spec.
- Do not scope-creep.
- Run the task's Verify block.

On completion:
- Commit with the exact convention.
- Append progress row with commit sha.
- Update STATUS.json: task status=done, commit=<sha>, current_task=null.

On pause/failure:
- Append progress row with event=pause or verify_fail + detailed notes.
- Update STATUS.json: task status=pending or blocked + notes.

Report back: commit sha OR reason for pause.
```

---

## Handoff / Resume Checklist

When a new session starts (context reset, new agent, new day):

1. `cat DASHBOARD_DX_STATUS.json | jq .` — see where you are.
2. `tail -20 DASHBOARD_DX_PROGRESS.md` — last 20 events.
3. `git log --oneline -20 | grep '(dx)'` — reality check on commits.
4. If STATUS.json says task `in_progress` but progress log shows no `done` — recover:
   - `git diff` to see uncommitted work.
   - Decide: finish or revert. Log decision.
5. Pick next task: first `pending` whose `blocked_by` all `done`.
6. Resume protocol from step 2 in Execution Protocol above.

---

## Decision Record (locked, do not revisit without new review)

1. **Persona primary:** YC founder self-hosting. Optimize here; don't regress backend dev + platform eng.
2. **Target TTHW:** <2 min (Champion tier).
3. **Mode:** POLISH.
4. **Magical moment:** proxy-first onboarding (Overview hero tile + Proxy wizard + get-started page).
5. **Create user:** full slide-over (email, password, roles, orgs) + backend route.
6. **Phase-gate demotion:** Compliance ship now; Session Debugger ship now; 7 others hide behind setting.
7. **Login hint:** bootstrap token flow (primary) + "where is my key" fallback link.
8. **Hardcoded lies:** fix all four (env, version, badges, footer).
9. **Help menu:** profile menu + floating `?` + Cmd+K — all three.
10. **Ship checklist:** Overview card + topbar score badge.
11. **Feedback:** button only, no telemetry.
12. **Theme:** dark-only.
13. **Per-screen polish:** keep RBAC matrix + API keys scope picker + webhook delivery history; skip vault connection auto-discovery + flow-builder drag-drop.
14. **CLIFooter blanket add:** yes, all 6 pages.
15. **alert() → toast:** yes, now.
16. **6 silent 404s:** fix now, toast + disable broken buttons; docs_url when backend provides.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Backend POST /admin/users has existing security model gap | M | H | T04 includes audit entry + role-scoped auth check; pair with eng review |
| Bootstrap token leaks via server logs | M | M | Document: stdout print only on TTY; env var to disable; single-use+10min expiry |
| Magical moment proxy wizard depends on Wave D CRUD endpoints | L | H | Verify commit ffdf540 shipped endpoints on T10 start; escalate if missing |
| 7-day timeline slips by 1-2 days | H | M | T24 ship review may need to cut T19-T21 to post-launch |
| Subagent drift / scope creep | M | M | Execution Protocol "Never" list; per-task acceptance; squash extra work |
| Progress log not updated (human handoff) | M | H | This plan requires it; rejected PRs missing log rows |

---

## Appendix — Files Touched Summary

```
admin/src/components/layout.tsx            T01, T14, T16, T17
admin/src/components/overview.tsx          T02, T09, T16
admin/src/components/signing_keys.tsx      T03, T07
admin/src/components/users.tsx             T05, T07
admin/src/components/organizations.tsx     T06, T08
admin/src/components/dev_inbox.tsx         T07
admin/src/components/empty_shell.tsx       T12, T13, T14
admin/src/components/App.tsx               T11, T12, T13, T14
admin/src/components/proxy_config.tsx      T10
admin/src/components/api_keys.tsx          T19
admin/src/components/webhooks.tsx          T20
admin/src/components/rbac.tsx              T21
admin/src/components/vault_manage.tsx      T22
admin/src/components/consents_manage.tsx   T23
admin/src/components/settings.tsx          T14, T23
admin/src/components/authentication.tsx    T23
admin/src/components/CommandPalette.tsx    T17
admin/src/components/login.tsx             T15
new: admin/src/components/session_debugger.tsx     T12
new: admin/src/components/compliance.tsx           T13
new: admin/src/components/get_started.tsx          T11
new: admin/src/components/HelpButton.tsx           T17
backend new: internal/api/admin_user_handlers.go   T04
backend new: internal/api/admin_bootstrap_handlers.go  T15
backend edit: internal/api/router.go               T04, T08, T15, T20
backend edit: internal/storage/storage.go          T04 (optional)
cmd/shark edit: startup flow                       T15
```

---

**End of plan.** Initialize progress log + status file on first execution.
