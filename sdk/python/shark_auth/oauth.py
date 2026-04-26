"""OAuth 2.1 token utilities — revocation (RFC 7009), introspection (RFC 7662),
and DPoP-bound token requests (RFC 9449)."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, Literal, Optional

from . import _http
from .dpop import DPoPProver
from .errors import OAuthError
from .proxy_rules import _raise


@dataclass
class Token:
    """OAuth token returned by the server, optionally DPoP-bound.

    Attributes
    ----------
    access_token:
        The bearer/DPoP access token string.
    token_type:
        Usually ``"DPoP"`` for DPoP-bound tokens, ``"Bearer"`` otherwise.
    expires_in:
        Lifetime in seconds as reported by the server, or ``None``.
    scope:
        Space-separated granted scopes, or ``None``.
    refresh_token:
        Refresh token if returned by the server, or ``None``.
    cnf_jkt:
        JWK thumbprint of the bound DPoP keypair (from ``cnf.jkt``).
        Token theft alone is useless — the holder must also own the private key.
    raw:
        Full server JSON response, for debugging or future claims.
    """

    access_token: str
    token_type: str = "DPoP"
    expires_in: int | None = None
    scope: str | None = None
    refresh_token: str | None = None
    cnf_jkt: str | None = None
    raw: dict = field(default_factory=dict)


class OAuthClient:
    """Client for RFC 7009 token revocation and RFC 7662 token introspection.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    token:
        Admin API key (``sk_live_...``). Required for introspection.
    session:
        Optional pre-configured :class:`requests.Session`.

    Example
    -------
    >>> client = OAuthClient(base_url="https://auth.example.com", token="sk_live_...")
    >>> client.revoke_token("my_access_token")
    >>> info = client.introspect_token("my_access_token")
    >>> print(info["active"])
    """

    def __init__(
        self,
        base_url: str,
        token: str = "",
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._token = token
        self._session = session or _http.new_session()

    def _auth(self) -> Dict[str, str]:
        if self._token:
            return {"Authorization": f"Bearer {self._token}"}
        return {}

    def revoke_token(
        self,
        token: str,
        token_type_hint: Optional[Literal["access_token", "refresh_token"]] = None,
    ) -> None:
        """Revoke a token (RFC 7009).

        The server always returns 200 regardless of whether the token existed.

        Parameters
        ----------
        token:
            The access or refresh token to revoke.
        token_type_hint:
            Optional hint — ``"access_token"`` or ``"refresh_token"``.

        Example
        -------
        >>> client.revoke_token("eyJhbGci...", "access_token")
        """
        data: Dict[str, str] = {"token": token}
        if token_type_hint is not None:
            data["token_type_hint"] = token_type_hint
        url = f"{self._base}/oauth/revoke"
        resp = _http.request(
            self._session, "POST", url, headers=self._auth(), data=data
        )
        if resp.status_code == 200:
            return
        _raise(resp)

    def introspect_token(self, token: str) -> Dict[str, Any]:
        """Introspect a token (RFC 7662).

        Returns a dict with ``active: True`` and claims for valid tokens, or
        ``{"active": False}`` for invalid/expired tokens.

        Parameters
        ----------
        token:
            The token to introspect.

        Example
        -------
        >>> info = client.introspect_token("eyJhbGci...")
        >>> if info["active"]:
        ...     print("sub:", info.get("sub"))
        """
        url = f"{self._base}/oauth/introspect"
        resp = _http.request(
            self._session, "POST", url, headers=self._auth(), data={"token": token}
        )
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)

    # ------------------------------------------------------------------
    # Private helpers
    # ------------------------------------------------------------------

    def _post_token_request(
        self,
        form_body: Dict[str, str],
        dpop_proof: str,
    ) -> Token:
        """POST ``form_body`` to /oauth/token with a DPoP proof header.

        Shared by :meth:`get_token_with_dpop` and :meth:`token_exchange`.
        Parses the 200 response into a :class:`Token`; raises
        :class:`~shark_auth.OAuthError` on 4xx/5xx.
        """
        token_endpoint = f"{self._base}/oauth/token"
        headers = {**self._auth(), "DPoP": dpop_proof}

        resp = _http.request(
            self._session,
            "POST",
            token_endpoint,
            headers=headers,
            data=form_body,
        )

        if resp.status_code == 200:
            data: Dict[str, Any] = resp.json()
            cnf = data.get("cnf") or {}
            return Token(
                access_token=data["access_token"],
                token_type=data.get("token_type", "DPoP"),
                expires_in=data.get("expires_in"),
                scope=data.get("scope"),
                refresh_token=data.get("refresh_token"),
                cnf_jkt=cnf.get("jkt"),
                raw=data,
            )

        # Parse RFC 6749 error body when possible.
        try:
            err_body = resp.json()
            error = err_body.get("error", "token_request_failed")
            error_description = err_body.get("error_description")
        except Exception:
            error = "token_request_failed"
            error_description = resp.text or None

        raise OAuthError(
            error=error,
            error_description=error_description,
            status_code=resp.status_code,
        )

    # ------------------------------------------------------------------
    # Public methods
    # ------------------------------------------------------------------

    def get_token_with_dpop(
        self,
        *,
        grant_type: str,
        dpop_prover: DPoPProver,
        client_id: str,
        client_secret: str | None = None,
        scope: str | None = None,
        **extra: Any,
    ) -> Token:
        """Request an OAuth token with a DPoP proof header.

        Wraps POST /oauth/token. The DPoP proof is signed with the prover's
        keypair and bound to the request method+URL. The returned token has
        ``cnf.jkt`` matching the prover's public-key thumbprint — token theft
        alone is useless.

        Supports: ``client_credentials``, ``authorization_code``,
        ``refresh_token`` grants.

        Parameters
        ----------
        grant_type:
            OAuth 2.1 grant type, e.g. ``"client_credentials"``.
        dpop_prover:
            A :class:`~shark_auth.DPoPProver` instance holding the keypair used
            to sign the DPoP proof JWT.
        client_id:
            The client / agent identifier.
        client_secret:
            Optional client secret (omit for public clients).
        scope:
            Space-separated scopes to request. ``None`` omits the parameter.
        **extra:
            Any additional form fields forwarded to the token endpoint
            (e.g. ``code``, ``redirect_uri``, ``refresh_token``).

        Returns
        -------
        Token:
            Populated :class:`Token` dataclass. ``cnf_jkt`` is set when the
            server embeds ``cnf.jkt`` in the token response.

        Raises
        ------
        OAuthError:
            On any 4xx or 5xx response from the token endpoint.

        Example
        -------
        >>> prover = DPoPProver.generate()
        >>> client = OAuthClient(base_url="https://auth.example.com")
        >>> token = client.get_token_with_dpop(
        ...     grant_type="client_credentials",
        ...     dpop_prover=prover,
        ...     client_id="my-agent",
        ...     client_secret="s3cr3t",
        ...     scope="read write",
        ... )
        >>> print(token.access_token)
        >>> print(token.cnf_jkt == prover.jkt)  # True
        """
        token_endpoint = f"{self._base}/oauth/token"

        # Generate DPoP proof bound to this exact request.
        proof = dpop_prover.make_proof(htm="POST", htu=token_endpoint)

        # Build form body.
        body: Dict[str, str] = {"grant_type": grant_type, "client_id": client_id}
        if client_secret is not None:
            body["client_secret"] = client_secret
        if scope is not None:
            body["scope"] = scope
        for k, v in extra.items():
            body[k] = str(v)

        return self._post_token_request(body, proof)

    def token_exchange(
        self,
        *,
        subject_token: str,
        dpop_prover: DPoPProver,
        scope: str | None = None,
        audience: str | None = None,
        actor_token: str | None = None,
        subject_token_type: str = "urn:ietf:params:oauth:token-type:access_token",
        requested_token_type: str = "urn:ietf:params:oauth:token-type:access_token",
        **extra: Any,
    ) -> Token:
        """RFC 8693 OAuth Token Exchange.

        Issues a downscoped or audience-restricted token derived from
        ``subject_token``. The new token is DPoP-bound to the same prover
        keypair (``cnf.jkt`` matches). Use this for delegation chains: agent A
        receives a token, narrows it to a sub-scope, hands it to agent B (which
        carries the ``act``-claim chain proving the original human's
        authorization).

        Parameters
        ----------
        subject_token:
            The existing access token to exchange (e.g. ``token.access_token``).
        dpop_prover:
            Same :class:`~shark_auth.DPoPProver` used for the original token
            (key binding preserved).
        scope:
            Narrower scope to downscope to (e.g. ``"mcp:read"`` from
            ``"mcp:write mcp:read"``). ``None`` omits the parameter.
        audience:
            Restrict the new token to a specific resource server. ``None``
            omits the parameter.
        actor_token:
            Optional ``act``-claim parent (the agent doing the delegation).
            When provided, ``actor_token_type`` is automatically included.
        subject_token_type:
            RFC 8693 type URI for ``subject_token``.
            Defaults to ``"urn:ietf:params:oauth:token-type:access_token"``.
        requested_token_type:
            RFC 8693 type URI for the token to be issued.
            Defaults to ``"urn:ietf:params:oauth:token-type:access_token"``.
        **extra:
            Any additional form fields forwarded to the token endpoint.

        Returns
        -------
        Token:
            Populated :class:`Token` dataclass with new ``access_token``.
            ``scope`` / ``audience`` reflect the narrowing; ``cnf_jkt`` is
            unchanged from the original (same keypair).

        Raises
        ------
        OAuthError:
            On any 4xx or 5xx response, e.g. ``"invalid_token"`` if
            ``subject_token`` is revoked, or ``"invalid_scope"`` if the
            requested scope exceeds the parent's grant.

        Example
        -------
        >>> prover = DPoPProver.generate()
        >>> client = OAuthClient(base_url="https://auth.example.com")
        >>> parent = client.get_token_with_dpop(
        ...     grant_type="client_credentials",
        ...     dpop_prover=prover,
        ...     client_id="agent-a",
        ...     client_secret="s3cr3t",
        ...     scope="mcp:read mcp:write",
        ... )
        >>> child = client.token_exchange(
        ...     subject_token=parent.access_token,
        ...     dpop_prover=prover,
        ...     scope="mcp:read",
        ...     audience="https://mcp.example.com",
        ... )
        >>> assert child.cnf_jkt == parent.cnf_jkt  # same keypair bound
        >>> print(child.scope)  # "mcp:read"
        """
        token_endpoint = f"{self._base}/oauth/token"

        # Generate DPoP proof bound to this exact request.
        proof = dpop_prover.make_proof(htm="POST", htu=token_endpoint)

        # Build RFC 8693 form body.
        body: Dict[str, str] = {
            "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
            "subject_token": subject_token,
            "subject_token_type": subject_token_type,
            "requested_token_type": requested_token_type,
        }
        if scope is not None:
            body["scope"] = scope
        if audience is not None:
            body["audience"] = audience
        if actor_token is not None:
            body["actor_token"] = actor_token
            body["actor_token_type"] = "urn:ietf:params:oauth:token-type:access_token"
        for k, v in extra.items():
            body[k] = str(v)

        return self._post_token_request(body, proof)
