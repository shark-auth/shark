"""CRM agent — hop 2. Narrows scope to crm:write, audience salesforce-api.
Subject token must allow may_act=['crm-agent'] for full enforcement (see seed.sh)."""
from __future__ import annotations

import os

from lib import token_exchange, decode, render_token


AGENT_NAME = "crm-agent"
AUDIENCE = "salesforce-api"
SCOPE = "crm:write"


def run(triage_token: str) -> str:
    auth = os.environ["SHARK_AUTH_URL"]
    cid = os.environ["CRM_CLIENT_ID"]
    csec = os.environ["CRM_CLIENT_SECRET"]

    res = token_exchange(
        auth, cid, csec,
        subject_token=triage_token,
        scope=SCOPE,
        audience=AUDIENCE,
    )

    claims = decode(res.access_token, auth, AUDIENCE)
    print(render_token(claims, label="HOP 2 · crm-agent"))
    return res.access_token
