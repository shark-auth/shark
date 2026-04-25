"""OAuth 2.1 token utilities — revocation (RFC 7009) and introspection (RFC 7662)."""

from __future__ import annotations

from typing import Any, Dict, Literal, Optional

from . import _http
from .proxy_rules import _raise


class OAuthClient:
    """Client for RFC 7009 token revocation and RFC 7662 token introspection.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    token:
        Admin API key (``sk_live_...``). Required for introspection.
    session:
        Optional pre-configured :class:`requests.Session`.

    Example
    -------
    >>> client = OAuthClient(base_url="https://auth.example.com", token="sk_live_...")
    >>> client.revoke_token("my_access_token")
    >>> info = client.introspect_token("my_access_token")
    >>> print(info["active"])
    """

    def __init__(
        self,
        base_url: str,
        token: str = "",
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._token = token
        self._session = session or _http.new_session()

    def _auth(self) -> Dict[str, str]:
        if self._token:
            return {"Authorization": f"Bearer {self._token}"}
        return {}

    def revoke_token(
        self,
        token: str,
        token_type_hint: Optional[Literal["access_token", "refresh_token"]] = None,
    ) -> None:
        """Revoke a token (RFC 7009).

        The server always returns 200 regardless of whether the token existed.

        Parameters
        ----------
        token:
            The access or refresh token to revoke.
        token_type_hint:
            Optional hint — ``"access_token"`` or ``"refresh_token"``.

        Example
        -------
        >>> client.revoke_token("eyJhbGci...", "access_token")
        """
        data: Dict[str, str] = {"token": token}
        if token_type_hint is not None:
            data["token_type_hint"] = token_type_hint
        url = f"{self._base}/oauth/revoke"
        resp = _http.request(
            self._session, "POST", url, headers=self._auth(), data=data
        )
        if resp.status_code == 200:
            return
        _raise(resp)

    def introspect_token(self, token: str) -> Dict[str, Any]:
        """Introspect a token (RFC 7662).

        Returns a dict with ``active: True`` and claims for valid tokens, or
        ``{"active": False}`` for invalid/expired tokens.

        Parameters
        ----------
        token:
            The token to introspect.

        Example
        -------
        >>> info = client.introspect_token("eyJhbGci...")
        >>> if info["active"]:
        ...     print("sub:", info.get("sub"))
        """
        url = f"{self._base}/oauth/introspect"
        resp = _http.request(
            self._session, "POST", url, headers=self._auth(), data={"token": token}
        )
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)
