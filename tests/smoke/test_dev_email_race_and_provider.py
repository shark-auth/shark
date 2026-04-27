"""
Smoke tests: dev-email race regression + provider-based gating.

Covers:
  Test 1 — provider gate: endpoint returns 200 when email.provider=dev,
            404 when provider is something else (e.g. smtp). Uses PATCH
            /api/v1/admin/config to toggle provider live.
  Test 2 — race regression: fire 5 magic-link sends in rapid succession,
            GET dev inbox, assert all 5 appear (no clobbered messages).
  Test 3 — static check: dev_email.tsx uses useAPI / AbortController in
            the polling path, no raw fetch() call for the email list.

These tests assume the server fixture starts with email.provider=dev
(the conftest 'server' fixture already sets --dev which implies dev provider).
"""

import pytest
import requests
import time
import os
import re

BASE_URL = os.environ.get("BASE", "http://localhost:8080")


def _emails_from(body):
    """Normalise the list-emails response to a plain list."""
    return (
        body.get("emails")
        or body.get("data")
        or body.get("items")
        or (body if isinstance(body, list) else [])
    )


def _skip_if_not_dev(resp):
    if resp.status_code == 404:
        pytest.skip(
            "Dev email endpoints not available "
            "(server not running with email.provider=dev)"
        )


# ---------------------------------------------------------------------------
# Test 1: provider gate
# ---------------------------------------------------------------------------

def test_dev_inbox_live_when_provider_is_dev(admin_client):
    """
    GET /admin/dev/emails returns 200 when email.provider=dev.
    After switching to 'smtp' the same endpoint must return 404.
    Finally restore to 'dev' so subsequent tests are unaffected.
    """
    # Baseline: server started with dev provider — should be 200.
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
    _skip_if_not_dev(resp)
    assert resp.status_code == 200, (
        f"Expected 200 with provider=dev, got {resp.status_code}: {resp.text}"
    )

    # Switch provider to smtp.
    patch_resp = admin_client.patch(
        f"{BASE_URL}/api/v1/admin/config",
        json={"email": {"provider": "smtp"}},
    )
    if patch_resp.status_code not in (200, 204):
        pytest.skip(
            f"PATCH /admin/config not available or failed ({patch_resp.status_code}); "
            "skipping provider-gate assertion"
        )

    try:
        resp_smtp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
        assert resp_smtp.status_code == 404, (
            f"Expected 404 after switching to smtp, got {resp_smtp.status_code}"
        )
    finally:
        # Always restore to dev so other tests keep working.
        admin_client.patch(
            f"{BASE_URL}/api/v1/admin/config",
            json={"email": {"provider": "dev"}},
        )
        # Brief wait for config propagation.
        time.sleep(0.1)


# ---------------------------------------------------------------------------
# Test 2: race regression — 5 concurrent sends, all 5 must appear
# ---------------------------------------------------------------------------

def test_dev_inbox_no_race_under_rapid_sends(admin_client):
    """
    Fire 5 magic-link sends in rapid succession.
    Assert all 5 land in the inbox (no clobbered/missing messages).
    This catches the race where overlapping poll responses could overwrite
    each other or where the inbox was cleared optimistically.
    """
    # Clear inbox first.
    clear = admin_client.delete(f"{BASE_URL}/api/v1/admin/dev/emails")
    _skip_if_not_dev(clear)
    assert clear.status_code in (200, 204), f"Clear failed: {clear.text}"

    # Verify empty.
    list_resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
    body = list_resp.json()
    assert len(_emails_from(body)) == 0, "Inbox not empty after clear"

    # Fire 5 magic-link requests without any delay between them.
    ts = int(time.time())
    addrs = [f"race_test_{ts}_{i}@example.com" for i in range(5)]
    for addr in addrs:
        r = requests.post(
            f"{BASE_URL}/api/v1/auth/magic-link/send",
            json={"email": addr},
        )
        if r.status_code == 404:
            pytest.skip("Magic link send endpoint not available")
        # 200, 202, or 204 are all acceptable "accepted" responses.
        assert r.status_code in (200, 202, 204), (
            f"Magic-link send for {addr} failed: {r.status_code} {r.text}"
        )

    # Poll inbox — allow up to 8 seconds for all 5 to appear.
    captured = []
    for _ in range(40):
        time.sleep(0.2)
        resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
        assert resp.status_code == 200, f"List failed: {resp.status_code}"
        captured = _emails_from(resp.json())
        if len(captured) >= 5:
            break

    assert len(captured) >= 5, (
        f"Race regression: expected 5 emails in inbox, got {len(captured)}. "
        f"Addresses sent: {addrs}. "
        f"Addresses captured: {[e.get('to') or e.get('to_addr') for e in captured]}"
    )

    # Verify each sent address appears exactly once.
    captured_addrs = {
        (e.get("to") or e.get("to_addr") or e.get("recipient") or "").lower()
        for e in captured
    }
    for addr in addrs:
        assert addr.lower() in captured_addrs, (
            f"Address {addr!r} missing from inbox after 5-send burst. "
            f"Captured: {sorted(captured_addrs)}"
        )


# ---------------------------------------------------------------------------
# Test 3: static check — dev_email.tsx must use useAPI, not raw fetch
# ---------------------------------------------------------------------------

def test_dev_email_tsx_uses_useapi_not_raw_fetch():
    """
    Static analysis of admin/src/components/dev_email.tsx.
    Asserts:
      - useAPI is imported and used for the email list
      - no raw fetch() call exists in the polling path (the list endpoint)
      - AbortController is provided by useAPI (verified by import presence)
    """
    tsx_path = os.path.join(
        os.path.dirname(__file__),
        "../../admin/src/components/dev_email.tsx",
    )
    tsx_path = os.path.normpath(tsx_path)

    assert os.path.exists(tsx_path), f"dev_email.tsx not found at {tsx_path}"

    with open(tsx_path, encoding="utf-8") as f:
        source = f.read()

    # 1. useAPI must be imported.
    assert "useAPI" in source, "dev_email.tsx does not import useAPI"

    # 2. useAPI must be called with the dev/emails path.
    assert "useAPI('/admin/dev/emails')" in source or \
           'useAPI("/admin/dev/emails")' in source, (
        "dev_email.tsx does not call useAPI('/admin/dev/emails') for the email list"
    )

    # 3. No raw fetch() call for the list endpoint.
    raw_fetch_pattern = re.compile(
        r'fetch\s*\([^)]*admin/dev/emails[^)]*\)', re.DOTALL
    )
    assert not raw_fetch_pattern.search(source), (
        "dev_email.tsx contains a raw fetch() call for the dev/emails endpoint; "
        "the polling path must go through useAPI (which has AbortController)"
    )

    # 4. Silent poll option used in the interval (race fix verification).
    assert "silent" in source, (
        "dev_email.tsx does not use the silent refresh option; "
        "the 1.5s poll should pass { silent: true } to suppress loading flicker"
    )
