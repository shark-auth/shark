"""Triage agent — hop 1. Receives the user/platform token, exchanges for its own
delegated token bound to the supportflow-core-api audience."""
from __future__ import annotations

import os

from lib import token_exchange, decode, render_token


AGENT_NAME = "triage-agent"
AUDIENCE = "supportflow-core-api"
SCOPE = "ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write"


def run(subject_token: str) -> str:
    auth = os.environ["SHARK_AUTH_URL"]
    cid = os.environ["TRIAGE_CLIENT_ID"]
    csec = os.environ["TRIAGE_CLIENT_SECRET"]

    res = token_exchange(
        auth, cid, csec,
        subject_token=subject_token,
        scope=SCOPE,
        audience=AUDIENCE,
    )

    claims = decode(res.access_token, auth, AUDIENCE)
    print(render_token(claims, label="HOP 1 · triage-agent"))
    return res.access_token
