import pytest
import requests
import time
import random

import os
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def test_session_list_and_revocation(auth_session, smoke_user):
    """Section 16 & 17: List and revoke user sessions."""
    # List
    resp = auth_session.get(f"{BASE_URL}/api/v1/auth/sessions")
    assert resp.status_code == 200
    sessions = resp.json().get("data", [])
    assert len(sessions) >= 1
    sess_id = sessions[0]["id"]
    
    # Revoke (different session)
    # We'll create a second session for this
    s2 = requests.Session()
    s2.post(f"{BASE_URL}/api/v1/auth/login", json={"email": smoke_user["email"], "password": smoke_user["password"]})
    resp = s2.get(f"{BASE_URL}/api/v1/auth/sessions")
    s2_sessions = resp.json().get("data", [])
    s2_sess_id = next(s for s in s2_sessions if s["id"] != sess_id)["id"]
    
    # Delete S2 from S1
    resp = auth_session.delete(f"{BASE_URL}/api/v1/auth/sessions/{s2_sess_id}")
    assert resp.status_code == 204
    
    # Verify S2 is revoked
    resp = s2.get(f"{BASE_URL}/api/v1/auth/sessions")
    assert resp.status_code == 401

def test_admin_session_filtering(admin_client, smoke_user):
    """Section 20: Admin session filters."""
    # Filter by user_id
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/sessions?user_id={smoke_user['id']}")
    assert resp.status_code == 200
    for s in resp.json()["data"]:
        assert s["user_id"] == smoke_user["id"]

def test_password_change(admin_client):
    """Section 23: Password Change."""
    email = f"pass_{int(time.time())}_{random.randint(1000, 9999)}@test.com"
    old_pw, new_pw = "OldPass123!", "NewPass999!"
    
    # 1. Signup
    resp = requests.post(f"{BASE_URL}/api/v1/auth/signup", json={"email": email, "password": old_pw})
    assert resp.status_code in [200, 201], f"Signup failed: {resp.text}"
    
    # 2. Login
    sess = requests.Session()
    sess.post(f"{BASE_URL}/api/v1/auth/login", json={"email": email, "password": old_pw})

    # 3. Change password multiple times to verify state
    for i in range(2):
        resp = sess.post(f"{BASE_URL}/api/v1/auth/password/change", json={
             "current_password": old_pw, "new_password": new_pw
        })
        assert resp.status_code == 200, f"Password change failed: {resp.text}"
        old_pw, new_pw = new_pw, f"New{random.randint(1000, 9999)}!"
