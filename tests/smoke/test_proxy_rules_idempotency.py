"""Proxy rules idempotency smoke test (Lane D — D5).

Validates the idempotency invariants that the rules admin API ships
today:

1. POST with a fresh payload → 201 Created, server mints an id.
2. PATCH same rule with new fields → 200 OK, list count unchanged,
   fields updated in place.
3. DELETE → 204, list count drops by one.
4. Repeated identical POSTs do NOT collapse (each mints a new id) —
   the DB does not enforce name uniqueness. This matches the current
   storage schema; the client-id-upsert pattern described in D5 of
   LANE_D_SCOPE.md is tracked for the Lane E CLI port, which will use
   `--id` to drive upsert semantics on top of the same HTTP surface.

Contract: docs/proxy_v1_5/api/admin_proxy_rules_db.md +
          docs/proxy_v1_5/contracts/rule_shape.md
Handler: internal/api/proxy_rules_handlers.go
"""
import os
import re
import socket
import subprocess
import time

import pytest
import requests
import yaml


BIN_PATH = "./shark.exe" if os.name == "nt" else "./shark"


def find_free_port():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("", 0))
        return s.getsockname()[1]


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
def idem_env(tmp_path):
    port = find_free_port()
    base = f"http://127.0.0.1:{port}"
    cfg_path = str(tmp_path / "idem.yaml")
    log_path = str(tmp_path / "idem.log")

    cfg = {
        "server": {
            "port": port,
            "base_url": base,
            "secret": "idem-smoke-secret-xxxxxxxxxxxxxxxxxxxxxxxxxx",
        },
        "auth": {
            "jwt": {
                "enabled": True,
                "mode": "session",
                "issuer": base,
                "audience": "shark-smoke",
            }
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


def _admin(k):
    return {"Authorization": f"Bearer {k}"}


def _list_rules(base, admin_key):
    r = requests.get(
        f"{base}/api/v1/admin/proxy/rules/db",
        headers=_admin(admin_key),
        timeout=5,
    )
    assert r.status_code == 200, r.text
    payload = r.json()
    return payload.get("data", []) or []


def test_proxy_rules_post_patch_delete_idempotency(idem_env):
    """POST → PATCH → PATCH → DELETE. Tests that repeated PATCHes are
    idempotent (same fields, no new row), list count is stable across
    updates, and DELETE cleanly removes the row."""
    base = idem_env["base"]
    admin_key = idem_env["admin_key"]

    baseline = _list_rules(base, admin_key)
    start_count = len(baseline)

    # 1. POST — 201, server-generated id.
    payload = {
        "name": "idem-rule-1",
        "pattern": "/idem/*",
        "methods": ["GET"],
        "require": "authenticated",
        "priority": 50,
    }
    r = requests.post(
        f"{base}/api/v1/admin/proxy/rules/db",
        headers=_admin(admin_key),
        json=payload,
        timeout=5,
    )
    assert r.status_code == 201, r.text
    rule = r.json()["data"]
    rid = rule["id"]
    assert rid, rule
    assert rule["pattern"] == "/idem/*"
    assert rule["priority"] == 50

    after_create = _list_rules(base, admin_key)
    assert len(after_create) == start_count + 1

    # 2. PATCH same rule with new priority — 200, list count unchanged.
    r = requests.patch(
        f"{base}/api/v1/admin/proxy/rules/db/{rid}",
        headers=_admin(admin_key),
        json={"priority": 75},
        timeout=5,
    )
    assert r.status_code == 200, r.text
    updated = r.json()["data"]
    assert updated["id"] == rid
    assert updated["priority"] == 75

    after_patch = _list_rules(base, admin_key)
    assert len(after_patch) == start_count + 1, after_patch

    # 3. Repeated PATCH with the same body — still 200, still no new row,
    # priority stays 75.
    r = requests.patch(
        f"{base}/api/v1/admin/proxy/rules/db/{rid}",
        headers=_admin(admin_key),
        json={"priority": 75},
        timeout=5,
    )
    assert r.status_code == 200, r.text
    assert r.json()["data"]["priority"] == 75

    again = _list_rules(base, admin_key)
    assert len(again) == start_count + 1, again

    # 4. DELETE — list count drops by one.
    r = requests.delete(
        f"{base}/api/v1/admin/proxy/rules/db/{rid}",
        headers=_admin(admin_key),
        timeout=5,
    )
    assert r.status_code in (200, 204), r.text

    after_delete = _list_rules(base, admin_key)
    assert len(after_delete) == start_count, after_delete


def test_proxy_rule_post_does_not_upsert_on_repeated_body(idem_env):
    """Documents the current behaviour: POST always mints a new id,
    even when the wire payload is byte-identical. Upsert semantics by
    client-id are tracked for the Lane E CLI port (per LANE_D_SCOPE.md)
    which will layer `--id`-driven upsert on top of the same HTTP
    endpoints. If a future change adds upsert-on-POST, this test will
    go red and should be updated.
    """
    base = idem_env["base"]
    admin_key = idem_env["admin_key"]

    payload = {
        "name": "idem-dup-probe",
        "pattern": "/idem-dup/*",
        "methods": ["GET"],
        "allow": "anonymous",
        "priority": 10,
    }

    r1 = requests.post(
        f"{base}/api/v1/admin/proxy/rules/db",
        headers=_admin(admin_key),
        json=payload,
        timeout=5,
    )
    assert r1.status_code == 201, r1.text
    r2 = requests.post(
        f"{base}/api/v1/admin/proxy/rules/db",
        headers=_admin(admin_key),
        json=payload,
        timeout=5,
    )
    assert r2.status_code == 201, r2.text

    id1 = r1.json()["data"]["id"]
    id2 = r2.json()["data"]["id"]
    assert id1 != id2, (id1, id2)

    # Clean up both rows so the fixture's DB doesn't pollute a sibling test.
    for rid in (id1, id2):
        requests.delete(
            f"{base}/api/v1/admin/proxy/rules/db/{rid}",
            headers=_admin(admin_key),
            timeout=5,
        )
