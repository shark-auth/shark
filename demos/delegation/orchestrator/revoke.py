"""Surgical revocation: revoke ONE agent + verify blast radius.

Calls DELETE /api/v1/agents/{agent_id} which sets active=false AND
revokes all outstanding tokens for that client_id.

Then introspects every saved token from .last_run.json and prints which
went inactive vs which still pass."""
from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

import requests

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

try:
    from dotenv import load_dotenv  # type: ignore
    load_dotenv(ROOT / ".env")
except Exception:
    pass

AGENT_KEY_BY_NAME = {
    "triage-agent": ("TRIAGE_CLIENT_ID", "triage"),
    "knowledge-agent": ("KNOWLEDGE_CLIENT_ID", "knowledge"),
    "email-agent": ("EMAIL_CLIENT_ID", "email"),
    "gmail-tool": ("GMAIL_TOOL_CLIENT_ID", "gmail"),
    "crm-agent": ("CRM_CLIENT_ID", "crm"),
    "followup-agent": ("FOLLOWUP_CLIENT_ID", "followup"),
}


def introspect(auth: str, platform_id: str, platform_secret: str, token: str) -> bool:
    resp = requests.post(
        f"{auth}/oauth/introspect",
        data={"token": token},
        auth=(platform_id, platform_secret),
        timeout=10,
    )
    if resp.status_code != 200:
        return False
    return bool(resp.json().get("active"))


def find_agent_id(auth: str, admin_key: str, client_id: str) -> str:
    resp = requests.get(
        f"{auth}/api/v1/agents",
        params={"search": client_id, "limit": 50},
        headers={"Authorization": f"Bearer {admin_key}"},
        timeout=10,
    )
    resp.raise_for_status()
    for a in resp.json().get("agents", []):
        if a.get("client_id") == client_id:
            return a["id"]
    raise RuntimeError(f"agent not found for client_id={client_id}")


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("agent", choices=AGENT_KEY_BY_NAME.keys())
    p.add_argument("--reason", default="key_compromise")
    args = p.parse_args()

    auth = os.environ["SHARK_AUTH_URL"]
    admin_key = os.environ["SHARK_ADMIN_KEY"]
    platform_id = os.environ["PLATFORM_CLIENT_ID"]
    platform_secret = os.environ["PLATFORM_CLIENT_SECRET"]

    env_var, run_key = AGENT_KEY_BY_NAME[args.agent]
    target_client_id = os.environ[env_var]

    run = json.loads((ROOT / ".last_run.json").read_text())
    tokens = run["tokens"]

    print(f"BEFORE · introspect every token issued in last run")
    before = {k: introspect(auth, platform_id, platform_secret, t) for k, t in tokens.items()}
    for k, alive in before.items():
        print(f"  {k:10} active={alive}")

    print(f"\nREVOKE · {args.agent}  (reason={args.reason})")
    aid = find_agent_id(auth, admin_key, target_client_id)
    resp = requests.delete(
        f"{auth}/api/v1/agents/{aid}",
        headers={"Authorization": f"Bearer {admin_key}"},
        timeout=10,
    )
    if resp.status_code not in (200, 204):
        print(f"FAIL · DELETE returned {resp.status_code}: {resp.text}", file=sys.stderr)
        return 2
    print(f"  agent_id={aid} revoked. all outstanding tokens for {target_client_id} invalidated.")

    print(f"\nAFTER · re-introspect")
    after = {k: introspect(auth, platform_id, platform_secret, t) for k, t in tokens.items()}
    for k, alive in after.items():
        flipped = " ← REVOKED" if before[k] and not alive else (
            " (still alive)" if alive else " (was already inactive)"
        )
        print(f"  {k:10} active={alive}{flipped}")

    blast = [k for k, was in before.items() if was and not after[k]]
    survived = [k for k, alive in after.items() if alive]
    print(f"\nBLAST RADIUS · revoked={blast}")
    print(f"SURVIVED     · {survived}")
    if run_key not in blast:
        print(f"FAIL · expected {run_key} in blast radius", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
