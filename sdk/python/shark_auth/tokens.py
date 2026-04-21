"""Verify Shark-issued agent access tokens via JWKS."""

from __future__ import annotations

import threading
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Union

import jwt
from jwt import PyJWKClient, PyJWKSet

from . import _http
from .errors import TokenError


@dataclass
class AgentTokenClaims:
    """Parsed claims from a Shark agent access token.

    Preserves all claims — including RFC 8693 ``act`` (actor),
    RFC 9449 ``cnf`` (confirmation, e.g. ``cnf.jkt``), and
    RFC 9396 ``authorization_details``.
    """

    sub: str
    iss: str
    aud: Union[str, List[str]]
    exp: int
    iat: int
    scope: Optional[str] = None
    agent_id: Optional[str] = None
    act: Optional[Dict[str, Any]] = None
    cnf: Optional[Dict[str, Any]] = None
    authorization_details: Optional[List[Dict[str, Any]]] = None
    raw: Dict[str, Any] = field(default_factory=dict)

    @property
    def jkt(self) -> Optional[str]:
        """The DPoP JWK thumbprint bound to this token, if any."""
        if self.cnf and isinstance(self.cnf, dict):
            return self.cnf.get("jkt")
        return None


# ---------------------------------------------------------------------------
# JWKS cache — module-level, thread-safe
# ---------------------------------------------------------------------------
_jwks_cache: Dict[str, Dict[str, Any]] = {}
_jwks_lock = threading.Lock()


def _fetch_jwks(jwks_url: str) -> Dict[str, Any]:
    resp = _http.request(_http.new_session(), "GET", jwks_url)
    if resp.status_code != 200:
        raise TokenError(
            f"failed to fetch JWKS: HTTP {resp.status_code}: {resp.text[:200]}"
        )
    return resp.json()


def _get_signing_key(jwks_url: str, kid: str, *, allow_refresh: bool = True):
    with _jwks_lock:
        entry = _jwks_cache.get(jwks_url)
    if entry is None:
        jwks = _fetch_jwks(jwks_url)
        with _jwks_lock:
            _jwks_cache[jwks_url] = jwks
        entry = jwks

    try:
        jwk_set = PyJWKSet.from_dict(entry)
        for key in jwk_set.keys:
            if key.key_id == kid:
                return key.key
    except Exception:
        pass

    if allow_refresh:
        # kid miss: single retry after forced refresh
        jwks = _fetch_jwks(jwks_url)
        with _jwks_lock:
            _jwks_cache[jwks_url] = jwks
        return _get_signing_key(jwks_url, kid, allow_refresh=False)

    raise TokenError(f"no JWK found for kid={kid!r} at {jwks_url}")


def clear_jwks_cache() -> None:
    """Clear the module-level JWKS cache (testing hook)."""
    with _jwks_lock:
        _jwks_cache.clear()


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------
def decode_agent_token(
    token: str,
    jwks_url: str,
    *,
    expected_issuer: str,
    expected_audience: Union[str, List[str]],
    leeway: int = 0,
) -> AgentTokenClaims:
    """Decode + verify a Shark-issued agent access token.

    Verifies signature via JWKS (with kid-miss refresh), ``exp``/``nbf`` via
    PyJWT, and ``iss`` / ``aud`` against the provided expected values.

    Raises
    ------
    TokenError
        On invalid signature, expired token, wrong issuer or audience,
        missing JWK, or malformed token.
    """
    try:
        header = jwt.get_unverified_header(token)
    except jwt.PyJWTError as exc:
        raise TokenError(f"malformed token: {exc}") from exc

    kid = header.get("kid")
    if not kid:
        raise TokenError("token header missing kid")
    alg = header.get("alg")
    if not alg or alg.lower() == "none":
        raise TokenError(f"unsupported alg: {alg!r}")

    key = _get_signing_key(jwks_url, kid)

    try:
        claims = jwt.decode(
            token,
            key=key,
            algorithms=[alg],
            issuer=expected_issuer,
            audience=expected_audience,
            leeway=leeway,
            options={"require": ["exp", "iat", "iss", "aud", "sub"]},
        )
    except jwt.ExpiredSignatureError as exc:
        raise TokenError("token expired") from exc
    except jwt.InvalidIssuerError as exc:
        raise TokenError(f"invalid issuer: {exc}") from exc
    except jwt.InvalidAudienceError as exc:
        raise TokenError(f"invalid audience: {exc}") from exc
    except jwt.InvalidSignatureError as exc:
        raise TokenError("invalid signature") from exc
    except jwt.PyJWTError as exc:
        raise TokenError(f"token validation failed: {exc}") from exc

    return AgentTokenClaims(
        sub=claims["sub"],
        iss=claims["iss"],
        aud=claims["aud"],
        exp=int(claims["exp"]),
        iat=int(claims["iat"]),
        scope=claims.get("scope"),
        agent_id=claims.get("agent_id"),
        act=claims.get("act"),
        cnf=claims.get("cnf"),
        authorization_details=claims.get("authorization_details"),
        raw=claims,
    )
