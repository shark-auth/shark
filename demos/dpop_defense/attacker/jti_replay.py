"""
Demo 05 — Attack 3: Full replay (stolen JWT + stolen DPoP proof).

Attacker captured both the Authorization header AND the DPoP header from
a previous request (e.g., MITM, logging of full request headers, or
prompt-injection leakage).  Replays the exact proof within the 60-second
window.

Expected: 401 invalid_token — JTI replay detected.

Demonstrates: dpop.go:74-75 — MarkSeen → "dpop: jti already seen (replay detected)"
The JTI cache sees the same jti string and rejects it.
"""
from __future__ import annotations
import json, sys
import requests

with open("/tmp/shark_demo_state.json") as f:
    state = json.load(f)

ACCESS_TOKEN = state["access_token"]
STOLEN_PROOF = state["dpop_proof"]   # exact proof from defender's successful call
URL          = state["resource_url"]

print("=" * 60)
print("  ATTACK 3 — JTI Replay (stolen JWT + stolen DPoP proof)")
print("  Attacker replays the exact proof used by Lin's agent.")
print("=" * 60)
print(f"\n  Replaying proof (first 40): {STOLEN_PROOF[:40]}…")
print(f"  This proof's jti was already registered in the JTI cache.\n")

r = requests.get(URL, headers={
    "Authorization": f"DPoP {ACCESS_TOKEN}",
    "DPoP": STOLEN_PROOF,
})
print(f"  HTTP {r.status_code}")
print(f"  Body  : {r.text}\n")

if r.status_code == 401:
    print("  DEFENSE HELD — JTI replay blocked.")
    print("  Error: dpop: jti already seen (replay detected)  (dpop.go:75)")
    print("  Audit: auth.dpop_invalid  |  Webhook: agent.dpop_replay_detected")
    print()
    print("  NOTE: JTI cache is process-local (in-memory). This defense holds")
    print("        for single-replica deployments. See SCALE.md §1 for the")
    print("        horizontal-scale gap and Q3-2026 Redis fix roadmap.")
else:
    print("  !! UNEXPECTED PASS — is this a fresh token? Rerun agent.py first !!")
    sys.exit(1)
