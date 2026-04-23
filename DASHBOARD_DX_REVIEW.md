# Dashboard DX Review — Every Screen & Flow

**Date:** 2026-04-20
**Reviewer:** plan-devex-review (gstack) + Claude subagent outside voice
**Scope:** `admin/src/components/**` (React 18 + Vite, embedded in Go binary)
**Mode:** DX POLISH
**Ship target:** 2026-04-27 (7 days)

---

## Target Developer Persona

```
Who:       YC/indie founder self-hosting SharkAuth (primary)
           + backend dev at Series A-B (secondary)
           + platform/DevOps engineer (tertiary)
Context:   First login after `shark init` binary boot, wants admin UI for auth ops
Tolerance: ~5 min to "trustworthy"; abandons on MOCK labels / fake data / 404s
Expects:   Admin key handed by CLI, Linear-density UI, keyboard-first, every API
           action clickable, empty states teach, zero lies
```

## Empathy Narrative (validated with user as accurate)

T+0:00 — Fresh `shark init`. Navigate `localhost:PORT/admin`. Centered Sharky logo, password input, hint "Paste admin API key".
T+0:30 — No "where is my key" link. Fumble back to terminal. Grep history. ~90s wasted.
T+1:10 — Paste. Land on Overview. 6 metric cards zero. Mostly-missing sparklines. Top bar shows env chip **nimbus-prod** (literal lie). Sidebar badges show **device-flow: 4, tokens: 1.2k** (fabricated). Footer **healthy · dev** (hardcoded string).
T+2:00 — Click sidebar OIDC / Impersonation / Compliance — 9 empty "Phase N" shells.
T+3:00 — Click Users → Quick Create "New User" → alert: *"Create user from the dashboard ships in a later phase. Use the CLI"*.
T+5:00 — No onboarding. No ship-readiness. No magical moment. Open Audit, see admin actions, close.
T+7:00 — Don't know what to do next. Likely churn.

## Competitive Benchmark

| Tool | TTHW | Magical moment |
|------|------|----------------|
| Clerk | ~2 min | Paste sign-in component, see first user land |
| Supabase | ~3 min | Auth flow live in SQL editor preview |
| Auth0 | ~15 min | Must configure tenant + app + rule |
| WorkOS | ~10 min | AdminPortal embed in your app |
| **SharkAuth (today)** | **~7-10 min** | **None yet** |
| **SharkAuth (target)** | **<2 min (Champion)** | **Proxy-first onboarding** |

## Magical Moment Specification

**Vehicle:** Overview hero tile + Proxy empty-state wizard + dedicated get-started (all three, shared logic).

**Flow:** New admin lands → Overview detects users=0, proxy=off → giant tile: *"Add auth to any app in 60s. Paste URL →"*. User enters upstream URL → spawns proxy config with sample route → opens preview → redirect flow → authed session appears → Overview tile disappears, metrics start moving.

Closest analog: Vercel's push-to-deploy moment, but for "add auth without code."

## Developer Journey Map

| Stage | Developer does | Friction | Status |
|-------|---------------|----------|--------|
| Discover | `shark init` | Low — README good | OK |
| Install | Run binary | Low — Go single binary | OK |
| **Hello World** | `/admin` → paste key → see zero data | **HIGH** — no hint, no magic | **FIX: bootstrap token + proxy hero** |
| Real Usage | Create user, app, webhook | **HIGH** — alert on user-create, broken org invites, fake data | **FIX: backend POST /admin/users, hardcoded lies, org invites** |
| Debug | Audit log, error messages | MED — 4 alert() calls, 6 silent 404s, no docs_url | **FIX: toast migration, tier-1 errors** |
| Upgrade | Version badge stale, no changelog link | MED — v0.8.2 hardcoded | **FIX: wire version** |

## First-Time Developer Confusion Report (annotated)

```
T+0:00  Login. No "where is my key" link.              [FIX: bootstrap token]
T+0:30  env shows nimbus-prod. Wrong instance?         [FIX: wire from health]
T+0:45  Sidebar: device-flow 4 pending — my queue?     [FIX: real count or drop]
T+1:00  Sidebar: tokens 1.2k — tokens of what?         [FIX: real count or drop]
T+1:30  Overview: six zeros, missing sparklines.       [FIX: magical-moment tile]
T+2:00  Click OIDC Provider → "Coming Phase 8".        [FIX: move to Roadmap group]
T+2:30  Click Impersonation → "Coming Phase 9".        [FIX: hide or Roadmap]
T+3:00  Click Users → + → alert "use CLI".             [FIX: ship backend + UI]
T+4:00  Orgs → invitations → "use CLI" text.           [FIX: invite slide-over]
T+5:00  Click signing-key rotate → native alert.       [FIX: alert→toast]
T+6:00  No Help menu. No Discord. No status page.      [FIX: profile menu + ?]
T+7:00  Gave up. Churn.
```

---

## DX Scorecard

```
+=====================================================================+
|                DX PLAN REVIEW — SCORECARD                           |
+=====================================================================+
| Dimension            | Score  | Target | Gap                        |
|----------------------|--------|--------|----------------------------|
| Getting Started      | 3/10   | 9/10   | -6 (bootstrap, hero tile)  |
| API/CLI/SDK design   | 6/10   | 9/10   | -3 (broken promises)       |
| Error messages       | 4/10   | 8/10   | -4 (alert, silent 404s)    |
| Documentation        | 6/10   | 8/10   | -2 (CLIFooter coverage)    |
| Upgrade path         | 5/10   | 7/10   | -2 (wire version + chlog)  |
| Dev environment      | 7/10   | 8/10   | -1 (already strong)        |
| Community            | 2/10   | 7/10   | -5 (Help menu, Discord)    |
| DX measurement       | 1/10   | 4/10   | -3 (feedback button only)  |
+---------------------------------------------------------------------+
| TTHW current         | ~7-10 min                                    |
| TTHW target          | <2 min (Champion tier)                       |
| Competitive rank     | Needs Work → Competitive → Champion          |
| Magical moment       | Designed: proxy-first onboarding             |
| Product type         | Platform / Admin Dashboard for Auth Provider |
| Mode                 | POLISH                                       |
| Overall DX           | 4.25/10 → target 7.5/10                      |
+=====================================================================+
| DX PRINCIPLE COVERAGE                                               |
| Zero Friction            | GAP (bootstrap token fixes)              |
| Learn by Doing           | PARTIAL (TeachEmptyState used widely)    |
| Fight Uncertainty        | GAP (4 alerts, 6 silent 404s, lies)      |
| Opinionated + Escape     | GOOD (CLI parity footer pattern)         |
| Code in Context          | PARTIAL (CLIFooter ~50% coverage)        |
| Magical Moments          | GAP (none shipped yet)                   |
+=====================================================================+
```

---

## Action Plan — Sequenced for 7-Day Ship

### P0 — Credibility (Day 1, ~3 hours CC)

Block: dashboard looks dishonest on screenshot #1.

- [ ] `layout.tsx:284` — replace hardcoded `nimbus-prod` env chip → read from `adminConfig.env` or hostname
- [ ] `layout.tsx:103` — replace hardcoded `v0.8.2` → wire from `/admin/health` version
- [ ] `layout.tsx:22` — remove fake sidebar badges (`device-flow:'4'`, `tokens:'1.2k'`) OR wire to real counts
- [ ] `layout.tsx:182-186` — wire `healthy · dev` footer to `/admin/health` + `adminConfig.dev_mode`
- [ ] `overview.tsx:49,60-65,~370` — DASHBOARD_GAPS Wave A (strip mocked sparklines, wire MOCK.attention → /admin/health warnings)
- [ ] `signing_keys.tsx:~` — wire rotate button (endpoint exists, currently disabled)

### P0 — Adoption blockers (Day 1-2, ~5 hours CC)

- [ ] **Backend:** ship `POST /admin/users` handler (email + optional password + role + org)
- [ ] **Frontend:** `users.tsx:65` replace alert with full create slide-over (email, password, roles, orgs)
- [ ] **Frontend:** `organizations.tsx` invitations tab — replace "use CLI" with full invite slide-over (shares component with user create)
- [ ] **Frontend:** replace all 4 `alert()` calls with `useToast()`:
  - `users.tsx:65` (superseded by slide-over)
  - `signing_keys.tsx:96`
  - `dev_inbox.tsx:58`
- [ ] **Frontend:** fix 6 A-class silent 404s in `organizations.tsx` (A2-A6 per DASHBOARD_GAPS.md) — toast on failure, disable broken buttons

### P1 — Magical moment (Day 2-4, ~6 hours CC)

- [ ] **Overview hero tile** when users=0 AND proxy=off: "Add auth to any app in 60s. Paste URL →"
- [ ] **Proxy empty-state wizard** — same CTA, deeper surface, 3-step flow (paste URL, pick route, open preview)
- [ ] **Dedicated `/admin/get-started`** — first-login redirect, combines Overview tile + Proxy wizard

### P1 — Sidebar cleanup (Day 2, ~2 hours CC)

- [ ] Ship Session Debugger now (pure client-side JWT decode + JWKS validate — no backend)
- [ ] Demote Compliance from phase-gate (audit CSV export + GDPR "request via CLI" stub)
- [ ] Remaining 7 phase-gated stubs → settings toggle "Show preview features" (default off)

### P1 — Login bootstrap (Day 3-4, ~3 hours CC)

- [ ] **Backend:** on startup, if no admin session ever used, mint one-time `bootstrap_token`, print URL `/admin/?bootstrap=tok_xyz` to server log
- [ ] **Frontend:** if `?bootstrap=` present, consume, provision session, redirect to Overview with "Generate admin key" CTA
- [ ] Fallback: "Where is my key?" link always visible on Login for regression paths

### P1 — Ship readiness (Day 3-4, ~2.5 hours CC)

- [ ] Overview "Ready to ship" checklist: SMTP, first app, first user, signing key, webhook test, redirect whitelist, branding, audit reviewed (8 items, live progress)
- [ ] Topbar health score (0-100) badge, click → modal with breakdown

### P2 — Trust & community (Day 5, ~1.5 hours CC)

- [ ] Profile menu: Help → Docs / Discord / GitHub / Changelog / Report bug
- [ ] Floating `?` button bottom-right + Cmd+K commands: `help`, `bug`, `docs`
- [ ] In-app feedback button: screenshot + comment → mailto or GH issue template with auto-filled version/page/console

### P2 — Per-screen polish (Day 5-6, ~5 hours CC)

- [ ] `api_keys.tsx` — scope picker at create (checkbox grid)
- [ ] `webhooks.tsx` — last-5-deliveries panel with retry
- [ ] `rbac.tsx` — permission matrix (role × permission grid, toggle cells)
- [ ] `vault_manage.tsx` — toast on add/remove fail (replace silent catch)

### P2 — CLI parity (Day 6, ~30 min CC)

- [ ] Add `<CLIFooter>` to: consents, vault, dev_inbox, proxy, settings, authentication

### P3 — Deferred to post-1.0

- Light/dark theme toggle (dropped; dark-only brand)
- Telemetry opt-in (privacy-first; feedback button covers minimum viable)
- Flow Builder drag-drop + conditional branches (already TODO'd)
- Full tier-1 (Elm-style) error UX with docs_url (depends on backend structured error envelope)
- Sign-out confirm modal (topbar:298) — single-click nuke is hostile, but low frequency

---

## NOT in scope (explicitly deferred, with rationale)

- **DX EXPANSION** proposals: not chosen (mode = POLISH)
- **Migration wizards / Impersonation / Branding / OIDC Provider** — ph:9 items, no backend, defer to Phase 9
- **OpenTelemetry / Prometheus integration** — self-hosted, out of DX review scope
- **SDK examples in docs** — separate /plan-devex-review pass on SDK itself
- **Email template editor** — Phase 9 branding feature

## What already exists (reuse, don't rebuild)

- `TeachEmptyState.tsx` — teaching empty-state pattern. Extend to sessions, dev_inbox-before-first-email, device_flow-pending.
- `CLIFooter.tsx` — CLI parity footer. 50% coverage; blanket add to 6 missing pages.
- `CommandPalette.tsx` — Cmd+K with `g X` shortcuts. Missing `g p`, `g f`, `g r` — add.
- `QuickCreate.tsx` — + button menu. 8 items, but 2 items (New User, New Org Invite) break promise. Wire real create flow.
- `toast.tsx` + `useToast` — already present. Migrate 4 alert() calls.
- `CopyField` — IDs reusable. Wrap remaining secret reveals (api_keys, webhooks, vault).
- `PhaseGate` + `empty_shell.tsx` — demotion path already built; just remove entries as backends ship.
- `useKeyboardShortcuts` + `Kbd` component — strong foundation, extend.

---

## TODOS.md candidates

For items deferred to post-1.0, append to TODOS.md:

- **Tier-1 (Elm-style) error UX** — problem+cause+fix+docs_url on every error. Requires backend structured error envelope. Depends on: backend error standard.
- **Telemetry opt-in** — settings → "Help improve" → anonymous page-view + error-rate ping. Depends on: privacy doc + user consent flow.
- **Light theme toggle** — if user demand emerges. Depends on: design audit of component palette.
- **Flow builder drag-drop + conditional branches** — existing TODO in `flow_builder.tsx:16-21`. Depends on: canvas library decision.
- **Bulk export (JSON) on every list page** — users, sessions, audit, apps, keys, webhooks. Depends on: consistent list pagination.
- **Deep-link URL encoding for open slideovers** — `/admin/users/usr_abc/security` pattern. Infrastructure exists (App.tsx:67-81), no page consumes subpath form.
- **Sign-out confirm modal** — topbar:298 single-click session nuke.
- **`g p / g f / g r` keyboard shortcuts** — complete Cmd+K coverage.

---

## Unresolved Decisions

None. All 16 decision points across 4 AskUserQuestion batches answered.

---

## Recommended Sequencing (7-day burndown)

```
Day 1 (Mon 4/21): P0 credibility (hardcoded lies + Wave A + rotate button)  ~4h
Day 2 (Tue 4/22): Backend POST /admin/users + user create slide-over        ~5h
Day 3 (Wed 4/23): Org invite UI + 4 alerts → toast + 6 silent 404s          ~5h
Day 4 (Thu 4/24): Magical moment (Overview hero + proxy wizard)             ~6h
Day 5 (Fri 4/25): Login bootstrap token + ship-readiness checklist          ~5h
Day 6 (Sat 4/26): Help menu + feedback button + per-screen polish           ~6h
Day 7 (Sun 4/27): Session Debugger + CLI parity footer + ship review        ~4h
                  SHIP
```

Total: ~35 hours CC-time.

---

## Outside Voice — Claude Subagent Independent Challenge

Full findings merged into the plan above. Key corrections applied:
1. Hardcoded `nimbus-prod` / `v0.8.2` / fake sidebar badges / `healthy · dev` elevated to P0 (missed in original review)
2. `POST /admin/users` backend gap surfaced → user-create scope adjusted to include backend
3. `/admin/readiness` endpoint needed for checklist (note: partial — `/admin/health` exists, readiness derivation in frontend is acceptable)
4. Post-launch polish (RBAC matrix, webhook history, API keys scope) retained per user's explicit decision
5. Login bootstrap token adopted over "where is my key" link

User retained sovereignty on all cross-model tension points.

---

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | — |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 0 | — | — |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |
| DX Review | `/plan-devex-review` | Developer experience gaps | 1 | ISSUES_FOUND | overall 4.25/10, 16 decisions locked, ~35h to Champion tier |

**CROSS-MODEL:** Outside voice (Claude subagent) surfaced 4 critical misses; all incorporated. User sovereign on 3 tension points (kept RBAC/webhooks/keys polish per batch 4).
**UNRESOLVED:** 0
**VERDICT:** DX REVIEW CLEARED — ready to implement per 7-day sequence above. Eng review recommended before user-create backend ship.
