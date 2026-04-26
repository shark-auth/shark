"""Identity header spoof-strip smoke test (Lane D — D2).

Spec §6.2: the proxy MUST strip client-supplied ``X-User-*``, ``X-Agent-*``,
and ``X-Shark-*`` headers before forwarding so an unauthenticated caller
cannot pretend to be a privileged identity. The proxy then injects its
own authenticated identity headers. Verified by pointing an echo upstream
at the proxy and asserting what the upstream actually received.

Contract: internal/proxy/headers.go StripIdentityHeaders + InjectIdentity.
"""
import http.server
import json
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


class _HeaderEchoUpstream:
    def __init__(self):
        self.port = find_free_port()
        self._httpd = None
        self._thread = None

    def start(self):
        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):  # noqa: N802
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                # Echo every X-* header back so tests can assert exactly
                # which ones reached the upstream.
                headers = {
                    k: v for k, v in self.headers.items()
                    if k.lower().startswith("x-")
                }
                self.wfile.write(json.dumps({"headers": headers}).encode())

            def log_message(self, *a, **kw):
                pass

        self._httpd = http.server.HTTPServer(("127.0.0.1", self.port), Handler)
        self._thread = threading.Thread(
            target=self._httpd.serve_forever, daemon=True
        )
        self._thread.start()
        assert wait_for_port(self.port)

    def stop(self):
        if self._httpd:
            self._httpd.shutdown()
            self._httpd.server_close()


@pytest.fixture
def spoof_env(tmp_path):
    upstream = _HeaderEchoUpstream()
    upstream.start()

    port = find_free_port()
    base = f"http://127.0.0.1:{port}"
    log_path = str(tmp_path / "spoof.log")

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

        key_file = tmp_path / "admin.key.firstboot"
        for _ in range(50):
            if key_file.exists():
                break
            time.sleep(0.1)
        if not key_file.exists():
            pytest.fail(f"admin key file never appeared at {key_file}")
        admin_key = key_file.read_text().strip()

        yield {"base": base, "admin_key": admin_key}
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        log.close()
        upstream.stop()


def _admin(k):
    return {"Authorization": f"Bearer {k}"}


def _create_rule(base, admin_key, payload):
    r = requests.post(
        f"{base}/api/v1/admin/proxy/rules/db",
        headers=_admin(admin_key),
        json=payload,
        timeout=5,
    )
    assert r.status_code == 201, r.text
    return r.json()["data"]


def _signup_and_token(base, email, password="Password123!"):
    r = requests.post(
        f"{base}/api/v1/auth/signup",
        json={"email": email, "password": password},
        timeout=5,
    )
    assert r.status_code in (200, 201), r.text
    data = r.json()
    return data["id"], data["token"]


def _hdr_keys_lower(body):
    return {k.lower(): v for k, v in body["headers"].items()}


def test_authenticated_caller_spoof_attempt_is_stripped(spoof_env):
    """An authenticated user tries to impersonate admin/hijack headers.
    Proxy must strip all X-User-*, X-Agent-*, X-Shark-* before forwarding,
    then re-inject the authenticated identity. Upstream must never see
    the client-supplied spoofs."""
    base = spoof_env["base"]
    admin_key = spoof_env["admin_key"]

    _create_rule(
        base,
        admin_key,
        {
            "name": "spoof-gate",
            "pattern": "/echo/*",
            "methods": ["GET"],
            "require": "authenticated",
            "priority": 10,
        },
    )

    email = f"spoof_user_{int(time.time()*1000)}@test.com"
    user_id, token = _signup_and_token(base, email)

    spoofed = {
        "Authorization": f"Bearer {token}",
        "X-Shark-User-ID": "evil",
        "X-Shark-Scope": "admin",
        "X-Shark-App-ID": "hijack",
        "X-User-ID": "attacker",
        "X-User-Email": "attacker@evil.com",
        "X-User-Roles": "admin,superuser",
        "X-Agent-ID": "rogue-agent",
        "X-Agent-Name": "evil-agent",
        "X-Auth-Method": "forged",
    }
    r = requests.get(
        f"{base}/echo/ping",
        headers=spoofed,
        allow_redirects=False,
        timeout=5,
    )
    assert r.status_code == 200, r.text
    hdrs = _hdr_keys_lower(r.json())

    # The proxy injects the authenticated identity onto X-User-ID /
    # X-User-Email / X-Auth-Method — those values MUST be the authenticated
    # identity, not the client-supplied spoofs.
    assert hdrs.get("x-user-id") == user_id, hdrs
    assert hdrs.get("x-user-id") != "attacker"
    assert hdrs.get("x-user-email") == email, hdrs
    assert hdrs.get("x-user-email") != "attacker@evil.com"
    # Auth-method is the resolver-set value, never the forged string.
    assert hdrs.get("x-auth-method") in ("jwt", "dpop", "live"), hdrs
    assert hdrs.get("x-auth-method") != "forged"

    # Client-supplied X-Shark-* must be fully stripped — no "evil"/"admin"/
    # "hijack" substrings anywhere in the X-* headers. The proxy may emit
    # its own X-Shark-* (Cache-Age, Auth-Mode) so check the specific keys.
    assert hdrs.get("x-shark-user-id") is None, hdrs
    assert hdrs.get("x-shark-scope") is None, hdrs
    assert hdrs.get("x-shark-app-id") is None, hdrs

    # Agent headers should be absent on a non-agent (human) caller.
    assert hdrs.get("x-agent-id") is None, hdrs
    assert hdrs.get("x-agent-name") is None, hdrs

    # Defensive: no header value anywhere equals the spoofed strings.
    joined = " ".join(v.lower() for v in hdrs.values())
    assert "evil" not in joined
    assert "hijack" not in joined
    assert "attacker" not in joined
    assert "forged" not in joined


def test_anonymous_spoof_attempt_is_stripped_on_public_route(spoof_env):
    """On an anonymous-allowed route, a caller with no credentials but
    spoofed identity headers must still have those stripped before the
    upstream sees the request."""
    base = spoof_env["base"]
    admin_key = spoof_env["admin_key"]

    _create_rule(
        base,
        admin_key,
        {
            "name": "spoof-public",
            "pattern": "/public/*",
            "methods": ["GET"],
            "allow": "anonymous",
            "priority": 10,
        },
    )

    spoofed = {
        "X-Shark-User-ID": "evil",
        "X-Shark-Scope": "admin",
        "X-Shark-App-ID": "hijack",
        "X-User-ID": "attacker",
        "X-User-Email": "attacker@evil.com",
        "X-Agent-ID": "rogue-agent",
    }
    r = requests.get(
        f"{base}/public/ping",
        headers=spoofed,
        allow_redirects=False,
        timeout=5,
    )
    assert r.status_code == 200, r.text
    hdrs = _hdr_keys_lower(r.json())

    # On anonymous, the injected identity is empty; the spoofs must not
    # leak through under any X-* key.
    for key in (
        "x-shark-user-id",
        "x-shark-scope",
        "x-shark-app-id",
        "x-user-id",
        "x-user-email",
        "x-agent-id",
    ):
        assert hdrs.get(key) is None, f"{key}={hdrs.get(key)!r}"

    joined = " ".join(v.lower() for v in hdrs.values())
    assert "evil" not in joined
    assert "hijack" not in joined
    assert "attacker" not in joined
