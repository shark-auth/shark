"""Organization management — admin API.

Wraps the admin-key authenticated routes mounted under
``/api/v1/admin/organizations`` plus the user-facing accept-invitation
route at ``/api/v1/organizations/invitations/{token}/accept``.

Backend reference: ``internal/api/router.go``.
"""

from __future__ import annotations

import json
from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


class OrganizationsClient:
    """Admin client for managing organizations, members, and invitations.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    admin_api_key:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

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

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def _auth(self) -> Dict[str, str]:
        return {"Authorization": f"Bearer {self._token}"}

    @staticmethod
    def _unwrap(body: Any) -> Any:
        if isinstance(body, dict) and "data" in body and len(body) <= 2:
            return body["data"]
        return body

    @staticmethod
    def _coerce_metadata(value: Any) -> Any:
        """Backend stores org metadata as a JSON-encoded *string.

        If the caller passes a dict we serialize it; if it's already a
        string we leave it; ``None`` is passed through.
        """
        if value is None or isinstance(value, str):
            return value
        return json.dumps(value)

    # ------------------------------------------------------------------
    # CRUD
    # ------------------------------------------------------------------

    def create(self, name: str, slug: Optional[str] = None, **extra: Any) -> Dict[str, Any]:
        """Create an organization. POST /api/v1/admin/organizations."""
        body: Dict[str, Any] = {"name": name}
        if slug is not None:
            body["slug"] = slug
        if "metadata" in extra:
            extra["metadata"] = self._coerce_metadata(extra["metadata"])
        body.update(extra)
        url = f"{self._base}/api/v1/admin/organizations"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)

    def list(self) -> List[Dict[str, Any]]:
        """List organizations. GET /api/v1/admin/organizations."""
        url = f"{self._base}/api/v1/admin/organizations"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            body = resp.json()
            if isinstance(body, dict):
                return body.get("data", []) or []
            return body
        _raise(resp)

    def get(self, org_id: str) -> Dict[str, Any]:
        """Get a single organization. GET /api/v1/admin/organizations/{id}."""
        url = f"{self._base}/api/v1/admin/organizations/{org_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return self._unwrap(resp.json())
        _raise(resp)

    def update(self, org_id: str, **fields: Any) -> Dict[str, Any]:
        """Update an organization. PATCH /api/v1/admin/organizations/{id}."""
        if "metadata" in fields:
            fields["metadata"] = self._coerce_metadata(fields["metadata"])
        url = f"{self._base}/api/v1/admin/organizations/{org_id}"
        resp = _http.request(self._session, "PATCH", url, headers=self._auth(), json=fields)
        if resp.status_code == 200:
            return self._unwrap(resp.json())
        _raise(resp)

    def delete(self, org_id: str) -> None:
        """Delete an organization. DELETE /api/v1/admin/organizations/{id}."""
        url = f"{self._base}/api/v1/admin/organizations/{org_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return None
        _raise(resp)

    # ------------------------------------------------------------------
    # Members
    # ------------------------------------------------------------------

    def list_members(self, org_id: str) -> List[Dict[str, Any]]:
        """GET /api/v1/admin/organizations/{id}/members."""
        url = f"{self._base}/api/v1/admin/organizations/{org_id}/members"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            body = resp.json()
            if isinstance(body, dict):
                return body.get("data", []) or body.get("members", []) or []
            return body
        _raise(resp)

    def add_member(self, org_id: str, user_id: str, role: str) -> Dict[str, Any]:
        """Add a member directly.

        Note
        ----
        The backend admin surface does not currently expose a direct
        ``POST /members`` route — membership is established via
        ``create_invitation`` + ``accept_invitation``. This method is a
        thin convenience that POSTs to ``/admin/organizations/{id}/members``;
        if the route does not exist the server will return 404/405 which
        is surfaced to the caller via :class:`SharkAPIError`.
        """
        url = f"{self._base}/api/v1/admin/organizations/{org_id}/members"
        body = {"user_id": user_id, "role": role}
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)

    def update_member_role(self, org_id: str, member_id: str, role: str) -> Dict[str, Any]:
        """PATCH /api/v1/admin/organizations/{id}/members/{member_id}."""
        url = f"{self._base}/api/v1/admin/organizations/{org_id}/members/{member_id}"
        resp = _http.request(self._session, "PATCH", url, headers=self._auth(), json={"role": role})
        if resp.status_code == 200:
            return self._unwrap(resp.json())
        _raise(resp)

    def remove_member(self, org_id: str, member_id: str) -> None:
        """DELETE /api/v1/admin/organizations/{id}/members/{member_id}."""
        url = f"{self._base}/api/v1/admin/organizations/{org_id}/members/{member_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return None
        _raise(resp)

    # ------------------------------------------------------------------
    # Invitations
    # ------------------------------------------------------------------

    def list_invitations(self, org_id: str) -> List[Dict[str, Any]]:
        """GET /api/v1/admin/organizations/{id}/invitations."""
        url = f"{self._base}/api/v1/admin/organizations/{org_id}/invitations"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            body = resp.json()
            if isinstance(body, dict):
                return body.get("data", []) or body.get("invitations", []) or []
            return body
        _raise(resp)

    def create_invitation(self, org_id: str, email: str, role: str) -> Dict[str, Any]:
        """POST /api/v1/admin/organizations/{id}/invitations."""
        url = f"{self._base}/api/v1/admin/organizations/{org_id}/invitations"
        body = {"email": email, "role": role}
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)

    def delete_invitation(self, org_id: str, invitation_id: str) -> None:
        """DELETE /api/v1/admin/organizations/{id}/invitations/{invitationId}."""
        url = (
            f"{self._base}/api/v1/admin/organizations/{org_id}"
            f"/invitations/{invitation_id}"
        )
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code in (200, 204):
            return None
        _raise(resp)

    def resend_invitation(self, org_id: str, invitation_id: str) -> None:
        """POST /api/v1/admin/organizations/{id}/invitations/{invitationId}/resend."""
        url = (
            f"{self._base}/api/v1/admin/organizations/{org_id}"
            f"/invitations/{invitation_id}/resend"
        )
        resp = _http.request(self._session, "POST", url, headers=self._auth())
        if resp.status_code in (200, 202, 204):
            try:
                return self._unwrap(resp.json()) if resp.text else None
            except Exception:
                return None
        _raise(resp)

    def accept_invitation(self, token: str) -> Dict[str, Any]:
        """Accept an organization invitation.

        Calls ``POST /api/v1/organizations/invitations/{token}/accept`` —
        note the *user-facing* prefix (no ``/admin/``). This route is
        normally session-cookie authenticated; the admin key header is
        sent for parity with the rest of this client. Callers who need
        true session auth should use a session-bearing client.
        """
        url = f"{self._base}/api/v1/organizations/invitations/{token}/accept"
        resp = _http.request(self._session, "POST", url, headers=self._auth())
        if resp.status_code in (200, 201):
            return self._unwrap(resp.json())
        _raise(resp)
