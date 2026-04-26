"""
W1.7 Edit 1 — Dev-email banner state smoke tests.

Verifies the contract the frontend banner relies on:
  - GET /api/v1/admin/dev/emails returns the expected envelope shape
    (emails list with id, to/to_addr/recipient, subject, timestamp field)
  - The endpoint requires authentication (returns 401 without credentials)

DO NOT RUN WITH PYTEST DIRECTLY — orchestrator-only (see MEMORY.md).
"""

import os
import pytest
import requests

BASE_URL = os.environ.get("BASE", "http://localhost:8080")
INBOX_URL = f"{BASE_URL}/api/v1/admin/dev/emails"


def _skip_if_not_dev(resp):
    if resp.status_code == 404:
        pytest.skip("Dev email endpoints not available (server not in dev mode)")


# ---------------------------------------------------------------------------
# Happy path: GET /api/v1/admin/dev/emails — envelope shape
# ---------------------------------------------------------------------------

def test_dev_inbox_envelope_shape(admin_client):
    """
    GET /api/v1/admin/dev/emails returns 200 with a recognised list envelope.

    Banner state contract: the frontend derives lastDelivered by sorting
    allEmails on created_at | received_at | timestamp. Each item must carry
    at least one of those timestamp keys and a to/to_addr/recipient field
    so the banner can display the recipient.
    """
    resp = admin_client.get(INBOX_URL)
    _skip_if_not_dev(resp)
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"

    body = resp.json()
    emails = (
        body.get("emails")
        or body.get("data")
        or body.get("items")
        or (body if isinstance(body, list) else None)
    )
    assert emails is not None, f"Response missing recognised list key: {body}"
    assert isinstance(emails, list), f"Email list should be a list, got {type(emails)}"

    # If there are any emails, verify each has the fields the banner depends on
    for item in emails:
        assert isinstance(item, dict), f"Email item should be a dict: {item!r}"

        # Must have an id
        assert item.get("id"), f"Email item missing 'id': {item}"

        # Must have at least one recipient field
        recipient = item.get("to") or item.get("to_addr") or item.get("recipient")
        assert recipient, f"Email item missing recipient field (to/to_addr/recipient): {item}"

        # Must have at least one timestamp field for banner sort
        ts = item.get("created_at") or item.get("received_at") or item.get("timestamp")
        assert ts, f"Email item missing timestamp field (created_at/received_at/timestamp): {item}"


# ---------------------------------------------------------------------------
# Negative: unauthenticated request must return 401
# ---------------------------------------------------------------------------

def test_dev_inbox_requires_auth():
    """
    GET /api/v1/admin/dev/emails without credentials returns 401.

    The banner reads live backend data; this confirms the endpoint is
    properly guarded so unauthenticated callers cannot read captured mail.
    """
    resp = requests.get(INBOX_URL)
    # 404 means dev mode off — skip rather than fail
    if resp.status_code == 404:
        pytest.skip("Dev email endpoints not available (server not in dev mode)")
    assert resp.status_code == 401, (
        f"Expected 401 for unauthenticated request, got {resp.status_code}: {resp.text}"
    )
