"""
Demo 05 — Final scoreboard graphic.

Renders a terminal scoreboard showing all 6 attack vectors, their
SharkAuth defenses, and the contrast with a vanilla bearer server.

Usage:
    python demos/dpop_defense/scoreboard.py
"""
from __future__ import annotations
import os, time

# ANSI colours
RED    = "\033[91m"
GREEN  = "\033[92m"
YELLOW = "\033[93m"
CYAN   = "\033[96m"
BOLD   = "\033[1m"
RESET  = "\033[0m"
DIM    = "\033[2m"

WIDTH = 76

def hr(char="═"):
    print(char * WIDTH)

def row(attack: str, claim: str, file_ref: str, shark: str, vanilla: str):
    blocked  = f"{GREEN}BLOCKED ✓{RESET}"
    pwned    = f"{RED}PWNED  ✗{RESET}"
    s = blocked if shark == "BLOCKED" else pwned
    v = pwned   if vanilla == "PWNED"  else blocked
    print(f"  {attack:<30} {claim:<10} {s}   {v}  {DIM}{file_ref}{RESET}")

print()
hr()
print(f"{BOLD}{'  DEMO 05 — TOKEN THEFT LIVE ATTACK':^{WIDTH}}{RESET}")
print(f"{'  DPoP Stops Stolen Agent Tokens Cold (RFC 9449)':^{WIDTH}}")
hr()
print()
print(f"  {'ATTACK VECTOR':<30} {'CLAIM':<10} {'SHARKAUTH':<12} {'VANILLA BEARER':<16} {'FILE'}")
print(f"  {'-'*30} {'-'*10} {'-'*12} {'-'*16} {'-'*20}")

attacks = [
    ("1. Bearer replay (no proof)",     "—",    "dpop.go:96",  "BLOCKED", "PWNED"),
    ("2. Forged key (attacker keypair)", "cnf",  "dpop.go:223", "BLOCKED", "PWNED"),
    ("3. JTI replay (exact proof copy)","jti",  "dpop.go:75",  "BLOCKED", "PWNED"),
    ("4. htu mismatch (wrong endpoint)","htu",  "dpop.go:199", "BLOCKED", "PWNED"),
    ("5. Time-travel (iat > 60s old)",  "iat",  "dpop.go:178", "BLOCKED", "PWNED"),
    ("6. Refresh token theft",          "cnf",  "handlers:383","BLOCKED", "PWNED"),
]

for (attack, claim, ref, shark, vanilla) in attacks:
    row(attack, claim, ref, shark, vanilla)
    time.sleep(0.25)  # dramatic effect

print()
hr("─")

# Totals
print(f"\n  {GREEN}{BOLD}SharkAuth  : 6/6 attacks BLOCKED{RESET}    "
      f"{RED}{BOLD}Vanilla Bearer: 6/6 attacks PWNED{RESET}")
print(f"\n  {BOLD}Attacker time-to-pwn (SharkAuth) = ∞{RESET}")
print(f"  {BOLD}Attacker time-to-pwn (Bearer-only) = {RED}< 10 seconds{RESET}{BOLD}{RESET}")

hr("─")
print(f"\n  {CYAN}{BOLD}HOW IT WORKS{RESET}")
print(f"  Every SharkAuth agent token carries a {BOLD}cnf.jkt{RESET} claim: the")
print(f"  SHA-256 thumbprint of the agent's ECDSA P-256 public key.")
print(f"  Every protected request requires a signed {BOLD}DPoP proof{RESET} JWT that:")
print(f"    • binds to the exact HTTP method  ({BOLD}htm{RESET}  claim)")
print(f"    • binds to the exact endpoint URL ({BOLD}htu{RESET}  claim)")
print(f"    • binds to the access token       ({BOLD}ath{RESET}  claim, SHA-256)")
print(f"    • has a unique nonce              ({BOLD}jti{RESET}  claim, replay cache)")
print(f"    • is timestamped within 60 seconds({BOLD}iat{RESET}  claim)")
print(f"  Steal the token. You still need the {BOLD}private key{RESET}. Game over.")

hr("─")
print(f"\n  {YELLOW}HONEST GAPS{RESET}")
print(f"  • JTI cache is {BOLD}process-local{RESET} (SCALE.md §1).")
print(f"    Horizontal scale → replay possible across replicas until Q3 2026")
print(f"    (Redis-backed JTI store on roadmap).")
print(f"  • Clock skew tolerance: ±60 s. NTP sync required on agent hosts.")

hr("─")
print(f"\n  {CYAN}WHO SHOULD CARE{RESET}")
print(f"  • Regulated finance (PCI DSS 3.2.1 key-bound credentials)")
print(f"  • Healthcare AI (HIPAA, PHI-adjacent agent tokens)")
print(f"  • Any org with a bearer-token incident in their history")
print()
hr()
print()
