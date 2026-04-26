"""DPoP-authenticated HTTP helpers for shark-protected resources.

Provides :class:`DPoPHTTPClient` which wraps ``requests`` with automatic
DPoP proof generation per RFC 9449.  Use this for any endpoint that
requires ``Authorization: DPoP <token>`` rather than a plain bearer token.
"""

from __future__ import annotations

import hashlib
import base64
from typing import Any, Optional

import requests

from .dpop import DPoPProver
from . import _http


def _ath(token: str) -> str:
    """Compute ath = base64url(sha256(token)) without padding (RFC 9449 §4.3)."""
    digest = hashlib.sha256(token.encode()).digest()
    return base64.urlsafe_b64encode(digest).rstrip(b"=").decode()


class DPoPHTTPClient:
    """HTTP client that attaches a fresh DPoP proof to every request.

    Parameters
    ----------
    base_url:
        Root URL of the resource server (e.g. ``https://auth.example.com``).
        Trailing slash is stripped automatically.
    timeout:
        Default request timeout in seconds.
    session:
        Optional shared :class:`requests.Session`.  When omitted a new
        session is created.

    Example
    -------
    >>> prover = DPoPProver.generate()
    >>> client = DPoPHTTPClient("https://auth.example.com")
    >>> resp = client.get_with_dpop("/api/v1/auth/me", token=at, prover=prover)
    >>> resp.raise_for_status()
    """

    def __init__(
        self,
        base_url: str,
        *,
        timeout: float = 30.0,
        session: Optional[requests.Session] = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._session = session or _http.new_session()

    # ------------------------------------------------------------------
    # Public helpers
    # ------------------------------------------------------------------

    def get_with_dpop(
        self,
        path: str,
        *,
        token: str,
        prover: DPoPProver,
        **kwargs: Any,
    ) -> requests.Response:
        """HTTP GET with DPoP proof attached."""
        return self._request("GET", path, token=token, prover=prover, **kwargs)

    def post_with_dpop(
        self,
        path: str,
        *,
        token: str,
        prover: DPoPProver,
        json: Any = None,
        **kwargs: Any,
    ) -> requests.Response:
        """HTTP POST with DPoP proof attached."""
        return self._request("POST", path, token=token, prover=prover, json=json, **kwargs)

    def delete_with_dpop(
        self,
        path: str,
        *,
        token: str,
        prover: DPoPProver,
        **kwargs: Any,
    ) -> requests.Response:
        """HTTP DELETE with DPoP proof attached."""
        return self._request("DELETE", path, token=token, prover=prover, **kwargs)

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    def _request(
        self,
        method: str,
        path: str,
        *,
        token: str,
        prover: DPoPProver,
        **kwargs: Any,
    ) -> requests.Response:
        url = (
            f"{self.base_url}{path}"
            if path.startswith("/")
            else f"{self.base_url}/{path}"
        )
        # RFC 9449 §4.3: ath claim binds the proof to the access token.
        proof = prover.make_proof(htm=method, htu=url, access_token=token)

        headers: dict = kwargs.pop("headers", {}) or {}
        headers["Authorization"] = f"DPoP {token}"
        headers["DPoP"] = proof

        timeout = kwargs.pop("timeout", self._timeout)
        return self._session.request(
            method, url, headers=headers, timeout=timeout, **kwargs
        )

    def close(self) -> None:
        """Close the underlying session."""
        self._session.close()

    def __enter__(self) -> "DPoPHTTPClient":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()
