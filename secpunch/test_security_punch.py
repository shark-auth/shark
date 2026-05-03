import base64
import json
import os
import platform
import re
import shutil
import socket
import subprocess
import threading
import tempfile
import time
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen

import pytest


ROOT = Path(__file__).resolve().parents[1]


def read_text(relpath: str) -> str:
    return (ROOT / relpath).read_text(encoding="utf-8", errors="replace")


def strip_comments(source: str) -> str:
    source = re.sub(r"/\*[\s\S]*?\*/", "", source)
    return "\n".join(line.split("//", 1)[0] for line in source.splitlines())


def existing_files(*candidates: str):
    return [ROOT / p for p in candidates if (ROOT / p).exists()]


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


class SharkClient:
    def __init__(self, base_url: str, admin_key: str | None = None):
        self.base_url = base_url.rstrip("/")
        self.admin_key = admin_key

    def request(self, method, path, *, headers=None, json_body=None, form=None, basic=None, bearer=None):
        headers = dict(headers or {})
        body = None
        if json_body is not None:
            body = json.dumps(json_body).encode("utf-8")
            headers["Content-Type"] = "application/json"
        if form is not None:
            body = urlencode(form).encode("utf-8")
            headers["Content-Type"] = "application/x-www-form-urlencoded"
        if basic is not None:
            raw = f"{basic[0]}:{basic[1]}".encode("utf-8")
            headers["Authorization"] = "Basic " + base64.b64encode(raw).decode("ascii")
        if bearer is not None:
            headers["Authorization"] = "Bearer " + bearer
        req = Request(self.base_url + path, data=body, headers=headers, method=method)
        try:
            with urlopen(req, timeout=10) as resp:
                data = resp.read()
                return resp.status, dict(resp.headers), decode_body(data)
        except HTTPError as exc:
            return exc.code, dict(exc.headers), decode_body(exc.read())

    def admin(self, method, path, **kwargs):
        if not self.admin_key:
            pytest.skip("SHARK_SECPUNCH_ADMIN_KEY is required when testing an external server")
        return self.request(method, path, bearer=self.admin_key, **kwargs)


def decode_body(data: bytes):
    if not data:
        return None
    text = data.decode("utf-8", errors="replace")
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        return text


class ManagedServer:
    def __init__(self, binary: Path):
        self.binary = binary
        self.tmp = tempfile.TemporaryDirectory(prefix="shark-secpunch-")
        self.root = Path(self.tmp.name)
        self.port = free_port()
        self.base_url = f"http://127.0.0.1:{self.port}"
        self.proc: subprocess.Popen | None = None
        self.admin_key = None
        self._stdout_lines: list[str] = []

    def start(self):
        env = os.environ.copy()
        env.update(
            {
                "SHARK_PORT": str(self.port),
                "SHARK_DB_PATH": str(self.root / "shark.db"),
                "SHARK_DATA_DIR": str(self.root),
                "SHARK_DEV_MODE": "1",
            }
        )
        self.proc = subprocess.Popen(
            [str(self.binary), "serve", "--no-prompt"],
            cwd=str(ROOT),
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            encoding="utf-8",
            errors="replace",
        )

        def _drain_stdout():
            if self.proc and self.proc.stdout:
                for line in self.proc.stdout:
                    self._stdout_lines.append(line)
        threading.Thread(target=_drain_stdout, daemon=True).start()

        self._wait_ready()
        self.admin_key = self._read_firstboot_key()
        return SharkClient(self.base_url, self.admin_key)

    def restart(self):
        self.stop()
        return self.start()

    def stop(self):
        if self.proc and self.proc.poll() is None:
            self.proc.terminate()
            try:
                self.proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.proc.kill()
                self.proc.wait(timeout=10)
        self.proc = None

    def cleanup(self):
        self.stop()
        self.tmp.cleanup()

    def _wait_ready(self):
        deadline = time.time() + 30
        last_error = None
        while time.time() < deadline:
            if self.proc and self.proc.poll() is not None:
                output = "".join(self._stdout_lines)
                raise RuntimeError(f"shark exited early with {self.proc.returncode}:\n{output}")
            try:
                with urlopen(self.base_url + "/healthz", timeout=1) as resp:
                    if resp.status < 500:
                        return
            except (HTTPError, URLError, TimeoutError, ConnectionError) as exc:
                last_error = exc
            time.sleep(0.2)
        raise RuntimeError(f"shark did not become ready: {last_error}")

    def _read_firstboot_key(self):
        deadline = time.time() + 10
        candidates = [
            self.root / "admin.key.firstboot",
            self.root / "admin.key",
            ROOT / "admin.key.firstboot",
        ]
        while time.time() < deadline:
            for path in candidates:
                if path.exists():
                    text = path.read_text(encoding="utf-8", errors="replace").strip()
                    match = re.search(r"sk_live_[A-Za-z0-9_-]+", text)
                    if match:
                        return match.group(0)
            time.sleep(0.1)
        return None


@pytest.fixture(scope="session")
def shark_server():
    if os.environ.get("SHARK_SECPUNCH_SKIP_INTEGRATION") == "1":
        pytest.skip("integration checks disabled by SHARK_SECPUNCH_SKIP_INTEGRATION")
    base_url = os.environ.get("SHARK_SECPUNCH_BASE_URL")
    if base_url:
        yield SharkClient(base_url, os.environ.get("SHARK_SECPUNCH_ADMIN_KEY"))
        return

    default_name = "shark.exe" if platform.system().lower().startswith("win") else "shark"
    binary = Path(os.environ.get("SHARK_SECPUNCH_BINARY", ROOT / default_name))
    if not binary.exists():
        pytest.skip(f"no shark binary found at {binary}; set SHARK_SECPUNCH_BINARY")
    managed = ManagedServer(binary)
    try:
        yield managed.start()
    finally:
        managed.cleanup()


def make_agent(client: SharkClient, name: str, scopes=None):
    scopes = scopes or ["email:read", "email:write", "vault:read"]
    payload = {
        "name": name,
        "scopes": scopes,
        "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
    }
    status, _, body = client.admin("POST", "/api/v1/agents", json_body=payload)
    assert status in (200, 201), body
    agent = body.get("agent", body) if isinstance(body, dict) else {}
    client_id = agent.get("client_id") or agent.get("id")
    client_secret = agent.get("client_secret") or agent.get("secret")
    assert client_id and client_secret, body
    return client_id, client_secret


def issue_client_credentials(client: SharkClient, agent_id, agent_secret, scope):
    status, _, body = client.request(
        "POST",
        "/oauth/token",
        form={"grant_type": "client_credentials", "scope": scope},
        basic=(agent_id, agent_secret),
    )
    assert status == 200, body
    return body["access_token"]


def exchange_token(client: SharkClient, agent_id, agent_secret, subject_token, scope):
    return client.request(
        "POST",
        "/oauth/token",
        form={
            "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
            "subject_token": subject_token,
            "subject_token_type": "urn:ietf:params:oauth:token-type:access_token",
            "requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
            "scope": scope,
        },
        basic=(agent_id, agent_secret),
    )


def test_firstboot_key_endpoint_never_publicly_returns_secret(shark_server):
    status, _, body = shark_server.request("GET", "/api/v1/admin/firstboot/key")
    serialized = json.dumps(body) if not isinstance(body, str) else body
    assert status in (401, 403, 404, 410), (status, body)
    assert "sk_live_" not in serialized


def test_runtime_config_survives_restart():
    if os.environ.get("SHARK_SECPUNCH_BASE_URL"):
        pytest.skip("restart persistence check requires a managed local process")
    if os.environ.get("SHARK_SECPUNCH_SKIP_INTEGRATION") == "1":
        pytest.skip("integration checks disabled")
    default_name = "shark.exe" if platform.system().lower().startswith("win") else "shark"
    binary = Path(os.environ.get("SHARK_SECPUNCH_BINARY", ROOT / default_name))
    if not binary.exists():
        pytest.skip(f"no shark binary found at {binary}; set SHARK_SECPUNCH_BINARY")

    managed = ManagedServer(binary)
    try:
        client = managed.start()
        patch = {"server": {"cors_relaxed": True}, "auth": {"password_min_length": 13}}
        status, _, body = client.admin("PATCH", "/api/v1/admin/config", json_body=patch)
        assert status in (200, 204), body
        client = managed.restart()
        status, _, body = client.admin("GET", "/api/v1/admin/config")
        assert status == 200, body
        serialized = json.dumps(body)
        assert '"cors_relaxed": true' in serialized.lower(), body
        assert '"min_length": 13' in serialized.lower() or '"password_min_length": 13' in serialized.lower(), body
    finally:
        managed.cleanup()


def test_server_config_fields_are_not_silently_ignored(shark_server):
    desired = {
        "server": {
            "base_url": "https://auth.example.test",
            "cors_origins": ["https://app.example.test"],
            "port": 9443,
        }
    }
    status, _, body = shark_server.admin("PATCH", "/api/v1/admin/config", json_body=desired)
    if status >= 400:
        return
    assert status in (200, 204), body
    status, _, body = shark_server.admin("GET", "/api/v1/admin/config")
    assert status == 200, body
    serialized = json.dumps(body)
    assert "https://auth.example.test" in serialized, body
    assert "https://app.example.test" in serialized, body


def test_revoked_subject_token_cannot_be_exchanged(shark_server):
    actor_id, actor_secret = make_agent(shark_server, "secpunch-revoked-actor")
    delegate_id, delegate_secret = make_agent(shark_server, "secpunch-revoked-delegate")
    subject = issue_client_credentials(shark_server, actor_id, actor_secret, "email:read")
    status, _, body = shark_server.request(
        "POST",
        "/oauth/revoke",
        form={"token": subject, "token_type_hint": "access_token"},
        bearer=shark_server.admin_key,
    )
    assert status in (200, 204), body
    status, _, body = exchange_token(shark_server, delegate_id, delegate_secret, subject, "email:read")
    assert status in (400, 401, 403), (status, body)
    assert "invalid" in json.dumps(body).lower() or "revoked" in json.dumps(body).lower(), body


def test_token_exchange_requires_live_may_act_grant(shark_server):
    actor_id, actor_secret = make_agent(shark_server, "secpunch-no-grant-actor")
    delegate_id, delegate_secret = make_agent(shark_server, "secpunch-no-grant-delegate")
    subject = issue_client_credentials(shark_server, actor_id, actor_secret, "email:read")
    status, _, body = exchange_token(shark_server, delegate_id, delegate_secret, subject, "email:read")
    assert status in (400, 401, 403), (status, body)
    assert "may_act" in json.dumps(body).lower() or "not authorized" in json.dumps(body).lower(), body


def test_bootstrap_completion_not_based_on_recent_audit_rows():
    source = read_text("internal/api/admin_bootstrap_handlers.go")
    function = re.search(r"func \(.*?\) MintBootstrapToken[\s\S]+?\n}", source)
    assert function, "MintBootstrapToken function not found"
    body = function.group(0)
    assert "QueryAuditLogs" not in body
    assert "ActorType" not in body or '"admin"' not in body


def test_invalid_env_bootstrap_key_is_rejected():
    sources = "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in existing_files("internal/api/admin_bootstrap_handlers.go", "internal/server/server.go", "internal/config/config.go", "internal/server/firstboot.go")
    )
    key_region = "\n".join(line for line in sources.splitlines() if "SHARKAUTH_BOOTSTRAP_ADMIN_KEY" in line or "sk_live_" in line)
    assert "SHARKAUTH_BOOTSTRAP_ADMIN_KEY" in sources
    assert re.search(r"SHARKAUTH_BOOTSTRAP_ADMIN_KEY[\s\S]{0,1200}sk_live_", sources), key_region


def test_login_frontend_does_not_fetch_firstboot_key():
    frontend = "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in existing_files(
            "admin/src/components/login.tsx",
            "admin/src/components/App.tsx",
            "admin/src/components/api.tsx",
        )
    )
    assert "/admin/firstboot/key" not in frontend
    assert "firstboot/key" not in frontend


def test_admin_key_not_persisted_in_local_storage():
    admin_files = list((ROOT / "admin" / "src").rglob("*.ts")) + list((ROOT / "admin" / "src").rglob("*.tsx"))
    offenders = []
    for path in admin_files:
        text = path.read_text(encoding="utf-8", errors="replace")
        if re.search(r"localStorage\.(setItem|getItem|removeItem)\(['\"]shark_admin_key['\"]", text):
            offenders.append(str(path.relative_to(ROOT)))
    assert not offenders, "admin bearer key must not use persistent localStorage: " + ", ".join(offenders)


def test_dev_email_links_reject_unsafe_schemes():
    sources = "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in existing_files(
            "admin/src/components/dev_email.tsx",
        )
    )
    assert sources, "dev email frontend source not found"
    assert re.search(r"protocol\s*===?\s*['\"]https?:['\"]|startsWith\(['\"]https?://", sources), (
        "dev email magic/reset links must whitelist http/https before rendering hrefs"
    )
    assert "javascript:" not in sources.lower()


def test_delegation_grant_drawer_uses_canonical_agent_ids():
    sources = "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in list((ROOT / "admin" / "src").rglob("*delegation*.tsx"))
    )
    assert sources, "delegation frontend source not found"
    assert "AgentGrantsTables" in sources
    grant_region = re.search(r"AgentGrantsTables[\s\S]{0,1200}", sources)
    assert grant_region, "AgentGrantsTables use not found"
    assert re.search(r"strip|canonical|agentId|realId", grant_region.group(0), re.I), grant_region.group(0)


def test_delegation_chain_loader_follows_audit_pagination():
    sources = "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in list((ROOT / "admin" / "src" / "components").rglob("*Delegation*.tsx"))
        + list((ROOT / "admin" / "src" / "components").rglob("*Overview*.tsx"))
    )
    assert sources, "admin frontend source not found"
    code = strip_comments(sources)
    assert not re.search(r"\blimit\s*[:=]\s*50\b", code), "fixed 50-row chain loads hide later pages"
    assert re.search(r"next[_-]?cursor", code, re.I), "delegation chain loader must read next_cursor from responses"
    assert re.search(r"\bcursor\b", code, re.I), "delegation chain loader must send cursor on follow-up requests"
    assert re.search(r"\b(while|for)\b[\s\S]{0,800}\b(fetch|API\.get)\b", code, re.I), (
        "delegation chain loader needs a loop that fetches all audit pages"
    )


def test_vault_reads_validate_dpop_for_bound_tokens():
    sources = "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in existing_files(
            "internal/api/vault_handlers.go",
            "internal/api/vault.go",
            "internal/server/server.go",
        )
    )
    assert sources, "vault handler source not found"
    vault_region = re.search(r"func .*Vault.*Token[\s\S]+?(?=\nfunc |\Z)", sources, re.I)
    assert vault_region, "vault token handler not found"
    body = vault_region.group(0)
    assert "DPoP" in body
    assert re.search(r"jkt|thumbprint|ValidateDPoP|cnf", body, re.I), body


def test_session_jwt_auth_checks_live_session_or_revocation():
    sources = "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in existing_files(
            "internal/api/middleware.go",
            "internal/api/session_handlers.go",
            "internal/store/sessions.go",
            "internal/auth/session.go",
        )
    )
    assert sources, "session auth source not found"
    assert re.search(r"revoked_jti|RevokedJTI|session.*revok|GetSession|SessionBy", sources, re.I), (
        "session JWT middleware must reject deleted sessions or revoked session JTIs"
    )


def test_dpop_htu_path_is_case_sensitive():
    sources = "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in existing_files("internal/oauth/dpop.go", "internal/api/dpop.go")
    )
    assert sources, "DPoP source not found"
    htu_region = re.search(r"func\s+htuMatches[\s\S]+?\n}", sources)
    assert htu_region, "DPoP htuMatches function not found"
    body = strip_comments(htu_region.group(0))
    assert "strings.EqualFold(claim, request)" not in body, "DPoP path/query comparison must be case-sensitive"
    assert "url.Parse" in body or re.search(r"Host|Hostname\(\)", body), (
        "DPoP HTU comparison must isolate scheme/host from path before applying case-insensitive comparison"
    )
    assert "claim[ci+len(prefix):] == request[ri+len(prefix):]" not in body, (
        "this compares host+path case-sensitively; host must be case-insensitive while path remains exact"
    )


def test_dockerfile_go_version_matches_go_mod():
    go_mod = read_text("go.mod")
    dockerfile = read_text("Dockerfile")
    go_version = re.search(r"^go\s+([0-9]+\.[0-9]+(?:\.[0-9]+)?)", go_mod, re.M)
    image_version = re.search(r"golang:([0-9]+\.[0-9]+(?:\.[0-9]+)?)", dockerfile)
    assert go_version and image_version, "could not parse Go versions"
    assert image_version.group(1) == go_version.group(1), (
        f"Dockerfile uses Go {image_version.group(1)}, go.mod requires {go_version.group(1)}"
    )
