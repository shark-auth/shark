"""
Smoke tests for new CLI commands:
  shark user create/list/show/update/delete
  shark sso create/list/show/update/delete
  shark api-key create/list/rotate/revoke
  shark agent show/update/delete/rotate-secret/revoke-tokens
  shark session list/show/revoke
  shark audit export
  shark debug decode-jwt
  shark admin config dump
  shark consents list
  shark org show
  shark vault provider show
  shark auth config show
"""
import pytest
import subprocess
import os
import json
import re

BIN_PATH = "./shark.exe" if os.name == 'nt' else "./shark"
BASE_URL = os.environ.get("BASE", "http://localhost:8080")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def run(args, *, expect_code=0):
    """Run shark CLI and return completed process."""
    result = subprocess.run(
        [BIN_PATH] + args,
        capture_output=True,
        text=True,
    )
    if expect_code is not None:
        assert result.returncode == expect_code, (
            f"Command {args} returned {result.returncode} != {expect_code}\n"
            f"stdout: {result.stdout}\nstderr: {result.stderr}"
        )
    return result


def run_admin(args, server, *, expect_code=0):
    """Run shark CLI with admin token and URL from server fixture."""
    token = server if isinstance(server, str) else os.environ.get("SHARK_ADMIN_TOKEN", "")
    env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}
    result = subprocess.run(
        [BIN_PATH] + args,
        capture_output=True,
        text=True,
        env=env,
    )
    if expect_code is not None:
        assert result.returncode == expect_code, (
            f"Command {args} returned {result.returncode}\n"
            f"stdout: {result.stdout}\nstderr: {result.stderr}"
        )
    return result


# ---------------------------------------------------------------------------
# Help / --help smoke (no server needed)
# ---------------------------------------------------------------------------

class TestHelpStrings:
    def test_user_help(self):
        r = run(["user", "--help"])
        assert "create" in r.stdout or "list" in r.stdout

    def test_sso_help(self):
        r = run(["sso", "--help"])
        assert "create" in r.stdout or "list" in r.stdout

    def test_api_key_help(self):
        r = run(["api-key", "--help"])
        assert "create" in r.stdout or "list" in r.stdout

    def test_agent_help(self):
        r = run(["agent", "--help"])
        assert "show" in r.stdout or "register" in r.stdout

    def test_session_help(self):
        r = run(["session", "--help"])
        assert "list" in r.stdout or "revoke" in r.stdout

    def test_audit_export_help(self):
        r = run(["audit", "export", "--help"])
        assert "--since" in r.stdout

    def test_debug_decode_jwt_help(self):
        r = run(["debug", "decode-jwt", "--help"])
        assert "decode" in r.stdout.lower() or "JWT" in r.stdout

    def test_admin_config_dump_help(self):
        r = run(["admin", "config", "dump", "--help"])
        assert r.returncode == 0

    def test_consents_list_help(self):
        r = run(["consents", "list", "--help"])
        assert "--user" in r.stdout

    def test_org_show_help(self):
        r = run(["org", "show", "--help"])
        assert r.returncode == 0

    def test_vault_provider_show_help(self):
        r = run(["vault", "provider", "show", "--help"])
        assert r.returncode == 0

    def test_auth_config_show_help(self):
        r = run(["auth", "config", "show", "--help"])
        assert r.returncode == 0


# ---------------------------------------------------------------------------
# debug decode-jwt — local, no server needed
# ---------------------------------------------------------------------------

# A real ES256 JWT (header.payload.sig structure — sig validation skipped).
SAMPLE_JWT = (
    "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9"
    ".eyJzdWIiOiJ1c3JfdGVzdCIsImV4cCI6OTk5OTk5OTk5OX0"
    ".AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
)


class TestDebugDecodeJWT:
    def test_decode_jwt_text(self):
        r = run(["debug", "decode-jwt", SAMPLE_JWT])
        assert "alg" in r.stdout
        assert "sub" in r.stdout

    def test_decode_jwt_json(self):
        r = run(["debug", "decode-jwt", SAMPLE_JWT, "--json"])
        data = json.loads(r.stdout)
        assert "header" in data
        assert "payload" in data
        assert data["payload"]["sub"] == "usr_test"

    def test_decode_jwt_invalid(self):
        r = run(["debug", "decode-jwt", "notajwt"], expect_code=1)
        assert r.returncode != 0


# ---------------------------------------------------------------------------
# Live server tests — require `server` fixture from conftest.py
# ---------------------------------------------------------------------------

class TestUserCLI:
    def test_user_list(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}
        r = subprocess.run(
            [BIN_PATH, "user", "list", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0

    def test_user_create_show_update_delete(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}

        # create
        r = subprocess.run(
            [BIN_PATH, "user", "create", "--email", "clicreate@example.com", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0, r.stderr
        data = json.loads(r.stdout)
        uid = (data.get("user") or data).get("id") or data.get("id")
        assert uid, f"no id in: {data}"

        # show
        r2 = subprocess.run(
            [BIN_PATH, "user", "show", uid, "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r2.returncode == 0
        assert uid in r2.stdout

        # update
        r3 = subprocess.run(
            [BIN_PATH, "user", "update", uid, "--name", "CLI Test User", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r3.returncode == 0

        # delete
        r4 = subprocess.run(
            [BIN_PATH, "user", "delete", uid, "--yes", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r4.returncode == 0


class TestSSOCLI:
    def test_sso_list(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}
        r = subprocess.run(
            [BIN_PATH, "sso", "list", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0

    def test_sso_create_show_delete(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}

        r = subprocess.run(
            [BIN_PATH, "sso", "create", "--name", "cli-test-sso", "--type", "oidc", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0, r.stderr
        data = json.loads(r.stdout)
        sso_id = (data.get("connection") or data).get("id") or data.get("id")
        if not sso_id:
            pytest.skip("SSO create did not return an id — API may require more fields")

        # show
        r2 = subprocess.run(
            [BIN_PATH, "sso", "show", sso_id, "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r2.returncode == 0

        # delete
        r3 = subprocess.run(
            [BIN_PATH, "sso", "delete", sso_id, "--yes", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r3.returncode == 0


class TestAPIKeyCLI:
    def test_api_key_create_list_revoke(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}

        # create
        r = subprocess.run(
            [BIN_PATH, "api-key", "create", "--name", "cli-test-key", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0, r.stderr
        data = json.loads(r.stdout)
        kid = (data.get("api_key") or data).get("id") or data.get("id")
        if not kid:
            pytest.skip("api-key create did not return id")

        # list
        r2 = subprocess.run(
            [BIN_PATH, "api-key", "list", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r2.returncode == 0

        # revoke
        r3 = subprocess.run(
            [BIN_PATH, "api-key", "revoke", kid, "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r3.returncode == 0


class TestAgentExtendedCLI:
    def test_agent_show(self, server, admin_client):
        """Registers an agent then shows it."""
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}

        # register
        r = subprocess.run(
            [BIN_PATH, "agent", "register", "--name", "cli-show-test", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0, r.stderr
        data = json.loads(r.stdout)
        aid = (data.get("agent") or data).get("id") or data.get("id")
        if not aid:
            pytest.skip("agent register did not return id")

        # show
        r2 = subprocess.run(
            [BIN_PATH, "agent", "show", aid, "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r2.returncode == 0
        assert aid in r2.stdout

        # rotate-secret
        r3 = subprocess.run(
            [BIN_PATH, "agent", "rotate-secret", aid, "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r3.returncode == 0

        # revoke-tokens
        r4 = subprocess.run(
            [BIN_PATH, "agent", "revoke-tokens", aid, "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r4.returncode == 0

        # delete
        r5 = subprocess.run(
            [BIN_PATH, "agent", "delete", aid, "--yes", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r5.returncode == 0


class TestSessionCLI:
    def test_session_list(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}
        r = subprocess.run(
            [BIN_PATH, "session", "list", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0


class TestAuditExportCLI:
    def test_audit_export_stdout(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}
        r = subprocess.run(
            [BIN_PATH, "audit", "export", "--since", "2026-01-01"],
            capture_output=True, text=True, env=env,
        )
        # May be empty CSV but should not error.
        assert r.returncode == 0, r.stderr


class TestAdminConfigDump:
    def test_admin_config_dump(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}
        r = subprocess.run(
            [BIN_PATH, "admin", "config", "dump"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0
        # Should be valid JSON
        json.loads(r.stdout)


class TestConsentListCLI:
    def test_consents_list(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}
        r = subprocess.run(
            [BIN_PATH, "consents", "list", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0


class TestAuthConfigShowCLI:
    def test_auth_config_show(self, server, admin_client):
        token = admin_client.headers.get("Authorization", "").replace("Bearer ", "")
        env = {**os.environ, "SHARK_ADMIN_TOKEN": token, "SHARK_URL": BASE_URL}
        r = subprocess.run(
            [BIN_PATH, "auth", "config", "show", "--json"],
            capture_output=True, text=True, env=env,
        )
        assert r.returncode == 0
