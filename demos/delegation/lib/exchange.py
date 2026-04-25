"""RFC 8693 token-exchange wrapper.

The `shark_auth.exchange_token()` SDK helper does not expose the `audience`
form parameter (RFC 8707). This wrapper calls /oauth/token directly so each
hop binds a specific audience.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

import requests

GRANT_EXCHANGE = "urn:ietf:params:oauth:grant-type:token-exchange"
TOKEN_TYPE_ACCESS = "urn:ietf:params:oauth:token-type:access_token"


@dataclass
class TokenResult:
    access_token: str
    expires_in: int
    scope: str
    token_type: str
    raw: dict


def client_credentials(
    auth_url: str,
    client_id: str,
    client_secret: str,
    *,
    scope: str,
    audience: Optional[str] = None,
) -> TokenResult:
    data = {"grant_type": "client_credentials", "scope": scope}
    if audience:
        data["audience"] = audience
        data["resource"] = audience
    resp = requests.post(
        f"{auth_url.rstrip('/')}/oauth/token",
        data=data,
        auth=(client_id, client_secret),
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        timeout=10,
    )
    if resp.status_code != 200:
        raise RuntimeError(f"client_credentials failed: HTTP {resp.status_code}: {resp.text}")
    payload = resp.json()
    return TokenResult(
        access_token=payload["access_token"],
        expires_in=int(payload.get("expires_in", 0)),
        scope=payload.get("scope", scope),
        token_type=payload.get("token_type", "Bearer"),
        raw=payload,
    )


def token_exchange(
    auth_url: str,
    acting_client_id: str,
    acting_client_secret: str,
    *,
    subject_token: str,
    scope: str,
    audience: str,
) -> TokenResult:
    data = {
        "grant_type": GRANT_EXCHANGE,
        "subject_token": subject_token,
        "subject_token_type": TOKEN_TYPE_ACCESS,
        "requested_token_type": TOKEN_TYPE_ACCESS,
        "scope": scope,
        "audience": audience,
        "resource": audience,
    }
    resp = requests.post(
        f"{auth_url.rstrip('/')}/oauth/token",
        data=data,
        auth=(acting_client_id, acting_client_secret),
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        timeout=10,
    )
    if resp.status_code != 200:
        raise RuntimeError(
            f"token_exchange failed for actor={acting_client_id} aud={audience}: "
            f"HTTP {resp.status_code}: {resp.text}"
        )
    payload = resp.json()
    return TokenResult(
        access_token=payload["access_token"],
        expires_in=int(payload.get("expires_in", 0)),
        scope=payload.get("scope", scope),
        token_type=payload.get("token_type", "Bearer"),
        raw=payload,
    )
