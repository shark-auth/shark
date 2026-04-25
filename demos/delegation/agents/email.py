"""Email agent — hop 2. Narrows to email:draft. Audience: gmail-vault."""
from __future__ import annotations

import os

from lib import token_exchange, decode, render_token


AGENT_NAME = "email-agent"
AUDIENCE = "gmail-vault"
SCOPE = "email:draft"


def run(triage_token: str) -> str:
    auth = os.environ["SHARK_AUTH_URL"]
    cid = os.environ["EMAIL_CLIENT_ID"]
    csec = os.environ["EMAIL_CLIENT_SECRET"]

    res = token_exchange(
        auth, cid, csec,
        subject_token=triage_token,
        scope=SCOPE,
        audience=AUDIENCE,
    )

    claims = decode(res.access_token, auth, AUDIENCE)
    print(render_token(claims, label="HOP 2 · email-agent"))
    return res.access_token
