"""
Demo 05 — Attack 2: Forged DPoP proof (attacker's own keypair).

Attacker has the stolen JWT. Generates a *fresh* ECDSA P-256 keypair,
builds a syntactically valid DPoP proof signed with their own key.
Expected: 401 invalid_token — jkt in proof != cnf.jkt in token.

Demonstrates: handlers.go resource validation where proof.jkt is compared
against the cnf.jkt stored in the JWT's cnf claim. The server extracts
jkt from the DPoP proof header (ComputeJWKThumbprint → dpop.go:223) and
compares it against jwt.cnf.jkt from the access token claims.
Error path: "dpop: ath mismatch" or resource server cnf.jkt check.
"""
from __future__ import annotations
import json, sys
import requests
from shark_auth import DPoPProver

with open("/tmp/shark_demo_state.json") as f:
    state = json.load(f)

ACCESS_TOKEN  = state["access_token"]
URL           = state["resource_url"]
DEFENDER_JKT  = state["defender_jkt"]

print("=" * 60)
print("  ATTACK 2 — Forged DPoP proof (attacker's own key)")
print("  Attacker generates new keypair, builds valid-looking proof.")
print("=" * 60)

# Attacker generates their OWN fresh keypair — different from Lin's
attacker_prover = DPoPProver.generate()
print(f"\n  Defender jkt : {DEFENDER_JKT}")
print(f"  Attacker jkt : {attacker_prover.jkt}  ← DIFFERENT")
print(f"  JWT cnf.jkt  : {DEFENDER_JKT}  (baked into token at issuance)")
print(f"\n  Server will compare: proof.jkt ({attacker_prover.jkt[:16]}…)")
print(f"                  vs   jwt.cnf.jkt ({DEFENDER_JKT[:16]}…) → MISMATCH\n")

forged_proof = attacker_prover.make_proof(
    htm="GET",
    htu=URL,
    access_token=ACCESS_TOKEN,   # ath still matches — the token IS stolen
)

r = requests.get(URL, headers={
    "Authorization": f"DPoP {ACCESS_TOKEN}",
    "DPoP": forged_proof,
})
print(f"  HTTP {r.status_code}")
print(f"  Body  : {r.text}\n")

if r.status_code == 401:
    print("  DEFENSE HELD — forged-key proof blocked.")
    print("  cnf.jkt mismatch: resource server rejects proof from unknown key.")
    print("  The stolen JWT is worthless without Lin's private key.")
else:
    print("  !! UNEXPECTED PASS — check cnf.jkt enforcement !!")
    sys.exit(1)
