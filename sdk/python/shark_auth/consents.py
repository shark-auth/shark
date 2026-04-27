"""Self-service OAuth consent management — ``/api/v1/auth/consents``.

Lets users review and revoke the OAuth grants they've issued to agents.
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .auth import _raise_auth


class ConsentsClient:
    """Wraps the per-user OAuth-consent endpoints.

    All endpoints require the underlying :class:`requests.Session` to carry
    a valid session cookie (typically planted by :meth:`AuthClient.login`).

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    session:
        The :class:`requests.Session` carrying the user's session cookie.
    """

    _PREFIX = "/api/v1/auth/consents"

    def __init__(
        self,
        base_url: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._session = session or _http.new_session()

    def list(self) -> List[Dict[str, Any]]:
        """List active OAuth consents granted by the authenticated user."""
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "GET", url)
        if resp.status_code == 200:
            data = resp.json()
            if isinstance(data, dict):
                return list(data.get("data", []))
            return list(data) if isinstance(data, list) else []
        _raise_auth(resp)

    def revoke(self, consent_id: str) -> None:
        """Revoke a specific consent grant by id."""
        url = f"{self._base}{self._PREFIX}/{consent_id}"
        resp = _http.request(self._session, "DELETE", url)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)
