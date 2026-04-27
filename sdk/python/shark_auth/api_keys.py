"""Admin API-key management.

Wraps ``/api/v1/api-keys`` (mounted under the admin-key auth group; see
``internal/api/router.go``).
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


class APIKeysClient:
    """Admin client for managing admin API keys.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    admin_api_key:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    _PREFIX = "/api/v1/api-keys"

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

    @staticmethod
    def _unwrap(body: Any) -> Any:
        if isinstance(body, dict) and "data" in body and len(body) <= 2:
            return body["data"]
        return body

    def create(
        self,
        name: str,
        scopes: Optional[List[str]] = None,
        expires_at: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Create a new admin API key. POST /api/v1/api-keys.

        Returns a dict containing both ``key_id`` (or ``id``) *and* the
        full ``key`` value. The full key is only returned at creation —
        store it immediately.
        """
        body: Dict[str, Any] = {"name": name}
        if scopes is not None:
            body["scopes"] = scopes
        if expires_at is not None:
            body["expires_at"] = expires_at
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)

    def list(self) -> List[Dict[str, Any]]:
        """GET /api/v1/api-keys."""
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            body = resp.json()
            if isinstance(body, dict):
                return body.get("data", []) or body.get("api_keys", []) or []
            return body
        _raise(resp)

    def get(self, key_id: str) -> Dict[str, Any]:
        """GET /api/v1/api-keys/{id}."""
        url = f"{self._base}{self._PREFIX}/{key_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return self._unwrap(resp.json())
        _raise(resp)

    def revoke(self, key_id: str) -> None:
        """Revoke (delete) an API key. DELETE /api/v1/api-keys/{id}."""
        url = f"{self._base}{self._PREFIX}/{key_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return None
        _raise(resp)

    def rotate(self, key_id: str) -> Dict[str, Any]:
        """Rotate an API key. POST /api/v1/api-keys/{id}/rotate.

        Returns a dict with the new ``key`` and (typically) a new
        ``key_id``. Store the new key immediately — the server cannot
        return it again.
        """
        url = f"{self._base}{self._PREFIX}/{key_id}/rotate"
        resp = _http.request(self._session, "POST", url, headers=self._auth())
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)
