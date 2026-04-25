"""
Gemma Connect Flow — Demo server for SharkAuth Token Vault Demo 04
Handles user-facing "Connect Provider" clicks, proxying through Shark's vault OAuth.

Usage:
    SHARK_URL=http://localhost:8000 \
    ADMIN_TOKEN=demo-admin-key \
    python server.py

Endpoints:
    GET /                         — lists connected providers for demo user
    GET /connect?provider=X       — redirects to Shark vault connect URL
    GET /connected                — OAuth return_to landing page
    GET /disconnect?provider=X    — revokes via Shark admin API
"""

import os
import sys
import httpx
from flask import Flask, redirect, request, jsonify, render_template_string

SHARK_URL = os.environ.get("SHARK_URL", "http://localhost:8000")
ADMIN_TOKEN = os.environ.get("ADMIN_TOKEN", "demo-admin-key")
USER_ID = os.environ.get("DEMO_USER_ID", "user_42")
SERVER_PORT = int(os.environ.get("PORT", "3000"))

app = Flask(__name__)

PROVIDERS = ["google_gmail", "slack", "github", "notion", "linear"]

INDEX_HTML = """
<!DOCTYPE html>
<html>
<head><title>Gemma — Connect Your Tools</title>
<style>
  body { font-family: system-ui, sans-serif; max-width: 640px; margin: 60px auto; padding: 0 20px; }
  h1 { font-size: 1.4rem; }
  .card { border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 12px 0;
          display: flex; justify-content: space-between; align-items: center; }
  .connected { background: #f0fdf4; border-color: #86efac; }
  .btn { padding: 8px 16px; border-radius: 6px; border: none; cursor: pointer; font-size: 0.9rem; }
  .btn-connect { background: #3b82f6; color: white; }
  .btn-disconnect { background: #fee2e2; color: #dc2626; }
  .badge { font-size: 0.75rem; background: #22c55e; color: white; padding: 2px 8px; border-radius: 99px; }
</style>
</head>
<body>
<h1>Gemma — Connect Your Tools</h1>
<p style="color:#64748b">Demo user: <code>{{ user_id }}</code> &nbsp;|&nbsp; Powered by <strong>SharkAuth Token Vault</strong></p>
{% for p in providers %}
<div class="card {% if p.connected %}connected{% endif %}">
  <div>
    <strong>{{ p.display_name }}</strong>
    {% if p.connected %}<span class="badge">Connected</span>{% endif %}
    {% if p.scopes %}<br><small style="color:#64748b">Scopes: {{ p.scopes }}</small>{% endif %}
  </div>
  <div>
    {% if p.connected %}
      <a href="/disconnect?provider={{ p.name }}"><button class="btn btn-disconnect">Disconnect</button></a>
    {% else %}
      <a href="/connect?provider={{ p.name }}"><button class="btn btn-connect">Connect</button></a>
    {% endif %}
  </div>
</div>
{% endfor %}
<hr>
<p style="font-size:0.8rem;color:#94a3b8">
  All tokens stored AES-256-GCM encrypted by Shark. Agent never sees raw credentials.
</p>
</body>
</html>
"""

CONNECTED_HTML = """
<!DOCTYPE html>
<html>
<head><title>Connected!</title>
<style>body { font-family: system-ui, sans-serif; max-width: 480px; margin: 80px auto; text-align: center; }</style>
</head>
<body>
<h1>Connected!</h1>
<p><strong>{{ provider }}</strong> is now connected to Gemma.</p>
<p style="color:#64748b;font-size:0.9rem">
  Shark stored your token encrypted with AES-256-GCM.<br>
  Gemma's agent will never see your password or refresh token.
</p>
<p><a href="/">Back to dashboard</a></p>
</body>
</html>
"""


def get_connections() -> list[dict]:
    try:
        resp = httpx.get(
            f"{SHARK_URL}/api/v1/vault/connections",
            headers={"Authorization": f"Bearer {ADMIN_TOKEN}", "X-User-ID": USER_ID},
            timeout=5,
        )
        if resp.status_code == 200:
            return resp.json().get("connections", [])
    except Exception as e:
        print(f"[WARN] Could not fetch connections: {e}", file=sys.stderr)
    return []


@app.get("/")
def index():
    connections = get_connections()
    connected_names = {c.get("provider_name", "") for c in connections}
    scopes_by_name = {c.get("provider_name", ""): c.get("scopes", []) for c in connections}
    display = {
        "google_gmail": "Gmail",
        "slack": "Slack",
        "github": "GitHub",
        "notion": "Notion",
        "linear": "Linear",
    }
    provider_list = [
        {
            "name": p,
            "display_name": display.get(p, p.title()),
            "connected": p in connected_names,
            "scopes": ", ".join(scopes_by_name.get(p, [])),
        }
        for p in PROVIDERS
    ]
    return render_template_string(INDEX_HTML, providers=provider_list, user_id=USER_ID)


@app.get("/connect")
def connect():
    provider = request.args.get("provider", "")
    if not provider:
        return "Missing provider", 400
    return_to = f"http://localhost:{SERVER_PORT}/connected?provider={provider}"
    shark_connect_url = (
        f"{SHARK_URL}/api/v1/vault/connect/{provider}"
        f"?redirect_uri={return_to}&user_id={USER_ID}"
    )
    print(f"[CONNECT] Redirecting to Shark vault: {shark_connect_url}")
    return redirect(shark_connect_url)


@app.get("/connected")
def connected():
    provider = request.args.get("provider", "unknown")
    print(f"[CONNECTED] Provider={provider} user={USER_ID} — vault.connected webhook should have fired")
    return render_template_string(CONNECTED_HTML, provider=provider)


@app.get("/disconnect")
def disconnect():
    provider = request.args.get("provider", "")
    if not provider:
        return "Missing provider", 400
    connections = get_connections()
    conn_id = next(
        (c["id"] for c in connections if c.get("provider_name") == provider), None
    )
    if not conn_id:
        return redirect("/")
    resp = httpx.delete(
        f"{SHARK_URL}/api/v1/admin/vault/connections/{conn_id}",
        headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
        timeout=5,
    )
    print(f"[DISCONNECT] provider={provider} conn_id={conn_id} status={resp.status_code}")
    return redirect("/")


if __name__ == "__main__":
    print(f"[CONNECT-FLOW] Starting on port {SERVER_PORT}")
    print(f"[CONNECT-FLOW] Shark URL: {SHARK_URL}")
    app.run(port=SERVER_PORT, debug=True)
