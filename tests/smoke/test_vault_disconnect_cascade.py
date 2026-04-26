"""
Wave 1.6 Layer 5 smoke test — vault disconnect cascade.

DO NOT RUN directly — orchestrated via pytest from the repo root only.

Verifies that DELETE /api/v1/admin/vault/connections/{id}:
  1. Returns 200 with disconnected=true, revoked_agent_ids, revoked_token_count
  2. Emits a vault.disconnect_cascade audit event queryable via the admin API

Strategy: we cannot easily complete a full OAuth flow in a headless smoke test,
so we short-circuit by inserting rows directly into the SQLite DB:
  - vault_providers row (provider that the connection belongs to)
  - vault_connections row (the connection to disconnect)
  - agents row (an agent to be cascade-revoked)
  - oauth_tokens row (a live token for that agent)
  - audit_logs row with action=vault.token.retrieved, actor_id=agent.id,
    target_id=connection.id  (this is what ListAgentsByVaultRetrieval queries)

Then we DELETE via the admin API and assert the response shape + audit event.
"""

import os
import sqlite3
import time
import uuid

import pytest
import requests

BASE_URL = os.environ.get("BASE", "http://localhost:8080")
DB_PATH = os.environ.get("SHARK_DB", "shark.db")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_headers(admin_key):
    return {"Authorization": f"Bearer {admin_key}"}


def _now_iso():
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _future_iso(seconds=3600):
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + seconds))


# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------

def test_vault_disconnect_cascade(admin_key, db_conn):
    """
    Full Layer 5 cascade path via direct DB seeding + admin DELETE API call.
    """
    headers = _admin_headers(admin_key)
    now = _now_iso()
    future = _future_iso()

    # Unique IDs for this test run
    provider_id = f"vp_{uuid.uuid4().hex[:12]}"
    conn_id = f"vc_{uuid.uuid4().hex[:12]}"
    agent_id = f"agent_{uuid.uuid4().hex[:12]}"
    client_id = f"client_{uuid.uuid4().hex[:12]}"
    token_id = f"tok_{uuid.uuid4().hex[:12]}"
    audit_id = f"aud_{uuid.uuid4().hex[:12]}"

    cur = db_conn.cursor()

    # 1. Insert vault_provider
    cur.execute(
        """
        INSERT INTO vault_providers
            (id, name, display_name, auth_url, token_url, client_id, client_secret_enc,
             scopes, active, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
        """,
        (
            provider_id,
            f"test_provider_{provider_id}",
            "Test Provider",
            "https://provider.example.com/auth",
            "https://provider.example.com/token",
            "test_client_id",
            "enc_placeholder",
            "[]",
            now,
            now,
        ),
    )

    # 2. Insert vault_connection
    cur.execute(
        """
        INSERT INTO vault_connections
            (id, provider_id, user_id, access_token_enc, refresh_token_enc,
             scopes, needs_reauth, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)
        """,
        (
            conn_id,
            provider_id,
            "user_test_cascade",
            "enc_access",
            "enc_refresh",
            "[]",
            now,
            now,
        ),
    )

    # 3. Insert agent (the one that will be cascade-revoked)
    cur.execute(
        """
        INSERT INTO agents
            (id, name, client_id, secret_hash, redirect_uris, grant_types,
             scopes, active, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
        """,
        (
            agent_id,
            f"cascade_agent_{agent_id}",
            client_id,
            "sha256_placeholder",
            "[]",
            '["client_credentials"]',
            "vault:read",
            now,
            now,
        ),
    )

    # 4. Insert a live oauth_token for that agent (token_type=access_token)
    cur.execute(
        """
        INSERT INTO oauth_tokens
            (id, client_id, token_type, token_hash, jti,
             scope, expires_at, created_at)
        VALUES (?, ?, 'access_token', ?, ?, 'vault:read', ?, ?)
        """,
        (
            token_id,
            client_id,
            f"hash_{token_id}",
            f"jti_{token_id}",
            future,
            now,
        ),
    )

    # 5. Insert audit_log row that ListAgentsByVaultRetrieval queries
    cur.execute(
        """
        INSERT INTO audit_logs
            (id, actor_id, actor_type, action, target_type, target_id,
             status, metadata, created_at)
        VALUES (?, ?, 'agent', 'vault.token.retrieved', 'vault_connection', ?,
                'success', '{}', ?)
        """,
        (audit_id, agent_id, conn_id, now),
    )

    db_conn.commit()

    # 6. Call the admin DELETE endpoint
    resp = requests.delete(
        f"{BASE_URL}/api/v1/admin/vault/connections/{conn_id}",
        headers=headers,
        timeout=10,
    )
    assert resp.status_code == 200, (
        f"Expected 200 from admin vault disconnect, got {resp.status_code}: {resp.text}"
    )

    body = resp.json()
    assert body.get("disconnected") is True, f"disconnected flag missing/false: {body}"
    assert body.get("connection_id") == conn_id, f"connection_id mismatch: {body}"

    revoked_ids = body.get("revoked_agent_ids", [])
    assert isinstance(revoked_ids, list), f"revoked_agent_ids not a list: {body}"
    assert agent_id in revoked_ids, (
        f"agent {agent_id} not in revoked_agent_ids {revoked_ids}"
    )

    revoked_count = body.get("revoked_token_count", -1)
    assert revoked_count >= 0, f"revoked_token_count should be >= 0: {body}"
    # We inserted 1 active token; it should have been revoked
    assert revoked_count >= 1, (
        f"Expected at least 1 revoked token, got {revoked_count}: {body}"
    )

    # 7. Verify vault.disconnect_cascade audit event was emitted
    audit_resp = requests.get(
        f"{BASE_URL}/api/v1/admin/audit-logs",
        headers=headers,
        params={"action": "vault.disconnect_cascade", "limit": 10},
        timeout=10,
    )
    assert audit_resp.status_code == 200, (
        f"Audit log query failed: {audit_resp.status_code} {audit_resp.text}"
    )

    audit_body = audit_resp.json()
    # Response is either a list or {data: [...]}
    if isinstance(audit_body, list):
        events = audit_body
    else:
        events = audit_body.get("data", audit_body.get("items", audit_body.get("audit_logs", [])))

    cascade_events = [
        e for e in events
        if e.get("action") == "vault.disconnect_cascade"
        and e.get("target_id") == conn_id
    ]
    assert len(cascade_events) >= 1, (
        f"No vault.disconnect_cascade audit event found for connection {conn_id}. "
        f"Events returned: {events}"
    )
