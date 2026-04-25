"""Followup agent — hop 2. Narrows scope to calendar:write, audience gcal-api."""
from __future__ import annotations

import os

from lib import token_exchange, decode, render_token


AGENT_NAME = "followup-agent"
AUDIENCE = "gcal-api"
SCOPE = "calendar:write"


def run(triage_token: str) -> str:
    auth = os.environ["SHARK_AUTH_URL"]
    cid = os.environ["FOLLOWUP_CLIENT_ID"]
    csec = os.environ["FOLLOWUP_CLIENT_SECRET"]

    res = token_exchange(
        auth, cid, csec,
        subject_token=triage_token,
        scope=SCOPE,
        audience=AUDIENCE,
    )

    claims = decode(res.access_token, auth, AUDIENCE)
    print(render_token(claims, label="HOP 2 · followup-agent"))
    return res.access_token
