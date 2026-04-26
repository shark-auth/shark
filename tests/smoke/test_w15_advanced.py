import json
import os
import socket
import subprocess
import threading
import time

import pytest
import requests

BIN_PATH = "./shark.exe" if os.name == 'nt' else "./shark"


def find_free_port():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(('', 0))
        return s.getsockname()[1]


def wait_for_port(port, timeout=15):
    start = time.time()
    while time.time() - start < timeout:
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=1):
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


@pytest.fixture
def toy_upstreams():
    """Starts two echo upstreams for W15 isolation testing."""
    import http.server

    class EchoHandler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            tag = self.server.tag
            self.send_response(200)
            self.send_header("X-Backend-Tag", tag)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"tag": tag, "path": self.path}).encode())

        def log_message(self, *args):
            pass

    servers = []

    def serve(port, tag):
        srv = http.server.HTTPServer(("127.0.0.1", port), EchoHandler)
        srv.tag = tag
        servers.append(srv)
        srv.serve_forever()

    p1, p2 = find_free_port(), find_free_port()
    t1 = threading.Thread(target=serve, args=(p1, "A"), daemon=True)
    t2 = threading.Thread(target=serve, args=(p2, "B"), daemon=True)
    t1.start()
    t2.start()

    yield (p1, p2)

    for srv in servers:
        try:
            srv.shutdown()
            srv.server_close()
        except Exception:
            pass


def test_w15_multi_listener_isolation(toy_upstreams, tmp_path):
    """Section 72: W15 admin-API-driven proxy rule enforcement.

    W17 removed per-listener YAML ``rules:`` and multi-listener yaml config.
    Rules live in the DB and are pushed via the admin API after boot. This
    test verifies the same rule-enforcement behaviour (public open, private
    auth-required) using a single --proxy-upstream server started with
    SHARK_PORT / SHARK_DB_PATH env vars instead of --config.

    Note: multi-listener isolation (tag=A vs tag=B via separate listener
    ports) is not exercisable without yaml listener config, which was
    hard-removed in W17 Phase H. The core invariants — anonymous access to
    /public/* passes, anonymous access to protected /* is denied — are
    unchanged and remain tested here.
    """
    p_upstream_a, _p_upstream_b = toy_upstreams
    p_admin = find_free_port()
    log_path = str(tmp_path / "w15_server.log")

    bin_abs = os.path.abspath(BIN_PATH)
    log = open(log_path, "w")
    env = os.environ.copy()
    env["SHARK_PORT"] = str(p_admin)
    env["SHARK_DB_PATH"] = str(tmp_path / "w15.db")
    proc = subprocess.Popen(
        [bin_abs, "serve", "--no-prompt", "--proxy-upstream", f"http://127.0.0.1:{p_upstream_a}"],
        stdout=log,
        stderr=log,
        cwd=str(tmp_path),
        env=env,
    )

    try:
        admin_base = f"http://127.0.0.1:{p_admin}"
        # 30 s — Windows SQLite bootstrap can take several seconds.
        if not wait_for_health(f"{admin_base}/healthz", timeout=30):
            with open(log_path) as f:
                pytest.fail(f"admin port failed to come up: {f.read()}")

        # W17: admin key written to <db_dir>/admin.key.firstboot.
        key_file = tmp_path / "admin.key.firstboot"
        for _ in range(50):
            if key_file.exists():
                break
            time.sleep(0.1)
        assert key_file.exists(), f"admin key file never appeared at {key_file}"
        admin_key = key_file.read_text().strip()
        auth = {"Authorization": f"Bearer {admin_key}"}

        # Create an application bound to the single upstream.
        host_a = f"127.0.0.1:{p_admin}"
        r = requests.post(
            f"{admin_base}/api/v1/admin/apps",
            headers=auth,
            json={
                "name": "W15A",
                "slug": "w15a",
                "integration_mode": "proxy",
                "proxy_public_domain": host_a,
                "proxy_protected_url": f"http://127.0.0.1:{p_upstream_a}",
            },
            timeout=5,
        )
        assert r.status_code == 201, r.text
        app_a = r.json()["id"]

        # Post rules via the DB-backed endpoint.
        # Priority ordering: /public/* must beat /* in the engine's eval order.
        def create_rule(payload):
            r = requests.post(
                f"{admin_base}/api/v1/admin/proxy/rules/db",
                headers=auth,
                json=payload,
                timeout=5,
            )
            assert r.status_code == 201, r.text

        create_rule({
            "app_id": app_a,
            "name": "w15a-public",
            "pattern": "/public/*",
            "methods": ["GET"],
            "allow": "anonymous",
            "priority": 100,
        })
        create_rule({
            "app_id": app_a,
            "name": "w15a-catchall",
            "pattern": "/*",
            "methods": ["GET"],
            "require": "authenticated",
            "priority": 10,
        })

        # Reload so the engine picks up the new DB state.
        requests.post(
            f"{admin_base}/api/v1/admin/proxy/reload", headers=auth, timeout=5,
        )

        # 1. Anonymous on /public/* -> 200, upstream echoes tag=A.
        r = requests.get(
            f"{admin_base}/public/foo",
            allow_redirects=False,
            timeout=5,
        )
        assert r.status_code == 200, r.text
        assert r.json()["tag"] == "A"

        # 2. Anonymous on a protected path -> 401.
        r = requests.get(
            f"{admin_base}/secret",
            headers={"Accept": "application/json"},
            allow_redirects=False,
            timeout=5,
        )
        assert r.status_code == 401, r.text

        # 3. Anonymous on another public path still hits tag=A upstream.
        r = requests.get(
            f"{admin_base}/public/ping",
            allow_redirects=False,
            timeout=5,
        )
        assert r.status_code == 200, r.text
        assert r.json()["tag"] == "A"

    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        log.close()


# test_w15_standalone_proxy_jwt_verify: DELETED per LANE_D_SCOPE.md §D6.
# The `shark proxy` standalone command was deprecated in v1.5 Lane B —
# it prints a deprecation stub and exits 2. The reverse proxy lives
# inside `shark serve`, configured via the Admin API. There is no
# equivalent path to port this test to, so it is removed rather than
# left as permanent xfail. Embedded-proxy JWT verification is covered
# by tests/smoke/test_proxy_tier_gate.py and test_proxy_dpop.py.
