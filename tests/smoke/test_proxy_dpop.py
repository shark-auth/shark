"""DPoP proof flow smoke test (Lane D — D2).

Covers the request-side surface that ships today:

- Client mints a DPoP proof via the ``shark_auth.dpop`` SDK helpers.
- Sends it through the embedded proxy alongside a Bearer JWT.
- Asserts the upstream receives the authenticated identity headers.
- Negative: missing Authorization → 401 from the auth layer; a DPoP
  header alone does not authenticate.

Scope note: the embedded proxy's DPoP *enforcement* path
(internal/proxy/proxy.go:validateDPoPProof) only fires when the resolved
Identity carries AuthMethodDPoP. The embedded-proxy resolvers currently
mark every bearer as AuthMethod="jwt" (see JWTResolver), so the DPoP
gate stays dormant end-to-end even when a DPoP proof is attached. This
test exercises the observable surface: the client-side helper emits a
valid proof JWT, and the proxy passes the request through untouched.
Full AuthMethodDPoP wiring is tracked for a later lane.

Contract: docs/proxy_v1_5/contracts/decision_kinds.md (scope-only refs)
SDK: sdk/python/shark_auth/dpop.py.
"""
import http.server
import json
import os
import re
import socket
import subprocess
import sys
import threading
import time

import pytest
import requests
import yaml

# Make the shark_auth SDK importable without a pip install — the smoke
# venv may not have it on sys.path.
_SDK_PATH = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "..", "..", "sdk", "python")
)
if _SDK_PATH not in sys.path:
    sys.path.insert(0, _SDK_PATH)

from shark_auth.dpop import DPoPProver  # noqa: E402


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


class _EchoUpstream:
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
                body = {
                    "path": self.path,
                    "x_user_id": self.headers.get("X-User-ID", ""),
                    "x_auth_method": self.headers.get("X-Auth-Method", ""),
                    "dpop": self.headers.get("DPoP", ""),
                    "authorization": self.headers.get("Authorization", ""),
                }
                self.wfile.write(json.dumps(body).encode())

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
def dpop_env(tmp_path):
    upstream = _EchoUpstream()
    upstream.start()

    port = find_free_port()
    base = f"http://127.0.0.1:{port}"
    cfg_path = str(tmp_path / "dpop.yaml")
    log_path = str(tmp_path / "dpop.log")

    cfg = {
        "server": {
            "port": port,
            "base_url": base,
            "secret": "dpop-smoke-secret-xxxxxxxxxxxxxxxxxxxxxxxx",
        },
        "auth": {
            "jwt": {
                "enabled": True,
                "mode": "session",
                "issuer": base,
                "audience": "shark-smoke",
            }
        },
        "proxy": {
            "enabled": True,
            "upstream": f"http://127.0.0.1:{upstream.port}",
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
        if not wait_for_health(f"{base}/healthz"):
            proc.terminate()
            proc.wait(timeout=5)
            log.close()
            with open(log_path) as f:
                pytest.fail(f"server failed to come up: {f.read()}")

        admin_key = None
        deadline = time.time() + 10
        while time.time() < deadline and admin_key is None:
            with open(log_path) as f:
                m = re.findall(r"sk_live_[A-Za-z0-9_-]{30,}", f.read())
            if m:
                admin_key = m[-1]
                break
            time.sleep(0.2)
        if not admin_key:
            pytest.fail("admin key never surfaced in server log")

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


def test_dpop_proof_passes_through_and_upstream_sees_identity(dpop_env):
    """Client sends Bearer JWT + DPoP proof to the proxy. The proxy
    authenticates via the JWT, injects canonical identity headers, and
    forwards the DPoP header untouched to the upstream."""
    base = dpop_env["base"]
    admin_key = dpop_env["admin_key"]

    # Rule: /dpop/* requires authenticated — anything non-anonymous.
    _create_rule(
        base,
        admin_key,
        {
            "name": "dpop-authn",
            "pattern": "/dpop/*",
            "methods": ["GET"],
            "require": "authenticated",
            "priority": 10,
        },
    )

    email = f"dpop_user_{int(time.time()*1000)}@test.com"
    user_id, token = _signup_and_token(base, email)

    prover = DPoPProver.generate()
    target = f"{base}/dpop/ping"
    proof = prover.make_proof("GET", target, access_token=token)

    r = requests.get(
        target,
        headers={
            "Authorization": f"Bearer {token}",
            "DPoP": proof,
        },
        allow_redirects=False,
        timeout=5,
    )
    assert r.status_code == 200, r.text
    body = r.json()
    assert body["path"] == "/dpop/ping"
    # The JWT resolver sets AuthMethod=jwt on the identity; we verify the
    # proxy propagated the identity rather than leaking the raw bearer.
    assert body["x_user_id"] == user_id
    assert body["x_auth_method"] in ("jwt", "dpop"), body
    # DPoP header should be forwarded for the upstream to validate if it
    # participates in the proof-of-possession scheme.
    assert body["dpop"] == proof
    # Proof itself parses as a dotted JWT.
    assert proof.count(".") == 2


def test_dpop_missing_bearer_returns_401(dpop_env):
    """A DPoP header alone does not authenticate. The proxy's auth layer
    requires a Bearer token; an anonymous call to a protected route with
    only a DPoP header must be rejected."""
    base = dpop_env["base"]
    admin_key = dpop_env["admin_key"]

    _create_rule(
        base,
        admin_key,
        {
            "name": "dpop-authn-neg",
            "pattern": "/dpoplocked/*",
            "methods": ["GET"],
            "require": "authenticated",
            "priority": 10,
        },
    )

    prover = DPoPProver.generate()
    target = f"{base}/dpoplocked/ping"
    proof = prover.make_proof("GET", target)

    r = requests.get(
        target,
        headers={"DPoP": proof, "Accept": "application/json"},
        allow_redirects=False,
        timeout=5,
    )
    assert r.status_code == 401, r.text
