"""User management — admin API (v1.5)."""

from __future__ import annotations

from typing import Any, Dict, List, Literal, Optional

from . import _http
from .proxy_rules import _raise


class UsersClient:
    """Admin client for user management.

    Covers the ``/api/v1/users`` admin routes and the v1.5
    ``PATCH /api/v1/admin/users/{id}/tier`` endpoint.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    token:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    def __init__(
        self,
        base_url: str,
        token: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._token = token
        self._session = session or _http.new_session()

    def _auth(self) -> Dict[str, str]:
        return {"Authorization": f"Bearer {self._token}"}

    # ------------------------------------------------------------------
    # List users
    # ------------------------------------------------------------------

    def list_users(self, email: Optional[str] = None) -> List[Dict[str, Any]]:
        """List users, optionally filtered by *email*."""
        params: Dict[str, str] = {}
        if email is not None:
            params["email"] = email
        url = f"{self._base}/api/v1/users"
        resp = _http.request(self._session, "GET", url, headers=self._auth(), params=params)
        if resp.status_code == 200:
            body = resp.json()
            # Server returns either {data: [...]} or a plain list
            if isinstance(body, dict):
                return body.get("data", [])
            return body
        _raise(resp)

    # ------------------------------------------------------------------
    # Get user
    # ------------------------------------------------------------------

    def get_user(self, user_id: str) -> Dict[str, Any]:
        """Return a single user by *user_id*."""
        url = f"{self._base}/api/v1/users/{user_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            body = resp.json()
            return body.get("data", body) if isinstance(body, dict) else body
        _raise(resp)

    # ------------------------------------------------------------------
    # Set tier (v1.5)
    # ------------------------------------------------------------------

    def set_user_tier(
        self,
        user_id: str,
        tier: Literal["free", "pro"],
    ) -> Dict[str, Any]:
        """Persist *tier* on *user_id* and return ``{"user": {...}, "tier": "..."}``.

        Only ``"free"`` and ``"pro"`` are accepted; anything else will be
        rejected with 400 by the server.
        """
        url = f"{self._base}/api/v1/admin/users/{user_id}/tier"
        resp = _http.request(
            self._session, "PATCH", url, headers=self._auth(), json={"tier": tier}
        )
        if resp.status_code == 200:
            return resp.json().get("data", resp.json())
        _raise(resp)
