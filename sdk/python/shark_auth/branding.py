"""Branding design tokens — admin API (v1.5)."""

from __future__ import annotations

from typing import Any, Dict, Optional

from . import _http
from .proxy_rules import _raise


class BrandingClient:
    """Admin client for branding design tokens.

    The ``get_branding`` method calls ``GET /api/v1/admin/branding``.
    The ``set_branding`` method calls ``PATCH /api/v1/admin/branding/design-tokens``.

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

    def get_branding(self, app_slug: Optional[str] = None) -> Dict[str, Any]:
        """Return the current branding row.

        ``app_slug`` is accepted for API symmetry but is ignored by the
        server today — there is one global branding row.
        """
        url = f"{self._base}/api/v1/admin/branding"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return resp.json().get("data", resp.json())
        _raise(resp)

    def set_branding(self, app_slug: Optional[str] = None, tokens: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Persist design tokens on the global branding row.

        ``app_slug`` is accepted for API symmetry but the server operates on
        the global branding row regardless.  ``tokens`` is the free-form
        design-token JSON object (any depth).  An empty dict clears the tokens.

        Returns ``{"branding": {...}, "design_tokens": {...}}``.
        """
        payload: Dict[str, Any] = {"design_tokens": tokens if tokens is not None else {}}
        url = f"{self._base}/api/v1/admin/branding/design-tokens"
        resp = _http.request(self._session, "PATCH", url, headers=self._auth(), json=payload)
        if resp.status_code == 200:
            return resp.json().get("data", resp.json())
        _raise(resp)
