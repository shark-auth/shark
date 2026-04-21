"""Shared HTTP session helper with sane timeouts."""

from __future__ import annotations

from typing import Optional

import requests

DEFAULT_TIMEOUT = 10.0


def new_session(user_agent: str = "shark-auth-python/0.1.0") -> requests.Session:
    """Return a requests.Session with a default User-Agent set."""
    s = requests.Session()
    s.headers.update({"User-Agent": user_agent, "Accept": "application/json"})
    return s


def request(
    session: requests.Session,
    method: str,
    url: str,
    *,
    timeout: Optional[float] = None,
    **kwargs: object,
) -> requests.Response:
    """Thin wrapper applying a default timeout."""
    return session.request(method, url, timeout=timeout or DEFAULT_TIMEOUT, **kwargs)
