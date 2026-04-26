"""Unit tests for DPoPHTTPClient (Method 3 — W2 SDK).

Uses requests_mock to intercept all HTTP calls so no live server is needed.
"""

from __future__ import annotations

import base64
import hashlib
import json

import pytest
import requests_mock as req_mock

from shark_auth.dpop import DPoPProver
from shark_auth.http_client import DPoPHTTPClient

BASE = "https://auth.example.com"
TOKEN = "eyJmYWtlIjoidG9rZW4ifQ.test.sig"


def _ath(token: str) -> str:
    digest = hashlib.sha256(token.encode()).digest()
    return base64.urlsafe_b64encode(digest).rstrip(b"=").decode()


def _decode_proof_payload(proof: str) -> dict:
    """Decode the JWT payload (no signature verification needed for unit tests)."""
    parts = proof.split(".")
    assert len(parts) == 3, "DPoP proof must be a 3-part JWT"
    padded = parts[1] + "=" * (-len(parts[1]) % 4)
    return json.loads(base64.urlsafe_b64decode(padded))


def _decode_proof_header(proof: str) -> dict:
    parts = proof.split(".")
    padded = parts[0] + "=" * (-len(parts[0]) % 4)
    return json.loads(base64.urlsafe_b64decode(padded))


@pytest.fixture()
def prover() -> DPoPProver:
    return DPoPProver.generate()


@pytest.fixture()
def client() -> DPoPHTTPClient:
    return DPoPHTTPClient(BASE)


# ---------------------------------------------------------------------------
# Case 1: GET — verify Authorization + DPoP headers and proof claims
# ---------------------------------------------------------------------------

def test_get_with_dpop_headers(client: DPoPHTTPClient, prover: DPoPProver) -> None:
    with req_mock.Mocker() as m:
        m.get(f"{BASE}/api/v1/auth/me", json={"sub": "agent-1"}, status_code=200)

        resp = client.get_with_dpop("/api/v1/auth/me", token=TOKEN, prover=prover)

    assert resp.status_code == 200
    sent_req = m.last_request
    assert sent_req.headers["Authorization"] == f"DPoP {TOKEN}"

    proof = sent_req.headers["DPoP"]
    # Must be a JWT (3 parts)
    assert len(proof.split(".")) == 3

    payload = _decode_proof_payload(proof)
    assert payload["htm"] == "GET"
    assert payload["htu"] == f"{BASE}/api/v1/auth/me"
    assert payload["ath"] == _ath(TOKEN)


# ---------------------------------------------------------------------------
# Case 2: POST with json — same headers + body forwarded
# ---------------------------------------------------------------------------

def test_post_with_dpop_json(client: DPoPHTTPClient, prover: DPoPProver) -> None:
    body = {"action": "ping"}
    with req_mock.Mocker() as m:
        m.post(f"{BASE}/api/v1/actions", json={"ok": True}, status_code=201)

        resp = client.post_with_dpop("/api/v1/actions", token=TOKEN, prover=prover, json=body)

    assert resp.status_code == 201
    sent_req = m.last_request
    assert sent_req.headers["Authorization"] == f"DPoP {TOKEN}"
    assert "DPoP" in sent_req.headers

    payload = _decode_proof_payload(sent_req.headers["DPoP"])
    assert payload["htm"] == "POST"
    assert payload["htu"] == f"{BASE}/api/v1/actions"
    assert payload["ath"] == _ath(TOKEN)

    # Body must have been sent
    assert json.loads(sent_req.body) == body


# ---------------------------------------------------------------------------
# Case 3: DELETE — same header assertions
# ---------------------------------------------------------------------------

def test_delete_with_dpop_headers(client: DPoPHTTPClient, prover: DPoPProver) -> None:
    with req_mock.Mocker() as m:
        m.delete(f"{BASE}/api/v1/agents/x", status_code=204)

        resp = client.delete_with_dpop("/api/v1/agents/x", token=TOKEN, prover=prover)

    assert resp.status_code == 204
    sent_req = m.last_request
    assert sent_req.headers["Authorization"] == f"DPoP {TOKEN}"

    payload = _decode_proof_payload(sent_req.headers["DPoP"])
    assert payload["htm"] == "DELETE"
    assert payload["ath"] == _ath(TOKEN)


# ---------------------------------------------------------------------------
# Case 4: Each call generates a fresh proof with a different jti
# ---------------------------------------------------------------------------

def test_each_call_unique_jti(client: DPoPHTTPClient, prover: DPoPProver) -> None:
    with req_mock.Mocker() as m:
        m.get(f"{BASE}/api/v1/auth/me", json={}, status_code=200)
        m.post(f"{BASE}/api/v1/actions", json={}, status_code=200)
        m.delete(f"{BASE}/api/v1/agents/x", status_code=204)

        client.get_with_dpop("/api/v1/auth/me", token=TOKEN, prover=prover)
        req1 = m.request_history[0]

        client.post_with_dpop("/api/v1/actions", token=TOKEN, prover=prover)
        req2 = m.request_history[1]

        client.delete_with_dpop("/api/v1/agents/x", token=TOKEN, prover=prover)
        req3 = m.request_history[2]

    jti1 = _decode_proof_payload(req1.headers["DPoP"])["jti"]
    jti2 = _decode_proof_payload(req2.headers["DPoP"])["jti"]
    jti3 = _decode_proof_payload(req3.headers["DPoP"])["jti"]

    assert len({jti1, jti2, jti3}) == 3, "Each proof must have a unique jti"


# ---------------------------------------------------------------------------
# Case 5: Mock 401 — SDK returns it untouched (no exception raised)
# ---------------------------------------------------------------------------

def test_401_returned_untouched(client: DPoPHTTPClient, prover: DPoPProver) -> None:
    with req_mock.Mocker() as m:
        m.get(f"{BASE}/api/v1/auth/me", json={"error": "unauthorized"}, status_code=401)

        resp = client.get_with_dpop("/api/v1/auth/me", token=TOKEN, prover=prover)

    # SDK must NOT raise — caller decides how to handle HTTP errors.
    assert resp.status_code == 401
    assert resp.json()["error"] == "unauthorized"
