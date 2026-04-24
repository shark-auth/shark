import json
import os
import re
import socket
import subprocess
import threading
import time

import pytest
import requests
import yaml

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
    """Section 72: W15 multi-listener proxy (embedded).

    Lane D port: v1.5 deprecated per-listener YAML ``rules:``. Rules now
    live in the DB and are scoped to applications via ``app_id``. Each
    listener inherits the shared ``AppResolver`` which looks up the
    application by the inbound request Host. We create two applications,
    one per listener (bound via ``proxy_public_domain``), and push the
    original 4-rule matrix into the DB via the admin API after boot.

    The 4 assertions below are unchanged from the pre-v1.5 version — the
    test just reaches the same states through the new configuration path.
    """
    p_upstream_a, p_upstream_b = toy_upstreams
    p_proxy_a, p_proxy_b = find_free_port(), find_free_port()
    p_admin = find_free_port()

    db_path = str(tmp_path / "w15.db")
    cfg_path = str(tmp_path / "w15.yaml")
    log_path = str(tmp_path / "w15_server.log")

    # proxy_public_domain (the AppResolver key) matches the Host we'll
    # send from the client via requests. Hit 127.0.0.1:<port> so the
    # resolver's with-port lookup matches directly.
    host_a = f"127.0.0.1:{p_proxy_a}"
    host_b = f"127.0.0.1:{p_proxy_b}"

    cfg = {
        "server": {
            "port": p_admin,
            "base_url": f"http://127.0.0.1:{p_admin}",
            "secret": "w15-smoke-secret-xxxxxxxxxxxxxxxxxxxxxxxxxxx",
        },
        "storage": {"path": db_path},
        "auth": {
            "jwt": {
                "enabled": True,
                "mode": "session",
                "issuer": f"http://127.0.0.1:{p_admin}",
                "audience": "shark-smoke",
            }
        },
        "proxy": {
            "listeners": [
                {
                    "bind": f"127.0.0.1:{p_proxy_a}",
                    "upstream": f"http://127.0.0.1:{p_upstream_a}",
                },
                {
                    "bind": f"127.0.0.1:{p_proxy_b}",
                    "upstream": f"http://127.0.0.1:{p_upstream_b}",
                },
            ]
        },
    }

    with open(cfg_path, "w") as f:
        yaml.dump(cfg, f)

    bin_abs = os.path.abspath(BIN_PATH)
    log = open(log_path, "w")
    proc = subprocess.Popen(
        [bin_abs, "serve", "--dev", "--config", cfg_path],
        stdout=log,
        stderr=log,
        cwd=str(tmp_path),
    )

    try:
        admin_base = f"http://127.0.0.1:{p_admin}"
        if not wait_for_health(f"{admin_base}/healthz"):
            with open(log_path) as f:
                pytest.fail(f"admin port failed to come up: {f.read()}")
        assert wait_for_port(p_proxy_a), "proxy A listener never bound"
        assert wait_for_port(p_proxy_b), "proxy B listener never bound"

        # Scrape the admin key from the server log.
        admin_key = None
        deadline = time.time() + 10
        while time.time() < deadline and admin_key is None:
            with open(log_path) as f:
                m = re.findall(r"sk_live_[A-Za-z0-9_-]{30,}", f.read())
            if m:
                admin_key = m[-1]
                break
            time.sleep(0.2)
        assert admin_key, "admin key never surfaced in server log"
        auth = {"Authorization": f"Bearer {admin_key}"}

        # Create two applications, each bound to one listener via
        # proxy_public_domain. integration_mode=proxy is required by the
        # DBAppResolver's enforcement branch.
        def create_app(slug, public_domain, upstream):
            r = requests.post(
                f"{admin_base}/api/v1/admin/apps",
                headers=auth,
                json={
                    "name": slug.upper(),
                    "slug": slug,
                    "integration_mode": "proxy",
                    "proxy_public_domain": public_domain,
                    "proxy_protected_url": upstream,
                },
                timeout=5,
            )
            assert r.status_code == 201, r.text
            return r.json()["id"]

        app_a = create_app("w15a", host_a, f"http://127.0.0.1:{p_upstream_a}")
        app_b = create_app("w15b", host_b, f"http://127.0.0.1:{p_upstream_b}")

        # Post rules scoped to each app via the DB-backed endpoint.
        # Priority ordering matters: /public/* must beat /* in the
        # engine's evaluation order.
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
        create_rule({
            "app_id": app_b,
            "name": "w15b-open",
            "pattern": "/*",
            "methods": ["GET"],
            "allow": "anonymous",
            "priority": 100,
        })

        # Reload so any listener-local engine picks up the new DB state.
        requests.post(
            f"{admin_base}/api/v1/admin/proxy/reload", headers=auth, timeout=5,
        )

        # 1. Listener A: anonymous on public -> 200, tag=A.
        r = requests.get(
            f"http://127.0.0.1:{p_proxy_a}/public/foo",
            headers={"Host": host_a},
            allow_redirects=False,
            timeout=5,
        )
        assert r.status_code == 200, r.text
        assert r.json()["tag"] == "A"

        # 2. Listener A: anonymous on private -> 401.
        r = requests.get(
            f"http://127.0.0.1:{p_proxy_a}/secret",
            headers={"Host": host_a, "Accept": "application/json"},
            allow_redirects=False,
            timeout=5,
        )
        assert r.status_code == 401, r.text

        # 3. Listener B: anonymous on any -> 200, tag=B.
        r = requests.get(
            f"http://127.0.0.1:{p_proxy_b}/any",
            headers={"Host": host_b},
            allow_redirects=False,
            timeout=5,
        )
        assert r.status_code == 200, r.text
        assert r.json()["tag"] == "B"

        # 4. Isolation: Listener A should NOT hit B.
        r = requests.get(
            f"http://127.0.0.1:{p_proxy_a}/public/ping",
            headers={"Host": host_a},
            allow_redirects=False,
            timeout=5,
        )
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
