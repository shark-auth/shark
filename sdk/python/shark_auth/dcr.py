"""Dynamic Client Registration (RFC 7591) + Configuration Management (RFC 7592).

Public client onboarding for the OAuth server. Each registered client gets a
``client_id``, optional ``client_secret``, and a ``registration_access_token``
which authenticates subsequent management requests to the
``registration_client_uri``.
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from . import _http
from .errors import SharkAuthError
from .proxy_rules import _raise


class DCRClient:
    """Client for RFC 7591 Dynamic Client Registration and RFC 7592 management.

    Backend routes (read-only inventory from ``internal/api/router.go``):

    - ``POST   /oauth/register``                                  — register
    - ``GET    /oauth/register/{client_id}``                      — read
    - ``PUT    /oauth/register/{client_id}``                      — update
    - ``DELETE /oauth/register/{client_id}``                      — delete
    - ``POST   /oauth/register/{client_id}/secret``               — rotate secret
    - ``DELETE /oauth/register/{client_id}/registration-token``   — rotate registration token

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    session:
        Optional pre-configured :class:`requests.Session`.

    Example
    -------
    >>> dcr = DCRClient(base_url="https://auth.example.com")
    >>> reg = dcr.register(
    ...     client_name="My App",
    ...     redirect_uris=["https://app.example.com/cb"],
    ...     grant_types=["authorization_code", "refresh_token"],
    ... )
    >>> cid = reg["client_id"]
    >>> rat = reg["registration_access_token"]
    >>> info = dcr.get(cid, rat)
    """

    _PREFIX = "/oauth/register"

    def __init__(
        self,
        base_url: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._session = session or _http.new_session()

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _bearer(self, registration_access_token: str) -> Dict[str, str]:
        return {"Authorization": f"Bearer {registration_access_token}"}

    # ------------------------------------------------------------------
    # RFC 7591 — register
    # ------------------------------------------------------------------

    def register(
        self,
        client_name: str,
        *,
        redirect_uris: Optional[List[str]] = None,
        grant_types: Optional[List[str]] = None,
        response_types: Optional[List[str]] = None,
        scope: Optional[str] = None,
        token_endpoint_auth_method: Optional[str] = None,
        **extra: Any,
    ) -> Dict[str, Any]:
        """Register a new OAuth client (RFC 7591).

        Parameters
        ----------
        client_name:
            Human-readable client name (required).
        redirect_uris:
            Permitted redirect URIs.
        grant_types:
            Allowed grant types, e.g. ``["authorization_code", "refresh_token"]``.
        response_types:
            Allowed response types, e.g. ``["code"]``.
        scope:
            Space-separated scope string requested by the client.
        token_endpoint_auth_method:
            ``"client_secret_basic"``, ``"client_secret_post"``, ``"none"``, ...
        **extra:
            Additional RFC 7591 metadata forwarded verbatim.

        Returns
        -------
        dict
            Server response containing at minimum ``client_id``,
            ``client_secret`` (if confidential), ``registration_access_token``,
            ``registration_client_uri``.

        Raises
        ------
        ~shark_auth.SharkAPIError:
            On any 4xx/5xx response.
        """
        body: Dict[str, Any] = {"client_name": client_name}
        if redirect_uris is not None:
            body["redirect_uris"] = redirect_uris
        if grant_types is not None:
            body["grant_types"] = grant_types
        if response_types is not None:
            body["response_types"] = response_types
        if scope is not None:
            body["scope"] = scope
        if token_endpoint_auth_method is not None:
            body["token_endpoint_auth_method"] = token_endpoint_auth_method
        for k, v in extra.items():
            body[k] = v
        url = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code in (200, 201):
            return resp.json()
        _raise(resp)

    # ------------------------------------------------------------------
    # RFC 7592 — read / update / delete
    # ------------------------------------------------------------------

    def get(self, client_id: str, registration_access_token: str) -> Dict[str, Any]:
        """Fetch the current client metadata (RFC 7592)."""
        url = f"{self._base}{self._PREFIX}/{client_id}"
        resp = _http.request(
            self._session, "GET", url, headers=self._bearer(registration_access_token)
        )
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)

    def update(
        self,
        client_id: str,
        registration_access_token: str,
        **fields: Any,
    ) -> Dict[str, Any]:
        """Replace client metadata via PUT (RFC 7592).

        All fields supplied as kwargs are sent verbatim. RFC 7592 specifies a
        full replacement semantic — pass every metadata field you want kept.
        """
        url = f"{self._base}{self._PREFIX}/{client_id}"
        resp = _http.request(
            self._session,
            "PUT",
            url,
            headers=self._bearer(registration_access_token),
            json=fields,
        )
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)

    def delete(self, client_id: str, registration_access_token: str) -> None:
        """Delete the registered client (RFC 7592)."""
        url = f"{self._base}{self._PREFIX}/{client_id}"
        resp = _http.request(
            self._session,
            "DELETE",
            url,
            headers=self._bearer(registration_access_token),
        )
        if resp.status_code in (200, 204):
            return
        _raise(resp)

    # ------------------------------------------------------------------
    # SharkAuth extensions — secret + registration-token rotation
    # ------------------------------------------------------------------

    def rotate_secret(
        self,
        client_id: str,
        registration_access_token: str,
    ) -> Dict[str, Any]:
        """Rotate the client_secret.

        Backend path verified from ``internal/api/router.go`` line 778:
        ``POST /oauth/register/{client_id}/secret``.

        Returns the new client metadata including the rotated ``client_secret``.
        """
        url = f"{self._base}{self._PREFIX}/{client_id}/secret"
        resp = _http.request(
            self._session,
            "POST",
            url,
            headers=self._bearer(registration_access_token),
        )
        if resp.status_code in (200, 201):
            return resp.json()
        _raise(resp)

    def rotate_registration_token(
        self,
        client_id: str,
        registration_access_token: str,
    ) -> Dict[str, Any]:
        """Rotate the ``registration_access_token``.

        Backend path verified from ``internal/api/router.go`` line 779:
        ``DELETE /oauth/register/{client_id}/registration-token``. The DELETE
        verb invalidates the current token and the response carries the new
        ``registration_access_token`` to use henceforth.
        """
        url = f"{self._base}{self._PREFIX}/{client_id}/registration-token"
        resp = _http.request(
            self._session,
            "DELETE",
            url,
            headers=self._bearer(registration_access_token),
        )
        if resp.status_code in (200, 201):
            return resp.json()
        _raise(resp)


__all__ = ["DCRClient"]
