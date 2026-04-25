"""User management — admin API (v1.5)."""

from __future__ import annotations

from typing import Any, Dict, List, Literal, Optional

from . import _http
from .proxy_rules import _raise


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
