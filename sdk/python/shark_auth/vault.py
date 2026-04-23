"""Shark Token Vault client — fetch fresh 3rd-party OAuth credentials."""

from __future__ import annotations

from dataclasses import dataclass
from typing import List, Optional

from . import _http
from .errors import VaultError


@dataclass
class VaultToken:
    """A fresh, server-refreshed 3rd-party access token from the vault."""

    access_token: str
    expires_at: Optional[int]
    provider: Optional[str]
    scopes: List[str]


class VaultClient:
    """Fetches auto-refreshed 3rd-party tokens from the Shark Token Vault.

    Example
    -------
    >>> vault = VaultClient(auth_url="https://auth.example", access_token=agent_token)
    >>> fresh = vault.get_fresh_token(connection_id="conn_abc")
    >>> print(fresh.access_token)
    """

    def __init__(
        self,
        auth_url: str,
        access_token: str,
        *,
        session: Optional[object] = None,
        connections_path: str = "/admin/vault/connections",
    ) -> None:
        self.auth_url = auth_url.rstrip("/")
        self.access_token = access_token
        self._session = session or _http.new_session()
        self._connections_path = connections_path

    def _headers(self) -> dict:
        return {"Authorization": f"Bearer {self.access_token}"}

    def get_fresh_token(self, connection_id: str) -> VaultToken:
        """Retrieve a fresh access token for the given stored connection."""
        if not connection_id:
            raise VaultError("connection_id is required")
        url = f"{self.auth_url}{self._connections_path}/{connection_id}/token"
        resp = _http.request(self._session, "GET", url, headers=self._headers())

        if resp.status_code == 200:
            body = resp.json()
            scopes = body.get("scopes") or body.get("scope")
            if isinstance(scopes, str):
                scopes = scopes.split()
            elif scopes is None:
                scopes = []
            return VaultToken(
                access_token=body["access_token"],
                expires_at=body.get("expires_at"),
                provider=body.get("provider"),
                scopes=list(scopes),
            )

        if resp.status_code == 404:
            raise VaultError(
                f"connection not found: {connection_id}", status_code=404
            )
        if resp.status_code == 401:
            raise VaultError("agent not authorized (401)", status_code=401)
        if resp.status_code == 403:
            raise VaultError("missing scope for vault access (403)", status_code=403)

        raise VaultError(
            f"vault request failed: HTTP {resp.status_code}: {resp.text[:200]}",
            status_code=resp.status_code,
        )
