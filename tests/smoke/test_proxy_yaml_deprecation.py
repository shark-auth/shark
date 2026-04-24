"""YAML proxy.rules deprecation warning smoke test (Lane D — D4).

Writes a config file that still carries the legacy `proxy.rules:` block,
starts `shark serve`, and asserts the deprecation WARNING hits stderr
and that no rules were actually loaded (the proxy_rules table is empty,
so any `POST /proxy/simulate` against the purported pattern is denied
by the default-deny engine).

Contract: docs/proxy_v1_5/migration/yaml_deprecation.md
Warning source: internal/server/server.go ~line 108.
"""
import os
import re
import socket
import subprocess
import time

import pytest
import requests


BIN_PATH = "./shark.exe" if os.name == "nt" else "./shark"

DEPRECATION_SUBSTRING = (
    "WARNING: proxy.rules YAML section is deprecated in v1.5"
)


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
def legacy_yaml_server(tmp_path):
    """Start shark serve with a YAML file that still carries the legacy
    proxy.rules: block. Returns (base, admin_key, log_contents_fn)."""
    port = find_free_port()
    base = f"http://127.0.0.1:{port}"
    cfg_path = str(tmp_path / "legacy_proxy_rules.yaml")
    log_path = str(tmp_path / "legacy.log")

    # Hand-written YAML so we can force the legacy layout (yaml.dump would
    # not serialize keys the struct no longer has).
    cfg_yaml = f"""server:
  port: {port}
  base_url: {base}
  secret: yaml-dep-smoke-secret-xxxxxxxxxxxxxxxxxxxxxxxxx
auth:
  jwt:
    enabled: true
    mode: session
    issuer: {base}
    audience: shark-smoke
proxy:
  enabled: true
  upstream: http://127.0.0.1:{find_free_port()}
  rules:
    - path: /legacy/*
      allow: anonymous
    - path: /admin/*
      require: role:admin
"""
    with open(cfg_path, "w") as f:
        f.write(cfg_yaml)

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

        def read_log():
            # stdout + stderr are merged into log_path by Popen, so this
            # captures the deprecation stderr warning too.
            with open(log_path) as f:
                return f.read()

        yield {
            "base": base,
            "admin_key": admin_key,
            "read_log": read_log,
        }
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        log.close()


def test_yaml_proxy_rules_emits_deprecation_warning(legacy_yaml_server):
    """Boot with a legacy proxy.rules: block → WARNING hits stderr."""
    log = legacy_yaml_server["read_log"]()
    assert DEPRECATION_SUBSTRING in log, (
        f"expected deprecation warning in log, got:\n{log[:2000]}"
    )


def test_yaml_proxy_rules_are_not_loaded(legacy_yaml_server):
    """Despite the YAML carrying two rules, none of them get wired into
    the engine. The DB-backed list endpoint returns an empty set (modulo
    whatever other tests have left behind, which — with the isolated
    tmp_path CWD — is none)."""
    base = legacy_yaml_server["base"]
    admin_key = legacy_yaml_server["admin_key"]

    r = requests.get(
        f"{base}/api/v1/admin/proxy/rules/db",
        headers={"Authorization": f"Bearer {admin_key}"},
        timeout=5,
    )
    assert r.status_code == 200, r.text
    rules = r.json().get("data", [])
    # Legacy YAML declared two rules but v1.5 ignores them — the table
    # is the single source of truth and starts empty.
    assert rules == [] or len(rules) == 0, (
        f"expected zero DB rules from legacy YAML, got: {rules}"
    )

    # As an extra observability signal, /proxy/rules (the legacy
    # in-memory engine view) also has no entries for the legacy patterns.
    r = requests.get(
        f"{base}/api/v1/admin/proxy/rules",
        headers={"Authorization": f"Bearer {admin_key}"},
        timeout=5,
    )
    if r.status_code == 200:
        engine_rules = r.json().get("data", [])
        for rule in engine_rules:
            # None of the rules we defined (path /legacy/* or /admin/*)
            # should be visible through this view either.
            pattern = rule.get("pattern") or rule.get("path") or ""
            assert pattern not in ("/legacy/*", "/admin/*"), (
                f"legacy YAML rule leaked into engine: {rule}"
            )
