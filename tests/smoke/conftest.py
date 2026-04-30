import os
import subprocess
import time
import sqlite3
import pytest
import requests
import socket
import re

# Configuration — W17: no yaml, no --config. Server boots from defaults.
BASE_URL = os.environ.get("BASE", "http://localhost:8080")
# bootstrap.go resolves DB path as: flag > SHARK_DB_PATH env > db_state > ./shark.db
# config.go has "./data/sharkauth.db" but bootstrap takes priority. We honour the
# env var so CI can override, and fall back to the bootstrap default ./shark.db.
_shark_db_env = os.environ.get("SHARK_DB_PATH", "")
DB_PATH = _shark_db_env if _shark_db_env else "shark.db"
KEY_PATH = os.path.join(os.path.dirname(DB_PATH) or ".", "admin.key.firstboot")
BIN_PATH = "./shark.exe" if os.name == 'nt' else "./shark"

def kill_port(port):
    if os.name == 'nt':
        try:
            output = subprocess.check_output(f'netstat -ano | findstr LISTENING | findstr :{port}', shell=True).decode()
            for line in output.splitlines():
                if f':{port}' in line:
                    pid = line.strip().split()[-1]
                    subprocess.call(['taskkill', '/F', '/PID', pid], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        except subprocess.CalledProcessError:
            pass
    else:
        subprocess.call(['fuser', '-k', f'{port}/tcp'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    time.sleep(1)

@pytest.fixture(scope="session", autouse=True)
def server():
    """Bootstrap fresh DB, config and start server for the entire session."""
    kill_port(8080)
    for f in [DB_PATH, f"{DB_PATH}-journal", f"{DB_PATH}-wal", f"{DB_PATH}-shm",
              "shark.db", "shark.db-journal", "shark.db-wal", "shark.db-shm",
              "data/sharkauth.db", "data/sharkauth.db-journal", "data/sharkauth.db-wal", "data/sharkauth.db-shm",
              "sharkauth.db", "sharkauth.db-journal", "sharkauth.db-wal", "sharkauth.db-shm",
              "dev.db", "dev.db-journal", "dev.db-wal", "dev.db-shm",
              "admin.key.firstboot", "data/admin.key.firstboot",
              "server.log"]:
        if os.path.exists(f):
            try:
                os.remove(f)
            except OSError:
                pass

    if not os.path.exists(BIN_PATH):
        pytest.fail(f"Binary {BIN_PATH} not found. Build it first: go build -o {BIN_PATH} ./cmd/shark")

    # W17: no yaml file, no --config flag. Server boots from defaults; first boot
    # auto-generates server.secret + JWT signing key + admin API key (printed to stdout).
    with open("server.log", "w") as log:
        env = os.environ.copy()
        # Pin DB path so the binary writes ./shark.db + ./admin.key.firstboot in
        # the repo root, matching DB_PATH/KEY_PATH at top of this file. Without
        # this, the binary uses the config default (data/sharkauth.db) and the
        # admin_key fixture can't find the key file.
        env["SHARK_DB_PATH"] = DB_PATH
        proc = subprocess.Popen(
            [BIN_PATH, "serve", "--no-prompt"],
            stdout=log, stderr=log, text=True,
            env=env,
        )

    start_time = time.time()
    while time.time() - start_time < 15:
        try:
            if requests.get(f"{BASE_URL}/healthz", timeout=1).status_code == 200: break
        except: pass
        time.sleep(0.2)
    else:
        proc.terminate()
        pytest.fail("Server failed to come up")

    yield proc
    proc.terminate()
    proc.wait()
    kill_port(8080)

@pytest.fixture(scope="session")
def admin_key(server):
    # W17: first boot writes full admin key to <db_dir>/admin.key.firstboot.
    # KEY_PATH is computed from DB_PATH at module load so it always matches
    # whichever DB the server actually opened (./shark.db by default).
    key_path = KEY_PATH
    start_time = time.time()
    while time.time() - start_time < 30:
        if os.path.exists(key_path):
            key = open(key_path).read().strip()
            if key.startswith("sk_live_"):
                print(f"DEBUG: Captured Admin Key: {key[:15]}...")
                return key
        time.sleep(1)
    pytest.fail(f"No admin key captured from {key_path}")

# W1.5/W2 wave-test compatibility aliases — wave subagents invented these names
# instead of reading conftest.py first. Aliases keep their tests collectable.
@pytest.fixture(scope="session")
def admin_token(admin_key):
    return admin_key

@pytest.fixture(scope="session")
def shark_base_url():
    return BASE_URL

@pytest.fixture(scope="session")
def shark_admin_key(admin_key):
    return admin_key

@pytest.fixture(scope="session")
def base_url():
    return BASE_URL

@pytest.fixture(scope="session")
def api_session():
    return requests.Session()

@pytest.fixture(scope="session")
def smoke_user(api_session, server):
    """Provides a shared user for smoke tests."""
    email = f"sharedsmoke@test.com"
    password = "Password123!"
    resp = requests.post(f"{BASE_URL}/api/v1/auth/signup", json={"email": email, "password": password})
    if resp.status_code not in [200, 201]:
        resp = requests.post(f"{BASE_URL}/api/v1/auth/login", json={"email": email, "password": password})
    data = resp.json()
    return {"email": email, "password": password, "token": data.get("token"), "id": data.get("id")}

@pytest.fixture
def auth_session(smoke_user):
    """Fresh session with its own cookie jar, already logged in."""
    s = requests.Session()
    resp = s.post(
        f"{BASE_URL}/api/v1/auth/login",
        json={"email": smoke_user["email"], "password": smoke_user["password"]},
    )
    assert resp.status_code == 200, f"auth_session login failed: {resp.text}"
    return s

@pytest.fixture(scope="session")
def admin_client(admin_key):
    s = requests.Session()
    s.headers.update({"Authorization": f"Bearer {admin_key}"})
    return s

@pytest.fixture(scope="function")
def db_conn(server):
    # Function-scoped + 30s busy_timeout to avoid contending with shark's writer.
    # Session-scoped sqlite3 connections caused SQLITE_BUSY cascades against shark's
    # backend, since SQLite is single-writer even under WAL.
    conn = sqlite3.connect(DB_PATH, timeout=30.0)
    conn.execute("PRAGMA busy_timeout=30000")
    try:
        yield conn
    finally:
        try:
            conn.commit()
        except Exception:
            pass
        conn.close()
