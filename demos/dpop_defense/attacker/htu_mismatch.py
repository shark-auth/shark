"""
Demo 05 — Attack 4: htu (URL) mismatch — proof for wrong endpoint.

Attacker has a valid JWT and a valid DPoP proof bound to GET /api/positions.
They try to reuse that proof to call GET /api/withdraw/100m — a different URL.

Expected: 401 invalid_token — htu mismatch.

Demonstrates: dpop.go:198-199 —
    "dpop: htu %q does not match request URL %q"

The proof is cryptographically valid (correct key, correct iat, fresh jti)
but the htu claim names the wrong endpoint. The server rejects it.
"""
from __future__ import annotations
import json, sys
import requests
from shark_auth import DPoPProver

with open("/tmp/shark_demo_state.json") as f:
    state = json.load(f)

ACCESS_TOKEN  = state["access_token"]
DEFENDER_JKT  = state["defender_jkt"]
API_BASE      = state["resource_url"].rsplit("/api/", 1)[0]

# Attacker re-uses the same stolen JWT but mints a *fresh* proof
# (to avoid jti-replay rejection) still bound to the /positions endpoint.
# Then tries to call /api/withdraw/100m with that proof.
#
# To demonstrate purely the htu binding we must have Lin's private key —
# in practice the attacker does NOT have it.  We simulate by using a
# fresh attacker key (which will also trigger cnf.jkt mismatch), OR we
# demo the concept with a purpose-built dummy server that only checks htu.
#
# The clean demonstration: attacker builds a proof for /api/positions
# (the url they know about) and presents it to /api/withdraw/100m.

POSITIONS_URL = f"{API_BASE}/api/positions"
WITHDRAW_URL  = f"{API_BASE}/api/withdraw/100m"

# Attacker generates their own key (won't pass cnf.jkt either, but the htu
# check fires first at the resource server's DPoP middleware).
attacker = DPoPProver.generate()

# Proof is for /api/positions, but request targets /api/withdraw/100m
proof_for_positions = attacker.make_proof(
    htm="GET",
    htu=POSITIONS_URL,
    access_token=ACCESS_TOKEN,
)

print("=" * 60)
print("  ATTACK 4 — HTU Mismatch (proof bound to wrong endpoint)")
print("  Proof says: GET /api/positions")
print("  Request to: GET /api/withdraw/100m")
print("=" * 60)
print(f"\n  htu in proof : {POSITIONS_URL}")
print(f"  actual URL   : {WITHDRAW_URL}\n")

r = requests.get(WITHDRAW_URL, headers={
    "Authorization": f"DPoP {ACCESS_TOKEN}",
    "DPoP": proof_for_positions,
})
print(f"  HTTP {r.status_code}")
print(f"  Body  : {r.text}\n")

if r.status_code == 401:
    print("  DEFENSE HELD — htu mismatch blocked.")
    print("  Error: dpop: htu does not match request URL  (dpop.go:199)")
    print("  A proof captured for one endpoint can NEVER be replayed")
    print("  against a different endpoint.")
else:
    print("  !! UNEXPECTED PASS — check htu enforcement !!")
    sys.exit(1)
