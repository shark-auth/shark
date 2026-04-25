"""Agent management — admin API."""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


class AgentsClient:
    """Admin client for managing machine-to-machine agents.

    Agents correspond to OAuth 2.1 clients registered in SharkAuth.
    This client wraps the ``/api/v1/agents`` admin routes.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    token:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    _PREFIX = "/api/v1/agents"

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
    # Register (create)
    # ------------------------------------------------------------------

    def register_agent(
        self,
        app_id: str,
        name: str,
        scopes: Optional[List[str]] = None,
        **extra: Any,
    ) -> Dict[str, Any]:
        """Register a new agent and return the created agent object.

        The response includes a one-time ``client_secret`` — copy it now as
        the server will not return it again.

        Parameters
        ----------
        app_id:
            Application ID to scope the agent to (stored in ``metadata``).
        name:
            Human-readable agent name (required by the server).
        scopes:
            List of OAuth scopes to grant.  Defaults to ``[]``.
        **extra:
            Additional fields forwarded verbatim (e.g. ``description``,
            ``token_lifetime``, ``redirect_uris``).
        """
        body: Dict[str, Any] = {
            "name": name,
            "scopes": scopes or [],
            "metadata": {"app_id": app_id},
            **extra,
        }
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return resp.json()
        _raise(resp)

    # ------------------------------------------------------------------
    # List
    # ------------------------------------------------------------------

    def list_agents(self, app_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """List all agents.  Filter by *app_id* when supplied."""
        params: Dict[str, str] = {}
        if app_id is not None:
            params["search"] = app_id  # server supports ?search= for name/id lookup
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "GET", url, headers=self._auth(), params=params)
        if resp.status_code == 200:
            return resp.json().get("data", [])
        _raise(resp)

    # ------------------------------------------------------------------
    # Get single agent
    # ------------------------------------------------------------------

    def get_agent(self, agent_id: str) -> Dict[str, Any]:
        """Return a single agent by *agent_id* (or client_id).

        Parameters
        ----------
        agent_id:
            The ``id`` (``agent_*``) or ``client_id`` (``shark_agent_*``) of
            the agent to fetch.

        Example
        -------
        >>> client = AgentsClient(base_url="https://auth.example.com", token="sk_live_...")
        >>> agent = client.get_agent("agent_abc123")
        >>> print(agent["name"])
        """
        url = f"{self._base}{self._PREFIX}/{agent_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)

    # ------------------------------------------------------------------
    # Revoke (deactivate)
    # ------------------------------------------------------------------

    def revoke_agent(self, agent_id: str) -> None:
        """Deactivate an agent and revoke all of its tokens.

        The server returns 204 on success.
        """
        url = f"{self._base}{self._PREFIX}/{agent_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code == 204:
            return
        _raise(resp)
