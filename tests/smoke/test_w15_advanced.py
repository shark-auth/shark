import pytest
import requests
import subprocess
import time
import os
import yaml
import socket

import os
import socket
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def wait_for_port(port, timeout=10):
    start_time = time.time()
    while time.time() - start_time < timeout:
        try:
            with socket.create_connection(("localhost", port), timeout=1):
                return True
        except:
            time.sleep(0.5)
    return False
BIN_PATH = "./shark.exe" if os.name == 'nt' else "./shark"

def find_free_port():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(('', 0))
        return s.getsockname()[1]

def wait_for_port(port, timeout=10):
    start = time.time()
    while time.time() - start < timeout:
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=1):
                return True
        except:
            pass
        time.sleep(0.5)
    return False

@pytest.fixture
def toy_upstreams():
    """Starts two echo upstreams for W15 isolation testing."""
    import http.server
    import threading
    import json

    class EchoHandler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            tag = self.server.tag
            self.send_response(200)
            self.send_header("X-Backend-Tag", tag)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"tag": tag, "path": self.path}).encode())
        def log_message(self, *args): pass

    def serve(port, tag):
        srv = http.server.HTTPServer(("127.0.0.1", port), EchoHandler)
        srv.tag = tag
        srv.serve_forever()

    p1, p2 = find_free_port(), find_free_port()
    t1 = threading.Thread(target=serve, args=(p1, "A"), daemon=True)
    t2 = threading.Thread(target=serve, args=(p2, "B"), daemon=True)
    t1.start()
    t2.start()
    
    yield (p1, p2)

def test_w15_multi_listener_isolation(toy_upstreams, tmp_path):
    """Section 72: W15 multi-listener proxy (embedded)."""
    p_upstream_a, p_upstream_b = toy_upstreams
    p_proxy_a, p_proxy_b = find_free_port(), find_free_port()
    p_admin = find_free_port()
    
    db_path = str(tmp_path / "w15.db")
    cfg_path = str(tmp_path / "w15.yaml")
    
    cfg = {
        "server": {
            "port": p_admin,
            "base_url": f"http://localhost:{p_admin}",
            "secret": "w15-smoke-secret-xxxxxxxxxxxxxxxxxxxxxxxxxxx"
        },
        "storage": {"path": db_path},
        "proxy": {
            "listeners": [
                {"bind": f"127.0.0.1:{p_proxy_a}", "upstream": f"http://127.0.0.1:{p_upstream_a}", "rules": [
                    {"path": "/public/*", "allow": "anonymous"},
                    {"path": "/*", "require": "authenticated"}
                ]},
                {"bind": f"127.0.0.1:{p_proxy_b}", "upstream": f"http://127.0.0.1:{p_upstream_b}", "rules": [
                    {"path": "/*", "allow": "anonymous"}
                ]}
            ]
        }
    }
    
    with open(cfg_path, "w") as f:
        yaml.dump(cfg, f)
        
    with open("w15_server.log", "w") as log:
        proc = subprocess.Popen([BIN_PATH, "serve", "--dev", "--config", cfg_path], 
                                stdout=log, stderr=log)
    
    try:
        if not wait_for_port(p_admin):
            if os.path.exists("w15_server.log"):
                print(f"DEBUG: w15_server.log content:\n{open('w15_server.log').read()}")
            pytest.fail(f"Server failed to start on port {p_admin}")
        assert wait_for_port(p_proxy_a)
        assert wait_for_port(p_proxy_b)
        
        # 1. Listener A: anonymous on public -> 200
        resp = requests.get(f"http://localhost:{p_proxy_a}/public/foo")
        assert resp.status_code == 200
        assert resp.json()["tag"] == "A"
        
        # 2. Listener A: anonymous on private -> 401
        resp = requests.get(f"http://localhost:{p_proxy_a}/secret")
        assert resp.status_code == 401
        
        # 3. Listener B: anonymous on any -> 200
        resp = requests.get(f"http://localhost:{p_proxy_b}/any")
        assert resp.status_code == 200
        assert resp.json()["tag"] == "B"
        
        # 4. Isolation: Listener A should NOT hit B
        resp = requests.get(f"http://localhost:{p_proxy_a}/public/ping")
        assert resp.json()["tag"] == "A"
        
    finally:
        proc.terminate()
        proc.wait()

def test_w15_standalone_proxy_jwt_verify(smoke_user, tmp_path):
    """Section 73: W15 standalone shark proxy JWT verify."""
    import http.server
    import threading

    # 1. Start toy upstream
    p_up = find_free_port()
    def serve():
        class H(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"ok")
            def log_message(self, *a): pass
        http.server.HTTPServer(("127.0.0.1", p_up), H).serve_forever()
    
    threading.Thread(target=serve, daemon=True).start()
    assert wait_for_port(p_up)
    
    # 2. Rules file
    rules_path = tmp_path / "rules73.yaml"
    rules_path.write_text("rules:\n  - path: /*\n    require: authenticated\n")
    
    # 3. Start standalone proxy
    p_proxy = find_free_port()
    # f"proxy --upstream http://127.0.0.1:{p_up} --port {p_proxy} --auth {BASE_URL} --insecure-auth-http --rules {rules_path} --audience shark-smoke --issuer {BASE_URL}"
    args = [
        BIN_PATH, "proxy", 
        "--upstream", f"http://127.0.0.1:{p_up}",
        "--port", str(p_proxy),
        "--auth", BASE_URL,
        "--insecure-auth-http",
        "--rules", str(rules_path),
        "--audience", "shark-smoke",
        "--issuer", BASE_URL
    ]
    
    proc = subprocess.Popen(args, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    
    try:
        assert wait_for_port(p_proxy)
        
        # 4. No token -> 401
        assert requests.get(f"http://localhost:{p_proxy}/").status_code == 401
        
        # 5. Valid JWT -> 200
        token = smoke_user["token"]
        resp = requests.get(f"http://localhost:{p_proxy}/", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 200
        
        # 6. Tampered JWT -> 401
        tampered = token[:-2] + "aa"
        assert requests.get(f"http://localhost:{p_proxy}/", headers={"Authorization": f"Bearer {tampered}"}).status_code == 401
        
    finally:
        proc.terminate()
        proc.wait()
