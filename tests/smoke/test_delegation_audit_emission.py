"""Smoke — delegation audit emission (W19 fix).

Verifies that after a successful RFC 8693 token exchange:
  - GET /api/v1/audit-logs?action=oauth.token.exchanged returns >= 1 event
  - The event has actor_type == "agent"
  - The event metadata is non-empty JSON containing an "act_chain" key
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


def _create_agent(admin_key: str, name: str, scopes: list) -> dict:
    resp = requests.post(
        f"{BASE_URL}/api/v1/agents",
        json={"name": name, "grant_types": ["client_credentials"], "scopes": scopes},
        headers=_admin_headers(admin_key),
    )
    assert resp.status_code == 201, f"Agent create failed ({name}): {resp.text}"
    return resp.json()


def _delete_agent(admin_key: str, agent_id: str) -> None:
    requests.delete(
        f"{BASE_URL}/api/v1/agents/{agent_id}",
        headers=_admin_headers(admin_key),
    )


def _get_cc_token(client_id: str, client_secret: str, scope: str = "") -> str:
    data: dict = {"grant_type": "client_credentials"}
    if scope:
        data["scope"] = scope
    resp = requests.post(
        f"{BASE_URL}/oauth/token",
        data=data,
        auth=(client_id, client_secret),
    )
    assert resp.status_code == 200, f"CC token failed: {resp.text}"
    return resp.json()["access_token"]


def _token_exchange(actor_id: str, actor_secret: str, subject_token: str, scope: str = "") -> requests.Response:
    data: dict = {
        "grant_type": GRANT_TOKEN_EXCHANGE,
        "subject_token": subject_token,
        "subject_token_type": TOKEN_TYPE_ACCESS,
    }
    if scope:
        data["scope"] = scope
    return requests.post(
        f"{BASE_URL}/oauth/token",
        data=data,
        auth=(actor_id, actor_secret),
    )


class TestDelegationAuditEmission:
    @pytest.fixture
    def two_agents(self, admin_key):
        """Create subject (parent) + actor (child) agents; cleanup after."""
        parent = _create_agent(admin_key, "smoke-deleg-parent", ["read", "write"])
        actor = _create_agent(admin_key, "smoke-deleg-actor", ["read"])

        # configure may_act on parent to permit the actor
        requests.put(
            f"{BASE_URL}/api/v1/agents/{parent['id']}/policy",
            json={"may_act": [{"client_id": actor["client_id"]}]},
            headers=_admin_headers(admin_key),
        )

        yield parent, actor

        _delete_agent(admin_key, parent["id"])
        _delete_agent(admin_key, actor["id"])

    def test_token_exchange_emits_audit_event(self, admin_key, two_agents):
        """After a successful token exchange, audit-logs must contain the event."""
        parent, actor = two_agents

        # 1. Obtain a subject token for the parent agent.
        subject_token = _get_cc_token(parent["client_id"], parent["client_secret"], scope="read")

        # 2. Actor performs token exchange acting on behalf of parent.
        ex_resp = _token_exchange(
            actor["client_id"], actor["client_secret"],
            subject_token, scope="read",
        )
        assert ex_resp.status_code == 200, f"Token exchange failed: {ex_resp.text}"
        exchanged = ex_resp.json()
        assert "access_token" in exchanged, "No access_token in exchange response"

        # 3. Query audit logs for oauth.token.exchanged events.
        audit_resp = requests.get(
            f"{BASE_URL}/api/v1/audit-logs",
            params={"action": "oauth.token.exchanged", "limit": 50},
            headers=_admin_headers(admin_key),
        )
        assert audit_resp.status_code == 200, f"Audit log query failed: {audit_resp.text}"
        audit_data = audit_resp.json()

        events = audit_data.get("data") or audit_data.get("items") or []
        assert len(events) >= 1, (
            "Expected at least 1 audit event with action=oauth.token.exchanged, got 0. "
            "Backend is likely not calling AuditLogger.Log in HandleTokenExchange."
        )

        # 4. Find the event for this specific actor agent.
        matching = [
            ev for ev in events
            if ev.get("actor_id") == actor["id"] or ev.get("actor_id") == actor["client_id"]
        ]
        assert len(matching) >= 1, (
            f"No audit event found for actor_id={actor['id']}. "
            f"Events present: {[e.get('actor_id') for e in events]}"
        )

        ev = matching[0]

        # 5. actor_type must be "agent".
        assert ev.get("actor_type") == "agent", (
            f"Expected actor_type='agent', got '{ev.get('actor_type')}'"
        )

        # 6. metadata must be non-empty JSON containing act_chain.
        raw_meta = ev.get("metadata", "")
        assert raw_meta and raw_meta != "{}", (
            f"Event metadata is empty or bare {{}}: {raw_meta!r}"
        )

        try:
            meta = json.loads(raw_meta)
        except json.JSONDecodeError as exc:
            pytest.fail(f"Event metadata is not valid JSON: {exc!r}. Raw: {raw_meta!r}")

        assert "act_chain" in meta, (
            f"'act_chain' key missing from metadata JSON. Keys present: {list(meta.keys())}. "
            "Backend must embed act_chain in AuditLog.Metadata."
        )

        act_chain = meta["act_chain"]
        assert act_chain, "'act_chain' in metadata is falsy/empty"
