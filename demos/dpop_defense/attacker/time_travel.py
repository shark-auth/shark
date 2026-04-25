"""
Demo 05 — Attack 5: Time-travel replay (iat too old).

Attacker captured a valid JWT + valid DPoP proof but waited more than 60
seconds before replaying.  The DPoP proof's iat timestamp is now outside
the dpopWindow.

Expected: 401 invalid_token — proof expired.

Demonstrates: dpop.go:177-178 —
    const dpopWindow = 60 * time.Second
    "dpop: proof expired (iat %s is too old)"

Simulation: we manually build a proof with iat backdated 90 seconds,
bypassing the SDK's default iat=now. The SDK exposes an `iat` override
parameter for exactly this kind of test.
"""
from __future__ import annotations
import json, sys, time
import requests
from shark_auth import DPoPProver

with open("/tmp/shark_demo_state.json") as f:
    state = json.load(f)

ACCESS_TOKEN = state["access_token"]
URL          = state["resource_url"]

print("=" * 60)
print("  ATTACK 5 — Time-Travel Replay (iat too old, >60s ago)")
print("  Attacker waited 90 seconds after capturing the proof.")
print("=" * 60)

# Use attacker's own key — the iat rejection fires before cnf.jkt check
attacker = DPoPProver.generate()

stale_iat = int(time.time()) - 90  # 90 seconds ago — outside the 60s window
stale_proof = attacker.make_proof(
    htm="GET",
    htu=URL,
    access_token=ACCESS_TOKEN,
    iat=stale_iat,
)

print(f"\n  Current time : {int(time.time())}")
print(f"  Proof iat    : {stale_iat}  (90 seconds ago)")
print(f"  dpopWindow   : 60 seconds  (dpop.go:24)\n")

r = requests.get(URL, headers={
    "Authorization": f"DPoP {ACCESS_TOKEN}",
    "DPoP": stale_proof,
})
print(f"  HTTP {r.status_code}")
print(f"  Body  : {r.text}\n")

if r.status_code == 401:
    print("  DEFENSE HELD — stale proof blocked.")
    print("  Error: dpop: proof expired (iat … is too old)  (dpop.go:178)")
    print()
    print("  Clock-skew tolerance: ±60 seconds.  An attacker must replay")
    print("  within that window — AND the JTI cache (Attack 3) blocks that too.")
    print("  Both defenses must fail simultaneously for replay to succeed.")
else:
    print("  !! UNEXPECTED PASS — check iat enforcement !!")
    sys.exit(1)
