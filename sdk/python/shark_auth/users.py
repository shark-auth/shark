"""User management — admin API (v1.5)."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Literal, Optional

from . import _http
from .proxy_rules import _raise


# ---------------------------------------------------------------------------
# W2 Methods 6 & 7 — typed response dataclasses
# ---------------------------------------------------------------------------


@dataclass
class CascadeRevokeResult:
    """Result of a cascade-revoke operation.

    Attributes
    ----------
    revoked_agent_ids:
        List of agent IDs whose tokens were revoked.
    revoked_consent_count:
        Number of OAuth consent records also revoked.
    audit_event_id:
        Audit log event ID for this operation.
    """

    revoked_agent_ids: List[str] = field(default_factory=list)
    revoked_consent_count: int = 0
    audit_event_id: Optional[str] = None


@dataclass
class AgentList:
    """Result of listing agents tied to a user.

    Attributes
    ----------
    data:
        List of agent dicts (same shape as AgentsClient.get_agent()).
    total:
        Total count (may exceed ``len(data)`` when paginated).
    filter:
        The filter applied (``"created"``, ``"authorized"``, or ``"all"``).
    """

    data: List[Dict[str, Any]] = field(default_factory=list)
    total: int = 0
    filter: str = "all"


class UsersClient:
    """Admin client for user management.

    Covers the ``/api/v1/users`` admin routes and the v1.5
    ``PATCH /api/v1/admin/users/{id}/tier`` endpoint.

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

    # ------------------------------------------------------------------
    # List users
    # ------------------------------------------------------------------

    def list_users(self, email: Optional[str] = None) -> List[Dict[str, Any]]:
        """List users, optionally filtered by *email*."""
        params: Dict[str, str] = {}
        if email is not None:
            params["email"] = email
        url = f"{self._base}/api/v1/users"
        resp = _http.request(self._session, "GET", url, headers=self._auth(), params=params)
        if resp.status_code == 200:
            body = resp.json()
            # Server returns either {data: [...]} or a plain list
            if isinstance(body, dict):
                return body.get("data", [])
            return body
        _raise(resp)

    # ------------------------------------------------------------------
    # Get user
    # ------------------------------------------------------------------

    def get_user(self, user_id: str) -> Dict[str, Any]:
        """Return a single user by *user_id*."""
        url = f"{self._base}/api/v1/users/{user_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            body = resp.json()
            return body.get("data", body) if isinstance(body, dict) else body
        _raise(resp)

    # ------------------------------------------------------------------
    # Create user (admin)
    # ------------------------------------------------------------------

    def create_user(
        self,
        email: str,
        *,
        password: Optional[str] = None,
        name: Optional[str] = None,
        email_verified: bool = False,
    ) -> Dict[str, Any]:
        """Create a new user via the admin endpoint.

        Parameters
        ----------
        email:
            User email address.
        password:
            Optional plaintext password hashed server-side.
        name:
            Display name.
        email_verified:
            Pre-verify the email. Default: ``False``.

        Example
        -------
        >>> client = UsersClient(base_url="https://auth.example.com", token="sk_live_...")
        >>> user = client.create_user("new@example.com", name="Alice")
        """
        body: Dict[str, Any] = {"email": email, "email_verified": email_verified}
        if password is not None:
            body["password"] = password
        if name is not None:
            body["name"] = name
        url = f"{self._base}/api/v1/admin/users"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            body_resp = resp.json()
            return body_resp.get("data", body_resp) if isinstance(body_resp, dict) else body_resp
        _raise(resp)

    # ------------------------------------------------------------------
    # Update user
    # ------------------------------------------------------------------

    def update_user(
        self,
        user_id: str,
        *,
        email: Optional[str] = None,
        name: Optional[str] = None,
        email_verified: Optional[bool] = None,
        metadata: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Update a user by *user_id* (partial update — only supplied fields changed).

        Parameters
        ----------
        user_id:
            The ``usr_*`` identifier of the user.
        email:
            New email address.
        name:
            New display name.
        email_verified:
            Override email-verified flag.
        metadata:
            Raw JSON metadata string.

        Example
        -------
        >>> client.update_user("usr_abc", name="Bob")
        """
        body: Dict[str, Any] = {}
        if email is not None:
            body["email"] = email
        if name is not None:
            body["name"] = name
        if email_verified is not None:
            body["email_verified"] = email_verified
        if metadata is not None:
            body["metadata"] = metadata
        url = f"{self._base}/api/v1/users/{user_id}"
        resp = _http.request(self._session, "PATCH", url, headers=self._auth(), json=body)
        if resp.status_code == 200:
            body_resp = resp.json()
            return body_resp.get("data", body_resp) if isinstance(body_resp, dict) else body_resp
        _raise(resp)

    # ------------------------------------------------------------------
    # Delete user
    # ------------------------------------------------------------------

    def delete_user(self, user_id: str) -> None:
        """Delete a user by *user_id*. Returns ``None`` on success (204).

        Example
        -------
        >>> client.delete_user("usr_abc")
        """
        url = f"{self._base}/api/v1/users/{user_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code == 204:
            return
        _raise(resp)

    # ------------------------------------------------------------------
    # Set tier (v1.5)
    # ------------------------------------------------------------------

    def set_user_tier(
        self,
        user_id: str,
        tier: Literal["free", "pro"],
    ) -> Dict[str, Any]:
        """Persist *tier* on *user_id* and return ``{"user": {...}, "tier": "..."}``.

        Only ``"free"`` and ``"pro"`` are accepted; anything else will be
        rejected with 400 by the server.
        """
        url = f"{self._base}/api/v1/admin/users/{user_id}/tier"
        resp = _http.request(
            self._session, "PATCH", url, headers=self._auth(), json={"tier": tier}
        )
        if resp.status_code == 200:
            return resp.json().get("data", resp.json())
        _raise(resp)

    # ------------------------------------------------------------------
    # W2 Method 6 — cascade revoke agents
    # ------------------------------------------------------------------

    def revoke_agents(
        self,
        user_id: str,
        *,
        agent_ids: Optional[List[str]] = None,
        reason: Optional[str] = None,
    ) -> CascadeRevokeResult:
        """Cascade-revoke agents owned by a user (Layer 3 depth-of-defense).

        Wraps ``POST /api/v1/users/{id}/revoke-agents``.

        If *agent_ids* is provided, only those agents are revoked (still scoped
        to this user's agents). If ``None``, revokes ALL agents created by this
        user AND all consents granted by this user.

        Parameters
        ----------
        user_id:
            The ``usr_*`` identifier of the user.
        agent_ids:
            Optional list of specific ``agent_*`` IDs to revoke.
            Omit to revoke all agents belonging to this user.
        reason:
            Human-readable reason for the revocation (recorded in audit log).

        Returns
        -------
        CascadeRevokeResult
            Contains ``revoked_agent_ids``, ``revoked_consent_count``,
            and ``audit_event_id``.

        Raises
        ------
        SharkAuthError
            On HTTP error from the server.

        Example
        -------
        >>> result = client.users.revoke_agents("usr_abc")
        >>> print(result.revoked_agent_ids)
        >>> print(result.audit_event_id)
        """
        body: Dict[str, Any] = {}
        if agent_ids is not None:
            body["agent_ids"] = agent_ids
        if reason is not None:
            body["reason"] = reason
        url = f"{self._base}/api/v1/users/{user_id}/revoke-agents"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 204):
            if resp.status_code == 204 or not resp.content:
                return CascadeRevokeResult()
            data = resp.json()
            if isinstance(data, dict) and "data" in data:
                data = data["data"]
            return CascadeRevokeResult(
                revoked_agent_ids=data.get("revoked_agent_ids", []),
                revoked_consent_count=data.get("revoked_consent_count", 0),
                audit_event_id=data.get("audit_event_id"),
            )
        _raise(resp)

    # ------------------------------------------------------------------
    # W2 Method 7 — list user agents
    # ------------------------------------------------------------------

    def list_agents(
        self,
        user_id: str,
        *,
        filter: Literal["created", "authorized", "all"] = "all",
        limit: int = 100,
        offset: int = 0,
    ) -> AgentList:
        """List agents tied to a user.

        Wraps ``GET /api/v1/users/{id}/agents?filter=...``.

        Parameters
        ----------
        user_id:
            The ``usr_*`` identifier of the user.
        filter:
            - ``"created"``: agents where ``created_by = user_id``
            - ``"authorized"``: agents this user has granted consent to
            - ``"all"``: union of the above (server may return ``"created"`` as default)
        limit:
            Maximum agents to return. Default: 100.
        offset:
            Pagination offset. Default: 0.

        Returns
        -------
        AgentList
            ``data`` list, ``total`` count, and effective ``filter`` string.

        Example
        -------
        >>> result = client.users.list_agents("usr_abc", filter="created")
        >>> for agent in result.data:
        ...     print(agent["name"])
        """
        params: Dict[str, Any] = {
            "filter": filter,
            "limit": limit,
            "offset": offset,
        }
        url = f"{self._base}/api/v1/users/{user_id}/agents"
        resp = _http.request(self._session, "GET", url, headers=self._auth(), params=params)
        if resp.status_code == 200:
            data = resp.json()
            if isinstance(data, dict):
                return AgentList(
                    data=data.get("data", []),
                    total=data.get("total", len(data.get("data", []))),
                    filter=data.get("filter", filter),
                )
            return AgentList(data=data if isinstance(data, list) else [], total=len(data or []))
        _raise(resp)

    # ------------------------------------------------------------------
    # Admin MFA disable
    # ------------------------------------------------------------------

    def reset_mfa(self, user_id: str) -> None:
        """Admin-force clear a user's TOTP MFA without requiring their current code.

        Wraps ``DELETE /api/v1/users/{id}/mfa``.  Use when a user has lost their
        MFA device and cannot log in.  Returns ``None`` on success (204).

        Parameters
        ----------
        user_id:
            The ``usr_*`` identifier of the user.

        Raises
        ------
        SharkAuthError
            On HTTP error from the server.
        """
        url = f"{self._base}/api/v1/users/{user_id}/mfa"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return
        _raise(resp)
