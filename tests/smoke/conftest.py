import os
import subprocess
import time
import sqlite3
import yaml
import pytest
import requests
import socket
import re

# Configuration
BASE_URL = os.environ.get("BASE", "http://localhost:8080")
DB_PATH = "dev.db"
YAML_PATH = "smoke_test.yaml"
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
              "sharkauth.db", "sharkauth.db-journal", "sharkauth.db-wal", "sharkauth.db-shm",
              "dev.db", "dev.db-journal", "dev.db-wal", "dev.db-shm",
              YAML_PATH, "server.log"]:
        if os.path.exists(f):
            try:
                os.remove(f)
            except OSError:
                pass

    if not os.path.exists(BIN_PATH):
        pytest.fail(f"Binary {BIN_PATH} not found. Build it first: go build -o {BIN_PATH} ./cmd/shark")

    config = {
        "server": {
            "base_url": BASE_URL,
            "secret": "change-me-this-secret-is-not-secure-at-all-abc123456789"
        },
        "auth": {
            "jwt": {
                "enabled": True,
                "mode": "session",
                "issuer": BASE_URL,
                "audience": "shark-smoke"
            }
        },
        "storage": {
            "path": DB_PATH
        },
        "proxy": {
            "enabled": True,
            "upstream": "http://localhost:8080"
        }
    }
    with open(YAML_PATH, 'w') as f:
        yaml.dump(config, f)

    with open("server.log", "w") as log:
        proc = subprocess.Popen(
            [BIN_PATH, "serve", "--dev", "--config", YAML_PATH],
            stdout=log, stderr=log, text=True
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
    start_time = time.time()
    while time.time() - start_time < 30:
        if os.path.exists("server.log"):
            content = open("server.log").read()
            # Find ALL keys and take the last one to avoid using keys from previous runs
            matches = re.findall(r'sk_live_[A-Za-z0-9_-]{30,}', content)
            if matches: 
                key = matches[-1].strip()
                print(f"DEBUG: Captured Admin Key: {key[:15]}...")
                return key
        time.sleep(1)
    pytest.fail("No admin key captured from server.log")

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
def auth_session(api_session, smoke_user):
    api_session.post(f"{BASE_URL}/api/v1/auth/login", json={"email": smoke_user["email"], "password": smoke_user["password"]})
    return api_session

@pytest.fixture(scope="session")
def admin_client(admin_key):
    s = requests.Session()
    s.headers.update({"Authorization": f"Bearer {admin_key}"})
    return s

@pytest.fixture(scope="session")
def db_conn(server):
    conn = sqlite3.connect(DB_PATH)
    yield conn
    conn.close()
