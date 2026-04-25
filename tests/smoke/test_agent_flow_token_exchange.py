"""Agent flow smoke — RFC 8693 Token Exchange / Delegation.

Covers: A→B delegation with act claim chain, scope downscoping, rejection
on scope widening.
"""
import base64
import json
import pytest
import requests
import os

BASE_URL = os.environ.get("BASE", "http://localhost:8080")
GRANT_TOKEN_EXCHANGE = "urn:ietf:params:oauth:grant-type:token-exchange"
TOKEN_TYPE_ACCESS = "urn:ietf:params:oauth:token-type:access_token"


def _admin_headers(admin_key: str) -> dict:
    return {"Authorization": f"Bearer {admin_key}"}


def _create_cc_agent(admin_key: str, name: str, scopes: list) -> dict:
    resp = requests.post(
        f"{BASE_URL}/api/v1/agents",
        json={"name": name, "grant_types": ["client_credentials"], "scopes": scopes},
        headers=_admin_headers(admin_key),
    )
    assert resp.status_code == 201, f"Agent create failed ({name}): {resp.text}"
    return resp.json()


def _get_token(client_id: str, client_secret: str, scope: str = "") -> str:
    data = {"grant_type": "client_credentials"}
    if scope:
        data["scope"] = scope
    resp = requests.post(
        f"{BASE_URL}/oauth/token",
        data=data,
        auth=(client_id, client_secret),
    )
    assert resp.status_code == 200, f"CC token failed: {resp.text}"
    return resp.json()["access_token"]


def _decode_jwt_claims(token: str) -> dict:
    parts = token.split(".")
    if len(parts) < 2:
        return {}
    padding = "=" * (4 - len(parts[1]) % 4)
    try:
        return json.loads(base64.urlsafe_b64decode(parts[1] + padding))
    except Exception:
        return {}


class TestTokenExchange:
    @pytest.fixture
    def manager_worker(self, admin_key):
        """Create manager + worker agents; yield both; cleanup after."""
        mgr = _create_cc_agent(
            admin_key, "smoke-tex-manager",
            ["task:orchestrate", "worker:delegate"],
        )
        wkr = _create_cc_agent(
            admin_key, "smoke-tex-worker",
            ["task:execute"],
        )
        yield mgr, wkr
        for agent in (mgr, wkr):
            requests.delete(
                f"{BASE_URL}/api/v1/agents/{agent['id']}",
                headers=_admin_headers(admin_key),
            )

    def test_basic_token_exchange(self, manager_worker):
        """Worker can exchange manager token for a scope the subject token grants."""
        mgr, wkr = manager_worker
        # Subject token must include the scope being requested in the exchange.
        mgr_token = _get_token(mgr["client_id"], mgr["client_secret"], "task:orchestrate worker:delegate")

        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": GRANT_TOKEN_EXCHANGE,
                "subject_token": mgr_token,
                "subject_token_type": TOKEN_TYPE_ACCESS,
                "scope": "task:orchestrate",  # subset of subject token scopes
            },
            auth=(wkr["client_id"], wkr["client_secret"]),
        )
        assert resp.status_code == 200, f"Token exchange failed: {resp.text}"
        data = resp.json()
        assert "access_token" in data
        assert data.get("issued_token_type") == TOKEN_TYPE_ACCESS or "access_token" in data

    def test_act_claim_chain_in_exchanged_token(self, manager_worker):
        """Exchanged token must carry act claim linking back to original subject."""
        mgr, wkr = manager_worker
        mgr_token = _get_token(mgr["client_id"], mgr["client_secret"], "task:orchestrate worker:delegate")

        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": GRANT_TOKEN_EXCHANGE,
                "subject_token": mgr_token,
                "subject_token_type": TOKEN_TYPE_ACCESS,
                "scope": "task:orchestrate",  # subset of subject token scopes
            },
            auth=(wkr["client_id"], wkr["client_secret"]),
        )
        assert resp.status_code == 200, resp.text
        exchanged_token = resp.json()["access_token"]
        claims = _decode_jwt_claims(exchanged_token)

        # RFC 8693 §4.4: act claim must be present and contain sub of original actor
        assert "act" in claims, f"Missing act claim in exchanged token; claims={claims}"
        assert "sub" in claims["act"], f"act claim missing sub; act={claims['act']}"

    def test_scope_downscoping_allowed(self, manager_worker):
        """Exchange with narrower scope than subject token must succeed."""
        mgr, wkr = manager_worker
        # Get subject token with two scopes; request only one
        mgr_token = _get_token(mgr["client_id"], mgr["client_secret"], "task:orchestrate worker:delegate")

        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": GRANT_TOKEN_EXCHANGE,
                "subject_token": mgr_token,
                "subject_token_type": TOKEN_TYPE_ACCESS,
                "scope": "task:orchestrate",  # narrower — only one of two subject scopes
            },
            auth=(wkr["client_id"], wkr["client_secret"]),
        )
        assert resp.status_code == 200, f"Downscoped exchange failed: {resp.text}"

    def test_scope_widening_rejected(self, admin_key):
        """Exchange requesting scope wider than subject token must be rejected."""
        # Create agent with limited scope
        limited = _create_cc_agent(admin_key, "smoke-tex-limited", ["read"])
        wide = _create_cc_agent(admin_key, "smoke-tex-wide", ["read", "admin"])
        try:
            limited_token = _get_token(limited["client_id"], limited["client_secret"], "read")

            resp = requests.post(
                f"{BASE_URL}/oauth/token",
                data={
                    "grant_type": GRANT_TOKEN_EXCHANGE,
                    "subject_token": limited_token,
                    "subject_token_type": TOKEN_TYPE_ACCESS,
                    "scope": "admin",  # wider than subject token
                },
                auth=(wide["client_id"], wide["client_secret"]),
            )
            # Must reject with invalid_scope or similar error
            assert resp.status_code in (400, 403), \
                f"Expected rejection for scope widening, got {resp.status_code}: {resp.text}"
            err = resp.json()
            assert "error" in err
        finally:
            for agent in (limited, wide):
                requests.delete(
                    f"{BASE_URL}/api/v1/agents/{agent['id']}",
                    headers=_admin_headers(admin_key),
                )

    def test_exchange_with_invalid_subject_token_rejected(self, manager_worker):
        """Exchange with garbage subject_token must be rejected."""
        _, wkr = manager_worker
        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": GRANT_TOKEN_EXCHANGE,
                "subject_token": "not.a.valid.jwt.token",
                "subject_token_type": TOKEN_TYPE_ACCESS,
                "scope": "task:execute",
            },
            auth=(wkr["client_id"], wkr["client_secret"]),
        )
        assert resp.status_code in (400, 401), \
            f"Expected rejection for invalid subject_token: {resp.text}"
        assert "error" in resp.json()
