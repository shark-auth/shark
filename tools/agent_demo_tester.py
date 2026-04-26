"""
Interactive agent-features tester for SharkAuth.

Walks through every agent-side capability against a running `shark serve` on
localhost:8080. Pauses between steps so you can switch to the dashboard
(http://localhost:8080/admin) and see each effect appear in real time.

Usage:
    python tools/agent_demo_tester.py
    # Pastes the admin API key when prompted (sk_live_...)

What it exercises (in order):
    1. Create a synthetic human user (admin API)
    2. Register 3 agents under that user (created_by linkage)
    3. Issue DPoP-bound client_credentials tokens for each
    4. Configure may_act delegation policies between them
    5. Run the full delegation chain (token_exchange RFC 8693)
    6. Provision a synthetic Gmail vault entry for the user
    7. Fetch from vault using DPoP-bound delegated token
    8. Rotate one agent's DPoP key (Layer 4 primitive — Wave 2 Method 10)
    9. Bulk-revoke by client_id pattern (Layer 4 — Wave 1.6)
   10. Cascade-revoke all agents owned by user (Layer 3 — Wave 1.5)
   11. Vault disconnect cascade (Layer 5 — Wave 1.6)

After each step, the script prints the dashboard URL where the effect should
be visible (Audit log, Agent detail drawer, Vault tab, etc.).

Cleanup runs at the end unless you Ctrl-C — partial state is intentional so
you can inspect mid-run.
"""

from __future__ import annotations

import getpass
import json
import secrets
import sys
import time
from dataclasses import dataclass
from typing import Any

import requests

# Reuse the SDK's DPoP prover for proof signing — same code customers use.
try:
    from shark_auth.dpop import DPoPProver
except ImportError:
    print("ERROR: shark_auth SDK not installed. Run: pip install -e sdk/python/")
    sys.exit(1)


BASE_URL = "http://localhost:8080"
DASHBOARD = f"{BASE_URL}/admin"


# ---------------------------------------------------------------------------
# Pretty-print helpers
# ---------------------------------------------------------------------------

def step(n: int, title: str) -> None:
    print()
    print("=" * 70)
    print(f"  STEP {n}  ·  {title}")
    print("=" * 70)


def info(msg: str) -> None:
    print(f"  → {msg}")


def ok(msg: str) -> None:
    print(f"  ✓ {msg}")


def fail(msg: str) -> None:
    print(f"  ✗ {msg}")


def dashboard(path: str, hint: str) -> None:
    print(f"  [DASHBOARD] {DASHBOARD}{path}  —  {hint}")


def pause(msg: str = "Press ENTER to continue, Ctrl-C to stop") -> None:
    try:
        input(f"\n  ⏸  {msg} ")
    except (KeyboardInterrupt, EOFError):
        print("\n  Aborted.")
        sys.exit(0)


# ---------------------------------------------------------------------------
# HTTP helpers
# ---------------------------------------------------------------------------

class Client:
    def __init__(self, admin_key: str) -> None:
        self.admin_key = admin_key
        self.s = requests.Session()
        self.s.headers["Authorization"] = f"Bearer {admin_key}"

    def get(self, path: str, **kwargs: Any) -> requests.Response:
        return self.s.get(f"{BASE_URL}{path}", timeout=10, **kwargs)

    def post(self, path: str, json_body: Any = None, **kwargs: Any) -> requests.Response:
        return self.s.post(f"{BASE_URL}{path}", json=json_body, timeout=10, **kwargs)

    def delete(self, path: str, **kwargs: Any) -> requests.Response:
        return self.s.delete(f"{BASE_URL}{path}", timeout=10, **kwargs)


@dataclass
class Agent:
    id: str
    client_id: str
    client_secret: str
    name: str
    prover: DPoPProver


# ---------------------------------------------------------------------------
# Step implementations
# ---------------------------------------------------------------------------

def health_check(c: Client) -> None:
    r = c.get("/healthz")
    if r.status_code != 200:
        fail(f"shark not reachable at {BASE_URL} (status {r.status_code}). Run ./shark.exe serve in another terminal.")
        sys.exit(1)
    ok(f"shark reachable at {BASE_URL}")


def create_user(c: Client, suffix: str) -> dict:
    body = {
        "email": f"demo_tester_{suffix}@sharkauth.dev",
        "password": "DemoPassword!2026",
    }
    r = c.post("/api/v1/admin/users", json_body=body)
    if r.status_code not in (200, 201):
        fail(f"user create failed: {r.status_code} {r.text}")
        sys.exit(1)
    user = r.json()
    ok(f"user {user['email']} created  (id={user['id']})")
    return user


def register_agent(c: Client, name: str, user_id: str) -> Agent:
    prover = DPoPProver.generate()
    body = {
        "name": name,
        "scopes": ["email:read", "email:write", "vault:read", "mcp:read", "mcp:write"],
        "grant_types": [
            "client_credentials",
            "urn:ietf:params:oauth:grant-type:token-exchange",
        ],
        "created_by": user_id,
    }
    r = c.post("/api/v1/agents", json_body=body)
    if r.status_code not in (200, 201):
        fail(f"agent create failed: {r.status_code} {r.text}")
        sys.exit(1)
    a = r.json()
    ok(f"agent {name}  (id={a['id']}  jkt={prover.jkt[:12]}...)")
    return Agent(
        id=a["id"],
        client_id=a["client_id"],
        client_secret=a["client_secret"],
        name=name,
        prover=prover,
    )


def issue_dpop_token(agent: Agent) -> str:
    token_url = f"{BASE_URL}/oauth/token"
    proof = agent.prover.make_proof(htm="POST", htu=token_url)
    r = requests.post(
        token_url,
        data={"grant_type": "client_credentials", "scope": "email:read email:write vault:read"},
        auth=(agent.client_id, agent.client_secret),
        headers={"DPoP": proof},
        timeout=10,
    )
    if r.status_code != 200:
        fail(f"token issue failed for {agent.name}: {r.status_code} {r.text}")
        return ""
    tok = r.json()["access_token"]
    ok(f"token issued for {agent.name}  (cnf.jkt bound to keypair)")
    return tok


def configure_policies(c: Client, actor: Agent, target: Agent, scopes: list[str]) -> None:
    body = {"may_act": [{"agent_id": target.id, "scopes": scopes}]}
    r = c.post(f"/api/v1/agents/{actor.id}/policies", json_body=body)
    if r.status_code in (200, 201):
        ok(f"policy: {actor.name} may_act for {target.name} (scopes: {', '.join(scopes)})")
    else:
        fail(f"policy set failed: {r.status_code} {r.text}")


def token_exchange(c: Client, actor: Agent, subject_token: str) -> str:
    token_url = f"{BASE_URL}/oauth/token"
    proof = actor.prover.make_proof(htm="POST", htu=token_url)
    r = requests.post(
        token_url,
        data={
            "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
            "subject_token": subject_token,
            "subject_token_type": "urn:ietf:params:oauth:token-type:access_token",
            "actor_token": subject_token,
            "actor_token_type": "urn:ietf:params:oauth:token-type:access_token",
            "requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
            "scope": "email:read",
        },
        auth=(actor.client_id, actor.client_secret),
        headers={"DPoP": proof},
        timeout=10,
    )
    if r.status_code != 200:
        fail(f"exchange failed for {actor.name}: {r.status_code} {r.text}")
        return ""
    tok = r.json()["access_token"]
    ok(f"exchange: {actor.name} got delegated token (act-chain extends)")
    return tok


def vault_provision(c: Client, user_id: str) -> str | None:
    # Step 1: ensure a 'google_gmail' provider exists. Connection creation
    # requires a real OAuth flow (session-auth, browser redirect), so the
    # tester only sets up the provider + tells the user how to finish in
    # the dashboard. The vault hop is then exercisable on a re-run.
    provider_body = {
        "name": "google_gmail",
        "display_name": "Google Gmail (demo)",
        "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
        "token_url": "https://oauth2.googleapis.com/token",
        "client_id": "demo-client-id",
        "client_secret": "demo-client-secret",
        "scopes": ["https://www.googleapis.com/auth/gmail.readonly"],
    }
    r = c.post("/api/v1/vault/providers", json_body=provider_body)
    if r.status_code in (200, 201):
        ok("vault provider 'google_gmail' provisioned")
    elif r.status_code == 409 or "exists" in r.text.lower():
        ok("vault provider 'google_gmail' already exists (re-using)")
    else:
        fail(f"provider create failed: {r.status_code} {r.text}")
        return None

    # Connection creation requires real OAuth. Direct the operator.
    info("Vault CONNECTION requires a real OAuth flow — auto-provisioning skipped.")
    info(f"To finish step 7 manually, log into the dashboard as user_id={user_id}")
    info(f"and connect the 'google_gmail' provider, then re-run this tester.")
    return None


def vault_fetch(agent: Agent, token: str) -> bool:
    url = f"{BASE_URL}/api/v1/vault/google_gmail/token"
    proof = agent.prover.make_proof(htm="GET", htu=url, access_token=token)
    r = requests.get(
        url,
        headers={"Authorization": f"DPoP {token}", "DPoP": proof},
        timeout=10,
    )
    if r.status_code == 200:
        ok(f"{agent.name} fetched from vault (DPoP-bound, audited)")
        return True
    fail(f"vault fetch failed: {r.status_code} {r.text}")
    return False


def rotate_dpop_key(c: Client, agent: Agent) -> Agent:
    new_prover = DPoPProver.generate()
    body = {
        "new_public_jwk": new_prover.public_jwk,
        "reason": "demo tester key rotation",
    }
    r = c.post(f"/api/v1/agents/{agent.id}/rotate-dpop-key", json_body=body)
    if r.status_code != 200:
        fail(f"rotate failed: {r.status_code} {r.text}")
        return agent
    body = r.json()
    ok(f"rotated key for {agent.name}  old={body.get('old_jkt', '')[:12]}...  new={body.get('new_jkt', '')[:12]}...  revoked_tokens={body.get('revoked_token_count', 0)}")
    agent.prover = new_prover
    return agent


def bulk_revoke(c: Client, pattern: str) -> int:
    body = {"client_id_pattern": pattern, "reason": "demo tester bulk revoke"}
    r = c.post("/api/v1/admin/oauth/revoke-by-pattern", json_body=body)
    if r.status_code != 200:
        fail(f"bulk revoke failed: {r.status_code} {r.text}")
        return 0
    count = r.json().get("revoked_count", 0)
    ok(f"bulk-revoke pattern '{pattern}'  →  {count} tokens revoked")
    return count


def cascade_revoke(c: Client, user_id: str) -> dict:
    body = {"reason": "demo tester cascade"}
    r = c.post(f"/api/v1/users/{user_id}/revoke-agents", json_body=body)
    if r.status_code != 200:
        fail(f"cascade failed: {r.status_code} {r.text}")
        return {}
    body = r.json()
    ok(f"cascade-revoke user {user_id}  →  agents={len(body.get('revoked_agent_ids', []))}  consents={body.get('revoked_consent_count', 0)}")
    return body


def vault_disconnect(c: Client, connection_id: str) -> None:
    r = c.delete(f"/api/v1/admin/vault/connections/{connection_id}")
    if r.status_code != 200:
        fail(f"vault disconnect failed: {r.status_code} {r.text}")
        return
    body = r.json()
    ok(f"vault disconnect  →  cascade revoked {len(body.get('revoked_agent_ids', []))} agents, {body.get('revoked_token_count', 0)} tokens")


# ---------------------------------------------------------------------------
# Main flow
# ---------------------------------------------------------------------------

def main() -> None:
    print()
    print("SharkAuth · Agent Features Tester")
    print("=" * 70)
    print(f"  Target: {BASE_URL}")
    print(f"  Open the dashboard in your browser: {DASHBOARD}")
    print()

    admin_key = getpass.getpass("Paste your admin API key (sk_live_...): ").strip()
    if not admin_key.startswith("sk_"):
        print("WARNING: key doesn't start with sk_ — proceeding anyway")

    c = Client(admin_key)

    # ------------------------------------------------------------------ 0
    health_check(c)

    suffix = secrets.token_hex(4)

    # ------------------------------------------------------------------ 1
    step(1, "Create synthetic human user")
    user = create_user(c, suffix)
    dashboard("/users", f"user {user['email']} should appear in the list")
    pause()

    # ------------------------------------------------------------------ 2
    step(2, "Register 3 agents (created_by linkage to user)")
    a1 = register_agent(c, f"demo-{suffix}-user-proxy", user["id"])
    a2 = register_agent(c, f"demo-{suffix}-email-svc", user["id"])
    a3 = register_agent(c, f"demo-{suffix}-followup-svc", user["id"])
    dashboard("/agents", "3 agents w/ createdBy filter pointing to demo user")
    dashboard(f"/users/{user['id']}/agents", "user's agents view (W1.5 endpoint)")
    pause()

    # ------------------------------------------------------------------ 3
    step(3, "Issue DPoP-bound client_credentials tokens")
    t1 = issue_dpop_token(a1)
    t2 = issue_dpop_token(a2)
    t3 = issue_dpop_token(a3)
    dashboard("/audit", "audit log shows oauth.token.issued × 3 with cnf.jkt populated")
    dashboard(f"/agents/{a1.id}", "agent drawer Security tab shows the DPoP key thumbprint")
    pause()

    # ------------------------------------------------------------------ 4
    step(4, "Configure may_act delegation policies")
    configure_policies(c, a1, a2, ["email:read", "email:write", "vault:read"])
    configure_policies(c, a2, a3, ["email:read", "vault:read"])
    dashboard(f"/agents/{a1.id}", "agent drawer Delegation Policies tab shows configured policies")
    pause()

    # ------------------------------------------------------------------ 5
    step(5, "Run delegation chain (token_exchange RFC 8693)")
    t2_delegated = token_exchange(c, a2, t1)
    t3_delegated = token_exchange(c, a3, t2_delegated) if t2_delegated else ""
    dashboard("/audit", "audit log shows token_exchanged events with act-chain breadcrumb")
    pause()

    # ------------------------------------------------------------------ 6
    step(6, "Provision synthetic Gmail vault entry")
    connection_id = vault_provision(c, user["id"])
    if connection_id:
        dashboard("/vault", f"vault connection {connection_id} appears for the demo user")
    pause()

    # ------------------------------------------------------------------ 7
    step(7, "Fetch from vault with DPoP-bound delegated token (Layer 5 chain)")
    fetched = False
    if connection_id and t3_delegated:
        fetched = vault_fetch(a3, t3_delegated)
        dashboard("/audit", "audit log shows vault.token.retrieved by followup-svc")
    else:
        info("skipped — vault provision or delegation chain didn't complete")
    pause()

    # ------------------------------------------------------------------ 8
    step(8, "Rotate one agent's DPoP key (Wave 2 Method 10)")
    rotate_dpop_key(c, a2)
    dashboard("/audit", "agent.dpop_key_rotated event with old_jkt + new_jkt")
    dashboard(f"/agents/{a2.id}", "Security tab shows the new key thumbprint")
    pause()

    # ------------------------------------------------------------------ 9
    step(9, "Bulk-revoke tokens by client_id pattern (Layer 4 — Wave 1.6)")
    bulk_revoke(c, f"{a1.client_id}*")  # only matches a1
    dashboard("/audit", "oauth.bulk_revoke_pattern event with metadata")
    pause()

    # ------------------------------------------------------------------ 10
    step(10, "Cascade-revoke entire user fleet (Layer 3 — Wave 1.5)")
    cascade_revoke(c, user["id"])
    dashboard("/audit", "user.cascade_revoked_agents event with revoked_agent_ids")
    dashboard("/agents", "all 3 demo agents now show active=false")
    pause()

    # ------------------------------------------------------------------ 11
    if connection_id:
        step(11, "Vault disconnect cascade (Layer 5 — Wave 1.6)")
        vault_disconnect(c, connection_id)
        dashboard("/audit", "vault.disconnected + vault.disconnect_cascade events")
    else:
        info("step 11 skipped — no vault connection to disconnect")

    # ------------------------------------------------------------------
    print()
    print("=" * 70)
    print("  COMPLETE")
    print("=" * 70)
    print(f"  Run ID suffix: {suffix}")
    print(f"  Demo user:     {user['email']}  ({user['id']})")
    print(f"  Agents:        {a1.id}, {a2.id}, {a3.id}")
    print()
    print(f"  Inspect everything at: {DASHBOARD}/audit?actor_type=agent")
    print()
    print("  Cleanup: leaving state intact for inspection. Re-run anytime —")
    print("  each run uses a fresh suffix so no collisions.")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n  Stopped by user.")
        sys.exit(0)
