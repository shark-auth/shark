"""Knowledge agent — hop 2. Narrows scope to kb:read, audience kb-api."""
from __future__ import annotations

import os

from lib import token_exchange, decode, render_token


AGENT_NAME = "knowledge-agent"
AUDIENCE = "kb-api"
SCOPE = "kb:read"


def run(triage_token: str) -> str:
    auth = os.environ["SHARK_AUTH_URL"]
    cid = os.environ["KNOWLEDGE_CLIENT_ID"]
    csec = os.environ["KNOWLEDGE_CLIENT_SECRET"]

    res = token_exchange(
        auth, cid, csec,
        subject_token=triage_token,
        scope=SCOPE,
        audience=AUDIENCE,
    )

    claims = decode(res.access_token, auth, AUDIENCE)
    print(render_token(claims, label="HOP 2 · knowledge-agent"))
    return res.access_token
