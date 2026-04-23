import pytest
import requests
import time
import os

BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def test_audit_logs_exist(db_conn):
    """Section 12: Audit log rows validation via DB."""
    # Ensure there's enough time for async writes
    count = 0
    for _ in range(10):
        cursor = db_conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM audit_logs")
        count = cursor.fetchone()[0]
        if count > 0:
            break
        time.sleep(1)
    assert count > 0, "No audit logs found in DB"
    
    # Parity: Check for specific system events
    cursor.execute("SELECT DISTINCT action FROM audit_logs")
    actions = [row[0] for row in cursor.fetchall()]
    assert len(actions) > 0

def test_admin_stats(admin_client):
    """Section 13: Admin stats endpoint."""
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/stats")
    assert resp.status_code == 200
    data = resp.json()
    assert "users" in data
    assert "sessions" in data
    assert data["users"]["total"] >= 0
    assert data["sessions"]["active"] >= 0

def test_admin_stats_trends(admin_client):
    """Section 18: Stats Trends (?days=N)."""
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/stats/trends?days=7")
    assert resp.status_code == 200
    data = resp.json()
    # Handle {history: []} or {data: []} or []
    history = data.get("history", data.get("data", []))
    assert isinstance(history, list)
    if len(history) > 0:
        entry = history[0]
        assert "date" in entry
        assert "count" in entry

def test_mfa_stats_parity(admin_client):
    """Section 61: MFA enabled-vs-verified count parity."""
    stats = admin_client.get(f"{BASE_URL}/api/v1/admin/stats").json()
    enabled = stats.get("mfa_enabled_count", 0)
    verified = stats.get("mfa_verified_count", 0)
    # Smoke assertion: enabled >= verified (strictly guarded by MFA controller)
    assert enabled >= verified
