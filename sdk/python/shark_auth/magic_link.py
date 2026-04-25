"""Magic link — send sign-in links via email."""

from __future__ import annotations

from typing import Any, Dict, Optional

from . import _http
from .proxy_rules import _raise


class MagicLinkClient:
    """Send magic-link sign-in emails.

    The endpoint is public — no admin key is required.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    session:
        Optional pre-configured :class:`requests.Session`.

    Example
    -------
    >>> client = MagicLinkClient(base_url="https://auth.example.com")
    >>> client.send_magic_link("user@example.com")
    """

    def __init__(
        self,
        base_url: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._session = session or _http.new_session()

    def send_magic_link(
        self,
        email: str,
        redirect_uri: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Send a magic-link email to *email*.

        The server applies per-email rate limiting (1 per 60 s) and always
        returns success to avoid leaking account-existence information.

        Parameters
        ----------
        email:
            Recipient email address.
        redirect_uri:
            Optional redirect URI embedded in the link (must be on the
            server's allowlist).

        Example
        -------
        >>> client.send_magic_link("user@example.com", "https://app.example.com/auth/callback")
        """
        body: Dict[str, str] = {"email": email}
        if redirect_uri is not None:
            body["redirect_uri"] = redirect_uri
        url = f"{self._base}/api/v1/auth/magic-link/send"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)
