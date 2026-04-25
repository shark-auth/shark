"""
Demo 05 — Attack 6: Refresh token theft from a different machine.

Attacker has the opaque refresh token (e.g., stolen from a database leak
or from the same logs that exposed the access token).  Tries to use it to
get a fresh access token from a *different machine* (no private key).

Expected: 401 invalid_dpop_proof — refresh requires same-jkt DPoP.

Demonstrates: RFC 9449 §5 — refresh tokens are bound to the same key pair
used at initial issuance.  handlers.go:storeDPoPJKT records tok.DPoPJKT.
When the attacker presents a DPoP proof from a *different* key (or none),
the jkt doesn't match the stored tok.DPoPJKT → rejected.

The attacker can't generate a valid proof for the original key because they
don't have Lin's private key — only the opaque refresh token string.
"""
from __future__ import annotations
import json, sys
import requests
from shark_auth import DPoPProver

with open("/tmp/shark_demo_state.json") as f:
    state = json.load(f)

REFRESH_TOKEN = state.get("refresh_token", "")
TOKEN_URL     = state["token_url"]
CLIENT_ID     = state["client_id"]
CLIENT_SECRET = state["client_secret"]
DEFENDER_JKT  = state["defender_jkt"]

if not REFRESH_TOKEN:
    print("No refresh token in state (client_credentials flow). Skipping.")
    print("To demo refresh binding: use authorization_code flow and re-run agent.py")
    sys.exit(0)

print("=" * 60)
print("  ATTACK 6 — Refresh Token Theft (different machine, no private key)")
print("  Attacker has refresh_token but NOT Lin's ECDSA private key.")
print("=" * 60)
print(f"\n  refresh_token (first 20): {REFRESH_TOKEN[:20]}…")
print(f"  Bound to jkt            : {DEFENDER_JKT}")
print(f"  Attacker will present a proof from a DIFFERENT keypair.\n")

# Attacker generates a fresh keypair — different jkt from Lin's
attacker = DPoPProver.generate()
print(f"  Attacker jkt : {attacker.jkt}  ← does not match stored tok.DPoPJKT")

attacker_proof = attacker.make_proof(htm="POST", htu=TOKEN_URL)

r = requests.post(
    TOKEN_URL,
    data={
        "grant_type": "refresh_token",
        "refresh_token": REFRESH_TOKEN,
    },
    auth=(CLIENT_ID, CLIENT_SECRET),
    headers={"DPoP": attacker_proof},
)
print(f"\n  HTTP {r.status_code}")
print(f"  Body  : {r.text}\n")

if r.status_code == 401:
    print("  DEFENSE HELD — refresh token theft blocked.")
    print("  The refresh token is DPoP-bound: only the holder of the original")
    print("  private key can use it.  Stolen refresh_token → useless.")
    print()
    print("  RFC 9449 §5: 'the client MUST present a DPoP proof for the same")
    print("  key that was used to obtain the refresh token.'")
    print("  handlers.go:storeDPoPJKT — tok.DPoPJKT persisted at issuance.")
else:
    print("  !! UNEXPECTED PASS — check refresh DPoP binding !!")
    sys.exit(1)
