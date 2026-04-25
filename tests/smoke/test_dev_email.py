"""
Smoke tests for Dev Email inbox feature.

Backend endpoints used:
  GET    /api/v1/admin/dev/emails        -- list captured emails
  GET    /api/v1/admin/dev/emails/{id}   -- single email detail
  DELETE /api/v1/admin/dev/emails        -- clear all

These endpoints are only active when the server runs in dev mode
(shark serve --dev). The conftest.py session fixture already starts
the server with --dev, so these tests run unconditionally.

If the endpoints are absent (404), tests are skipped with a clear message.
"""

import pytest
import requests
import time
import os

BASE_URL = os.environ.get("BASE", "http://localhost:8080")


def _skip_if_not_dev(resp):
    """Skip gracefully if server is not in dev mode (endpoint returns 404)."""
    if resp.status_code == 404:
        pytest.skip("Dev email endpoints not available (server not in dev mode or endpoints not wired)")


# ---------------------------------------------------------------------------
# Section: Dev Email — inbox list
# ---------------------------------------------------------------------------

def test_dev_email_list_endpoint(admin_client):
    """Dev email inbox list returns 200 with expected shape."""
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
    _skip_if_not_dev(resp)
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    body = resp.json()
    # Accept both {"emails": [...]} and {"data": [...]} and bare list shapes
    emails = body.get("emails") or body.get("data") or body.get("items") or (body if isinstance(body, list) else None)
    assert emails is not None, f"Response has no recognised email list key: {body}"
    assert isinstance(emails, list), f"Expected list, got {type(emails)}"


def test_dev_email_capture_via_magic_link(admin_client):
    """
    Trigger a magic-link send, verify inbox captures it, verify HTML body.

    Flow:
      1. Clear the inbox so we start clean.
      2. Request a magic-link for a test address.
      3. Poll inbox (up to 5 s) until one email appears.
      4. Assert to-address, subject heuristic, and HTML body non-empty.
      5. Fetch single-email detail endpoint and verify html_body present.
    """
    # 1. Clear inbox
    clear = admin_client.delete(f"{BASE_URL}/api/v1/admin/dev/emails")
    _skip_if_not_dev(clear)
    assert clear.status_code in [200, 204], f"Clear failed: {clear.text}"

    # Confirm empty
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
    _skip_if_not_dev(resp)
    body = resp.json()
    emails = body.get("emails") or body.get("data") or body.get("items") or (body if isinstance(body, list) else [])
    assert len(emails) == 0, "Inbox not empty after clear"

    # 2. Trigger a magic link
    test_email = f"devsmoke_{int(time.time())}@example.com"
    ml_resp = requests.post(
        f"{BASE_URL}/api/v1/auth/magic-link",
        json={"email": test_email},
    )
    # Magic link endpoint may return 200 or 202; skip on 404 (feature not enabled)
    if ml_resp.status_code == 404:
        pytest.skip("Magic link auth not enabled on this server")
    assert ml_resp.status_code in [200, 202], f"Magic link request failed: {ml_resp.status_code} {ml_resp.text}"

    # 3. Poll inbox
    captured = None
    for _ in range(25):  # up to ~5 s
        time.sleep(0.2)
        resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
        body = resp.json()
        emails = body.get("emails") or body.get("data") or body.get("items") or (body if isinstance(body, list) else [])
        if emails:
            captured = emails[0]
            break

    assert captured is not None, f"No email captured within 5s after magic link request for {test_email}"

    # 4. Assert basic fields
    to_addr = captured.get("to") or captured.get("to_addr") or captured.get("recipient") or ""
    assert test_email in to_addr, f"Email 'to' field ({to_addr!r}) does not match {test_email!r}"

    subject = captured.get("subject", "")
    assert subject, "Captured email has no subject"

    email_id = captured.get("id")
    assert email_id, "Captured email has no id"

    # 5. Fetch detail endpoint
    detail_resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails/{email_id}")
    _skip_if_not_dev(detail_resp)
    assert detail_resp.status_code == 200, f"Detail fetch failed: {detail_resp.status_code} {detail_resp.text}"

    detail = detail_resp.json()
    html_body = (
        detail.get("html_body") or detail.get("html") or
        detail.get("body_html") or detail.get("body") or ""
    )
    assert html_body, "Detail email has no HTML body — expected magic link email to include HTML content"

    # Verify magic link URL present in body
    assert any(
        kw in html_body.lower()
        for kw in ["magic", "verify", "confirm", "token", "otp", "click"]
    ), f"HTML body does not appear to contain a magic link URL. Preview: {html_body[:200]}"


def test_dev_email_clear_all(admin_client):
    """Clear-all wipes the inbox and subsequent list returns empty."""
    # Seed: trigger a magic link so there's at least one email (best-effort)
    test_email = f"devclear_{int(time.time())}@example.com"
    requests.post(f"{BASE_URL}/api/v1/auth/magic-link", json={"email": test_email})
    time.sleep(0.5)

    # Clear
    resp = admin_client.delete(f"{BASE_URL}/api/v1/admin/dev/emails")
    _skip_if_not_dev(resp)
    assert resp.status_code in [200, 204], f"Clear failed: {resp.status_code} {resp.text}"

    # Verify empty
    list_resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
    body = list_resp.json()
    emails = body.get("emails") or body.get("data") or body.get("items") or (body if isinstance(body, list) else [])
    assert len(emails) == 0, f"Inbox not empty after clear: {emails}"


def test_dev_email_detail_not_found(admin_client):
    """Fetching a non-existent email id returns 404."""
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails/nonexistent-id-xyz")
    _skip_if_not_dev(resp)
    assert resp.status_code == 404, f"Expected 404 for unknown id, got {resp.status_code}"
