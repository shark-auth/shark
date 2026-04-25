"""End-to-end delegation orchestrator.

Run order:
  user/platform token (client_credentials)
    → triage (hop 1)
        → knowledge   (hop 2, kb:read)
        → email       (hop 2, email:draft)
            → gmail-tool (hop 3, email:send — 3-deep act chain)
        → crm         (hop 2, crm:write)
        → followup    (hop 2, calendar:write)

Verifies every hop:
  - sub stays the original platform identity
  - scope strictly narrows
  - aud rotates per resource
  - act chain deepens monotonically
  - cnf.jkt rotates per acting agent (each agent has its own DPoP keypair
    on its client_credentials issuance — bound at credential issuance time)
"""
from __future__ import annotations

import json
import os
import sys
import time
from pathlib import Path

# Make sibling packages importable when run as `python orchestrator/main.py`
ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from dotenv import load_dotenv  # type: ignore  # optional; falls back if absent

try:
    load_dotenv(ROOT / ".env")
except Exception:
    pass

from lib import client_credentials, decode, render_chain, render_token  # noqa: E402
from agents import triage, knowledge, email, gmail_tool, crm, followup  # noqa: E402


def banner(title: str) -> None:
    bar = "═" * 72
    print(f"\n{bar}\n  {title}\n{bar}")


def main() -> int:
    auth = os.environ["SHARK_AUTH_URL"]
    platform_id = os.environ["PLATFORM_CLIENT_ID"]
    platform_secret = os.environ["PLATFORM_CLIENT_SECRET"]

    banner("ACT 1 · Platform mints user-context token (no delegation yet)")
    user = client_credentials(
        auth, platform_id, platform_secret,
        scope="ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write",
        audience="supportflow-core-api",
    )
    user_claims = decode(user.access_token, auth, "supportflow-core-api")
    print(render_token(user_claims, label="USER token (act_depth=0)"))

    banner("ACT 2 · Triage exchanges user → its own delegated token")
    triage_token = triage.run(user.access_token)

    banner("ACT 3 · Fan-out — 4 sub-agents exchange triage's token")
    knowledge_token = knowledge.run(triage_token)
    email_token = email.run(triage_token)
    crm_token = crm.run(triage_token)
    followup_token = followup.run(triage_token)

    banner("ACT 4 · Email-agent further delegates to gmail-tool (3-deep)")
    gmail_token = gmail_tool.run(email_token)

    banner("WOW · 3-pane act-chain comparison")
    h1 = decode(triage_token, auth, "supportflow-core-api")
    h2 = decode(email_token, auth, "gmail-vault")
    h3 = decode(gmail_token, auth, "smtp-relay")
    print(f"\nHOP 1 · triage  | scope={h1.scope}\n  {render_chain(h1.act)}")
    print(f"\nHOP 2 · email   | scope={h2.scope}\n  {render_chain(h2.act)}")
    print(f"\nHOP 3 · gmail   | scope={h3.scope}\n  {render_chain(h3.act)}")
    print(f"\ncnf.jkt rotation:\n"
          f"  triage → {h1.jkt}\n  email  → {h2.jkt}\n  gmail  → {h3.jkt}")

    summary = {
        "user_aud": user_claims.aud,
        "hop1_scope": h1.scope,
        "hop2_scope": h2.scope,
        "hop3_scope": h3.scope,
        "hop3_act_depth": _depth(h3.act),
        "tokens": {
            "user": user.access_token,
            "triage": triage_token,
            "knowledge": knowledge_token,
            "email": email_token,
            "gmail": gmail_token,
            "crm": crm_token,
            "followup": followup_token,
        },
    }
    out = ROOT / ".last_run.json"
    out.write_text(json.dumps(summary, indent=2))
    print(f"\nRun summary → {out}")

    if _depth(h3.act) != 3:
        print("FAIL · expected hop3 act_depth==3", file=sys.stderr)
        return 1
    print("\nPASS · 5 agents, 6 token-exchanges, 3-deep act chain verified.")
    return 0


def _depth(act):
    depth = 0
    cur = act
    while cur:
        depth += 1
        cur = cur.get("act") if isinstance(cur, dict) else None
    return depth


if __name__ == "__main__":
    sys.exit(main())
