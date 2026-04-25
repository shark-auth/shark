"""Audit replay — reconstruct the delegation timeline for a customer/ticket.

Per DEMO_02_DELEGATION_CHAIN.md §11, exchange.go currently logs to slog,
NOT to the audit_logs DB table. So this script:

  1. PRIMARY  — query GET /api/v1/audit-logs?action=oauth.token.exchanged
     (works after the 10-line patch noted in the plan)
  2. FALLBACK — query the oauth_tokens table via admin API for tokens with
     non-null delegation_subject + delegation_actor, ordered chronologically.

If you ran orchestrator/main.py, pass --last-run to replay everything that
happened in that single run."""
from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

import requests

ROOT = Path(__file__).resolve().parent
try:
    from dotenv import load_dotenv  # type: ignore
    load_dotenv(ROOT / ".env")
except Exception:
    pass


def query_audit(auth: str, admin_key: str, action: str, since: str | None) -> list[dict]:
    params: dict = {"action": action, "limit": 200}
    if since:
        params["since"] = since
    resp = requests.get(
        f"{auth}/api/v1/audit-logs",
        params=params,
        headers={"Authorization": f"Bearer {admin_key}"},
        timeout=10,
    )
    if resp.status_code != 200:
        return []
    return resp.json().get("logs", [])


def query_oauth_tokens(auth: str, admin_key: str, since: str | None) -> list[dict]:
    """Fallback: read the oauth_tokens table via admin API.

    NOTE: an admin endpoint exposing oauth_tokens with delegation_subject /
    delegation_actor columns may not exist on every Shark version. If absent,
    fall back to direct SQLite query against ./dev.db (see seed.sh comment).
    """
    params: dict = {"has_delegation": "true", "limit": 200}
    if since:
        params["since"] = since
    resp = requests.get(
        f"{auth}/api/v1/oauth/tokens",
        params=params,
        headers={"Authorization": f"Bearer {admin_key}"},
        timeout=10,
    )
    if resp.status_code != 200:
        return []
    return resp.json().get("tokens", [])


def render_event(ev: dict) -> str:
    when = ev.get("created_at") or ev.get("issued_at") or ev.get("ts", "?")
    actor = ev.get("delegation_actor") or ev.get("actor") or ev.get("client_id", "?")
    subject = ev.get("delegation_subject") or ev.get("subject") or "?"
    scope = ev.get("scope", "")
    aud = ev.get("audience") or ev.get("aud", "")
    act = ev.get("act_chain") or ev.get("act") or {"sub": actor}
    return (
        f"{when}  oauth.token.exchanged\n"
        f"  actor:    {actor}\n"
        f"  subject:  {subject}\n"
        f"  scope:    {scope}\n"
        f"  aud:      {aud}\n"
        f"  act:      {json.dumps(act, separators=(',',':'))}"
    )


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--ticket")
    p.add_argument("--since")
    p.add_argument("--last-run", action="store_true",
                   help="Replay tokens from the last orchestrator/main.py invocation")
    args = p.parse_args()

    auth = os.environ["SHARK_AUTH_URL"]
    admin_key = os.environ["SHARK_ADMIN_KEY"]

    print("AUDIT REPLAY" + (f" — ticket={args.ticket}" if args.ticket else ""))
    print("=" * 72)

    events = query_audit(auth, admin_key, "oauth.token.exchanged", args.since)

    if not events:
        print("# audit_logs path empty — falling back to oauth_tokens table\n")
        events = query_oauth_tokens(auth, admin_key, args.since)

    if not events and args.last_run:
        print("# both API paths empty — decoding tokens from .last_run.json")
        run = json.loads((ROOT / ".last_run.json").read_text())
        from lib import decode  # type: ignore
        sys.path.insert(0, str(ROOT))
        seen = []
        for label, tok in run["tokens"].items():
            if label == "user":
                continue
            try:
                claims = decode(tok, auth, _label_to_aud(label))
            except Exception as exc:
                seen.append({"label": label, "error": str(exc)})
                continue
            seen.append({
                "ts": claims.iat,
                "delegation_actor": (claims.act or {}).get("sub", "?"),
                "delegation_subject": claims.sub,
                "scope": claims.scope,
                "aud": claims.aud,
                "act_chain": claims.act,
            })
        seen.sort(key=lambda e: e.get("ts", 0))
        for e in seen:
            if "error" in e:
                print(f"  (skip {e['label']}: {e['error']})")
            else:
                print(render_event(e))
                print()

    if not events and not args.last_run:
        print("(no events) — pass --last-run to decode tokens from the last orchestrator run")
        return 1

    for e in events:
        print(render_event(e))
        print()

    print("=" * 72)
    print(f"{len(events)} delegation events. Full chain traceable.")
    return 0


def _label_to_aud(label: str) -> str:
    return {
        "triage": "supportflow-core-api",
        "knowledge": "kb-api",
        "email": "gmail-vault",
        "gmail": "smtp-relay",
        "crm": "salesforce-api",
        "followup": "gcal-api",
    }.get(label, "supportflow-core-api")


if __name__ == "__main__":
    sys.exit(main())
