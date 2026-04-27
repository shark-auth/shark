# Pytest Port + Concurrency Plan (2026-04-26)

**Origin:** `smoke_test.sh` (3605 lines · ~80 sections · authoritative pre-W17 coverage).
**Target:** `tests/smoke/*.py` (58 files · post-W17 + post-yaml-removal).
**Two problems to fix:**

1. **Hallucinated ports.** Several pytest files were ported by agents that invented behaviors instead of faithfully porting the .sh. Need an audit + re-port.
2. **Serial execution.** Single shark instance, single worker, sequential requests. Doesn't load-test the server. Real concurrent shape unknown.

---

## Part 1 — Section-by-section port audit

Methodology: read each `section "<name>"` block in `smoke_test.sh`, find the matching pytest file, classify:

- ✅ **Faithful port** — pytest exercises the same surface
- ⚠️ **Drifted** — partial coverage, missing edge cases the .sh tested
- ❌ **Hallucinated/missing** — pytest file claims to test it but doesn't, OR no pytest equivalent exists
- 🔻 **W17-deprecated** — section was YAML/--dev-flag-specific and was correctly dropped per W17

Audit needs to be done by an agent that READS each .sh section + compares the pytest equivalent. Output format:

| .sh section | Pytest file (if any) | Status | Re-port effort |
|---|---|---|---|
| 1. bootstrap fresh DB | conftest.py spawn | ✅ | — |
| 2. signup issues JWT | test_auth.py::test_signup_login_flow | ✅ | — |
| 3. middleware dual-accept | test_auth.py::test_dual_accept_middleware | ✅ | — |
| 4. JWKS | test_oauth_advanced.py::test_jwks_es256 | ⚠️ | 30 min — only ES256, .sh tested RS256+ES256 both |
| 5. user revoke | test_user_sessions.py::test_session_list_and_revocation | ⚠️ | 20 min — .sh also tested check_per_request behaviour |
| 6. admin revoke gated | (likely missing) | ❌ | 30 min — .sh tested non-admin token blocked from /admin/sessions/revoke |
| 7. key rotation | test_admin_deep.py? | unknown | needs read |
| 8. apps CLI | test_cli_ops.py | ⚠️ | needs read |
| 9. admin apps HTTP | (search) | unknown | — |
| 10. redirect allowlist + magic-link | test_auth.py::test_redirect_allowlist + test_dev_email.py | ⚠️ | 20 min — .sh combined the two flows; pytest split them |
| 11. org RBAC | test_org_rbac.py | needs read | — |
| 12. audit logs | test_stats_audit.py | needs read | — |
| 13. regression | (drop — ad-hoc per-version) | 🔻 | — |
| 14. admin system endpoints | test_admin_deep.py | needs read | — |
| 15. user list filters | test_admin_deep.py::list_filters? | needs read | — |
| 16. sessions self-service | test_user_sessions.py | ⚠️ | 30 min — .sh covered logout + re-login flow, not just session list |
| 17. admin sessions | test_user_sessions.py::test_admin_session_filtering | ✅ | — |
| 18. stats + trends | test_stats_audit.py | needs read | — |
| 19. webhooks CRUD + delivery | (likely drift) | ❌ | 60 min — .sh exercised .data response shape + replay |
| 20. API key CRUD | test_cli_user_sso_agent_session.py::TestAPIKeyCLI? FAILING | ❌ | 30 min |
| 21. user CRUD admin | (partial via test_cascade_revoke fixtures) | ⚠️ | 30 min — full CRUD smoke missing |
| 22. dev inbox | test_dev_email.py | ✅ | — |
| 23. password change | test_user_sessions.py::test_password_change | ⚠️ | 20 min — .sh tested email-verify gate FIRST |
| 24. SSO connections CRUD admin | (likely missing) | ❌ | 45 min |
| 25. admin config + health | test_admin_deep.py? | needs read | — |
| 26. AS metadata RFC 8414 | test_metadata.py | ✅ | — |
| 27. OAuth tables exist | (spawn-time check) | 🔻 | — |
| 28. AS metadata advanced | test_metadata.py | ⚠️ | 15 min — .sh tested specific advanced fields |
| 29. agent CRUD admin | test_oauth_flows.py::test_agent_crud | ✅ | — |
| 30. client_credentials grant | test_oauth_flows.py::test_client_credentials_grant | ✅ | — |
| 31. auth code + PKCE | test_oauth_flows.py::test_auth_code_pkce_flow + test_agent_flow_pkce.py | ✅ | — |
| 32. PKCE enforcement | test_oauth_advanced.py::test_pkce_enforcement | ✅ | — |
| 33. refresh token rotation | test_oauth_advanced.py::test_refresh_token_rotation_and_reuse + test_agent_flow_refresh_rotation.py | ✅ | — |
| 34. device flow | test_oauth_advanced.py::test_device_flow + test_agent_flow_device.py | ✅ | — |
| 35. token exchange RFC 8693 | test_oauth_advanced.py::test_token_exchange + test_agent_flow_token_exchange.py + W2 method 2 | ✅ | — |
| 36. DPoP RFC 9449 | test_oauth_advanced.py::test_dpop_surface + test_agent_flow_dpop.py | ✅ | — |
| 37. token introspection | test_oauth_advanced.py::test_introspection_revocation | ✅ | — |
| 38. token revocation | test_oauth_advanced.py::test_introspection_revocation | ✅ | — |
| 39. DCR RFC 7591 | test_oauth_advanced.py::test_dcr_lifecycle + test_agent_flow_dcr.py | ✅ | — |
| 40. resource indicators RFC 8707 | test_oauth_advanced.py::test_resource_indicators | ✅ | — |
| 41. ES256 JWKS | test_oauth_advanced.py::test_jwks_es256 | ✅ | — |
| 42. consent management | test_vault_proxy_flows.py::test_consent_management (currently ERRORS) | ⚠️ | 30 min — fixture broken, re-port |
| 43. vault provider CRUD | test_vault_proxy_flows.py | ⚠️ | 30 min — verify port faithful |
| 44. vault templates discovery | (likely missing) | ❌ | 30 min |
| 45. vault connect flow session auth | test_vault_proxy_flows.py? | ⚠️ | 30 min |
| 46. agent token retrieval | test_w3_demo_vault_hop.py + W2 method 9 | ✅ | — |
| 47. vault connections list | (partial) | ⚠️ | 20 min |
| 48. audit events for vault ops | (partial) | ⚠️ | 20 min |
| 49. proxy admin endpoints proxy disabled | test_proxy_*.py | needs read | — |
| 50-54. auth flow CRUD + dry-run + signup gates + flow runs | (likely missing) | ❌ | 90 min |
| 55. webhook delivery replay | (likely missing) | ❌ | 30 min |
| 56-58. admin org CRUD/invitation/MFA-disable | (likely missing) | ❌ | 90 min |
| 59. audit actor_type filter | test_w1_edit2_audit_breadcrumb.py? FAILING | ⚠️ | 20 min |
| 60. failed_logins_24h accuracy | test_stats_audit.py? | ⚠️ | 30 min |
| 61. MFA enabled-vs-verified count | (likely missing) | ❌ | 30 min |
| 62. flow test metadata pass-through | (missing) | ❌ | 20 min |
| 63-66. user list filters + admin vault + admin consents + RBAC reverse | (mixed) | ⚠️ | 60 min |
| 67-68. proxy rules + status shape | test_proxy_rules_idempotency.py + test_proxy_lifecycle.py | needs read | — |
| 69. audit log CSV export | test_cli_user_sso_agent_session.py::TestAuditExportCLI FAILING | ❌ | 30 min |
| 70. POST /admin/users T04 | (likely covered transitively) | ⚠️ | 15 min — explicit smoke missing |
| 71. bootstrap token consume T15 | (W17-related; check) | needs read | — |
| 72-73. W15 multi-listener proxy + standalone JWT | test_w15_advanced.py + test_w15_gateway.py FAILING | ⚠️ | 60 min |
| 74. branding + email templates + integration_mode + per-app branding + welcome idempotent + logo upload + integration_mode validation | (mostly missing post-W1.7-coming-soon) | 🔻/❌ | 120 min — partial since branding/compliance are now coming-soon |
| 75. hosted pages shell | (likely missing) | ❌ | 45 min |
| 76. SDK integration example app | test_sdk_integration.py | needs read | — |
| F4. token exchange delegation | test_oauth_advanced.py::test_token_exchange | ✅ | — |
| F5. DPoP full flow | test_oauth_advanced.py::test_dpop_surface | ✅ | — |
| 70 (transparent gateway) | test_w15_gateway.py FAILING | ⚠️ | 60 min |

**Audit budget:** ~5h CC dispatched-agent work to read every .sh section + classify against pytest. Then ~10-15h CC of re-port work for the ❌/⚠️ rows.

---

## Part 2 — Concurrent execution

### Why it matters

Single-worker pytest = sequential requests = doesn't surface:
- DB lock contention under load
- Race conditions in admin endpoints
- Token issuance throughput cap
- DPoP nonce + jti uniqueness collisions
- Webhook async dispatcher backlog behavior

Marketing claim: "shark serves customer fleets." Without concurrent smoke, that's vapor.

### Two strategies (pick one or both)

#### Strategy A — `pytest-xdist` worker parallelism (cross-process)

Add `pytest-xdist` (already installed per current xdist-3.8.0 in plugins). Run with `pytest -n auto`.

**Required changes:**

1. **Per-worker shark instance.** Currently conftest.py spawns ONE session-scoped shark on :8080. With xdist, each worker process needs its own:
   - Worker ID via `PYTEST_XDIST_WORKER` env (e.g., `gw0`, `gw1`)
   - Compute port: `8080 + int(worker_id[2:])` → :8080, :8081, :8082, :8083
   - Compute DB path: `shark_${worker_id}.db`
   - Compute admin key path: `admin_${worker_id}.key.firstboot`

2. **Per-worker BASE_URL fixture** in conftest.py — replaces the module-level `BASE_URL = "http://localhost:8080"` with `@pytest.fixture(scope="session") def shark_base_url(server): ...`.

3. **Refactor module-level `BASE = "http://localhost:8080"` constants** in test files — convert to fixture consumption.

4. **Smoke tests that hit DB directly via sqlite3 (vault tests)** — must use the worker's DB path, not hardcoded `shark.db`.

**Effort:** ~3h CC + audit of all hardcoded `localhost:8080` / `shark.db` references in tests/smoke/.

**Win:** 4 workers × 50 tests each = 4× wall-clock speedup. Each worker runs full file battery → catches per-instance bugs but NOT cross-instance race conditions.

#### Strategy B — Within-test threading (single-process load)

Keep one shark, hit it with many concurrent requests inside SPECIFIC tests using `concurrent.futures.ThreadPoolExecutor` or `asyncio.gather`. New tests, not a refactor:

```python
def test_concurrent_token_issuance(admin_key, shark_base_url):
    """50 concurrent client_credentials requests must all succeed."""
    agent = create_test_agent(...)
    with ThreadPoolExecutor(max_workers=20) as ex:
        results = list(ex.map(lambda _: issue_token(agent), range(50)))
    assert all(r.status_code == 200 for r in results)
    assert len({r.json()["access_token"] for r in results}) == 50  # all unique

def test_concurrent_dpop_proofs_unique_jti(...):
    """100 concurrent DPoP proofs — all jti values unique, none rejected."""
    ...

def test_concurrent_cascade_revoke_no_phantom_revoke(...):
    """While 10 cascade-revokes run, 10 token re-issues run — no token issued AFTER its agent was revoked."""
    ...
```

**Effort:** ~2h CC for ~6 targeted load-shape tests.

**Win:** Proves shark handles concurrent writes ON ONE INSTANCE. Catches the actual SQLITE_BUSY / nonce-collision bugs. Doesn't speed up the suite, but is the *correct* "load smoke" answer.

### Recommendation

**Ship both.**
- **Strategy B first** (~2h, immediate signal on shark's concurrency story).
- **Strategy A second** (~3h, faster CI + per-worker isolation hygiene).

Together: realistic load shape + faster development feedback.

---

## Part 3 — Execution sequence

### Phase 1 — Faithful audit (5h CC, single agent)
Dispatch one sonnet agent w/ task: *"Read every `section \"...\"` block in smoke_test.sh. Classify each against tests/smoke/*.py. Output classified table. Identify the worst hallucinations first."* Output: `playbook/12b-section-audit.md`.

### Phase 2 — Re-port worst offenders (8-10h CC, parallel agents in worktrees)
Dispatch 3-4 agents in worktrees, each handling a chunk of ❌/⚠️ rows. Each agent re-ports faithfully against the .sh original.

### Phase 3 — Concurrency Strategy B (2h CC, single agent)
Add ~6 within-test load-shape tests in `tests/smoke/test_concurrent_load.py`. Each test exercises 1 concurrency invariant.

### Phase 4 — Concurrency Strategy A (3h CC, single agent)
Refactor conftest for per-worker shark spawning. Verify `pytest -n auto` works.

### Total budget: ~18-20h CC
Spread Tue-Wed post-launch. Mandatory before YC video Thursday.

---

## Stop conditions
- If section audit reveals < 30% port faithful, STOP — escalate. Means we ship Monday with mostly hallucinated coverage and need to be honest about it in the launch post.
- If concurrent test fails fundamentally (token issuance under 20-thread load drops below 95% success), STOP — investigate before any launch claims about scale.

---

# 2026-04-26 EVENING UPDATE — Lessons from F1-F10 Launch-Readiness Wave

**Trigger:** Friday-Sunday F1-F10 work shipped 9 fixes. New tests added (F4/F5/F8/F9/F10) exposed pytest fragilities not previously documented.

## New Smoke Baseline (post-F1-F10)

| Metric | Pre-F1-F10 (2026-04-26 morning) | Post-F1-F10 (2026-04-26 evening) | Delta |
|---|---|---|---|
| PASS | 225 | 282 | **+57** |
| FAIL | 13 | 14 | +1 (F10 install.sh regex too broad — cosmetic) |
| ERROR | 5 | 5 | 0 |
| XFAIL | 3 | 6 | +3 (F5 doctor live tests, wiring fix W+1) |
| SKIPPED | 33 | 37 | +4 |

**Run command:** `python -m pytest tests/smoke --tb=no -q --ignore=tests/smoke/test_f4_serve_rerun_banner.py`

**Why F4 ignored:** F4 spawns shark via subprocess.Popen + uses `signal.CTRL_BREAK_EVENT` historically (now `proc.kill()` since commit `d6527bf`). Even with kill, F4 must run isolated to avoid port conflicts with conftest's shared shark fixture.

## Test Authoring Conventions Discovered (MANDATORY for new pytest)

### 1. Subprocess signal handling (Windows)
- ❌ NEVER `proc.send_signal(signal.CTRL_BREAK_EVENT)` — kills pytest parent, exit 58 silent
- ✅ Use `proc.kill()` directly + `proc.wait(timeout=3)` in try/except
- ✅ For graceful shutdown: `proc.terminate()` first, fall back to `kill()` after timeout

### 2. Test binary refresh
- `tests/smoke/shark.exe` is a committed binary fixture — gets stale when shark code changes
- BEFORE running smoke: `go build -o shark.exe ./cmd/shark && cp -f shark.exe tests/smoke/shark.exe`
- Add to CI: rebuild + copy step BEFORE pytest invocation
- TODO: replace committed binary with `pytest fixture that rebuilds on demand`

### 3. Live-server test guards
- Tests calling running shark MUST guard with reachability probe:
  ```python
  def _shark_reachable() -> bool:
      try:
          parsed = urlparse(BASE_URL)
          with socket.create_connection((parsed.hostname, parsed.port or 8080), timeout=1):
              return True
      except OSError:
          return False
  
  _REQUIRES_LIVE_SERVER = pytest.mark.skipif(not _shark_reachable(), reason="...")
  ```
- See `tests/smoke/test_f10_openapi_scalar.py` for canonical pattern

### 4. Config-path conflicts
- `shark doctor` (and any standalone CLI test) reads default config path
- Conftest fixture spawns shark with custom DB/port — doctor doesn't see that config
- Fix options: (a) doctor accepts `--config-path` flag, (b) test sets env vars conftest exports, (c) test fixture writes config to default location
- F5 chose option (b/c) — pending W+1 wiring fix
- For new doctor-style tests: prefer env-var override pattern

### 5. Static doc-lint tests are gold
- F8 (20 PASS), F9 (19 PASS) = pure file-content checks, zero shark dependency
- Patterns: `assert "five-layer revocation" in content`, `assert not re.search(r"pip install shark-auth(?!.*git\+)", content)`
- Use these for: README, docs/, generated assets verification
- ZERO maintenance — survive shark refactors entirely

## Updated Section Audit Priorities (override Part 1 table where conflict)

Failing tests confirmed pre-existing or cut-feature, NOT regressions from F1-F10:

| Test | Status | Reason |
|---|---|---|
| `test_admin_mgmt::test_dev_inbox_access` | DEFERRED | PUNCH_LIST §C, 404 dev inbox path |
| `test_cli_user_sso_agent_session::TestAPIKeyCLI::test_api_key_create_list_revoke` | DEFERRED | PUNCH_LIST §C |
| `test_cli_user_sso_agent_session::TestAuditExportCLI::test_audit_export_stdout` | DEFERRED | PUNCH_LIST §C |
| `test_w15_advanced::test_w15_multi_listener_isolation` | DEFERRED | PUNCH_LIST §C, W15 deferred |
| `test_w15_gateway::test_transparent_gateway_porter` | DEFERRED | PUNCH_LIST §C, W15 deferred |
| `test_bulk_pattern_revoke::test_bulk_revoke_tokens_marked_revoked_in_token_list` | CUT | Bulk-pattern revoke soft-cut (CUT 2 in playbook/08) |
| `test_vault_disconnect_cascade::test_vault_disconnect_cascade` | CUT | Vault cascade soft-cut (CUT 2) |
| `test_w1_edit2_audit_breadcrumb::test_*` (×2) | PRE-EXISTING | W1.7 Edit 2 work — investigate W+1 |
| `test_w1_edit4_agent_security_card::test_audit_logs_token_exchange_fields` | PRE-EXISTING | W1 work — investigate W+1 |
| `test_w1_edit5_get_started_agent_track::TestAuditEndpoint::test_audit_page_reachable` | PRE-EXISTING | W1 work — investigate W+1 |
| `test_w17_get_started_rebuild::test_no_legacy_oauth_step_phrases` | RESOLVED-BY-F3 | F3 get-started rewrite (deferred) removes the offending `npm install @sharkauth/react` strings |
| `test_w17_identity_settings_cleanup::test_settings_page_http_200` | INVESTIGATE | ERROR — needs root cause |
| `test_w2_sdk_method_8/9/10::test_*` (×4) | INVESTIGATE | All ERROR on fixture — likely SDK fixture wiring break, not SDK regression |
| `test_f10::test_no_install_sh_in_public_docs` | F10 TEST BUG | regex matches legit `docs.sharkauth.dev` — tighten pattern |

## Pre-Launch Checklist (Monday morning)

- [ ] `go build -o shark.exe ./cmd/shark`
- [ ] `cp -f shark.exe tests/smoke/shark.exe`
- [ ] `python -m pytest tests/smoke --tb=no -q --ignore=tests/smoke/test_f4_serve_rerun_banner.py`
- [ ] Verify pass count ≥ 282 (current baseline)
- [ ] Run F4 separately: `python -m pytest tests/smoke/test_f4_serve_rerun_banner.py`
- [ ] Verify F4 passes 4/4 (or at least the static helper-existence test)
- [ ] Manual: `shark serve` → `/api/docs` → confirm Scalar UI loads
- [ ] Manual: `shark doctor` → verify clean exit on configured deployment
- [ ] Manual: `shark serve` twice → verify re-run banner shows "admin configured" or "setup pending"
- [ ] Manual: open dashboard → walk Get Started → register agent → see audit log

## Post-Launch W+1 Pytest Backlog (priority order)

1. **F5 doctor wiring fix** — make doctor honor SHARK_DB_PATH + SHARK_BASE_URL env vars set by conftest, then un-xfail the 3 tests
2. **F10 install.sh regex tighten** — pattern should be `r"sharkauth\.dev/install\.sh"` not bare `r"sharkauth\.dev"`
3. **F4 conftest integration** — port F4 tests onto shared `server` fixture so they don't need separate ignore flag
4. **Auto-rebuild fixture** — replace committed `tests/smoke/shark.exe` with session-scoped fixture that runs `go build` if source mtime > binary mtime
5. **Items 1-5 from playbook/11-pytest-human-auth-coverage.md** — email-verify, password-reset, account-self-delete, logout, failed-login-lockout
6. **W17 identity_settings + W2 SDK method ERRORs** — root-cause + fix, currently 5 errors total
7. **F3 get-started rewrite + its tests** — was deferred from launch wave; resolves test_w17_get_started_rebuild fail too
8. **F1 setup-token DB persistence** — was deferred from launch wave per founder; high UX impact for self-hosters

## Cross-References

- Test authoring conventions inherited from: HANDOFF D4 (impl + test in same diff)
- Cut-feature test classification: `playbook/08-launch-scope-cuts.md`
- Deferred test list: `playbook/PUNCH_LIST_W1_FINAL.md` §C
- F1-F10 plan: `playbook/plans/2026-04-26-launch-readiness-fixes.md`
- New test files added: `tests/smoke/test_f4_*.py` through `test_f10_*.py`
- Smoke triage commit: `d6527bf` (xfail wiring + F10 skipif)
