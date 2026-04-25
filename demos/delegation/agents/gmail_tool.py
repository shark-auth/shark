"""Gmail tool — hop 3. Receives email-agent's token, exchanges for email:send,
audience smtp-relay. Produces the 3-deep act chain."""
from __future__ import annotations

import os

from lib import token_exchange, decode, render_token


AGENT_NAME = "gmail-tool"
AUDIENCE = "smtp-relay"
SCOPE = "email:send"


def run(email_token: str) -> str:
    auth = os.environ["SHARK_AUTH_URL"]
    cid = os.environ["GMAIL_TOOL_CLIENT_ID"]
    csec = os.environ["GMAIL_TOOL_CLIENT_SECRET"]

    res = token_exchange(
        auth, cid, csec,
        subject_token=email_token,
        scope=SCOPE,
        audience=AUDIENCE,
    )

    claims = decode(res.access_token, auth, AUDIENCE)
    print(render_token(claims, label="HOP 3 · gmail-tool (3-deep act chain)"))
    return res.access_token
