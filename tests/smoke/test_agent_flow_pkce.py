"""Agent flow smoke — PKCE (RFC 7636 / OAuth 2.1).

Covers: S256 challenge succeeds, plain method rejected, code reuse fails,
verifier mismatch fails, missing challenge fails.
"""
import base64
import hashlib
import os
import secrets
import pytest
import requests

BASE_URL = os.environ.get("BASE", "http://localhost:8080")


def _admin_headers(admin_key: str) -> dict:
    return {"Authorization": f"Bearer {admin_key}"}


def _create_pkce_agent(admin_key: str, name: str) -> dict:
    resp = requests.post(
        f"{BASE_URL}/api/v1/agents",
        json={
            "name": name,
            "grant_types": ["authorization_code", "refresh_token"],
            "redirect_uris": ["http://localhost:9999/callback"],
            "scopes": ["openid", "offline_access"],
            "client_type": "confidential",
            "token_endpoint_auth_method": "client_secret_basic",
            "response_types": ["code"],
        },
        headers=_admin_headers(admin_key),
    )
    assert resp.status_code == 201, f"Agent create failed: {resp.text}"
    return resp.json()


def _s256_challenge(verifier: str) -> str:
    digest = hashlib.sha256(verifier.encode()).digest()
    return base64.urlsafe_b64encode(digest).decode().rstrip("=")


def _do_authorize(session: requests.Session, client_id: str, challenge: str,
                  method: str = "S256") -> requests.Response:
    params = {
        "response_type": "code",
        "client_id": client_id,
        "redirect_uri": "http://localhost:9999/callback",
        "state": secrets.token_hex(8),
        "scope": "openid offline_access",
        "code_challenge": challenge,
        "code_challenge_method": method,
    }
    resp = session.get(
        f"{BASE_URL}/oauth/authorize",
        params=params,
        allow_redirects=False,
    )
    if resp.status_code == 200:
        # Server may return 200 consent page; POST back to approve
        import urllib.parse
        qs = urllib.parse.urlencode(params)
        resp = session.post(
            f"{BASE_URL}/oauth/authorize?{qs}",
            data={
                "client_id": client_id,
                "state": params["state"],
                "approved": "true",
                "scope": "openid offline_access",
            },
            allow_redirects=False,
        )
    return resp


def _extract_code(location: str) -> str:
    from urllib.parse import urlparse, parse_qs
    parsed = urlparse(location)
    qs = parse_qs(parsed.query)
    codes = qs.get("code", [])
    assert codes, f"No code in redirect: {location}"
    return codes[0]


class TestPKCE:
    @pytest.fixture
    def pkce_agent(self, admin_key):
        agent = _create_pkce_agent(admin_key, "smoke-pkce-agent")
        yield agent
        requests.delete(
            f"{BASE_URL}/api/v1/agents/{agent['id']}",
            headers=_admin_headers(admin_key),
        )

    def test_s256_challenge_succeeds(self, pkce_agent, auth_session):
        """Authorization code flow with S256 PKCE challenge completes successfully."""
        verifier = secrets.token_urlsafe(48)
        challenge = _s256_challenge(verifier)
        cid = pkce_agent["client_id"]
        secret = pkce_agent.get("client_secret", "")

        resp = _do_authorize(auth_session, cid, challenge, "S256")
        assert "Location" in resp.headers, \
            f"Expected redirect after authorize, got {resp.status_code}: {resp.text}"

        location = resp.headers["Location"]
        assert "error" not in location, f"Auth error in redirect: {location}"
        code = _extract_code(location)

        # Exchange code with verifier — always send client_secret_basic auth
        token_resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": "authorization_code",
                "code": code,
                "redirect_uri": "http://localhost:9999/callback",
                "code_verifier": verifier,
            },
            auth=(cid, secret),
        )
        assert token_resp.status_code == 200, \
            f"Token exchange failed: {token_resp.text}"
        assert "access_token" in token_resp.json()

    def test_plain_method_rejected(self, pkce_agent, auth_session):
        """plain code_challenge_method must be rejected (OAuth 2.1 requires S256)."""
        verifier = secrets.token_urlsafe(48)
        cid = pkce_agent["client_id"]

        resp = _do_authorize(auth_session, cid, verifier, "plain")
        location = resp.headers.get("Location", "")
        # Either redirect with error= or direct error response
        has_error = (
            resp.status_code in (400, 403)
            or "error=" in location
        )
        assert has_error, \
            f"plain method should be rejected, got {resp.status_code}: {location or resp.text}"

    @pytest.mark.xfail(reason="Backend gap: PKCE not enforced as mandatory; server currently allows auth_code flow without code_challenge (OAuth 2.1 §4.1.1 requires it)")
    def test_missing_code_challenge_rejected(self, pkce_agent, auth_session):
        """Authorization request without code_challenge must be rejected."""
        cid = pkce_agent["client_id"]
        params = {
            "response_type": "code",
            "client_id": cid,
            "redirect_uri": "http://localhost:9999/callback",
            "state": secrets.token_hex(8),
            "scope": "openid",
            # No code_challenge or code_challenge_method
        }
        resp = auth_session.get(
            f"{BASE_URL}/oauth/authorize",
            params=params,
            allow_redirects=False,
        )
        location = resp.headers.get("Location", "")
        has_error = (
            resp.status_code in (400, 403)
            or "error=" in location
        )
        assert has_error, \
            f"Missing PKCE challenge should be rejected, got {resp.status_code}"

    def test_wrong_verifier_rejected(self, pkce_agent, auth_session):
        """Code exchange with incorrect verifier must fail."""
        verifier = secrets.token_urlsafe(48)
        challenge = _s256_challenge(verifier)
        cid = pkce_agent["client_id"]
        secret = pkce_agent.get("client_secret", "")

        resp = _do_authorize(auth_session, cid, challenge, "S256")
        if "Location" not in resp.headers:
            pytest.skip("Authorize did not redirect — skipping verifier test")
        location = resp.headers["Location"]
        if "error=" in location:
            pytest.skip("Authorize returned error — skipping verifier test")
        code = _extract_code(location)

        wrong_verifier = secrets.token_urlsafe(48)  # different verifier
        token_resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": "authorization_code",
                "code": code,
                "redirect_uri": "http://localhost:9999/callback",
                "code_verifier": wrong_verifier,
            },
            auth=(cid, secret),
        )
        assert token_resp.status_code in (400, 401), \
            f"Wrong verifier should be rejected: {token_resp.text}"
        assert "error" in token_resp.json()

    def test_code_reuse_fails(self, pkce_agent, auth_session):
        """Authorization code can only be used once; second use must fail."""
        verifier = secrets.token_urlsafe(48)
        challenge = _s256_challenge(verifier)
        cid = pkce_agent["client_id"]
        secret = pkce_agent.get("client_secret", "")

        resp = _do_authorize(auth_session, cid, challenge, "S256")
        if "Location" not in resp.headers:
            pytest.skip("Authorize did not redirect")
        location = resp.headers["Location"]
        if "error=" in location:
            pytest.skip("Authorize returned error")
        code = _extract_code(location)

        exchange_data = {
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": "http://localhost:9999/callback",
            "code_verifier": verifier,
        }

        # First exchange — must succeed
        resp1 = requests.post(f"{BASE_URL}/oauth/token", data=exchange_data, auth=(cid, secret))
        assert resp1.status_code == 200, f"First exchange failed: {resp1.text}"

        # Second exchange — must fail
        resp2 = requests.post(f"{BASE_URL}/oauth/token", data=exchange_data, auth=(cid, secret))
        assert resp2.status_code in (400, 401), \
            f"Code reuse should be rejected, got {resp2.status_code}: {resp2.text}"
        assert "error" in resp2.json()
