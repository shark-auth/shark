"""
Demo 05 — Attack 1: Bearer Replay (no DPoP proof at all).

Attacker stole the JWT from a log file. Tries to use it as a plain Bearer
token. Expected: 401 invalid_token — missing DPoP proof.

Demonstrates: dpop.go:96 — "dpop: missing proof JWT"
Resource middleware: dpop.go:305-308 — "DPoP header is required"
"""
from __future__ import annotations
import json, sys
import requests

with open("/tmp/shark_demo_state.json") as f:
    state = json.load(f)

ACCESS_TOKEN = state["access_token"]
URL          = state["resource_url"]

print("=" * 60)
print("  ATTACK 1 — Bearer Replay (no DPoP proof)")
print("  Attacker stole JWT from logs. Sending raw Bearer request.")
print("=" * 60)
print(f"\n  Target : GET {URL}")
print(f"  Token  : {ACCESS_TOKEN[:40]}…\n")

r = requests.get(URL, headers={"Authorization": f"Bearer {ACCESS_TOKEN}"})
print(f"  HTTP {r.status_code}")
print(f"  Body  : {r.text}\n")

if r.status_code == 401:
    print("  DEFENSE HELD — bearer replay blocked.")
    print("  Error: missing DPoP proof (dpop.go:96)")
    print("  Audit: auth.dpop_invalid  |  Webhook: agent.dpop_replay_detected")
else:
    print("  !! UNEXPECTED PASS — check server config !!")
    sys.exit(1)
