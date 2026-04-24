# Proxy v1.5 — Lane D Scope

**Status:** Ready to dispatch. Lane B shipped (HEAD `b589fd0`).
**Estimated time:** ~6h total.
**Owner:** agent (isolation=worktree) dispatched by orchestrator.

---

## Baseline test state (2026-04-24, post-Lane-B)

### Pre-existing stable reds (NOT Lane D scope — separate cleanup lanes)
- `go: internal/api` — `TestAdminConfigShape`, `TestAdminDeleteSessionRevokesAndAudits`, `TestSelfSessionsListAndRevoke`, `TestListAPIKeys`, `TestHandleHostedAssets_ServesBundle`, `TestScenario_AutonomousArchivist`, `TestScenario_SwarmOrchestrator`.
- `pytest smoke`: `test_admin_config_health`, `test_refresh_token_rotation_and_reuse`.

### Pre-existing flaky reds (shared state — Lane D does NOT fix root cause)
Pass in isolation, fail when suite run together due to `dev.db` / port 8080 / `smoke_test.yaml` pollution. Lane D does not attempt fixture isolation rewrite (too broad).
- pytest: `test_admin_org_mgmt`, `test_api_key_crud`, `test_proxy_rules_crud`, `test_rbac_reverse_lookup`.

### Pre-existing reds DISCOVERED this session (Lane D in scope)
1. `go: internal/oauth: TestAuthorizeEndpoint_NotLoggedIn` — expected 401, got 302 redirect to hosted login. Handler drift from some earlier phase.
2. `go: internal/testutil/cli: TestE2EServeFlow` — SQL "no such column: proxy_public_domain". A **fourth** migration dir (`internal/testutil/cli/testmigrations/`) stops at `00012_application_slug.sql` and has completely different naming than `cmd/shark/migrations/`. Handoff gotcha #7 only listed three dirs — this is the 4th.

### Lane B intentional xfails (Lane D unblocks)
- `test_w15_multi_listener_isolation` (YAML per-listener rules gone → port to Admin API).
- `test_w15_standalone_proxy_jwt_verify` (standalone `shark proxy` deprecated → port to embedded proxy or delete).

---

## Work breakdown

### D1 — Tier-gate proxy flow pytest (~1h)
**What:** Build the tier-gated proxy flow test from scratch. (The handoff said to port `stash@{0}` test 67, but all four current stashes contain different content — that blueprint is no longer available. Write from scratch using `docs/proxy_v1_5/contracts/decision_kinds.md` + `docs/proxy_v1_5/api/admin_users_tier.md` + `docs/proxy_v1_5/api/paywall_route.md` as contracts.)
**File:** `tests/smoke/test_proxy_tier_gate.py`.
**Flow:**
1. Signup smoke_user with default tier (likely "free").
2. Create application via admin API; get its slug.
3. Create proxy rule `{path: "/pro/*", require: "tier:pro", app_id: <slug>}` via `POST /api/v1/admin/proxy/rules`.
4. Configure proxy to forward to a toy upstream; reload engine.
5. GET `/pro/something` with free-tier token → expect 302 redirect to `/paywall/{slug}?tier=pro&return=...`. Assert `Location` header.
6. `POST /api/v1/admin/users/{id}/tier` with `{"tier": "pro"}` → upgrade.
7. Re-GET `/pro/something` → 200 from upstream.

### D2 — Tier-gate + paywall render + DPoP + spoof strip pytests (~2h)
**What:** Four new smoke tests covering features that shipped in Lane A + B but never got end-to-end coverage.
**Files:**
- `tests/smoke/test_proxy_paywall_render.py` — hit `/paywall/{slug}?tier=pro&return=...` directly, assert inline HTML template served (200), body contains tier name + return URL + brand tokens (read `applications.branding_design_tokens`).
- `tests/smoke/test_proxy_dpop.py` — create API key / agent token, craft a DPoP proof (reuse helpers in `sdk/python/shark_auth/dpop.py`), send request through proxy with `DPoP:` header, assert upstream receives identity headers. Negative case: missing proof → 401.
- `tests/smoke/test_proxy_header_spoof_strip.py` — client sends `X-Shark-User-ID: evil`, `X-Shark-Scope: admin`, `X-Shark-App-ID: hijack` headers. Proxy MUST strip these before forwarding (spec §6.2). Upstream echoes received headers; assert they're either empty or the authenticated identity — never the client-provided spoofs.
- Extend `test_proxy_tier_gate.py` from D1 with `tier_deny_anonymous` and `tier_match_agent_vs_human` cases.

### D3 — Lifecycle toggle pytest (~1h)
**File:** `tests/smoke/test_proxy_lifecycle.py`.
**What:** Exercise `POST /api/v1/admin/proxy/{start,stop,reload}` + `GET /status`. Flow:
1. `GET /status` → `state: "stopped"` (or `"running"` if default-on — pick based on current lifecycle init default).
2. `POST /stop` → 200, status flips to `stopped`. Proxy listener port refuses.
3. `POST /start` → 200, status `running`. Port listens.
4. Add rule via admin API → `POST /reload` → 200. New rule takes effect (verify by matching a request).

### D4 — YAML migration path pytest (~0.5h)
**File:** `tests/smoke/test_proxy_yaml_deprecation.py`.
**What:** Write a config with legacy `proxy.rules:` block, start `shark serve --config ...`, assert a WARNING message hits stderr containing the substring from `internal/server/server.go` (see `docs/proxy_v1_5/migration/yaml_deprecation.md`). Also assert that **no rules are actually loaded** (empty engine → default-deny).

### D5 — Idempotency pytest (~0.5h)
**File:** `tests/smoke/test_proxy_rules_idempotency.py`.
**What:** POST a rule with a client-specified `id` → 201. Same payload second time → 200 (upsert, no duplicate row). Different payload same `id` → 200 with updated fields. List rules → count unchanged. (Validate against whatever CLI idempotency convention lane E plans — `--id` upsert, exit 2. Storage layer already supports upsert per Lane B migration 00024.)

### D6 — Unxfail w15 tests (~0.5h)
**File:** `tests/smoke/test_w15_advanced.py`.
**What:** 
- `test_w15_multi_listener_isolation`: remove xfail. Instead of YAML `rules:`, after `wait_for_port(p_admin)` POST the rules via `/api/v1/admin/proxy/rules` (once per listener with appropriate `app_id` or listener binding). Call `POST /reload` if required. Verify the 4 existing assertions still hold.
- `test_w15_standalone_proxy_jwt_verify`: delete or convert to embedded-proxy equivalent (user decision — recommend delete since the standalone path is permanently gone per Lane B).

### D7 — Migration dir drift fix (~0.5h)
**Files:** 
- `internal/testutil/cli/testmigrations/` — sync with `cmd/shark/migrations/`. Options:
  - **Option A (recommended):** delete `testmigrations/` entirely and point the `//go:embed` in `harness.go` at `../../../cmd/shark/migrations/*.sql` (or refactor the harness to accept an injected migration FS and thread the canonical one through). Eliminates the 4th dir drift class.
  - **Option B:** copy-sync all 25 migrations into `testmigrations/` and add the dir to handoff gotcha #7 as a 4-dir sync. Worse; drift recurs.
- Verify `TestE2EServeFlow` passes after the fix.

### D8 — Fix `TestAuthorizeEndpoint_NotLoggedIn` (~0.1h)
**File:** `internal/oauth/handlers_test.go`.
**What:** Handler now redirects anonymous OAuth authorize calls to `/hosted/{slug}/login?return_to=...` (302) instead of 401 JSON. Either:
- Update the test to expect 302 + the `Location:` header pointing at `/hosted/` + return_to containing the original authorize URL, OR
- Add a `X-Requested-With: XMLHttpRequest` or `Accept: application/json` branch that still returns 401 JSON for programmatic callers and keeps 302 for browsers. Pick whichever matches the current product intent (check `internal/oauth/handlers.go` for the branch that emits the 302 — the test update is cheaper if the 302 redirect is the deliberate current behavior).

### D9 — Final verify
Run:
- `go test ./... -count=1` — all Lane-D-touched packages green; pre-existing stable reds unchanged.
- `.venv_smoke/bin/python -m pytest tests/smoke/ -q --tb=no` — expect 62+new-tests passing, 0 xfailed, 2 stable reds + occasional flakes only.
- `go build ./...` clean.

---

## Out of scope (do NOT touch)
- `internal/api` stable reds fix — separate lane (need session/api-key/hosted refactor).
- Fixture-isolation rewrite (per-test DB, per-test port, per-test config). Too broad — dedicated lane later.
- Dashboard UI (Lane H), SDKs (Lanes F/G), CLI (Lane E).
- Migrations 00026+ (no new schema in Lane D).

## Hard rules (inherited from Lane B dispatch template)
- `isolation: worktree`. Verify `git merge-base HEAD main` == current `main` tip as step 0.
- Atomic commits. Each subtask D1-D8 gets its own commit. Final verify commit only if a fix lands.
- Commit message footer: `Lane: D` + `Agent: <id>`.
- Agent MAY run `go test` and targeted `pytest tests/smoke/test_<file>.py`. Agent MAY NOT run the full pytest suite across other lanes or force-push / rewrite history.
- Word cap 600-800 on final report.
