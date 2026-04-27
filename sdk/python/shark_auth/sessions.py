"""Self-service session management — ``/api/v1/auth/sessions``.

Lists the authenticated user's active sessions and lets them revoke
individual sessions or all sessions.
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .auth import _raise_auth


class SessionsClient:
    """Wraps the per-user session-management endpoints.

    All endpoints require the underlying :class:`requests.Session` to carry
    a valid session cookie (typically planted by :meth:`AuthClient.login`).

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    session:
        The :class:`requests.Session` carrying the user's session cookie.
    """

    _PREFIX = "/api/v1/auth/sessions"

    def __init__(
        self,
        base_url: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._session = session or _http.new_session()

    def list(self) -> List[Dict[str, Any]]:
        """List the authenticated user's active sessions (``GET /sessions``)."""
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "GET", url)
        if resp.status_code == 200:
            data = resp.json()
            if isinstance(data, dict):
                return list(data.get("data", []))
            return list(data) if isinstance(data, list) else []
        _raise_auth(resp)

    def revoke(self, session_id: str) -> None:
        """Revoke a specific session by id (``DELETE /sessions/{id}``)."""
        url = f"{self._base}{self._PREFIX}/{session_id}"
        resp = _http.request(self._session, "DELETE", url)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    def revoke_all(self) -> None:
        """Revoke every session except the current one.

        The user-facing surface does not expose a bulk-revoke endpoint, so
        this iterates :meth:`list` and skips the row flagged ``current``.
        """
        for sess in self.list():
            if sess.get("current"):
                continue
            sid = sess.get("id") or sess.get("session_id")
            if sid:
                self.revoke(sid)
