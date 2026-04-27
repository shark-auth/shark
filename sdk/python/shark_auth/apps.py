"""Application management — admin API.

Wraps ``/api/v1/admin/apps`` (see ``internal/api/router.go``).
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


class AppsClient:
    """Admin client for managing OAuth/embed applications.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    admin_api_key:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    _PREFIX = "/api/v1/admin/apps"

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

    # ------------------------------------------------------------------
    # CRUD
    # ------------------------------------------------------------------

    def create(
        self,
        name: str,
        integration_mode: str = "custom",
        redirect_uris: Optional[List[str]] = None,
        **extra: Any,
    ) -> Dict[str, Any]:
        """Create an application. POST /api/v1/admin/apps."""
        body: Dict[str, Any] = {"name": name, "integration_mode": integration_mode}
        if redirect_uris is not None:
            body["redirect_uris"] = redirect_uris
        body.update(extra)
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)

    def list(self) -> List[Dict[str, Any]]:
        """List apps. GET /api/v1/admin/apps."""
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            body = resp.json()
            if isinstance(body, dict):
                return body.get("data", []) or body.get("apps", []) or []
            return body
        _raise(resp)

    def get(self, app_id: str) -> Dict[str, Any]:
        """GET /api/v1/admin/apps/{id}."""
        url = f"{self._base}{self._PREFIX}/{app_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return self._unwrap(resp.json())
        _raise(resp)

    def update(self, app_id: str, **fields: Any) -> Dict[str, Any]:
        """PATCH /api/v1/admin/apps/{id}."""
        url = f"{self._base}{self._PREFIX}/{app_id}"
        resp = _http.request(self._session, "PATCH", url, headers=self._auth(), json=fields)
        if resp.status_code == 200:
            return self._unwrap(resp.json())
        _raise(resp)

    def delete(self, app_id: str) -> None:
        """DELETE /api/v1/admin/apps/{id}."""
        url = f"{self._base}{self._PREFIX}/{app_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return None
        _raise(resp)

    # ------------------------------------------------------------------
    # Operations
    # ------------------------------------------------------------------

    def rotate_secret(self, app_id: str) -> Dict[str, Any]:
        """Rotate the app's client secret.

        Backend route: ``POST /api/v1/admin/apps/{id}/rotate-secret``.
        Returns the freshly issued secret in the response body — store it
        immediately, the server cannot return it again.
        """
        url = f"{self._base}{self._PREFIX}/{app_id}/rotate-secret"
        resp = _http.request(self._session, "POST", url, headers=self._auth())
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)

    def get_snippet(self, app_id: str) -> str:
        """GET /api/v1/admin/apps/{id}/snippet — returns the embed snippet text."""
        url = f"{self._base}{self._PREFIX}/{app_id}/snippet"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            ctype = (resp.headers.get("Content-Type") or "").lower()
            if "application/json" in ctype:
                body = resp.json()
                if isinstance(body, dict):
                    for key in ("snippet", "embed", "html", "code"):
                        if key in body and isinstance(body[key], str):
                            return body[key]
                    data = body.get("data")
                    if isinstance(data, str):
                        return data
                    if isinstance(data, dict):
                        for key in ("snippet", "embed", "html", "code"):
                            if isinstance(data.get(key), str):
                                return data[key]
                return resp.text
            return resp.text
        _raise(resp)

