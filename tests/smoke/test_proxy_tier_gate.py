"""Tier-gate proxy flow (Lane D — D1).

Exercises the end-to-end ReqTier predicate flow:

  1. User signs up at default ("free") tier.
  2. Admin creates an app + a DB proxy rule requiring tier:pro on /pro/*.
  3. Free-tier user hits /pro/x → proxy 302-redirects to /paywall/<slug>.
  4. Admin upgrades the user to pro via PATCH /admin/users/{id}/tier.
  5. User re-logs in (fresh JWT with new tier claim) → /pro/x passes through
     to the toy upstream (HTTP 200).

The test spins up a dedicated `shark serve` against a tmp_path DB on a random
port so it doesn't share state with the session-scoped conftest fixture. The
proxy upstream is a local-echo HTTP server started in a background thread.

Contracts:
  docs/proxy_v1_5/api/admin_proxy_rules_db.md
  docs/proxy_v1_5/api/admin_users_tier.md
  docs/proxy_v1_5/api/paywall_route.md
  docs/proxy_v1_5/contracts/decision_kinds.md
"""
import http.server
import os
import socket
import subprocess
import threading
import time

import pytest
import requests


BIN_PATH = "./shark.exe" if os.name == "nt" else "./shark"


def find_free_port():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("", 0))
        return s.getsockname()[1]


def wait_for_port(port, host="127.0.0.1", timeout=15):
    start = time.time()
    while time.time() - start < timeout:
        try:
            with socket.create_connection((host, port), timeout=1):
                return True
        except OSError:
            pass
        time.sleep(0.2)
    return False


def wait_for_health(url, timeout=15):
    start = time.time()
    while time.time() - start < timeout:
        try:
            if requests.get(url, timeout=1).status_code == 200:
                return True
        except requests.RequestException:
            pass
        time.sleep(0.2)
    return False


class _ToyUpstream:
    """A trivial HTTP echo server used as the proxy upstream."""

    def __init__(self):
        self.port = find_free_port()
        self._httpd = None
        self._thread = None

    def start(self):
        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):  # noqa: N802 — stdlib signature
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                # Echo useful identity headers so callers can assert on them.
                body = {
                    "path": self.path,
                    "x_user_id": self.headers.get("X-User-ID", ""),
                    "x_user_email": self.headers.get("X-User-Email", ""),
                    "x_auth_method": self.headers.get("X-Auth-Method", ""),
                }
                import json as _json
                self.wfile.write(_json.dumps(body).encode())

            def log_message(self, *args, **kwargs):  # silence
                pass

        self._httpd = http.server.HTTPServer(("127.0.0.1", self.port), Handler)
        self._thread = threading.Thread(target=self._httpd.serve_forever, daemon=True)
        self._thread.start()
        assert wait_for_port(self.port)

    def stop(self):
        if self._httpd is not None:
            self._httpd.shutdown()
            self._httpd.server_close()


@pytest.fixture
def tier_gate_env(tmp_path):
    """Spin up shark serve + toy upstream on fresh ports. Yields handy dict.

    W17: no yaml config, no --dev flag. Server boots from defaults via
    --no-prompt; port and DB path are injected via SHARK_PORT / SHARK_DB_PATH
    env vars so each test fixture is isolated from the session server.
    """
    upstream = _ToyUpstream()
    upstream.start()

    port = find_free_port()
    base = f"http://127.0.0.1:{port}"
    log_path = str(tmp_path / "tier_gate.log")

    # Absolute path required because the subprocess runs with cwd=tmp_path.
    bin_abs = os.path.abspath(BIN_PATH)
    log = open(log_path, "w")
    env = os.environ.copy()
    env["SHARK_PORT"] = str(port)
    env["SHARK_DB_PATH"] = str(tmp_path / "shark.db")
    proc = subprocess.Popen(
        [bin_abs, "serve", "--no-prompt", "--proxy-upstream", f"http://127.0.0.1:{upstream.port}"],
        stdout=log,
        stderr=log,
        cwd=str(tmp_path),
        env=env,
    )

    try:
        if not wait_for_health(f"{base}/healthz"):
            proc.terminate()
            proc.wait(timeout=5)
            log.close()
            with open(log_path) as f:
                pytest.fail(f"server failed to come up: {f.read()}")

        # W17: admin key is written to <db_dir>/admin.key.firstboot.
        key_file = tmp_path / "admin.key.firstboot"
        for _ in range(50):
            if key_file.exists():
                break
            time.sleep(0.1)
        if not key_file.exists():
            pytest.fail(f"admin key file never appeared at {key_file}")
        admin_key = key_file.read_text().strip()

        yield {
            "base": base,
            "admin_key": admin_key,
            "upstream_port": upstream.port,
        }
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        log.close()
        upstream.stop()


def _admin_headers(admin_key):
    return {"Authorization": f"Bearer {admin_key}"}


def _signup(base, email, password="Password123!"):
    r = requests.post(
        f"{base}/api/v1/auth/signup",
        json={"email": email, "password": password},
        timeout=5,
    )
    assert r.status_code in (200, 201), r.text
    data = r.json()
    return data["id"], data["token"]


def _login(base, email, password="Password123!"):
    r = requests.post(
        f"{base}/api/v1/auth/login",
        json={"email": email, "password": password},
        timeout=5,
    )
    assert r.status_code == 200, r.text
    return r.json()["token"]


def _create_app(base, admin_key, slug):
    r = requests.post(
        f"{base}/api/v1/admin/apps",
        headers=_admin_headers(admin_key),
        json={
            "name": "Tier Gate App",
            "slug": slug,
            "integration_mode": "proxy",
        },
        timeout=5,
    )
    assert r.status_code == 201, r.text
    return r.json()


def _create_rule(base, admin_key, payload):
    r = requests.post(
        f"{base}/api/v1/admin/proxy/rules/db",
        headers=_admin_headers(admin_key),
        json=payload,
        timeout=5,
    )
    assert r.status_code == 201, r.text
    return r.json()["data"]


def _reload_proxy(base, admin_key):
    # The engine is already refreshed by each mutation, but a reload is
    # idempotent and guarantees listeners pick up the newest rule set.
    r = requests.post(
        f"{base}/api/v1/admin/proxy/reload",
        headers=_admin_headers(admin_key),
        timeout=5,
    )
    # 200 happy path; 404 when Manager isn't wired (embedded-only); both OK.
    assert r.status_code in (200, 404), r.text


def test_tier_gate_free_user_hits_paywall(tier_gate_env):
    base = tier_gate_env["base"]
    admin_key = tier_gate_env["admin_key"]

    email = f"tier_gate_user_{int(time.time()*1000)}@test.com"
    user_id, _token_free = _signup(base, email)

    slug = f"tier-gate-{int(time.time()*1000)}"
    _create_app(base, admin_key, slug)

    _create_rule(
        base,
        admin_key,
        {
            "name": "pro-gate",
            "pattern": "/pro/*",
            "methods": ["GET"],
            "require": "tier:pro",
            "priority": 100,
        },
    )
    _reload_proxy(base, admin_key)

    # Free tier user hits /pro/x → PaywallRedirect (302 to /paywall/...)
    token = _login(base, email)
    r = requests.get(
        f"{base}/pro/something",
        headers={"Authorization": f"Bearer {token}"},
        allow_redirects=False,
        timeout=5,
    )
    assert r.status_code == 302, (
        f"expected 302 paywall redirect, got {r.status_code}: {r.text}"
    )
    loc = r.headers.get("Location", "")
    assert "/paywall/" in loc, f"expected paywall redirect URL, got {loc!r}"
    assert "tier=pro" in loc, f"expected tier=pro in URL, got {loc!r}"
    assert "return=" in loc, f"expected return= in URL, got {loc!r}"

    # Admin upgrades user to pro tier.
    r = requests.patch(
        f"{base}/api/v1/admin/users/{user_id}/tier",
        headers=_admin_headers(admin_key),
        json={"tier": "pro"},
        timeout=5,
    )
    assert r.status_code == 200, r.text
    assert r.json()["data"]["tier"] == "pro"

    # Re-login to get a fresh JWT carrying the new tier claim.
    token_pro = _login(base, email)
    r = requests.get(
        f"{base}/pro/something",
        headers={"Authorization": f"Bearer {token_pro}"},
        allow_redirects=False,
        timeout=5,
    )
    assert r.status_code == 200, (
        f"upgraded user expected 200 from upstream, got {r.status_code}: {r.text[:200]}"
    )
    body = r.json()
    assert body["path"] == "/pro/something"
    # Identity headers should have been injected onto the upstream request.
    assert body["x_user_id"] == user_id


def test_tier_gate_anonymous_denied_not_paywall(tier_gate_env):
    """Anonymous callers should not paywall-redirect; they hit the
    anonymous-deny branch (401 for API callers / login redirect for browsers).
    See contracts/decision_kinds.md — PaywallRedirect only fires for an
    already-authenticated caller whose tier doesn't match.
    """
    base = tier_gate_env["base"]
    admin_key = tier_gate_env["admin_key"]

    slug = f"tier-gate-anon-{int(time.time()*1000)}"
    _create_app(base, admin_key, slug)
    _create_rule(
        base,
        admin_key,
        {
            "name": "pro-gate-anon",
            "pattern": "/proanon/*",
            "methods": ["GET"],
            "require": "tier:pro",
            "priority": 100,
        },
    )
    _reload_proxy(base, admin_key)

    # No Authorization header. Explicit non-browser Accept to avoid the
    # hosted-login redirect branch.
    r = requests.get(
        f"{base}/proanon/x",
        headers={"Accept": "application/json"},
        allow_redirects=False,
        timeout=5,
    )
    # Anonymous deny: 401, not a /paywall redirect.
    assert r.status_code == 401, (
        f"expected 401 anonymous deny, got {r.status_code}: {r.text[:200]}"
    )
    assert "/paywall/" not in r.headers.get("Location", "")


def test_tier_gate_match_agent_vs_human(tier_gate_env):
    """Tier match + m2m flag interaction (D2 — tier_match_agent_vs_human).

    Two rules both require tier:pro. One has m2m=true (agent-only), the
    other doesn't (human-inclusive). A pro-upgraded human user should pass
    the human-inclusive rule and hit the upstream (200), but the m2m rule
    must deny with DecisionDenyForbidden (403) because the human caller's
    ActorType != agent — even though the tier check would otherwise pass.
    See contracts/m2m_rule_flag.md.
    """
    base = tier_gate_env["base"]
    admin_key = tier_gate_env["admin_key"]

    slug = f"tier-actor-{int(time.time()*1000)}"
    _create_app(base, admin_key, slug)

    _create_rule(
        base,
        admin_key,
        {
            "name": "pro-human",
            "pattern": "/prohuman/*",
            "methods": ["GET"],
            "require": "tier:pro",
            "priority": 100,
        },
    )
    _create_rule(
        base,
        admin_key,
        {
            "name": "pro-agent-only",
            "pattern": "/proagent/*",
            "methods": ["GET"],
            "require": "tier:pro",
            "m2m": True,
            "priority": 100,
        },
    )
    _reload_proxy(base, admin_key)

    email = f"tier_actor_user_{int(time.time()*1000)}@test.com"
    user_id, _t0 = _signup(base, email)
    # Upgrade to pro.
    r = requests.patch(
        f"{base}/api/v1/admin/users/{user_id}/tier",
        headers=_admin_headers(admin_key),
        json={"tier": "pro"},
        timeout=5,
    )
    assert r.status_code == 200, r.text
    token = _login(base, email)

    # Human-inclusive pro rule: passes → 200.
    r = requests.get(
        f"{base}/prohuman/ping",
        headers={"Authorization": f"Bearer {token}"},
        allow_redirects=False,
        timeout=5,
    )
    assert r.status_code == 200, (
        f"expected human pro to pass, got {r.status_code}: {r.text[:200]}"
    )

    # m2m=true rule: human caller denied with 403 (DecisionDenyForbidden).
    r = requests.get(
        f"{base}/proagent/ping",
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/json",
        },
        allow_redirects=False,
        timeout=5,
    )
    assert r.status_code == 403, (
        f"expected 403 (m2m blocks human), got {r.status_code}: {r.text[:200]}"
    )
