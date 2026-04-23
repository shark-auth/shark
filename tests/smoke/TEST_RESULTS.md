# Smoke Test Migration Status Report

This document summarizes the current state of the `pytest` migration for SharkAuth smoke tests, highlighting gaps between the legacy `smoke_test.sh` and the new Python suite.

## Summary
- **Total Tests:** 65
- **Passed:** 59
- **Failed:** 6
- **Stability:** ~90%

## Failed Tests & Diagnoses

### 1. `test_admin_deep.py::test_admin_org_mgmt`
- **Result:** `500 == 200`
- **Diagnosis:** Occurs during the Org RBAC section when trying to grant a role that might already be assigned or involves a deleted user/org from previous session scope state. Requires isolation check.

### 2. `test_admin_mgmt.py::test_admin_config_health`
- **Result:** `401 Unauthorized`
- **Diagnosis:** Intermittent failure in admin key capture. While `conftest.py` polls for the key, timing in `dev` mode occasionally leads to 401 if the server hasn't finished internal key registration despite the port being open.

### 3. `test_oauth_advanced.py::test_refresh_token_rotation_and_reuse`
- **Result:** `AssertionError: rt1 != rt2`
- **Diagnosis:** Refresh token rotation is not currently enabled in the `fosite` configuration of this build. The test expects a new refresh token upon exchange, but the server returns the static one.

### 4. `test_user_sessions.py::test_session_list_and_revocation`
- **Result:** `AssertionError: len(sessions) >= 1`
- **Diagnosis:** The session list occasionally returns empty if the database write from the login flow hasn't fully committed or if the user record ID mismatch occurs between signup and login checks.

### 5. `test_user_sessions.py::test_password_change`
- **Result:** `403 Forbidden (email_verification_required)`
- **Diagnosis:** The production backend now enforces email verification for password changes. The test suite needs to either simulate verification (via admin PATCH) or the server needs a `SKIP_VERIFY` flag for smoke testing.

### 6. `test_w15_advanced.py::test_w15_multi_listener_isolation`
- **Result:** `wait_for_port(p_admin) -> False`
- **Diagnosis:** Race condition when starting multiple `shark.exe` instances in the same container/environment. Port collision or database locking (SQLite) occurs when the sub-server tries to initialize on the same path.

## Gaps from Legacy `smoke_test.sh`
- **MFA Flow:** Legacy bash tests used a mock TOTP helper. Python tests currently placeholder MFA registration.
- **DPoP Compliance:** Advanced DPoP headers are partially implemented in Python but not yet fully asserted against RFC 9449 requirements.
- **Port Reliability:** Legacy bash used hardcoded sleep; Python uses polling which is faster but more sensitive to transient port binding errors on Windows.

## Recommendations
1. Implement a `VERIFY_EMAIL=false` dev flag for the server.
2. Force unique database paths for multi-server tests in `test_w15`.
3. Enable `RotateRefreshTokens` in `fosite` once the dependency structure is confirmed.
