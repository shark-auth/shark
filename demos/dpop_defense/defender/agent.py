"""
Demo 05 — Defender side: quant-trader-agent (Lin's side).

Registers agent, mints a DPoP-bound token via SharkAuth, then calls the
protected /api/positions endpoint using a proper DPoP proof.

Usage:
    python agent.py

Environment:
    SHARK_URL   — SharkAuth base URL (default: http://localhost:8080)
    API_URL     — protected resource URL (default: http://localhost:9000)
"""
from __future__ import annotations
import os, json, sys
import requests
from shark_auth import DPoPProver

SHARK = os.getenv("SHARK_URL", "http://localhost:8080")
API   = os.getenv("API_URL",   "http://localhost:9000")

def banner(msg: str) -> None:
    print(f"\n{'='*60}\n  {msg}\n{'='*60}")

# ── 1. Register the agent (DCR) ───────────────────────────────────────────────
banner("STEP 1  Register quant-trader-agent (RFC 7591 DCR)")
reg = requests.post(f"{SHARK}/oauth/register", json={
    "client_name": "quant-trader-agent",
    "grant_types": ["client_credentials"],
    "token_endpoint_auth_method": "client_secret_basic",
}).json()
CLIENT_ID     = reg["client_id"]
CLIENT_SECRET = reg["client_secret"]
print(f"  client_id: {CLIENT_ID}")

# ── 2. Generate DPoP keypair ──────────────────────────────────────────────────
banner("STEP 2  Generate ECDSA P-256 DPoP keypair")
prover = DPoPProver.generate()
print(f"  jkt (thumbprint): {prover.jkt}")

# ── 3. Mint DPoP-bound access token ──────────────────────────────────────────
banner("STEP 3  Mint DPoP-bound access token via client_credentials")
token_url = f"{SHARK}/oauth/token"
dpop_proof_for_token = prover.make_proof(htm="POST", htu=token_url)
resp = requests.post(
    token_url,
    data={"grant_type": "client_credentials", "scope": "positions:read"},
    auth=(CLIENT_ID, CLIENT_SECRET),
    headers={"DPoP": dpop_proof_for_token},
)
resp.raise_for_status()
tok = resp.json()
ACCESS_TOKEN   = tok["access_token"]
REFRESH_TOKEN  = tok.get("refresh_token", "")
print(f"  token_type : {tok['token_type']}")   # must be "DPoP"
print(f"  access_token (first 40): {ACCESS_TOKEN[:40]}…")
print(f"  refresh_token: {'yes' if REFRESH_TOKEN else 'none'}")

# ── 4. Call protected resource ────────────────────────────────────────────────
banner("STEP 4  Call /api/positions — SHOULD SUCCEED")
resource_url = f"{API}/api/positions"
dpop_proof_for_api = prover.make_proof(
    htm="GET",
    htu=resource_url,
    access_token=ACCESS_TOKEN,
)
r = requests.get(
    resource_url,
    headers={
        "Authorization": f"DPoP {ACCESS_TOKEN}",
        "DPoP": dpop_proof_for_api,
    },
)
print(f"  HTTP {r.status_code}: {r.text}")

# ── 5. Export state for attacker scripts ──────────────────────────────────────
state = {
    "access_token":    ACCESS_TOKEN,
    "refresh_token":   REFRESH_TOKEN,
    "dpop_proof":      dpop_proof_for_api,   # one valid proof captured
    "resource_url":    resource_url,
    "shark_url":       SHARK,
    "client_id":       CLIENT_ID,
    "client_secret":   CLIENT_SECRET,
    "token_url":       token_url,
    "defender_jkt":    prover.jkt,
}
with open("/tmp/shark_demo_state.json", "w") as f:
    json.dump(state, f, indent=2)
print("\n  [state written to /tmp/shark_demo_state.json for attacker scripts]")
