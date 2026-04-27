"""Webhooks admin client + HMAC-SHA256 signature verification helper.

Endpoints under ``/api/v1/admin/webhooks/`` (verified in
``internal/api/router.go`` lines 510-522):

- ``POST   /``                                — create
- ``GET    /``                                — list
- ``GET    /events``                          — list event types
- ``GET    /{id}``                            — get
- ``PATCH  /{id}``                            — update
- ``DELETE /{id}``                            — delete
- ``POST   /{id}/test``                       — fire a test delivery
- ``GET    /{id}/deliveries``                 — list past deliveries
- ``POST   /{id}/deliveries/{deliveryId}/replay`` — replay a delivery
"""

from __future__ import annotations

import hashlib
import hmac
import time
from typing import Any, Dict, List, Optional

from . import _http
from .proxy_rules import _raise


# ---------------------------------------------------------------------------
# Signature verification — module-level helper
# ---------------------------------------------------------------------------


def verify_signature(
    payload: bytes,
    header_signature: str,
    secret: str,
    tolerance_seconds: int = 300,
) -> bool:
    """Verify an HMAC-SHA256 signature on an incoming webhook payload.

    Two header formats are supported:

    1. Stripe-style: ``t=<unix_ts>,v1=<hex_hmac>`` (optionally with extra
       ``vN=...`` segments — only ``v1`` is checked). Signed payload is
       ``f"{ts}.".encode() + payload``. The timestamp is also enforced
       against ``tolerance_seconds`` to thwart replay attacks.
    2. Raw hex digest of HMAC-SHA256(secret, payload). No timestamp enforced.

    Uses :func:`hmac.compare_digest` for timing-safe comparison.

    Parameters
    ----------
    payload:
        Raw request body bytes (DO NOT pass JSON-decoded, must match what the
        server signed byte-for-byte).
    header_signature:
        Value of the signature header sent by SharkAuth.
    secret:
        Shared HMAC secret (the value returned/configured at webhook creation).
    tolerance_seconds:
        Max age of the timestamp in the Stripe-style format. Default 300s.

    Returns
    -------
    bool
        ``True`` if the signature is valid (and, for the timestamped form, the
        timestamp is within tolerance). ``False`` otherwise.

    Example
    -------
    >>> import hmac, hashlib, time
    >>> secret = "whsec_xxx"
    >>> body = b'{"event":"user.created"}'
    >>> ts = int(time.time())
    >>> sig = hmac.new(secret.encode(), f"{ts}.".encode() + body,
    ...                hashlib.sha256).hexdigest()
    >>> header = f"t={ts},v1={sig}"
    >>> assert verify_signature(body, header, secret)
    """
    if not header_signature or not isinstance(header_signature, str):
        return False
    secret_bytes = secret.encode("utf-8") if isinstance(secret, str) else secret

    # Format 1 — comma-separated key=value with t= and v1=.
    if "=" in header_signature and "," in header_signature:
        parts = {}
        for segment in header_signature.split(","):
            segment = segment.strip()
            if "=" in segment:
                k, _, v = segment.partition("=")
                parts[k.strip()] = v.strip()
        ts = parts.get("t")
        v1 = parts.get("v1")
        if ts is None or v1 is None:
            return False
        try:
            ts_int = int(ts)
        except ValueError:
            return False
        # Replay protection.
        if tolerance_seconds > 0 and abs(int(time.time()) - ts_int) > tolerance_seconds:
            return False
        signed = f"{ts}.".encode("utf-8") + payload
        expected = hmac.new(secret_bytes, signed, hashlib.sha256).hexdigest()
        return hmac.compare_digest(expected, v1)

    # Format 2 — raw hex digest (no timestamp; tolerance not enforced).
    expected = hmac.new(secret_bytes, payload, hashlib.sha256).hexdigest()
    # Strip an optional 'sha256=' prefix sometimes used.
    candidate = header_signature.strip()
    if candidate.lower().startswith("sha256="):
        candidate = candidate.split("=", 1)[1]
    return hmac.compare_digest(expected, candidate)


# ---------------------------------------------------------------------------
# Admin client
# ---------------------------------------------------------------------------


class WebhooksClient:
    """Admin client for webhook endpoints.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    admin_api_key:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.

    Example
    -------
    >>> wh = WebhooksClient(base_url="https://auth.example.com",
    ...                     admin_api_key="sk_live_xxx")
    >>> created = wh.register(
    ...     url="https://app.example.com/hooks",
    ...     events=["user.created", "user.deleted"],
    ...     secret="whsec_xxx",
    ... )
    """

    _PREFIX = "/api/v1/admin/webhooks"

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
    # CRUD
    # ------------------------------------------------------------------

    def register(
        self,
        url: str,
        events: List[str],
        *,
        secret: Optional[str] = None,
        **extra: Any,
    ) -> Dict[str, Any]:
        """Create a webhook subscription.

        Parameters
        ----------
        url:
            HTTPS endpoint that will receive POSTed events.
        events:
            List of event-type strings to subscribe to.
        secret:
            Optional HMAC secret for payload signing. If omitted the server
            may auto-generate one and return it in the response.
        **extra:
            Additional fields forwarded verbatim (e.g. ``description``,
            ``enabled``).
        """
        body: Dict[str, Any] = {"url": url, "events": events}
        if secret is not None:
            body["secret"] = secret
        for k, v in extra.items():
            body[k] = v
        endpoint = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "POST", endpoint, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return resp.json()
        _raise(resp)

    def list(self) -> List[Dict[str, Any]]:
        """List all webhook subscriptions."""
        endpoint = f"{self._base}{self._PREFIX}"
        resp = _http.request(self._session, "GET", endpoint, headers=self._auth())
        if resp.status_code == 200:
            data = resp.json()
            if isinstance(data, dict):
                return data.get("data", [])
            return data
        _raise(resp)

    def get(self, webhook_id: str) -> Dict[str, Any]:
        """Fetch a single webhook subscription."""
        endpoint = f"{self._base}{self._PREFIX}/{webhook_id}"
        resp = _http.request(self._session, "GET", endpoint, headers=self._auth())
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)

    def update(self, webhook_id: str, **fields: Any) -> Dict[str, Any]:
        """Patch a webhook subscription.

        Backend uses ``PATCH /{id}`` (verified in router.go line 517).
        """
        endpoint = f"{self._base}{self._PREFIX}/{webhook_id}"
        resp = _http.request(
            self._session, "PATCH", endpoint, headers=self._auth(), json=fields
        )
        if resp.status_code == 200:
            return resp.json()
        _raise(resp)

    def delete(self, webhook_id: str) -> None:
        """Delete a webhook subscription."""
        endpoint = f"{self._base}{self._PREFIX}/{webhook_id}"
        resp = _http.request(self._session, "DELETE", endpoint, headers=self._auth())
        if resp.status_code in (200, 204):
            return
        _raise(resp)

    # ------------------------------------------------------------------
    # Test + deliveries
    # ------------------------------------------------------------------

    def send_test(
        self,
        webhook_id: str,
        *,
        event_type: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Fire a synthetic test event.

        Backend path: ``POST /{id}/test`` (verified in router.go line 519 —
        note the path is ``/test``, not ``/send-test``).
        """
        endpoint = f"{self._base}{self._PREFIX}/{webhook_id}/test"
        body: Dict[str, Any] = {}
        if event_type is not None:
            body["event_type"] = event_type
        resp = _http.request(
            self._session,
            "POST",
            endpoint,
            headers=self._auth(),
            json=body if body else None,
        )
        if resp.status_code in (200, 201, 202):
            try:
                return resp.json()
            except Exception:
                return {}
        _raise(resp)

    def replay(self, webhook_id: str, delivery_id: str) -> Dict[str, Any]:
        """Replay a previous delivery."""
        endpoint = (
            f"{self._base}{self._PREFIX}/{webhook_id}/deliveries/{delivery_id}/replay"
        )
        resp = _http.request(self._session, "POST", endpoint, headers=self._auth())
        if resp.status_code in (200, 201, 202):
            try:
                return resp.json()
            except Exception:
                return {}
        _raise(resp)

    def list_deliveries(
        self,
        webhook_id: str,
        *,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """List past deliveries for a webhook."""
        endpoint = f"{self._base}{self._PREFIX}/{webhook_id}/deliveries"
        resp = _http.request(
            self._session,
            "GET",
            endpoint,
            headers=self._auth(),
            params={"limit": limit},
        )
        if resp.status_code == 200:
            data = resp.json()
            if isinstance(data, dict):
                return data.get("data", [])
            return data
        _raise(resp)


__all__ = ["WebhooksClient", "verify_signature"]
