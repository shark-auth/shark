"""High-level session for agents with automatic DPoP signing."""

from __future__ import annotations

from typing import Optional

import requests

from .dpop import DPoPProver


class AgentSession(requests.Session):
    """A requests-compatible session that auto-signs every request with DPoP.

    Wraps a :class:`DPoPProver` and an access token to automatically inject
    the ``Authorization: DPoP <token>`` and ``DPoP: <proof>`` headers into
    every outgoing request.

    Example
    -------
    >>> prover = DPoPProver.generate()
    >>> session = AgentSession(prover, "agent_token_abc")
    >>> r = session.get("https://api.example/data")
    """

    def __init__(
        self,
        prover: DPoPProver,
        access_token: str,
        *,
        user_agent: str = "shark-auth-python/0.1.0",
    ) -> None:
        super().__init__()
        self.prover = prover
        self.access_token = access_token
        self.headers.update({"User-Agent": user_agent, "Accept": "application/json"})

    def request(self, method: str, url: str, *args, **kwargs) -> requests.Response:
        """Generate a DPoP proof and inject headers before calling super()."""
        # Generate the RFC 9449 proof for this specific method and URL
        proof = self.prover.make_proof(
            method,
            url,
            access_token=self.access_token,
        )

        # Inject the standard DPoP headers
        headers = kwargs.setdefault("headers", {})
        headers["Authorization"] = f"DPoP {self.access_token}"
        headers["DPoP"] = proof

        return super().request(method, url, *args, **kwargs)
