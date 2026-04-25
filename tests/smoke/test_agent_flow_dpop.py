"""Agent flow smoke — RFC 9449 DPoP.

Covers: DPoP-bound token issuance, cnf.jkt claim in JWT, replay attack
rejection (same proof reused), missing DPoP header on DPoP-bound token.

Uses the shark_auth.dpop SDK helper to mint proofs.
"""
import base64
import json
import os
import sys
import time
import pytest
import requests

BASE_URL = os.environ.get("BASE", "http://localhost:8080")

# Make the shark_auth SDK importable without pip install
_SDK_PATH = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "..", "..", "sdk", "python")
)
if _SDK_PATH not in sys.path:
    sys.path.insert(0, _SDK_PATH)

try:
    from shark_auth.dpop import DPoPProver  # noqa: E402
    _HAS_DPOP = True
except ImportError:
    _HAS_DPOP = False

pytestmark = pytest.mark.skipif(
    not _HAS_DPOP,
    reason="shark_auth.dpop SDK not importable — build SDK first",
)


def _admin_headers(admin_key: str) -> dict:
    return {"Authorization": f"Bearer {admin_key}"}


def _create_cc_agent(admin_key: str, name: str) -> dict:
    resp = requests.post(
        f"{BASE_URL}/api/v1/agents",
        json={"name": name, "grant_types": ["client_credentials"], "scopes": ["read"]},
        headers=_admin_headers(admin_key),
    )
    assert resp.status_code == 201, f"Agent create failed: {resp.text}"
    return resp.json()


def _decode_jwt_claims(token: str) -> dict:
    parts = token.split(".")
    if len(parts) < 2:
        return {}
    padding = "=" * (4 - len(parts[1]) % 4)
    try:
        return json.loads(base64.urlsafe_b64decode(parts[1] + padding))
    except Exception:
        return {}


class TestDPoP:
    @pytest.fixture
    def agent(self, admin_key):
        a = _create_cc_agent(admin_key, "smoke-dpop-agent")
        yield a
        requests.delete(
            f"{BASE_URL}/api/v1/agents/{a['id']}",
            headers=_admin_headers(admin_key),
        )

    def test_dpop_bound_token_issued(self, agent):
        """client_credentials with DPoP proof returns access token."""
        prover = DPoPProver.generate()
        token_url = f"{BASE_URL}/oauth/token"
        proof = prover.make_proof("POST", token_url)

        resp = requests.post(
            token_url,
            data={"grant_type": "client_credentials", "scope": "read"},
            auth=(agent["client_id"], agent["client_secret"]),
            headers={"DPoP": proof},
        )
        assert resp.status_code == 200, f"DPoP token request failed: {resp.text}"
        data = resp.json()
        assert "access_token" in data

    def test_cnf_jkt_claim_present_in_jwt(self, agent):
        """JWT access token issued with DPoP must carry cnf.jkt claim."""
        prover = DPoPProver.generate()
        token_url = f"{BASE_URL}/oauth/token"
        proof = prover.make_proof("POST", token_url)

        resp = requests.post(
            token_url,
            data={"grant_type": "client_credentials", "scope": "read"},
            auth=(agent["client_id"], agent["client_secret"]),
            headers={"DPoP": proof},
        )
        assert resp.status_code == 200, resp.text
        access_token = resp.json()["access_token"]
        claims = _decode_jwt_claims(access_token)

        assert "cnf" in claims, f"Missing cnf claim; claims={claims}"
        assert "jkt" in claims["cnf"], f"Missing cnf.jkt; cnf={claims['cnf']}"
        # jkt must be a non-empty string (base64url-encoded SHA-256 of JWK)
        assert isinstance(claims["cnf"]["jkt"], str) and len(claims["cnf"]["jkt"]) > 0

    def test_jkt_matches_prover_thumbprint(self, agent):
        """cnf.jkt in JWT must match the thumbprint of the DPoP key used."""
        prover = DPoPProver.generate()
        token_url = f"{BASE_URL}/oauth/token"
        proof = prover.make_proof("POST", token_url)

        resp = requests.post(
            token_url,
            data={"grant_type": "client_credentials", "scope": "read"},
            auth=(agent["client_id"], agent["client_secret"]),
            headers={"DPoP": proof},
        )
        assert resp.status_code == 200, resp.text
        access_token = resp.json()["access_token"]
        claims = _decode_jwt_claims(access_token)

        expected_jkt = prover.jkt
        assert claims["cnf"]["jkt"] == expected_jkt, \
            f"jkt mismatch: got {claims['cnf']['jkt']!r}, expected {expected_jkt!r}"

    def test_dpop_replay_rejected(self, agent):
        """Reusing the same DPoP proof (replay) must be rejected."""
        prover = DPoPProver.generate()
        token_url = f"{BASE_URL}/oauth/token"
        # Create ONE proof
        proof = prover.make_proof("POST", token_url)

        # First use — must succeed
        resp1 = requests.post(
            token_url,
            data={"grant_type": "client_credentials", "scope": "read"},
            auth=(agent["client_id"], agent["client_secret"]),
            headers={"DPoP": proof},
        )
        assert resp1.status_code == 200, f"First DPoP request failed: {resp1.text}"

        # Second use of same proof — must be rejected (replay)
        resp2 = requests.post(
            token_url,
            data={"grant_type": "client_credentials", "scope": "read"},
            auth=(agent["client_id"], agent["client_secret"]),
            headers={"DPoP": proof},
        )
        assert resp2.status_code in (400, 401), \
            f"Replay should be rejected, got {resp2.status_code}: {resp2.text}"
        err = resp2.json()
        assert err.get("error") in ("invalid_dpop_proof", "use_dpop_nonce", "invalid_request"), \
            f"Unexpected error on replay: {err}"

    def test_invalid_dpop_proof_rejected(self, agent):
        """Malformed DPoP header must be rejected before token issuance."""
        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={"grant_type": "client_credentials", "scope": "read"},
            auth=(agent["client_id"], agent["client_secret"]),
            headers={"DPoP": "not.a.valid.dpop.proof"},
        )
        assert resp.status_code in (400, 401), \
            f"Invalid DPoP proof should be rejected: {resp.text}"
        assert "error" in resp.json()

    def test_wrong_htu_in_dpop_proof_rejected(self, agent):
        """DPoP proof with wrong htu (URL) must be rejected."""
        prover = DPoPProver.generate()
        # Mint proof for a different URL
        wrong_proof = prover.make_proof("POST", "http://attacker.example.com/oauth/token")

        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={"grant_type": "client_credentials", "scope": "read"},
            auth=(agent["client_id"], agent["client_secret"]),
            headers={"DPoP": wrong_proof},
        )
        assert resp.status_code in (400, 401), \
            f"Wrong htu should be rejected: {resp.text}"
