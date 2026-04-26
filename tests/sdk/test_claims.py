"""Unit tests for Method 4 — AgentTokenClaims.delegation_chain().

Pure unit tests — no shark server required. Fixtures build JWTs by hand
(base64url-encoding a crafted payload) since we skip signature verification.
"""

from __future__ import annotations

import base64
import json

import pytest

from shark_auth.claims import ActorClaim, AgentTokenClaims, _decode_payload


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _b64url(data: dict) -> str:
    raw = json.dumps(data).encode()
    return base64.urlsafe_b64encode(raw).rstrip(b"=").decode()


def _make_jwt(payload: dict) -> str:
    """Build a minimal unsigned JWT (header.payload.sig) for testing."""
    header = _b64url({"alg": "RS256", "typ": "JWT", "kid": "test-key"})
    body = _b64url(payload)
    return f"{header}.{body}.fakesig"


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture
def jwt_no_act():
    """Direct token — no act claim (0 hops)."""
    payload = {
        "sub": "shark_agent_direct",
        "iss": "https://auth.example.com",
        "aud": "https://api.example.com",
        "exp": 9999999999,
        "iat": 1700000000,
        "scope": "mcp:read mcp:write",
    }
    return _make_jwt(payload)


@pytest.fixture
def jwt_one_hop():
    """Delegated token — 1 act hop."""
    payload = {
        "sub": "shark_agent_subject",
        "iss": "https://auth.example.com",
        "aud": "https://api.example.com",
        "exp": 9999999999,
        "iat": 1700000000,
        "scope": "mcp:read",
        "act": {
            "sub": "shark_agent_actor1",
            "iat": 1700000001,
            "scope": "mcp:write",
            "cnf": {"jkt": "thumbprint-a1"},
        },
    }
    return _make_jwt(payload)


@pytest.fixture
def jwt_two_hops():
    """Doubly-delegated token — 2 act hops."""
    payload = {
        "sub": "shark_agent_subject",
        "iss": "https://auth.example.com",
        "aud": "https://api.example.com",
        "exp": 9999999999,
        "iat": 1700000000,
        "scope": "mcp:read",
        "act": {
            "sub": "shark_agent_actor1",
            "iat": 1700000001,
            "scope": "mcp:write",
            "act": {
                "sub": "shark_agent_actor2",
                "iat": 1700000002,
                "scope": "mcp:admin",
                "cnf": {"jkt": "thumbprint-a2"},
            },
        },
    }
    return _make_jwt(payload)


@pytest.fixture
def jwt_three_hops():
    """Triple-delegated token — 3 act hops."""
    payload = {
        "sub": "shark_agent_subject",
        "iss": "https://auth.example.com",
        "aud": "https://api.example.com",
        "exp": 9999999999,
        "iat": 1700000000,
        "scope": "mcp:read",
        "act": {
            "sub": "shark_agent_actor1",
            "iat": 1700000001,
            "act": {
                "sub": "shark_agent_actor2",
                "iat": 1700000002,
                "act": {
                    "sub": "shark_agent_actor3",
                    "iat": 1700000003,
                    "cnf": {"jkt": "thumbprint-a3"},
                },
            },
        },
    }
    return _make_jwt(payload)


# ---------------------------------------------------------------------------
# Tests — delegation_chain()
# ---------------------------------------------------------------------------

class TestDelegationChainLength:
    def test_zero_hops(self, jwt_no_act):
        claims = AgentTokenClaims.parse(jwt_no_act)
        assert claims.delegation_chain() == []

    def test_one_hop(self, jwt_one_hop):
        claims = AgentTokenClaims.parse(jwt_one_hop)
        chain = claims.delegation_chain()
        assert len(chain) == 1

    def test_two_hops(self, jwt_two_hops):
        claims = AgentTokenClaims.parse(jwt_two_hops)
        chain = claims.delegation_chain()
        assert len(chain) == 2

    def test_three_hops(self, jwt_three_hops):
        claims = AgentTokenClaims.parse(jwt_three_hops)
        chain = claims.delegation_chain()
        assert len(chain) == 3


class TestDelegationChainOrdering:
    def test_outermost_first(self, jwt_two_hops):
        """First element is the outermost (most recent) actor."""
        claims = AgentTokenClaims.parse(jwt_two_hops)
        chain = claims.delegation_chain()
        assert chain[0].sub == "shark_agent_actor1"
        assert chain[1].sub == "shark_agent_actor2"

    def test_three_hop_ordering(self, jwt_three_hops):
        claims = AgentTokenClaims.parse(jwt_three_hops)
        chain = claims.delegation_chain()
        assert chain[0].sub == "shark_agent_actor1"
        assert chain[1].sub == "shark_agent_actor2"
        assert chain[2].sub == "shark_agent_actor3"


class TestActorClaimFields:
    def test_one_hop_fields(self, jwt_one_hop):
        claims = AgentTokenClaims.parse(jwt_one_hop)
        hop = claims.delegation_chain()[0]
        assert hop.sub == "shark_agent_actor1"
        assert hop.iat == 1700000001
        assert hop.scope == "mcp:write"
        assert hop.jkt == "thumbprint-a1"

    def test_jkt_from_innermost_hop(self, jwt_three_hops):
        claims = AgentTokenClaims.parse(jwt_three_hops)
        chain = claims.delegation_chain()
        assert chain[2].jkt == "thumbprint-a3"

    def test_none_jkt_when_absent(self, jwt_one_hop):
        """Hops without cnf.jkt should have jkt=None."""
        claims = AgentTokenClaims.parse(jwt_one_hop)
        # jwt_one_hop has jkt on hop 0 (actor1)
        assert claims.delegation_chain()[0].jkt == "thumbprint-a1"


# ---------------------------------------------------------------------------
# Tests — is_delegated()
# ---------------------------------------------------------------------------

class TestIsDelegated:
    def test_direct_token_not_delegated(self, jwt_no_act):
        claims = AgentTokenClaims.parse(jwt_no_act)
        assert claims.is_delegated() is False

    def test_one_hop_is_delegated(self, jwt_one_hop):
        claims = AgentTokenClaims.parse(jwt_one_hop)
        assert claims.is_delegated() is True

    def test_three_hop_is_delegated(self, jwt_three_hops):
        claims = AgentTokenClaims.parse(jwt_three_hops)
        assert claims.is_delegated() is True


# ---------------------------------------------------------------------------
# Tests — has_scope()
# ---------------------------------------------------------------------------

class TestHasScope:
    def test_present_scope(self, jwt_no_act):
        claims = AgentTokenClaims.parse(jwt_no_act)
        assert claims.has_scope("mcp:read") is True
        assert claims.has_scope("mcp:write") is True

    def test_absent_scope(self, jwt_no_act):
        claims = AgentTokenClaims.parse(jwt_no_act)
        assert claims.has_scope("admin:all") is False

    def test_no_scope_field(self):
        payload = {
            "sub": "s", "iss": "i", "aud": "a",
            "exp": 9999999999, "iat": 1700000000,
        }
        jwt = _make_jwt(payload)
        claims = AgentTokenClaims.parse(jwt)
        assert claims.has_scope("mcp:read") is False


# ---------------------------------------------------------------------------
# Tests — properties
# ---------------------------------------------------------------------------

class TestProperties:
    def test_sub(self, jwt_no_act):
        claims = AgentTokenClaims.parse(jwt_no_act)
        assert claims.sub == "shark_agent_direct"

    def test_jkt_from_cnf(self):
        payload = {
            "sub": "s", "iss": "i", "aud": "a",
            "exp": 9999999999, "iat": 1700000000,
            "cnf": {"jkt": "top-level-jkt"},
        }
        claims = AgentTokenClaims.parse(_make_jwt(payload))
        assert claims.jkt == "top-level-jkt"

    def test_jkt_absent_when_no_cnf(self, jwt_no_act):
        claims = AgentTokenClaims.parse(jwt_no_act)
        assert claims.jkt is None

    def test_raw_payload_accessible(self, jwt_no_act):
        claims = AgentTokenClaims.parse(jwt_no_act)
        assert "sub" in claims.raw


# ---------------------------------------------------------------------------
# Tests — malformed input
# ---------------------------------------------------------------------------

class TestMalformedInput:
    def test_not_three_parts_raises(self):
        with pytest.raises(ValueError, match="3 parts"):
            AgentTokenClaims.parse("only.two")

    def test_invalid_base64_raises(self):
        with pytest.raises(Exception):
            AgentTokenClaims.parse("hdr.!!!.sig")
