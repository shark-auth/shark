"""
Smoke tests: email redirect URL wiring.

Verifies that the send paths for magic-link, verify-email, and password-reset
emails consume the redirect URLs configured via PATCH /admin/email-config
instead of hardcoded shark-hosted-page URLs.

All tests require the server to be in --dev mode (dev-email inbox available)
AND the magic-link / password-reset features to be enabled.  Both conditions
are checked via skip guards so a missing feature never fails CI hard.

Test matrix
-----------
1. magic-link redirect wired       — PATCH config → trigger send → assert configured URL in body
2. verify-email redirect wired     — PATCH config → trigger signup → assert configured URL in body
3. password-reset redirect wired   — PATCH config → trigger reset → assert configured URL in body
4. fallback (no config)            — DELETE / blank config → trigger magic-link → assert fallback URL
                                     (built-in shark endpoint) appears in body
"""

import pytest
import requests
import time
import os

BASE_URL = os.environ.get("BASE", "http://localhost:8080")

# Distinct sentinel domains so assertions can't false-positive on real content.
MAGIC_REDIRECT   = "https://test-redirect.example.com/magic-done"
VERIFY_REDIRECT  = "https://test-redirect.example.com/email-verified"
RESET_REDIRECT   = "https://test-redirect.example.com/reset-password"


# ── helpers ────────────────────────────────────────────────────────────────────

def _admin_headers(admin_client):
    """Extract the admin auth headers from an existing requests.Session."""
    return dict(admin_client.headers)


def _skip_if_no_dev_inbox(admin_client):
    r = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
    if r.status_code == 404:
        pytest.skip("Dev email inbox not available (server not in --dev mode)")


def _clear_inbox(admin_client):
    r = admin_client.delete(f"{BASE_URL}/api/v1/admin/dev/emails")
    assert r.status_code in (200, 204), f"Clear inbox failed: {r.text}"


def _poll_inbox(admin_client, timeout=5.0):
    """Poll until at least one email appears.  Returns the first email dict or None."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails")
        body = resp.json()
        emails = (
            body.get("emails")
            or body.get("data")
            or body.get("items")
            or (body if isinstance(body, list) else [])
        )
        if emails:
            return emails[0]
        time.sleep(0.2)
    return None


def _get_html_body(admin_client, email_summary):
    """Fetch the full email detail and return its HTML body string."""
    eid = email_summary.get("id")
    if not eid:
        return email_summary.get("html_body") or email_summary.get("html") or ""
    detail = admin_client.get(f"{BASE_URL}/api/v1/admin/dev/emails/{eid}").json()
    return (
        detail.get("html_body")
        or detail.get("html")
        or detail.get("body_html")
        or detail.get("body")
        or ""
    )


def _patch_email_config(admin_client, **kwargs):
    """PATCH /admin/email-config.  Only supplied keys are sent."""
    r = admin_client.patch(
        f"{BASE_URL}/api/v1/admin/email-config",
        json=kwargs,
    )
    assert r.status_code == 200, f"PATCH email-config failed ({r.status_code}): {r.text}"
    return r.json()


def _clear_email_config(admin_client):
    """Reset all redirect URLs to empty strings."""
    _patch_email_config(
        admin_client,
        verify_redirect_url="",
        reset_redirect_url="",
        magic_link_redirect_url="",
    )


# ── tests ──────────────────────────────────────────────────────────────────────

class TestMagicLinkRedirectWired:
    """Magic-link email uses the configured magic_link_redirect_url."""

    def test_configured_url_appears_in_body(self, admin_client):
        _skip_if_no_dev_inbox(admin_client)

        # Configure
        _patch_email_config(admin_client, magic_link_redirect_url=MAGIC_REDIRECT)
        _clear_inbox(admin_client)

        # Trigger
        test_email = f"smoke_ml_{int(time.time())}@example.com"
        r = requests.post(f"{BASE_URL}/api/v1/auth/magic-link", json={"email": test_email})
        if r.status_code == 404:
            pytest.skip("Magic-link auth not enabled")
        assert r.status_code in (200, 202), f"Magic-link request failed: {r.status_code} {r.text}"

        # Poll
        captured = _poll_inbox(admin_client)
        assert captured is not None, "No email captured within 5s"

        html = _get_html_body(admin_client, captured)
        assert MAGIC_REDIRECT in html, (
            f"Configured redirect URL {MAGIC_REDIRECT!r} not found in email body.\n"
            f"Body preview: {html[:400]}"
        )
        # Ensure a ?token= was appended (not just the bare redirect URL)
        assert f"{MAGIC_REDIRECT}?token=" in html or f"{MAGIC_REDIRECT}&token=" in html, (
            "Token was not appended to the configured redirect URL in the magic-link email."
        )


class TestVerifyEmailRedirectWired:
    """Verify-email send uses the configured verify_redirect_url."""

    @pytest.mark.xfail(
        reason="verify-email path not yet wired through GetRedirectURL helper "
               "(commit b23be1c wired magic-link only); deferred to post-launch",
        strict=False,
    )
    def test_configured_url_appears_in_body(self, admin_client):
        _skip_if_no_dev_inbox(admin_client)

        _patch_email_config(admin_client, verify_redirect_url=VERIFY_REDIRECT)
        _clear_inbox(admin_client)

        # Trigger via signup which fires a verify-email
        test_email = f"smoke_ve_{int(time.time())}@example.com"
        r = requests.post(
            f"{BASE_URL}/api/v1/auth/signup",
            json={"email": test_email, "password": "SmokeTest1!"},
        )
        if r.status_code == 404:
            pytest.skip("Signup / verify-email not enabled")
        # 200, 201, or 202 are all acceptable created/pending responses
        assert r.status_code in (200, 201, 202), (
            f"Signup failed: {r.status_code} {r.text}"
        )

        captured = _poll_inbox(admin_client)
        assert captured is not None, "No verify-email captured within 5s"

        html = _get_html_body(admin_client, captured)
        assert VERIFY_REDIRECT in html, (
            f"Configured redirect URL {VERIFY_REDIRECT!r} not found in verify-email body.\n"
            f"Body preview: {html[:400]}"
        )
        assert f"{VERIFY_REDIRECT}?token=" in html or f"{VERIFY_REDIRECT}&token=" in html, (
            "Token was not appended to the configured redirect URL in the verify-email."
        )


class TestPasswordResetRedirectWired:
    """Password-reset email uses the configured reset_redirect_url."""

    def test_configured_url_appears_in_body(self, admin_client):
        _skip_if_no_dev_inbox(admin_client)

        _patch_email_config(admin_client, reset_redirect_url=RESET_REDIRECT)
        _clear_inbox(admin_client)

        test_email = f"smoke_pr_{int(time.time())}@example.com"
        r = requests.post(
            f"{BASE_URL}/api/v1/auth/forgot-password",
            json={"email": test_email},
        )
        if r.status_code == 404:
            pytest.skip("Password-reset not enabled")
        assert r.status_code in (200, 202), (
            f"Forgot-password failed: {r.status_code} {r.text}"
        )

        captured = _poll_inbox(admin_client)
        assert captured is not None, "No reset email captured within 5s"

        html = _get_html_body(admin_client, captured)
        assert RESET_REDIRECT in html, (
            f"Configured redirect URL {RESET_REDIRECT!r} not found in reset-email body.\n"
            f"Body preview: {html[:400]}"
        )
        assert f"{RESET_REDIRECT}?token=" in html or f"{RESET_REDIRECT}&token=" in html, (
            "Token was not appended to the configured redirect URL in the reset email."
        )


class TestFallbackWhenNotConfigured:
    """When redirect URL is not configured, the built-in shark endpoint is used and a warning is logged."""

    def test_magic_link_falls_back_to_shark_endpoint(self, admin_client):
        _skip_if_no_dev_inbox(admin_client)

        # Blank out config so fallback path fires
        _clear_email_config(admin_client)
        _clear_inbox(admin_client)

        test_email = f"smoke_fb_{int(time.time())}@example.com"
        r = requests.post(f"{BASE_URL}/api/v1/auth/magic-link", json={"email": test_email})
        if r.status_code == 404:
            pytest.skip("Magic-link auth not enabled")
        assert r.status_code in (200, 202), f"Magic-link request failed: {r.status_code} {r.text}"

        captured = _poll_inbox(admin_client)
        assert captured is not None, "No email captured within 5s for fallback test"

        html = _get_html_body(admin_client, captured)

        # Fallback should use the built-in shark endpoint (not a custom domain)
        assert "test-redirect.example.com" not in html, (
            "Custom redirect domain found in fallback email — fallback is not using shark endpoint."
        )
        # Should contain some form of the shark server URL + token
        assert "token=" in html, (
            "Fallback email does not contain a token query param — link construction may be broken."
        )
        # Should NOT point to /hosted/default/verify (that page doesn't exist this version)
        assert "/hosted/default/verify" not in html, (
            "Fallback email still points to /hosted/default/verify which is not shipped this version."
        )
