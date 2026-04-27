"""MayAct grant client — admin API.

Wraps ``/api/v1/admin/may-act`` (list/create/revoke). Operator-issued
delegation grants letting subject ``from_id`` act on behalf of ``to_id``,
verified during RFC 8693 token exchange.
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


class MayActClient:
    """Admin client for managing may_act_grants rows.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    admin_api_key:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    _PREFIX = "/api/v1/admin/may-act"

    def __init__(
        self,
        base_url: str,
        admin_api_key: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._token = admin_api_key
        self._session = session or _http.new_session()

    def _auth(self) -> Dict[str, str]:
        return {"Authorization": f"Bearer {self._token}"}

    def find(
        self,
        from_id: Optional[str] = None,
        to_id: Optional[str] = None,
        include_revoked: bool = False,
    ) -> List[Dict[str, Any]]:
        """List grants matching the filter.

        Returns the unwrapped ``grants`` array directly.
        """
        params: Dict[str, Any] = {}
        if from_id is not None:
            params["from_id"] = from_id
        if to_id is not None:
            params["to_id"] = to_id
        if include_revoked:
            params["include_revoked"] = "true"
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(
            self._session, "GET", url, headers=self._auth(), params=params
        )
        if resp.status_code == 200:
            body = resp.json() or {}
            return list(body.get("grants") or [])
        _raise(resp)

    def create(
        self,
        from_id: str,
        to_id: str,
        *,
        max_hops: int = 1,
        scopes: Optional[List[str]] = None,
        expires_at: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Create a new grant. Returns the created row."""
        body: Dict[str, Any] = {
            "from_id": from_id,
            "to_id": to_id,
            "max_hops": max_hops,
            "scopes": scopes or [],
        }
        if expires_at is not None:
            body["expires_at"] = expires_at
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(
            self._session, "POST", url, headers=self._auth(), json=body
        )
        if resp.status_code in (200, 201):
            return resp.json()
        _raise(resp)

    def revoke(self, grant_id: str) -> Dict[str, Any]:
        """Revoke a grant by ID. Returns the updated row (revoked_at populated)."""
        url = f"{self._base}{self._PREFIX}/{grant_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)
