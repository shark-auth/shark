"""High-level SharkAuth admin client."""

from __future__ import annotations

from typing import Optional

from . import _http
from .agents import AgentsClient
from .branding import BrandingClient
from .http_client import DPoPHTTPClient
from .paywall import PaywallClient
from .proxy_lifecycle import ProxyLifecycleClient
from .proxy_rules import ProxyRulesClient
from .users import UsersClient


class Client:
    """SharkAuth admin client — unified namespace for all v1.5 admin APIs.

    Exposes sub-clients as namespaced attributes that mirror the TypeScript
    SDK surface (Lane F):

    - ``.proxy_rules``     — DB-backed proxy rules CRUD
    - ``.proxy_lifecycle`` — start / stop / reload / status
    - ``.branding``        — design tokens
    - ``.paywall``         — paywall URL builder + HTML fetch
    - ``.users``           — user list / get / tier
    - ``.agents``          — agent register / list / revoke
    - ``.http``            — DPoP-protected resource requests (get/post/delete)

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server (e.g. ``https://auth.example.com``).
    token:
        Admin API key (``sk_live_...``).
    session:
        Optional shared :class:`requests.Session`.  When omitted each
        sub-client creates its own session.

    Example
    -------
    >>> c = Client(base_url="https://auth.example.com", token="sk_live_abc")
    >>> rules = c.proxy_rules.list_rules()
    >>> status = c.proxy_lifecycle.get_proxy_status()
    """

    def __init__(
        self,
        base_url: str,
        token: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.token = token

        # Shared session — all sub-clients reuse the same connection pool.
        _session = session or _http.new_session()

        self.proxy_rules = ProxyRulesClient(base_url, token, session=_session)
        self.proxy_lifecycle = ProxyLifecycleClient(base_url, token, session=_session)
        self.branding = BrandingClient(base_url, token, session=_session)
        self.paywall = PaywallClient(base_url, token, session=_session)
        self.users = UsersClient(base_url, token, session=_session)
        self.agents = AgentsClient(base_url, token, session=_session)
        # DPoP-authenticated resource client — shares the same session pool.
        self.http = DPoPHTTPClient(base_url, session=_session)

    def close(self) -> None:
        """Close the underlying HTTP session."""
        self.http.close()

    def __repr__(self) -> str:  # pragma: no cover
        return f"Client(base_url={self.base_url!r})"
