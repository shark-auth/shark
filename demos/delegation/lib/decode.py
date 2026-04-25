"""Decode + render Shark agent tokens with full RFC 8693 act chain visualization."""
from __future__ import annotations

import json
from typing import Any, Dict

from shark_auth import AgentTokenClaims, decode_agent_token


def decode(token: str, auth_url: str, expected_audience: str) -> AgentTokenClaims:
    return decode_agent_token(
        token,
        f"{auth_url.rstrip('/')}/.well-known/jwks.json",
        expected_issuer=auth_url.rstrip("/"),
        expected_audience=expected_audience,
        leeway=5,
    )


def _act_depth(act: Dict[str, Any] | None) -> int:
    depth = 0
    cur = act
    while cur:
        depth += 1
        cur = cur.get("act") if isinstance(cur, dict) else None
    return depth


def render_token(claims: AgentTokenClaims, *, label: str) -> str:
    payload = {
        "label": label,
        "sub": claims.sub,
        "scope": claims.scope,
        "aud": claims.aud,
        "act_depth": _act_depth(claims.act),
        "act": claims.act,
        "cnf_jkt": claims.jkt,
        "exp": claims.exp,
    }
    return json.dumps(payload, indent=2, default=str)


def render_chain(act: Dict[str, Any] | None) -> str:
    """Render the act chain as an indented tree."""
    if not act:
        return "(no delegation — direct token)"
    lines = []
    cur = act
    indent = 0
    while cur:
        sub = cur.get("sub", "?")
        prefix = "  " * indent + ("└─ " if indent else "")
        lines.append(f"{prefix}{sub}")
        cur = cur.get("act") if isinstance(cur, dict) else None
        indent += 1
    return "\n".join(lines)
