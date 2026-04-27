"""Shark Token Vault client — fetch fresh 3rd-party OAuth credentials,
disconnect vault connections (with optional agent cascade), and issue
DPoP-authenticated token fetches.

W2 Method 9 adds:
- VaultDisconnectResult
- VaultTokenResult
- VaultClient.disconnect()
- VaultClient.fetch_token()
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import List, Optional

from . import _http
from .dpop import DPoPProver
from .errors import VaultError


@dataclass
class VaultToken:
    """A fresh, server-refreshed 3rd-party access token from the vault."""

    access_token: str
    expires_at: Optional[int]
    provider: Optional[str]
    scopes: List[str]


# ---------------------------------------------------------------------------
# W2 Method 9 — new typed results
# ---------------------------------------------------------------------------


@dataclass
class VaultDisconnectResult:
    """Result of disconnecting a vault connection.

    Attributes
    ----------
    connection_id:
        The connection that was deleted.
    revoked_agent_ids:
        Agent IDs whose tokens were cascade-revoked (empty when
        ``cascade_to_agents=False``).
    revoked_token_count:
        Total OAuth tokens revoked across all cascade-revoked agents.
    cascade_audit_event_id:
        Audit event ID for the cascade operation, or ``None`` when no
        cascade occurred.
    """

    connection_id: str
    revoked_agent_ids: List[str]
    revoked_token_count: int
    cascade_audit_event_id: Optional[str] = None


@dataclass
class VaultTokenResult:
    """A vault-brokered token returned via a DPoP-authenticated agent request.

    Attributes
    ----------
    access_token:
        The decrypted 3rd-party access token.
    token_type:
        Token type, typically ``"Bearer"``.
    expires_at:
        ISO-8601 expiry string, or ``None`` if the server did not supply one.
    provider:
        Provider name (e.g. ``"google_gmail"``).
    """

    access_token: str
    token_type: str = "Bearer"
    expires_at: Optional[str] = None
    provider: Optional[str] = None


class VaultClient:
    """Shark Token Vault client — connection management + delegated token fetch.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server (e.g. ``"https://auth.example.com"``).
    admin_key:
        Admin API key (``sk_live_...``). Required for admin operations such
        as :meth:`disconnect`.
    http_client:
        Optional pre-configured :class:`requests.Session`.

    Backward Compatibility
    ----------------------
    The legacy constructor signature ``VaultClient(auth_url=..., access_token=...)``
    is preserved: ``auth_url`` maps to ``base_url`` and ``access_token`` maps to the
    bearer token used by :meth:`get_fresh_token`.

    Example
    -------
    >>> vault = VaultClient(base_url="https://auth.example.com", admin_key="sk_live_...")
    >>> result = vault.disconnect("conn_abc123", cascade_to_agents=True)
    >>> print(result.revoked_agent_ids)
    """

    def __init__(
        self,
        base_url: str = "",
        admin_key: Optional[str] = None,
        http_client=None,
        *,
        # Legacy kwargs kept for backward compat
        auth_url: str = "",
        access_token: str = "",
        session: Optional[object] = None,
        connections_path: str = "/admin/vault/connections",
    ) -> None:
        # Resolve base_url from new or legacy kwarg
        self._base = (base_url or auth_url).rstrip("/")
        self._admin_key = admin_key or access_token
        self._session = http_client or session or _http.new_session()
        self._connections_path = connections_path

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def _admin_headers(self) -> dict:
        if self._admin_key:
            return {"Authorization": f"Bearer {self._admin_key}"}
        return {}

    def _headers(self) -> dict:
        """Alias for backward compat (legacy get_fresh_token path)."""
        return self._admin_headers()

    # ------------------------------------------------------------------
    # Legacy method — kept for backward compat
    # ------------------------------------------------------------------

    def get_fresh_token(self, connection_id: str) -> VaultToken:
        """REMOVED — there is no admin-key endpoint that returns a fresh
        3rd-party access token by ``connection_id``.

        The historical default path ``/admin/vault/connections/{id}/token``
        was never mounted by the backend; only ``DELETE
        /api/v1/admin/vault/connections/{id}`` exists for admin scope (see
        ``internal/api/router.go``).  Token retrieval is intentionally gated
        behind a DPoP-bound agent token so the vault can validate the
        ``cnf.jkt`` binding before decrypting the stored credential.

        Migration
        ---------
        Use :meth:`fetch_token` instead — request a DPoP-bound bearer via
        :meth:`OAuthClient.get_token_with_dpop` (scope ``vault:read``), then
        call ``vault.fetch_token(provider=..., bearer_token=..., prover=...)``
        to retrieve the decrypted 3rd-party token.

        Raises
        ------
        VaultError:
            Always — this method is intentionally unreachable.
        """
        raise VaultError(
            "VaultClient.get_fresh_token has been removed: no admin-key endpoint "
            "exists for this operation. Use VaultClient.fetch_token() with a "
            "DPoP-bound agent token (scope 'vault:read') instead. See the "
            "method docstring for the migration recipe."
        )

    # ------------------------------------------------------------------
    # W2 Method 9 — disconnect (with cascade)
    # ------------------------------------------------------------------

    def disconnect(
        self,
        connection_id: str,
        *,
        cascade_to_agents: bool = True,
    ) -> VaultDisconnectResult:
        """Disconnect a vault connection.

        If ``cascade_to_agents=True`` (default), also revokes tokens for any
        agent that has ever fetched from this vault connection (Layer 5 cascade).

        Wraps ``DELETE /api/v1/vault/connections/{id}``. Sends
        ``?cascade=true`` query parameter when cascading.

        Parameters
        ----------
        connection_id:
            The ``conn_*`` identifier of the vault connection to delete.
        cascade_to_agents:
            When ``True`` (default), the server cascade-revokes tokens for
            all agents that accessed this connection. Pass ``False`` to
            disconnect silently without agent impact.

        Returns
        -------
        VaultDisconnectResult
            Identifiers of cascade-revoked agents, total token count, and
            the optional cascade audit event ID.

        Raises
        ------
        VaultError:
            On 4xx/5xx responses.

        Example
        -------
        >>> result = vault.disconnect("conn_abc123")
        >>> print(result.revoked_agent_ids, result.revoked_token_count)
        """
        params = {"cascade": "true" if cascade_to_agents else "false"}
        url = f"{self._base}/api/v1/vault/connections/{connection_id}"
        resp = _http.request(
            self._session, "DELETE", url, headers=self._admin_headers(), params=params
        )

        if resp.status_code in (200, 204):
            if resp.status_code == 204 or not resp.content:
                return VaultDisconnectResult(
                    connection_id=connection_id,
                    revoked_agent_ids=[],
                    revoked_token_count=0,
                )
            body = resp.json()
            return VaultDisconnectResult(
                connection_id=body.get("connection_id", connection_id),
                revoked_agent_ids=body.get("revoked_agent_ids", []),
                revoked_token_count=body.get("revoked_token_count", 0),
                cascade_audit_event_id=body.get("cascade_audit_event_id"),
            )

        if resp.status_code == 404:
            raise VaultError(
                f"vault connection not found: {connection_id}", status_code=404
            )
        if resp.status_code == 401:
            raise VaultError("not authorized (401)", status_code=401)
        if resp.status_code == 403:
            raise VaultError("forbidden — admin key required (403)", status_code=403)
        raise VaultError(
            f"disconnect failed: HTTP {resp.status_code}: {resp.text[:200]}",
            status_code=resp.status_code,
        )

    # ------------------------------------------------------------------
    # W2 Method 9 — fetch_token (DPoP-authenticated)
    # ------------------------------------------------------------------

    def fetch_token(
        self,
        *,
        provider: str,
        bearer_token: str,
        prover: DPoPProver,
    ) -> VaultTokenResult:
        """Fetch a fresh OAuth token from the vault using a DPoP-bound agent token.

        ``GET /api/v1/vault/{provider}/token`` with
        ``Authorization: DPoP <bearer_token>`` and a DPoP proof header.
        The server validates the DPoP proof, ``vault:read`` scope, and the
        ``tok.UserID`` binding before decrypting and returning the stored token.

        Parameters
        ----------
        provider:
            Provider slug, e.g. ``"google_gmail"``.
        bearer_token:
            The agent's DPoP-bound access token (issued by SharkAuth).
        prover:
            The :class:`~shark_auth.DPoPProver` instance whose private key
            matches the ``cnf.jkt`` bound to ``bearer_token``.

        Returns
        -------
        VaultTokenResult
            Decrypted 3rd-party access token, type, expiry, and provider name.

        Raises
        ------
        VaultError:
            On any 4xx/5xx response.

        Example
        -------
        >>> prover = DPoPProver.generate()
        >>> token = client.oauth.get_token_with_dpop(
        ...     grant_type="client_credentials",
        ...     dpop_prover=prover,
        ...     client_id="shark_agent_...",
        ...     client_secret="...",
        ...     scope="vault:read",
        ... )
        >>> result = vault.fetch_token(
        ...     provider="google_gmail",
        ...     bearer_token=token.access_token,
        ...     prover=prover,
        ... )
        >>> print(result.access_token)
        """
        url = f"{self._base}/api/v1/vault/{provider}/token"
        proof = prover.make_proof(htm="GET", htu=url)
        headers = {
            "Authorization": f"DPoP {bearer_token}",
            "DPoP": proof,
        }
        resp = _http.request(self._session, "GET", url, headers=headers)

        if resp.status_code == 200:
            body = resp.json()
            return VaultTokenResult(
                access_token=body["access_token"],
                token_type=body.get("token_type", "Bearer"),
                expires_at=body.get("expires_at"),
                provider=body.get("provider", provider),
            )

        if resp.status_code == 401:
            raise VaultError("invalid or expired DPoP token (401)", status_code=401)
        if resp.status_code == 403:
            raise VaultError("missing vault:read scope (403)", status_code=403)
        if resp.status_code == 404:
            raise VaultError(
                f"no vault connection for provider '{provider}' (404)", status_code=404
            )
        raise VaultError(
            f"vault fetch_token failed: HTTP {resp.status_code}: {resp.text[:200]}",
            status_code=resp.status_code,
        )
