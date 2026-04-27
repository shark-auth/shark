"""High-level SharkAuth admin client."""

from __future__ import annotations

from typing import Optional

from . import _http
from .agents import AgentsClient
from .api_keys import APIKeysClient
from .apps import AppsClient
from .audit import AuditClient
from .auth import AuthClient
from .branding import BrandingClient
from .consents import ConsentsClient
from .dcr import DCRClient
from .http_client import DPoPHTTPClient
from .mfa import MFAClient
from .organizations import OrganizationsClient
from .paywall import PaywallClient
from .proxy_lifecycle import ProxyLifecycleClient
from .proxy_rules import ProxyRulesClient
from .rbac import RBACClient
from .sessions import SessionsClient
from .users import UsersClient
from .webhooks import WebhooksClient


class Client:
    """SharkAuth admin client — unified namespace for all v1.5 admin APIs.

    Exposes sub-clients as namespaced attributes:

    Human auth + identity:
    - ``.auth``            — signup / login / logout / me / password / email-verify
    - ``.mfa``             — TOTP enroll / verify / challenge / disable
    - ``.sessions``        — list + revoke (self-service)
    - ``.consents``        — list + revoke OAuth consents

    OAuth + DCR:
    - ``.dcr``             — RFC 7591/7592 dynamic client registration
    - (use ``shark_auth.OAuthClient`` directly for token grants + introspect + revoke)

    Agent platform:
    - ``.agents``          — agent CRUD + token ops + DPoP rotation + audit
    - ``.users``           — admin user CRUD + cascade-revoke
    - ``.organizations``   — org CRUD + members + invitations
    - ``.apps``            — application CRUD + secret rotate
    - ``.api_keys``        — admin API key CRUD + rotate
    - ``.rbac``            — role + permission CRUD + assignment
    - ``.audit``           — audit log query + export + purge

    Integration:
    - ``.webhooks``        — webhook CRUD + test + replay + deliveries
    - ``.proxy_rules``     — DB-backed proxy rules CRUD
    - ``.proxy_lifecycle`` — proxy start / stop / reload / status
    - ``.branding``        — design tokens
    - ``.paywall``         — paywall URL builder + HTML fetch
    - ``.http``            — DPoP-protected resource requests

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

        # Human auth + identity (no admin key needed; uses session cookies)
        self.auth = AuthClient(base_url, session=_session)
        self.mfa = MFAClient(base_url, session=_session)
        self.sessions = SessionsClient(base_url, session=_session)
        self.consents = ConsentsClient(base_url, session=_session)

        # OAuth dynamic client registration
        self.dcr = DCRClient(base_url, session=_session)

        # Admin (require admin API key)
        self.users = UsersClient(base_url, token, session=_session)
        self.agents = AgentsClient(base_url, token, session=_session)
        self.organizations = OrganizationsClient(base_url, token, session=_session)
        self.apps = AppsClient(base_url, token, session=_session)
        self.api_keys = APIKeysClient(base_url, token, session=_session)
        self.rbac = RBACClient(base_url, token, session=_session)
        self.audit = AuditClient(base_url, token, session=_session)
        self.webhooks = WebhooksClient(base_url, token, session=_session)

        # Proxy + branding + paywall
        self.proxy_rules = ProxyRulesClient(base_url, token, session=_session)
        self.proxy_lifecycle = ProxyLifecycleClient(base_url, token, session=_session)
        self.branding = BrandingClient(base_url, token, session=_session)
        self.paywall = PaywallClient(base_url, token, session=_session)

        # DPoP-authenticated resource client — shares the same session pool.
        self.http = DPoPHTTPClient(base_url, session=_session)

    def close(self) -> None:
        """Close the underlying HTTP session."""
        self.http.close()

    def __repr__(self) -> str:  # pragma: no cover
        return f"Client(base_url={self.base_url!r})"
