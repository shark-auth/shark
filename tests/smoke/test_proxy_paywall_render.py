"""Paywall HTML render smoke test (Lane D — D2).

Hits `GET /paywall/{slug}?tier=...&return=...` directly (public route, no
auth) and asserts the inline HTML template rendered correctly with the
app's branding tokens + tier label + return URL.

Contract: docs/proxy_v1_5/api/paywall_route.md
Handler: internal/api/hosted_handlers.go handlePaywallPage.

The server is spun up on a fresh port + tmp_path DB so this test doesn't
share state with conftest's session server (different branding, slug
collisions).
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
def paywall_env(tmp_path):
    port = find_free_port()
    base = f"http://127.0.0.1:{port}"
    cfg_path = str(tmp_path / "paywall.yaml")
    log_path = str(tmp_path / "paywall.log")

    cfg = {
        "server": {
            "port": port,
            "base_url": base,
            "secret": "paywall-smoke-secret-xxxxxxxxxxxxxxxxxxxxxx",
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


def _create_app(base, admin_key, slug, name="Paywall Demo App"):
    r = requests.post(
        f"{base}/api/v1/admin/apps",
        headers=_admin(admin_key),
        json={
            "name": name,
            "slug": slug,
            "integration_mode": "proxy",
        },
        timeout=5,
    )
    assert r.status_code == 201, r.text
    return r.json()


def test_paywall_render_happy_path(paywall_env):
    """/paywall/{slug}?tier=pro&return=... returns 402 + HTML carrying
    tier label, app name, upgrade CTA pointed at the return URL, and the
    branding CSS variables from ResolveBranding."""
    base = paywall_env["base"]
    admin_key = paywall_env["admin_key"]

    slug = f"paywall-render-{int(time.time()*1000)}"
    app_name = "Paywall Render Demo"
    _create_app(base, admin_key, slug, name=app_name)

    # Also patch design tokens on the global branding row. These tokens
    # aren't embedded in the current paywall template directly (paywall
    # renders the classic primary/secondary/font triple), but exercising
    # the PATCH ensures the Lane A plumbing doesn't 500 at render time.
    r = requests.patch(
        f"{base}/api/v1/admin/branding/design-tokens",
        headers=_admin(admin_key),
        json={
            "design_tokens": {
                "colors": {"primary": "#6366f1", "secondary": "#4f46e5"},
                "typography": {"font_family": "Inter, system-ui, sans-serif"},
            }
        },
        timeout=5,
    )
    assert r.status_code == 200, r.text

    return_url = "https://example.com/app/dashboard"
    r = requests.get(
        f"{base}/paywall/{slug}",
        params={"tier": "pro", "return": return_url},
        timeout=5,
    )
    assert r.status_code == 402, r.text
    assert r.headers["Content-Type"].startswith("text/html")

    body = r.text
    # Tier label appears in the headline + CTA per template.
    assert "pro" in body.lower(), body
    # App name appears in the <title> and in the .app div at the bottom.
    assert app_name in body, body
    # Return URL is embedded in the upgrade CTA href, possibly URL-encoded.
    assert (
        return_url in body
        or "example.com/app/dashboard" in body
    ), body
    # Branding CSS custom properties are present (default fallback colors
    # when no per-app branding row has been provisioned).
    assert "--shark-primary" in body
    assert "--shark-secondary" in body
    assert "--font-display" in body


def test_paywall_render_missing_tier_returns_400(paywall_env):
    base = paywall_env["base"]
    admin_key = paywall_env["admin_key"]

    slug = f"paywall-missing-tier-{int(time.time()*1000)}"
    _create_app(base, admin_key, slug)

    r = requests.get(f"{base}/paywall/{slug}", timeout=5)
    assert r.status_code == 400, r.text
    assert "tier" in r.text.lower()


def test_paywall_render_unknown_slug_returns_404(paywall_env):
    base = paywall_env["base"]
    r = requests.get(
        f"{base}/paywall/never-created-app-xyz",
        params={"tier": "pro"},
        timeout=5,
    )
    assert r.status_code == 404, r.text
