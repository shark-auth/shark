"""Method 4 — AgentTokenClaims.delegation_chain()

Pure JWT parsing (no signature verification) for walking RFC 8693 ``act`` claim chains.
The resource server is responsible for signature verification; this helper gives SDK users
a structured view of the delegation lineage.
"""

from __future__ import annotations

import base64
import json
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


def _b64url_decode(s: str) -> bytes:
    """Decode a base64url-encoded string without padding."""
    # Add padding
    pad = 4 - len(s) % 4
    if pad != 4:
        s += "=" * pad
    return base64.urlsafe_b64decode(s)


def _decode_payload(jwt_token: str) -> Dict[str, Any]:
    """Split a JWT and base64url-decode the payload segment."""
    parts = jwt_token.split(".")
    if len(parts) != 3:
        raise ValueError(f"malformed JWT: expected 3 parts, got {len(parts)}")
    payload_bytes = _b64url_decode(parts[1])
    return json.loads(payload_bytes)


@dataclass
class ActorClaim:
    """A single hop in an RFC 8693 delegation chain.

    Fields map directly from the JWT ``act`` claim object at that depth.

    Attributes
    ----------
    sub:
        The actor's subject identifier (agent ``client_id``).
    iat:
        Unix timestamp when this delegation was issued.
    scope:
        Space-delimited scopes at this delegation hop, if present.
    jkt:
        The DPoP JWK thumbprint bound at this hop (from ``cnf.jkt``), if any.
    """

    sub: str
    iat: int
    scope: Optional[str] = None
    jkt: Optional[str] = None


def _walk_act(act: Optional[Dict[str, Any]]) -> List[ActorClaim]:
    """Recursively walk the ``act`` claim bottom-up, returning each hop in order.

    The first element of the returned list is the outermost (most recent) actor;
    the last element is the innermost (original) actor.
    """
    if act is None:
        return []

    cnf = act.get("cnf")
    jkt = cnf.get("jkt") if isinstance(cnf, dict) else None

    hop = ActorClaim(
        sub=act.get("sub", ""),
        iat=int(act.get("iat", 0)),
        scope=act.get("scope"),
        jkt=jkt,
    )

    # Recurse into nested act
    nested = _walk_act(act.get("act"))
    return [hop] + nested


class AgentTokenClaims:
    """Parsed claims from a Shark agent access token (no signature verification).

    Use :meth:`parse` to construct from a raw JWT string.  This class is safe
    to use on the resource-server side where the signature has already been
    verified by the framework/middleware layer.

    Parameters
    ----------
    _payload:
        Raw decoded payload dict (private — use ``parse()`` instead).
    """

    def __init__(self, _payload: Dict[str, Any]) -> None:
        self._payload: Dict[str, Any] = _payload

    # ------------------------------------------------------------------
    # Construction
    # ------------------------------------------------------------------

    @classmethod
    def parse(cls, jwt_token: str) -> "AgentTokenClaims":
        """Parse a JWT token without verifying the signature.

        Parameters
        ----------
        jwt_token:
            A compact serialisation JWT (three base64url segments joined by ``.``).

        Returns
        -------
        AgentTokenClaims
            Parsed claims object.

        Raises
        ------
        ValueError
            If the token is malformed or the payload is not valid JSON.

        Example
        -------
        >>> claims = AgentTokenClaims.parse(token.access_token)
        >>> print(claims.sub)
        >>> for hop in claims.delegation_chain():
        ...     print(hop.sub, hop.scope)
        """
        payload = _decode_payload(jwt_token)
        return cls(payload)

    # ------------------------------------------------------------------
    # Properties
    # ------------------------------------------------------------------

    @property
    def sub(self) -> str:
        """Subject identifier of the token."""
        return self._payload.get("sub", "")

    @property
    def iss(self) -> str:
        """Issuer."""
        return self._payload.get("iss", "")

    @property
    def exp(self) -> int:
        """Expiry as Unix timestamp."""
        return int(self._payload.get("exp", 0))

    @property
    def iat(self) -> int:
        """Issued-at as Unix timestamp."""
        return int(self._payload.get("iat", 0))

    @property
    def scope(self) -> Optional[str]:
        """Space-delimited scope string, if present."""
        return self._payload.get("scope")

    @property
    def jkt(self) -> Optional[str]:
        """DPoP JWK thumbprint from ``cnf.jkt``, if bound."""
        cnf = self._payload.get("cnf")
        if isinstance(cnf, dict):
            return cnf.get("jkt")
        return None

    @property
    def raw(self) -> Dict[str, Any]:
        """The raw decoded payload dictionary."""
        return self._payload

    # ------------------------------------------------------------------
    # Delegation chain
    # ------------------------------------------------------------------

    def delegation_chain(self) -> List[ActorClaim]:
        """Walk the ``act`` claim chain bottom-up and return each hop in order.

        Returns an empty list when the token is a direct (non-delegated) token.
        The first element is the outermost (most recent) actor; the last is the
        innermost (original) actor.

        Returns
        -------
        list[ActorClaim]
            Each hop in the delegation chain, ordered outermost-first.

        Example
        -------
        >>> chain = claims.delegation_chain()
        >>> for hop in chain:
        ...     print(f"{hop.sub} delegated at {hop.iat} with scope {hop.scope}")
        """
        return _walk_act(self._payload.get("act"))

    def is_delegated(self) -> bool:
        """Return ``True`` if this token has at least one delegation hop.

        Example
        -------
        >>> if claims.is_delegated():
        ...     print("token is a delegated sub-token")
        """
        return bool(self._payload.get("act"))

    def has_scope(self, scope: str) -> bool:
        """Return ``True`` if *scope* is present in the token's scope string.

        Parameters
        ----------
        scope:
            A single scope string (e.g. ``"mcp:write"``).

        Example
        -------
        >>> assert claims.has_scope("mcp:read")
        """
        token_scope = self._payload.get("scope", "")
        if not token_scope:
            return False
        return scope in token_scope.split()
