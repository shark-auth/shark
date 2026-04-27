"""RFC 8628 OAuth 2.0 Device Authorization Grant client."""

from __future__ import annotations

import time
from dataclasses import dataclass
from typing import Optional

from . import _http
from .dpop import DPoPProver
from .errors import DeviceFlowError

DEVICE_GRANT = "urn:ietf:params:oauth:grant-type:device_code"


@dataclass
class DeviceInit:
    """Response to the device authorization request."""

    device_code: str
    user_code: str
    verification_uri: str
    verification_uri_complete: Optional[str]
    expires_in: int
    interval: int


@dataclass
class TokenResponse:
    """Access token response returned by the auth server."""

    access_token: str
    token_type: str
    expires_in: Optional[int] = None
    refresh_token: Optional[str] = None
    scope: Optional[str] = None


class DeviceFlow:
    """Runs the RFC 8628 device authorization grant.

    Example
    -------
    >>> flow = DeviceFlow(auth_url="https://auth.example", client_id="agent_xyz")
    >>> init = flow.begin()
    >>> print(init.verification_uri, init.user_code)
    >>> token = flow.wait_for_approval(timeout_s=300)
    """

    def __init__(
        self,
        auth_url: str,
        client_id: str,
        scope: Optional[str] = None,
        *,
        dpop_prover: Optional[DPoPProver] = None,
        # Backend mounts the RFC 8628 device-authorization endpoint at
        # /oauth/device (see internal/api/router.go:784) — NOT the literal
        # /oauth/device_authorization spec path.  Default updated to match.
        device_authorization_path: str = "/oauth/device",
        token_path: str = "/oauth/token",
        session: Optional[object] = None,
    ) -> None:
        self.auth_url = auth_url.rstrip("/")
        self.client_id = client_id
        self.scope = scope
        self.dpop_prover = dpop_prover
        self._device_auth_url = self.auth_url + device_authorization_path
        self._token_url = self.auth_url + token_path
        self._session = session or _http.new_session()
        self._init: Optional[DeviceInit] = None

    # ------------------------------------------------------------------
    def begin(self) -> DeviceInit:
        """Initiate the device authorization request."""
        data = {"client_id": self.client_id}
        if self.scope:
            data["scope"] = self.scope
        resp = _http.request(
            self._session, "POST", self._device_auth_url, data=data
        )
        if resp.status_code != 200:
            raise DeviceFlowError(
                f"device_authorization failed: HTTP {resp.status_code}: {resp.text}"
            )
        body = resp.json()
        required = ("device_code", "user_code", "verification_uri", "expires_in")
        missing = [k for k in required if k not in body]
        if missing:
            raise DeviceFlowError(f"device_authorization missing keys: {missing}")
        init = DeviceInit(
            device_code=body["device_code"],
            user_code=body["user_code"],
            verification_uri=body["verification_uri"],
            verification_uri_complete=body.get("verification_uri_complete"),
            expires_in=int(body["expires_in"]),
            interval=int(body.get("interval", 5)),
        )
        self._init = init
        return init

    # ------------------------------------------------------------------
    def wait_for_approval(
        self,
        timeout_s: float = 300.0,
        *,
        clock: Optional[callable] = None,
        sleeper: Optional[callable] = None,
    ) -> TokenResponse:
        """Poll ``/oauth/token`` until the user approves or the flow errors.

        Handles ``authorization_pending`` (continue), ``slow_down``
        (increase interval by 5s per RFC 8628 §3.5), ``access_denied``
        and ``expired_token`` (raise).

        ``clock`` and ``sleeper`` exist for deterministic testing.
        """
        if self._init is None:
            raise DeviceFlowError("begin() must be called before wait_for_approval()")

        now = clock or time.monotonic
        sleep = sleeper or time.sleep

        deadline = now() + timeout_s
        interval = max(1, int(self._init.interval))

        while True:
            if now() >= deadline:
                raise DeviceFlowError("device flow timed out before approval")

            body = {
                "grant_type": DEVICE_GRANT,
                "device_code": self._init.device_code,
                "client_id": self.client_id,
            }
            headers = {}
            if self.dpop_prover is not None:
                headers["DPoP"] = self.dpop_prover.make_proof("POST", self._token_url)

            resp = _http.request(
                self._session,
                "POST",
                self._token_url,
                data=body,
                headers=headers,
            )
            try:
                payload = resp.json()
            except ValueError:
                raise DeviceFlowError(
                    f"token endpoint returned non-JSON: HTTP {resp.status_code}: {resp.text[:200]}"
                )

            if resp.status_code == 200:
                return TokenResponse(
                    access_token=payload["access_token"],
                    token_type=payload.get("token_type", "Bearer"),
                    expires_in=payload.get("expires_in"),
                    refresh_token=payload.get("refresh_token"),
                    scope=payload.get("scope"),
                )

            err = payload.get("error")
            if err == "authorization_pending":
                pass
            elif err == "slow_down":
                interval += 5
            elif err == "access_denied":
                raise DeviceFlowError("user denied the authorization request")
            elif err == "expired_token":
                raise DeviceFlowError("device code expired before user approved")
            elif err == "invalid_client":
                raise DeviceFlowError("invalid client_id")
            else:
                raise DeviceFlowError(
                    f"token endpoint error: {err or 'unknown'} "
                    f"(HTTP {resp.status_code}): {payload.get('error_description', '')}"
                )

            remaining = deadline - now()
            if remaining <= 0:
                raise DeviceFlowError("device flow timed out before approval")
            sleep(min(interval, remaining))
