"""Proxy lifecycle control — start / stop / reload / status (v1.5)."""

from __future__ import annotations

from typing import Dict, Literal, Optional, TypedDict

from . import _http
from .proxy_rules import SharkAPIError, _raise


class ProxyStatus(TypedDict):
    """Live status snapshot from the proxy Manager."""

    state: int
    state_str: Literal["stopped", "running", "reloading", "unknown"]
    listeners: int
    rules_loaded: int
    started_at: str
    last_error: str


class ProxyLifecycleClient:
    """Admin client for proxy lifecycle management.

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

    def _post(self, path: str) -> ProxyStatus:
        url = f"{self._base}{path}"
        resp = _http.request(self._session, "POST", url, headers=self._auth())
        if resp.status_code == 200:
            return resp.json().get("data", resp.json())
        _raise(resp)

    def _get(self, path: str) -> ProxyStatus:
        url = f"{self._base}{path}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return resp.json().get("data", resp.json())
        _raise(resp)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def get_proxy_status(self) -> ProxyStatus:
        """Return the current proxy Manager status."""
        return self._get("/api/v1/admin/proxy/lifecycle")

    def start_proxy(self) -> ProxyStatus:
        """Start the proxy.  Returns status after transition."""
        return self._post("/api/v1/admin/proxy/start")

    def stop_proxy(self) -> ProxyStatus:
        """Stop the proxy.  Idempotent — stopping a stopped proxy returns 200."""
        return self._post("/api/v1/admin/proxy/stop")

    def reload_proxy(self) -> ProxyStatus:
        """Reload the proxy (stop + start in one critical section)."""
        return self._post("/api/v1/admin/proxy/reload")
