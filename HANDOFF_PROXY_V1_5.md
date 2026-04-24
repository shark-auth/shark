# PROXY V1.5 — HANDOFF (mid-execution)

**Status:** Session 1 in flight. Lane A complete + merged. Lane B ~60% complete + merged. Lanes D/E/F/G/H not started.
**Main HEAD:** `04a9070 feat(api): paywall inline template + proxy redirect on tier mismatch`
**Branch:** `main`
**Date handoff written:** 2026-04-24

---

## What's merged to main

### Lane A — Foundation ✓ COMPLETE (5 commits)
- `523bce3` refactor(proxy): atomic.Pointer rule swap, remove RWMutex
- `6985f8f` feat(proxy): ReqTier + ReqGlobalRole + Decision kinds
- `cd0145d` feat(proxy): DPoP proof validation in ServeHTTP
- `9196bfa` migration(00023): proxy_rules.tier_match column
- `e8b850e` feat(jwt): bake tier + global roles into Claims
- Plus 3 pre-A commits: identity pkg extraction + proxy/api migrations (595e2c8, cd8de6c, 5d818ef)

### Lane B — Admin API + Lifecycle + Paywall + YAML + M2M ~60% COMPLETE (6 commits)
- `bc2fcd1` feat(proxy): lifecycle Manager (start/stop/reload/status)
- `cb5a398` feat(storage): proxy rules CRUD + user tier helpers
- `dfddcca` migration(00024): proxy_rules.m2m column
- `bca114d` feat(api): proxy lifecycle admin routes (start/stop/reload/status)
- `3638c01` feat(api): user tier + branding design_tokens + YAML import routes
- `04a9070` feat(api): paywall inline template + proxy redirect on tier mismatch

**Extra deviation**: `00025_branding_design_tokens.sql` migration added (spec said reuse `applications.branding_override`, agent added proper column for query-ability). Acceptable.

---

## What's REMAINING in Lane B

### B5 — YAML deprecation (~1.5h)
Not done. Tasks:

**B5a** — `internal/config/config.go`:
- Remove `Rules []ProxyRule` from `ProxyConfig` struct.
- Remove `Resolve()` fan-out of rules into listeners.
- Keep: `proxy.enabled`, `listeners`, `upstream`, `timeout_seconds`, `trusted_headers`, `strip_incoming`.
- Commit: `refactor(config): remove proxy.rules YAML section`

**B5b** — `cmd/shark/cmd/proxy.go`:
- Replace entire file w/ deprecation stub (see PROXYV1_5.md §4.11).
- Update `cmd/shark/cmd/proxy_test.go` to assert error + message.
- Commit: `refactor(cmd): shark proxy deprecated stub`

**B5c** — `internal/server/server.go`:
- Detect legacy `proxy.rules` YAML presence. Print stderr warning.
- Commit: `feat(server): warn on legacy proxy.rules YAML presence`

**B5d** — `cmd/shark/cmd/init.go` template:
- Ensure no `proxy.rules:` section written (verify, remove if present).

### B6b — M2M rule flag + actor_type audit (~0.5h)
Partially done (migration 00024 + storage fields persist m2m column). Remaining:

In `internal/proxy/rules.go`:
- Add `M2M bool` to `Rule` struct + `RuleSpec.M2M`.
- `Evaluate`: if `rule.M2M && identity.ActorType != identity.ActorTypeAgent` → `Decision{Allow: false, Kind: DecisionDenyForbidden, Reason: "rule requires agent (m2m)"}`.
- `parseRequirement` / `compileRule` / `compileSpecs` carry M2M through.
- Tests in `rules_test.go`: m2m+agent → allow; m2m+human → deny.

In `internal/proxy/proxy.go ServeHTTP`:
- Audit: include `actor_type=identity.ActorType` field in logs + audit entries.

Commit: `feat(proxy): m2m rule flag + actor_type audit field`

### B7 — Structured docs (~1h) ⭐ MANDATORY FOR FRONTEND WIRING
Create `docs/proxy_v1_5/` w/ this tree (18 files):

```
docs/proxy_v1_5/
├── README.md
├── api/
│   ├── admin_proxy_rules_db.md
│   ├── admin_proxy_lifecycle.md
│   ├── admin_proxy_rules_import.md
│   ├── admin_users_tier.md
│   ├── admin_branding_design_tokens.md
│   └── paywall_route.md
├── lifecycle/
│   ├── state_machine.md
│   └── reload_behavior.md
├── contracts/
│   ├── decision_kinds.md
│   ├── m2m_rule_flag.md
│   ├── require_grammar.md
│   └── rule_shape.md
├── migration/
│   ├── yaml_deprecation.md
│   ├── 00023_tier_match.md
│   ├── 00024_m2m.md
│   └── 00025_branding_design_tokens.md
└── integration/
    ├── frontend_wiring_notes.md
    └── sdk_surface.md
```

Every doc must include: title/purpose, route/signature, auth required, request shape (JSON schema + example), response shape (success + error), status codes, side effects, frontend hint paragraph.

`frontend_wiring_notes.md` must cover: Proxy tab (list/create/edit/delete rules), lifecycle toggle button + status indicator, user tier UI on user detail, design tokens editor on branding page, paywall preview button/iframe, YAML rules import drag-drop or textarea.

`sdk_surface.md` lists every endpoint needing TS+Python SDK method w/ exact shapes.

Commit: `docs(proxy_v1_5): structured API + contract + migration + integration docs`

### B8 — Final verify (~0.5h)
```
go test ./internal/proxy/... -count=1
go test ./internal/api/... -count=1
go test ./internal/storage/... -count=1
go test ./internal/config/... -count=1
go test ./cmd/shark/cmd/... -count=1
```
Fix failures atomically. `git log --oneline main..HEAD` final enumeration.

---

## Lanes NOT STARTED (Session 2 backlog)

### Lane D — Backend Smoke (~5h)
Per PROXYV1_5.md §5 Session 1 Lane D — runs after Lane B complete.
- Port stash@{0} `smoke_test.sh` test 67 → pytest (1h).
- Tier-gate pytest + paywall render + DPoP proxy + spoof strip (2h).
- Lifecycle toggle pytest (1h).
- YAML migration path pytest (0.5h).
- Idempotency pytest (0.5h).

### Lane E — CLI (~4h)
- `shark proxy {start,stop,reload,status}`
- `shark proxy rules {list,add,show,delete,import}` + idempotency (`--id` upsert, exit 2)
- `shark paywall preview`, `shark branding set`, `shark user tier`, `shark agent register`, `shark whoami`

### Lane F — TS SDK (~2.5h)
Six new modules: `proxyRules.ts`, `proxyLifecycle.ts`, `branding.ts`, `paywall.ts`, `users.ts`, `agents.ts`. Extend `sharkClient.ts`. Tests.

### Lane G — Python SDK (~2.5h, parallel F)
Six new modules: `proxy_rules.py`, `proxy_lifecycle.py`, `branding.py`, `paywall.py`, `users.py`, `agents.py`. Tests.

### Lane H — Dashboard + SKILL.md + First-boot (~3h)
- Reference stash@{0} `proxy_config.tsx` Override Rules UI → rebuild on current main.
- Proxy lifecycle toggle + status indicator.
- Billing/Tier + Design Tokens + Paywall Preview sections.
- SKILL.md at repo root.
- First-boot `[Y/n]` prompt in `shark serve`.

---

## Environment state

- **Main HEAD**: `04a9070`
- **Baseline smoke** (pre-Lane-A): 64 passed, 4 failed
- **Last smoke run** (post-Lane-A, pre Lane B merge): 65 passed, 3 failed
- **Smoke NOT re-run post-Lane-B**. Do this first thing on resume: `pytest tests/smoke/ -q --tb=no`
- **Stable reds** (pre-existing, not v1.5 regression): `admin_config_health`, `refresh_token_rotation`
- **Flaky reds** (intermittent, shared-state pollution): `session_list`, `w15_multi_listener`, `proxy_rules_crud`, `sdk_ergo::agent_session`, `sso_connections_crud`

**Worktrees**: all removed except main.

**Stashes preserved** (don't destroy):
- `stash@{0}` — blueprint for Lane B router.go + storage.go + proxy_config.tsx + smoke_test.sh test 67 (Lane H + Lane D reference)
- `stash@{1}` — MFA TOTP WIP (unrelated)

---

## Resume checklist (next session)

1. `git log --oneline -5` — confirm at `04a9070` on main.
2. `go build -o shark.exe ./cmd/shark` — verify compiles. Should exit 0.
3. `pytest tests/smoke/ -q --tb=no` — baseline smoke. Expect ~65 passed / 3-5 reds (flake range).
4. Read `PROXYV1_5.md` full spec (especially §10 subagent directives + §11 orchestrator checklist).
5. Pick up Lane B remaining: dispatch agent for **B5 + B6b + B7 + B8**. Use PROXYV1_5.md §10.6 agent prompt template.
6. After Lane B fully complete + smoke green → Lane D.
7. Session 2: Lanes E + F + G + H parallel in worktrees.

---

## Key architecture reminders

- **Main HEAD parent chain**: BrandStudio → Lane A (identity pkg, atomic.Pointer, rule kinds, DPoP, migration 00023, JWT bake) → Lane B partial (lifecycle, storage, migration 00024, admin routes, paywall).
- **JWT now carries `tier` + `roles[]` + `scope`**. Proxy evaluates entirely from JWT crypto on hot path (industry standard — Kong/Envoy/Oathkeeper pattern).
- **Identity unified** at `internal/identity/`. Single canonical struct. Both middleware + proxy import.
- **Decision kinds**: `Allow`, `DenyAnonymous`, `DenyForbidden`, `PaywallRedirect`.
- **Proxy lifecycle**: dashboard-toggled child of `shark serve`. Admin API: POST `/api/v1/admin/proxy/{start,stop,reload}`, GET `/status`.
- **Paywall**: `/paywall/{app_slug}?tier=pro&return=<url>` inline template w/ CSS-var injection.
- **YAML deprecation (partial)**: rule YAML fields still parsed. Full deprecation remaining (B5).
- **M2M flag**: column added but rule engine logic + audit field remaining (B6b).
- **Standalone `shark proxy` binary**: STILL EXISTS. Deprecation stub remaining (B5b).

---

## Known gotchas

1. **Worktree harness creates from stale HEAD**. Always verify `git merge-base HEAD main` == current main tip. Rebase if stale. Lane A + B both hit this.
2. **PowerShell cwd leaks**. If Set-Location runs in one PS call, it persists to next call. Always use absolute paths OR `Set-Location C:\Users\raulg\Desktop\projects\shark;` prefix.
3. **Tee-Object -Encoding** not in PowerShell 5.1. Use `Out-File -Encoding UTF8` instead.
4. **Pytest smoke requires `./shark.exe` rebuild** after Go changes. Skip rebuild = test stale code.
5. **Smoke suite flakes on shared state**. `dev.db` + port 8080 + `smoke_test.yaml` all shared across tests. Isolation tests pass, ordered fail. Lane D target: fixture isolation.
6. **Stash@{0} is older than main**. Its parent is `6ffaa5c` (pre-Lane-A, pre-BrandStudio). Read diffs as blueprint, don't checkout.
7. **3 migration dirs must stay synced**: `cmd/shark/migrations/`, `internal/testutil/migrations/`, `cmd/shark/cmd/testdata/migrations/`.
8. **PROXYV1_5.md has been getting reverted** by a linter or external process during the session. Check before assuming latest spec is on disk.

---

## Agent dispatch template (for resume)

Use PROXYV1_5.md §10.6 verbatim. Key rules:
- `isolation: "worktree"` always.
- Verify merge-base == current main HEAD as step 0.
- Atomic commits w/ `Lane: X` + `Agent: <id>` footer.
- Agent forbidden from running smoke/pytest. Orchestrator-only.
- Agent allowed: `go test ./internal/<pkg>/... -count=1` isolated package tests.
- Scope forbid list enforced per lane (see §10.7).
- Word cap 600-800 on final report.

---

## Files referenced in this handoff

- `PROXYV1_5.md` — full spec (may be reverted; re-generate from conversation if missing).
- `MEMORY.md` (at `~/.claude/projects/C--Users-raulg-Desktop-projects-shark/memory/`) — durable preferences.
- `tests/smoke/conftest.py` — pytest fixture scaffolding.
- `.claude/worktrees/` — should be empty except during active lane work.

## Current file inventory deltas (Lane A + Lane B partial)

**New files:**
- `internal/identity/identity.go` + `identity_test.go`
- `internal/proxy/lifecycle.go` + `lifecycle_test.go`
- `internal/proxy/proxy_dpop_test.go`
- `internal/storage/user_tier.go`
- `internal/api/proxy_lifecycle_handlers.go`
- `internal/api/proxy_admin_v15_handlers.go`
- `cmd/shark/migrations/00023_proxy_rules_tier_match.sql` (+ testutil + cmd/testdata)
- `cmd/shark/migrations/00024_proxy_rules_m2m.sql` (+ testutil + cmd/testdata)
- `cmd/shark/migrations/00025_branding_design_tokens.sql` (+ testutil + cmd/testdata)

**Modified files:**
- `internal/proxy/rules.go` — atomic.Pointer, ReqTier, ReqGlobalRole, Decision kinds, (M2M field pending)
- `internal/proxy/proxy.go` — DPoP validation, (paywall redirect pending for tier mismatch), Identity package migration, (actor_type audit pending)
- `internal/proxy/headers.go` — moved Identity to identity pkg
- `internal/auth/jwt/manager.go` — Claims extended w/ Tier + Roles + Scope
- `internal/api/hosted_handlers.go` — paywall handler added
- `internal/api/router.go` — admin routes wired
- `internal/storage/storage.go` — Store interface extended
- `internal/storage/proxy_rules.go` + `_sqlite.go` — CRUD + tier_match + m2m columns
- `internal/storage/branding.go` — design_tokens support

**Untouched (still remaining Lane B work):**
- `internal/config/config.go` — still has `proxy.rules` parsing
- `cmd/shark/cmd/proxy.go` — still standalone command (not deprecated stub)
- `cmd/shark/cmd/init.go` template — verify no `proxy.rules` written
- `internal/server/server.go` — no legacy YAML warning yet

---

End of handoff. See PROXYV1_5.md §5 for full execution plan + §10/§11 for subagent directives + orchestrator checklist.
