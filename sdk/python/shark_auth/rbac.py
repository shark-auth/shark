"""Role and permission management — admin API.

Wraps:

- ``/api/v1/roles`` — CRUD + permission attach/detach
- ``/api/v1/permissions`` — list/create/delete
- ``/api/v1/users/{user_id}/roles`` and
  ``/api/v1/users/{user_id}/roles/{rid}`` — role assign/revoke

See ``internal/api/router.go`` for the canonical route inventory.
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


class RBACClient:
    """Admin client for the global RBAC surface.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    admin_api_key:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    _ROLES = "/api/v1/roles"
    _PERMS = "/api/v1/permissions"
    _USERS = "/api/v1/users"

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

    @staticmethod
    def _list(body: Any, *keys: str) -> List[Dict[str, Any]]:
        if isinstance(body, list):
            return body
        if isinstance(body, dict):
            for k in (*keys, "data"):
                v = body.get(k)
                if isinstance(v, list):
                    return v
        return []

    # ------------------------------------------------------------------
    # Roles
    # ------------------------------------------------------------------

    def list_roles(self) -> List[Dict[str, Any]]:
        """GET /api/v1/roles."""
        url = f"{self._base}{self._ROLES}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return self._list(resp.json(), "roles")
        _raise(resp)

    def create_role(
        self, name: str, description: Optional[str] = None
    ) -> Dict[str, Any]:
        """POST /api/v1/roles."""
        body: Dict[str, Any] = {"name": name}
        if description is not None:
            body["description"] = description
        url = f"{self._base}{self._ROLES}"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)

    def get_role(self, role_id: str) -> Dict[str, Any]:
        """GET /api/v1/roles/{id}."""
        url = f"{self._base}{self._ROLES}/{role_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return self._unwrap(resp.json())
        _raise(resp)

    def update_role(self, role_id: str, **fields: Any) -> Dict[str, Any]:
        """PUT /api/v1/roles/{id}.

        The backend exposes update as PUT (not PATCH) — see router.go.
        """
        url = f"{self._base}{self._ROLES}/{role_id}"
        resp = _http.request(self._session, "PUT", url, headers=self._auth(), json=fields)
        if resp.status_code == 200:
            return self._unwrap(resp.json())
        _raise(resp)

    def delete_role(self, role_id: str) -> None:
        """DELETE /api/v1/roles/{id}."""
        url = f"{self._base}{self._ROLES}/{role_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return None
        _raise(resp)

    # ------------------------------------------------------------------
    # Permissions
    # ------------------------------------------------------------------

    def list_permissions(self) -> List[Dict[str, Any]]:
        """GET /api/v1/permissions."""
        url = f"{self._base}{self._PERMS}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return self._list(resp.json(), "permissions")
        _raise(resp)

    def create_permission(self, action: str, resource: str) -> Dict[str, Any]:
        """POST /api/v1/permissions."""
        body = {"action": action, "resource": resource}
        url = f"{self._base}{self._PERMS}"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)

    def attach_permission(self, role_id: str, permission_id: str) -> None:
        """POST /api/v1/roles/{role_id}/permissions."""
        url = f"{self._base}{self._ROLES}/{role_id}/permissions"
        resp = _http.request(
            self._session,
            "POST",
            url,
            headers=self._auth(),
            json={"permission_id": permission_id},
        )
        if resp.status_code in (200, 201, 204):
            return None
        _raise(resp)

    def detach_permission(self, role_id: str, permission_id: str) -> None:
        """DELETE /api/v1/roles/{role_id}/permissions/{permission_id}."""
        url = f"{self._base}{self._ROLES}/{role_id}/permissions/{permission_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return None
        _raise(resp)

    # ------------------------------------------------------------------
    # User <-> role assignment
    # ------------------------------------------------------------------

    def assign_role(self, user_id: str, role_id: str) -> None:
        """POST /api/v1/users/{user_id}/roles.

        Backend handler: ``handleAssignRole`` (router.go).
        """
        url = f"{self._base}{self._USERS}/{user_id}/roles"
        resp = _http.request(
            self._session,
            "POST",
            url,
            headers=self._auth(),
            json={"role_id": role_id},
        )
        if resp.status_code in (200, 201, 204):
            return None
        _raise(resp)

    def revoke_role(self, user_id: str, role_id: str) -> None:
        """DELETE /api/v1/users/{user_id}/roles/{role_id}.

        Backend handler: ``handleRemoveRole`` (router.go).
        """
        url = f"{self._base}{self._USERS}/{user_id}/roles/{role_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return None
        _raise(resp)

    # ------------------------------------------------------------------
    # Read-only helpers
    # ------------------------------------------------------------------

    def list_user_roles(self, user_id: str) -> List[Dict[str, Any]]:
        """GET /api/v1/users/{user_id}/roles."""
        url = f"{self._base}{self._USERS}/{user_id}/roles"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return self._list(resp.json(), "roles")
        _raise(resp)

    def list_user_permissions(self, user_id: str) -> List[Dict[str, Any]]:
        """GET /api/v1/users/{user_id}/permissions."""
        url = f"{self._base}{self._USERS}/{user_id}/permissions"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return self._list(resp.json(), "permissions")
        _raise(resp)
