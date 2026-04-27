"""Human-auth client — signup / login / logout / me / password / email verify / magic link.

Wraps the public ``/api/v1/auth/...`` routes.  No DPoP required — these
endpoints use cookie-based sessions plus optional bearer-JWT.

The :class:`AuthClient` keeps a :class:`requests.Session` so the session
cookie planted by ``/login`` (and ``/signup``) flows through to follow-up
calls like :meth:`get_me`, :meth:`change_password`, etc.
"""

from __future__ import annotations

from typing import Any, Dict, Optional

from . import _http
from .errors import SharkAuthError


def _raise_auth(resp) -> None:
    """Raise SharkAuthError with backend ``error`` + ``message`` fields when present."""
    try:
        body = resp.json()
    except ValueError:
        body = {}
    err = body.get("error") if isinstance(body, dict) else None
    msg = body.get("message") if isinstance(body, dict) else None
    detail = f"{err}: {msg}" if err and msg else err or msg or resp.text[:200]
    raise SharkAuthError(f"HTTP {resp.status_code}: {detail}")


class AuthClient:
    """Public human-auth client.

    Wraps the password / magic-link / email-verify / password-reset endpoints
    under ``/api/v1/auth/``.  Stateful — the underlying ``requests.Session``
    receives the ``shark_session`` cookie on a successful :meth:`login` or
    :meth:`signup`, so follow-up calls (e.g. :meth:`get_me`,
    :meth:`change_password`, :meth:`request_email_verification`) authenticate
    automatically.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server (e.g. ``"https://auth.example.com"``).
    session:
        Optional pre-configured :class:`requests.Session`.  Provide one to
        share cookies with other clients in the same workflow.

    Example
    -------
    >>> client = AuthClient("http://localhost:8080")
    >>> client.signup(email="alice@test.local", password="HuntER!2026", full_name="Alice")
    >>> client.login("alice@test.local", "HuntER!2026")
    >>> client.get_me()["email"]
    'alice@test.local'
    """

    _PREFIX = "/api/v1/auth"

    def __init__(
        self,
        base_url: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._session = session or _http.new_session()

    # ------------------------------------------------------------------
    # Signup / Login / Logout
    # ------------------------------------------------------------------

    def signup(
        self,
        email: str,
        password: str,
        full_name: Optional[str] = None,
        **extra: Any,
    ) -> Dict[str, Any]:
        """Create a new user account.

        On success the server plants the session cookie on this client's
        :class:`requests.Session` and returns the new user object.

        Parameters
        ----------
        email:
            Login email (lowercased server-side).
        password:
            Plaintext password — must satisfy server-configured complexity.
        full_name:
            Optional display name.  Mapped to the server's ``name`` field.
        **extra:
            Additional fields forwarded verbatim.

        Raises
        ------
        SharkAuthError:
            On any non-2xx response.
        """
        body: Dict[str, Any] = {"email": email, "password": password}
        if full_name is not None:
            # Server's request struct uses "name", not "full_name".
            body["name"] = full_name
        body.update(extra)
        url = f"{self._base}{self._PREFIX}/signup"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code in (200, 201):
            return resp.json()
        _raise_auth(resp)

    def login(self, email: str, password: str) -> Dict[str, Any]:
        """Authenticate and plant the session cookie on the underlying session.

        Returns the user object as returned by the server.  When MFA is
        enabled the server returns ``{"mfaRequired": True}`` instead and the
        caller must follow up with :meth:`MFAClient.challenge`.
        """
        body = {"email": email, "password": password}
        url = f"{self._base}{self._PREFIX}/login"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code == 200:
            return resp.json()
        _raise_auth(resp)

    def logout(self) -> None:
        """Revoke the current session.  204 on success."""
        url = f"{self._base}{self._PREFIX}/logout"
        resp = _http.request(self._session, "POST", url)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    # ------------------------------------------------------------------
    # /me
    # ------------------------------------------------------------------

    def get_me(self) -> Dict[str, Any]:
        """Return the authenticated user (``GET /me``)."""
        url = f"{self._base}{self._PREFIX}/me"
        resp = _http.request(self._session, "GET", url)
        if resp.status_code == 200:
            return resp.json()
        _raise_auth(resp)

    def delete_me(self) -> None:
        """Delete the authenticated user (``DELETE /me``).  Requires verified email."""
        url = f"{self._base}{self._PREFIX}/me"
        resp = _http.request(self._session, "DELETE", url)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    # ------------------------------------------------------------------
    # Password management
    # ------------------------------------------------------------------

    def change_password(self, old: str, new: str) -> None:
        """Change the authenticated user's password.

        Wraps ``POST /password/change``.  Requires a fully authenticated
        session with verified email.
        """
        body = {"current_password": old, "new_password": new}
        url = f"{self._base}{self._PREFIX}/password/change"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    def request_password_reset(self, email: str) -> None:
        """Send a password reset email.

        Always returns success — the server intentionally does not reveal
        whether the email exists.
        """
        body = {"email": email}
        url = f"{self._base}{self._PREFIX}/password/send-reset-link"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    def confirm_password_reset(self, token: str, new_password: str) -> None:
        """Complete a password reset with the token from the reset email."""
        body = {"token": token, "password": new_password}
        url = f"{self._base}{self._PREFIX}/password/reset"
        resp = _http.request(self._session, "POST", url, json=body)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    # ------------------------------------------------------------------
    # Email verification
    # ------------------------------------------------------------------

    def request_email_verification(self) -> None:
        """Send the email-verification link to the authenticated user.

        Wraps ``POST /email/verify/send``.  Uses the session cookie, so
        :meth:`login` (or :meth:`signup`) must precede.
        """
        url = f"{self._base}{self._PREFIX}/email/verify/send"
        resp = _http.request(self._session, "POST", url)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    def consume_email_verification(self, token: str) -> None:
        """Consume an email-verification token (``GET /email/verify?token=...``)."""
        url = f"{self._base}{self._PREFIX}/email/verify"
        resp = _http.request(self._session, "GET", url, params={"token": token})
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)

    # ------------------------------------------------------------------
    # Magic link verify (companion to MagicLinkClient.send_magic_link)
    # ------------------------------------------------------------------

    def verify_magic_link(self, token: str) -> Dict[str, Any]:
        """Consume a magic-link token and return the user object.

        Wraps ``GET /magic-link/verify?token=...``.  On success the server
        plants the session cookie on this client's session.
        """
        url = f"{self._base}{self._PREFIX}/magic-link/verify"
        resp = _http.request(self._session, "GET", url, params={"token": token})
        if resp.status_code == 200:
            return resp.json()
        _raise_auth(resp)

    # ------------------------------------------------------------------
    # Permission check
    # ------------------------------------------------------------------

    def check(self, action: str, resource: str) -> Dict[str, Any]:
        """Check whether the authenticated principal has permission for *action* on *resource*.

        Wraps ``POST /api/v1/auth/check``.  Requires a valid session or bearer token.

        Parameters
        ----------
        action:
            The action to check (e.g. ``"read"``, ``"write"``).
        resource:
            The resource to check against (e.g. ``"documents:123"``).

        Returns
        -------
        dict
            Server response — typically ``{"allowed": True/False}``.
        """
        url = f"{self._base}{self._PREFIX}/check"
        resp = _http.request(self._session, "POST", url, json={"action": action, "resource": resource})
        if resp.status_code == 200:
            return resp.json()
        _raise_auth(resp)

    # ------------------------------------------------------------------
    # Self-revoke
    # ------------------------------------------------------------------

    def revoke_self(self) -> None:
        """Revoke the calling user's own JWT / session.

        Wraps ``POST /api/v1/auth/revoke``.  Requires a valid session or bearer token.
        """
        url = f"{self._base}{self._PREFIX}/revoke"
        resp = _http.request(self._session, "POST", url)
        if resp.status_code in (200, 204):
            return
        _raise_auth(resp)
