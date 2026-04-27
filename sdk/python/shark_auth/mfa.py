"""MFA (TOTP + recovery codes) — wraps ``/api/v1/auth/mfa/*``.

Includes a stdlib-only RFC 6238 :func:`compute_totp` so callers can drive
``enroll`` -> ``verify`` end-to-end in tests without ``pyotp``.
"""

from __future__ import annotations

import base64
import hashlib
import hmac
import struct
import time as _time
from typing import Any, Dict, List, Optional

from . import _http
from .auth import _raise_auth


# ---------------------------------------------------------------------------
# RFC 6238 TOTP — pure stdlib helper
# ---------------------------------------------------------------------------


def compute_totp(secret: str, t: Optional[float] = None) -> str:
    """Compute a 6-digit RFC 6238 TOTP code.

    Uses HMAC-SHA1 over a 30-second time-step (the SharkAuth default).  The
    secret is base32-encoded, matching what the server returns from
    :meth:`MFAClient.enroll` and embeds in the ``otpauth://`` QR URI.

    Parameters
    ----------
    secret:
        Base32-encoded shared secret (case-insensitive, optional ``=`` padding).
    t:
        Optional UNIX timestamp; defaults to ``time.time()``.

    Returns
    -------
    str
        Zero-padded 6-digit code.

    Example
    -------
    >>> mfa = MFAClient("http://localhost:8080", session=auth_session)
    >>> enroll = mfa.enroll()
    >>> mfa.verify(compute_totp(enroll["secret"]))
    """
    if t is None:
        t = _time.time()
    # Pad to a multiple of 8 chars before base32-decoding.
    s = secret.upper().replace(" ", "")
    pad = (-len(s)) % 8
    key = base64.b32decode(s + ("=" * pad))
    counter = int(t // 30)
    msg = struct.pack(">Q", counter)
    digest = hmac.new(key, msg, hashlib.sha1).digest()
    offset = digest[-1] & 0x0F
    code_int = (
        ((digest[offset] & 0x7F) << 24)
        | ((digest[offset + 1] & 0xFF) << 16)
        | ((digest[offset + 2] & 0xFF) << 8)
        | (digest[offset + 3] & 0xFF)
    ) % 1_000_000
    return f"{code_int:06d}"


# ---------------------------------------------------------------------------
# MFA client
# ---------------------------------------------------------------------------


class MFAClient:
    """Manage TOTP MFA on the authenticated user's account.

    All endpoints assume the underlying :class:`requests.Session` already
    carries a valid session cookie (typically planted by
    :meth:`AuthClient.login`).

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server.
    session:
        The :class:`requests.Session` carrying the user's session cookie.

    Example
    -------
    >>> auth = AuthClient("http://localhost:8080")
    >>> auth.login("alice@test.local", "...")
    >>> mfa = MFAClient("http://localhost:8080", session=auth._session)
    >>> e = mfa.enroll()
    >>> mfa.verify(compute_totp(e["secret"]))
    """

    _PREFIX = "/api/v1/auth/mfa"

    def __init__(
        self,
        base_url: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._session = session or _http.new_session()

    # ------------------------------------------------------------------
    # Enroll / Verify (full-session endpoints)
    # ------------------------------------------------------------------

    def enroll(self) -> Dict[str, Any]:
        """Generate a TOTP secret + QR provisioning URI.

        Returns ``{"secret": ..., "qr_uri": ..., "otpauth_url": ...}``.  The
        server returns ``secret`` and ``qr_uri``; ``otpauth_url`` is mirrored
        from ``qr_uri`` for naming-compat with other SDKs.
        """
        url = f"{self._base}{self._PREFIX}/enroll"
        resp = _http.request(self._session, "POST", url)
        if resp.status_code == 200:
            data = resp.json()
            # Mirror qr_uri -> otpauth_url for caller convenience.
            if "otpauth_url" not in data and "qr_uri" in data:
                data["otpauth_url"] = data["qr_uri"]
            return data
        _raise_auth(resp)

    def verify(self, code: str) -> Dict[str, Any]:
        """Confirm enrollment with the first TOTP code.

        Returns ``{"mfa_enabled": True, "recovery_codes": [...]}``.
        """
        body = {"code": code}
        url = f"{self._base}{self._PREFIX}/verify"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code == 200:
            return resp.json()
        _raise_auth(resp)

    # ------------------------------------------------------------------
    # Login-time challenge (partial session) + disable + recovery codes
    # ------------------------------------------------------------------

    def challenge(self, code: str) -> None:
        """Upgrade a partial (post-login) session by submitting a TOTP code.

        Wraps ``POST /challenge``.  The server upgrades the session's
        ``mfa_passed`` flag in place — no response body to consume.
        """
        body = {"code": code}
        url = f"{self._base}{self._PREFIX}/challenge"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    def disable(self, code: str) -> None:
        """Disable MFA for the authenticated user.

        Wraps ``DELETE /api/v1/auth/mfa`` — the user must re-prove possession
        of the device by supplying a current TOTP code.
        """
        body = {"code": code}
        url = f"{self._base}{self._PREFIX}/"
        resp = _http.request(self._session, "DELETE", url, json=body)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    def regenerate_recovery_codes(self) -> List[str]:
        """Regenerate the current user's MFA recovery codes.

        Wraps ``GET /api/v1/auth/mfa/recovery-codes``.  Note: the server
        exposes this as ``GET`` (not POST) — calling it invalidates the
        previously-issued set and returns a fresh list.
        """
        url = f"{self._base}{self._PREFIX}/recovery-codes"
        resp = _http.request(self._session, "GET", url)
        if resp.status_code == 200:
            data = resp.json()
            # Server may return {"recovery_codes": [...]} or a bare list.
            if isinstance(data, dict):
                codes = data.get("recovery_codes", data.get("codes", []))
                return list(codes)
            return list(data) if isinstance(data, list) else []
        _raise_auth(resp)
