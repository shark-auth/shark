"""Agent management — admin API."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


# ---------------------------------------------------------------------------
# W2 Method 5 — typed response dataclasses
# ---------------------------------------------------------------------------


@dataclass
class TokenInfo:
    """A single active token record for an agent.

    Attributes
    ----------
    token_id:
        Opaque token identifier.
    agent_id:
        The agent this token belongs to.
    jkt:
        DPoP JWK thumbprint, if the token is DPoP-bound.
    scope:
        Space-delimited scopes granted.
    expires_at:
        ISO-8601 expiry timestamp.
    created_at:
        ISO-8601 creation timestamp.
    """

    token_id: str
    agent_id: str
    jkt: Optional[str] = None
    scope: Optional[str] = None
    expires_at: Optional[str] = None
    created_at: Optional[str] = None


@dataclass
class RevokeResult:
    """Result of a revoke-all operation.

    Attributes
    ----------
    revoked_count:
        Number of tokens revoked.
    agent_id:
        The agent whose tokens were revoked.
    """

    revoked_count: int
    agent_id: str


@dataclass
class AgentCredentials:
    """New credentials returned after a secret rotation.

    Attributes
    ----------
    agent_id:
        The agent whose secret was rotated.
    client_id:
        OAuth client ID (unchanged).
    client_secret:
        New client secret — copy now, will not be shown again.
    rotated_at:
        ISO-8601 timestamp of the rotation.
    """

    agent_id: str
    client_id: str
    client_secret: str
    rotated_at: Optional[str] = None


@dataclass
class AuditEvent:
    """A single audit-log entry.

    Attributes
    ----------
    id:
        Unique audit event identifier.
    event:
        Event type string (e.g. ``"agent.token_issued"``).
    actor_id:
        ID of the principal that triggered the event.
    target_id:
        ID of the affected resource, if applicable.
    metadata:
        Arbitrary event metadata dict.
    created_at:
        ISO-8601 timestamp.
    """

    id: str
    event: str
    actor_id: Optional[str] = None
    target_id: Optional[str] = None
    metadata: Dict[str, Any] = field(default_factory=dict)
    created_at: Optional[str] = None


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

    # ------------------------------------------------------------------
    # W2 Method 5 — token management extras
    # ------------------------------------------------------------------

    def list_tokens(self, agent_id: str) -> List[TokenInfo]:
        """List active tokens for an agent.

        Wraps ``GET /api/v1/agents/{id}/tokens``.

        Parameters
        ----------
        agent_id:
            The ``agent_*`` identifier of the agent.

        Returns
        -------
        list[TokenInfo]
            Active token records for the agent.

        Example
        -------
        >>> tokens = client.agents.list_tokens("agent_abc")
        >>> for tok in tokens:
        ...     print(tok.token_id, tok.scope)
        """
        url = f"{self._base}{self._PREFIX}/{agent_id}/tokens"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            data = resp.json()
            items = data.get("data", data) if isinstance(data, dict) else data
            return [
                TokenInfo(
                    token_id=t.get("id", t.get("token_id", "")),
                    agent_id=t.get("agent_id", agent_id),
                    jkt=t.get("jkt"),
                    scope=t.get("scope"),
                    expires_at=t.get("expires_at"),
                    created_at=t.get("created_at"),
                )
                for t in (items if isinstance(items, list) else [])
            ]
        _raise(resp)

    def revoke_all(self, agent_id: str) -> RevokeResult:
        """Revoke all active tokens for an agent.

        Wraps ``POST /api/v1/agents/{id}/tokens/revoke-all``.

        Parameters
        ----------
        agent_id:
            The ``agent_*`` identifier of the agent.

        Returns
        -------
        RevokeResult
            The number of tokens revoked.

        Example
        -------
        >>> result = client.agents.revoke_all("agent_abc")
        >>> print(result.revoked_count)
        """
        url = f"{self._base}{self._PREFIX}/{agent_id}/tokens/revoke-all"
        resp = _http.request(self._session, "POST", url, headers=self._auth())
        if resp.status_code in (200, 204):
            if resp.status_code == 204 or not resp.content:
                return RevokeResult(revoked_count=0, agent_id=agent_id)
            data = resp.json()
            return RevokeResult(
                revoked_count=data.get("revoked_count", data.get("count", 0)),
                agent_id=agent_id,
            )
        _raise(resp)

    def rotate_secret(self, agent_id: str) -> AgentCredentials:
        """Rotate the agent's client secret.

        Wraps ``POST /api/v1/agents/{id}/rotate-secret``.

        Parameters
        ----------
        agent_id:
            The ``agent_*`` identifier of the agent.

        Returns
        -------
        AgentCredentials
            New credentials — copy ``client_secret`` now, not shown again.

        Example
        -------
        >>> creds = client.agents.rotate_secret("agent_abc")
        >>> print(creds.client_secret)
        """
        url = f"{self._base}{self._PREFIX}/{agent_id}/rotate-secret"
        resp = _http.request(self._session, "POST", url, headers=self._auth())
        if resp.status_code in (200, 201):
            data = resp.json()
            # Server may nest under "data" key
            if isinstance(data, dict) and "data" in data:
                data = data["data"]
            return AgentCredentials(
                agent_id=data.get("agent_id", agent_id),
                client_id=data.get("client_id", ""),
                client_secret=data.get("client_secret", ""),
                rotated_at=data.get("rotated_at"),
            )
        _raise(resp)

    def get_audit_logs(
        self,
        agent_id: str,
        *,
        limit: int = 100,
        since: Optional[datetime] = None,
    ) -> List[AuditEvent]:
        """Fetch audit events filtered to this agent.

        Wraps ``GET /api/v1/audit-logs?actor_id=<id>``.

        Parameters
        ----------
        agent_id:
            The ``agent_*`` identifier of the agent (used as ``actor_id`` filter).
        limit:
            Maximum number of events to return. Default: 100.
        since:
            Only return events after this datetime. Default: no filter.

        Returns
        -------
        list[AuditEvent]
            Audit log entries for the agent, newest first.

        Example
        -------
        >>> events = client.agents.get_audit_logs("agent_abc", limit=10)
        >>> for ev in events:
        ...     print(ev.event, ev.created_at)
        """
        params: Dict[str, Any] = {"actor_id": agent_id, "limit": limit}
        if since is not None:
            params["since"] = since.isoformat()
        url = f"{self._base}/api/v1/audit-logs"
        resp = _http.request(self._session, "GET", url, headers=self._auth(), params=params)
        if resp.status_code == 200:
            data = resp.json()
            items = data.get("data", data) if isinstance(data, dict) else data
            return [
                AuditEvent(
                    id=e.get("id", ""),
                    event=e.get("event", e.get("action", "")),
                    actor_id=e.get("actor_id"),
                    target_id=e.get("target_id"),
                    metadata=e.get("metadata", {}),
                    created_at=e.get("created_at"),
                )
                for e in (items if isinstance(items, list) else [])
            ]
        _raise(resp)
