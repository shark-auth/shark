"""Agent flow smoke — Refresh token rotation + reuse detection.

Covers: RT rotation on use, family revoke on reuse (RT theft detection),
expired RT rejected.
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


def _s256_challenge(verifier: str) -> str:
    digest = hashlib.sha256(verifier.encode()).digest()
    return base64.urlsafe_b64encode(digest).decode().rstrip("=")


def _create_rt_agent(admin_key: str, name: str) -> dict:
    resp = requests.post(
        f"{BASE_URL}/api/v1/agents",
        json={
            "name": name,
            "grant_types": ["authorization_code", "refresh_token"],
            "redirect_uris": ["http://localhost:9999/callback"],
            "scopes": ["openid", "offline_access"],
            "client_type": "confidential",
            "response_types": ["code"],
        },
        headers=_admin_headers(admin_key),
    )
    assert resp.status_code == 201, f"Agent create failed: {resp.text}"
    return resp.json()


def _get_auth_code_and_tokens(session: requests.Session, agent: dict) -> dict:
    """Complete auth code + PKCE flow; return token response dict."""
    cid = agent["client_id"]
    secret = agent.get("client_secret", "")
    verifier = secrets.token_urlsafe(48)
    challenge = _s256_challenge(verifier)

    params = {
        "response_type": "code",
        "client_id": cid,
        "redirect_uri": "http://localhost:9999/callback",
        "state": secrets.token_hex(8),
        "scope": "openid offline_access",
        "code_challenge": challenge,
        "code_challenge_method": "S256",
    }

    resp = session.get(
        f"{BASE_URL}/oauth/authorize",
        params=params,
        allow_redirects=False,
    )
    if resp.status_code == 200:
        import urllib.parse
        qs = urllib.parse.urlencode(params)
        resp = session.post(
            f"{BASE_URL}/oauth/authorize?{qs}",
            data={
                "client_id": cid,
                "state": params["state"],
                "approved": "true",
                "scope": "openid offline_access",
            },
            allow_redirects=False,
        )

    assert "Location" in resp.headers, \
        f"Authorize did not redirect: {resp.status_code}: {resp.text}"
    location = resp.headers["Location"]
    assert "error=" not in location, f"Auth error: {location}"

    from urllib.parse import urlparse, parse_qs
    code = parse_qs(urlparse(location).query)["code"][0]

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
    assert token_resp.status_code == 200, f"Token exchange failed: {token_resp.text}"
    data = token_resp.json()
    assert "refresh_token" in data, "No refresh_token in response"
    return data


class TestRefreshTokenRotation:
    @pytest.fixture
    def rt_agent(self, admin_key):
        agent = _create_rt_agent(admin_key, "smoke-rt-rotation")
        yield agent
        requests.delete(
            f"{BASE_URL}/api/v1/agents/{agent['id']}",
            headers=_admin_headers(admin_key),
        )

    def test_refresh_token_rotation(self, rt_agent, admin_key, auth_session):
        """Refresh token use issues a new RT and invalidates the old one."""
        cid = rt_agent["client_id"]
        secret = rt_agent.get("client_secret", "")

        tokens = _get_auth_code_and_tokens(auth_session, rt_agent)
        rt1 = tokens["refresh_token"]

        # Use RT1 → get new tokens including RT2
        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": "refresh_token",
                "refresh_token": rt1,
            },
            auth=(cid, secret),
        )
        assert resp.status_code == 200, f"First refresh failed: {resp.text}"
        rotated = resp.json()
        assert "refresh_token" in rotated, "No RT in rotated response"
        rt2 = rotated["refresh_token"]

        # RT2 must be usable (verify before invalidating the family)
        resp_rt2 = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": "refresh_token",
                "refresh_token": rt2,
            },
            auth=(cid, secret),
        )
        assert resp_rt2.status_code == 200, f"RT2 use failed: {resp_rt2.text}"

        # RT1 must now be invalid (superseded by RT2 which was already used above)
        resp_reuse = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": "refresh_token",
                "refresh_token": rt1,
            },
            auth=(cid, secret),
        )
        assert resp_reuse.status_code in (400, 401), \
            f"Old RT1 should be revoked after rotation: {resp_reuse.text}"

    def test_refresh_token_reuse_detection_revokes_family(self, rt_agent, admin_key, auth_session):
        """Reusing a superseded RT triggers family revoke (breach detection)."""
        cid = rt_agent["client_id"]
        secret = rt_agent.get("client_secret", "")

        tokens = _get_auth_code_and_tokens(auth_session, rt_agent)
        rt1 = tokens["refresh_token"]

        # Rotate once: RT1 → RT2
        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={"grant_type": "refresh_token", "refresh_token": rt1},
            auth=(cid, secret),
        )
        assert resp.status_code == 200, f"Rotation failed: {resp.text}"
        rt2 = resp.json()["refresh_token"]

        # Reuse RT1 (attacker using stolen RT) — must be rejected
        resp_stolen = requests.post(
            f"{BASE_URL}/oauth/token",
            data={"grant_type": "refresh_token", "refresh_token": rt1},
            auth=(cid, secret),
        )
        assert resp_stolen.status_code in (400, 401), \
            f"Reuse of old RT should be rejected: {resp_stolen.text}"

        # RT2 should also now be revoked (family revoke)
        resp_rt2 = requests.post(
            f"{BASE_URL}/oauth/token",
            data={"grant_type": "refresh_token", "refresh_token": rt2},
            auth=(cid, secret),
        )
        assert resp_rt2.status_code in (400, 401), \
            f"RT2 should be revoked after family revoke, got {resp_rt2.status_code}: {resp_rt2.text}"

    def test_expired_refresh_token_rejected(self, rt_agent, admin_key):
        """Completely made-up / expired refresh token must return error."""
        cid = rt_agent["client_id"]
        secret = rt_agent.get("client_secret", "")

        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": "refresh_token",
                "refresh_token": "totally.fake.refresh.token.xyz123",
            },
            auth=(cid, secret),
        )
        assert resp.status_code in (400, 401), \
            f"Fake RT should be rejected: {resp.text}"
        err = resp.json()
        assert "error" in err

    def test_new_access_token_issued_on_refresh(self, rt_agent, admin_key, auth_session):
        """Refresh must issue a new access_token."""
        cid = rt_agent["client_id"]
        secret = rt_agent.get("client_secret", "")

        tokens = _get_auth_code_and_tokens(auth_session, rt_agent)
        original_at = tokens["access_token"]
        rt1 = tokens["refresh_token"]

        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={"grant_type": "refresh_token", "refresh_token": rt1},
            auth=(cid, secret),
        )
        assert resp.status_code == 200, resp.text
        refreshed = resp.json()
        assert "access_token" in refreshed
        # New access token must differ from the original
        assert refreshed["access_token"] != original_at, \
            "Refreshed access_token should be a new token"
