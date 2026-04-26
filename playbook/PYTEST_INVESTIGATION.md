# PYTEST_INVESTIGATION.md
_Generated: 2026-04-26 — branch: main_

## Background

Smoke suite: 198 tests collected. Run shows heavy `E` clusters (fixture/setup errors,
not assertion failures). This report focuses exclusively on the **new-tonight test files**
added across waves W1-edit1-5, W1.5, W1.7, W1.8, W2, W3, and their interaction with
the shared conftest.py fixture set.

---

## Section 1: Per-Test-File Problems

### `test_w1_edit1_dpop_security_tab.py`

- **CRITICAL — shadow `admin_key` fixture**: The file defines its own `@pytest.fixture(scope="module") def admin_key(server)` that reads from `data/admin.key.firstboot`. The conftest already defines `admin_key` as a `session`-scoped fixture reading from `admin.key.firstboot` (the root, not `data/`). This shadow fixture at module scope conflicts: pytest will use whichever wins by fixture resolution, but in most runs the file's own version silently fails because `data/admin.key.firstboot` does not exist (server writes to `./admin.key.firstboot`).
- **CRITICAL — wrong key path**: `key_path = "data/admin.key.firstboot"` — conftest uses `KEY_PATH = os.path.join(os.path.dirname(DB_PATH) or ".", "admin.key.firstboot")` which resolves to `./admin.key.firstboot` (root). The file's fixture hits the wrong path and skips rather than fails.
- **MEDIUM — endpoint `/api/v1/agents/{id}/audit` may not exist**: conftest/pre-existing tests use `/api/v1/audit-logs` as the audit endpoint. The file asserts a per-agent audit endpoint at `GET /api/v1/agents/{id}/audit` with a `data` key. If this sub-route is not wired, all 4 tests that fetch agent detail + the audit test will error at fixture level (`agent` fixture calls `_create_agent` which is OK, but `test_audit_endpoint_reachable` will 404 and assert away).
- **LOW — `dpop_jkt`/`dpop_key_id` assertion is likely correct as a skip-graceful check**: newly created agents won't have DPoP fields. The test accepts null values, so this is fine.

**Verdict: 3 of 6 tests will E/fail (wrong key path → fixture skip cascade); 1 test likely 404 on audit sub-route.**

---

### `test_w1_edit2_audit_breadcrumb.py`

- **CRITICAL — no fixtures used, relies on module-level `ADMIN_KEY` env var**: `ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")`. In the smoke suite `SHARK_ADMIN_KEY` is never set (conftest reads from file, not env). This means `HEADERS = {"Authorization": "Bearer "}` — an empty Bearer token. Every authenticated request will get 401.
- **CRITICAL — wrong base URL**: `BASE = os.environ.get("SHARK_BASE_URL", "http://localhost:9999")`. Conftest uses `http://localhost:8080`. In CI with no env override the test hits port 9999 → `ConnectionRefused` → `E` at collection or runtime.
- **CRITICAL — wrong audit endpoint**: Uses `/api/v1/audit-logs` (with hyphen). Pre-existing tests (`test_stats_audit.py`) use the DB directly; `test_admin_deep.py` uses `/api/v1/admin/audit-logs`. There are two different URL forms in the codebase. The form `/api/v1/audit-logs` (no `admin/` prefix) may or may not be wired — needs verification, but the wrong port will kill it first.
- Both test functions will `E` at the `requests.get` call due to connection refusal on port 9999.

**Verdict: 2/2 tests will E (connection refused on port 9999).**

---

### `test_w1_edit3_delegation_policies.py`

- **CRITICAL — invented fixture `admin_token`**: `@pytest.fixture def auth_headers(admin_token)` — `admin_token` does not exist in conftest.py. Conftest has `admin_key` and `admin_client`. Collection error: `fixture 'admin_token' not found`.
- **CRITICAL — hardcoded base URL `http://localhost:9002/api/v1`**: conftest uses `http://localhost:8080`. Port 9002 → connection refused.
- **CRITICAL — endpoint `/api/v1/agents/{id}/policies`**: No evidence this endpoint exists in the backend. The admin agents API creates/lists/deletes agents but the policies sub-resource is a new surface the wave may or may not have wired.
- **All 7 tests will E at collection** due to `fixture 'admin_token' not found`.

**Verdict: 7/7 tests will E (collection error: missing fixture `admin_token`).**

---

### `test_w1_edit4_agent_security_card.py`

- **CRITICAL — module-level empty `ADMIN_KEY`**: Same pattern as edit2. `ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")`. No env var set → `Authorization: Bearer ` → 401 on every authenticated call.
- **MEDIUM — endpoint mismatch**: Uses `GET /api/v1/admin/audit-logs` (with `admin/` prefix). Pre-existing test_admin_deep.py also uses `/api/v1/admin/audit-logs`, so the URL form is likely correct. But the empty auth key will 401 it.
- **MEDIUM — endpoint `/api/v1/agents/{id}/tokens`**: Not confirmed to exist. No pre-existing test uses this endpoint.
- **LOW — `has_more` field assertion**: Tests asserts `"has_more" in body` on the audit-logs response. Pre-existing admin_deep.py doesn't assert this field — unknown if backend includes it.
- **4/4 tests will fail** (authenticated ones get 401 due to empty key; unauth tests will pass if server is up on port 8080, but these are in class `TestNegative` with hardcoded invalid key so they should pass).

**Verdict: 2 positive tests will 401-fail; 2 negative/unauth tests should pass.**

---

### `test_w1_edit5_get_started_agent_track.py`

- **CRITICAL — hardcoded `BASE = "http://localhost:9000"`**: conftest uses 8080. Port 9000 → connection refused.
- **MEDIUM — `admin_key()` is a plain function, not a fixture**: Defined as `def admin_key()` (no `@pytest.fixture`), calls `os.getenv("SHARK_ADMIN_KEY", "")` and `pytest.skip`. This is not a pytest fixture, so no fixture injection. Each test that calls `admin_key()` works but relies on `SHARK_ADMIN_KEY` env var which is not set.
- **MEDIUM — endpoint `/api/v1/admin/audit`**: Uses `/api/v1/admin/audit` (no `-logs` suffix) in `test_audit_delegation_filter_unauth`. Pre-existing tests use `/api/v1/admin/audit-logs`. These are different URLs.
- **All 4 tests will E** (connection refused on port 9000).

**Verdict: 4/4 tests will E (connection refused on port 9000).**

---

### `test_w17_coming_soon_routes.py`

- **CRITICAL — `admin_key` used as a function argument, not fixture**: `def test_audit_endpoint_still_reachable(admin_key)` — this is correct syntax (pytest will inject the conftest `admin_key` fixture). BUT the test then uses `headers={"X-Admin-Key": admin_key}` — the `X-Admin-Key` header is NOT how the backend does auth. CORS middleware lists it as an allowed header, but the actual auth middleware checks `Authorization: Bearer <token>`. This means the request goes out unauthenticated → 401, defeating the test.
- **MEDIUM — `test_audit_endpoint_unauth_returns_401(admin_key)`**: This test takes `admin_key` as a parameter but doesn't use it (unauthenticated request). That's fine for the negative test. Should pass.
- **MEDIUM — `test_agents_endpoint_still_reachable(admin_key)`**: Same wrong header `X-Admin-Key` → 401 instead of 200 → assertion failure.
- **LOW — `test_agents_endpoint_unauth_returns_401()`**: No fixtures, no args, just hits endpoint. Should pass.

**Verdict: 2 tests fail (wrong auth header); 2 tests should pass.**

---

### `test_w17_dev_email_banner.py`

- **GOOD — correct fixture usage**: Uses `admin_client` (conftest session fixture, properly pre-authenticated). URL from `BASE_URL` env or `http://localhost:8080`. This is the correct pattern.
- **MEDIUM — dev email endpoint may 404**: `GET /api/v1/admin/dev/emails` only exists when server is in dev mode. Test handles this with `pytest.skip` on 404. Graceful.
- **LOW — `test_dev_inbox_requires_auth()`**: Uses raw `requests.get(INBOX_URL)` without fixtures. Correct for negative test. Handles 404 gracefully.

**Verdict: Both tests should work correctly (1 may skip if not dev mode). Cleanest new file.**

---

### `test_w18_boot_polish.py`

- **GOOD — correct fixture usage**: Uses `server` fixture (conftest session). Reads `server.log` which conftest creates. Port-agnostic.
- **MEDIUM — banner regex may be too strict**: `re.search(r"Dashboard.*http://localhost:\d+/admin", content, re.DOTALL)` — if the boot banner uses ANSI escape codes that interleave between "Dashboard" and the URL, the regex may miss it. The comment in the file notes this but says it "accepts either bare or ANSI-wrapped form." However `re.DOTALL` allows newlines between matches, so ANSI codes in between should still be caught as long as the two strings appear on the same or adjacent lines.
- **LOW — panic check**: Simply `assert "panic:" not in content`. Reliable.

**Verdict: Both tests should pass in most runs. Minor fragility on banner regex.**

---

### `test_w2_sdk_get_token_dpop.py`

- **CRITICAL — `shark_auth` SDK may not be installed**: Imports `from shark_auth.dpop import DPoPProver`, `from shark_auth.oauth import OAuthClient, Token`. The SDK lives at `sdk/python/shark_auth/` but there's no evidence it's installed as a package in the smoke test environment (no `pip install -e sdk/python` in conftest or setup). If not installed → `ModuleNotFoundError` → collection error → `E` on all tests in this file.
- **CRITICAL — wrong agent registration URL**: `registered_agent` fixture POSTs to `/admin/agents` (no `/api/v1/` prefix). Should be `/api/v1/agents`.
- **MEDIUM — hardcoded `BASE_URL = "http://localhost:9090"`**: conftest uses 8080. With no env var set → connection refused.
- **LOW — `Token.cnf_jkt` vs `prover.jkt`**: SDK `oauth.py` line 43 shows `cnf_jkt: str | None`. `DPoPProver.jkt` property exists (dpop.py line 92). This assertion should work if the rest works.

**Verdict: 2/2 tests will E (ModuleNotFoundError or connection refused on 9090).**

---

### `test_w2_sdk_http_with_dpop.py`

- **CRITICAL — `shark_auth` SDK import**: Same as above. `from shark_auth import Client`, `from shark_auth.dpop import DPoPProver`. If not installed → collection error.
- **CRITICAL — invented fixtures `shark_base_url` and `admin_token`**: `def shark_client(shark_base_url: str, admin_token: str)` — neither `shark_base_url` nor `admin_token` exists in conftest.py. Collection error: `fixture 'shark_base_url' not found`.
- **CRITICAL — `shark_auth.tokens.get_token_with_dpop`**: `from shark_auth.tokens import get_token_with_dpop` — `tokens.py` exists in the SDK but need to verify this function is exported. Even if it is, the missing fixtures kill this first.

**Verdict: 1 test + 1 skipped. The live test will E (fixture not found).**

---

### `test_w2_sdk_token_exchange.py`

- **CRITICAL — `shark_auth` SDK import**: Same collection error if not installed.
- **CRITICAL — invented fixtures**: `oauth_client(shark_base_url)`, `parent_token(registered_agent)`, `registered_agent` — none of these exist in conftest. Three fixture dependencies that don't exist.
- **MEDIUM — `prover.thumbprint()` vs `prover.jkt`**: `test_token_exchange_happy_path` asserts `child.cnf_jkt == prover.thumbprint()`. But `DPoPProver` exposes `jkt` as a property (dpop.py line 92), not `thumbprint()`. Calling `prover.thumbprint()` would raise `AttributeError`. (The other SDK test uses `prover.jkt` correctly.)

**Verdict: 2/2 tests will E (missing fixtures + wrong method name `thumbprint()`).**

---

### `test_w3_demo_command.py`

- **MEDIUM — `demo delegation-with-trace` subcommand**: Uses `server` and `admin_key` fixtures correctly (conftest). Calls `./shark.exe demo delegation-with-trace`. If this CLI subcommand is not implemented, `returncode != 0` → assertion failure (not a fixture error, so shows as `F` not `E`).
- **MEDIUM — strict output assertions**: Asserts exact strings like `"[1/3] Registering agents"`, `"DPoP proofs: 3/3 verified"`. If the demo command output format changed, these fail.
- **LOW — binary path**: Uses `./shark.exe` directly without checking the platform, but conftest already defines `BIN_PATH` with the OS check. This test re-hardcodes it. On Linux CI this breaks.

**Verdict: 1 test — will fail (F) if `demo` subcommand not implemented; may E on Linux.**

---

### `test_cascade_revoke.py`

- **GOOD — mostly correct**: Uses `admin_key` from conftest (correct). Defines local `admin_headers` fixture correctly from `admin_key`. `BASE_URL` from env with 8080 default.
- **MEDIUM — endpoint `/api/v1/users/{id}/revoke-agents`**: New endpoint. May not be wired.
- **MEDIUM — endpoint `/api/v1/users/{id}/agents`**: New endpoint for user-scoped agent listing.
- **MEDIUM — endpoint `/api/v1/me/agents`**: New endpoint for self-service agent listing.
- **MEDIUM — endpoint `/api/v1/audit-logs` (no `admin/` prefix)**: Different from what other tests use. If not wired → 404.
- **LOW — `created_by` field on agent creation**: POSTs `"created_by": user_id` in agent payload. Unknown if the API accepts/ignores this.
- **Fixture chain is clean**: 6 local fixtures all properly inject from conftest `admin_key`. This is well-written.

**Verdict: 6/6 tests may fail (F) if new endpoints not wired, but no collection errors. Best-structured new file after w17_dev_email_banner.**

---

### `test_w15_disable_revokes_tokens.py` and `test_w15_delete_user_revokes_tokens.py`

- **SAFE — entire module skipped**: `pytestmark = pytest.mark.skip(...)`. These will show as `s` not `E`. No impact on E-count.

**Verdict: Correctly neutralized. 0 failures.**

---

## Section 2: Root-Cause Categorization

### Category A — Wrong/Missing Fixture (collection-time errors → `E`)
Files: `test_w1_edit3_delegation_policies.py`, `test_w2_sdk_http_with_dpop.py`, `test_w2_sdk_token_exchange.py`
- `admin_token` does not exist in conftest
- `shark_base_url` does not exist in conftest
- `registered_agent` does not exist in conftest

### Category B — Wrong Base URL / Port (runtime connection errors → `E`)
Files: `test_w1_edit2_audit_breadcrumb.py` (port 9999), `test_w1_edit3_delegation_policies.py` (port 9002), `test_w1_edit5_get_started_agent_track.py` (port 9000), `test_w2_sdk_get_token_dpop.py` (port 9090)
- Each agent that wrote these files used a different dev port. None match conftest's 8080.

### Category C — Wrong Auth Header (`X-Admin-Key` vs `Authorization: Bearer`)
Files: `test_w17_coming_soon_routes.py`
- Backend auth middleware uses `Authorization: Bearer`. `X-Admin-Key` is only listed in CORS allowed-headers, not processed by auth middleware.

### Category D — Shadow Fixture / Wrong Key Path
Files: `test_w1_edit1_dpop_security_tab.py`
- Redefines `admin_key` at module scope pointing to `data/admin.key.firstboot` instead of `./admin.key.firstboot`.

### Category E — SDK Not Installed / Missing Import
Files: `test_w2_sdk_get_token_dpop.py`, `test_w2_sdk_http_with_dpop.py`, `test_w2_sdk_token_exchange.py`
- `shark_auth` SDK package lives in `sdk/python/` but may not be `pip install -e`'d in the test environment. If missing → `ModuleNotFoundError` at collection.

### Category F — Wrong Method Name (SDK API mismatch)
Files: `test_w2_sdk_token_exchange.py`
- Calls `prover.thumbprint()` but `DPoPProver` exposes `.jkt` property (no `thumbprint()` method). → `AttributeError` at runtime.

### Category G — Wrong Endpoint URL
Files: `test_w1_edit2_audit_breadcrumb.py` (`/api/v1/audit-logs` no `admin/` prefix), `test_w1_edit5_get_started_agent_track.py` (`/api/v1/admin/audit` no `-logs`), `test_w17_coming_soon_routes.py` (`/api/v1/admin/audit-logs`)
- Multiple different audit endpoint URL forms used. Pre-existing admin tests use `/api/v1/admin/audit-logs`.

### Category H — Module-level empty env-var auth (runtime 401 → `F`)
Files: `test_w1_edit2_audit_breadcrumb.py`, `test_w1_edit4_agent_security_card.py`
- Read `SHARK_ADMIN_KEY` env var at module load time, defaulting to `""`. Suite doesn't set this env var; conftest reads the key from a file. Empty Bearer token → 401 on every admin call.

### Category I — Unimplemented backend endpoints (runtime 404/500 → `F`)
Files: `test_cascade_revoke.py`, `test_w1_edit3_delegation_policies.py`, `test_w3_demo_command.py`
- `/api/v1/users/{id}/revoke-agents`, `/api/v1/users/{id}/agents`, `/api/v1/me/agents`, `/api/v1/agents/{id}/policies`, `shark.exe demo delegation-with-trace` — new surfaces not confirmed wired.

---

## Section 3: Recommended Fix Per File

### `test_w1_edit1_dpop_security_tab.py`
**Fix**: Remove the local `admin_key` fixture entirely. Let conftest's session-scoped `admin_key` inject. Change `agent` fixture to module or session scope and ensure it doesn't shadow conftest.
```python
# DELETE lines 42-48 (local admin_key fixture)
# Keep: @pytest.fixture(scope="module") def agent(admin_key): ...
```
The `admin_key` parameter in `agent` fixture will then resolve to conftest's version. Also drop the `server` parameter from the now-deleted local fixture (conftest's `admin_key` already depends on `server`).

### `test_w1_edit2_audit_breadcrumb.py`
**Fix**: Replace module-level env-var auth with proper fixture injection. Change both test functions to accept `admin_client` fixture. Change `BASE` to use conftest's `BASE_URL`.
```python
# Replace:
BASE = os.environ.get("SHARK_BASE_URL", "http://localhost:9999")
ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")
HEADERS = {"Authorization": f"Bearer {ADMIN_KEY}"}

# With:
import os
BASE = os.environ.get("BASE", "http://localhost:8080")

# Change function signatures:
def test_agent_actor_event_has_act_chain_shape(admin_client): ...
def test_human_actor_event_has_no_act_chain(admin_client): ...

# Replace requests.get calls to use admin_client.get(...)
# Replace HEADERS with nothing (admin_client already has auth headers)
```
Also fix endpoint: use `/api/v1/admin/audit-logs` to match backend.

### `test_w1_edit3_delegation_policies.py`
**Fix**: Replace `admin_token` with `admin_key`, rewrite `auth_headers` fixture to match conftest pattern. Fix base URL.
```python
# Replace:
BASE = "http://localhost:9002/api/v1"
@pytest.fixture(scope="module")
def auth_headers(admin_token): ...

# With:
import os
BASE = os.environ.get("BASE", "http://localhost:8080") + "/api/v1"
@pytest.fixture(scope="module")
def auth_headers(admin_key):
    return {"Authorization": f"Bearer {admin_key}"}
```
Additionally: mark all tests `@pytest.mark.skip(reason="policies endpoint not yet wired — verify backend before enabling")` until `/api/v1/agents/{id}/policies` is confirmed implemented.

### `test_w1_edit4_agent_security_card.py`
**Fix**: Replace env-var auth pattern with fixture injection.
```python
# Remove:
ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")
AUTH = {"Authorization": f"Bearer {ADMIN_KEY}"}

# Add admin_client parameter to each test function:
def test_audit_logs_token_exchange_fields(admin_client): ...
    resp = admin_client.get(f"{BASE}/api/v1/admin/audit-logs", params={...})
```

### `test_w1_edit5_get_started_agent_track.py`
**Fix**: Change `BASE = "http://localhost:9000"` to `BASE = os.environ.get("BASE", "http://localhost:8080")`. Rewrite `admin_key()` helper as a proper fixture or use conftest's `admin_client` directly. Fix endpoint: `/api/v1/admin/audit` → `/api/v1/admin/audit-logs`.

### `test_w17_coming_soon_routes.py`
**Fix**: Change auth header from `X-Admin-Key` to `Authorization: Bearer`.
```python
# Replace:
headers={"X-Admin-Key": admin_key}
# With:
headers={"Authorization": f"Bearer {admin_key}"}
```
This is a 2-line fix. The fixture injection (`admin_key` parameter) is already correct.

### `test_w17_dev_email_banner.py`
**No fix needed.** Already correct. Keep as-is.

### `test_w18_boot_polish.py`
**No fix needed.** Already correct. Keep as-is.

### `test_w2_sdk_get_token_dpop.py`
**Fix priority 1**: Ensure `pip install -e sdk/python` in the smoke test environment (conftest setup or pytest.ini). Without this, entire file errors at collection.
**Fix priority 2**: Change `BASE_URL` default from `9090` to `8080`.
**Fix priority 3**: Fix agent registration URL: `/admin/agents` → `/api/v1/agents`.
**Alternative**: Skip entire file until SDK install is confirmed: add `pytestmark = pytest.mark.skip(reason="SDK not installed in smoke env — see sdk/python README")`.

### `test_w2_sdk_http_with_dpop.py`
**Fix**: Add `shark_base_url` and `admin_token` fixtures to conftest, or rewrite test to use existing conftest fixtures (`admin_key`, `server`).
```python
# In conftest.py, add:
@pytest.fixture(scope="session")
def shark_base_url():
    return BASE_URL

@pytest.fixture(scope="session")
def admin_token(admin_key):
    return admin_key
```
**Alternative**: Skip file. The SDK-based tests are integration tests that need their own conftest extension.

### `test_w2_sdk_token_exchange.py`
**Fix 1**: Same missing fixtures as above — add `shark_base_url`, `registered_agent`.
**Fix 2 (critical)**: Change `prover.thumbprint()` → `prover.jkt`.
```python
# Replace:
assert child.cnf_jkt == prover.thumbprint()
# With:
assert child.cnf_jkt == prover.jkt
```

### `test_w3_demo_command.py`
**Fix**: Add platform-aware binary path (use conftest's `BIN_PATH`). Mark skip if `demo` subcommand is not implemented.
```python
import os
BIN_PATH = "./shark.exe" if os.name == 'nt' else "./shark"
```
Also verify `shark.exe demo delegation-with-trace` is a real implemented subcommand before enabling this test.

### `test_cascade_revoke.py`
**Mostly fine structurally.** Mark individual tests skip until new endpoints confirmed:
- `/api/v1/users/{id}/revoke-agents`
- `/api/v1/users/{id}/agents`
- `/api/v1/me/agents`
- `/api/v1/audit-logs` (verify vs `/api/v1/admin/audit-logs`)

---

## Section 4: Tally

### New-tonight files (14 total, counting skipped-by-design as 0)

| File | Tests | Status | Cause |
|---|---|---|---|
| test_w1_edit1_dpop_security_tab.py | 6 | 4 E + 2 may pass | Shadow fixture, wrong key path |
| test_w1_edit2_audit_breadcrumb.py | 2 | 2 E | Wrong port 9999 + empty auth |
| test_w1_edit3_delegation_policies.py | 7 | 7 E | Missing fixture `admin_token` |
| test_w1_edit4_agent_security_card.py | 4 | 2 F + 2 pass | Empty ADMIN_KEY env var |
| test_w1_edit5_get_started_agent_track.py | 4 | 4 E | Wrong port 9000 |
| test_w17_coming_soon_routes.py | 4 | 2 F + 2 pass | Wrong auth header |
| test_w17_dev_email_banner.py | 2 | 2 pass (1 may skip) | Clean |
| test_w18_boot_polish.py | 2 | 2 pass | Clean |
| test_w2_sdk_get_token_dpop.py | 2 | 2 E | SDK not installed + wrong port |
| test_w2_sdk_http_with_dpop.py | 1+1s | 1 E | Missing fixtures |
| test_w2_sdk_token_exchange.py | 2 | 2 E | Missing fixtures + wrong method |
| test_w3_demo_command.py | 1 | 1 F | Demo subcommand may not exist |
| test_w15_disable_revokes_tokens.py | 2 | 2 s | Intentionally skipped (correct) |
| test_w15_delete_user_revokes_tokens.py | 2 | 2 s | Intentionally skipped (correct) |
| test_cascade_revoke.py | 6 | 6 F | New endpoints not confirmed |

**Summary:**
- Total new-tonight tests: ~47 (excluding intentional skips)
- **Borked (E = collection/fixture error)**: ~24 tests across 6 files
- **Likely failing (F = runtime assertion)**: ~10 tests across 4 files
- **Should pass cleanly**: ~4 tests (w17_dev_email_banner × 2, w18_boot_polish × 2)
- **Intentional skips (correct)**: 4 tests (w15 files)

### Pre-existing tests untouched

All ~150 pre-existing tests are structurally unchanged. Their fixture usage (conftest `admin_key`, `admin_client`, `auth_session`, `smoke_user`, `server`, `db_conn`) is correct. Any failures in pre-existing tests are backend regressions, not test code issues.

---

## Top 3 Most-Impactful Fixes

**Fix 1 — Add 2 fixtures to conftest.py (unblocks all W2 SDK tests)**
```python
@pytest.fixture(scope="session")
def shark_base_url(): return BASE_URL

@pytest.fixture(scope="session")
def admin_token(admin_key): return admin_key
```
Unblocks `test_w2_sdk_http_with_dpop.py` and `test_w2_sdk_token_exchange.py`.

**Fix 2 — Fix auth header in `test_w17_coming_soon_routes.py` (2 tests flip from F to pass)**
Change `headers={"X-Admin-Key": admin_key}` → `headers={"Authorization": f"Bearer {admin_key}"}` in both authenticated test functions. 2-line change.

**Fix 3 — Remove shadow `admin_key` fixture from `test_w1_edit3_delegation_policies.py` and fix its BASE URL (unblocks 7 E-tests, though endpoint availability is a separate issue)**
Replace `admin_token` fixture dependency with `admin_key`, change `BASE = "http://localhost:9002/api/v1"` to `BASE = os.environ.get("BASE", "http://localhost:8080") + "/api/v1"`. Fixes the collection error; tests may then fail at runtime if the policies endpoint is not wired.

---

_Investigation complete. No test files modified. Report only._
