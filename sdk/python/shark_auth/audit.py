"""Audit log access — admin API.

Wraps ``/api/v1/audit-logs`` (list/get/export) and
``/api/v1/admin/audit-logs/purge`` (see ``internal/api/router.go``).
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


class AuditClient:
    """Admin client for querying and exporting audit logs.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    admin_api_key:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    _PREFIX = "/api/v1/audit-logs"
    _PURGE = "/api/v1/admin/audit-logs/purge"

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

    # ------------------------------------------------------------------
    # List
    # ------------------------------------------------------------------

    def list(
        self,
        *,
        actor_id: Optional[str] = None,
        actor_type: Optional[str] = None,
        action: Optional[str] = None,
        target_id: Optional[str] = None,
        target_type: Optional[str] = None,
        since: Optional[str] = None,
        until: Optional[str] = None,
        limit: int = 100,
        cursor: Optional[str] = None,
    ) -> Dict[str, Any]:
        """List audit log events. GET /api/v1/audit-logs.

        Returns
        -------
        dict
            ``{"events": [...], "next_cursor": ...}`` (shape may vary by
            server version — callers should access fields defensively).
        """
        params: Dict[str, Any] = {"limit": limit}
        if actor_id is not None:
            params["actor_id"] = actor_id
        if actor_type is not None:
            params["actor_type"] = actor_type
        if action is not None:
            params["action"] = action
        if target_id is not None:
            params["target_id"] = target_id
        if target_type is not None:
            params["target_type"] = target_type
        if since is not None:
            params["since"] = since
        if until is not None:
            params["until"] = until
        if cursor is not None:
            params["cursor"] = cursor

        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(
            self._session, "GET", url, headers=self._auth(), params=params
        )
        if resp.status_code == 200:
            body = resp.json()
            if isinstance(body, list):
                return {"events": body, "next_cursor": None}
            if isinstance(body, dict):
                if "events" in body:
                    return body
                if "data" in body and isinstance(body["data"], list):
                    return {
                        "events": body["data"],
                        "next_cursor": body.get("next_cursor"),
                    }
            return body
        _raise(resp)

    # ------------------------------------------------------------------
    # Single event
    # ------------------------------------------------------------------

    def get(self, event_id: str) -> Dict[str, Any]:
        """GET /api/v1/audit-logs/{id}."""
        url = f"{self._base}{self._PREFIX}/{event_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            body = resp.json()
            if isinstance(body, dict) and "data" in body and len(body) <= 2:
                return body["data"]
            return body
        _raise(resp)

    # ------------------------------------------------------------------
    # Export
    # ------------------------------------------------------------------

    def export(self, format: str = "ndjson", **filters: Any) -> str:
        """Export audit logs.

        Backend route: ``POST /api/v1/audit-logs/export``. The server
        currently emits CSV; the ``format`` argument is forwarded for
        forward compatibility (``ndjson``/``csv``/``json``).

        ``since`` and ``until`` (ISO 8601) are required by the backend.

        Returns
        -------
        str
            Raw export body text (CSV, NDJSON, or JSON depending on the
            server).
        """
        body: Dict[str, Any] = {"format": format}
        body.update(filters)
        url = f"{self._base}{self._PREFIX}/export"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code == 200:
            return resp.text
        _raise(resp)

    # ------------------------------------------------------------------
    # Purge
    # ------------------------------------------------------------------

    def purge(
        self,
        *,
        before: Optional[str] = None,
        dry_run: bool = False,
    ) -> Dict[str, Any]:
        """Purge old audit log entries.

        Backend route: ``POST /api/v1/admin/audit-logs/purge``. Deletes
        all entries strictly older than ``before`` (ISO 8601 timestamp).

        Returns the server response body — typically
        ``{"deleted": <int>}`` (or similar).
        """
        body: Dict[str, Any] = {"dry_run": dry_run}
        if before is not None:
            body["before"] = before
        url = f"{self._base}{self._PURGE}"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 202):
            try:
                payload = resp.json()
                if isinstance(payload, dict) and "data" in payload and len(payload) <= 2:
                    return payload["data"]
                return payload
            except Exception:
                return {"raw": resp.text}
        _raise(resp)
