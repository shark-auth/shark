"""
Gemma Worker — Demo Agent for SharkAuth Token Vault Demo 04
Reads Gmail / Slack / GitHub / Notion / Linear on behalf of user_42
via Shark's Token Vault. Zero raw credentials in agent memory.

Usage:
    SHARK_URL=http://localhost:8000 \
    AGENT_CLIENT_ID=gemma-worker \
    AGENT_CLIENT_SECRET=demo-secret \
    python agent.py
"""

import os
import sys
import json
import time
import uuid
import httpx

SHARK_URL = os.environ.get("SHARK_URL", "http://localhost:8000")
CLIENT_ID = os.environ.get("AGENT_CLIENT_ID", "gemma-worker")
CLIENT_SECRET = os.environ.get("AGENT_CLIENT_SECRET", "demo-secret")
USER_ID = os.environ.get("DEMO_USER_ID", "user_42")

PROVIDERS = [
    "google_gmail",
    "slack",
    "github",
    "notion",
    "linear",
]


def get_shark_token() -> str:
    """Authenticate to Shark via client_credentials. Returns agent JWT."""
    resp = httpx.post(
        f"{SHARK_URL}/oauth/token",
        data={
            "grant_type": "client_credentials",
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
            "scope": "vault:read",
        },
        timeout=10,
    )
    resp.raise_for_status()
    token = resp.json().get("access_token")
    if not token:
        print("[ERROR] No access_token in Shark response", file=sys.stderr)
        sys.exit(1)
    print(f"[AUTH] Got Shark JWT (scope=vault:read) — {token[:40]}...")
    return token


def read_vault_token(shark_jwt: str, provider: str, user_id: str) -> dict:
    """Read a fresh provider token from Shark's vault. Never sees refresh token."""
    request_id = f"req_{uuid.uuid4().hex[:12]}"
    resp = httpx.get(
        f"{SHARK_URL}/api/v1/vault/{provider}/token",
        headers={
            "Authorization": f"Bearer {shark_jwt}",
            "X-User-ID": user_id,
            "X-Request-ID": request_id,
        },
        timeout=10,
    )
    if resp.status_code == 403:
        body = resp.json()
        print(f"[BLOCKED] provider={provider} user={user_id} error={body.get('error')} "
              f"(www-auth: {resp.headers.get('WWW-Authenticate', 'n/a')})")
        return {}
    if resp.status_code == 404:
        print(f"[NO-CONNECTION] provider={provider} user={user_id} — user hasn't connected yet")
        return {}
    resp.raise_for_status()
    data = resp.json()
    print(f"[VAULT-READ] provider={provider} user={user_id} "
          f"expires_at={data.get('expires_at')} "
          f"scopes={data.get('scopes')} "
          f"request_id={request_id}")
    return data


def call_gmail(token: str) -> None:
    """Call Gmail API with the Shark-vended token. Agent never touches refresh token."""
    if not token:
        return
    resp = httpx.get(
        "https://gmail.googleapis.com/gmail/v1/users/me/messages",
        params={"maxResults": 3},
        headers={"Authorization": f"Bearer {token}"},
        timeout=10,
    )
    if resp.status_code == 200:
        msgs = resp.json().get("messages", [])
        print(f"  [Gmail] {len(msgs)} messages in inbox (first id: {msgs[0]['id'] if msgs else 'none'})")
    else:
        print(f"  [Gmail] HTTP {resp.status_code} — {resp.text[:120]}")


def call_slack(token: str) -> None:
    if not token:
        return
    resp = httpx.get(
        "https://slack.com/api/conversations.list",
        headers={"Authorization": f"Bearer {token}"},
        params={"limit": 3},
        timeout=10,
    )
    if resp.status_code == 200 and resp.json().get("ok"):
        channels = [c["name"] for c in resp.json().get("channels", [])]
        print(f"  [Slack] channels: {channels}")
    else:
        print(f"  [Slack] {resp.status_code} — {resp.text[:120]}")


def call_github(token: str) -> None:
    if not token:
        return
    resp = httpx.get(
        "https://api.github.com/user/repos",
        headers={"Authorization": f"Bearer {token}", "X-GitHub-Api-Version": "2022-11-28"},
        params={"per_page": 3},
        timeout=10,
    )
    if resp.status_code == 200:
        repos = [r["full_name"] for r in resp.json()]
        print(f"  [GitHub] repos: {repos}")
    else:
        print(f"  [GitHub] {resp.status_code} — {resp.text[:120]}")


def run_once(shark_jwt: str) -> None:
    print(f"\n{'='*60}")
    print(f"  GEMMA WORKER — vault read cycle for user={USER_ID}")
    print(f"{'='*60}")
    for provider in PROVIDERS:
        vault_data = read_vault_token(shark_jwt, provider, USER_ID)
        access_token = vault_data.get("access_token", "")
        if provider == "google_gmail":
            call_gmail(access_token)
        elif provider == "slack":
            call_slack(access_token)
        elif provider == "github":
            call_github(access_token)
        else:
            if access_token:
                print(f"  [{provider}] token available — {access_token[:30]}...")
        time.sleep(0.2)


def main() -> None:
    print("[GEMMA-WORKER] Starting. Authenticating to Shark...")
    shark_jwt = get_shark_token()
    cycles = int(os.environ.get("DEMO_CYCLES", "1"))
    for i in range(cycles):
        if i > 0:
            print(f"\n[CYCLE {i+1}/{cycles}] Sleeping 5s...")
            time.sleep(5)
        run_once(shark_jwt)
    print("\n[GEMMA-WORKER] Done.")


if __name__ == "__main__":
    main()
