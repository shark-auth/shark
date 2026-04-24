"""Paywall page helpers — URL builder + HTML fetch (v1.5)."""

from __future__ import annotations

from typing import Any, Dict, Optional
from urllib.parse import urlencode

from . import _http
from .proxy_rules import SharkAPIError, _raise


class PaywallClient:
    """Client for the paywall page endpoint.

    The paywall at ``GET /paywall/{app_slug}`` is a **public** unauthenticated
    endpoint — no admin key is required.  The primary helper is
    :meth:`paywall_url` which builds the URL without making a network call;
    :meth:`render_paywall` and :meth:`preview_paywall` fetch the HTML.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    token:
        Optional admin API key — only used if you want to call authenticated
        endpoints alongside paywall helpers from the same client object.
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    def __init__(
        self,
        base_url: str,
        token: Optional[str] = None,
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

    # ------------------------------------------------------------------
    # URL builder (no network call)
    # ------------------------------------------------------------------

    def paywall_url(
        self,
        app_slug: str,
        tier: str,
        return_url: Optional[str] = None,
    ) -> str:
        """Build the paywall redirect URL without making a network request.

        Returns the full URL the proxy redirects to on a tier mismatch.
        """
        params: Dict[str, str] = {"tier": tier}
        if return_url:
            params["return"] = return_url
        qs = urlencode(params)
        return f"{self._base}/paywall/{app_slug}?{qs}"

    # ------------------------------------------------------------------
    # Render (fetches HTML)
    # ------------------------------------------------------------------

    def render_paywall(
        self,
        app_slug: str,
        tier: str,
        return_url: Optional[str] = None,
    ) -> str:
        """Fetch the rendered paywall HTML page.

        Returns the ``text/html`` response body as a string.
        Raises :class:`~shark_auth.proxy_rules.SharkAPIError` on 400/404.
        """
        url = self.paywall_url(app_slug, tier, return_url)
        resp = _http.request(self._session, "GET", url)
        if resp.status_code == 200:
            return resp.text
        _raise(resp)

    # ------------------------------------------------------------------
    # Preview
    # ------------------------------------------------------------------

    def preview_paywall(
        self,
        app_slug: str,
        tier: str,
        return_url: Optional[str] = None,
        format: str = "html",  # noqa: A002
    ) -> Any:
        """Fetch the paywall page and return it in the requested format.

        ``format`` may be ``"html"`` (default, returns the raw HTML string)
        or ``"url"`` (returns the URL without fetching).
        """
        if format == "url":
            return self.paywall_url(app_slug, tier, return_url)
        return self.render_paywall(app_slug, tier, return_url)
