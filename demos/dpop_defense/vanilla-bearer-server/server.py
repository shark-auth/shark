"""
Demo 05 — Contrast: Vanilla Bearer-only mock server (Auth0-style).

This is the VULNERABLE comparison.  It accepts any valid Bearer JWT without
proof-of-possession.  All 6 attacks succeed in under 10 seconds.

Run:
    uvicorn server:app --port 9001

Then point the attacker scripts at API_URL=http://localhost:9001 and watch
every attack succeed — no defense, no audit log, no webhook.
"""
from __future__ import annotations
import os
import jwt as pyjwt
from fastapi import FastAPI, Header, HTTPException
from fastapi.responses import JSONResponse

app = FastAPI(title="Vulnerable Bearer Server (Demo contrast)")

# In a real Auth0 deployment you'd fetch JWKS. For demo we accept any JWT
# with valid structure (no signature verification needed to show the flaw —
# in production Auth0 servers DO verify sigs but do NOT check key binding).
SKIP_SIG_VERIFY = os.getenv("SKIP_SIG", "true").lower() == "true"

def _extract_bearer(authorization: str | None) -> str:
    if not authorization or not authorization.startswith(("Bearer ", "DPoP ")):
        raise HTTPException(status_code=401, detail="missing Authorization header")
    return authorization.split(" ", 1)[1]

@app.get("/api/positions")
def get_positions(authorization: str | None = Header(default=None)):
    token = _extract_bearer(authorization)
    # *** NO DPoP check. NO key-binding check. Bearer token = full access. ***
    try:
        if SKIP_SIG_VERIFY:
            payload = pyjwt.decode(token, options={"verify_signature": False})
        else:
            payload = {"sub": "unknown"}
    except Exception:
        payload = {"sub": "unknown"}
    return JSONResponse({
        "positions": [
            {"symbol": "BTC-USD",  "qty": 100,  "value_usd": 6_500_000},
            {"symbol": "ETH-USD",  "qty": 500,  "value_usd": 1_250_000},
            {"symbol": "AAPL",     "qty": 10000,"value_usd": 1_750_000},
        ],
        "total_value_usd": 9_500_000,
        "WARNING": "BEARER-ONLY SERVER — token theft = full access",
    })

@app.get("/api/withdraw/{amount}")
def withdraw(amount: str, authorization: str | None = Header(default=None)):
    token = _extract_bearer(authorization)
    # *** No DPoP. No key binding. Stolen bearer token = money gone. ***
    return JSONResponse({
        "status": "TRANSFER_INITIATED",
        "amount": amount,
        "WARNING": "BEARER-ONLY — attacker just moved your money",
    })

@app.post("/api/trade")
def place_trade(authorization: str | None = Header(default=None)):
    _extract_bearer(authorization)
    return JSONResponse({"status": "ORDER_PLACED", "WARNING": "BEARER-ONLY"})

# Contrast summary endpoint
@app.get("/")
def root():
    return {
        "server": "vanilla-bearer (vulnerable)",
        "dpop": False,
        "cnf_jkt": False,
        "jti_replay_cache": False,
        "ath_binding": False,
        "attacks_that_succeed": 6,
        "attacks_that_fail": 0,
    }
