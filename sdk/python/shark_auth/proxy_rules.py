"""Proxy rules CRUD — DB-backed admin API (v1.5)."""

from __future__ import annotations

from typing import Any, Dict, List, Optional, TypedDict

from . import _http
from .errors import SharkAuthError


class SharkAPIError(SharkAuthError):
    """Error returned by the SharkAuth admin API."""

    def __init__(self, message: str, code: str = "", status: int = 0) -> None:
        super().__init__(message)
        self.code = code
        self.status = status


class ProxyRule(TypedDict):
    """A single DB-backed proxy rule."""

    id: str
    app_id: str
    name: str
    pattern: str
    methods: List[str]
    require: str
    allow: str
    scopes: List[str]
    enabled: bool
    priority: int
    tier_match: str
    m2m: bool
    created_at: str
    updated_at: str


class CreateProxyRuleInput(TypedDict, total=False):
    """Input shape for creating a proxy rule."""

    app_id: str
    name: str
    pattern: str
    methods: List[str]
    require: str
    allow: str
    scopes: List[str]
    enabled: bool
    priority: int
    tier_match: str
    m2m: bool


# UpdateProxyRuleInput is identical to CreateProxyRuleInput (all fields optional).
UpdateProxyRuleInput = CreateProxyRuleInput


class ImportResult(TypedDict):
    """Result of a YAML bulk-import operation."""

    imported: int
    errors: List[Dict[str, str]]


def _raise(resp: Any) -> None:
    """Raise SharkAPIError from a non-2xx response."""
    try:
        body = resp.json()
        err = body.get("error", {})
        code = err.get("code", "unknown_error") if isinstance(err, dict) else "unknown_error"
        msg = err.get("message", resp.text[:200]) if isinstance(err, dict) else resp.text[:200]
    except Exception:
        code = "unknown_error"
        msg = resp.text[:200]
    raise SharkAPIError(msg, code=code, status=resp.status_code)


class ProxyRulesClient:
    """Admin client for DB-backed proxy rules.

    All methods require an admin API key which is passed as a Bearer token.

    Parameters
    ----------
    base_url:
        Base URL of the SharkAuth server (e.g. ``https://auth.example.com``).
    token:
        Admin API key (``sk_live_...``).
    session:
        Optional pre-configured :class:`requests.Session`.
    """

    _PREFIX = "/api/v1/admin/proxy/rules"

    def __init__(
        self,
        base_url: str,
        token: str,
        *,
        session: Optional[object] = None,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._token = token
        self._session = session or _http.new_session()

    def _auth(self) -> Dict[str, str]:
        return {"Authorization": f"Bearer {self._token}"}

    # ------------------------------------------------------------------
    # List
    # ------------------------------------------------------------------

    def list_rules(self, app_id: Optional[str] = None) -> List[ProxyRule]:
        """Return all proxy rules, optionally filtered by *app_id*."""
        params: Dict[str, str] = {}
        if app_id is not None:
            params["app_id"] = app_id
        url = f"{self._base}{self._PREFIX}/db"
        resp = _http.request(self._session, "GET", url, headers=self._auth(), params=params)
        if resp.status_code == 200:
            return resp.json().get("data", [])
        _raise(resp)

    # ------------------------------------------------------------------
    # Create
    # ------------------------------------------------------------------

    def create_rule(
        self,
        spec: CreateProxyRuleInput,
        id: Optional[str] = None,  # noqa: A002
    ) -> ProxyRule:
        """Create a new proxy rule.  Returns the created rule object."""
        body: Dict[str, Any] = dict(spec)
        if id is not None:
            body["id"] = id
        url = f"{self._base}{self._PREFIX}/db"
        resp = _http.request(self._session, "POST", url, headers=self._auth(), json=body)
        if resp.status_code in (200, 201):
            return resp.json().get("data", resp.json())
        _raise(resp)

    # ------------------------------------------------------------------
    # Get
    # ------------------------------------------------------------------

    def get_rule(self, rule_id: str) -> ProxyRule:
        """Fetch a single proxy rule by *rule_id*."""
        url = f"{self._base}{self._PREFIX}/db/{rule_id}"
        resp = _http.request(self._session, "GET", url, headers=self._auth())
        if resp.status_code == 200:
            return resp.json().get("data", resp.json())
        _raise(resp)

    # ------------------------------------------------------------------
    # Update
    # ------------------------------------------------------------------

    def update_rule(self, rule_id: str, **patch: Any) -> ProxyRule:
        """Partially update a proxy rule.  Only supplied fields are mutated."""
        url = f"{self._base}{self._PREFIX}/db/{rule_id}"
        resp = _http.request(self._session, "PATCH", url, headers=self._auth(), json=patch)
        if resp.status_code == 200:
            return resp.json().get("data", resp.json())
        _raise(resp)

    # ------------------------------------------------------------------
    # Delete
    # ------------------------------------------------------------------

    def delete_rule(self, rule_id: str) -> None:
        """Delete a proxy rule by *rule_id*."""
        url = f"{self._base}{self._PREFIX}/db/{rule_id}"
        resp = _http.request(self._session, "DELETE", url, headers=self._auth())
        if resp.status_code == 204:
            return
        _raise(resp)

