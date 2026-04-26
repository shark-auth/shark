"""Smoke — Method 4: AgentTokenClaims.delegation_chain() — pure unit, no shark needed.

This file intentionally does NOT use the server fixtures so it runs in any
environment where the SDK is importable.
"""

from __future__ import annotations

import base64
import json

import pytest

from shark_auth.claims import ActorClaim, AgentTokenClaims


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _b64url(data: dict) -> str:
    raw = json.dumps(data).encode()
    return base64.urlsafe_b64encode(raw).rstrip(b"=").decode()


def _make_jwt(payload: dict) -> str:
    header = _b64url({"alg": "RS256", "typ": "JWT"})
    body = _b64url(payload)
    return f"{header}.{body}.fakesig"


# ---------------------------------------------------------------------------
# 0 hops
# ---------------------------------------------------------------------------

def test_method4_zero_hops():
    jwt = _make_jwt({
        "sub": "agent_direct", "iss": "https://auth.example.com",
        "aud": "api", "exp": 9999999999, "iat": 1700000000,
        "scope": "mcp:read",
    })
    claims = AgentTokenClaims.parse(jwt)
    assert claims.delegation_chain() == []
    assert claims.is_delegated() is False
    assert claims.has_scope("mcp:read") is True


# ---------------------------------------------------------------------------
# 1 hop
# ---------------------------------------------------------------------------

def test_method4_one_hop():
    jwt = _make_jwt({
        "sub": "agent_subj", "iss": "https://auth.example.com",
        "aud": "api", "exp": 9999999999, "iat": 1700000000,
        "scope": "mcp:read",
        "act": {
            "sub": "agent_actor1", "iat": 1700000001,
            "scope": "mcp:write", "cnf": {"jkt": "jkt-abc"},
        },
    })
    claims = AgentTokenClaims.parse(jwt)
    chain = claims.delegation_chain()
    assert len(chain) == 1
    assert claims.is_delegated() is True
    assert chain[0].sub == "agent_actor1"
    assert chain[0].scope == "mcp:write"
    assert chain[0].jkt == "jkt-abc"
    assert chain[0].iat == 1700000001


# ---------------------------------------------------------------------------
# 2 hops
# ---------------------------------------------------------------------------

def test_method4_two_hops():
    jwt = _make_jwt({
        "sub": "agent_subj", "iss": "https://auth.example.com",
        "aud": "api", "exp": 9999999999, "iat": 1700000000,
        "act": {
            "sub": "agent_actor1", "iat": 1700000001,
            "act": {
                "sub": "agent_actor2", "iat": 1700000002,
                "cnf": {"jkt": "jkt-inner"},
            },
        },
    })
    claims = AgentTokenClaims.parse(jwt)
    chain = claims.delegation_chain()
    assert len(chain) == 2
    assert chain[0].sub == "agent_actor1"
    assert chain[1].sub == "agent_actor2"
    assert chain[1].jkt == "jkt-inner"


# ---------------------------------------------------------------------------
# 3 hops
# ---------------------------------------------------------------------------

def test_method4_three_hops():
    jwt = _make_jwt({
        "sub": "agent_subj", "iss": "https://auth.example.com",
        "aud": "api", "exp": 9999999999, "iat": 1700000000,
        "act": {
            "sub": "a1", "iat": 1,
            "act": {
                "sub": "a2", "iat": 2,
                "act": {"sub": "a3", "iat": 3},
            },
        },
    })
    claims = AgentTokenClaims.parse(jwt)
    chain = claims.delegation_chain()
    assert len(chain) == 3
    assert [h.sub for h in chain] == ["a1", "a2", "a3"]


# ---------------------------------------------------------------------------
# has_scope edge cases
# ---------------------------------------------------------------------------

def test_method4_has_scope_absent():
    jwt = _make_jwt({
        "sub": "s", "iss": "i", "aud": "a",
        "exp": 9999999999, "iat": 1700000000,
        "scope": "mcp:read",
    })
    claims = AgentTokenClaims.parse(jwt)
    assert claims.has_scope("admin") is False


def test_method4_no_scope_field():
    jwt = _make_jwt({
        "sub": "s", "iss": "i", "aud": "a",
        "exp": 9999999999, "iat": 1700000000,
    })
    claims = AgentTokenClaims.parse(jwt)
    assert claims.has_scope("anything") is False


# ---------------------------------------------------------------------------
# Malformed
# ---------------------------------------------------------------------------

def test_method4_malformed_jwt_raises():
    with pytest.raises(ValueError):
        AgentTokenClaims.parse("not.a.valid.jwt.parts")
