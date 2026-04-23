"""Verify Shark-issued agent access tokens via JWKS."""

from __future__ import annotations

import threading
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Union

import jwt
from jwt import PyJWKClient, PyJWKSet

from . import _http
from .device_flow import TokenResponse
from .dpop import DPoPProver
from .errors import TokenError

TOKEN_EXCHANGE_GRANT = "urn:ietf:params:oauth:grant-type:token-exchange"
ACCESS_TOKEN_TYPE = "urn:ietf:params:oauth:token-type:access_token"


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


def exchange_token(
    auth_url: str,
    client_id: str,
    subject_token: str,
    *,
    client_secret: Optional[str] = None,
    subject_token_type: str = ACCESS_TOKEN_TYPE,
    actor_token: Optional[str] = None,
    actor_token_type: str = ACCESS_TOKEN_TYPE,
    scope: Optional[str] = None,
    requested_token_type: str = ACCESS_TOKEN_TYPE,
    dpop_prover: Optional[DPoPProver] = None,
    token_path: str = "/oauth/token",
    session: Optional[object] = None,
) -> TokenResponse:
    """Perform an RFC 8693 Token Exchange.

    This enables \"Act On Behalf Of\" delegation chains. An agent can exchange
    a user's token (the subject) for a new token that identifies the agent
    as the actor acting on that user's behalf.

    Parameters
    ----------
    auth_url
        Base URL of the SharkAuth server.
    client_id
        OAuth 2.0 client ID of the acting agent.
    subject_token
        The token representing the identity being acted upon.
    client_secret
        Optional client secret for confidential clients.
    subject_token_type
        Type of subject token. Default: access_token.
    actor_token
        Optional token representing the actor's own identity.
    actor_token_type
        Type of actor token. Default: access_token.
    scope
        Optional space-delimited scope string for the new token.
    requested_token_type
        Type of token requested. Default: access_token.
    dpop_prover
        Optional DPoPProver for generating a DPoP-bound token.
    token_path
        Override token endpoint path. Default: /oauth/token.
    """
    token_url = auth_url.rstrip("/") + token_path
    sess = session or _http.new_session()

    data = {
        "grant_type": TOKEN_EXCHANGE_GRANT,
        "client_id": client_id,
        "subject_token": subject_token,
        "subject_token_type": subject_token_type,
        "requested_token_type": requested_token_type,
    }

    if client_secret:
        data["client_secret"] = client_secret
    if actor_token:
        data["actor_token"] = actor_token
        data["actor_token_type"] = actor_token_type
    if scope:
        data["scope"] = scope

    headers = {}
    if dpop_prover:
        headers["DPoP"] = dpop_prover.make_proof("POST", token_url)

    resp = _http.request(sess, "POST", token_url, data=data, headers=headers)

    try:
        payload = resp.json()
    except ValueError:
        raise TokenError(
            f"token endpoint returned non-JSON: HTTP {resp.status_code}: {resp.text[:200]}"
        )

    if resp.status_code == 200:
        return TokenResponse(
            access_token=payload["access_token"],
            token_type=payload.get("token_type", "Bearer"),
            expires_in=payload.get("expires_in"),
            refresh_token=payload.get("refresh_token"),
            scope=payload.get("scope"),
        )

    err = payload.get("error", "unknown")
    desc = payload.get("error_description", "")
    raise TokenError(f"token exchange failed: {err} (HTTP {resp.status_code}): {desc}")
